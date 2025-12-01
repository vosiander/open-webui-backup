package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

type ChatsPlugin struct {
	// Store config for subcommands to use
	config *config.Config

	// Common flags
	jsonOutput bool
	verbose    bool

	// List/search flags
	page           int
	includePinned  bool
	includeFolders bool
	query          string

	// Get flags
	chatID string

	// Folder flags
	folderID string

	// Archived flags
	orderBy   string
	direction string

	// Shared flags
	shareID string

	// Live flags
	timeframe string
	interval  string
}

func NewChatsPlugin() *ChatsPlugin {
	return &ChatsPlugin{}
}

// SetConfig stores the config for use by subcommands
func (p *ChatsPlugin) SetConfig(cfg *config.Config) {
	p.config = cfg
}

func (p *ChatsPlugin) Name() string {
	return "chats"
}

func (p *ChatsPlugin) Description() string {
	return "Manage and explore chat conversations from Open WebUI"
}

func (p *ChatsPlugin) SetupFlags(cmd *cobra.Command) {
	// Common flags
	cmd.PersistentFlags().BoolVar(&p.jsonOutput, "json", false, "Output in JSON format")
	cmd.PersistentFlags().BoolVarP(&p.verbose, "verbose", "v", false, "Show detailed information")

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List chats with pagination",
		Long:  "List chats from the paginated API endpoint with optional filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.config.OpenWebUIAPIKey == "" {
				return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
			}
			client := openwebui.NewClient(p.config.OpenWebUIURL, p.config.OpenWebUIAPIKey)
			return p.executeList(client)
		},
	}
	listCmd.Flags().IntVar(&p.page, "page", 1, "Page number (1-indexed)")
	listCmd.Flags().BoolVar(&p.includePinned, "include-pinned", true, "Include pinned chats")
	listCmd.Flags().BoolVar(&p.includeFolders, "include-folders", true, "Include folder information")

	// All subcommand (paginated endpoint, fetches all pages)
	allCmd := &cobra.Command{
		Use:   "all",
		Short: "Get all chats (paginated endpoint)",
		Long:  "Fetch all chats by iterating through pages of the paginated list endpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.config.OpenWebUIAPIKey == "" {
				return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
			}
			client := openwebui.NewClient(p.config.OpenWebUIURL, p.config.OpenWebUIAPIKey)
			return p.executeAll(client)
		},
	}

	// All-db subcommand (database endpoint)
	allDbCmd := &cobra.Command{
		Use:   "all-db",
		Short: "Get all chats from database",
		Long:  "Fetch all chats directly from the database endpoint (non-paginated)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.config.OpenWebUIAPIKey == "" {
				return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
			}
			client := openwebui.NewClient(p.config.OpenWebUIURL, p.config.OpenWebUIAPIKey)
			return p.executeAllDB(client)
		},
	}

	// Get subcommand
	getCmd := &cobra.Command{
		Use:   "get <chat-id>",
		Short: "Get a specific chat by ID",
		Long:  "Retrieve detailed information about a specific chat",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.config.OpenWebUIAPIKey == "" {
				return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
			}
			p.chatID = args[0]
			client := openwebui.NewClient(p.config.OpenWebUIURL, p.config.OpenWebUIAPIKey)
			return p.executeGet(client)
		},
	}

	// Search subcommand
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search chats",
		Long:  "Search for chats by query string",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.config.OpenWebUIAPIKey == "" {
				return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
			}
			p.query = args[0]
			client := openwebui.NewClient(p.config.OpenWebUIURL, p.config.OpenWebUIAPIKey)
			return p.executeSearch(client)
		},
	}
	searchCmd.Flags().IntVar(&p.page, "page", 1, "Page number (1-indexed)")

	// Folder subcommand
	folderCmd := &cobra.Command{
		Use:   "folder <folder-id>",
		Short: "Get chats in a folder",
		Long:  "Retrieve all chats within a specific folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.config.OpenWebUIAPIKey == "" {
				return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
			}
			p.folderID = args[0]
			client := openwebui.NewClient(p.config.OpenWebUIURL, p.config.OpenWebUIAPIKey)
			return p.executeFolder(client)
		},
	}
	folderCmd.Flags().IntVar(&p.page, "page", 1, "Page number (1-indexed)")

	// Archived subcommand
	archivedCmd := &cobra.Command{
		Use:   "archived",
		Short: "Get archived chats",
		Long:  "Retrieve archived chats with optional sorting",
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.config.OpenWebUIAPIKey == "" {
				return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
			}
			client := openwebui.NewClient(p.config.OpenWebUIURL, p.config.OpenWebUIAPIKey)
			return p.executeArchived(client)
		},
	}
	archivedCmd.Flags().IntVar(&p.page, "page", 1, "Page number (1-indexed)")
	archivedCmd.Flags().StringVar(&p.orderBy, "order-by", "updated_at", "Field to order by")
	archivedCmd.Flags().StringVar(&p.direction, "direction", "desc", "Sort direction (asc/desc)")
	archivedCmd.Flags().StringVar(&p.query, "query", "", "Search query for archived chats")

	// Shared subcommand
	sharedCmd := &cobra.Command{
		Use:   "shared <share-id>",
		Short: "Get a shared chat",
		Long:  "Retrieve a chat by its share ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.config.OpenWebUIAPIKey == "" {
				return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
			}
			p.shareID = args[0]
			client := openwebui.NewClient(p.config.OpenWebUIURL, p.config.OpenWebUIAPIKey)
			return p.executeShared(client)
		},
	}

	// Live subcommand
	liveCmd := &cobra.Command{
		Use:   "live",
		Short: "Monitor recent chats in real-time",
		Long:  "Continuously display chats updated within a specified timeframe, refreshing at regular intervals",
		RunE: func(cmd *cobra.Command, args []string) error {
			if p.config.OpenWebUIAPIKey == "" {
				return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
			}
			client := openwebui.NewClient(p.config.OpenWebUIURL, p.config.OpenWebUIAPIKey)
			return p.executeLive(client)
		},
	}
	liveCmd.Flags().StringVar(&p.timeframe, "timeframe", "5m", "Show chats updated in last duration (e.g., 5m, 1h, 30s)")
	liveCmd.Flags().StringVar(&p.interval, "interval", "15s", "Refresh interval (e.g., 15s, 30s, 1m)")

	// Add subcommands to main command
	cmd.AddCommand(listCmd, allCmd, allDbCmd, getCmd, searchCmd, folderCmd, archivedCmd, sharedCmd, liveCmd)
}

func (p *ChatsPlugin) Execute(cfg *config.Config) error {
	// This method is required by the Plugin interface but not used
	// since subcommands execute directly via their RunE functions
	return fmt.Errorf("no subcommand specified. Use --help to see available subcommands")
}

func (p *ChatsPlugin) executeList(client *openwebui.Client) error {
	if p.verbose {
		logrus.Infof("Fetching chats list (page %d)...", p.page)
	}

	chats, err := client.GetChatsList(p.page, p.includePinned, p.includeFolders)
	if err != nil {
		return fmt.Errorf("failed to fetch chats: %w", err)
	}

	if p.jsonOutput {
		return p.outputJSON(chats)
	}

	// Get user lookup map
	userMap, err := p.getUserLookup(client)
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v. Displaying without user names.", err)
		userMap = make(map[string]string)
	}

	return p.displayChatTitleIDTable(chats, userMap)
}

func (p *ChatsPlugin) executeAll(client *openwebui.Client) error {
	if p.verbose {
		logrus.Info("Fetching all chats (iterating through pages)...")
	}

	allChats := []openwebui.ChatTitleID{}
	page := 1

	for {
		chats, err := client.GetChatsList(page, true, true)
		if err != nil {
			return fmt.Errorf("failed to fetch chats on page %d: %w", page, err)
		}

		if len(chats) == 0 {
			break
		}

		allChats = append(allChats, chats...)

		if p.verbose {
			logrus.Infof("Fetched page %d (%d chats)", page, len(chats))
		}

		page++
	}

	if p.verbose {
		logrus.Infof("Total chats fetched: %d", len(allChats))
	}

	if p.jsonOutput {
		return p.outputJSON(allChats)
	}

	userMap, err := p.getUserLookup(client)
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v. Displaying without user names.", err)
		userMap = make(map[string]string)
	}

	return p.displayChatTitleIDTable(allChats, userMap)
}

func (p *ChatsPlugin) executeAllDB(client *openwebui.Client) error {
	if p.verbose {
		logrus.Info("Fetching all chats from database...")
	}

	chats, err := client.GetAllChatsDB()
	if err != nil {
		return fmt.Errorf("failed to fetch chats: %w", err)
	}

	if p.verbose {
		logrus.Infof("Total chats fetched: %d", len(chats))
	}

	if p.jsonOutput {
		return p.outputJSON(chats)
	}

	userMap, err := p.getUserLookup(client)
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v. Displaying without user names.", err)
		userMap = make(map[string]string)
	}

	return p.displayChatTable(chats, userMap)
}

func (p *ChatsPlugin) executeGet(client *openwebui.Client) error {
	if p.verbose {
		logrus.Infof("Fetching chat %s...", p.chatID)
	}

	chat, err := client.GetChatByID(p.chatID)
	if err != nil {
		return fmt.Errorf("failed to fetch chat: %w", err)
	}

	if p.jsonOutput {
		return p.outputJSON(chat)
	}

	userMap, err := p.getUserLookup(client)
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v. Displaying without user names.", err)
		userMap = make(map[string]string)
	}

	return p.displaySingleChat(chat, userMap)
}

func (p *ChatsPlugin) executeSearch(client *openwebui.Client) error {
	if p.verbose {
		logrus.Infof("Searching chats for '%s' (page %d)...", p.query, p.page)
	}

	chats, err := client.SearchChats(p.query, p.page)
	if err != nil {
		return fmt.Errorf("failed to search chats: %w", err)
	}

	if p.jsonOutput {
		return p.outputJSON(chats)
	}

	userMap, err := p.getUserLookup(client)
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v. Displaying without user names.", err)
		userMap = make(map[string]string)
	}

	return p.displayChatTitleIDTable(chats, userMap)
}

func (p *ChatsPlugin) executeFolder(client *openwebui.Client) error {
	if p.verbose {
		logrus.Infof("Fetching chats in folder %s (page %d)...", p.folderID, p.page)
	}

	chats, err := client.GetChatsByFolder(p.folderID, p.page)
	if err != nil {
		return fmt.Errorf("failed to fetch chats: %w", err)
	}

	if p.jsonOutput {
		return p.outputJSON(chats)
	}

	userMap, err := p.getUserLookup(client)
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v. Displaying without user names.", err)
		userMap = make(map[string]string)
	}

	return p.displayChatTable(chats, userMap)
}

func (p *ChatsPlugin) executeArchived(client *openwebui.Client) error {
	if p.verbose {
		logrus.Infof("Fetching archived chats (page %d)...", p.page)
	}

	chats, err := client.GetArchivedChats(p.page, p.query, p.orderBy, p.direction)
	if err != nil {
		return fmt.Errorf("failed to fetch archived chats: %w", err)
	}

	if p.jsonOutput {
		return p.outputJSON(chats)
	}

	userMap, err := p.getUserLookup(client)
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v. Displaying without user names.", err)
		userMap = make(map[string]string)
	}

	return p.displayChatTitleIDTable(chats, userMap)
}

func (p *ChatsPlugin) executeShared(client *openwebui.Client) error {
	if p.verbose {
		logrus.Infof("Fetching shared chat %s...", p.shareID)
	}

	chat, err := client.GetSharedChat(p.shareID)
	if err != nil {
		return fmt.Errorf("failed to fetch shared chat: %w", err)
	}

	if p.jsonOutput {
		return p.outputJSON(chat)
	}

	userMap, err := p.getUserLookup(client)
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v. Displaying without user names.", err)
		userMap = make(map[string]string)
	}

	return p.displaySingleChat(chat, userMap)
}

// getUserLookup fetches all users and creates a lookup map
func (p *ChatsPlugin) getUserLookup(client *openwebui.Client) (map[string]string, error) {
	users, err := client.GetAllUsers()
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]string)
	for _, user := range users {
		userMap[user.ID] = user.Name
	}

	return userMap, nil
}

// displayChatTitleIDTable displays a table of ChatTitleID objects
func (p *ChatsPlugin) displayChatTitleIDTable(chats []openwebui.ChatTitleID, userMap map[string]string) error {
	if len(chats) == 0 {
		logrus.Info("No chats found")
		return nil
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)

	// Header
	fmt.Fprintln(w, "ID\tTitle\tUser ID\tUser Name\tCreated\tUpdated")
	fmt.Fprintln(w, "--\t-----\t-------\t---------\t-------\t-------")

	// Data rows
	for _, chat := range chats {
		// Display "-" for empty User ID
		userID := chat.UserID
		if userID == "" {
			userID = "-"
		}

		userName := userMap[chat.UserID]
		if userName == "" {
			userName = "-"
		}

		createdTime := formatTimestamp(chat.CreatedAt)
		updatedTime := formatTimestamp(chat.UpdatedAt)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			chat.ID, // Full ID, no truncation
			truncate(chat.Title, 40),
			userID, // Show "-" if empty
			truncate(userName, 20),
			createdTime,
			updatedTime,
		)
	}

	w.Flush()
	fmt.Println(buf.String())

	logrus.Infof("Total: %d chats", len(chats))

	return nil
}

// displayChatTable displays a table of full Chat objects
func (p *ChatsPlugin) displayChatTable(chats []openwebui.Chat, userMap map[string]string) error {
	if len(chats) == 0 {
		logrus.Info("No chats found")
		return nil
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)

	// Header
	fmt.Fprintln(w, "ID\tTitle\tUser ID\tUser Name\tMessages\tCreated\tUpdated")
	fmt.Fprintln(w, "--\t-----\t-------\t---------\t--------\t-------\t-------")

	// Data rows
	for _, chat := range chats {
		// Display "-" for empty User ID
		userID := chat.UserID
		if userID == "" {
			userID = "-"
		}

		userName := userMap[chat.UserID]
		if userName == "" {
			userName = "-"
		}

		messageCount := len(chat.Chat.Messages)
		createdTime := formatTimestamp(chat.CreatedAt)
		updatedTime := formatTimestamp(chat.UpdatedAt)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			chat.ID, // Full ID, no truncation
			truncate(chat.Title, 40),
			userID, // Show "-" if empty
			truncate(userName, 20),
			messageCount,
			createdTime,
			updatedTime,
		)
	}

	w.Flush()
	fmt.Println(buf.String())

	logrus.Infof("Total: %d chats", len(chats))

	return nil
}

// displaySingleChat displays detailed information about a single chat
func (p *ChatsPlugin) displaySingleChat(chat *openwebui.Chat, userMap map[string]string) error {
	userName := userMap[chat.UserID]
	if userName == "" {
		userName = "-"
	}

	fmt.Println("Chat Details")
	fmt.Println("============")
	fmt.Printf("ID:           %s\n", chat.ID)
	fmt.Printf("Title:        %s\n", chat.Title)
	fmt.Printf("User ID:      %s\n", chat.UserID)
	fmt.Printf("User Name:    %s\n", userName)
	fmt.Printf("Messages:     %d\n", len(chat.Chat.Messages))
	fmt.Printf("Created:      %s\n", formatTimestamp(chat.CreatedAt))
	fmt.Printf("Updated:      %s\n", formatTimestamp(chat.UpdatedAt))

	if len(chat.Chat.Messages) > 0 {
		fmt.Println("\nMessages:")
		fmt.Println("---------")

		for i, msg := range chat.Chat.Messages {
			fmt.Printf("\n[%d] %s:\n", i+1, msg.Role)
			fmt.Printf("    %s\n", truncate(msg.Content, 200))
			if msg.Model != "" {
				fmt.Printf("    Model: %s\n", msg.Model)
			}
		}
	}

	return nil
}

// outputJSON outputs the data as JSON
func (p *ChatsPlugin) outputJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// formatTimestamp converts Unix timestamp to readable format
func formatTimestamp(ts int64) string {
	if ts == 0 {
		return "-"
	}
	t := time.Unix(ts, 0)
	return t.Format("2006-01-02 15:04")
}

// truncate shortens a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// executeLive continuously monitors and displays recent chats
func (p *ChatsPlugin) executeLive(client *openwebui.Client) error {
	// Parse timeframe duration
	timeframeDuration, err := time.ParseDuration(p.timeframe)
	if err != nil {
		return fmt.Errorf("invalid timeframe '%s': %w", p.timeframe, err)
	}

	// Parse interval duration
	intervalDuration, err := time.ParseDuration(p.interval)
	if err != nil {
		return fmt.Errorf("invalid interval '%s': %w", p.interval, err)
	}

	// Setup signal handler for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Get user lookup map once
	userMap, err := p.getUserLookup(client)
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v. Displaying without user names.", err)
		userMap = make(map[string]string)
	}

	// Create a done channel to coordinate shutdown
	done := make(chan bool)

	// Start the refresh loop in a goroutine
	go func() {
		// Immediate first fetch
		p.fetchAndDisplayLive(client, timeframeDuration, userMap)

		ticker := time.NewTicker(intervalDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.fetchAndDisplayLive(client, timeframeDuration, userMap)
			case <-done:
				return
			}
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	close(done)

	return nil
}

// fetchAndDisplayLive fetches and displays chats updated within the timeframe
func (p *ChatsPlugin) fetchAndDisplayLive(client *openwebui.Client, timeframe time.Duration, userMap map[string]string) {
	// Calculate cutoff time
	cutoffTime := time.Now().Add(-timeframe)
	cutoffUnix := cutoffTime.Unix()

	// Fetch all chats
	allChats, err := client.GetAllChatsDB()
	if err != nil {
		logrus.Errorf("Failed to fetch chats: %v", err)
		return
	}

	// Filter chats by timeframe
	var recentChats []openwebui.Chat
	for _, chat := range allChats {
		if chat.UpdatedAt >= cutoffUnix {
			recentChats = append(recentChats, chat)
		}
	}

	// Sort by UpdatedAt descending (most recent first)
	// Simple bubble sort since we expect small numbers
	for i := 0; i < len(recentChats)-1; i++ {
		for j := i + 1; j < len(recentChats); j++ {
			if recentChats[i].UpdatedAt < recentChats[j].UpdatedAt {
				recentChats[i], recentChats[j] = recentChats[j], recentChats[i]
			}
		}
	}

	// Clear screen
	clearScreen()

	// Display header
	now := time.Now()
	fmt.Println("=== Live Chat Monitor ===")
	fmt.Printf("Showing chats updated in last %s | Refreshing every %s\n", p.timeframe, p.interval)
	fmt.Printf("Last updated: %s\n\n", now.Format("2006-01-02 15:04:05"))

	// Display table
	if len(recentChats) == 0 {
		fmt.Println("No chats found in the specified timeframe")
	} else {
		var buf bytes.Buffer
		w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)

		// Header
		fmt.Fprintln(w, "ID\tTitle\tUser Name\tUpdated")
		fmt.Fprintln(w, "--------\t------------------------------------\t------------------\t----------")

		// Data rows
		for _, chat := range recentChats {
			userName := userMap[chat.UserID]
			if userName == "" {
				userName = "-"
			}

			relativeTime := formatRelativeTime(chat.UpdatedAt)

			// Truncate ID to first 8 characters for display
			displayID := chat.ID
			if len(displayID) > 8 {
				displayID = displayID[:8]
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				displayID,
				truncate(chat.Title, 35),
				truncate(userName, 18),
				relativeTime,
			)
		}

		w.Flush()
		fmt.Print(buf.String())
		fmt.Printf("\nTotal: %d chats\n", len(recentChats))
	}
}

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// formatRelativeTime formats a Unix timestamp as relative time (e.g., "2m ago", "1h ago")
func formatRelativeTime(ts int64) string {
	if ts == 0 {
		return "-"
	}

	now := time.Now()
	t := time.Unix(ts, 0)
	duration := now.Sub(t)

	if duration < 0 {
		return "just now"
	}

	seconds := int(duration.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24

	if seconds < 60 {
		if seconds < 10 {
			return "just now"
		}
		return fmt.Sprintf("%ds ago", seconds)
	} else if minutes < 60 {
		return fmt.Sprintf("%dm ago", minutes)
	} else if hours < 24 {
		return fmt.Sprintf("%dh ago", hours)
	} else {
		return fmt.Sprintf("%dd ago", days)
	}
}
