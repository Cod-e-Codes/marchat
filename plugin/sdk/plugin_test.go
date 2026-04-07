package sdk

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
	msg := Message{
		Sender:    "test-plugin",
		Content:   "Hello from plugin!",
		CreatedAt: time.Now(),
	}

	if msg.Sender != "test-plugin" {
		t.Errorf("Expected sender 'test-plugin', got %s", msg.Sender)
	}

	if msg.Content != "Hello from plugin!" {
		t.Errorf("Expected content 'Hello from plugin!', got %s", msg.Content)
	}

	if msg.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestMessageJSON(t *testing.T) {
	now := time.Now()
	msg := Message{
		Sender:    "test-plugin",
		Content:   "JSON test message",
		CreatedAt: now,
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal Message: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Message
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Message: %v", err)
	}

	if unmarshaled.Sender != msg.Sender {
		t.Errorf("Expected sender %s, got %s", msg.Sender, unmarshaled.Sender)
	}

	if unmarshaled.Content != msg.Content {
		t.Errorf("Expected content %s, got %s", msg.Content, unmarshaled.Content)
	}

	// Note: Time comparison might be tricky due to JSON precision
	if !unmarshaled.CreatedAt.Equal(msg.CreatedAt) {
		t.Errorf("Expected CreatedAt %v, got %v", msg.CreatedAt, unmarshaled.CreatedAt)
	}
}

func TestMessageEmptyFields(t *testing.T) {
	msg := Message{}

	if msg.Sender != "" {
		t.Errorf("Expected empty sender, got %s", msg.Sender)
	}

	if msg.Content != "" {
		t.Errorf("Expected empty content, got %s", msg.Content)
	}

	if !msg.CreatedAt.IsZero() {
		t.Error("Expected zero CreatedAt for empty message")
	}
}

func TestMessageExtendedFields(t *testing.T) {
	msg := Message{
		Sender:    "alice",
		Content:   "hello",
		CreatedAt: time.Now(),
		Type:      "text",
		Channel:   "dev",
		Encrypted: true,
		MessageID: 42,
		Recipient: "bob",
		Edited:    true,
	}

	if msg.Channel != "dev" {
		t.Errorf("Expected channel 'dev', got %s", msg.Channel)
	}
	if !msg.Encrypted {
		t.Error("Expected Encrypted to be true")
	}
	if msg.MessageID != 42 {
		t.Errorf("Expected MessageID 42, got %d", msg.MessageID)
	}
	if msg.Recipient != "bob" {
		t.Errorf("Expected recipient 'bob', got %s", msg.Recipient)
	}
	if !msg.Edited {
		t.Error("Expected Edited to be true")
	}
}

func TestMessageExtendedFieldsJSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	msg := Message{
		Sender:    "alice",
		Content:   "encrypted payload",
		CreatedAt: now,
		Type:      "text",
		Channel:   "general",
		Encrypted: true,
		MessageID: 99,
		Recipient: "bob",
		Edited:    true,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var got Message
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if got.Channel != msg.Channel {
		t.Errorf("Channel: want %q, got %q", msg.Channel, got.Channel)
	}
	if got.Encrypted != msg.Encrypted {
		t.Errorf("Encrypted: want %v, got %v", msg.Encrypted, got.Encrypted)
	}
	if got.MessageID != msg.MessageID {
		t.Errorf("MessageID: want %d, got %d", msg.MessageID, got.MessageID)
	}
	if got.Recipient != msg.Recipient {
		t.Errorf("Recipient: want %q, got %q", msg.Recipient, got.Recipient)
	}
	if got.Edited != msg.Edited {
		t.Errorf("Edited: want %v, got %v", msg.Edited, got.Edited)
	}
}

func TestMessageExtendedFieldsOmitEmpty(t *testing.T) {
	msg := Message{
		Sender:    "bot",
		Content:   "hi",
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	raw := string(data)
	for _, key := range []string{`"channel"`, `"encrypted"`, `"message_id"`, `"recipient"`, `"edited"`} {
		if strings.Contains(raw, key) {
			t.Errorf("Zero-value field %s should be omitted from JSON, got: %s", key, raw)
		}
	}
}

func TestMessageBackwardsCompatUnknownFieldsIgnored(t *testing.T) {
	jsonWithExtra := `{"sender":"hub","content":"test","created_at":"2025-01-01T00:00:00Z","channel":"dev","encrypted":true,"message_id":7,"recipient":"bob","edited":true,"some_future_field":"ignored"}`

	var msg Message
	if err := json.Unmarshal([]byte(jsonWithExtra), &msg); err != nil {
		t.Fatalf("Unmarshal should ignore unknown fields: %v", err)
	}
	if msg.Channel != "dev" {
		t.Errorf("Channel: want dev, got %s", msg.Channel)
	}
	if msg.MessageID != 7 {
		t.Errorf("MessageID: want 7, got %d", msg.MessageID)
	}
}

func TestMessageWithSpecialCharacters(t *testing.T) {
	specialContent := "Hello 世界! 🚀 Special chars: @#$%^&*()"
	msg := Message{
		Sender:    "plugin-测试",
		Content:   specialContent,
		CreatedAt: time.Now(),
	}

	if msg.Sender != "plugin-测试" {
		t.Errorf("Expected sender 'plugin-测试', got %s", msg.Sender)
	}

	if msg.Content != specialContent {
		t.Errorf("Expected content %s, got %s", specialContent, msg.Content)
	}

	// Test JSON roundtrip with special characters
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal Message with special chars: %v", err)
	}

	var unmarshaled Message
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Message with special chars: %v", err)
	}

	if unmarshaled.Sender != msg.Sender {
		t.Errorf("Expected sender %s, got %s", msg.Sender, unmarshaled.Sender)
	}

	if unmarshaled.Content != msg.Content {
		t.Errorf("Expected content %s, got %s", msg.Content, unmarshaled.Content)
	}
}

func TestMessageTimestamp(t *testing.T) {
	before := time.Now()
	msg := Message{
		Sender:    "test",
		Content:   "timestamp test",
		CreatedAt: time.Now(),
	}
	after := time.Now()

	// Verify the message fields are set correctly
	if msg.Sender != "test" {
		t.Errorf("Expected sender 'test', got %s", msg.Sender)
	}
	if msg.Content != "timestamp test" {
		t.Errorf("Expected content 'timestamp test', got %s", msg.Content)
	}

	if msg.CreatedAt.Before(before) {
		t.Error("CreatedAt should not be before creation time")
	}

	if msg.CreatedAt.After(after) {
		t.Error("CreatedAt should not be after creation time")
	}
}

func TestMessageLongContent(t *testing.T) {
	// Create a long message
	longContent := ""
	for i := 0; i < 1000; i++ {
		longContent += "This is a very long message content. "
	}

	msg := Message{
		Sender:    "long-message-plugin",
		Content:   longContent,
		CreatedAt: time.Now(),
	}

	if len(msg.Content) != len(longContent) {
		t.Errorf("Expected content length %d, got %d", len(longContent), len(msg.Content))
	}

	// Test JSON marshaling/unmarshaling with long content
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal Message with long content: %v", err)
	}

	var unmarshaled Message
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Message with long content: %v", err)
	}

	if unmarshaled.Content != msg.Content {
		t.Error("Long content should be preserved through JSON roundtrip")
	}
}
