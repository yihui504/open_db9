package secrets

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

var (
	// ErrSecretNotFound is returned when a secret is not found
	ErrSecretNotFound = errors.New("secret not found")
	// ErrInvalidKeyFormat is returned when the key format is invalid
	ErrInvalidKeyFormat = errors.New("invalid key format")
	// ErrSecretsFileLocked is returned when the secrets file is locked
	ErrSecretsFileLocked = errors.New("secrets file is locked")
	// ErrInvalidVersion is returned when the secrets file version is invalid
	ErrInvalidVersion = errors.New("invalid secrets file version")
)

const (
	// CurrentVersion is the current secrets file format version
	CurrentVersion = 2
	// PBKDF2Iterations is the number of iterations for PBKDF2
	PBKDF2Iterations = 100000
	// KeyLength is the derived key length in bytes
	KeyLength = 32
	// SaltLength is the salt length in bytes
	SaltLength = 32
)

// Secret represents a stored secret
type Secret struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Encrypted bool   `json:"encrypted"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// SecretsFile represents the encrypted secrets storage file
type SecretsFile struct {
	Version  int              `json:"version"`
	Secrets  map[string]Secret `json:"secrets"`
	Checksum string           `json:"checksum"`
	Salt     string           `json:"salt,omitempty"` // Salt for PBKDF2 (v2+)
	mu       sync.RWMutex
}

// Manager handles encrypted secrets storage
type Manager struct {
	filePath string
	key      []byte
	file     *SecretsFile
}

// NewManager creates a new secrets manager
func NewManager(filePath string, masterKey string) (*Manager, error) {
	if filePath == "" {
		return nil, errors.New("file path cannot be empty")
	}

	if masterKey == "" {
		return nil, errors.New("master key cannot be empty")
	}

	m := &Manager{
		filePath: filePath,
		file: &SecretsFile{
			Version: CurrentVersion,
			Secrets: make(map[string]Secret),
		},
	}

	// Load existing secrets file if it exists
	if _, err := os.Stat(filePath); err == nil {
		if err := m.Load(); err != nil {
			return nil, fmt.Errorf("failed to load secrets file: %w", err)
		}
		// Derive key using existing salt (v2) or default (v1)
		if m.file.Version == 2 && m.file.Salt != "" {
			salt, err := base64.StdEncoding.DecodeString(m.file.Salt)
			if err != nil {
				return nil, fmt.Errorf("failed to decode salt: %w", err)
			}
			m.key = deriveKey(masterKey, salt)
		} else {
			// Version 1: use SHA-256 for backward compatibility
			key := sha256.Sum256([]byte(masterKey))
			m.key = key[:]
		}
	} else {
		// New file: generate salt and derive key
		salt := make([]byte, SaltLength)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}
		m.file.Salt = base64.StdEncoding.EncodeToString(salt)
		m.key = deriveKey(masterKey, salt)
	}

	return m, nil
}

// deriveKey derives a 32-byte key from the master key using PBKDF2
func deriveKey(masterKey string, salt []byte) []byte {
	return pbkdf2.Key([]byte(masterKey), salt, PBKDF2Iterations, KeyLength, sha256.New)
}

// Load loads secrets from the encrypted file
func (m *Manager) Load() error {
	m.file.mu.Lock()
	defer m.file.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to read secrets file: %w", err)
	}

	// File format: base64(nonce+ciphertext)
	combined, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return fmt.Errorf("failed to decode secrets file: %w", err)
	}

	// Decrypt the data (combined nonce + ciphertext)
	decrypted, err := decrypt(m.key, combined)
	if err != nil {
		return fmt.Errorf("failed to decrypt secrets: %w", err)
	}

	// Verify checksum
	storedParts := bytes.Split(decrypted, []byte("|"))
	if len(storedParts) != 2 {
		return errors.New("invalid decrypted data format")
	}

	checksum := sha256.Sum256(storedParts[1])
	expectedChecksum := base64.StdEncoding.EncodeToString(checksum[:])

	if string(storedParts[0]) != expectedChecksum {
		return errors.New("secrets file checksum mismatch - file may be corrupted")
	}

	// Parse JSON
	var sf SecretsFile
	if err := json.Unmarshal(storedParts[1], &sf); err != nil {
		return fmt.Errorf("failed to parse secrets JSON: %w", err)
	}

	m.file = &sf
	return nil
}

// Save saves secrets to the encrypted file
func (m *Manager) Save() error {
	m.file.mu.Lock()
	defer m.file.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.Marshal(m.file)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Calculate checksum
	checksum := sha256.Sum256(data)
	checksumStr := base64.StdEncoding.EncodeToString(checksum[:])

	// Combine checksum and data
	payload := bytes.Join([][]byte{[]byte(checksumStr), data}, []byte("|"))

	// Encrypt the payload
	combined, err := encrypt(m.key, payload)
	if err != nil {
		return fmt.Errorf("failed to encrypt secrets: %w", err)
	}

	// Write to file with restricted permissions
	tempPath := m.filePath + ".tmp"
	fileData := []byte(base64.StdEncoding.EncodeToString(combined))

	if err := os.WriteFile(tempPath, fileData, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, m.filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Set stores a secret (encrypted by default)
func (m *Manager) Set(key, value string, encrypted bool) error {
	m.file.mu.Lock()
	defer m.file.mu.Unlock()

	now := time.Now().Unix()

	secret := Secret{
		Key:       key,
		Value:     value,
		Encrypted: encrypted,
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.file.Secrets[key] = secret
	return nil
}

// Get retrieves a secret
func (m *Manager) Get(key string) (string, error) {
	m.file.mu.RLock()
	defer m.file.mu.RUnlock()

	secret, exists := m.file.Secrets[key]
	if !exists {
		return "", ErrSecretNotFound
	}

	if secret.Encrypted {
		// The value is stored encrypted, decrypt it
		// Format: base64(nonce+ciphertext)
		combined, err := base64.StdEncoding.DecodeString(secret.Value)
		if err != nil {
			return "", fmt.Errorf("failed to decode encrypted secret: %w", err)
		}

		decrypted, err := decrypt(m.key, combined)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt secret: %w", err)
		}

		return string(decrypted), nil
	}

	return secret.Value, nil
}

// SetEncrypted stores an encrypted secret
func (m *Manager) SetEncrypted(key, value string) error {
	// Encrypt the value before storing
	combined, err := encrypt(m.key, []byte(value))
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Store as base64(nonce+ciphertext)
	encryptedValue := base64.StdEncoding.EncodeToString(combined)

	return m.Set(key, encryptedValue, true)
}

// Delete removes a secret
func (m *Manager) Delete(key string) error {
	m.file.mu.Lock()
	defer m.file.mu.Unlock()

	if _, exists := m.file.Secrets[key]; !exists {
		return ErrSecretNotFound
	}

	delete(m.file.Secrets, key)
	return nil
}

// List returns all secret keys (without values)
func (m *Manager) List() []string {
	m.file.mu.RLock()
	defer m.file.mu.RUnlock()

	keys := make([]string, 0, len(m.file.Secrets))
	for key := range m.file.Secrets {
		keys = append(keys, key)
	}
	return keys
}

// encrypt encrypts data using AES-GCM
// Returns: nonce + ciphertext combined (standard GCM format)
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

	// Seal appends ciphertext to nonce and returns the combined result
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decrypt decrypts data using AES-GCM
// Expects: nonce + ciphertext combined (standard GCM format)
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

	// Split nonce and ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
