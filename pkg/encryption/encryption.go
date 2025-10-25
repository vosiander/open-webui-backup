package encryption

import (
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/sirupsen/logrus"
)

// EncryptOptions contains encryption configuration
type EncryptOptions struct {
	Passphrase string   // Passphrase for symmetric encryption
	Recipients []string // Age recipient public keys for asymmetric encryption
}

// DecryptOptions contains decryption configuration
type DecryptOptions struct {
	Passphrase    string   // Passphrase for symmetric decryption
	IdentityFiles []string // Paths to age identity files (private keys)
}

// EncryptFile encrypts a file using age encryption
func EncryptFile(inputPath, outputPath string, opts *EncryptOptions) error {
	if opts == nil {
		return fmt.Errorf("encryption options are required")
	}

	// Read input file
	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Create output file
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	var w io.Writer
	var recipients []age.Recipient

	// Determine encryption mode
	if opts.Passphrase != "" {
		// Passphrase-based encryption
		recipient, err := age.NewScryptRecipient(opts.Passphrase)
		if err != nil {
			return fmt.Errorf("failed to create passphrase recipient: %w", err)
		}
		recipients = []age.Recipient{recipient}
		logrus.Debug("Using passphrase-based encryption")
	} else if len(opts.Recipients) > 0 {
		// Public key encryption
		for _, recipientStr := range opts.Recipients {
			recipient, err := age.ParseX25519Recipient(recipientStr)
			if err != nil {
				return fmt.Errorf("failed to parse recipient %s: %w", recipientStr, err)
			}
			recipients = append(recipients, recipient)
		}
		logrus.Debugf("Using public key encryption with %d recipient(s)", len(recipients))
	} else {
		return fmt.Errorf("either passphrase or recipients must be provided")
	}

	// Create age writer with armor (ASCII output)
	armorWriter := armor.NewWriter(out)
	defer armorWriter.Close()

	w, err = age.Encrypt(armorWriter, recipients...)
	if err != nil {
		return fmt.Errorf("failed to create encryption writer: %w", err)
	}

	// Write encrypted data
	if _, err := w.Write(inputData); err != nil {
		return fmt.Errorf("failed to write encrypted data: %w", err)
	}

	// Close the writer to finalize encryption
	if closer, ok := w.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("failed to finalize encryption: %w", err)
		}
	}

	logrus.Infof("File encrypted: %s -> %s", inputPath, outputPath)
	return nil
}

// DecryptFile decrypts an age-encrypted file
func DecryptFile(inputPath, outputPath string, opts *DecryptOptions) error {
	if opts == nil {
		return fmt.Errorf("decryption options are required")
	}

	// Read encrypted file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open encrypted file: %w", err)
	}
	defer inputFile.Close()

	var identities []age.Identity

	// Determine decryption mode
	if opts.Passphrase != "" {
		// Passphrase-based decryption
		identity, err := age.NewScryptIdentity(opts.Passphrase)
		if err != nil {
			return fmt.Errorf("failed to create passphrase identity: %w", err)
		}
		identities = []age.Identity{identity}
		logrus.Debug("Using passphrase-based decryption")
	} else if len(opts.IdentityFiles) > 0 {
		// Identity file decryption
		for _, identityPath := range opts.IdentityFiles {
			idFile, err := os.Open(identityPath)
			if err != nil {
				return fmt.Errorf("failed to open identity file %s: %w", identityPath, err)
			}
			defer idFile.Close()

			ids, err := age.ParseIdentities(idFile)
			if err != nil {
				return fmt.Errorf("failed to parse identity file %s: %w", identityPath, err)
			}
			identities = append(identities, ids...)
		}
		logrus.Debugf("Using identity file decryption with %d identity file(s)", len(opts.IdentityFiles))
	} else {
		return fmt.Errorf("either passphrase or identity files must be provided")
	}

	// Try to read with armor decoder first (ASCII format)
	armorReader := armor.NewReader(inputFile)

	// Decrypt
	r, err := age.Decrypt(armorReader, identities...)
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	// Read decrypted data
	decryptedData, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read decrypted data: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(outputPath, decryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	logrus.Infof("File decrypted: %s -> %s", inputPath, outputPath)
	return nil
}

// IsEncrypted checks if a file appears to be age-encrypted
func IsEncrypted(path string) bool {
	// Check by file extension
	if strings.HasSuffix(path, ".age") {
		return true
	}

	// Check by reading the file header
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first few bytes to check for age armor header
	header := make([]byte, 30)
	n, err := file.Read(header)
	if err != nil || n < 10 {
		return false
	}

	// Check for age armor header
	return strings.HasPrefix(string(header), "-----BEGIN AGE ENCRYPTED FILE-----")
}

// ValidatePassphrase checks if a passphrase meets minimum requirements
func ValidatePassphrase(passphrase string) error {
	if len(passphrase) < 12 {
		return fmt.Errorf("passphrase must be at least 12 characters long")
	}
	return nil
}

// EncryptFileWithPassphrase is a convenience function for passphrase encryption
func EncryptFileWithPassphrase(inputPath, outputPath, passphrase string) error {
	if err := ValidatePassphrase(passphrase); err != nil {
		return err
	}
	return EncryptFile(inputPath, outputPath, &EncryptOptions{
		Passphrase: passphrase,
	})
}

// DecryptFileWithPassphrase is a convenience function for passphrase decryption
func DecryptFileWithPassphrase(inputPath, outputPath, passphrase string) error {
	return DecryptFile(inputPath, outputPath, &DecryptOptions{
		Passphrase: passphrase,
	})
}

// EncryptFileWithRecipients is a convenience function for public key encryption
func EncryptFileWithRecipients(inputPath, outputPath string, recipients []string) error {
	if len(recipients) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	return EncryptFile(inputPath, outputPath, &EncryptOptions{
		Recipients: recipients,
	})
}

// DecryptFileWithIdentities is a convenience function for identity-based decryption
func DecryptFileWithIdentities(inputPath, outputPath string, identityFiles []string) error {
	if len(identityFiles) == 0 {
		return fmt.Errorf("at least one identity file is required")
	}
	return DecryptFile(inputPath, outputPath, &DecryptOptions{
		IdentityFiles: identityFiles,
	})
}
