package shared

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

// EncryptedMessage represents an E2E encrypted message (AEAD payload and metadata).
type EncryptedMessage struct {
	Type        MessageType `json:"type"`
	Sender      string      `json:"sender"`
	CreatedAt   time.Time   `json:"created_at"`
	Content     string      `json:"content,omitempty"`      // Plaintext for system messages
	Encrypted   []byte      `json:"encrypted,omitempty"`    // Encrypted payload
	Nonce       []byte      `json:"nonce,omitempty"`        // For encrypted messages
	Recipient   string      `json:"recipient,omitempty"`    // For direct messages
	IsEncrypted bool        `json:"is_encrypted,omitempty"` // Flag for encrypted messages
	File        *FileMeta   `json:"file,omitempty"`         // For file messages
}

// SessionKey holds 32-byte ChaCha20-Poly1305 key material for the global E2E model.
// KeyID is a base64(SHA256(key)) fingerprint for logs and display, not a wire identifier.
type SessionKey struct {
	Key       []byte    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
	KeyID     string    `json:"key_id"`
}

// EncryptMessage encrypts a message using ChaCha20-Poly1305.
func EncryptMessage(sessionKey *SessionKey, plaintext []byte) (*EncryptedMessage, error) {
	aead, err := chacha20poly1305.New(sessionKey.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	return &EncryptedMessage{
		Encrypted:   ciphertext,
		Nonce:       nonce,
		IsEncrypted: true,
		CreatedAt:   time.Now(),
	}, nil
}

// DecryptMessage decrypts a message using ChaCha20-Poly1305.
func DecryptMessage(sessionKey *SessionKey, encrypted *EncryptedMessage) ([]byte, error) {
	if !encrypted.IsEncrypted {
		return nil, errors.New("message is not encrypted")
	}

	aead, err := chacha20poly1305.New(sessionKey.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD: %w", err)
	}

	plaintext, err := aead.Open(nil, encrypted.Nonce, encrypted.Encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt message: %w", err)
	}

	return plaintext, nil
}

// EncryptTextMessage encrypts a text message.
func EncryptTextMessage(sessionKey *SessionKey, sender, content string) (*EncryptedMessage, error) {
	payload := Message{
		Sender:    sender,
		Content:   content,
		Type:      TextMessage,
		CreatedAt: time.Now(),
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	encrypted, err := EncryptMessage(sessionKey, plaintext)
	if err != nil {
		return nil, err
	}

	encrypted.Sender = sender
	encrypted.Type = TextMessage
	return encrypted, nil
}

// DecryptTextMessage decrypts a text message and returns the original Message.
func DecryptTextMessage(sessionKey *SessionKey, encrypted *EncryptedMessage) (*Message, error) {
	plaintext, err := DecryptMessage(sessionKey, encrypted)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(plaintext, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decrypted message: %w", err)
	}

	return &msg, nil
}
