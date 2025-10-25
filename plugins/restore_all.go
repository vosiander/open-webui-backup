package plugins

import (
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vosiander/open-webui-backup/pkg/config"
	"github.com/vosiander/open-webui-backup/pkg/encryption"
	"github.com/vosiander/open-webui-backup/pkg/openwebui"
	"github.com/vosiander/open-webui-backup/pkg/restore"
)

type RestoreAllPlugin struct {
	dir             string
	overwrite       bool
	decrypt         bool
	decryptIdentity []string
}

func NewRestoreAllPlugin() *RestoreAllPlugin {
	return &RestoreAllPlugin{}
}

// Name returns the name of the plugin (used as command name)
func (p *RestoreAllPlugin) Name() string {
	return "restore-all"
}

// Description returns a short description of the plugin
func (p *RestoreAllPlugin) Description() string {
	return "Restore all knowledge bases and models to Open WebUI from a backup directory"
}

func (p *RestoreAllPlugin) SetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&p.dir, "dir", "f", "", "Filename of the backup")
	cmd.Flags().BoolVarP(&p.overwrite, "overwrite", "w", false, "Overwrite existing models and files")
	cmd.Flags().BoolVar(&p.decrypt, "decrypt", false, "Decrypt backup with passphrase (interactive)")
	cmd.Flags().StringSliceVar(&p.decryptIdentity, "decrypt-identity", nil, "Decrypt backup with age identity dir(s)")
	cmd.MarkFlagRequired("dir")
}

// Execute runs the plugin with the given configuration
func (p *RestoreAllPlugin) Execute(cfg *config.Config) error {
	logrus.Info("Starting full restore...")

	if cfg.OpenWebUIAPIKey == "" {
		logrus.Fatalf("OPEN_WEBUI_API_KEY environment variable is required")
	}

	if p.dir == "" {
		logrus.Fatalf("directory is required (use --dir flag)")
	}

	restorePath := p.dir
	var tempFile string

	// Check if dir is encrypted and handle decryption
	if encryption.IsEncrypted(p.dir) || p.decrypt || len(p.decryptIdentity) > 0 {
		logrus.Info("Encrypted backup detected, decrypting...")

		// Create temporary dir for decrypted content
		tmpFile, err := ioutil.TempFile("", "owui-restore-*.zip")
		if err != nil {
			logrus.Fatalf("Failed to create temporary dir: %v", err)
		}
		tempFile = tmpFile.Name()
		tmpFile.Close()

		// Defer cleanup of temporary dir
		defer func() {
			if tempFile != "" {
				os.Remove(tempFile)
			}
		}()

		// Prepare decryption options
		var passphrase string
		var identityFiles []string

		if len(p.decryptIdentity) > 0 {
			// Identity dir mode
			identityFiles = p.decryptIdentity
		} else {
			// Passphrase mode - check env first, then prompt
			passphrase = encryption.GetPassphraseFromEnv()
			if passphrase == "" {
				var err error
				passphrase, err = encryption.ReadPassphraseForDecryption()
				if err != nil {
					logrus.Fatalf("Failed to read passphrase: %v", err)
				}
			}
		}

		// Decrypt the backup
		opts := &encryption.DecryptOptions{
			Passphrase:    passphrase,
			IdentityFiles: identityFiles,
		}

		if err := encryption.DecryptFile(p.dir, tempFile, opts); err != nil {
			logrus.Fatalf("Failed to decrypt backup: %v", err)
		}

		logrus.Info("Backup decrypted successfully")
		restorePath = tempFile
	}

	client := openwebui.NewClient(cfg.OpenWebUIURL, cfg.OpenWebUIAPIKey)

	if err := restore.RestoreAll(client, restorePath, p.overwrite); err != nil {
		logrus.Fatalf("Failed to restore: %v", err)
	}

	logrus.Info("Full restore completed successfully")
	return nil
}
