package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

const (
	PBKDF2Iterations = 100000
	KeyLength        = 32
	SaltLength       = 32
)

// SecretsFile represents the encrypted secrets storage file
type SecretsFile struct {
	Version  int              `json:"version"`
	Secrets  map[string]Secret `json:"secrets"`
	Checksum string           `json:"checksum"`
	Salt     string           `json:"salt,omitempty"`
}

// Secret represents a stored secret
type Secret struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Encrypted bool   `json:"encrypted"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func main() {
	filePath := flag.String("file", "", "Path to secrets file (required)")
	masterKey := flag.String("key", "", "Master key (required)")
	dryRun := flag.Bool("dry-run", false, "Validate without migrating")
	flag.Parse()

	if *filePath == "" || *masterKey == "" {
		fmt.Println("Usage: secrets-migrate -file <path> -key <master-key> [-dry-run]")
		os.Exit(1)
	}

	if err := migrate(*filePath, *masterKey, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func migrate(filePath, masterKey string, dryRun bool) error {
	// Check if file exists
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("secrets file does not exist: %s", filePath)
	}

	// Load and decrypt existing file
	oldFile, err := loadSecretsFile(filePath, masterKey)
	if err != nil {
		return fmt.Errorf("failed to load secrets file: %w", err)
	}

	// Detect version
	fmt.Printf("Detected version: %d\n", oldFile.Version)

	if oldFile.Version == 2 {
		fmt.Println("File is already at version 2, no migration needed")
		return nil
	}

	if oldFile.Version != 1 {
		return fmt.Errorf("unsupported version: %d", oldFile.Version)
	}

	fmt.Printf("Found %d secret(s)\n", len(oldFile.Secrets))

	if dryRun {
		fmt.Println("Dry run: would migrate to version 2")
		return nil
	}

	// Create backup
	backupPath := filePath + ".backup.v1"
	if err := backupFile(filePath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	fmt.Printf("Created backup: %s\n", backupPath)

	// Generate new salt
	salt := make([]byte, SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive new key using PBKDF2
	newKey := deriveKey(masterKey, salt)

	// Update file to version 2
	oldFile.Version = 2
	oldFile.Salt = base64.StdEncoding.EncodeToString(salt)

	// Save with new encryption
	if err := saveSecretsFile(filePath, newKey, oldFile); err != nil {
		return fmt.Errorf("failed to save migrated file: %w", err)
	}

	fmt.Println("Migration complete!")
	return nil
}

func loadSecretsFile(filePath, masterKey string) (*SecretsFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Decode base64
	combined, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	// Derive key using SHA-256 (v1)
	key := sha256.Sum256([]byte(masterKey))

	// Decrypt
	decrypted, err := decrypt(key[:], combined)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Parse checksum|json
	parts := bytes.Split(decrypted, []byte("|"))
	if len(parts) != 2 {
		return nil, errors.New("invalid file format")
	}

	// Verify checksum
	checksum := sha256.Sum256(parts[1])
	expectedChecksum := base64.StdEncoding.EncodeToString(checksum[:])
	if string(parts[0]) != expectedChecksum {
		return nil, errors.New("checksum mismatch")
	}

	// Parse JSON
	var sf SecretsFile
	if err := json.Unmarshal(parts[1], &sf); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &sf, nil
}

func saveSecretsFile(filePath string, key []byte, sf *SecretsFile) error {
	// Marshal JSON
	data, err := json.Marshal(sf)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	// Calculate checksum
	checksum := sha256.Sum256(data)
	checksumStr := base64.StdEncoding.EncodeToString(checksum[:])

	// Combine checksum and data
	payload := bytes.Join([][]byte{[]byte(checksumStr), data}, []byte("|"))

	// Encrypt
	combined, err := encrypt(key, payload)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	// Write to temp file
	tempPath := filePath + ".tmp"
	fileData := []byte(base64.StdEncoding.EncodeToString(combined))

	if err := os.WriteFile(tempPath, fileData, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename: %w", err)
	}

	return nil
}

func backupFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0600)
}

func deriveKey(masterKey string, salt []byte) []byte {
	return pbkdf2.Key([]byte(masterKey), salt, PBKDF2Iterations, KeyLength, sha256.New)
}

func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
