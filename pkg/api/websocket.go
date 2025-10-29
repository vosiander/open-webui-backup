package api

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// Hub manages WebSocket connections
type Hub struct {
	clients    map[*Connection]bool
	broadcast  chan WebSocketMessage
	register   chan *Connection
	unregister chan *Connection
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Connection]bool),
		broadcast:  make(chan WebSocketMessage, 256),
		register:   make(chan *Connection),
		unregister: make(chan *Connection),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			logrus.Debug("WebSocket client connected")

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				close(conn.send)
			}
			h.mu.Unlock()
			logrus.Debug("WebSocket client disconnected")

		case message := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				select {
				case conn.send <- message:
				default:
					// Client is slow, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(msg WebSocketMessage) {
	select {
	case h.broadcast <- msg:
	default:
		// Channel full, skip
	}
}

// Connection represents a WebSocket connection
type Connection struct {
	ws   *websocket.Conn
	send chan WebSocketMessage
	hub  *Hub
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// HandleWebSocket upgrades HTTP connection to WebSocket
func (h *Hub) HandleWebSocket(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to upgrade to WebSocket")
		return err
	}

	conn := &Connection{
		ws:   ws,
		send: make(chan WebSocketMessage, 256),
		hub:  h,
	}

	h.register <- conn

	// Start goroutines for reading and writing
	go conn.writePump()
	go conn.readPump()

	return nil
}

// readPump handles incoming messages from the client
func (c *Connection) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()

	for {
		_, _, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).Error("WebSocket read error")
			}
			break
		}
		// We don't expect messages from client, just ping/pong
	}
}

// writePump handles outgoing messages to the client
func (c *Connection) writePump() {
	defer func() {
		c.ws.Close()
	}()

	for message := range c.send {
		err := c.ws.WriteJSON(message)
		if err != nil {
			logrus.WithError(err).Error("WebSocket write error")
			return
		}
	}
}
