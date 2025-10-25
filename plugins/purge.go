package plugins

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
)

type PurgePlugin struct {
	force     bool
	chats     bool
	files     bool
	models    bool
	knowledge bool
	prompts   bool
	tools     bool
	functions bool
	memories  bool
	feedbacks bool
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
	cmd.Flags().BoolVar(&p.chats, "chats", false, "Purge only chats")
	cmd.Flags().BoolVar(&p.files, "files", false, "Purge only files")
	cmd.Flags().BoolVar(&p.models, "models", false, "Purge only models")
	cmd.Flags().BoolVar(&p.knowledge, "knowledge", false, "Purge only knowledge bases")
	cmd.Flags().BoolVar(&p.prompts, "prompts", false, "Purge only prompts")
	cmd.Flags().BoolVar(&p.tools, "tools", false, "Purge only tools")
	cmd.Flags().BoolVar(&p.functions, "functions", false, "Purge only functions")
	cmd.Flags().BoolVar(&p.memories, "memories", false, "Purge only memories")
	cmd.Flags().BoolVar(&p.feedbacks, "feedbacks", false, "Purge only feedbacks")
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
		!p.prompts && !p.tools && !p.functions && !p.memories && !p.feedbacks

	// Count items
	counts := make(map[string]int)
	var err error

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
		fmt.Println("\n[Dry Run] Would delete the following:")
		for resource, count := range counts {
			if count > 0 {
				fmt.Printf("  - %s: %d items\n", resource, count)
			}
		}
		fmt.Printf("\nTotal: %d items\n", total)
		fmt.Println("\nRun with --force to actually delete these items.")
		return nil
	}

	// Force mode - show warning and ask for confirmation
	fmt.Println("\n⚠️  WARNING: This will permanently delete the following:")
	for resource, count := range counts {
		if count > 0 {
			fmt.Printf("  - %s: %d items\n", resource, count)
		}
	}
	fmt.Printf("\nTotal: %d items\n", total)
	fmt.Print("\nAre you sure you want to continue? Type 'yes' to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" {
		logrus.Info("Operation cancelled")
		return nil
	}

	// Perform deletions
	fmt.Println()
	if err := p.performDeletions(client, counts); err != nil {
		return err
	}

	logrus.Infof("Successfully deleted %d items", total)
	return nil
}

func (p *PurgePlugin) performDeletions(client *openwebui.Client, counts map[string]int) error {
	purgeAll := !p.chats && !p.files && !p.models && !p.knowledge &&
		!p.prompts && !p.tools && !p.functions && !p.memories && !p.feedbacks

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

	// Delete feedbacks (if supported by API)
	if purgeAll || p.feedbacks {
		logrus.Info("Deleting feedbacks...")
		if err := client.DeleteAllFeedbacks(); err != nil {
			logrus.Warnf("Failed to delete feedbacks: %v (may not be supported)", err)
		} else {
			logrus.Info("✓ Feedbacks deleted")
		}
	}

	return nil
}
