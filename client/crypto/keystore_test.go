package crypto

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

func TestNewKeyStore(t *testing.T) {
	path := "/tmp/test-keystore.dat"
	ks := NewKeyStore(path)

	if ks.keystorePath != path {
		t.Errorf("Expected keystore path %s, got %s", path, ks.keystorePath)
	}

	if ks.globalKey != nil {
		t.Error("Expected global key to be nil initially")
	}
}

func TestKeyStoreInitialize(t *testing.T) {
	// Clear environment to ensure test isolation
	originalKey := os.Getenv("MARCHAT_GLOBAL_E2E_KEY")
	os.Unsetenv("MARCHAT_GLOBAL_E2E_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("MARCHAT_GLOBAL_E2E_KEY", originalKey)
		}
	}()

	// Create temporary directory for test
	tmpDir := t.TempDir()
	keystorePath := filepath.Join(tmpDir, "test-keystore.dat")

	ks := NewKeyStore(keystorePath)

	// Test initialization with new keystore
	err := ks.Initialize("test-passphrase")
	if err != nil {
		t.Fatalf("Failed to initialize keystore: %v", err)
	}

	// Check that global key was created
	globalKey := ks.GetGlobalKey()
	if globalKey == nil {
		t.Fatal("Expected global key to be created")
	}

	if len(globalKey.Key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(globalKey.Key))
	}

	if globalKey.KeyID == "" {
		t.Error("Expected KeyID to be set")
	}

	// Test that keystore file was created
	if _, err := os.Stat(keystorePath); os.IsNotExist(err) {
		t.Error("Expected keystore file to be created")
	}
}

func TestKeyStoreLoad(t *testing.T) {
	// Clear environment to ensure test isolation
	originalKey := os.Getenv("MARCHAT_GLOBAL_E2E_KEY")
	os.Unsetenv("MARCHAT_GLOBAL_E2E_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("MARCHAT_GLOBAL_E2E_KEY", originalKey)
		}
	}()

	// Create temporary directory for test
	tmpDir := t.TempDir()
	keystorePath := filepath.Join(tmpDir, "test-keystore.dat")

	// First, create a keystore
	ks1 := NewKeyStore(keystorePath)
	err := ks1.Initialize("test-passphrase")
	if err != nil {
		t.Fatalf("Failed to initialize keystore: %v", err)
	}

	originalGlobalKey := ks1.GetGlobalKey()
	if originalGlobalKey == nil {
		t.Fatal("Expected original key to be created")
	}

	// Now test loading it
	ks2 := NewKeyStore(keystorePath)
	err = ks2.Load("test-passphrase")
	if err != nil {
		t.Fatalf("Failed to load keystore: %v", err)
	}

	loadedKey := ks2.GetGlobalKey()
	if loadedKey == nil {
		t.Fatal("Expected loaded key to be available")
	}

	// Keys should match
	if string(originalGlobalKey.Key) != string(loadedKey.Key) {
		t.Error("Loaded key does not match original key")
	}

	if originalGlobalKey.KeyID != loadedKey.KeyID {
		t.Error("Loaded KeyID does not match original KeyID")
	}
}

func TestKeyStoreLoadWrongPassphrase(t *testing.T) {
	// Clear environment to ensure test isolation
	originalKey := os.Getenv("MARCHAT_GLOBAL_E2E_KEY")
	os.Unsetenv("MARCHAT_GLOBAL_E2E_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("MARCHAT_GLOBAL_E2E_KEY", originalKey)
		}
	}()

	// Create temporary directory for test
	tmpDir := t.TempDir()
	keystorePath := filepath.Join(tmpDir, "test-keystore.dat")

	// Create a keystore with one passphrase
	ks1 := NewKeyStore(keystorePath)
	err := ks1.Initialize("correct-passphrase")
	if err != nil {
		t.Fatalf("Failed to initialize keystore: %v", err)
	}

	// Try to load with wrong passphrase
	ks2 := NewKeyStore(keystorePath)
	err = ks2.Load("wrong-passphrase")
	if err == nil {
		t.Error("Expected error when loading with wrong passphrase")
	}
}

func TestKeyStoreGetSessionKey(t *testing.T) {
	// Clear environment to ensure test isolation
	originalKey := os.Getenv("MARCHAT_GLOBAL_E2E_KEY")
	os.Unsetenv("MARCHAT_GLOBAL_E2E_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("MARCHAT_GLOBAL_E2E_KEY", originalKey)
		}
	}()

	tmpDir := t.TempDir()
	keystorePath := filepath.Join(tmpDir, "test-keystore.dat")

	ks := NewKeyStore(keystorePath)
	err := ks.Initialize("test-passphrase")
	if err != nil {
		t.Fatalf("Failed to initialize keystore: %v", err)
	}

	// Test getting session key (should return global key)
	sessionKey := ks.GetSessionKey("any-conversation-id")
	if sessionKey == nil {
		t.Error("Expected session key to be available")
	}

	globalKey := ks.GetGlobalKey()
	if sessionKey != globalKey {
		t.Error("Session key should be the same as global key")
	}
}

func TestKeyStoreEncryptDecryptMessage(t *testing.T) {
	// Clear environment to ensure test isolation
	originalKey := os.Getenv("MARCHAT_GLOBAL_E2E_KEY")
	os.Unsetenv("MARCHAT_GLOBAL_E2E_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("MARCHAT_GLOBAL_E2E_KEY", originalKey)
		}
	}()

	tmpDir := t.TempDir()
	keystorePath := filepath.Join(tmpDir, "test-keystore.dat")

	ks := NewKeyStore(keystorePath)
	err := ks.Initialize("test-passphrase")
	if err != nil {
		t.Fatalf("Failed to initialize keystore: %v", err)
	}

	// Test encrypting a message
	sender := "testuser"
	content := "Hello, world!"
	conversationID := "global"

	encrypted, err := ks.EncryptMessage(sender, content, conversationID)
	if err != nil {
		t.Fatalf("Failed to encrypt message: %v", err)
	}

	if encrypted.Sender != sender {
		t.Errorf("Expected sender %s, got %s", sender, encrypted.Sender)
	}

	// Content field is not set in encrypted message - it's encrypted in the payload
	if encrypted.Content != "" {
		t.Errorf("Expected empty content in encrypted message, got %s", encrypted.Content)
	}

	if !encrypted.IsEncrypted {
		t.Error("Expected message to be marked as encrypted")
	}

	if len(encrypted.Encrypted) == 0 {
		t.Error("Expected encrypted data to be present")
	}

	if len(encrypted.Nonce) == 0 {
		t.Error("Expected nonce to be present")
	}

	// Test decrypting the message
	decrypted, err := ks.DecryptMessage(encrypted, conversationID)
	if err != nil {
		t.Fatalf("Failed to decrypt message: %v", err)
	}

	if decrypted.Sender != sender {
		t.Errorf("Expected decrypted sender %s, got %s", sender, decrypted.Sender)
	}

	if decrypted.Content != content {
		t.Errorf("Expected decrypted content %s, got %s", content, decrypted.Content)
	}

	if decrypted.Encrypted {
		t.Error("Expected decrypted message to not be marked as encrypted")
	}
}

func TestKeyStoreEncryptMessageNoSessionKey(t *testing.T) {
	// Create keystore without initializing
	tmpDir := t.TempDir()
	keystorePath := filepath.Join(tmpDir, "test-keystore.dat")

	ks := NewKeyStore(keystorePath)

	// Try to encrypt without session key
	_, err := ks.EncryptMessage("testuser", "hello", "global")
	if err == nil {
		t.Error("Expected error when encrypting without session key")
	}
}

func TestKeyStoreDecryptMessageNoSessionKey(t *testing.T) {
	// Create keystore without initializing
	tmpDir := t.TempDir()
	keystorePath := filepath.Join(tmpDir, "test-keystore.dat")

	ks := NewKeyStore(keystorePath)

	// Create a dummy encrypted message
	encrypted := &shared.EncryptedMessage{
		Sender:      "testuser",
		Content:     "hello",
		IsEncrypted: true,
		Encrypted:   []byte("encrypted-data"),
		Nonce:       []byte("nonce"),
		CreatedAt:   time.Now(),
	}

	// Try to decrypt without session key
	_, err := ks.DecryptMessage(encrypted, "global")
	if err == nil {
		t.Error("Expected error when decrypting without session key")
	}
}

func TestDeriveKeyFromPassphrase(t *testing.T) {
	passphrase := "test-passphrase"
	keystorePath := "/tmp/test-keystore.dat"
	key := deriveKeyFromPassphrase([]byte(passphrase), keystorePath)

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}

	// Same passphrase and path should produce same key
	key2 := deriveKeyFromPassphrase([]byte(passphrase), keystorePath)
	if string(key) != string(key2) {
		t.Error("Same passphrase and path should produce same key")
	}

	// Different passphrase should produce different key
	differentKey := deriveKeyFromPassphrase([]byte("different-passphrase"), keystorePath)
	if string(key) == string(differentKey) {
		t.Error("Different passphrase should produce different key")
	}

	// Different path should produce different key (even with same passphrase)
	differentPathKey := deriveKeyFromPassphrase([]byte(passphrase), "/tmp/different-path.dat")
	if string(key) == string(differentPathKey) {
		t.Error("Different keystore path should produce different key")
	}
}

func TestEncryptDecryptData(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	data := []byte("test data to encrypt")

	// Encrypt
	encrypted, err := encryptData(key, data)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	if len(encrypted) <= len(data) {
		t.Error("Encrypted data should be longer than original data")
	}

	// Decrypt
	decrypted, err := decryptData(key, encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt data: %v", err)
	}

	if string(decrypted) != string(data) {
		t.Errorf("Decrypted data does not match original: expected %s, got %s", string(data), string(decrypted))
	}
}

func TestEncryptDecryptDataWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 1 // Make keys different

	data := []byte("test data")

	// Encrypt with key1
	encrypted, err := encryptData(key1, data)
	if err != nil {
		t.Fatalf("Failed to encrypt data: %v", err)
	}

	// Try to decrypt with key2
	_, err = decryptData(key2, encrypted)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key")
	}
}

func TestDecryptDataInvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)

	// Test with too short ciphertext
	_, err := decryptData(key, []byte("short"))
	if err == nil {
		t.Error("Expected error with too short ciphertext")
	}

	// Test with empty ciphertext
	_, err = decryptData(key, []byte{})
	if err == nil {
		t.Error("Expected error with empty ciphertext")
	}
}
