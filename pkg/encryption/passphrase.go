package encryption

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// ReadPassphrase reads a passphrase from stdin without echoing
func ReadPassphrase(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)

	// Get the file descriptor for stdin
	fd := int(syscall.Stdin)

	// Check if stdin is a terminal
	if !term.IsTerminal(fd) {
		// Not a terminal, read from stdin normally (for piped input)
		reader := bufio.NewReader(os.Stdin)
		passphrase, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read passphrase: %w", err)
		}
		return strings.TrimSpace(passphrase), nil
	}

	// Terminal input - read without echo
	bytePassword, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr) // Print newline after password input

	if err != nil {
		return "", fmt.Errorf("failed to read passphrase: %w", err)
	}

	return string(bytePassword), nil
}

// ReadAndConfirmPassphrase reads a passphrase twice and ensures they match
func ReadAndConfirmPassphrase() (string, error) {
	passphrase, err := ReadPassphrase("Enter passphrase: ")
	if err != nil {
		return "", err
	}

	// Validate passphrase
	if err := ValidatePassphrase(passphrase); err != nil {
		return "", err
	}

	confirmation, err := ReadPassphrase("Confirm passphrase: ")
	if err != nil {
		return "", err
	}

	if passphrase != confirmation {
		return "", fmt.Errorf("passphrases do not match")
	}

	return passphrase, nil
}

// ReadPassphraseForDecryption reads a passphrase for decryption
func ReadPassphraseForDecryption() (string, error) {
	return ReadPassphrase("Enter passphrase: ")
}

// GetPassphraseFromEnv retrieves passphrase from environment variable
func GetPassphraseFromEnv() string {
	return os.Getenv("OWUI_ENCRYPT_PASSPHRASE")
}

// GetRecipientsFromEnv retrieves recipients from environment variable
func GetRecipientsFromEnv() []string {
	recipientStr := os.Getenv("OWUI_ENCRYPT_RECIPIENT")
	if recipientStr == "" {
		return nil
	}

	// Support comma-separated recipients
	recipients := strings.Split(recipientStr, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	return recipients
}

// GetIdentityFilesFromEnv retrieves identity file paths from environment variable
func GetIdentityFilesFromEnv() []string {
	identityStr := os.Getenv("OWUI_DECRYPT_IDENTITY")
	if identityStr == "" {
		return nil
	}

	// Support comma-separated identity files
	files := strings.Split(identityStr, ",")
	for i := range files {
		files[i] = strings.TrimSpace(files[i])
	}

	return files
}

// Environment variable constants
const (
	EnvEncryptRecipient = "OWUI_ENCRYPTED_RECIPIENT"
	EnvDecryptIdentity  = "OWUI_DECRYPT_IDENTITY"
)

// GetDecryptIdentityFilesFromEnvOrFlag returns decryption identity files from flag or environment variable
// Priority: flag values > environment variable
// Returns error if no identity files are provided from either source
func GetDecryptIdentityFilesFromEnvOrFlag(flagIdentities []string) ([]string, error) {
	// If flag is provided, use it
	if len(flagIdentities) > 0 {
		return flagIdentities, nil
	}

	// Otherwise, try environment variable
	identityStr := os.Getenv(EnvDecryptIdentity)
	if identityStr == "" {
		return nil, fmt.Errorf("no decryption identity files provided: use --decrypt-identity flag or set %s environment variable", EnvDecryptIdentity)
	}

	// Support comma-separated identity files
	files := strings.Split(identityStr, ",")
	for i := range files {
		files[i] = strings.TrimSpace(files[i])
	}

	// Filter out empty strings
	validFiles := []string{}
	for _, f := range files {
		if f != "" {
			validFiles = append(validFiles, f)
		}
	}

	if len(validFiles) == 0 {
		return nil, fmt.Errorf("no valid identity files found in %s environment variable", EnvDecryptIdentity)
	}

	return validFiles, nil
}
