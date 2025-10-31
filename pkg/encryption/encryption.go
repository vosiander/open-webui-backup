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
	Passphrase string   // Passphrase for symmetric decryption
	Identities []string // Raw age identity content as strings
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
	} else if len(opts.Identities) > 0 {
		// Identity string decryption
		for i, identityContent := range opts.Identities {
			ids, err := age.ParseIdentities(strings.NewReader(identityContent))
			if err != nil {
				return fmt.Errorf("failed to parse identity %d: %w", i+1, err)
			}
			identities = append(identities, ids...)
		}
		logrus.Debugf("Using identity decryption with %d identity string(s)", len(opts.Identities))
	} else {
		return fmt.Errorf("either passphrase or identities must be provided")
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
	// Check by reading the file header first (more reliable than extension)
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first few bytes to check for age armor header
	header := make([]byte, 35)
	n, err := file.Read(header)
	if err != nil || n < 10 {
		return false
	}

	// Check for age armor header (most reliable indicator)
	if strings.HasPrefix(string(header), "-----BEGIN AGE ENCRYPTED FILE-----") {
		return true
	}

	// If content doesn't match, file is not encrypted regardless of extension
	// (This handles cases where .age files are actually plain ZIP files)
	return false
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
// identities should contain the raw age identity content as strings
func DecryptFileWithIdentities(inputPath, outputPath string, identities []string) error {
	if len(identities) == 0 {
		return fmt.Errorf("at least one identity is required")
	}
	return DecryptFile(inputPath, outputPath, &DecryptOptions{
		Identities: identities,
	})
}

// GenerateIdentity generates a new X25519 age identity
func GenerateIdentity() (*age.X25519Identity, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("failed to generate X25519 identity: %w", err)
	}
	logrus.Debug("Generated new age X25519 identity")
	return identity, nil
}
