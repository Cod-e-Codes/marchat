package main

import (
	"errors"
	"fmt"

	"github.com/Cod-e-Codes/marchat/shared"
)

// errMessageTooLarge is returned when plaintext or E2E wire content exceeds the cap.
var errMessageTooLarge = errors.New("message too large")

func maxMessageBytes() int64 {
	return shared.MaxBytesFromEnv(shared.EnvMaxMessageBytes, shared.EnvMaxMessageMB, shared.DefaultMaxMessageBytes)
}

func messageTooLargeBanner() string {
	return "[ERROR] Message too large (max " + shared.FormatMessageLimit(maxMessageBytes()) + ")"
}

func contentExceedsMessageLimit(content string) bool {
	return shared.ContentExceedsLimit(content, maxMessageBytes())
}

func messageTooLargeError() error {
	return fmt.Errorf("%w (max %s)", errMessageTooLarge, shared.FormatMessageLimit(maxMessageBytes()))
}
