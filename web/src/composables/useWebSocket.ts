import {onMounted, onUnmounted, ref} from 'vue';
import {getWebSocketService} from '../services/websocket';
import type {WebSocketMessage} from '../types/api';

export function useWebSocket() {
  const ws = getWebSocketService();
  const isConnected = ref(false);
  const lastMessage = ref<WebSocketMessage | null>(null);

  const messageHandlers: Array<() => void> = [];

  const addMessageHandler = (handler: (message: WebSocketMessage) => void) => {
    const unsubscribe = ws.onMessage((message) => {
      lastMessage.value = message;
      handler(message);
    });
    messageHandlers.push(unsubscribe);
    return unsubscribe;
  };

  onMounted(() => {
    ws.connect();
    
    const unsubscribeConnect = ws.onConnect(() => {
      isConnected.value = true;
    });

    const unsubscribeDisconnect = ws.onDisconnect(() => {
      isConnected.value = false;
    });

    messageHandlers.push(unsubscribeConnect, unsubscribeDisconnect);
    
    isConnected.value = ws.isConnected();
  });

  onUnmounted(() => {
    messageHandlers.forEach(unsubscribe => unsubscribe());
  });

  return {
    isConnected,
    lastMessage,
    addMessageHandler,
    send: ws.send.bind(ws),
  };
}
