package main

import (
	"os"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

func TestMain(m *testing.M) {
	// Lipgloss is inert without a TTY; force ANSI256 so render/hyperlink tests
	// emit real SGR sequences in CI and headless environments.
	lipgloss.Writer.Profile = colorprofile.ANSI256
	os.Exit(m.Run())
}
