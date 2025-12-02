package encryption

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// EncryptBackupFile encrypts a backup file if encryption options are provided
// Returns the path to the encrypted file (or original if no encryption)
func EncryptBackupFile(backupPath string, encrypt bool, recipients []string) (string, error) {
	if !encrypt && len(recipients) == 0 {
		return backupPath, nil // No encryption requested
	}

	encryptedPath := backupPath + ".age"

	// Prepare encryption options
	var passphrase string

	if len(recipients) > 0 {
		// Public key mode
		logrus.Info("Encrypting backup with public key(s)...")
	} else {
		// Passphrase mode - check env first, then prompt
		passphrase = GetPassphraseFromEnv()
		if passphrase == "" {
			var err error
			passphrase, err = ReadAndConfirmPassphrase()
			if err != nil {
				return "", err
			}
		}
		logrus.Info("Encrypting backup with passphrase...")
	}

	// Encrypt the backup
	opts := &EncryptOptions{
		Passphrase: passphrase,
		Recipients: recipients,
	}

	if err := EncryptFile(backupPath, encryptedPath, opts); err != nil {
		return "", err
	}

	// Remove unencrypted backup
	if err := os.Remove(backupPath); err != nil {
		logrus.Warnf("Failed to remove unencrypted backup: %v", err)
	}

	logrus.Infof("Backup encrypted: %s", filepath.Base(encryptedPath))
	return encryptedPath, nil
}

// DecryptBackupFile decrypts a backup file if needed
// Returns the path to the decrypted file (temp file or original if not encrypted)
// The caller is responsible for cleaning up the temp file if one was created
func DecryptBackupFile(backupPath string, decrypt bool, identityFiles []string) (string, bool, error) {
	// Check if decryption is needed
	if !IsEncrypted(backupPath) && !decrypt && len(identityFiles) == 0 {
		return backupPath, false, nil // No decryption needed
	}

	if !IsEncrypted(backupPath) {
		logrus.Warn("Decryption flags provided but file is not encrypted")
		return backupPath, false, nil
	}

	logrus.Info("Encrypted backup detected, decrypting...")

	// Create temporary file for decrypted content
	tmpFile, err := ioutil.TempFile("", "owui-restore-*.zip")
	if err != nil {
		return "", false, err
	}
	tempPath := tmpFile.Name()
	tmpFile.Close()

	// Prepare decryption options
	var passphrase string
	var identities []string

	if len(identityFiles) > 0 {
		// Identity file mode - read file contents
		logrus.Debug("Using identity file decryption")
		for _, identityFile := range identityFiles {
			content, err := os.ReadFile(identityFile)
			if err != nil {
				os.Remove(tempPath)
				return "", false, fmt.Errorf("failed to read identity file %s: %w", identityFile, err)
			}
			identities = append(identities, string(content))
		}
	} else {
		// Passphrase mode - check env first, then prompt
		passphrase = GetPassphraseFromEnv()
		if passphrase == "" {
			var err error
			passphrase, err = ReadPassphraseForDecryption()
			if err != nil {
				os.Remove(tempPath)
				return "", false, err
			}
		}
	}

	// Decrypt the backup
	opts := &DecryptOptions{
		Passphrase: passphrase,
		Identities: identities,
	}

	if err := DecryptFile(backupPath, tempPath, opts); err != nil {
		os.Remove(tempPath)
		return "", false, err
	}

	logrus.Info("Backup decrypted successfully")
	return tempPath, true, nil
}

// FindLatestBackup finds the most recent backup file matching the pattern
func FindLatestBackup(dir, pattern string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(files) == 0 {
		return "", err
	}

	// Return the most recent file (last in sorted list)
	return files[len(files)-1], nil
}

// GetDecryptIdentitiesFromEnvOrFlag gets decryption identities from flag or environment
// Returns slice of identity strings (file contents)
func GetDecryptIdentitiesFromEnvOrFlag(identityFiles []string) ([]string, error) {
	var identities []string

	// If identity files provided via flag, read them
	if len(identityFiles) > 0 {
		for _, identityFile := range identityFiles {
			content, err := os.ReadFile(identityFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read identity file %s: %w", identityFile, err)
			}
			identities = append(identities, string(content))
		}
		return identities, nil
	}

	// Check environment variable
	envIdentity := os.Getenv("OWUI_DECRYPTION_IDENTITY")
	if envIdentity != "" {
		// Check if it's a file path or direct identity content
		if _, err := os.Stat(envIdentity); err == nil {
			// It's a file path
			content, err := os.ReadFile(envIdentity)
			if err != nil {
				return nil, fmt.Errorf("failed to read identity file from OWUI_DECRYPTION_IDENTITY: %w", err)
			}
			identities = append(identities, string(content))
		} else {
			// Assume it's direct identity content
			identities = append(identities, envIdentity)
		}
		return identities, nil
	}

	return nil, fmt.Errorf("no decryption identity provided (use --decrypt-identity flag or OWUI_DECRYPTION_IDENTITY environment variable)")
}

// GetEncryptRecipientsFromEnvOrFlag gets encryption recipients from flag or environment
// Supports both file paths and direct recipient strings
// Returns slice of recipient strings (file contents or direct values)
func GetEncryptRecipientsFromEnvOrFlag(recipientInputs []string) ([]string, error) {
	var recipients []string

	// If recipients provided via flag, process them
	if len(recipientInputs) > 0 {
		for _, recipientInput := range recipientInputs {
			// Check if it's a file path or direct recipient string
			fileInfo, err := os.Stat(recipientInput)
			if err == nil && !fileInfo.IsDir() {
				// It's a file path - read the recipient from file
				content, err := os.ReadFile(recipientInput)
				if err != nil {
					return nil, fmt.Errorf("failed to read recipient file %s: %w", recipientInput, err)
				}
				recipients = append(recipients, strings.TrimSpace(string(content)))
			} else if err != nil && os.IsNotExist(err) {
				// File doesn't exist - check if it looks like a file path
				if strings.Contains(recipientInput, "/") || strings.Contains(recipientInput, "\\") || strings.HasSuffix(recipientInput, ".txt") {
					return nil, fmt.Errorf("recipient file not found: %s (use absolute path or ensure file exists)", recipientInput)
				}
				// Assume it's a direct recipient string
				recipients = append(recipients, recipientInput)
			} else if fileInfo != nil && fileInfo.IsDir() {
				return nil, fmt.Errorf("recipient path is a directory, not a file: %s", recipientInput)
			} else {
				// Other error accessing file, but try as recipient string anyway
				recipients = append(recipients, recipientInput)
			}
		}
		return recipients, nil
	}

	// Check environment variable
	envRecipient := os.Getenv("OWUI_ENCRYPTED_RECIPIENT")
	if envRecipient != "" {
		// Support comma-separated recipients
		recipientList := splitAndTrim(envRecipient)

		for _, recipientInput := range recipientList {
			// Check if it's a file path or direct recipient string
			if _, err := os.Stat(recipientInput); err == nil {
				// It's a file path - read the recipient from file
				content, err := os.ReadFile(recipientInput)
				if err != nil {
					return nil, fmt.Errorf("failed to read recipient file %s from OWUI_ENCRYPTED_RECIPIENT: %w", recipientInput, err)
				}
				recipients = append(recipients, string(content))
			} else {
				// Assume it's a direct recipient string
				recipients = append(recipients, recipientInput)
			}
		}

		if len(recipients) > 0 {
			return recipients, nil
		}
	}

	return nil, fmt.Errorf("no encryption recipients provided (use --encrypt-recipient flag or OWUI_ENCRYPTED_RECIPIENT environment variable)")
}

// splitAndTrim splits a comma-separated string and trims whitespace from each part
func splitAndTrim(s string) []string {
	parts := []string{}
	for _, part := range splitString(s, ',') {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString splits a string by a delimiter
func splitString(s string, delim rune) []string {
	var parts []string
	var current []rune

	for _, r := range s {
		if r == delim {
			parts = append(parts, string(current))
			current = []rune{}
		} else {
			current = append(current, r)
		}
	}
	parts = append(parts, string(current))
	return parts
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && isSpace(s[start]) {
		start++
	}

	// Trim trailing whitespace
	for end > start && isSpace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isSpace checks if a byte is whitespace
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
