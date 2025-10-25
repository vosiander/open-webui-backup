package plugins

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

type PurgePlugin struct {
	force        bool
	waitDuration time.Duration
	chats        bool
	files        bool
	models       bool
	knowledge    bool
	prompts      bool
	tools        bool
	functions    bool
	memories     bool
	feedbacks    bool
	groups       bool
	users        bool
}

func NewPurgePlugin() *PurgePlugin {
	return &PurgePlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *PurgePlugin) Name() string {
	return "purge"
}

// Description returns a short description of the plugin
func (p *PurgePlugin) Description() string {
	return "Remove all content from Open WebUI API (use with caution)"
}

func (p *PurgePlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&p.force, "force", "f", false, "Actually perform deletion (required)")
	cmd.Flags().DurationVarP(&p.waitDuration, "wait", "w", 5*time.Second, "Wait duration before performing deletions")
	cmd.Flags().BoolVar(&p.chats, "chats", false, "Purge only chats")
	cmd.Flags().BoolVar(&p.files, "files", false, "Purge only files")
	cmd.Flags().BoolVar(&p.models, "models", false, "Purge only models")
	cmd.Flags().BoolVar(&p.knowledge, "knowledge", false, "Purge only knowledge bases")
	cmd.Flags().BoolVar(&p.prompts, "prompts", false, "Purge only prompts")
	cmd.Flags().BoolVar(&p.tools, "tools", false, "Purge only tools")
	cmd.Flags().BoolVar(&p.functions, "functions", false, "Purge only functions")
	cmd.Flags().BoolVar(&p.memories, "memories", false, "Purge only memories")
	cmd.Flags().BoolVar(&p.feedbacks, "feedbacks", false, "Purge only feedbacks (purged before users)")
	cmd.Flags().BoolVar(&p.groups, "groups", false, "Purge only groups (purged before users)")
	cmd.Flags().BoolVar(&p.users, "users", false, "Purge only users (purged LAST, skips current user)")
}

// Execute runs the plugin with the given configuration
func (p *PurgePlugin) Execute(cfg *config.Config) error {
	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatal("OPEN_WEBUI_API_KEY environment variable is required")
	}

	// Create client
	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	// Determine what to purge
	purgeAll := !p.chats && !p.files && !p.models && !p.knowledge &&
		!p.prompts && !p.tools && !p.functions && !p.memories && !p.feedbacks && !p.groups && !p.users

	// Count items
	counts := make(map[string]int)
	if purgeAll || p.chats {
		chats, err := client.GetAllChats()
		if err != nil {
			logrus.Warnf("Failed to count chats: %v", err)
		} else {
			counts["chats"] = len(chats)
		}
	}

	if purgeAll || p.files {
		files, err := client.ListFiles()
		if err != nil {
			logrus.Warnf("Failed to count files: %v", err)
		} else {
			counts["files"] = len(files)
		}
	}

	if purgeAll || p.models {
		models, err := client.ExportModels()
		if err != nil {
			logrus.Warnf("Failed to count models: %v", err)
		} else {
			counts["models"] = len(models)
		}
	}

	if purgeAll || p.knowledge {
		knowledgeBases, err := client.ListKnowledge()
		if err != nil {
			logrus.Warnf("Failed to count knowledge bases: %v", err)
		} else {
			counts["knowledge"] = len(knowledgeBases)
		}
	}

	if purgeAll || p.prompts {
		prompts, err := client.ListPrompts()
		if err != nil {
			logrus.Warnf("Failed to count prompts: %v", err)
		} else {
			counts["prompts"] = len(prompts)
		}
	}

	if purgeAll || p.tools {
		tools, err := client.ListTools()
		if err != nil {
			logrus.Warnf("Failed to count tools: %v", err)
		} else {
			counts["tools"] = len(tools)
		}
	}

	if purgeAll || p.functions {
		functions, err := client.ListFunctions()
		if err != nil {
			logrus.Warnf("Failed to count functions: %v", err)
		} else {
			counts["functions"] = len(functions)
		}
	}

	if purgeAll || p.memories {
		memories, err := client.ListMemories()
		if err != nil {
			logrus.Warnf("Failed to count memories: %v", err)
		} else {
			counts["memories"] = len(memories)
		}
	}

	if purgeAll || p.groups {
		groups, err := client.GetAllGroups()
		if err != nil {
			logrus.Warnf("Failed to count groups: %v", err)
		} else {
			counts["groups"] = len(groups)
		}
	}

	if purgeAll || p.users {
		users, err := client.GetAllUsers()
		if err != nil {
			logrus.Warnf("Failed to count users: %v", err)
		} else {
			// Count users, but we'll skip the current user during deletion
			counts["users"] = len(users)
		}
	}

	// Calculate total
	total := 0
	for _, count := range counts {
		total += count
	}

	if total == 0 {
		logrus.Info("No items to purge")
		return nil
	}

	// Display what will be deleted
	if !p.force {
		logrus.Info("\n[Dry Run] Would delete the following:")
		for resource, count := range counts {
			if count > 0 {
				logrus.Infof("  - %s: %d items", resource, count)
			}
		}
		logrus.Infof("Total: %d items", total)
		logrus.Info("Run with --force to actually delete these items.")
		return nil
	}

	// Force mode - show warning and ask for confirmation
	logrus.Warn("⚠️  WARNING: This will permanently delete the following:")
	for resource, count := range counts {
		if count > 0 {
			logrus.Infof("  - %s: %d items", resource, count)
		}
	}
	logrus.Infof("Total: %d items", total)

	logrus.Infof("Waiting %s before proceeding...", p.waitDuration)
	time.Sleep(p.waitDuration)
	logrus.Info("Proceeding with deletion...")

	// Perform deletions
	if err := p.performDeletions(client, counts); err != nil {
		return err
	}

	logrus.Infof("Successfully deleted %d items", total)
	return nil
}

func (p *PurgePlugin) performDeletions(client *openwebui.Client, counts map[string]int) error {
	purgeAll := !p.chats && !p.files && !p.models && !p.knowledge &&
		!p.prompts && !p.tools && !p.functions && !p.memories && !p.feedbacks && !p.groups && !p.users

	// Delete chats
	if (purgeAll || p.chats) && counts["chats"] > 0 {
		logrus.Infof("Deleting chats... [%d items]", counts["chats"])
		if err := client.DeleteAllChats(); err != nil {
			return fmt.Errorf("failed to delete chats: %w", err)
		}
		logrus.Info("✓ Chats deleted")
	}

	// Delete files
	if (purgeAll || p.files) && counts["files"] > 0 {
		logrus.Infof("Deleting files... [%d items]", counts["files"])
		if err := client.DeleteAllFiles(); err != nil {
			return fmt.Errorf("failed to delete files: %w", err)
		}
		logrus.Info("✓ Files deleted")
	}

	// Delete models
	if (purgeAll || p.models) && counts["models"] > 0 {
		logrus.Infof("Deleting models... [%d items]", counts["models"])
		if err := client.DeleteAllModels(); err != nil {
			return fmt.Errorf("failed to delete models: %w", err)
		}
		logrus.Info("✓ Models deleted")
	}

	// Delete knowledge bases (must be done individually)
	if (purgeAll || p.knowledge) && counts["knowledge"] > 0 {
		logrus.Infof("Deleting knowledge bases... [%d items]", counts["knowledge"])
		knowledgeBases, err := client.ListKnowledge()
		if err != nil {
			return fmt.Errorf("failed to list knowledge bases: %w", err)
		}
		for i, kb := range knowledgeBases {
			if err := client.DeleteKnowledgeByID(kb.ID); err != nil {
				logrus.Warnf("Failed to delete knowledge base %s: %v", kb.ID, err)
			}
			if (i+1)%10 == 0 || i == len(knowledgeBases)-1 {
				logrus.Infof("  Progress: %d/%d", i+1, len(knowledgeBases))
			}
		}
		logrus.Info("✓ Knowledge bases deleted")
	}

	// Delete prompts (must be done individually)
	if (purgeAll || p.prompts) && counts["prompts"] > 0 {
		logrus.Infof("Deleting prompts... [%d items]", counts["prompts"])
		prompts, err := client.ListPrompts()
		if err != nil {
			return fmt.Errorf("failed to list prompts: %w", err)
		}
		for i, prompt := range prompts {
			if err := client.DeletePromptByCommand(prompt.Command); err != nil {
				logrus.Warnf("Failed to delete prompt %s: %v", prompt.Command, err)
			}
			if (i+1)%10 == 0 || i == len(prompts)-1 {
				logrus.Infof("  Progress: %d/%d", i+1, len(prompts))
			}
		}
		logrus.Info("✓ Prompts deleted")
	}

	// Delete tools (must be done individually)
	if (purgeAll || p.tools) && counts["tools"] > 0 {
		logrus.Infof("Deleting tools... [%d items]", counts["tools"])
		tools, err := client.ListTools()
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}
		for i, tool := range tools {
			if err := client.DeleteToolByID(tool.ID); err != nil {
				logrus.Warnf("Failed to delete tool %s: %v", tool.ID, err)
			}
			if (i+1)%10 == 0 || i == len(tools)-1 {
				logrus.Infof("  Progress: %d/%d", i+1, len(tools))
			}
		}
		logrus.Info("✓ Tools deleted")
	}

	// Delete functions (must be done individually)
	if (purgeAll || p.functions) && counts["functions"] > 0 {
		logrus.Infof("Deleting functions... [%d items]", counts["functions"])
		functions, err := client.ListFunctions()
		if err != nil {
			return fmt.Errorf("failed to list functions: %w", err)
		}
		for i, function := range functions {
			if err := client.DeleteFunctionByID(function.ID); err != nil {
				logrus.Warnf("Failed to delete function %s: %v", function.ID, err)
			}
			if (i+1)%10 == 0 || i == len(functions)-1 {
				logrus.Infof("  Progress: %d/%d", i+1, len(functions))
			}
		}
		logrus.Info("✓ Functions deleted")
	}

	// Delete memories
	if (purgeAll || p.memories) && counts["memories"] > 0 {
		logrus.Infof("Deleting memories... [%d items]", counts["memories"])
		if err := client.DeleteAllMemories(); err != nil {
			return fmt.Errorf("failed to delete memories: %w", err)
		}
		logrus.Info("✓ Memories deleted")
	}

	// Delete feedbacks (before users)
	if (purgeAll || p.feedbacks) && counts["feedbacks"] > 0 {
		logrus.Infof("Deleting feedbacks... [%d items]", counts["feedbacks"])
		feedbacks, err := client.GetAllFeedbacks()
		if err != nil {
			logrus.Warnf("Failed to list feedbacks: %v", err)
		} else {
			for i, feedback := range feedbacks {
				if err := client.DeleteFeedbackByID(feedback.ID); err != nil {
					logrus.Warnf("Failed to delete feedback %s: %v", feedback.ID, err)
				}
				if (i+1)%10 == 0 || i == len(feedbacks)-1 {
					logrus.Infof("  Progress: %d/%d", i+1, len(feedbacks))
				}
			}
			logrus.Info("✓ Feedbacks deleted")
		}
	}

	// Delete groups (before users)
	if (purgeAll || p.groups) && counts["groups"] > 0 {
		logrus.Infof("Deleting groups... [%d items]", counts["groups"])
		groups, err := client.GetAllGroups()
		if err != nil {
			return fmt.Errorf("failed to list groups: %w", err)
		}
		for i, group := range groups {
			if err := client.DeleteGroupByID(group.ID); err != nil {
				logrus.Warnf("Failed to delete group %s: %v", group.ID, err)
			}
			if (i+1)%10 == 0 || i == len(groups)-1 {
				logrus.Infof("  Progress: %d/%d", i+1, len(groups))
			}
		}
		logrus.Info("✓ Groups deleted")
	}

	// Delete users LAST (must be done individually with safety check)
	if (purgeAll || p.users) && counts["users"] > 0 {
		logrus.Infof("Deleting users... [%d items]", counts["users"])
		users, err := client.GetAllUsers()
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		// Get current session's API key for safety check
		currentAPIKey := client.GetAPIKey()

		deleted := 0
		skipped := 0
		for i, user := range users {
			// Safety check: Don't delete user with same API key as current session
			if user.APIKey != "" && user.APIKey == currentAPIKey {
				logrus.Warnf("  ⚠️  Skipping user %s (email: %s) - same API key as current session", user.Name, user.Email)
				skipped++
				continue
			}

			if err := client.DeleteUserByID(user.ID); err != nil {
				logrus.Warnf("  Failed to delete user %s (ID: %s): %v", user.Email, user.ID, err)
			} else {
				deleted++
			}

			if (i+1)%10 == 0 || i == len(users)-1 {
				logrus.Infof("  Progress: %d/%d (deleted: %d, skipped: %d)", i+1, len(users), deleted, skipped)
			}
		}
		if skipped > 0 {
			logrus.Infof("✓ Users deleted (skipped %d for safety)", skipped)
		} else {
			logrus.Info("✓ Users deleted")
		}
	}

	return nil
}
