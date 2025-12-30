package crypto

import (
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

	"github.com/Cod-e-Codes/marchat/shared"
)

// KeyStore manages cryptographic keys for the client (global encryption only)
type KeyStore struct {
	mu           sync.RWMutex
	globalKey    *shared.SessionKey // Global key for public channels
	keystorePath string
	passphrase   []byte
}

// NewKeyStore creates a new key store
func NewKeyStore(keystorePath string) *KeyStore {
	return &KeyStore{
		keystorePath: keystorePath,
	}
}

// Initialize initializes the keystore for global encryption only
func (ks *KeyStore) Initialize(passphrase string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.passphrase = []byte(passphrase)

	// Check if keystore already exists
	if _, err := os.Stat(ks.keystorePath); err == nil {
		if err := ks.load(); err != nil {
			return err
		}
	}

	// Initialize global key
	return ks.initializeGlobalKey()
}

// Load loads the keystore from disk
func (ks *KeyStore) Load(passphrase string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.passphrase = []byte(passphrase)
	if err := ks.load(); err != nil {
		return err
	}

	// Initialize global key after loading
	return ks.initializeGlobalKey()
}

// GetGlobalKey returns the global session key
func (ks *KeyStore) GetGlobalKey() *shared.SessionKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.globalKey
}

// initializeGlobalKey initializes the global key from environment variable or generates a new one
// Note: This method should be called from methods that already hold the lock
func (ks *KeyStore) initializeGlobalKey() error {

	// Check for environment variable first
	if envKey := os.Getenv("MARCHAT_GLOBAL_E2E_KEY"); envKey != "" {
		// Decode base64 key from environment
		keyBytes, err := base64.StdEncoding.DecodeString(envKey)
		if err != nil {
			return fmt.Errorf("invalid global key format in environment: %w", err)
		}

		// Validate key length (should be 32 bytes for ChaCha20-Poly1305)
		if len(keyBytes) != 32 {
			return fmt.Errorf("global key must be 32 bytes, got %d", len(keyBytes))
		}

		// Create session key from environment key
		keyID := sha256.Sum256(keyBytes)
		ks.globalKey = &shared.SessionKey{
			Key:       keyBytes,
			CreatedAt: time.Now(),
			KeyID:     base64.StdEncoding.EncodeToString(keyID[:]),
		}

		fmt.Printf("🔐 Using global E2E key from environment variable\n")
		return nil
	}

	// Generate new global key if none exists
	if ks.globalKey == nil {
		keyBytes := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
			return fmt.Errorf("failed to generate global key: %w", err)
		}

		keyID := sha256.Sum256(keyBytes)
		ks.globalKey = &shared.SessionKey{
			Key:       keyBytes,
			CreatedAt: time.Now(),
			KeyID:     base64.StdEncoding.EncodeToString(keyID[:]),
		}

		fmt.Printf("🔐 Generated new global E2E key (ID: %s)\n", ks.globalKey.KeyID)
		fmt.Printf("💡 Set MARCHAT_GLOBAL_E2E_KEY=%s to share this key across clients\n",
			base64.StdEncoding.EncodeToString(ks.globalKey.Key))

		// Save the newly generated key to disk
		if err := ks.save(); err != nil {
			return fmt.Errorf("failed to save new global key: %w", err)
		}
	}

	return nil
}

// GetSessionKey retrieves the global session key (only global encryption supported)
func (ks *KeyStore) GetSessionKey(conversationID string) *shared.SessionKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	// Only global conversation is supported
	return ks.globalKey
}

// EncryptMessage encrypts a message for a specific conversation
func (ks *KeyStore) EncryptMessage(sender, content, conversationID string) (*shared.EncryptedMessage, error) {
	sessionKey := ks.GetSessionKey(conversationID)
	if sessionKey == nil {
		return nil, fmt.Errorf("no session key for conversation: %s", conversationID)
	}

	return shared.EncryptTextMessage(sessionKey, sender, content)
}

// DecryptMessage decrypts a message using the appropriate session key
func (ks *KeyStore) DecryptMessage(encrypted *shared.EncryptedMessage, conversationID string) (*shared.Message, error) {
	sessionKey := ks.GetSessionKey(conversationID)
	if sessionKey == nil {
		return nil, fmt.Errorf("no session key for conversation: %s", conversationID)
	}

	return shared.DecryptTextMessage(sessionKey, encrypted)
}

// save encrypts and saves the keystore to disk
func (ks *KeyStore) save() error {
	// Create the keystore data (global encryption only)
	keystoreData := struct {
		GlobalKey *shared.SessionKey `json:"global_key"`
		Version   string             `json:"version"`
	}{
		GlobalKey: ks.globalKey,
		Version:   "2.0", // Version 2.0 for global-only encryption
	}

	// Serialize the data
	data, err := json.Marshal(keystoreData)
	if err != nil {
		return fmt.Errorf("failed to marshal keystore: %w", err)
	}

	// Derive encryption key from passphrase
	key := deriveKeyFromPassphrase(ks.passphrase, ks.keystorePath)

	// Encrypt the data
	encryptedData, err := encryptData(key, data)
	if err != nil {
		return fmt.Errorf("failed to encrypt keystore: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(ks.keystorePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create keystore directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(ks.keystorePath, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write keystore: %w", err)
	}

	return nil
}

// load decrypts and loads the keystore from disk
func (ks *KeyStore) load() error {
	// Read encrypted data
	encryptedData, err := os.ReadFile(ks.keystorePath)
	if err != nil {
		return fmt.Errorf("failed to read keystore: %w", err)
	}

	// Derive decryption key from passphrase
	key := deriveKeyFromPassphrase(ks.passphrase, ks.keystorePath)

	// Decrypt the data
	data, err := decryptData(key, encryptedData)
	if err != nil {
		return fmt.Errorf("failed to decrypt keystore: %w", err)
	}

	// Deserialize the data (supports both old and new formats)
	var keystoreData struct {
		// Legacy fields for backward compatibility
		Keypair     *shared.KeyPair                  `json:"keypair,omitempty"`
		PublicKeys  map[string]*shared.PublicKeyInfo `json:"public_keys,omitempty"`
		SessionKeys map[string]*shared.SessionKey    `json:"session_keys,omitempty"`
		// Current field
		GlobalKey *shared.SessionKey `json:"global_key"`
		Version   string             `json:"version"`
	}

	if err := json.Unmarshal(data, &keystoreData); err != nil {
		return fmt.Errorf("failed to unmarshal keystore: %w", err)
	}

	// Only load the global key (ignore legacy individual encryption data)
	ks.globalKey = keystoreData.GlobalKey

	return nil
}

// deriveKeyFromPassphrase derives a 32-byte key from a passphrase using PBKDF2.
// This provides secure key derivation that is resistant to brute-force attacks.
// The keystorePath is used as a salt to ensure different keystores produce different keys
// even with the same passphrase.
func deriveKeyFromPassphrase(passphrase []byte, keystorePath string) []byte {
	// Use the keystore path as salt for deterministic but unique key derivation
	// This ensures different keystore files produce different keys even with same passphrase
	salt := []byte(keystorePath)

	// PBKDF2 with SHA256, 100,000 iterations, 32-byte key length
	// 100k iterations provides good security while remaining reasonably fast
	const iterations = 100000
	const keyLen = 32

	return pbkdf2.Key(passphrase, salt, iterations, keyLen, sha256.New)
}

// encryptData encrypts data using AES-GCM
func encryptData(key, data []byte) ([]byte, error) {
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

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decryptData decrypts data using AES-GCM
func decryptData(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
