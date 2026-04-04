package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Startup / pre-TUI CLI styling (distinct from in-app Bubble Tea theme).
var (
	cliDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("#9E9E9E"))
	cliEmph  = lipgloss.NewStyle().Foreground(lipgloss.Color("#EEEEEE")).Bold(true)
	cliOK    = lipgloss.NewStyle().Foreground(lipgloss.Color("#81C784"))
	cliWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB74D"))
	cliErr   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E57373"))
	cliInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4DD0E1"))
	cliURL   = lipgloss.NewStyle().Foreground(lipgloss.Color("#4FC3F7"))
	cliPath  = lipgloss.NewStyle().Foreground(lipgloss.Color("#B0BEC5"))
	cliTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFE082")).Bold(true)
	cliID    = lipgloss.NewStyle().Foreground(lipgloss.Color("#CE93D8"))
)

func cliPrintConnecting(serverURL, username string) {
	line := cliDim.Render("Connecting to ") +
		cliURL.Render(serverURL) +
		cliDim.Render(" as ") +
		cliEmph.Render(username) +
		cliDim.Render("...")
	fmt.Println(line)
}

func cliPrintOK(msg string) {
	fmt.Println(cliOK.Render(msg))
}

func cliPrintWarn(msg string) {
	fmt.Println(cliWarn.Render(msg))
}

func cliPrintErr(msg string) {
	fmt.Println(cliErr.Render(msg))
}

func cliPrintAccent(msg string) {
	fmt.Println(cliInfo.Render(msg))
}

func cliPrintMuted(msg string) {
	fmt.Println(cliDim.Render(msg))
}

func cliPrintGlobalKeyID(keyID string) {
	fmt.Println(cliDim.Render("Global chat encryption: ") +
		cliOK.Render("ENABLED") +
		cliDim.Render(" (Key ID: ") +
		cliID.Render(keyID) +
		cliDim.Render(")"))
}

func cliPrintKeystorePath(path string) {
	fmt.Println(cliDim.Render("E2E encryption enabled with keystore: ") + cliPath.Render(path))
}

func cliWelcomeLine(msg string) {
	fmt.Println(cliTitle.Render(msg))
}

func cliSelectedProfile(name string) {
	fmt.Println(cliDim.Render("Selected: ") + cliEmph.Render(name))
}
