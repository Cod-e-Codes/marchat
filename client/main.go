package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Cod-e-Codes/marchat/client/config"
	"github.com/Cod-e-Codes/marchat/client/crypto"
	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/alecthomas/chroma/quick"

	"os/exec"
	"os/signal"
	"syscall"

	"encoding/base64"
	"encoding/json"

	"context"
	"sync"

	"log"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
)

const maxMessages = 100
const maxUsersDisplay = 20
const userListWidth = 18
const pingPeriod = 50 * time.Second        // moved from magic number
const reconnectMaxDelay = 30 * time.Second // for exponential backoff

var mentionRegex *regexp.Regexp
var urlRegex *regexp.Regexp

// keyMap defines all keybindings for the help system
type keyMap struct {
	Send        key.Binding
	ScrollUp    key.Binding
	ScrollDown  key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	Copy        key.Binding
	Paste       key.Binding
	Cut         key.Binding
	SelectAll   key.Binding
	Help        key.Binding
	Quit        key.Binding
	TimeFormat  key.Binding
	Clear       key.Binding
	QuitCommand key.Binding
	// Commands with both text commands and hotkey alternatives
	SendFile    key.Binding
	SaveFile    key.Binding
	Theme       key.Binding
	CodeSnippet key.Binding
	// Hotkey alternatives for commands (work even in encrypted sessions)
	SendFileHotkey    key.Binding
	ThemeHotkey       key.Binding
	TimeFormatHotkey  key.Binding
	ClearHotkey       key.Binding
	CodeSnippetHotkey key.Binding
	// Notification controls
	NotifyDesktop key.Binding
	// Admin UI commands
	DatabaseMenu key.Binding
	SelectUser   key.Binding
	CloseMenu    key.Binding
	// Admin action hotkeys
	BanUser             key.Binding
	KickUser            key.Binding
	UnbanUser           key.Binding
	AllowUser           key.Binding
	ForceDisconnectUser key.Binding
	// Plugin management hotkeys (admin only)
	PluginList      key.Binding
	PluginStore     key.Binding
	PluginRefresh   key.Binding
	PluginInstall   key.Binding
	PluginUninstall key.Binding
	PluginEnable    key.Binding
	PluginDisable   key.Binding
	// Legacy admin commands (for help display only)
	ClearDB key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Send, k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown},
		{k.Copy, k.Paste, k.Cut, k.SelectAll},
		{k.TimeFormat, k.Clear, k.Help, k.Quit},
	}
}

// GetCommandHelp returns command-specific help based on user permissions
func (k keyMap) GetCommandHelp(isAdmin, useE2E bool) [][]key.Binding {
	commands := [][]key.Binding{
		{k.SendFile, k.SaveFile, k.Theme, k.CodeSnippet},
		{k.SendFileHotkey, k.ThemeHotkey, k.TimeFormatHotkey, k.ClearHotkey, k.CodeSnippetHotkey},
	}

	// Individual E2E commands removed - only global E2E encryption is supported

	if isAdmin {
		commands = append(commands, []key.Binding{k.DatabaseMenu, k.SelectUser, k.BanUser, k.KickUser, k.UnbanUser, k.AllowUser, k.ForceDisconnectUser})
	}

	return commands
}

// newKeyMap creates a new keymap for the application
// This function is kept for potential future use in help documentation
func newKeyMap() keyMap {
	return keyMap{
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "page down"),
		),
		Copy: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "copy"),
		),
		Paste: key.NewBinding(
			key.WithKeys("ctrl+v"),
			key.WithHelp("ctrl+v", "paste"),
		),
		Cut: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl+x", "cut"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("ctrl+h", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close menus"),
		),
		TimeFormat: key.NewBinding(
			key.WithKeys(":time"),
			key.WithHelp(":time", "toggle 12/24h format"),
		),
		Clear: key.NewBinding(
			key.WithKeys(":clear"),
			key.WithHelp(":clear", "clear chat history"),
		),
		QuitCommand: key.NewBinding(
			key.WithKeys(":q"),
			key.WithHelp(":q", "quit client"),
		),
		SendFile: key.NewBinding(
			key.WithKeys(":sendfile"),
			key.WithHelp(":sendfile <path>", "send a file"),
		),
		SaveFile: key.NewBinding(
			key.WithKeys(":savefile"),
			key.WithHelp(":savefile <name>", "save received file"),
		),
		Theme: key.NewBinding(
			key.WithKeys(":theme"),
			key.WithHelp(":theme <name>", "change theme"),
		),
		CodeSnippet: key.NewBinding(
			key.WithKeys(":code"),
			key.WithHelp(":code", "create syntax highlighted code snippet"),
		),
		// Hotkey alternatives for commands (work even in encrypted sessions)
		SendFileHotkey: key.NewBinding(
			key.WithKeys("alt+f"),
			key.WithHelp("alt+f", "send a file (file picker)"),
		),
		ThemeHotkey: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "cycle through themes"),
		),
		TimeFormatHotkey: key.NewBinding(
			key.WithKeys("alt+t"),
			key.WithHelp("alt+t", "toggle 12/24h time format"),
		),
		ClearHotkey: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear chat history"),
		),
		CodeSnippetHotkey: key.NewBinding(
			key.WithKeys("alt+c"),
			key.WithHelp("alt+c", "create code snippet"),
		),
		// Notification controls
		NotifyDesktop: key.NewBinding(
			key.WithKeys("alt+n"),
			key.WithHelp("alt+n", "toggle desktop notifications"),
		),
		// Admin UI commands
		DatabaseMenu: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "database menu (admin)"),
		),
		SelectUser: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "select/cycle user (admin)"),
		),
		CloseMenu: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close menu/clear selection"),
		),
		// Admin action hotkeys
		BanUser: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "ban selected user (admin)"),
		),
		KickUser: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "kick selected user (admin)"),
		),
		UnbanUser: key.NewBinding(
			key.WithKeys("ctrl+shift+b"),
			key.WithHelp("ctrl+shift+b", "unban user (admin)"),
		),
		AllowUser: key.NewBinding(
			key.WithKeys("ctrl+shift+a"),
			key.WithHelp("ctrl+shift+a", "allow user (admin)"),
		),
		ForceDisconnectUser: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "force disconnect selected user (admin)"),
		),
		// Plugin management hotkeys (admin only)
		PluginList: key.NewBinding(
			key.WithKeys("alt+p"),
			key.WithHelp("alt+p", "list plugins (admin)"),
		),
		PluginStore: key.NewBinding(
			key.WithKeys("alt+s"),
			key.WithHelp("alt+s", "plugin store (admin)"),
		),
		PluginRefresh: key.NewBinding(
			key.WithKeys("alt+r"),
			key.WithHelp("alt+r", "refresh plugins (admin)"),
		),
		PluginInstall: key.NewBinding(
			key.WithKeys("alt+i"),
			key.WithHelp("alt+i", "install plugin (admin)"),
		),
		PluginUninstall: key.NewBinding(
			key.WithKeys("alt+u"),
			key.WithHelp("alt+u", "uninstall plugin (admin)"),
		),
		PluginEnable: key.NewBinding(
			key.WithKeys("alt+e"),
			key.WithHelp("alt+e", "enable plugin (admin)"),
		),
		PluginDisable: key.NewBinding(
			key.WithKeys("alt+d"),
			key.WithHelp("alt+d", "disable plugin (admin)"),
		),
		// Legacy admin commands (for help display only)
		ClearDB: key.NewBinding(
			key.WithKeys(""),
			key.WithHelp("", ""),
		),
	}
}

// sortMessagesByTimestamp ensures messages are displayed in chronological order
// This provides client-side protection against server ordering issues
func sortMessagesByTimestamp(messages []shared.Message) {
	sort.Slice(messages, func(i, j int) bool {
		// Primary sort: by timestamp
		if !messages[i].CreatedAt.Equal(messages[j].CreatedAt) {
			return messages[i].CreatedAt.Before(messages[j].CreatedAt)
		}
		// Secondary sort: by sender for deterministic ordering when timestamps are identical
		if messages[i].Sender != messages[j].Sender {
			return messages[i].Sender < messages[j].Sender
		}
		// Tertiary sort: by content for full deterministic ordering
		return messages[i].Content < messages[j].Content
	})
}

func init() {
	mentionRegex = regexp.MustCompile(`\B@([a-zA-Z0-9_]+)\b`)
	// URL regex pattern to match http/https URLs and common domain patterns
	// This pattern matches URLs more comprehensively
	urlRegex = regexp.MustCompile(`(https?://[^\s<>"{}|\\^` + "`" + `\[\]]+|www\.[^\s<>"{}|\\^` + "`" + `\[\]]+\.[a-zA-Z]{2,})`)

	// Set up client debug logging to config directory
	configDir := getClientConfigDir()
	debugLogPath := filepath.Join(configDir, "marchat-client-debug.log")
	f, err := os.OpenFile(debugLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(f)
	}
	// If file creation fails, logs will go to stdout (but won't interfere with TUI)

	// Load custom themes
	if err := LoadCustomThemes(); err != nil {
		log.Printf("Warning: Failed to load custom themes: %v", err)
	}
}

// getClientConfigDir returns the client config directory using same logic as server
func getClientConfigDir() string {
	// Check environment variable first
	if envConfigDir := os.Getenv("MARCHAT_CONFIG_DIR"); envConfigDir != "" {
		return envConfigDir
	}

	// Check if we're in development mode (running from project root)
	if _, err := os.Stat("go.mod"); err == nil {
		return "./config"
	}

	// Production mode - use XDG config home
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "marchat")
	}

	// Fallback to user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./config"
	}

	return filepath.Join(homeDir, ".config", "marchat")
}

var (
	configPath         = flag.String("config", "", "Path to config file (optional)")
	serverURL          = flag.String("server", "", "Server URL")
	username           = flag.String("username", "", "Username")
	theme              = flag.String("theme", "", "Theme")
	isAdmin            = flag.Bool("admin", false, "Connect as admin (requires --admin-key)")
	adminKey           = flag.String("admin-key", "", "Admin key for privileged commands")
	useE2E             = flag.Bool("e2e", false, "Enable end-to-end encryption")
	keystorePassphrase = flag.String("keystore-passphrase", "", "Passphrase for keystore (required for E2E)")
	skipTLSVerify      = flag.Bool("skip-tls-verify", false, "Skip TLS certificate verification")
	quickStart         = flag.Bool("quick-start", false, "Use last connection or select from saved profiles")
	autoConnect        = flag.Bool("auto", false, "Automatically connect using most recent profile")
	nonInteractive     = flag.Bool("non-interactive", false, "Skip interactive prompts (require all flags)")
)

// isTermux detects if the client is running in Termux environment
func isTermux() bool {
	return os.Getenv("TERMUX_VERSION") != "" ||
		os.Getenv("PREFIX") == "/data/data/com.termux/files/usr" ||
		(os.Getenv("ANDROID_DATA") != "" && os.Getenv("ANDROID_ROOT") != "")
}

// safeClipboardOperation wraps clipboard operations with a timeout to prevent freezing
func safeClipboardOperation(operation func() error, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- operation()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// checkClipboardSupport tests if clipboard operations work in the current environment
func checkClipboardSupport() bool {
	err := safeClipboardOperation(func() error {
		return clipboard.WriteAll("test")
	}, 1*time.Second)
	return err == nil
}

// Add these helper functions after the existing imports and before the model struct

// debugEncryptAndSend provides comprehensive logging around encryption
func debugEncryptAndSend(recipients []string, plaintext string, ws *websocket.Conn, keystore *crypto.KeyStore, username string) error {
	log.Printf("DEBUG: Starting global encryption for %d recipients", len(recipients))
	log.Printf("DEBUG: Plaintext length: %d", len(plaintext))

	// Check keystore status
	if keystore == nil {
		log.Printf("ERROR: Keystore is nil")
		return fmt.Errorf("keystore not initialized")
	}
	log.Printf("DEBUG: Keystore loaded: %t", keystore != nil)

	// Verify global key is available
	globalKey := keystore.GetSessionKey("global")
	if globalKey == nil {
		log.Printf("ERROR: Global key not found")
		return fmt.Errorf("global key not available - global E2E encryption not initialized")
	}
	log.Printf("DEBUG: Global key available (ID: %s)", globalKey.KeyID)

	// Perform encryption using global key
	conversationID := "global"
	encryptedMsg, err := keystore.EncryptMessage(username, plaintext, conversationID)
	if err != nil {
		log.Printf("ERROR: Global encryption failed: %v", err)
		return fmt.Errorf("global encryption failed: %v", err)
	}

	log.Printf("DEBUG: Global encryption successful - encrypted length: %d", len(encryptedMsg.Encrypted))

	// Guard against empty ciphertext
	if len(encryptedMsg.Encrypted) == 0 {
		log.Printf("ERROR: Encryption returned empty ciphertext")
		return fmt.Errorf("encryption returned empty ciphertext; aborting send")
	}

	// Combine nonce + encrypted data and base64 encode for safe JSON transport
	combinedData := make([]byte, 0, len(encryptedMsg.Nonce)+len(encryptedMsg.Encrypted))
	combinedData = append(combinedData, encryptedMsg.Nonce...)
	combinedData = append(combinedData, encryptedMsg.Encrypted...)

	finalContent := base64.StdEncoding.EncodeToString(combinedData)
	log.Printf("DEBUG: Base64 encoded nonce+ciphertext - length: %d", len(finalContent))

	// Verify final content is not empty
	if len(finalContent) == 0 {
		log.Printf("ERROR: Final content is empty after encoding")
		return fmt.Errorf("final content is empty after encoding")
	}

	// Create a regular Message struct for the server
	msg := shared.Message{
		Content:   finalContent,
		Sender:    username,
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
		Encrypted: true, // Mark as encrypted
	}

	log.Printf("DEBUG: Final message - Content length: %d, Type: %s",
		len(msg.Content), msg.Type)

	// Send message
	if err := ws.WriteJSON(msg); err != nil {
		log.Printf("ERROR: WebSocket write failed: %v", err)
		return err
	}

	log.Printf("DEBUG: Global encrypted message sent successfully")
	return nil
}

// validateEncryptionRoundtrip tests encryption primitives using global key
func validateEncryptionRoundtrip(keystore *crypto.KeyStore, username string) error {
	testPlaintext := "Hello, global encryption test!"

	log.Printf("DEBUG: Testing global encryption roundtrip")

	// Test global conversation encryption
	conversationID := "global"

	// Verify global key exists
	globalKey := keystore.GetSessionKey(conversationID)
	if globalKey == nil {
		return fmt.Errorf("global key not found - global E2E encryption not available")
	}

	log.Printf("DEBUG: Global key found (ID: %s)", globalKey.KeyID)

	// Test encryption using global key
	encryptedMsg, err := keystore.EncryptMessage(username, testPlaintext, conversationID)
	if err != nil {
		return fmt.Errorf("global encryption test failed: %v", err)
	}

	if len(encryptedMsg.Encrypted) == 0 {
		return fmt.Errorf("global encryption test produced empty ciphertext")
	}

	log.Printf("DEBUG: Global encryption test successful - ciphertext length: %d", len(encryptedMsg.Encrypted))

	// Test decryption roundtrip
	decryptedMsg, err := keystore.DecryptMessage(encryptedMsg, conversationID)
	if err != nil {
		return fmt.Errorf("global decryption test failed: %v", err)
	}

	if decryptedMsg.Content != testPlaintext {
		return fmt.Errorf("global decryption roundtrip failed: expected '%s', got '%s'", testPlaintext, decryptedMsg.Content)
	}

	log.Printf("DEBUG: Global encryption roundtrip test successful")
	return nil
}

// verifyKeystoreUnlocked verifies keystore is properly unlocked (global encryption only)
func verifyKeystoreUnlocked(keystore *crypto.KeyStore) error {
	if keystore == nil {
		return fmt.Errorf("keystore is nil")
	}

	// Check if global key is available
	globalKey := keystore.GetGlobalKey()
	if globalKey == nil {
		return fmt.Errorf("global key not available")
	}

	log.Printf("DEBUG: Keystore properly unlocked for global encryption")
	return nil
}

// debugWebSocketWrite logs what's being sent over the wire
func debugWebSocketWrite(ws *websocket.Conn, msg interface{}) error {
	// Marshal to JSON to see what's being sent
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ERROR: JSON marshal failed: %v", err)
		return err
	}

	// Log without exposing sensitive content
	log.Printf("DEBUG: Sending WebSocket message - length: %d bytes", len(jsonData))

	// Check if content field is present and non-empty
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err == nil {
		if content, exists := parsed["content"]; exists {
			if contentStr, ok := content.(string); ok {
				log.Printf("DEBUG: Message content length: %d", len(contentStr))
				if len(contentStr) == 0 {
					log.Printf("WARNING: Sending message with empty content!")
				}
			}
		}
	}

	return ws.WriteJSON(msg)
}

// getCurrentProfileName finds the profile name that matches the current config
func getCurrentProfileName(cfg *config.Config) string {
	loader, err := config.NewInteractiveConfigLoader()
	if err != nil {
		return "" // Can't determine profile name
	}

	profiles, err := loader.LoadProfiles()
	if err != nil {
		return "" // Can't load profiles
	}

	// Find matching profile
	for _, profile := range profiles.Profiles {
		if profile.Username == cfg.Username &&
			profile.ServerURL == cfg.ServerURL &&
			profile.IsAdmin == cfg.IsAdmin &&
			profile.UseE2E == cfg.UseE2E {
			return profile.Name
		}
	}

	return "" // No matching profile found
}

// updateProfileTheme updates the theme in the corresponding profile in profiles.json
func (m *model) updateProfileTheme(newTheme string) error {
	if m.profileName == "" {
		return nil // No profile to update
	}

	loader, err := config.NewInteractiveConfigLoader()
	if err != nil {
		return err
	}

	profiles, err := loader.LoadProfiles()
	if err != nil {
		return err
	}

	// Find and update the matching profile
	for i, profile := range profiles.Profiles {
		if profile.Name == m.profileName {
			profiles.Profiles[i].Theme = newTheme
			return loader.SaveProfiles(profiles)
		}
	}

	return nil // Profile not found, nothing to update
}

type model struct {
	cfg            config.Config
	configFilePath string // Store the config file path for saving
	profileName    string // Store the current profile name for updating profiles.json
	textarea       textarea.Model
	viewport       viewport.Model
	messages       []shared.Message
	styles         themeStyles
	banner         string
	connected      bool

	users []string // NEW: user list

	width  int // NEW: track window width
	height int // NEW: track window height

	userListViewport viewport.Model // NEW: scrollable user list

	twentyFourHour bool // NEW: timestamp format toggle

	sending bool // NEW: sending message feedback

	conn    *websocket.Conn // persistent WebSocket connection
	msgChan chan tea.Msg    // channel for incoming messages from WS goroutine
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	reconnectDelay time.Duration               // for exponential backoff
	receivedFiles  map[string]*shared.FileMeta // filename -> filemeta for saving

	// E2E Encryption
	keystore *crypto.KeyStore
	useE2E   bool // Flag to enable/disable E2E encryption

	// Help system
	keys         keyMap
	help         help.Model
	showHelp     bool
	helpViewport viewport.Model // NEW: scrollable help viewport

	// Admin UI system
	showDBMenu     bool
	dbMenuViewport viewport.Model

	// User selection system
	selectedUserIndex int    // Index of currently selected user (-1 = none selected)
	selectedUser      string // Username of currently selected user

	// Code snippet system
	showCodeSnippet  bool
	codeSnippetModel codeSnippetModel

	// File picker system
	showFilePicker  bool
	filePickerModel filePickerModel

	// Notification system
	notificationManager *NotificationManager

	// Plugin command input system
	pendingPluginAction string // e.g., "install", "uninstall", "enable", "disable"
}

// configToNotificationConfig converts Config to NotificationConfig
func configToNotificationConfig(cfg config.Config) NotificationConfig {
	notifCfg := DefaultNotificationConfig()

	// Apply legacy bell settings if present
	if cfg.EnableBell {
		notifCfg.BellEnabled = true
	}
	if cfg.BellOnMention {
		notifCfg.BellOnMention = true
	}

	// Apply new notification settings from config
	switch cfg.NotificationMode {
	case "none":
		notifCfg.Mode = NotificationModeNone
	case "bell":
		notifCfg.Mode = NotificationModeBell
	case "desktop":
		notifCfg.Mode = NotificationModeDesktop
	case "both":
		notifCfg.Mode = NotificationModeBoth
	default:
		// Default to bell if not specified
		notifCfg.Mode = NotificationModeBell
	}

	notifCfg.DesktopEnabled = cfg.DesktopNotifications
	notifCfg.DesktopOnMention = cfg.DesktopOnMention
	notifCfg.DesktopOnDM = cfg.DesktopOnDM
	notifCfg.DesktopOnAll = cfg.DesktopOnAll
	notifCfg.QuietHoursEnabled = cfg.QuietHoursEnabled
	notifCfg.QuietHoursStart = cfg.QuietHoursStart
	notifCfg.QuietHoursEnd = cfg.QuietHoursEnd

	return notifCfg
}

// notificationConfigToConfig saves NotificationConfig back to Config
func notificationConfigToConfig(notifCfg NotificationConfig, cfg *config.Config) {
	// Update legacy bell settings for backward compatibility
	cfg.EnableBell = notifCfg.BellEnabled
	cfg.BellOnMention = notifCfg.BellOnMention

	// Update new notification settings
	switch notifCfg.Mode {
	case NotificationModeNone:
		cfg.NotificationMode = "none"
	case NotificationModeBell:
		cfg.NotificationMode = "bell"
	case NotificationModeDesktop:
		cfg.NotificationMode = "desktop"
	case NotificationModeBoth:
		cfg.NotificationMode = "both"
	}

	cfg.DesktopNotifications = notifCfg.DesktopEnabled
	cfg.DesktopOnMention = notifCfg.DesktopOnMention
	cfg.DesktopOnDM = notifCfg.DesktopOnDM
	cfg.DesktopOnAll = notifCfg.DesktopOnAll
	cfg.QuietHoursEnabled = notifCfg.QuietHoursEnabled
	cfg.QuietHoursStart = notifCfg.QuietHoursStart
	cfg.QuietHoursEnd = notifCfg.QuietHoursEnd
}

// shouldNotify determines the notification level for a message
func (m *model) shouldNotify(msg shared.Message) (bool, NotificationLevel) {
	// Don't notify for our own messages
	if msg.Sender == m.cfg.Username {
		return false, NotificationLevelInfo
	}

	// Check if the message mentions the current user
	mentionPattern := fmt.Sprintf("@%s", m.cfg.Username)
	isMention := strings.Contains(strings.ToLower(msg.Content), strings.ToLower(mentionPattern))

	// Determine notification level
	level := NotificationLevelInfo
	if isMention {
		level = NotificationLevelMention
	}

	// TODO: Add DM detection logic when DM support is added
	// if msg.IsDM {
	//     level = NotificationLevelDM
	// }

	return true, level
}

type themeStyles struct {
	User      lipgloss.Style
	Time      lipgloss.Style
	Msg       lipgloss.Style
	Banner    lipgloss.Style
	Box       lipgloss.Style // frame color
	Mention   lipgloss.Style // mention highlighting
	Hyperlink lipgloss.Style // hyperlink highlighting

	UserList lipgloss.Style // NEW: user list panel
	Me       lipgloss.Style // NEW: current user style
	Other    lipgloss.Style // NEW: other user style

	Background lipgloss.Style // NEW: main background
	Header     lipgloss.Style // NEW: header background
	Footer     lipgloss.Style // NEW: footer background
	Input      lipgloss.Style // NEW: input background

	// Help styles
	HelpOverlay lipgloss.Style
	HelpTitle   lipgloss.Style
}

// Base theme style helper
func baseThemeStyles() themeStyles {
	return themeStyles{
		User:       lipgloss.NewStyle().Bold(true),
		Time:       lipgloss.NewStyle().Faint(true),
		Msg:        lipgloss.NewStyle(),
		Banner:     lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F")).Bold(true),
		Box:        lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#AAAAAA")),
		Mention:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700")),
		Hyperlink:  lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("#4A9EFF")),
		UserList:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#AAAAAA")).Padding(0, 1),
		Me:         lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true),
		Other:      lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")),
		Background: lipgloss.NewStyle(),
		Header:     lipgloss.NewStyle(),
		Footer:     lipgloss.NewStyle(),
		Input:      lipgloss.NewStyle(),
		HelpOverlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(1, 2),
		HelpTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true).
			MarginBottom(1),
	}
}

func getThemeStyles(theme string) themeStyles {
	// Check for custom themes first
	if IsCustomTheme(theme) {
		if customTheme, ok := GetCustomTheme(theme); ok {
			return ApplyCustomTheme(customTheme)
		}
	}

	// Fall back to built-in themes
	s := baseThemeStyles()
	switch strings.ToLower(theme) {
	case "system":
		// System theme uses minimal styling to respect terminal defaults
		s.User = lipgloss.NewStyle().Bold(true)
		s.Time = lipgloss.NewStyle().Faint(true)
		s.Msg = lipgloss.NewStyle()
		s.Banner = lipgloss.NewStyle().Bold(true)
		s.Box = lipgloss.NewStyle().Border(lipgloss.NormalBorder())
		s.Mention = lipgloss.NewStyle().Bold(true)
		s.Hyperlink = lipgloss.NewStyle().Underline(true)
		s.UserList = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
		s.Me = lipgloss.NewStyle().Bold(true)
		s.Other = lipgloss.NewStyle()
		// Background and UI elements use no colors to respect terminal theme
		s.Background = lipgloss.NewStyle()
		s.Header = lipgloss.NewStyle().Bold(true)
		s.Footer = lipgloss.NewStyle()
		s.Input = lipgloss.NewStyle()
		s.HelpOverlay = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	case "patriot":
		s.User = s.User.Foreground(lipgloss.Color("#002868"))              // Navy blue
		s.Time = s.Time.Foreground(lipgloss.Color("#BF0A30")).Faint(false) // Red
		s.Msg = s.Msg.Foreground(lipgloss.Color("#FFFFFF"))
		s.Box = s.Box.BorderForeground(lipgloss.Color("#BF0A30"))
		s.Mention = s.Mention.Foreground(lipgloss.Color("#FFD700"))     // Gold
		s.Hyperlink = s.Hyperlink.Foreground(lipgloss.Color("#87CEEB")) // Sky blue
		s.UserList = s.UserList.BorderForeground(lipgloss.Color("#002868"))
		s.Me = s.Me.Foreground(lipgloss.Color("#BF0A30"))
		// Background and UI
		s.Background = lipgloss.NewStyle().Background(lipgloss.Color("#00203F")) // Deep navy
		s.Header = lipgloss.NewStyle().Background(lipgloss.Color("#BF0A30")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
		s.Footer = lipgloss.NewStyle().Background(lipgloss.Color("#00203F")).Foreground(lipgloss.Color("#FFD700"))
		s.Input = lipgloss.NewStyle().Background(lipgloss.Color("#002868")).Foreground(lipgloss.Color("#FFFFFF"))
		s.HelpOverlay = s.HelpOverlay.BorderForeground(lipgloss.Color("#BF0A30")).Background(lipgloss.Color("#00203F"))
	case "retro":
		s.User = s.User.Foreground(lipgloss.Color("#FF8800"))              // Orange
		s.Time = s.Time.Foreground(lipgloss.Color("#00FF00")).Faint(false) // Green
		s.Msg = s.Msg.Foreground(lipgloss.Color("#FFFFAA"))
		s.Box = s.Box.BorderForeground(lipgloss.Color("#FF8800"))
		s.Mention = s.Mention.Foreground(lipgloss.Color("#00FFFF"))     // Cyan
		s.Hyperlink = s.Hyperlink.Foreground(lipgloss.Color("#00FFFF")) // Cyan
		s.UserList = s.UserList.BorderForeground(lipgloss.Color("#FF8800"))
		s.Me = s.Me.Foreground(lipgloss.Color("#FF8800"))
		// Background and UI
		s.Background = lipgloss.NewStyle().Background(lipgloss.Color("#181818")) // Retro dark
		s.Header = lipgloss.NewStyle().Background(lipgloss.Color("#FF8800")).Foreground(lipgloss.Color("#181818")).Bold(true)
		s.Footer = lipgloss.NewStyle().Background(lipgloss.Color("#181818")).Foreground(lipgloss.Color("#00FF00"))
		s.Input = lipgloss.NewStyle().Background(lipgloss.Color("#222200")).Foreground(lipgloss.Color("#FFFFAA"))
		s.HelpOverlay = s.HelpOverlay.BorderForeground(lipgloss.Color("#FF8800")).Background(lipgloss.Color("#181818"))
	case "modern":
		s.User = s.User.Foreground(lipgloss.Color("#4F8EF7"))              // Blue
		s.Time = s.Time.Foreground(lipgloss.Color("#A0A0A0")).Faint(false) // Gray
		s.Msg = s.Msg.Foreground(lipgloss.Color("#E0E0E0"))
		s.Box = s.Box.BorderForeground(lipgloss.Color("#4F8EF7"))
		s.Mention = s.Mention.Foreground(lipgloss.Color("#FF5F5F"))     // Red
		s.Hyperlink = s.Hyperlink.Foreground(lipgloss.Color("#4A9EFF")) // Bright blue
		s.UserList = s.UserList.BorderForeground(lipgloss.Color("#4F8EF7"))
		s.Me = s.Me.Foreground(lipgloss.Color("#4F8EF7"))
		// Background and UI
		s.Background = lipgloss.NewStyle().Background(lipgloss.Color("#181C24")) // Modern dark blue-gray
		s.Header = lipgloss.NewStyle().Background(lipgloss.Color("#4F8EF7")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
		s.Footer = lipgloss.NewStyle().Background(lipgloss.Color("#181C24")).Foreground(lipgloss.Color("#4F8EF7"))
		s.Input = lipgloss.NewStyle().Background(lipgloss.Color("#23272E")).Foreground(lipgloss.Color("#E0E0E0"))
		s.HelpOverlay = s.HelpOverlay.BorderForeground(lipgloss.Color("#4F8EF7")).Background(lipgloss.Color("#181C24"))
	}
	return s
}

func renderMessages(msgs []shared.Message, styles themeStyles, username string, users []string, width int, twentyFourHour bool) string {
	const max = maxMessages
	if len(msgs) > max {
		msgs = msgs[len(msgs)-max:]
	}

	// CRITICAL FIX: Sort messages client-side to ensure consistent ordering
	// This handles cases where server-side ordering may be inconsistent
	sortMessagesByTimestamp(msgs)

	var b strings.Builder
	var prevDate string
	for _, msg := range msgs {
		sender := msg.Sender
		align := lipgloss.Left
		msgBoxStyle := lipgloss.NewStyle().Width(width - 4)
		if sender == username {
			align = lipgloss.Right
			msgBoxStyle = msgBoxStyle.Background(lipgloss.Color("#222244")).Foreground(lipgloss.Color("#FFFFFF"))
		} else {
			msgBoxStyle = msgBoxStyle.Background(lipgloss.Color("#222222")).Foreground(lipgloss.Color("#AAAAAA"))
		}
		// Date header if date changes
		dateStr := msg.CreatedAt.Format("2006-01-02")
		if dateStr != prevDate {
			b.WriteString(styles.Time.Render(dateStr) + "\n")
			prevDate = dateStr
		}
		// Time format
		timeFmt := "15:04:05"
		if !twentyFourHour {
			timeFmt = "03:04:05 PM"
		}
		timestamp := styles.Time.Render(msg.CreatedAt.Format(timeFmt))
		var content string
		if msg.Type == shared.FileMessageType && msg.File != nil {
			fileInfo := styles.Mention.Render("[File] ") + styles.User.Render(msg.File.Filename) + styles.Time.Render(fmt.Sprintf(" (%d bytes)", msg.File.Size))
			content = fileInfo + "\n" + styles.Msg.Render("Type :savefile "+msg.File.Filename+" to save.")
		} else {
			content = renderEmojis(msg.Content)
			// Render code blocks with syntax highlighting
			content = renderCodeBlocks(content)
			// Render hyperlinks
			content = renderHyperlinks(content, styles)
			// Improved mention highlighting: highlight if any @username in user list (case-insensitive)
			matches := mentionRegex.FindAllStringSubmatch(msg.Content, -1)
			highlight := false
			for _, m := range matches {
				if len(m) > 1 {
					for _, u := range users {
						if strings.EqualFold(m[1], u) {
							highlight = true
							break
						}
					}
					if highlight {
						break
					}
				}
			}
			if highlight {
				content = styles.Mention.Render(content)
			} else {
				content = styles.Msg.Render(content)
			}
		}
		meta := styles.User.Render(sender) + " " + timestamp
		wrapped := msgBoxStyle.Render(content)
		msgBlock := lipgloss.JoinVertical(lipgloss.Left, meta, wrapped)
		b.WriteString(msgBoxStyle.Align(align).Render(msgBlock) + "\n\n")
	}
	return b.String()
}

type wsMsg struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type wsErr error

// wsUsernameError represents a username-related connection error
type wsUsernameError struct {
	message string
}

func (e wsUsernameError) Error() string {
	return e.message
}

type wsConnected bool

type UserList struct {
	Users []string `json:"users"`
}

type codeSnippetMsg struct {
	content string
}

type fileSendMsg struct {
	filePath string
}

func (m *model) connectWebSocket(serverURL string) error {
	escapedUsername := url.QueryEscape(m.cfg.Username)
	fullURL := serverURL + "?username=" + escapedUsername

	log.Printf("Attempting to connect to: %s", fullURL)
	log.Printf("Username: %s, Admin: %v", m.cfg.Username, *isAdmin)
	if *isAdmin {
		log.Printf("Admin key: %s", *adminKey)
	}

	// Create custom dialer with TLS configuration
	dialer := websocket.DefaultDialer
	if *skipTLSVerify {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	log.Printf("Attempting WebSocket connection to: %s", fullURL)
	conn, resp, err := dialer.Dial(fullURL, nil)
	if err != nil {
		log.Printf("WebSocket dial failed - Error: %v (Type: %T)", err, err)
		if resp != nil {
			log.Printf("HTTP Response - Status: %d, Headers: %v", resp.StatusCode, resp.Header)
			// Try to read response body for more details
			if resp.Body != nil {
				body := make([]byte, 1024)
				if n, readErr := resp.Body.Read(body); readErr == nil && n > 0 {
					log.Printf("Response body: %s", string(body[:n]))
				}
				resp.Body.Close()
			}
		}

		// Check if this might be a duplicate username error based on response
		if resp != nil && resp.StatusCode == 403 {
			log.Printf("Connection forbidden - likely duplicate username")
			return wsUsernameError{message: "Username already taken - please choose a different username"}
		}

		return err
	}

	log.Printf("WebSocket connection established successfully")
	m.conn = conn
	m.connected = true
	m.banner = "✅ Connected to server!"
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.wg.Add(1)

	// Send handshake as first message
	handshake := shared.Handshake{
		Username: m.cfg.Username,
		Admin:    *isAdmin,
		AdminKey: "",
	}
	if *isAdmin {
		handshake.AdminKey = *adminKey
	}

	log.Printf("Sending handshake: %+v", handshake)
	if err := m.conn.WriteJSON(handshake); err != nil {
		log.Printf("Failed to send handshake: %v", err)
		return err
	}
	log.Printf("Handshake sent successfully")

	// Brief pause to allow server to process handshake and potentially close connection
	time.Sleep(100 * time.Millisecond)

	// Test if connection is still alive after handshake
	if err := m.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		log.Printf("Connection test failed after handshake: %v", err)
		log.Printf("Error type: %T", err)

		// Check different types of errors that might indicate connection was closed
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			if ce, ok := err.(*websocket.CloseError); ok {
				log.Printf("Close error detected - Code: %d, Text: '%s'", ce.Code, ce.Text)
				if strings.Contains(ce.Text, "Username already taken") || strings.Contains(ce.Text, "already taken") {
					return wsUsernameError{message: "Username already taken - please choose a different username"}
				}
			}
		}

		// Also check for "connection closed" type errors which might indicate duplicate username
		errStr := err.Error()
		log.Printf("Error string: '%s'", errStr)
		if strings.Contains(errStr, "use of closed network connection") ||
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "broken pipe") {
			// Connection was closed immediately after handshake - likely duplicate username
			log.Printf("Connection closed immediately after handshake - assuming duplicate username")
			return wsUsernameError{message: "Username already taken - please choose a different username"}
		}

		return err
	}

	// Set pong handler
	m.conn.SetPongHandler(func(appData string) error {
		return nil
	})

	// Start ping goroutine
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				_ = m.conn.WriteMessage(websocket.PingMessage, nil)
			}
		}
	}()

	go func() {
		defer m.wg.Done()
		for {
			select {
			case <-m.ctx.Done():
				return
			default:
				msgType, raw, err := conn.ReadMessage()
				if err != nil {
					// Check if it's a close error with a specific message
					if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						if ce, ok := err.(*websocket.CloseError); ok {
							log.Printf("WebSocket closed: %d - %s", ce.Code, ce.Text)
							// Check for duplicate username error
							if strings.Contains(ce.Text, "Username already taken") {
								m.msgChan <- wsUsernameError{message: ce.Text}
								return
							}
						}
					}

					// Check for the specific "bad close code" error that indicates duplicate username
					errStr := err.Error()
					if strings.Contains(errStr, "bad close code") {
						log.Printf("Detected bad close code - likely duplicate username: %v", err)
						m.msgChan <- wsUsernameError{message: "Username already taken - please choose a different username"}
						return
					}

					log.Printf("WebSocket read error: %v", err)
					m.msgChan <- wsErr(err)
					return
				}

				// Handle close messages explicitly
				if msgType == websocket.CloseMessage {
					log.Printf("Received close message: %s", string(raw))
					if strings.Contains(string(raw), "Username already taken") {
						m.msgChan <- wsUsernameError{message: string(raw)}
						return
					}
					m.msgChan <- wsErr(fmt.Errorf("connection closed: %s", string(raw)))
					return
				}

				log.Printf("Received message: %s", string(raw))

				// Try to unmarshal as shared.Message first
				var msg shared.Message
				if err := json.Unmarshal(raw, &msg); err == nil {
					if msg.Sender != "" {
						// Check if this is an encrypted message
						if m.useE2E && msg.Encrypted && msg.Content != "" {
							// Try to decode base64 encrypted content (nonce + encrypted_data)
							if decoded, err := base64.StdEncoding.DecodeString(msg.Content); err == nil && len(decoded) > 12 {
								// This might be encrypted content, try to decrypt it
								log.Printf("DEBUG: Detected potential encrypted content, attempting decryption")

								// Extract nonce (first 12 bytes) and encrypted data (rest)
								nonce := decoded[:12]
								encryptedData := decoded[12:]

								// Create an EncryptedMessage struct for decryption
								encryptedMsg := shared.EncryptedMessage{
									Sender:      msg.Sender,
									CreatedAt:   msg.CreatedAt,
									Encrypted:   encryptedData,
									Nonce:       nonce,
									IsEncrypted: true,
									Type:        msg.Type,
								}

								conversationID := "global" // Same as sending
								decryptedMsg, err := m.keystore.DecryptMessage(&encryptedMsg, conversationID)
								if err != nil {
									log.Printf("DEBUG: Failed to decrypt message: %v", err)
									// Keep original message but mark as failed decryption
									msg.Content = "[ENCRYPTED - DECRYPTION FAILED]"
									m.msgChan <- msg
									continue
								}

								log.Printf("DEBUG: Successfully decrypted message")
								m.msgChan <- *decryptedMsg
								continue
							}
						}

						// Regular message (not encrypted or decryption not needed)
						m.msgChan <- msg
						continue
					}
				}

				// Then try as wsMsg
				var ws wsMsg
				if err := json.Unmarshal(raw, &ws); err == nil && ws.Type != "" {
					log.Printf("Received wsMsg type: %s", ws.Type)
					m.msgChan <- ws
					continue
				}

				log.Printf("Could not parse message: %s", string(raw))
			}
		}
	}()
	return nil
}

func (m *model) closeWebSocket() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.conn != nil {
		m.conn.Close()
	}
	m.wg.Wait()
}

func (m *model) Init() tea.Cmd {
	m.msgChan = make(chan tea.Msg, 10) // buffered to avoid blocking
	m.reconnectDelay = time.Second     // reset on each Init
	return func() tea.Msg {
		err := m.connectWebSocket(m.cfg.ServerURL)
		if err != nil {
			log.Printf("connectWebSocket returned error: %v (type: %T)", err, err)
			// Preserve wsUsernameError type
			if usernameErr, ok := err.(wsUsernameError); ok {
				log.Printf("Detected username error: %s", usernameErr.message)
				return usernameErr
			}
			log.Printf("Returning generic wsErr")
			return wsErr(err)
		}
		return wsConnected(true)
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case wsConnected:
		m.connected = true
		m.banner = "✅ Connected to server!"
		m.reconnectDelay = time.Second // reset on success
		return m, m.listenWebSocket()
	case wsMsg:
		if v.Type == "userlist" {
			var ul UserList
			if err := json.Unmarshal(v.Data, &ul); err == nil {
				m.users = ul.Users
				userListWidth := 18
				m.userListViewport.SetContent(renderUserList(m.users, m.cfg.Username, m.styles, userListWidth, *isAdmin, m.selectedUserIndex))
			}
			return m, m.listenWebSocket()
		}
		if v.Type == "auth_failed" {
			log.Printf("Authentication failed - admin key rejected")
			var authFail map[string]string
			if err := json.Unmarshal(v.Data, &authFail); err == nil {
				log.Printf("Auth failure reason: %s", authFail["reason"])
			}
			fmt.Printf("❌ Authentication failed: %s\n", authFail["reason"])
			fmt.Printf("Check your --admin-key matches the server's MARCHAT_ADMIN_KEY\n")
			os.Exit(1)
		}
		return m, m.listenWebSocket()
	case codeSnippetMsg:
		// Handle code snippet message from the code snippet interface
		m.sending = true
		if m.conn != nil {
			if m.useE2E {
				// Use E2E encryption for global chat
				recipients := m.users
				if len(recipients) == 0 {
					recipients = []string{m.cfg.Username}
				}
				if err := debugEncryptAndSend(recipients, v.content, m.conn, m.keystore, m.cfg.Username); err != nil {
					log.Printf("Failed to send code snippet: %v", err)
					m.banner = "❌ Failed to send code snippet"
				}
			} else {
				// Send plain text message
				msg := shared.Message{Sender: m.cfg.Username, Content: v.content}
				if err := debugWebSocketWrite(m.conn, msg); err != nil {
					log.Printf("Failed to send code snippet: %v", err)
					m.banner = "❌ Failed to send code snippet"
				}
			}
		}
		m.sending = false
		m.showCodeSnippet = false
		return m, m.listenWebSocket()
	case fileSendMsg:
		// Handle file send message from the file picker interface
		m.sending = true
		if m.conn != nil {
			// Read the file
			data, err := os.ReadFile(v.filePath)
			if err != nil {
				m.banner = "❌ Failed to read file: " + err.Error()
				m.sending = false
				m.showFilePicker = false
				return m, nil
			}

			// Check file size (configurable limit; default 1MB)
			var maxBytes int64 = 1024 * 1024
			if envBytes := os.Getenv("MARCHAT_MAX_FILE_BYTES"); envBytes != "" {
				if v, err := strconv.ParseInt(envBytes, 10, 64); err == nil && v > 0 {
					maxBytes = v
				}
			} else if envMB := os.Getenv("MARCHAT_MAX_FILE_MB"); envMB != "" {
				if v, err := strconv.ParseInt(envMB, 10, 64); err == nil && v > 0 {
					maxBytes = v * 1024 * 1024
				}
			}
			if int64(len(data)) > maxBytes {
				// Try to format friendly message in MB when divisible, else show bytes
				limitMsg := fmt.Sprintf("%d bytes", maxBytes)
				if maxBytes%(1024*1024) == 0 {
					limitMsg = fmt.Sprintf("%dMB", maxBytes/(1024*1024))
				}
				m.banner = "❌ File too large (max " + limitMsg + ")"
				m.sending = false
				m.showFilePicker = false
				return m, nil
			}

			filename := filepath.Base(v.filePath)
			msg := shared.Message{
				Sender:    m.cfg.Username,
				Type:      shared.FileMessageType,
				CreatedAt: time.Now(),
				File: &shared.FileMeta{
					Filename: filename,
					Size:     int64(len(data)),
					Data:     data,
				},
			}

			err = m.conn.WriteJSON(msg)
			if err != nil {
				m.banner = "❌ Failed to send file (connection lost)"
				m.sending = false
				m.showFilePicker = false
				return m, m.listenWebSocket()
			}

			m.banner = "File sent: " + filename
		}
		m.sending = false
		m.showFilePicker = false
		return m, m.listenWebSocket()
	case shared.Message:
		// Check if we should notify for this message
		if shouldNotify, level := m.shouldNotify(v); shouldNotify {
			m.notificationManager.Notify(v.Sender, v.Content, level)
		}

		if len(m.messages) >= maxMessages {
			m.messages = m.messages[len(m.messages)-maxMessages+1:]
		}
		m.messages = append(m.messages, v)

		// CRITICAL FIX: Sort messages after adding new ones to maintain order
		sortMessagesByTimestamp(m.messages)

		if v.Type == shared.FileMessageType && v.File != nil {
			if m.receivedFiles == nil {
				m.receivedFiles = make(map[string]*shared.FileMeta)
			}
			m.receivedFiles[v.File.Filename] = v.File
		}
		m.viewport.SetContent(renderMessages(m.messages, m.styles, m.cfg.Username, m.users, m.viewport.Width, m.twentyFourHour))
		m.viewport.GotoBottom()
		m.sending = false
		return m, m.listenWebSocket()
	case wsUsernameError:
		log.Printf("Handling wsUsernameError: %s", v.message)
		m.connected = false
		m.banner = "❌ " + v.message + " - Please restart with a different username"
		m.closeWebSocket()
		// Don't attempt to reconnect for username errors
		return m, nil
	case wsErr:
		m.connected = false
		m.banner = "🚫 Connection lost. Reconnecting..."
		m.closeWebSocket()
		delay := m.reconnectDelay
		if delay < reconnectMaxDelay {
			m.reconnectDelay *= 2
			if m.reconnectDelay > reconnectMaxDelay {
				m.reconnectDelay = reconnectMaxDelay
			}
		}
		return m, tea.Tick(delay, func(time.Time) tea.Msg {
			return m.Init()()
		})
	case tea.KeyMsg:
		switch {
		case key.Matches(v, m.keys.Help):
			// Close any open menus first
			if m.showDBMenu {
				m.showDBMenu = false
				return m, nil
			}
			if m.showCodeSnippet {
				m.showCodeSnippet = false
				return m, nil
			}
			m.showHelp = !m.showHelp
			if m.showHelp {
				// Set help content when help is shown
				m.helpViewport.SetContent(m.generateHelpContent())
				m.helpViewport.GotoTop()
			}
			return m, nil
		case m.showCodeSnippet:
			// Handle code snippet interface
			var cmd tea.Cmd
			updatedModel, cmd := m.codeSnippetModel.Update(v)
			if csModel, ok := updatedModel.(codeSnippetModel); ok {
				m.codeSnippetModel = csModel
			}
			return m, cmd
		case m.showFilePicker:
			// Handle file picker interface
			var cmd tea.Cmd
			updatedModel, cmd := m.filePickerModel.Update(v)
			if fpModel, ok := updatedModel.(filePickerModel); ok {
				m.filePickerModel = fpModel
			}
			return m, cmd
		case key.Matches(v, m.keys.Quit):
			// If waiting for plugin input, cancel it
			if m.pendingPluginAction != "" {
				m.pendingPluginAction = ""
				m.textarea.SetValue("")
				m.banner = "Plugin action cancelled"
				return m, nil
			}
			// If help is open, close it instead of quitting
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			// If code snippet is open, close it instead of quitting
			if m.showCodeSnippet {
				m.showCodeSnippet = false
				return m, nil
			}
			// If file picker is open, close it instead of quitting
			if m.showFilePicker {
				m.showFilePicker = false
				return m, nil
			}
			// If a menu is open or user selected, clear it instead of quitting
			if m.showDBMenu || m.selectedUserIndex >= 0 {
				m.showDBMenu = false
				m.selectedUserIndex = -1
				m.selectedUser = ""
				return m, nil
			}
			// ESC no longer quits - use :q command instead
			return m, nil
		case key.Matches(v, m.keys.DatabaseMenu):
			// Only show database menu if admin and no other menus are open
			if *isAdmin && !m.showHelp {
				m.showDBMenu = !m.showDBMenu
				if m.showDBMenu {
					m.dbMenuViewport.SetContent(m.generateDBMenuContent())
					m.dbMenuViewport.GotoTop()
				}
			}
			return m, nil
		// Plugin management hotkey handlers (must be before SelectUser to prevent Ctrl+Shift+U from matching Ctrl+U)
		case key.Matches(v, m.keys.PluginList):
			if *isAdmin {
				return m.executePluginCommand(":list")
			}
			return m, nil
		case key.Matches(v, m.keys.PluginStore):
			if *isAdmin {
				return m.executePluginCommand(":store")
			}
			return m, nil
		case key.Matches(v, m.keys.PluginRefresh):
			if *isAdmin {
				return m.executePluginCommand(":refresh")
			}
			return m, nil
		case key.Matches(v, m.keys.PluginInstall):
			if *isAdmin {
				return m.promptForPluginName("install")
			}
			return m, nil
		case key.Matches(v, m.keys.PluginUninstall):
			if *isAdmin {
				return m.promptForPluginName("uninstall")
			}
			return m, nil
		case key.Matches(v, m.keys.PluginEnable):
			if *isAdmin {
				return m.promptForPluginName("enable")
			}
			return m, nil
		case key.Matches(v, m.keys.PluginDisable):
			if *isAdmin {
				return m.promptForPluginName("disable")
			}
			return m, nil
		// Hotkey alternatives for common commands
		case key.Matches(v, m.keys.SendFileHotkey):
			// Open file picker (same as :sendfile without path)
			m.textarea.SetValue("")
			m.showFilePicker = true
			m.filePickerModel = newFilePickerModel(m.styles, m.width, m.height,
				func(filePath string) {
					select {
					case m.msgChan <- fileSendMsg{filePath: filePath}:
					default:
						log.Printf("Failed to send file message")
					}
				},
				func() {
					m.showFilePicker = false
				})
			return m, nil
		case key.Matches(v, m.keys.ThemeHotkey):
			// Cycle through themes (built-in + custom)
			themes := ListAllThemes()
			currentIndex := 0
			for i, theme := range themes {
				if theme == m.cfg.Theme {
					currentIndex = i
					break
				}
			}
			nextIndex := (currentIndex + 1) % len(themes)
			newTheme := themes[nextIndex]
			m.cfg.Theme = newTheme
			m.styles = getThemeStyles(m.cfg.Theme)
			_ = config.SaveConfig(m.configFilePath, m.cfg)

			// Update profile with new theme
			_ = m.updateProfileTheme(newTheme)

			// Show theme info in banner
			themeInfo := GetThemeInfo(m.cfg.Theme)
			m.banner = fmt.Sprintf("Theme: %s", themeInfo)

			// Redraw viewport and user list with new theme
			m.viewport.SetContent(renderMessages(m.messages, m.styles, m.cfg.Username, m.users, m.viewport.Width, m.twentyFourHour))
			m.userListViewport.SetContent(renderUserList(m.users, m.cfg.Username, m.styles, m.width, *isAdmin, m.selectedUserIndex))
			return m, nil
		case key.Matches(v, m.keys.TimeFormatHotkey):
			// Toggle time format
			m.twentyFourHour = !m.twentyFourHour
			m.cfg.TwentyFourHour = m.twentyFourHour
			_ = config.SaveConfig(m.configFilePath, m.cfg)
			m.banner = "Timestamp format: " + map[bool]string{true: "24h", false: "12h"}[m.twentyFourHour]
			m.viewport.SetContent(renderMessages(m.messages, m.styles, m.cfg.Username, m.users, m.viewport.Width, m.twentyFourHour))
			return m, nil
		case key.Matches(v, m.keys.ClearHotkey):
			// Clear chat history
			m.messages = nil
			m.viewport.SetContent("")
			m.banner = "Chat cleared."
			return m, nil
		case key.Matches(v, m.keys.CodeSnippetHotkey):
			// Launch code snippet interface
			m.textarea.SetValue("")
			m.showCodeSnippet = true
			m.codeSnippetModel = newCodeSnippetModel(m.styles, m.width, m.height,
				func(code string) {
					select {
					case m.msgChan <- codeSnippetMsg{content: code}:
					default:
						log.Printf("Failed to send code snippet message")
					}
				},
				func() {
					m.showCodeSnippet = false
				})
			return m, nil
		case key.Matches(v, m.keys.NotifyDesktop):
			// Toggle desktop notifications (Alt+N)
			if !m.notificationManager.IsDesktopSupported() {
				m.banner = "Desktop notifications not supported on this platform"
			} else {
				enabled := m.notificationManager.ToggleDesktop()
				status := "disabled"
				if enabled {
					status = "enabled"
					m.notificationManager.Notify("System", "Desktop notifications enabled", NotificationLevelInfo)
				}
				m.banner = fmt.Sprintf("Desktop notifications %s", status)
				// Save to config
				notifCfg := m.notificationManager.GetConfig()
				notificationConfigToConfig(notifCfg, &m.cfg)
				_ = config.SaveConfig(m.configFilePath, m.cfg)
			}
			return m, nil
		case key.Matches(v, m.keys.SelectUser):
			// Cycle through users for admin selection
			if *isAdmin && !m.showHelp && !m.showDBMenu && len(m.users) > 0 {
				// Find next user that isn't the current user
				for i := 0; i < len(m.users); i++ {
					m.selectedUserIndex = (m.selectedUserIndex + 1) % len(m.users)
					if m.users[m.selectedUserIndex] != m.cfg.Username {
						m.selectedUser = m.users[m.selectedUserIndex]
						m.banner = fmt.Sprintf("Selected user: %s", m.selectedUser)
						break
					}
				}
				// If we only have ourselves in the list, clear selection
				if m.users[m.selectedUserIndex] == m.cfg.Username {
					m.selectedUserIndex = -1
					m.selectedUser = ""
					m.banner = "No other users to select"
				}
			}
			return m, nil
		case key.Matches(v, m.keys.BanUser):
			if *isAdmin && m.selectedUser != "" && m.selectedUser != m.cfg.Username {
				return m.executeAdminAction("ban", m.selectedUser)
			}
			return m, nil
		case key.Matches(v, m.keys.KickUser):
			if *isAdmin && m.selectedUser != "" && m.selectedUser != m.cfg.Username {
				return m.executeAdminAction("kick", m.selectedUser)
			}
			return m, nil
		case key.Matches(v, m.keys.UnbanUser):
			if *isAdmin {
				// For unban, we need to prompt for username since banned users aren't in the list
				return m.promptForUsername("unban")
			}
			return m, nil
		case key.Matches(v, m.keys.AllowUser):
			if *isAdmin {
				// For allow, we need to prompt for username since kicked users aren't in the list
				return m.promptForUsername("allow")
			}
			return m, nil
		case key.Matches(v, m.keys.ForceDisconnectUser):
			if *isAdmin && m.selectedUser != "" && m.selectedUser != m.cfg.Username {
				return m.executeAdminAction("forcedisconnect", m.selectedUser)
			}
			return m, nil
		case key.Matches(v, m.keys.ScrollUp):
			if m.showHelp {
				m.helpViewport.ScrollUp(1)
			} else if m.textarea.Focused() {
				m.viewport.ScrollUp(1)
			} else {
				m.userListViewport.ScrollUp(1)
			}
			return m, nil
		case key.Matches(v, m.keys.ScrollDown):
			if m.showHelp {
				m.helpViewport.ScrollDown(1)
			} else if m.textarea.Focused() {
				m.viewport.ScrollDown(1)
			} else {
				m.userListViewport.ScrollDown(1)
			}
			return m, nil
		case key.Matches(v, m.keys.PageUp):
			if m.showHelp {
				m.helpViewport.ScrollUp(m.helpViewport.Height)
			} else {
				m.viewport.ScrollUp(m.viewport.Height)
			}
			return m, nil
		case key.Matches(v, m.keys.PageDown):
			if m.showHelp {
				m.helpViewport.ScrollDown(m.helpViewport.Height)
			} else {
				m.viewport.ScrollDown(m.viewport.Height)
			}
			return m, nil
		case key.Matches(v, m.keys.Copy): // Custom Copy
			if m.textarea.Focused() {
				text := m.textarea.Value()
				if text != "" {
					err := safeClipboardOperation(func() error {
						return clipboard.WriteAll(text)
					}, 2*time.Second)

					if err != nil {
						if isTermux() {
							m.banner = fmt.Sprintf("⚠️ Clipboard unavailable in Termux. Text: %s", text)
						} else if err == context.DeadlineExceeded {
							m.banner = "⚠️ Clipboard operation timed out"
						} else {
							m.banner = "❌ Failed to copy to clipboard: " + err.Error()
						}
					} else {
						m.banner = "✅ Copied to clipboard"
					}
				}
				return m, nil
			}
			return m, nil
		case key.Matches(v, m.keys.Paste): // Custom Paste
			if m.textarea.Focused() {
				var text string
				err := safeClipboardOperation(func() error {
					var readErr error
					text, readErr = clipboard.ReadAll()
					return readErr
				}, 2*time.Second)

				if err != nil {
					if isTermux() {
						m.banner = "⚠️ Clipboard unavailable in Termux. Paste manually or use other methods."
					} else if err == context.DeadlineExceeded {
						m.banner = "⚠️ Clipboard operation timed out"
					} else {
						m.banner = "❌ Failed to paste from clipboard: " + err.Error()
					}
				} else {
					m.textarea.SetValue(m.textarea.Value() + text)
					m.banner = "✅ Pasted from clipboard"
				}
				return m, nil
			}
			return m, nil
		case key.Matches(v, m.keys.Cut): // Custom Cut
			if m.textarea.Focused() {
				text := m.textarea.Value()
				if text != "" {
					err := safeClipboardOperation(func() error {
						return clipboard.WriteAll(text)
					}, 2*time.Second)

					if err != nil {
						if isTermux() {
							m.banner = fmt.Sprintf("⚠️ Clipboard unavailable in Termux. Text cleared: %s", text)
						} else if err == context.DeadlineExceeded {
							m.banner = "⚠️ Clipboard operation timed out"
						} else {
							m.banner = "❌ Failed to cut to clipboard: " + err.Error()
						}
					} else {
						m.banner = "✅ Cut to clipboard"
					}
					m.textarea.SetValue("")
				}
				return m, nil
			}
			return m, nil
		case key.Matches(v, m.keys.SelectAll): // Custom Select All
			if m.textarea.Focused() {
				text := m.textarea.Value()
				if text != "" {
					err := safeClipboardOperation(func() error {
						return clipboard.WriteAll(text)
					}, 2*time.Second)

					if err != nil {
						if isTermux() {
							m.banner = fmt.Sprintf("⚠️ Clipboard unavailable in Termux. Full text: %s", text)
						} else if err == context.DeadlineExceeded {
							m.banner = "⚠️ Clipboard operation timed out"
						} else {
							m.banner = "❌ Failed to select all: " + err.Error()
						}
					} else {
						m.banner = "✅ Selected all and copied to clipboard"
					}
				}
				return m, nil
			}
			return m, nil
		case key.Matches(v, m.keys.Send):
			text := m.textarea.Value()

			// Check if we're waiting for plugin name input
			if m.pendingPluginAction != "" {
				pluginName := strings.TrimSpace(text)
				if pluginName == "" {
					m.banner = "❌ Plugin name cannot be empty"
					m.textarea.SetValue("")
					m.pendingPluginAction = ""
					return m, nil
				}

				// Build the command based on the pending action
				var command string
				switch m.pendingPluginAction {
				case "install":
					command = fmt.Sprintf(":install %s", pluginName)
				case "uninstall":
					command = fmt.Sprintf(":uninstall %s", pluginName)
				case "enable":
					command = fmt.Sprintf(":enable %s", pluginName)
				case "disable":
					command = fmt.Sprintf(":disable %s", pluginName)
				}

				// Clear the textarea and pending action
				m.textarea.SetValue("")
				m.pendingPluginAction = ""

				// Execute the plugin command
				return m.executePluginCommand(command)
			}

			if text == ":sendfile" {
				// Open file picker when no path provided
				m.textarea.SetValue("")
				m.showFilePicker = true
				// Initialize file picker model
				m.filePickerModel = newFilePickerModel(m.styles, m.width, m.height,
					func(filePath string) {
						// Send the file using a channel to avoid race conditions
						select {
						case m.msgChan <- fileSendMsg{filePath: filePath}:
						default:
							log.Printf("Failed to send file message")
						}
					},
					func() {
						// Cancel - just hide the file picker interface
						m.showFilePicker = false
					})
				return m, nil
			}
			if strings.HasPrefix(text, ":sendfile ") {
				parts := strings.SplitN(text, " ", 2)
				if len(parts) == 2 {
					path := strings.TrimSpace(parts[1])
					if path != "" {
						// Send file with provided path (existing functionality)
						data, err := os.ReadFile(path)
						if err != nil {
							m.banner = "❌ Failed to read file: " + err.Error()
							m.textarea.SetValue("")
							return m, nil
						}
						// Enforce configurable file size limit (default 1MB)
						var maxBytes int64 = 1024 * 1024
						if envBytes := os.Getenv("MARCHAT_MAX_FILE_BYTES"); envBytes != "" {
							if v, err := strconv.ParseInt(envBytes, 10, 64); err == nil && v > 0 {
								maxBytes = v
							}
						} else if envMB := os.Getenv("MARCHAT_MAX_FILE_MB"); envMB != "" {
							if v, err := strconv.ParseInt(envMB, 10, 64); err == nil && v > 0 {
								maxBytes = v * 1024 * 1024
							}
						}
						if int64(len(data)) > maxBytes {
							limitMsg := fmt.Sprintf("%d bytes", maxBytes)
							if maxBytes%(1024*1024) == 0 {
								limitMsg = fmt.Sprintf("%dMB", maxBytes/(1024*1024))
							}
							m.banner = "❌ File too large (max " + limitMsg + ")"
							m.textarea.SetValue("")
							return m, nil
						}
						filename := filepath.Base(path)
						msg := shared.Message{
							Sender:    m.cfg.Username,
							Type:      shared.FileMessageType,
							CreatedAt: time.Now(),
							File: &shared.FileMeta{
								Filename: filename,
								Size:     int64(len(data)),
								Data:     data,
							},
						}
						if m.conn != nil {
							err := m.conn.WriteJSON(msg)
							if err != nil {
								m.banner = "❌ Failed to send file (connection lost)"
								m.textarea.SetValue("")
								return m, m.listenWebSocket()
							}
							m.banner = "File sent: " + filename
						}
						m.textarea.SetValue("")
						return m, m.listenWebSocket()
					}
				}
				return m, nil
			}
			if strings.HasPrefix(text, ":savefile ") {
				filename := strings.TrimSpace(strings.TrimPrefix(text, ":savefile "))
				if m.receivedFiles == nil || m.receivedFiles[filename] == nil {
					m.banner = "❌ No files received yet."
					m.textarea.SetValue("")
					return m, nil
				}
				file := m.receivedFiles[filename]
				// Check for duplicate filenames and append suffix if needed
				saveName := file.Filename
				base := saveName
				ext := ""
				if dot := strings.LastIndex(saveName, "."); dot != -1 {
					base = saveName[:dot]
					ext = saveName[dot:]
				}
				tryName := saveName
				for i := 1; ; i++ {
					if _, err := os.Stat(tryName); os.IsNotExist(err) {
						saveName = tryName
						break
					}
					tryName = fmt.Sprintf("%s[%d]%s", base, i, ext)
				}
				err := os.WriteFile(saveName, file.Data, 0644)
				if err != nil {
					m.banner = "❌ Failed to save file: " + err.Error()
				} else {
					m.banner = "✅ File saved as: " + saveName
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":themes" {
				// List all available themes as a system message
				themes := ListAllThemes()
				var themeList strings.Builder
				themeList.WriteString("📋 Available themes:\n\n")
				for _, themeName := range themes {
					themeList.WriteString("  • ")
					themeList.WriteString(GetThemeInfo(themeName))
					if themeName == m.cfg.Theme {
						themeList.WriteString(" ⭐ [current]")
					}
					themeList.WriteString("\n")
				}
				themeList.WriteString("\nUse :theme <name> to switch or Ctrl+T to cycle")

				// Add as a system message
				systemMsg := shared.Message{
					Sender:    "System",
					Content:   themeList.String(),
					CreatedAt: time.Now(),
					Type:      shared.TextMessage,
				}
				if len(m.messages) >= maxMessages {
					m.messages = m.messages[len(m.messages)-maxMessages+1:]
				}
				m.messages = append(m.messages, systemMsg)
				m.viewport.SetContent(renderMessages(m.messages, m.styles, m.cfg.Username, m.users, m.viewport.Width, m.twentyFourHour))
				m.viewport.GotoBottom()

				m.textarea.SetValue("")
				return m, nil
			}
			if strings.HasPrefix(text, ":theme ") {
				parts := strings.SplitN(text, " ", 2)
				if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
					themeName := strings.TrimSpace(parts[1])

					// Check if theme exists using case-insensitive lookup
					allThemes := ListAllThemes()
					themeExists := false
					actualThemeName := themeName

					// First try exact match
					for _, t := range allThemes {
						if t == themeName {
							themeExists = true
							actualThemeName = t
							break
						}
					}

					// If not found, try case-insensitive match for custom themes
					if !themeExists {
						if key, _, found := GetCustomThemeByName(themeName); found {
							themeExists = true
							actualThemeName = key
						}
					}

					if !themeExists {
						m.banner = fmt.Sprintf("Theme '%s' not found. Use :themes to list available themes.", themeName)
					} else {
						m.cfg.Theme = actualThemeName
						m.styles = getThemeStyles(m.cfg.Theme)
						_ = config.SaveConfig(m.configFilePath, m.cfg)

						// Update profile with new theme
						_ = m.updateProfileTheme(actualThemeName)

						m.banner = fmt.Sprintf("Theme changed to: %s", GetThemeInfo(actualThemeName))

						// Redraw viewport and user list with new theme
						m.viewport.SetContent(renderMessages(m.messages, m.styles, m.cfg.Username, m.users, m.viewport.Width, m.twentyFourHour))
						m.userListViewport.SetContent(renderUserList(m.users, m.cfg.Username, m.styles, m.width, *isAdmin, m.selectedUserIndex))
					}
				} else {
					m.banner = "Please provide a theme name. Use :themes to list available themes."
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":clear" {
				m.messages = nil
				m.viewport.SetContent("")
				m.banner = "Chat cleared."
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":q" {
				m.closeWebSocket()
				m.textarea.SetValue("")
				return m, tea.Quit
			}
			// Individual E2E encryption commands removed - only global E2E encryption supported
			if text == ":time" {
				m.twentyFourHour = !m.twentyFourHour
				m.cfg.TwentyFourHour = m.twentyFourHour
				_ = config.SaveConfig(m.configFilePath, m.cfg)
				m.banner = "Timestamp format: " + map[bool]string{true: "24h", false: "12h"}[m.twentyFourHour]
				m.viewport.SetContent(renderMessages(m.messages, m.styles, m.cfg.Username, m.users, m.viewport.Width, m.twentyFourHour))
				m.viewport.GotoBottom()
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":bell" {
				enabled := m.notificationManager.ToggleBell()
				status := "disabled"
				if enabled {
					status = "enabled"
					// Test notification
					m.notificationManager.Notify("System", "Bell test", NotificationLevelInfo)
				}
				m.banner = fmt.Sprintf("Message bell %s", status)
				// Save to config
				notifCfg := m.notificationManager.GetConfig()
				notificationConfigToConfig(notifCfg, &m.cfg)
				_ = config.SaveConfig(m.configFilePath, m.cfg)
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":bell-mention" {
				enabled := m.notificationManager.ToggleBellOnMention()
				var status string
				if enabled {
					status = "enabled (mention only)"
					// Test notification
					m.notificationManager.Notify("System", "Bell test", NotificationLevelMention)
				} else {
					status = "enabled (all messages)"
				}
				m.banner = fmt.Sprintf("Bell notifications %s", status)
				// Save to config
				notifCfg := m.notificationManager.GetConfig()
				notificationConfigToConfig(notifCfg, &m.cfg)
				_ = config.SaveConfig(m.configFilePath, m.cfg)
				m.textarea.SetValue("")
				return m, nil
			}

			// New enhanced notification commands
			if strings.HasPrefix(text, ":notify-mode ") {
				mode := strings.TrimSpace(strings.TrimPrefix(text, ":notify-mode "))
				switch mode {
				case "none":
					m.notificationManager.SetMode(NotificationModeNone)
					m.banner = "Notifications disabled"
				case "bell":
					m.notificationManager.SetMode(NotificationModeBell)
					m.banner = "Notifications: Bell only"
					m.notificationManager.Notify("System", "Bell mode test", NotificationLevelInfo)
				case "desktop":
					if m.notificationManager.IsDesktopSupported() {
						m.notificationManager.SetMode(NotificationModeDesktop)
						m.banner = "Notifications: Desktop only"
						m.notificationManager.Notify("System", "Desktop notification test", NotificationLevelInfo)
					} else {
						m.banner = "Desktop notifications not supported on this platform"
					}
				case "both":
					if m.notificationManager.IsDesktopSupported() {
						m.notificationManager.SetMode(NotificationModeBoth)
						m.banner = "Notifications: Bell + Desktop"
						m.notificationManager.Notify("System", "Combined notification test", NotificationLevelInfo)
					} else {
						m.banner = "Desktop notifications not supported, using bell only"
						m.notificationManager.SetMode(NotificationModeBell)
					}
				default:
					m.banner = "Usage: :notify-mode <none|bell|desktop|both>"
					m.textarea.SetValue("")
					return m, nil
				}
				// Save to config
				notifCfg := m.notificationManager.GetConfig()
				notificationConfigToConfig(notifCfg, &m.cfg)
				_ = config.SaveConfig(m.configFilePath, m.cfg)
				m.textarea.SetValue("")
				return m, nil
			}

			if text == ":notify-desktop" {
				if !m.notificationManager.IsDesktopSupported() {
					m.banner = "Desktop notifications not supported on this platform"
				} else {
					enabled := m.notificationManager.ToggleDesktop()
					status := "disabled"
					if enabled {
						status = "enabled"
						m.notificationManager.Notify("System", "Desktop notifications enabled", NotificationLevelInfo)
					}
					m.banner = fmt.Sprintf("Desktop notifications %s", status)
					// Save to config
					notifCfg := m.notificationManager.GetConfig()
					notificationConfigToConfig(notifCfg, &m.cfg)
					_ = config.SaveConfig(m.configFilePath, m.cfg)
				}
				m.textarea.SetValue("")
				return m, nil
			}

			if strings.HasPrefix(text, ":quiet ") {
				parts := strings.Fields(text)
				if len(parts) == 3 {
					start, err1 := strconv.Atoi(parts[1])
					end, err2 := strconv.Atoi(parts[2])
					if err1 == nil && err2 == nil && start >= 0 && start < 24 && end >= 0 && end < 24 {
						m.notificationManager.SetQuietHours(true, start, end)
						m.banner = fmt.Sprintf("Quiet hours enabled: %02d:00 to %02d:00", start, end)
						// Save to config
						notifCfg := m.notificationManager.GetConfig()
						notificationConfigToConfig(notifCfg, &m.cfg)
						_ = config.SaveConfig(m.configFilePath, m.cfg)
					} else {
						m.banner = "Invalid hours (use 0-23). Usage: :quiet 22 8"
					}
				} else {
					m.banner = "Usage: :quiet <start-hour> <end-hour> (e.g., :quiet 22 8)"
				}
				m.textarea.SetValue("")
				return m, nil
			}

			if text == ":quiet-off" {
				m.notificationManager.SetQuietHours(false, 22, 8)
				m.banner = "Quiet hours disabled"
				// Save to config
				notifCfg := m.notificationManager.GetConfig()
				notificationConfigToConfig(notifCfg, &m.cfg)
				_ = config.SaveConfig(m.configFilePath, m.cfg)
				m.textarea.SetValue("")
				return m, nil
			}

			if strings.HasPrefix(text, ":focus") {
				parts := strings.Fields(text)
				if len(parts) == 1 {
					// Default to 30 minutes
					m.notificationManager.EnableFocusMode(30 * time.Minute)
					m.banner = "Focus mode enabled for 30 minutes"
				} else if len(parts) == 2 {
					durationStr := parts[1]
					duration, err := time.ParseDuration(durationStr)
					if err != nil {
						m.banner = "Invalid duration. Examples: 30m, 1h, 2h30m"
					} else {
						m.notificationManager.EnableFocusMode(duration)
						m.banner = fmt.Sprintf("Focus mode enabled for %s", duration)
					}
				} else {
					m.banner = "Usage: :focus [duration] (e.g., :focus 30m, :focus 1h)"
				}
				m.textarea.SetValue("")
				return m, nil
			}

			if text == ":focus-off" {
				m.notificationManager.DisableFocusMode()
				m.banner = "Focus mode disabled"
				m.textarea.SetValue("")
				return m, nil
			}

			if text == ":notify-status" {
				notifCfg := m.notificationManager.GetConfig()
				var mode string
				switch notifCfg.Mode {
				case NotificationModeNone:
					mode = "none"
				case NotificationModeBell:
					mode = "bell"
				case NotificationModeDesktop:
					mode = "desktop"
				case NotificationModeBoth:
					mode = "both"
				}
				statusLines := []string{
					fmt.Sprintf("Mode: %s", mode),
					fmt.Sprintf("Bell: %t (mention-only: %t)", notifCfg.BellEnabled, notifCfg.BellOnMention),
					fmt.Sprintf("Desktop: %t (supported: %t)", notifCfg.DesktopEnabled, m.notificationManager.IsDesktopSupported()),
				}
				if notifCfg.QuietHoursEnabled {
					statusLines = append(statusLines, fmt.Sprintf("Quiet hours: %02d:00 - %02d:00", notifCfg.QuietHoursStart, notifCfg.QuietHoursEnd))
				}
				if notifCfg.FocusModeEnabled {
					remaining := time.Until(notifCfg.FocusModeUntil)
					if remaining > 0 {
						statusLines = append(statusLines, fmt.Sprintf("Focus mode: active (%s remaining)", remaining.Round(time.Minute)))
					}
				}
				m.banner = strings.Join(statusLines, " | ")
				m.textarea.SetValue("")
				return m, nil
			}

			if text == ":code" {
				// Launch code snippet interface
				m.textarea.SetValue("")
				m.showCodeSnippet = true
				// Initialize code snippet model
				m.codeSnippetModel = newCodeSnippetModel(m.styles, m.width, m.height,
					func(code string) {
						// Send the code as a message using a channel to avoid race conditions
						select {
						case m.msgChan <- codeSnippetMsg{content: code}:
						default:
							log.Printf("Failed to send code snippet message")
						}
					},
					func() {
						// Cancel - just hide the code snippet interface
						m.showCodeSnippet = false
					})
				return m, nil
			}
			if text != "" {
				m.sending = true
				if m.conn != nil {
					// Check if this is a server-side command (admin/plugin) that should bypass encryption
					// Client-side commands are handled above and never reach this point
					clientOnlyCommands := []string{":theme", ":time", ":clear", ":bell", ":bell-mention", ":code", ":sendfile", ":savefile", ":q"}
					isClientCommand := false
					for _, cmd := range clientOnlyCommands {
						// Check if text is exactly the command or starts with "command "
						if text == cmd || strings.HasPrefix(text, cmd+" ") {
							isClientCommand = true
							break
						}
					}

					// If it starts with : and is NOT a client command, it's a server command
					// This includes both built-in admin commands and dynamic plugin commands
					isServerCommand := *isAdmin && strings.HasPrefix(text, ":") && !isClientCommand

					if isServerCommand {
						// Send as admin command type to bypass encryption
						msg := shared.Message{
							Sender:  m.cfg.Username,
							Content: text,
							Type:    shared.AdminCommandType,
						}
						err := m.conn.WriteJSON(msg)
						if err != nil {
							m.banner = "❌ Failed to send admin command (connection lost)"
							m.sending = false
							return m, m.listenWebSocket()
						}
						m.banner = ""
					} else if m.useE2E {
						// Use E2E encryption for global chat
						log.Printf("DEBUG: Attempting to send global encrypted message: '%s'", text)

						// Validate keystore is unlocked
						if err := verifyKeystoreUnlocked(m.keystore); err != nil {
							m.banner = fmt.Sprintf("❌ Keystore not unlocked: %v", err)
							m.sending = false
							m.textarea.SetValue("")
							return m, nil
						}

						// For global chat, we don't need individual recipient keys
						// All users in the chat will receive the message encrypted with the global key
						recipients := m.users
						if len(recipients) == 0 {
							recipients = []string{m.cfg.Username} // Fallback to self
						}

						// Use the debug encryption function for global chat
						if err := debugEncryptAndSend(recipients, text, m.conn, m.keystore, m.cfg.Username); err != nil {
							m.banner = fmt.Sprintf("❌ Global encryption failed: %v", err)
							m.sending = false
							m.textarea.SetValue("")
							return m, nil
						}

						log.Printf("DEBUG: Global encrypted message sent successfully")
						m.banner = ""
					} else {
						// Send plain text message
						msg := shared.Message{Sender: m.cfg.Username, Content: text}
						if err := debugWebSocketWrite(m.conn, msg); err != nil {
							m.banner = "❌ Failed to send (connection lost)"
							m.sending = false
							return m, m.listenWebSocket()
						}
						m.banner = ""
					}
				}
				m.textarea.SetValue("")
				return m, m.listenWebSocket()
			}
			return m, nil
		default:
			// Handle database menu selections
			if m.showDBMenu && len(v.Runes) > 0 {
				switch string(v.Runes) {
				case "1":
					return m.executeDBAction("cleardb")
				case "2":
					return m.executeDBAction("backup")
				case "3":
					return m.executeDBAction("stats")
				}
			}

			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(v)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
		m.help.Width = v.Width
		chatWidth := m.width - userListWidth - 4
		if chatWidth < 20 {
			chatWidth = 20
		}
		m.viewport.Width = chatWidth
		m.viewport.Height = m.height - m.textarea.Height() - 6
		m.textarea.SetWidth(chatWidth)
		m.userListViewport.Width = userListWidth
		m.userListViewport.Height = m.height - m.textarea.Height() - 6

		// Update help viewport dimensions to be responsive
		helpWidth := m.width - 8   // Leave reasonable margins
		helpHeight := m.height - 8 // Leave reasonable margins

		// Ensure minimum usable size for very small screens
		if helpWidth < 60 {
			helpWidth = 60
		}
		if helpHeight < 15 {
			helpHeight = 15
		}

		// For very wide screens, limit width for readability but allow more height
		if helpWidth > 120 {
			helpWidth = 120
		}
		// Don't limit height - let it use the full available space

		m.helpViewport.Width = helpWidth
		m.helpViewport.Height = helpHeight

		m.viewport.SetContent(renderMessages(m.messages, m.styles, m.cfg.Username, m.users, m.viewport.Width, m.twentyFourHour))
		m.viewport.GotoBottom()
		m.userListViewport.SetContent(renderUserList(m.users, m.cfg.Username, m.styles, userListWidth, *isAdmin, m.selectedUserIndex))
		return m, nil
	case quitMsg:
		return m, tea.Quit
	case tea.MouseMsg:
		// Handle mouse events for hyperlinks
		switch v.Action {
		case tea.MouseActionPress:
			if v.Button == tea.MouseButtonLeft {
				// Check if click is within the viewport area
				if v.X >= 0 && v.X < m.viewport.Width && v.Y >= 0 && v.Y < m.viewport.Height {
					// Try to find a URL at the click position
					clickedURL := m.findURLAtClickPosition(v.X, v.Y)
					if clickedURL != "" {
						if err := openURL(clickedURL); err != nil {
							m.banner = "❌ Failed to open URL: " + err.Error()
						} else {
							m.banner = "✅ Opening URL: " + clickedURL
						}
					}
				}
			}
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(v)
		return m, cmd
	}
}

func (m *model) listenWebSocket() tea.Cmd {
	return func() tea.Msg {
		return <-m.msgChan
	}
}

// renderMessagesContent returns the raw content of messages for URL detection
func (m *model) renderMessagesContent() string {
	var content strings.Builder
	for _, msg := range m.messages {
		content.WriteString(msg.Content)
		content.WriteString(" ")
	}
	return content.String()
}

// findURLAtClickPosition attempts to find a URL at the given click position
func (m *model) findURLAtClickPosition(clickX, clickY int) string {
	// Get all URLs from the current messages
	allURLs := urlRegex.FindAllString(m.renderMessagesContent(), -1)
	if len(allURLs) == 0 {
		return ""
	}

	// Adjust clickY to account for header and other UI elements
	// This is an approximation - the exact calculation would need to account for
	// header height, banner height, etc.
	adjustedY := clickY - 3 // Approximate offset for header/banner

	// If the click is in a reasonable area of the viewport, return the first URL
	// This is a simplified approach - in a full implementation, you'd need to
	// track the exact position of each URL in the rendered text
	if adjustedY >= 0 && adjustedY < m.viewport.Height && clickX >= 0 && clickX < m.viewport.Width {
		// For now, we'll return the first URL found in the visible area
		// This works reasonably well for most use cases where there's typically
		// only one URL visible at a time
		return allURLs[0]
	}

	return ""
}

func (m *model) generateHelpContent() string {
	title := m.styles.HelpTitle.Render("marchat help")

	// Session status first
	var sessionInfo string
	if m.useE2E {
		sessionInfo = "Session: 🔒 E2E Encrypted (messages are encrypted for privacy)\n"
	} else {
		sessionInfo = "Session: 🔓 Unencrypted (messages are sent in plain text)\n"
	}

	// Basic keyboard shortcuts
	shortcuts := "\nKeyboard Shortcuts:\n"
	shortcuts += "  Ctrl+H               Toggle this help\n"
	shortcuts += "  Esc                  Quit / Close menus\n"
	shortcuts += "  Enter                Send message\n"
	shortcuts += "  ↑/↓                  Scroll chat\n"
	shortcuts += "  PgUp/PgDn            Page through chat\n"
	shortcuts += "  Ctrl+C/V/X/A         Copy/Paste/Cut/Select all\n"
	shortcuts += "  Alt+F                Send file (file picker)\n"
	shortcuts += "  Alt+C                Create code snippet\n"
	shortcuts += "  Ctrl+T               Cycle themes\n"
	shortcuts += "  Alt+T                Toggle 12/24h time\n"
	shortcuts += "  Alt+N                Toggle desktop notifications\n"
	shortcuts += "  Ctrl+L               Clear chat history\n"

	// Text commands
	commands := "\nText Commands:\n"
	commands += "  :sendfile [path]     Send a file (or Alt+F)\n"
	commands += "  :savefile <name>     Save received file\n"
	commands += "  :theme <name>        Change theme (or Ctrl+T to cycle)\n"
	commands += "  :themes              List all available themes\n"
	commands += "  :time                Toggle 12/24h time (or Alt+T)\n"
	commands += "  :clear               Clear chat history (or Ctrl+L)\n"
	commands += "  :code                Create code snippet (or Alt+C)\n"
	commands += "\nNotifications:\n"
	commands += "  :bell                Toggle message bell\n"
	commands += "  :bell-mention        Bell on mentions only\n"
	commands += "  :notify-mode <mode>  Set notification mode (none/bell/desktop/both)\n"
	commands += "  :notify-desktop      Toggle desktop notifications\n"
	commands += "  :notify-status       Show notification settings\n"
	commands += "  :quiet <start> <end> Enable quiet hours (e.g., :quiet 22 8)\n"
	commands += "  :quiet-off           Disable quiet hours\n"
	commands += "  :focus [duration]    Enable focus mode (e.g., :focus 30m)\n"
	commands += "  :focus-off           Disable focus mode\n"

	// Admin section
	var adminSection string
	if *isAdmin {
		adminSection = "\nAdmin Features:\n"
		adminSection += "\n  User Management:\n"
		adminSection += "    Ctrl+U             Select/cycle user\n"
		adminSection += "    Ctrl+K             Kick selected user (or :kick <user>)\n"
		adminSection += "    Ctrl+B             Ban selected user (or :ban <user>)\n"
		adminSection += "    Ctrl+F             Force disconnect (or :forcedisconnect <user>)\n"
		adminSection += "    Ctrl+Shift+B       Unban user (or :unban <user>)\n"
		adminSection += "    Ctrl+Shift+A       Allow user (or :allow <user>)\n"
		adminSection += "    :cleanup           Clean stale connections\n"
		adminSection += "\n  Plugin Management:\n"
		adminSection += "    Alt+P              List plugins (or :list)\n"
		adminSection += "    Alt+S              Plugin store (or :store)\n"
		adminSection += "    Alt+R              Refresh plugins (or :refresh)\n"
		adminSection += "    Alt+I              Install plugin (or :install <name>)\n"
		adminSection += "    Alt+U              Uninstall plugin (or :uninstall <name>)\n"
		adminSection += "    Alt+E              Enable plugin (or :enable <name>)\n"
		adminSection += "    Alt+D              Disable plugin (or :disable <name>)\n"
		adminSection += "\n  Database:\n"
		adminSection += "    Ctrl+D             Database menu (or :cleardb, :backup, :stats)\n"
		adminSection += "\n  Note: Both hotkeys and text commands work in encrypted sessions.\n"
	}

	return title + "\n\n" + sessionInfo + shortcuts + commands + adminSection
}

// generateDBMenuContent creates the database operations menu content
func (m *model) generateDBMenuContent() string {
	title := m.styles.HelpTitle.Render("Database Operations")

	content := "\nAvailable Operations:\n\n"
	content += "  1. Clear Database (delete all messages)\n"
	content += "  2. Backup Database (save current state)\n"
	content += "  3. Show Database Stats\n\n"
	content += "Press 1-3 to select operation, Esc to cancel"

	return title + content
}

// executeAdminAction performs the selected admin action
func (m *model) executeAdminAction(action, targetUser string) (tea.Model, tea.Cmd) {
	if !*isAdmin || targetUser == "" {
		return m, nil
	}

	var command string
	switch action {
	case "kick":
		command = fmt.Sprintf(":kick %s", targetUser)
	case "ban":
		command = fmt.Sprintf(":ban %s", targetUser)
	case "unban":
		command = fmt.Sprintf(":unban %s", targetUser)
	case "allow":
		command = fmt.Sprintf(":allow %s", targetUser)
	case "forcedisconnect":
		command = fmt.Sprintf(":forcedisconnect %s", targetUser)
	default:
		return m, nil
	}

	// Send the admin command directly (unencrypted for server processing)
	if m.conn != nil {
		msg := shared.Message{
			Sender:  m.cfg.Username,
			Content: command,
			Type:    shared.AdminCommandType, // Special type for admin commands
		}
		err := m.conn.WriteJSON(msg)
		if err != nil {
			m.banner = "❌ Failed to send admin command"
		} else {
			m.banner = fmt.Sprintf("✅ %s action sent for %s", action, targetUser)
			// Clear selection after successful action
			if action == "kick" || action == "ban" || action == "forcedisconnect" {
				m.selectedUserIndex = -1
				m.selectedUser = ""
			}
		}
	}

	return m, m.listenWebSocket()
}

// promptForUsername prompts for a username for actions like unban/allow
func (m *model) promptForUsername(action string) (tea.Model, tea.Cmd) {
	// For now, we'll use the textarea to get the username
	// This is a simple implementation - could be improved with a dedicated prompt
	switch action {
	case "unban":
		m.banner = "Type username to unban in chat and press Enter (prefix with :unban)"
	case "allow":
		m.banner = "Type username to allow in chat and press Enter (prefix with :allow)"
	}
	return m, nil
}

// promptForPluginName prompts for a plugin name for plugin management actions
func (m *model) promptForPluginName(action string) (tea.Model, tea.Cmd) {
	// Set pending action and update banner
	m.pendingPluginAction = action
	switch action {
	case "install":
		m.banner = "Enter plugin name to install (press Enter to confirm, Esc to cancel)"
	case "uninstall":
		m.banner = "Enter plugin name to uninstall (press Enter to confirm, Esc to cancel)"
	case "enable":
		m.banner = "Enter plugin name to enable (press Enter to confirm, Esc to cancel)"
	case "disable":
		m.banner = "Enter plugin name to disable (press Enter to confirm, Esc to cancel)"
	}
	// Focus the textarea for input
	m.textarea.Focus()
	return m, nil
}

// executePluginCommand executes a plugin management command
func (m *model) executePluginCommand(command string) (tea.Model, tea.Cmd) {
	if !*isAdmin {
		return m, nil
	}

	// Send the plugin command as an admin command (unencrypted)
	if m.conn != nil {
		msg := shared.Message{
			Sender:  m.cfg.Username,
			Content: command,
			Type:    shared.AdminCommandType, // Use admin command type to bypass encryption
		}
		err := m.conn.WriteJSON(msg)
		if err != nil {
			m.banner = "❌ Failed to send plugin command (connection lost)"
		} else {
			m.banner = fmt.Sprintf("✅ Sent: %s", command)
		}
	}

	return m, m.listenWebSocket()
}

// executeDBAction performs the selected database action
func (m *model) executeDBAction(action string) (tea.Model, tea.Cmd) {
	if !*isAdmin {
		m.showDBMenu = false
		return m, nil
	}

	switch action {
	case "cleardb":
		if m.conn != nil {
			msg := shared.Message{
				Sender:  m.cfg.Username,
				Content: ":cleardb",
				Type:    shared.AdminCommandType,
			}
			err := m.conn.WriteJSON(msg)
			if err != nil {
				m.banner = "❌ Failed to send cleardb command"
			} else {
				m.banner = "✅ Database clear command sent"
			}
		}
	case "backup":
		if m.conn != nil {
			msg := shared.Message{
				Sender:  m.cfg.Username,
				Content: ":backup",
				Type:    shared.AdminCommandType,
			}
			err := m.conn.WriteJSON(msg)
			if err != nil {
				m.banner = "❌ Failed to send backup command"
			} else {
				m.banner = "✅ Database backup command sent"
			}
		}
	case "stats":
		if m.conn != nil {
			msg := shared.Message{
				Sender:  m.cfg.Username,
				Content: ":stats",
				Type:    shared.AdminCommandType,
			}
			err := m.conn.WriteJSON(msg)
			if err != nil {
				m.banner = "❌ Failed to send stats command"
			} else {
				m.banner = "✅ Database stats command sent"
			}
		}
	}

	m.showDBMenu = false
	return m, m.listenWebSocket()
}

func (m *model) View() string {
	// Header with version
	headerText := fmt.Sprintf(" marchat %s ", shared.ClientVersion)
	header := m.styles.Header.Width(m.viewport.Width + userListWidth + 4).Render(headerText)

	// Footer with encryption status
	footerText := "Press Ctrl+H for help"
	if m.showHelp {
		footerText = "Press Ctrl+H to close help"
	}
	// Add encryption status indicator
	if m.useE2E {
		footerText += " | 🔒 E2E Encrypted"
	} else {
		footerText += " | 🔓 Unencrypted"
	}
	footer := m.styles.Footer.Width(m.viewport.Width + userListWidth + 4).Render(footerText)

	// Banner
	var bannerBox string
	if m.banner != "" || m.sending {
		bannerText := m.banner
		if m.sending {
			if bannerText != "" {
				bannerText += " ⏳ Sending..."
			} else {
				bannerText = "⏳ Sending..."
			}
		}
		bannerBox = m.styles.Banner.
			Width(m.viewport.Width).
			PaddingLeft(1).
			Background(lipgloss.Color("#FF5F5F")).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Render(bannerText)
	}

	// Chat and user list layout
	chatBoxStyle := m.styles.Box
	chatPanel := chatBoxStyle.Width(m.viewport.Width).Render(m.viewport.View())
	userPanel := m.userListViewport.View()
	row := lipgloss.JoinHorizontal(lipgloss.Top, userPanel, chatPanel)

	// Input
	inputPanel := m.styles.Input.Width(m.viewport.Width).Render(m.textarea.View())

	// Compose layout
	ui := lipgloss.JoinVertical(lipgloss.Left,
		header,
		bannerBox,
		row,
		inputPanel,
		footer,
	)

	// Show code snippet interface as full-screen if shown
	if m.showCodeSnippet {
		// Use most of the available screen space for code snippet
		codeWidth := m.width - 8   // Leave reasonable margins
		codeHeight := m.height - 8 // Leave reasonable margins

		// Ensure minimum usable size for very small screens
		if codeWidth < 60 {
			codeWidth = 60
		}
		if codeHeight < 15 {
			codeHeight = 15
		}

		// Update code snippet model dimensions
		m.codeSnippetModel.width = codeWidth
		m.codeSnippetModel.height = codeHeight

		// Create code snippet content
		codeContent := m.styles.HelpOverlay.
			Width(codeWidth).
			Height(codeHeight).
			Render(m.codeSnippetModel.View())

		// Center the code snippet modal on the screen
		ui = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, codeContent)
		return m.styles.Background.Render(ui)
	}

	// Show file picker interface as full-screen if shown
	if m.showFilePicker {
		// Use most of the available screen space for file picker
		fileWidth := m.width - 8   // Leave reasonable margins
		fileHeight := m.height - 8 // Leave reasonable margins

		// Ensure minimum usable size for very small screens
		if fileWidth < 60 {
			fileWidth = 60
		}
		if fileHeight < 15 {
			fileHeight = 15
		}

		// Update file picker model dimensions
		m.filePickerModel.width = fileWidth
		m.filePickerModel.height = fileHeight

		// Create file picker content
		fileContent := m.styles.HelpOverlay.
			Width(fileWidth).
			Height(fileHeight).
			Render(m.filePickerModel.View())

		// Center the file picker modal on the screen
		ui = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fileContent)
		return m.styles.Background.Render(ui)
	}

	// Show help as full-screen modal if shown
	if m.showHelp {
		// Use most of the available screen space for help
		helpWidth := m.width - 8   // Leave reasonable margins
		helpHeight := m.height - 8 // Leave reasonable margins

		// Ensure minimum usable size for very small screens
		if helpWidth < 60 {
			helpWidth = 60
		}
		if helpHeight < 15 {
			helpHeight = 15
		}

		// For very wide screens, limit width for readability but allow more height
		if helpWidth > 120 {
			helpWidth = 120
		}
		// Don't limit height - let it use the full available space

		// Create help footer with navigation instructions
		helpFooter := "Use ↑/↓ or PgUp/PgDn to scroll • Press Ctrl+H to close help"
		footerStyle := lipgloss.NewStyle().
			Width(helpWidth).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#888888")).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			PaddingTop(1)

		// Adjust content height to leave room for footer
		contentHeight := helpHeight - 3 // Reserve 3 lines for footer (border + padding + text)
		if contentHeight < 10 {
			contentHeight = 10
		}

		// Create help content viewport
		helpContent := m.styles.HelpOverlay.
			Width(helpWidth).
			Height(contentHeight).
			BorderBottom(false). // Remove bottom border since footer will have top border
			Render(m.helpViewport.View())

		// Combine content and footer
		helpModal := lipgloss.JoinVertical(lipgloss.Left,
			helpContent,
			footerStyle.Render(helpFooter),
		)

		// Center the help modal on the screen
		ui = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpModal)
	}

	// Show admin menus if open
	if m.showDBMenu {
		menuWidth := 60
		menuHeight := 15

		// Ensure minimum size
		if m.width < menuWidth+4 {
			menuWidth = m.width - 4
		}
		if m.height < menuHeight+4 {
			menuHeight = m.height - 4
		}

		dbMenu := m.styles.HelpOverlay.
			Width(menuWidth).
			Height(menuHeight).
			Render(m.dbMenuViewport.View())

		ui = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dbMenu)
	}

	return m.styles.Background.Render(ui)
}

func renderEmojis(s string) string {
	emojis := map[string]string{
		":)": "😊",
		":(": "🙁",
		":D": "😃",
		"<3": "❤️",
		":P": "😛",
	}
	for k, v := range emojis {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

// renderHyperlinks detects and formats URLs in text
func renderHyperlinks(content string, styles themeStyles) string {
	return urlRegex.ReplaceAllStringFunc(content, func(url string) string {
		return styles.Hyperlink.Render(url)
	})
}

// openURL opens a URL in the default browser
func openURL(url string) error {
	// Ensure URL has a protocol
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Try multiple methods for Windows
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		if err := cmd.Start(); err != nil {
			// Fallback to start command
			cmd = exec.Command("cmd", "/c", "start", url)
			return cmd.Start()
		}
		return nil
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		// Try xdg-open first, then fallback to other methods
		cmd = exec.Command("xdg-open", url)
		if err := cmd.Start(); err != nil {
			// Try other common Linux methods
			cmd = exec.Command("sensible-browser", url)
			if err := cmd.Start(); err != nil {
				cmd = exec.Command("firefox", url)
				return cmd.Start()
			}
			return nil
		}
		return nil
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// renderCodeBlocks detects and renders syntax highlighted code blocks in messages
func renderCodeBlocks(content string) string {
	// Look for markdown code blocks
	codeBlockRegex := regexp.MustCompile("```([a-zA-Z0-9+]*)\n([\\s\\S]*?)```")

	return codeBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract language and code
		parts := codeBlockRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match // Return original if parsing fails
		}

		language := parts[1]
		code := parts[2]

		// Use Chroma directly for syntax highlighting
		var sb strings.Builder
		err := quick.Highlight(&sb, code, language, "terminal256", "monokai")
		if err != nil {
			return match // Return original if highlighting fails
		}

		return sb.String()
	})
}

func renderUserList(users []string, me string, styles themeStyles, width int, isAdmin bool, selectedUserIndex int) string {
	var b strings.Builder
	title := " Users "
	b.WriteString(styles.UserList.Width(width).Render(title) + "\n")
	max := maxUsersDisplay
	for i, u := range users {
		if i >= max {
			b.WriteString(lipgloss.NewStyle().Italic(true).Faint(true).Width(width).Render(fmt.Sprintf("+%d more", len(users)-max)) + "\n")
			break
		}

		var userStyle lipgloss.Style
		var prefix string

		if u == me {
			userStyle = styles.Me
			prefix = "• "
		} else {
			userStyle = styles.Other
			prefix = "• "

			// Highlight selected user
			if isAdmin && selectedUserIndex == i {
				userStyle = userStyle.Background(lipgloss.Color("#444444")).Bold(true)
				prefix = "► " // Arrow to indicate selection
			}
		}

		b.WriteString(userStyle.Render(prefix+u) + "\n")
	}
	return b.String()
}

// Add a custom quitMsg type
type quitMsg struct{}

func main() {
	flag.Parse()

	// Auto-connect to most recent profile
	if *autoConnect {
		loader, err := config.NewInteractiveConfigLoader()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		cfg, err := loader.AutoConnect()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Get sensitive data and connect
		adminKey, keystorePass, err := loader.PromptSensitiveData(cfg.IsAdmin, cfg.UseE2E)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		initializeClient(cfg, adminKey, keystorePass)
		return
	}

	// Quick start menu - actually connects using saved profiles
	if *quickStart {
		loader, err := config.NewInteractiveConfigLoader()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		cfg, err := loader.QuickStartConnect()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Get sensitive data and connect
		adminKey, keystorePass, err := loader.PromptSensitiveData(cfg.IsAdmin, cfg.UseE2E)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		initializeClient(cfg, adminKey, keystorePass)
		return
	}

	var cfg *config.Config
	var err error

	// Check if all required flags are provided for non-interactive mode
	if *nonInteractive || (allFlagsProvided(*serverURL, *username, *isAdmin, *adminKey, *useE2E, *keystorePassphrase)) {
		// Use traditional flag-based configuration
		cfg, err = loadConfigFromFlags(*configPath, *serverURL, *username, *theme, *isAdmin, *useE2E, *skipTLSVerify)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Validate required flags for non-interactive mode
		if err := validateFlags(*isAdmin, *adminKey, *useE2E, *keystorePassphrase); err != nil {
			fmt.Printf("Error: %v\n", err)
			flag.Usage()
			os.Exit(1)
		}

		// Continue with existing client initialization using flag values
		initializeClient(cfg, *adminKey, *keystorePassphrase)

	} else {
		// Check if this is a first-time user (no profiles exist)
		loader, err := config.NewInteractiveConfigLoader()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		profiles, err := loader.LoadProfiles()
		isFirstTime := err != nil || len(profiles.Profiles) == 0

		var cfg *config.Config
		var adminKeyFromConfig, keystorePassFromConfig string

		if isFirstTime {
			// First time user - show welcome and go straight to config creation
			fmt.Println("🎉 Welcome to marchat! Let's get you set up...")

			configResult, keystorePass, err := config.RunInteractiveConfig()
			if err != nil {
				fmt.Printf("Configuration error: %v\n", err)
				os.Exit(1)
			}
			cfg = configResult
			adminKeyFromConfig = cfg.AdminKey
			keystorePassFromConfig = keystorePass

			// Save as the default profile
			profile := &config.ConnectionProfile{
				Name:      "Default",
				ServerURL: cfg.ServerURL,
				Username:  cfg.Username,
				IsAdmin:   cfg.IsAdmin,
				UseE2E:    cfg.UseE2E,
				Theme:     cfg.Theme,
				LastUsed:  time.Now().Unix(),
			}
			profiles.Profiles = append(profiles.Profiles, *profile)
			if err := loader.SaveProfiles(profiles); err != nil {
				fmt.Printf("Warning: Could not save profile: %v\n", err)
			}
			fmt.Println("✅ Configuration saved! Next time you can use --auto or --quick-start for faster connections.")

		} else {
			// Existing user - show profile selection with option to create new
			fmt.Println("📝 Select a connection profile or create a new one...")

			// Sort profiles by last used (most recent first)
			sort.Slice(profiles.Profiles, func(i, j int) bool {
				return profiles.Profiles[i].LastUsed > profiles.Profiles[j].LastUsed
			})

			selectedProfile, isCreateNew, err := config.RunProfileSelectionWithNew(profiles.Profiles, loader)
			if err != nil {
				fmt.Printf("Profile selection error: %v\n", err)
				os.Exit(1)
			}

			if isCreateNew {
				// User chose to create a new profile
				fmt.Println("Creating a new connection profile...")

				configResult, keystorePass, err := config.RunInteractiveConfig()
				if err != nil {
					fmt.Printf("Configuration error: %v\n", err)
					os.Exit(1)
				}
				cfg = configResult
				adminKeyFromConfig = cfg.AdminKey
				keystorePassFromConfig = keystorePass

				// Save as a new profile
				profileName := fmt.Sprintf("Profile-%d", len(profiles.Profiles)+1)
				profile := &config.ConnectionProfile{
					Name:      profileName,
					ServerURL: cfg.ServerURL,
					Username:  cfg.Username,
					IsAdmin:   cfg.IsAdmin,
					UseE2E:    cfg.UseE2E,
					Theme:     cfg.Theme,
					LastUsed:  time.Now().Unix(),
				}
				profiles.Profiles = append(profiles.Profiles, *profile)
				if err := loader.SaveProfiles(profiles); err != nil {
					fmt.Printf("Warning: Could not save profile: %v\n", err)
				}
				fmt.Printf("✅ Configuration saved as '%s'! You can use --auto or --quick-start for faster connections.\n", profileName)

			} else {
				// We now have the actual profile object, not just an index!
				// Reload profiles in case they were modified during selection
				profiles, err = loader.LoadProfiles()
				if err != nil {
					fmt.Printf("Error reloading profiles: %v\n", err)
					os.Exit(1)
				}

				// Find the selected profile in the reloaded list
				var profileIndex = -1
				for i, p := range profiles.Profiles {
					if p.Name == selectedProfile.Name &&
						p.ServerURL == selectedProfile.ServerURL &&
						p.Username == selectedProfile.Username {
						profileIndex = i
						break
					}
				}

				if profileIndex == -1 {
					fmt.Println("Error: Selected profile no longer exists")
					os.Exit(1)
				}

				// User selected an existing profile
				profile := &profiles.Profiles[profileIndex]
				fmt.Printf("Selected: %s\n", profile.Name)

				// Update last used timestamp
				profile.LastUsed = time.Now().Unix()
				if err := loader.SaveProfiles(profiles); err != nil {
					// Log error but don't fail the connection
					fmt.Printf("Warning: Could not update profile usage timestamp: %v\n", err)
				}

				// Convert profile to config
				cfg = &config.Config{
					Username:       profile.Username,
					ServerURL:      profile.ServerURL,
					IsAdmin:        profile.IsAdmin,
					UseE2E:         profile.UseE2E,
					Theme:          profile.Theme,
					TwentyFourHour: true, // Default value
				}

				// Get sensitive data
				adminKeyFromConfig, keystorePassFromConfig, err = loader.PromptSensitiveData(cfg.IsAdmin, cfg.UseE2E)
				if err != nil {
					fmt.Printf("Error getting sensitive data: %v\n", err)
					os.Exit(1)
				}
			}
		}

		// Continue with existing client initialization...
		initializeClient(cfg, adminKeyFromConfig, keystorePassFromConfig)
	}
}

func allFlagsProvided(serverURL, username string, isAdmin bool, adminKey string, useE2E bool, keystorePassphrase string) bool {
	if serverURL == "" || username == "" {
		return false
	}

	if isAdmin && adminKey == "" {
		return false
	}

	if useE2E && keystorePassphrase == "" {
		return false
	}

	return true
}

func loadConfigFromFlags(configPath, serverURL, username, theme string, isAdmin, useE2E, skipTLSVerify bool) (*config.Config, error) {
	var cfg config.Config

	// Try to load existing config file if specified
	if configPath != "" {
		if existingCfg, err := config.LoadConfig(configPath); err == nil {
			cfg = existingCfg
		}
	} else {
		// Use platform-appropriate config path
		defaultConfigPath, err := config.GetConfigPath()
		if err == nil {
			if existingCfg, err := config.LoadConfig(defaultConfigPath); err == nil {
				cfg = existingCfg
			}
		}
	}

	// Override with flags
	if serverURL != "" {
		cfg.ServerURL = serverURL
	}
	if username != "" {
		cfg.Username = username
	}
	if theme != "" {
		cfg.Theme = theme
	}

	cfg.IsAdmin = isAdmin
	cfg.UseE2E = useE2E
	cfg.SkipTLSVerify = skipTLSVerify

	// Set defaults
	if cfg.ServerURL == "" {
		cfg.ServerURL = "ws://localhost:8080/ws"
	}
	if cfg.Theme == "" {
		cfg.Theme = "system"
	}

	return &cfg, nil
}

func validateFlags(isAdmin bool, adminKey string, useE2E bool, keystorePassphrase string) error {
	if isAdmin && adminKey == "" {
		return fmt.Errorf("--admin flag requires --admin-key")
	}

	if useE2E && keystorePassphrase == "" {
		return fmt.Errorf("--e2e flag requires --keystore-passphrase")
	}

	return nil
}

func initializeClient(cfg *config.Config, adminKeyParam, keystorePassphraseParam string) {
	// Your existing client initialization code here...
	fmt.Printf("Connecting to %s as %s...\n", cfg.ServerURL, cfg.Username)

	// Termux clipboard availability notice
	if isTermux() {
		fmt.Println("⚠️  Termux environment detected")
		if !checkClipboardSupport() {
			fmt.Println("⚠️  Clipboard operations may be unavailable - text will be shown in banner")
		}
	}

	// Use platform-appropriate config path for saving
	var configFilePath string
	defaultConfigPath, err := config.GetConfigPath()
	if err == nil {
		configFilePath = defaultConfigPath
	} else {
		configFilePath = "config.json" // fallback
	}

	// Initialize keystore if E2E is enabled
	var keystore *crypto.KeyStore
	if cfg.UseE2E {
		keystorePath, err := config.GetKeystorePath()
		if err != nil {
			fmt.Printf("Error getting keystore path: %v\n", err)
			os.Exit(1)
		}
		keystore = crypto.NewKeyStore(keystorePath)

		if err := keystore.Initialize(keystorePassphraseParam); err != nil {
			fmt.Printf("Error initializing keystore: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("E2E encryption enabled\n")
	}

	// Setup textarea
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 2000
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(80, 20)

	userListVp := viewport.New(18, 10) // height will be set on resize
	userListVp.SetContent(renderUserList([]string{cfg.Username}, cfg.Username, getThemeStyles(cfg.Theme), 18, cfg.IsAdmin, -1))

	helpVp := viewport.New(70, 20) // initial size, will be adjusted on resize

	// Initialize admin menu viewports
	dbMenuVp := viewport.New(60, 15)

	// Additional keystore initialization if E2E is enabled
	if cfg.UseE2E && keystore != nil {
		// Check environment variable status
		if envKey := os.Getenv("MARCHAT_GLOBAL_E2E_KEY"); envKey != "" {
			fmt.Printf("Using global E2E key from environment variable\n")
		} else {
			fmt.Printf("No MARCHAT_GLOBAL_E2E_KEY environment variable found\n")
		}

		// Verify keystore is properly unlocked
		if err := verifyKeystoreUnlocked(keystore); err != nil {
			fmt.Printf("Keystore unlock verification failed: %v\n", err)
			os.Exit(1)
		}

		// Display global key info
		if globalKey := keystore.GetGlobalKey(); globalKey != nil {
			fmt.Printf("Global chat encryption: ENABLED (Key ID: %s)\n", globalKey.KeyID)
		} else {
			fmt.Printf("Global key not available\n")
			os.Exit(1)
		}

		// Test encryption roundtrip (non-blocking for production use)
		if err := validateEncryptionRoundtrip(keystore, cfg.Username); err != nil {
			fmt.Printf("Encryption validation failed: %v\n", err)
			fmt.Printf("E2E encryption will continue but may have issues\n")
			log.Printf("WARNING: Encryption validation failed: %v", err)
		} else {
			fmt.Printf("Encryption validation passed\n")
		}

		keystorePath, _ := config.GetKeystorePath()
		fmt.Printf("E2E encryption enabled with keystore: %s\n", keystorePath)
	}

	// Update global flags for compatibility with existing code
	*isAdmin = cfg.IsAdmin
	*useE2E = cfg.UseE2E
	*skipTLSVerify = cfg.SkipTLSVerify
	if len(adminKeyParam) > 0 {
		*adminKey = adminKeyParam
	}
	if len(keystorePassphraseParam) > 0 {
		*keystorePassphrase = keystorePassphraseParam
	}

	m := &model{
		cfg:               *cfg,
		configFilePath:    configFilePath,
		profileName:       getCurrentProfileName(cfg),
		textarea:          ta,
		viewport:          vp,
		styles:            getThemeStyles(cfg.Theme),
		users:             []string{cfg.Username},
		userListViewport:  userListVp,
		helpViewport:      helpVp,
		dbMenuViewport:    dbMenuVp,
		twentyFourHour:    cfg.TwentyFourHour,
		keystore:          keystore,
		useE2E:            cfg.UseE2E,
		keys:              newKeyMap(),
		selectedUserIndex: -1, // No user selected initially
	}

	// Initialize notification manager with config settings
	notifConfig := configToNotificationConfig(*cfg)
	m.notificationManager = NewNotificationManager(notifConfig)

	p := tea.NewProgram(m, tea.WithAltScreen())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		m.closeWebSocket()
		p.Send(quitMsg{})
	}()

	if _, err := p.Run(); err != nil {
		log.Printf("Error running program: %v", err)
		os.Exit(1)
	}
	m.wg.Wait() // Wait for all goroutines to finish
}
