package encryption

import (
	"io/ioutil"
	"os"
	"path/filepath"

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

	if len(identityFiles) > 0 {
		// Identity file mode
		logrus.Debug("Using identity file decryption")
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
		Passphrase:    passphrase,
		IdentityFiles: identityFiles,
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
