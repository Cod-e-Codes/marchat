package main

import (
	"errors"
	"testing"

	"github.com/Cod-e-Codes/marchat/client/crypto"
	"github.com/Cod-e-Codes/marchat/shared"
)

func TestContentExceedsMessageLimit(t *testing.T) {
	t.Setenv(shared.EnvMaxMessageBytes, "4")
	t.Setenv(shared.EnvMaxMessageMB, "")
	if contentExceedsMessageLimit("abcd") {
		t.Fatal("content at the cap must be allowed")
	}
	if !contentExceedsMessageLimit("abcde") {
		t.Fatal("content over the cap must be rejected")
	}
	if messageTooLargeBanner() != "[ERROR] Message too large (max 4 bytes)" {
		t.Fatalf("banner = %q", messageTooLargeBanner())
	}
}

func TestSendEncryptedChatMessageRejectsOversizedWire(t *testing.T) {
	t.Setenv(shared.EnvMaxMessageBytes, "20")
	t.Setenv(shared.EnvMaxMessageMB, "")

	ks := crypto.NewKeyStore(t.TempDir() + "/keystore.dat")
	if err := ks.Initialize("test-passphrase"); err != nil {
		t.Fatal(err)
	}
	plaintext := "hi"
	if contentExceedsMessageLimit(plaintext) {
		t.Fatal("plaintext should fit under the wire-only case")
	}
	msg, err := buildEncryptedOutboundMessage(ks, "alice", plaintext, shared.TextMessage, "")
	if err != nil {
		t.Fatal(err)
	}
	if !contentExceedsMessageLimit(msg.Content) {
		t.Fatalf("encrypted wire length %d should exceed cap", len(msg.Content))
	}
	err = sendEncryptedChatMessage(nil, ks, "alice", plaintext, shared.TextMessage, "")
	if !errors.Is(err, errMessageTooLarge) {
		t.Fatalf("got %v, want errMessageTooLarge", err)
	}
}
