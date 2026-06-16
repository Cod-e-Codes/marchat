package main

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	// Lipgloss is inert without a TTY; force ANSI256 so render/hyperlink tests
	// emit real SGR sequences in CI and headless environments.
	lipgloss.SetColorProfile(termenv.ANSI256)
	os.Exit(m.Run())
}
