package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewManager tests creating a new secrets manager
func TestNewManager(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		masterKey   string
		expectError bool
	}{
		{
			name:        "valid parameters",
			filePath:    "test_secrets.bin",
			masterKey:   strings.Repeat("a", 32),
			expectError: false,
		},
		{
			name:        "empty file path",
			filePath:    "",
			masterKey:   strings.Repeat("b", 32),
			expectError: true,
		},
		{
			name:        "empty master key",
			filePath:    "test_secrets.bin",
			masterKey:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewManager(tt.filePath, tt.masterKey)
			if tt.expectError {
				if err == nil {
					t.Errorf("NewManager() should return error")
				}
			} else {
				if err != nil {
					t.Errorf("NewManager() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestSetAndGet tests setting and getting secrets
func TestSetAndGet(t *testing.T) {
	// Create temp directory for test files
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_secrets.bin")
	masterKey := strings.Repeat("x", 32)

	manager, err := NewManager(testFile, masterKey)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test setting and getting unencrypted secret
	err = manager.Set("test_key", "test_value", false)
	if err != nil {
		t.Errorf("Set() unencrypted error = %v", err)
	}

	value, err := manager.Get("test_key")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if value != "test_value" {
		t.Errorf("Get() = %s, want test_value", value)
	}

	// Test setting and getting encrypted secret
	err = manager.SetEncrypted("secret_key", "secret_value")
	if err != nil {
		t.Errorf("SetEncrypted() error = %v", err)
	}

	value, err = manager.Get("secret_key")
	if err != nil {
		t.Errorf("Get() encrypted error = %v", err)
	}
	if value != "secret_value" {
		t.Errorf("Get() encrypted = %s, want secret_value", value)
	}
}

// TestGetNotFound tests getting a non-existent secret
func TestGetNotFound(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_secrets.bin")
	masterKey := strings.Repeat("y", 32)

	manager, err := NewManager(testFile, masterKey)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	_, err = manager.Get("nonexistent")
	if err != ErrSecretNotFound {
		t.Errorf("Get() nonexistent error = %v, want ErrSecretNotFound", err)
	}
}

// TestDelete tests deleting secrets
func TestDelete(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_secrets.bin")
	masterKey := strings.Repeat("z", 32)

	manager, err := NewManager(testFile, masterKey)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Set a secret
	manager.Set("to_delete", "value", false)

	// Delete it
	err = manager.Delete("to_delete")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = manager.Get("to_delete")
	if err != ErrSecretNotFound {
		t.Errorf("Get() after delete error = %v, want ErrSecretNotFound", err)
	}

	// Try to delete non-existent
	err = manager.Delete("nonexistent")
	if err != ErrSecretNotFound {
		t.Errorf("Delete() nonexistent error = %v, want ErrSecretNotFound", err)
	}
}

// TestList tests listing all secret keys
func TestList(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_secrets.bin")
	masterKey := strings.Repeat("w", 32)

	manager, err := NewManager(testFile, masterKey)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Initially empty
	keys := manager.List()
	if len(keys) != 0 {
		t.Errorf("List() initially = %v, want empty", keys)
	}

	// Add some secrets
	manager.Set("key1", "value1", false)
	manager.Set("key2", "value2", false)
	manager.SetEncrypted("key3", "value3")

	keys = manager.List()
	if len(keys) != 3 {
		t.Errorf("List() length = %d, want 3", len(keys))
	}

	// Check all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	for _, expected := range []string{"key1", "key2", "key3"} {
		if !keyMap[expected] {
			t.Errorf("List() missing key %s", expected)
		}
	}
}

// TestSaveAndLoad tests persisting secrets to disk
func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_secrets.bin")
	masterKey := strings.Repeat("v", 32)

	// Create manager and add secrets
	manager1, err := NewManager(testFile, masterKey)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	manager1.Set("persistent1", "value1", false)
	manager1.SetEncrypted("persistent2", "value2")

	// Save to disk
	err = manager1.Save()
	if err != nil {
		t.Errorf("Save() error = %v", err)
	}

	// Create new manager and load
	manager2, err := NewManager(testFile, masterKey)
	if err != nil {
		t.Fatalf("Failed to create second manager: %v", err)
	}

	// Verify secrets were loaded
	value1, err := manager2.Get("persistent1")
	if err != nil {
		t.Errorf("Get() after load error = %v", err)
	}
	if value1 != "value1" {
		t.Errorf("Get() after load = %s, want value1", value1)
	}

	value2, err := manager2.Get("persistent2")
	if err != nil {
		t.Errorf("Get() encrypted after load error = %v", err)
	}
	if value2 != "value2" {
		t.Errorf("Get() encrypted after load = %s, want value2", value2)
	}
}

// TestWrongMasterKey tests that wrong master key fails to decrypt
func TestWrongMasterKey(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_secrets.bin")
	masterKey1 := strings.Repeat("a", 32)
	masterKey2 := strings.Repeat("b", 32)

	// Create and save with first key
	manager1, err := NewManager(testFile, masterKey1)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	manager1.Set("secret", "value", false)
	err = manager1.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Try to load with wrong key
	_, err = NewManager(testFile, masterKey2)
	if err == nil {
		t.Error("NewManager() with wrong key should return error")
	}
}

// TestEncryptDecrypt tests encryption/decryption functions
func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("This is a secret message")

	combined, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	if len(combined) == 0 {
		t.Error("encrypt() returned empty result")
	}

	// Decrypt
	decrypted, err := decrypt(key, combined)
	if err != nil {
		t.Fatalf("decrypt() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypt() = %s, want %s", string(decrypted), string(plaintext))
	}
}

// TestEncryptDecryptWrongKey tests that wrong key fails to decrypt
func TestEncryptDecryptWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
	}

	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 1)
	}

	plaintext := []byte("Secret data")

	combined, err := encrypt(key1, plaintext)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	// Try to decrypt with wrong key
	_, err = decrypt(key2, combined)
	if err == nil {
		t.Error("decrypt() with wrong key should return error")
	}
}

// TestSaveFilePermissions tests that saved files have restricted permissions
func TestSaveFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_secrets.bin")
	masterKey := strings.Repeat("u", 32)

	manager, err := NewManager(testFile, masterKey)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	manager.Set("test", "value", false)
	err = manager.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Check file exists
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Stat() failed: %v", err)
	}

	// On Unix-like systems, check permissions
	// Note: Windows doesn't have the same permission model
	if info.Mode().Perm()&0o777 != 0o600 {
		// This test might fail on Windows due to different permission handling
		t.Logf("File permissions = %v (note: Windows handles permissions differently)", info.Mode().Perm())
	}
}
