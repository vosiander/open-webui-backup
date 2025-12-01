package plugins

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/database"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

type StatisticsPlugin struct {
	verbose bool
}

func NewStatisticsPlugin() *StatisticsPlugin {
	return &StatisticsPlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *StatisticsPlugin) Name() string {
	return "statistics"
}

// Description returns a short description of the plugin
func (p *StatisticsPlugin) Description() string {
	return "Display statistics about backup content types and estimated sizes"
}

func (p *StatisticsPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&p.verbose, "verbose", "v", false, "Show detailed information")
}

// Execute runs the plugin with the given configuration
func (p *StatisticsPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Gathering backup statistics...")

	if cfg.OpenWebUIAPIKey == "" {
		return fmt.Errorf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	// Create client
	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	// Collect statistics
	stats := &BackupStatistics{}

	// Knowledge bases
	if p.verbose {
		logrus.Info("Fetching knowledge bases...")
	}
	knowledgeBases, err := client.ListKnowledge()
	if err != nil {
		logrus.Warnf("Failed to fetch knowledge bases: %v", err)
	} else {
		stats.KnowledgeCount = len(knowledgeBases)
		// Calculate size of files in knowledge bases
		for _, kb := range knowledgeBases {
			if kb.Data != nil && kb.Data.FileIDs != nil {
				for _, fileID := range kb.Data.FileIDs {
					fileData, err := client.GetFile(fileID)
					if err != nil {
						if p.verbose {
							logrus.Warnf("Failed to get file %s: %v", fileID, err)
						}
						continue
					}
					stats.KnowledgeSize += fileData.Meta.Size
				}
			}
		}
	}

	// Models
	if p.verbose {
		logrus.Info("Fetching models...")
	}
	models, err := client.ExportModels()
	if err != nil {
		logrus.Warnf("Failed to fetch models: %v", err)
	} else {
		stats.ModelsCount = len(models)
	}

	// Tools
	if p.verbose {
		logrus.Info("Fetching tools...")
	}
	tools, err := client.ExportTools()
	if err != nil {
		logrus.Warnf("Failed to fetch tools: %v", err)
	} else {
		stats.ToolsCount = len(tools)
	}

	// Prompts
	if p.verbose {
		logrus.Info("Fetching prompts...")
	}
	prompts, err := client.ListPrompts()
	if err != nil {
		logrus.Warnf("Failed to fetch prompts: %v", err)
	} else {
		stats.PromptsCount = len(prompts)
	}

	// Files
	if p.verbose {
		logrus.Info("Fetching files...")
	}
	files, err := client.ListFiles()
	if err != nil {
		logrus.Warnf("Failed to fetch files: %v", err)
	} else {
		stats.FilesCount = len(files)
		for _, file := range files {
			stats.FilesSize += file.Meta.Size
		}
	}

	// Chats
	if p.verbose {
		logrus.Info("Fetching chats...")
	}
	chats, err := client.GetAllChats()
	if err != nil {
		logrus.Warnf("Failed to fetch chats: %v", err)
	} else {
		stats.ChatsCount = len(chats)
	}

	// Groups
	if p.verbose {
		logrus.Info("Fetching groups...")
	}
	groups, err := client.GetAllGroups()
	if err != nil {
		logrus.Warnf("Failed to fetch groups: %v", err)
	} else {
		stats.GroupsCount = len(groups)
	}

	// Feedbacks
	if p.verbose {
		logrus.Info("Fetching feedbacks...")
	}
	feedbacks, err := client.GetAllFeedbacks()
	if err != nil {
		logrus.Warnf("Failed to fetch feedbacks: %v", err)
	} else {
		stats.FeedbacksCount = len(feedbacks)
	}

	// Users
	if p.verbose {
		logrus.Info("Fetching users...")
	}
	users, err := client.GetAllUsers()
	if err != nil {
		logrus.Warnf("Failed to fetch users: %v", err)
	} else {
		stats.UsersCount = len(users)
	}

	// Database
	stats.DatabaseConfigured = database.IsPostgresURLSet()
	if stats.DatabaseConfigured {
		postgresURL := database.GetPostgresURLFromEnv()
		dbConfig, err := database.ParsePostgresURL(postgresURL)
		if err == nil {
			stats.DatabaseName = dbConfig.Database
		}
	}

	// Display statistics
	p.displayStatistics(stats)

	return nil
}

// BackupStatistics holds statistics about backup content
type BackupStatistics struct {
	KnowledgeCount     int
	KnowledgeSize      int64
	ModelsCount        int
	ToolsCount         int
	PromptsCount       int
	FilesCount         int
	FilesSize          int64
	ChatsCount         int
	GroupsCount        int
	FeedbacksCount     int
	UsersCount         int
	DatabaseConfigured bool
	DatabaseName       string
}

// displayStatistics prints the statistics in a formatted table
func (p *StatisticsPlugin) displayStatistics(stats *BackupStatistics) {
	logrus.Info("Backup Statistics")
	logrus.Info("=================\n")

	// Create tab writer for aligned columns using buffer
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)

	// Header
	fmt.Fprintln(w, "Content Type\tCount\tSize")
	fmt.Fprintln(w, "------------\t-----\t----")

	// Data rows
	fmt.Fprintf(w, "Knowledge Bases\t%d\t%s\n", stats.KnowledgeCount, formatSize(stats.KnowledgeSize))
	fmt.Fprintf(w, "Models\t%d\tN/A\n", stats.ModelsCount)
	fmt.Fprintf(w, "Tools\t%d\tN/A\n", stats.ToolsCount)
	fmt.Fprintf(w, "Prompts\t%d\tN/A\n", stats.PromptsCount)
	fmt.Fprintf(w, "Files\t%d\t%s\n", stats.FilesCount, formatSize(stats.FilesSize))
	fmt.Fprintf(w, "Chats\t%d\tN/A\n", stats.ChatsCount)
	fmt.Fprintf(w, "Groups\t%d\tN/A\n", stats.GroupsCount)
	fmt.Fprintf(w, "Feedbacks\t%d\tN/A\n", stats.FeedbacksCount)
	fmt.Fprintf(w, "Users\t%d\tN/A\n", stats.UsersCount)

	if stats.DatabaseConfigured {
		fmt.Fprintf(w, "Database\tConfigured\t%s\n", stats.DatabaseName)
	} else {
		fmt.Fprintf(w, "Database\tNot Configured\t-\n")
	}

	w.Flush()

	// Output the table via logrus
	logrus.Info(buf.String())

	// Summary
	totalItems := stats.KnowledgeCount + stats.ModelsCount + stats.ToolsCount + stats.PromptsCount +
		stats.FilesCount + stats.ChatsCount + stats.GroupsCount + stats.FeedbacksCount + stats.UsersCount

	logrus.Infof("Total Items: %d", totalItems)

	// Calculate total download size (only files and knowledge base files)
	totalSize := stats.FilesSize + stats.KnowledgeSize
	if totalSize > 0 {
		logrus.Infof("Estimated Download Size: %s (files and knowledge base content)", formatSize(totalSize))
	} else {
		logrus.Info("Estimated Download Size: No file content available")
	}

	// Notes
	logrus.Info("Note: Size estimates only available for Files and Knowledge Base content.")
	if stats.DatabaseConfigured {
		logrus.Info("Database backup size depends on actual database content.")
	}
}

// formatSize converts bytes to human-readable format
func formatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
