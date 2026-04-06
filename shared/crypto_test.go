package shared

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"testing"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

func testSessionKey(t *testing.T) *SessionKey {
	t.Helper()
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(key)
	return &SessionKey{
		Key:       key,
		CreatedAt: time.Now(),
		KeyID:     base64.StdEncoding.EncodeToString(sum[:]),
	}
}

func TestEncryptDecryptMessage(t *testing.T) {
	sessionKey := testSessionKey(t)
	plaintext := []byte("Hello, World! This is a test message.")

	encrypted, err := EncryptMessage(sessionKey, plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt message: %v", err)
	}

	if encrypted == nil {
		t.Fatal("Encrypted message should not be nil")
	}

	if !encrypted.IsEncrypted {
		t.Error("IsEncrypted should be true")
	}

	if len(encrypted.Encrypted) == 0 {
		t.Error("Encrypted data should not be empty")
	}

	if len(encrypted.Nonce) == 0 {
		t.Error("Nonce should not be empty")
	}

	decrypted, err := DecryptMessage(sessionKey, encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt message: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted message doesn't match original. Expected: %s, Got: %s", plaintext, decrypted)
	}
}

func TestEncryptDecryptTextMessage(t *testing.T) {
	sessionKey := testSessionKey(t)
	sender := "alice"
	content := "Hello, Bob! This is a test message."

	encrypted, err := EncryptTextMessage(sessionKey, sender, content)
	if err != nil {
		t.Fatalf("Failed to encrypt text message: %v", err)
	}

	if encrypted == nil {
		t.Fatal("Encrypted message should not be nil")
	}

	if encrypted.Sender != sender {
		t.Errorf("Expected sender %s, got %s", sender, encrypted.Sender)
	}

	if encrypted.Type != TextMessage {
		t.Errorf("Expected type %s, got %s", TextMessage, encrypted.Type)
	}

	if !encrypted.IsEncrypted {
		t.Error("IsEncrypted should be true")
	}

	decrypted, err := DecryptTextMessage(sessionKey, encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt text message: %v", err)
	}

	if decrypted == nil {
		t.Fatal("Decrypted message should not be nil")
	}

	if decrypted.Sender != sender {
		t.Errorf("Expected sender %s, got %s", sender, decrypted.Sender)
	}

	if decrypted.Content != content {
		t.Errorf("Expected content %s, got %s", content, decrypted.Content)
	}

	if decrypted.Type != TextMessage {
		t.Errorf("Expected type %s, got %s", TextMessage, decrypted.Type)
	}
}

func TestDecryptMessageInvalidData(t *testing.T) {
	sessionKey := testSessionKey(t)

	nonEncrypted := &EncryptedMessage{
		IsEncrypted: false,
		Encrypted:   []byte("fake data"),
		Nonce:       make([]byte, 12),
	}

	_, err := DecryptMessage(sessionKey, nonEncrypted)
	if err == nil {
		t.Error("Expected error when decrypting non-encrypted message")
	}

	plaintext := []byte("test message")
	encrypted, err := EncryptMessage(sessionKey, plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt message: %v", err)
	}

	encrypted.Nonce[0] = ^encrypted.Nonce[0]

	_, err = DecryptMessage(sessionKey, encrypted)
	if err == nil {
		t.Error("Expected error when decrypting with corrupted nonce")
	}
}
