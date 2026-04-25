package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Cod-e-Codes/marchat/client/config"
	"github.com/Cod-e-Codes/marchat/client/crypto"
	"github.com/Cod-e-Codes/marchat/client/exthook"
	"github.com/Cod-e-Codes/marchat/internal/doctor"
	"github.com/Cod-e-Codes/marchat/shared"

	"os/signal"
	"syscall"

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
	"golang.org/x/term"
)

const maxMessages = 100
const maxUsersDisplay = 20
const userListWidth = 18
const pingPeriod = 50 * time.Second        // moved from magic number
const reconnectMaxDelay = 30 * time.Second // for exponential backoff

var mentionRegex *regexp.Regexp
var urlRegex *regexp.Regexp

func init() {
	mentionRegex = regexp.MustCompile(`\B@([a-zA-Z0-9_]+)\b`)
	// URL regex pattern to match http/https URLs and common domain patterns
	// This pattern matches URLs more comprehensively
	urlRegex = regexp.MustCompile(`(https?://[^\s<>"{}|\\^` + "`" + `\[\]]+|www\.[^\s<>"{}|\\^` + "`" + `\[\]]+\.[a-zA-Z]{2,})`)

	// Set up client debug logging to config directory
	if err := config.EnsureClientConfigDir(); err != nil {
		log.Printf("Warning: could not create client config directory: %v", err)
	}
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

// getClientConfigDir returns the client config directory (same rules as
// config.ResolveClientConfigDir).
func getClientConfigDir() string {
	return config.ResolveClientConfigDir()
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
	nonInteractive     = flag.Bool("non-interactive", false, "Skip interactive prompts (with --e2e, require --keystore-passphrase on the command line)")
	runDoctor          = flag.Bool("doctor", false, "Print environment and configuration diagnostics, then exit")
	runDoctorJSON      = flag.Bool("doctor-json", false, "Same as -doctor with JSON output (if both are set, JSON is used)")
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

	twentyFourHour      bool // NEW: timestamp format toggle
	showMessageMetadata bool

	sending bool // NEW: sending message feedback

	conn    *websocket.Conn // persistent WebSocket connection
	msgChan chan tea.Msg    // channel for incoming messages from WS goroutine
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	reconnectDelay time.Duration               // for exponential backoff
	receivedFiles  map[string]*shared.FileMeta // "sender/filename" -> filemeta for saving

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

	typingUsers    map[string]time.Time
	typingScopeDM  map[string]bool // latest typing from this sender was DM-scoped (non-empty recipient)
	typingChannel  map[string]string
	typingTimeout  time.Duration
	dmRecipient    string
	activeDMThread string
	dmUnread       map[string]int
	dmLastSeenID   map[string]int64
	dmHidden       map[string]bool
	dmStatePath    string
	reactions      map[int64]map[string]map[string]bool
	lastTypingSent time.Time
	unreadCount    int

	// Hub channel (lowercase); empty means unknown or general for display.
	currentChannel string
	// Read receipts: debounced tail cursor sent to the server.
	lastReadReceiptSentID     int64
	readReceiptFlushScheduled bool
}

type dmUIState struct {
	LastSeen map[string]int64 `json:"last_seen"`
	Hidden   map[string]bool  `json:"hidden"`
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
	if msg.Sender == m.cfg.Username {
		return false, NotificationLevelInfo
	}

	switch msg.Type {
	case shared.TypingMessage, shared.ReadReceiptType, shared.ReactionMessage,
		shared.EditMessageType, shared.DeleteMessage, shared.PinMessage:
		return false, NotificationLevelInfo
	}

	if msg.Type == shared.DirectMessage {
		return true, NotificationLevelDM
	}

	mentionPattern := fmt.Sprintf("@%s", m.cfg.Username)
	if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(mentionPattern)) {
		return true, NotificationLevelMention
	}

	return true, NotificationLevelInfo
}

// messageIncrementsUnread is true when an inbound message should bump the footer
// unread count while the transcript viewport is not at the bottom. Ephemeral or
// in-place update types (typing, reactions, edits, etc.) must not increment.
func messageIncrementsUnread(m *model, v shared.Message) bool {
	if v.Sender == m.cfg.Username {
		return false
	}
	switch v.Type {
	case shared.TypingMessage, shared.ReadReceiptType, shared.ReactionMessage,
		shared.EditMessageType, shared.DeleteMessage, shared.PinMessage,
		shared.SearchMessage, shared.AdminCommandType,
		shared.JoinChannelType, shared.LeaveChannelType, shared.ListChannelsType:
		return false
	case shared.TextMessage, shared.DirectMessage, shared.FileMessageType:
		return true
	case "":
		return true
	default:
		return false
	}
}

// dmTypingVisibleForTranscript returns whether inbound typing should update the typing footer.
// Empty recipient means channel or global typing: show only when not focused on a DM thread,
// so channel typing does not appear to be part of the private DM. Non-empty recipient means
// DM-scoped typing (sender composing toward that recipient); show only while the active DM
// thread is with the sender.
func dmTypingVisibleForTranscript(msg shared.Message, activeDMThread string) bool {
	if msg.Type != shared.TypingMessage {
		return false
	}
	active := strings.TrimSpace(activeDMThread)
	if strings.TrimSpace(msg.Recipient) == "" {
		return active == ""
	}
	return strings.EqualFold(active, strings.TrimSpace(msg.Sender))
}

func dmPartnerForMessage(msg shared.Message, me string) string {
	if msg.Type != shared.DirectMessage {
		return ""
	}
	if strings.EqualFold(msg.Sender, me) {
		return strings.TrimSpace(msg.Recipient)
	}
	if strings.EqualFold(msg.Recipient, me) {
		return strings.TrimSpace(msg.Sender)
	}
	return ""
}

func normalizeChannel(channel string) string {
	normalized := strings.ToLower(strings.TrimSpace(channel))
	if normalized == "" {
		return "general"
	}
	return normalized
}

func (m *model) visibleMessages() []shared.Message {
	if strings.TrimSpace(m.activeDMThread) == "" {
		filtered := make([]shared.Message, 0, len(m.messages))
		currentChannel := normalizeChannel(m.currentChannel)
		for _, msg := range m.messages {
			if msg.Type != shared.DirectMessage && normalizeChannel(msg.Channel) == currentChannel {
				filtered = append(filtered, msg)
			}
		}
		return filtered
	}

	filtered := make([]shared.Message, 0, len(m.messages))
	for _, msg := range m.messages {
		partner := dmPartnerForMessage(msg, m.cfg.Username)
		if partner != "" && strings.EqualFold(partner, m.activeDMThread) {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

func (m *model) dmContacts() []string {
	set := make(map[string]struct{})
	for _, msg := range m.messages {
		if partner := dmPartnerForMessage(msg, m.cfg.Username); partner != "" {
			set[partner] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for user := range set {
		out = append(out, user)
	}
	sort.Strings(out)
	return out
}

func (m *model) dmThreadMaxMessageID(user string) int64 {
	var maxID int64
	for _, msg := range m.messages {
		partner := dmPartnerForMessage(msg, m.cfg.Username)
		if partner == "" || !strings.EqualFold(partner, user) {
			continue
		}
		if msg.MessageID > maxID {
			maxID = msg.MessageID
		}
	}
	return maxID
}

func normalizeDMUser(user string) string {
	return strings.ToLower(strings.TrimSpace(user))
}

func (m *model) markDMThreadRead(user string) {
	if strings.TrimSpace(user) == "" {
		return
	}
	if m.dmUnread == nil {
		m.dmUnread = make(map[string]int)
	}
	if m.dmLastSeenID == nil {
		m.dmLastSeenID = make(map[string]int64)
	}
	key := normalizeDMUser(user)
	maxID := m.dmThreadMaxMessageID(user)
	if maxID > 0 {
		m.dmLastSeenID[key] = maxID
	}
	m.dmUnread[key] = 0
	m.saveDMUIState()
}

func (m *model) rebuildDMUnreadCounts() {
	if m.dmUnread == nil {
		m.dmUnread = make(map[string]int)
	}
	if m.dmLastSeenID == nil {
		m.dmLastSeenID = make(map[string]int64)
	}
	for k := range m.dmUnread {
		delete(m.dmUnread, k)
	}
	for _, msg := range m.messages {
		partner := dmPartnerForMessage(msg, m.cfg.Username)
		if partner == "" {
			continue
		}
		key := normalizeDMUser(partner)
		if m.activeDMThread != "" && strings.EqualFold(partner, m.activeDMThread) {
			continue
		}
		if strings.EqualFold(msg.Sender, m.cfg.Username) {
			continue
		}
		if msg.MessageID > 0 && msg.MessageID <= m.dmLastSeenID[key] {
			continue
		}
		m.dmUnread[key]++
	}
}

func (m *model) sidebarDMThreads() []dmSidebarEntry {
	latest := make(map[string]string)
	for _, msg := range m.messages {
		partner := dmPartnerForMessage(msg, m.cfg.Username)
		if partner == "" {
			continue
		}
		latest[normalizeDMUser(partner)] = partner
	}

	entries := make([]dmSidebarEntry, 0, len(latest))
	for key, user := range latest {
		if m.dmHidden[key] {
			continue
		}
		entries = append(entries, dmSidebarEntry{
			User:   user,
			Unread: m.dmUnread[key],
			Active: strings.EqualFold(user, m.activeDMThread),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Unread != entries[j].Unread {
			return entries[i].Unread > entries[j].Unread
		}
		return strings.ToLower(entries[i].User) < strings.ToLower(entries[j].User)
	})
	return entries
}

func (m *model) updateSidebar() {
	sidebarWidth := m.userListViewport.Width
	if sidebarWidth <= 0 {
		sidebarWidth = userListWidth
	}
	m.userListViewport.SetContent(renderUserList(m.users, m.cfg.Username, m.styles, sidebarWidth, *isAdmin, m.selectedUserIndex, m.sidebarDMThreads()))
}

func (m *model) loadDMUIState() {
	if strings.TrimSpace(m.dmStatePath) == "" {
		return
	}
	data, err := os.ReadFile(m.dmStatePath)
	if err != nil {
		return
	}
	var state dmUIState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	if state.LastSeen != nil {
		m.dmLastSeenID = state.LastSeen
	}
	if state.Hidden != nil {
		m.dmHidden = state.Hidden
	}
}

func (m *model) saveDMUIState() {
	if strings.TrimSpace(m.dmStatePath) == "" {
		return
	}
	state := dmUIState{
		LastSeen: m.dmLastSeenID,
		Hidden:   m.dmHidden,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(m.dmStatePath, data, 0600)
}

func (m *model) refreshTranscript() {
	m.viewport.SetContent(renderMessages(m.visibleMessages(), m.styles, m.cfg.Username, m.users, m.viewport.Width, m.twentyFourHour, m.showMessageMetadata, m.reactions))
	m.updateSidebar()
}

type themeStyles struct {
	User        lipgloss.Style
	Time        lipgloss.Style
	Info        lipgloss.Style // empty state and date headers (not System transcript body)
	Timestamp   lipgloss.Style // bracketed message times in transcript
	Msg         lipgloss.Style
	Banner      lipgloss.Style // accent foreground (inline highlights); not the full-width strip
	BannerError lipgloss.Style
	BannerWarn  lipgloss.Style
	BannerInfo  lipgloss.Style
	// Transcript: lines from sender "System" (distinct from top banner strip).
	SystemMsg      lipgloss.Style
	SystemMsgError lipgloss.Style
	SystemMsgWarn  lipgloss.Style
	Box            lipgloss.Style // frame color
	Mention        lipgloss.Style // mention highlighting
	Hyperlink      lipgloss.Style // hyperlink highlighting

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
	timeStyle := lipgloss.NewStyle().Faint(true)
	return themeStyles{
		User:           lipgloss.NewStyle().Bold(true),
		Time:           timeStyle,
		Info:           timeStyle,
		Timestamp:      timeStyle,
		Msg:            lipgloss.NewStyle(),
		Banner:         lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F")).Bold(true),
		BannerError:    lipgloss.NewStyle().Background(lipgloss.Color("#C42B2B")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true),
		BannerWarn:     lipgloss.NewStyle().Background(lipgloss.Color("#B8860B")).Foreground(lipgloss.Color("#000000")).Bold(true),
		BannerInfo:     lipgloss.NewStyle().Background(lipgloss.Color("#2D4A68")).Foreground(lipgloss.Color("#E8E8E8")).Bold(true),
		SystemMsg:      timeStyle,
		SystemMsgError: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555")),
		SystemMsgWarn:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D7AF00")),
		Box:            lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#AAAAAA")),
		Mention:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFD700")),
		Hyperlink:      lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("#4A9EFF")),
		UserList:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#AAAAAA")).Padding(0, 1),
		Me:             lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true),
		Other:          lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")),
		Background:     lipgloss.NewStyle(),
		Header:         lipgloss.NewStyle(),
		Footer:         lipgloss.NewStyle(),
		Input:          lipgloss.NewStyle(),
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

// applyBuiltinBannerStrips sets full-width banner strip styles per built-in theme.
func applyBuiltinBannerStrips(s *themeStyles, theme string) {
	switch theme {
	case "system":
		s.BannerError = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555"))
		s.BannerWarn = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D7AF00"))
		s.BannerInfo = lipgloss.NewStyle().Bold(true)
	case "patriot":
		// BannerError must differ from header red (#BF0A30) so errors read as their own band.
		s.BannerError = lipgloss.NewStyle().Background(lipgloss.Color("#5C1018")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
		s.BannerWarn = lipgloss.NewStyle().Background(lipgloss.Color("#FFD700")).Foreground(lipgloss.Color("#002868")).Bold(true)
		s.BannerInfo = lipgloss.NewStyle().Background(lipgloss.Color("#002868")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	case "retro":
		s.BannerError = lipgloss.NewStyle().Background(lipgloss.Color("#CC2200")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
		// Avoid same fill as header orange strip.
		s.BannerWarn = lipgloss.NewStyle().Background(lipgloss.Color("#CC6600")).Foreground(lipgloss.Color("#181818")).Bold(true)
		s.BannerInfo = lipgloss.NewStyle().Background(lipgloss.Color("#222200")).Foreground(lipgloss.Color("#FFFFAA")).Bold(true)
	case "modern":
		s.BannerError = lipgloss.NewStyle().Background(lipgloss.Color("#D32F2F")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
		s.BannerWarn = lipgloss.NewStyle().Background(lipgloss.Color("#F9A825")).Foreground(lipgloss.Color("#000000")).Bold(true)
		s.BannerInfo = lipgloss.NewStyle().Background(lipgloss.Color("#23272E")).Foreground(lipgloss.Color("#E0E0E0")).Bold(true)
	}
}

// applySemanticStylesForTheme sets date-divider Info (not always equal to chat
// timestamp color) and System transcript line styles so errors are not the
// same color as normal system text.
func applySemanticStylesForTheme(s *themeStyles, theme string) {
	switch theme {
	case "patriot":
		s.Info = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#8FA3B8"))
		s.SystemMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("#E8EAED"))
		s.SystemMsgError = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8A80")).Bold(true)
		s.SystemMsgWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD54F")).Bold(true)
	case "retro":
		s.Info = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#66AA66"))
		s.SystemMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFCC"))
		s.SystemMsgError = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6E6E")).Bold(true)
		s.SystemMsgWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")).Bold(true)
	case "modern":
		s.Info = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#78909C"))
		s.SystemMsg = lipgloss.NewStyle().Foreground(lipgloss.Color("#B0BEC5"))
		s.SystemMsgError = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF5350")).Bold(true)
		s.SystemMsgWarn = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCA28")).Bold(true)
	case "system":
		s.Info = s.Time
		s.SystemMsg = s.Time
		s.SystemMsgError = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555"))
		s.SystemMsgWarn = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D7AF00"))
	default:
		s.Info = s.Time
		s.SystemMsg = s.Time
		s.SystemMsgError = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555"))
		s.SystemMsgWarn = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D7AF00"))
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
	themeKey := strings.ToLower(theme)
	applyBuiltinBannerStrips(&s, themeKey)
	s.Timestamp = s.Time
	applySemanticStylesForTheme(&s, themeKey)
	return s
}

type codeSnippetMsg struct {
	content string
}

type fileSendMsg struct {
	filePath string
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
			return wsErr{err}
		}
		return wsConnected{}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case wsConnected:
		m.connected = true
		m.banner = "[OK] Connected to server."
		m.reconnectDelay = time.Second // reset on success
		m.lastReadReceiptSentID = 0
		m.readReceiptFlushScheduled = false
		// Server replays history on each handshake; drop local transcript so we do
		// not duplicate messages after a disconnect or server restart.
		m.messages = nil
		m.reactions = make(map[int64]map[string]map[string]bool)
		m.typingUsers = make(map[string]time.Time)
		m.typingScopeDM = make(map[string]bool)
		m.typingChannel = make(map[string]string)
		m.receivedFiles = nil
		m.unreadCount = 0
		m.refreshTranscript()
		return m, m.listenWebSocket()
	case readReceiptFlushMsg:
		m.readReceiptFlushScheduled = false
		return m, m.flushReadReceipt()
	case wsReaderClosed:
		return m, nil
	case wsMsg:
		if v.Type == "userlist" {
			var ul UserList
			if err := json.Unmarshal(v.Data, &ul); err == nil {
				m.users = ul.Users
				m.updateSidebar()
			}
			return m, m.listenWebSocket()
		}
		if v.Type == "auth_failed" {
			log.Printf("Authentication failed - admin key rejected")
			var authFail map[string]string
			if err := json.Unmarshal(v.Data, &authFail); err == nil {
				log.Printf("Auth failure reason: %s", authFail["reason"])
			}
			fmt.Printf("[ERROR] Authentication failed: %s\n", authFail["reason"])
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
					m.banner = "[ERROR] Failed to send code snippet"
				}
			} else {
				// Send plain text message
				msg := shared.Message{Sender: m.cfg.Username, Content: v.content}
				if err := debugWebSocketWrite(m.conn, msg); err != nil {
					log.Printf("Failed to send code snippet: %v", err)
					m.banner = "[ERROR] Failed to send code snippet"
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
				m.banner = "[ERROR] Failed to read file: " + err.Error()
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
				m.banner = "[ERROR] File too large (max " + limitMsg + ")"
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

			if m.useE2E && m.keystore != nil {
				globalKey := m.keystore.GetSessionKey("global")
				if globalKey != nil {
					encData, encErr := m.keystore.EncryptRaw(data, "global")
					if encErr != nil {
						m.banner = "[ERROR] Failed to encrypt file: " + encErr.Error()
						m.sending = false
						m.showFilePicker = false
						return m, nil
					}
					msg.File.Data = encData
					msg.Encrypted = true
				}
			}

			err = m.conn.WriteJSON(msg)
			if err != nil {
				m.banner = "[ERROR] Failed to send file (connection lost)"
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
		if shouldNotify, level := m.shouldNotify(v); shouldNotify {
			m.notificationManager.Notify(v.Sender, v.Content, level)
		}

		switch v.Type {
		case shared.EditMessageType:
			for i, m2 := range m.messages {
				if m2.MessageID == v.MessageID {
					m.messages[i].Content = v.Content
					m.messages[i].Edited = true
					m.messages[i].Encrypted = v.Encrypted
					break
				}
			}
		case shared.DeleteMessage:
			for i, m2 := range m.messages {
				if m2.MessageID == v.MessageID {
					m.messages[i].Content = "[deleted]"
					m.messages[i].Type = shared.DeleteMessage
					break
				}
			}
		case shared.TypingMessage:
			if v.Sender != m.cfg.Username && dmTypingVisibleForTranscript(v, m.activeDMThread) {
				if m.typingScopeDM == nil {
					m.typingScopeDM = make(map[string]bool)
				}
				if m.typingChannel == nil {
					m.typingChannel = make(map[string]string)
				}
				m.typingUsers[v.Sender] = time.Now()
				m.typingScopeDM[v.Sender] = strings.TrimSpace(v.Recipient) != ""
				m.typingChannel[v.Sender] = normalizeChannel(v.Channel)
			}
		case shared.ReactionMessage:
			if v.Reaction != nil {
				tid := v.Reaction.TargetID
				if m.reactions[tid] == nil {
					m.reactions[tid] = make(map[string]map[string]bool)
				}
				if m.reactions[tid][v.Reaction.Emoji] == nil {
					m.reactions[tid][v.Reaction.Emoji] = make(map[string]bool)
				}
				if v.Reaction.IsRemoval {
					delete(m.reactions[tid][v.Reaction.Emoji], v.Sender)
					if len(m.reactions[tid][v.Reaction.Emoji]) == 0 {
						delete(m.reactions[tid], v.Reaction.Emoji)
					}
				} else {
					m.reactions[tid][v.Reaction.Emoji][v.Sender] = true
				}
			}
		case shared.ReadReceiptType:
			// Display-only; no state change needed
		default:
			if len(m.messages) >= maxMessages {
				m.messages = m.messages[len(m.messages)-maxMessages+1:]
			}
			m.messages = append(m.messages, v)
			sortMessagesByTimestamp(m.messages)

			if v.Type == shared.FileMessageType && v.File != nil {
				if m.receivedFiles == nil {
					m.receivedFiles = make(map[string]*shared.FileMeta)
				}
				m.receivedFiles[v.Sender+"/"+v.File.Filename] = v.File
			}

			if v.Type == shared.DirectMessage {
				partner := dmPartnerForMessage(v, m.cfg.Username)
				if partner != "" {
					if m.dmHidden != nil && strings.EqualFold(v.Sender, partner) && !strings.EqualFold(v.Sender, m.cfg.Username) {
						delete(m.dmHidden, normalizeDMUser(partner))
						m.saveDMUIState()
					}
					if strings.EqualFold(m.activeDMThread, partner) {
						m.markDMThreadRead(partner)
					}
				}
			}
		}

		m.rebuildDMUnreadCounts()

		wasAtBottom := m.viewport.AtBottom()
		m.refreshTranscript()
		if wasAtBottom {
			m.viewport.GotoBottom()
			m.unreadCount = 0
		} else if messageIncrementsUnread(m, v) {
			m.unreadCount++
		}
		m.sending = false
		cmds := []tea.Cmd{m.listenWebSocket()}
		if m.viewport.AtBottom() {
			if rr := m.scheduleReadReceiptFlush(); rr != nil {
				cmds = append(cmds, rr)
			}
		}
		return m, tea.Batch(cmds...)
	case wsUsernameError:
		log.Printf("Handling wsUsernameError: %s", v.message)
		m.connected = false
		m.banner = "[ERROR] " + v.message + " - Please restart with a different username"
		m.closeWebSocket()
		// Don't attempt to reconnect for username errors
		return m, nil
	case wsErr:
		m.connected = false
		m.banner = "[WARN] Connection lost. Reconnecting..."
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
			m.refreshTranscript()
			return m, nil
		case key.Matches(v, m.keys.TimeFormatHotkey):
			// Toggle time format
			m.twentyFourHour = !m.twentyFourHour
			m.cfg.TwentyFourHour = m.twentyFourHour
			_ = config.SaveConfig(m.configFilePath, m.cfg)
			m.banner = "Timestamp format: " + map[bool]string{true: "24h", false: "12h"}[m.twentyFourHour]
			m.refreshTranscript()
			return m, nil
		case key.Matches(v, m.keys.MessageInfoHotkey):
			m.showMessageMetadata = !m.showMessageMetadata
			m.banner = "Msg info: " + map[bool]string{true: "full", false: "minimal"}[m.showMessageMetadata]
			m.refreshTranscript()
			return m, nil
		case key.Matches(v, m.keys.ClearHotkey):
			// Clear chat history
			m.messages = nil
			m.viewport.SetContent("")
			m.updateSidebar()
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
			if m.viewport.AtBottom() {
				m.unreadCount = 0
				if rr := m.scheduleReadReceiptFlush(); rr != nil {
					return m, rr
				}
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
			if m.viewport.AtBottom() {
				m.unreadCount = 0
				if rr := m.scheduleReadReceiptFlush(); rr != nil {
					return m, rr
				}
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
							m.banner = fmt.Sprintf("[WARN] Clipboard unavailable in Termux. Text: %s", text)
						} else if err == context.DeadlineExceeded {
							m.banner = "[WARN] Clipboard operation timed out"
						} else {
							m.banner = "[ERROR] Failed to copy to clipboard: " + err.Error()
						}
					} else {
						m.banner = "[OK] Copied to clipboard"
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
						m.banner = "[WARN] Clipboard unavailable in Termux. Paste manually or use other methods."
					} else if err == context.DeadlineExceeded {
						m.banner = "[WARN] Clipboard operation timed out"
					} else {
						m.banner = "[ERROR] Failed to paste from clipboard: " + err.Error()
					}
				} else {
					m.textarea.SetValue(m.textarea.Value() + text)
					m.banner = "[OK] Pasted from clipboard"
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
							m.banner = fmt.Sprintf("[WARN] Clipboard unavailable in Termux. Text cleared: %s", text)
						} else if err == context.DeadlineExceeded {
							m.banner = "[WARN] Clipboard operation timed out"
						} else {
							m.banner = "[ERROR] Failed to cut to clipboard: " + err.Error()
						}
					} else {
						m.banner = "[OK] Cut to clipboard"
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
							m.banner = fmt.Sprintf("[WARN] Clipboard unavailable in Termux. Full text: %s", text)
						} else if err == context.DeadlineExceeded {
							m.banner = "[WARN] Clipboard operation timed out"
						} else {
							m.banner = "[ERROR] Failed to select all: " + err.Error()
						}
					} else {
						m.banner = "[OK] Selected all and copied to clipboard"
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
					m.banner = "[ERROR] Plugin name cannot be empty"
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
							m.banner = "[ERROR] Failed to read file: " + err.Error()
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
							m.banner = "[ERROR] File too large (max " + limitMsg + ")"
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
						if m.useE2E && m.keystore != nil {
							globalKey := m.keystore.GetSessionKey("global")
							if globalKey != nil {
								encData, encErr := m.keystore.EncryptRaw(data, "global")
								if encErr != nil {
									m.banner = "[ERROR] Failed to encrypt file: " + encErr.Error()
									m.textarea.SetValue("")
									return m, nil
								}
								msg.File.Data = encData
								msg.Encrypted = true
							}
						}
						if m.conn != nil {
							err := m.conn.WriteJSON(msg)
							if err != nil {
								m.banner = "[ERROR] Failed to send file (connection lost)"
								m.textarea.SetValue("")
								return m, m.listenWebSocket()
							}
							m.banner = "File sent: " + filename
						} else {
							m.banner = "[ERROR] Not connected to server"
						}
						m.textarea.SetValue("")
						return m, m.listenWebSocket()
					}
				}
				return m, nil
			}
			if strings.HasPrefix(text, ":savefile ") {
				filename := strings.TrimSpace(strings.TrimPrefix(text, ":savefile "))
				var file *shared.FileMeta
				if m.receivedFiles != nil {
					for _, f := range m.receivedFiles {
						if f.Filename == filename {
							file = f
						}
					}
				}
				if file == nil {
					m.banner = "[ERROR] No file with that name received."
					m.textarea.SetValue("")
					return m, nil
				}
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
					m.banner = "[ERROR] Failed to save file: " + err.Error()
				} else {
					m.banner = "[OK] File saved as: " + saveName
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":themes" {
				// List all available themes as a system message
				themes := ListAllThemes()
				var themeList strings.Builder
				themeList.WriteString("Available themes:\n\n")
				for _, themeName := range themes {
					themeList.WriteString("  • ")
					themeList.WriteString(GetThemeInfo(themeName))
					if themeName == m.cfg.Theme {
						themeList.WriteString(" [current]")
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
				m.refreshTranscript()
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
						m.refreshTranscript()
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
				m.updateSidebar()
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
				m.refreshTranscript()
				m.viewport.GotoBottom()
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":msginfo" {
				m.showMessageMetadata = !m.showMessageMetadata
				m.banner = "Msg info: " + map[bool]string{true: "full", false: "minimal"}[m.showMessageMetadata]
				m.refreshTranscript()
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
			}
			if strings.HasPrefix(text, ":edit ") {
				parts := strings.Fields(text)
				if len(parts) < 3 {
					m.messages = append(m.messages, shared.Message{Sender: "System", Content: "Usage: :edit <message_id> <new content>", CreatedAt: time.Now(), Type: shared.TextMessage})
				} else {
					id, err := strconv.ParseInt(parts[1], 10, 64)
					if err != nil {
						m.messages = append(m.messages, shared.Message{Sender: "System", Content: "Invalid message ID", CreatedAt: time.Now(), Type: shared.TextMessage})
					} else {
						newContent := strings.Join(parts[2:], " ")
						editMsg := shared.Message{Type: shared.EditMessageType, MessageID: id, Sender: m.cfg.Username}
						okToSend := true
						if m.useE2E && m.keystore != nil && m.keystore.GetSessionKey("global") != nil {
							if err := verifyKeystoreUnlocked(m.keystore); err != nil {
								m.messages = append(m.messages, shared.Message{Sender: "System", Content: "[ERROR] Keystore not unlocked: " + err.Error(), CreatedAt: time.Now(), Type: shared.TextMessage})
								okToSend = false
							} else if wire, encErr := encryptGlobalTextWireContent(m.keystore, m.cfg.Username, newContent); encErr != nil {
								m.messages = append(m.messages, shared.Message{Sender: "System", Content: "[ERROR] Failed to encrypt edit: " + encErr.Error(), CreatedAt: time.Now(), Type: shared.TextMessage})
								okToSend = false
							} else {
								editMsg.Content = wire
								editMsg.Encrypted = true
							}
						} else {
							editMsg.Content = newContent
						}
						if okToSend && m.conn != nil {
							_ = m.conn.WriteJSON(editMsg)
						}
					}
				}
				m.refreshTranscript()
				m.viewport.GotoBottom()
				m.textarea.SetValue("")
				return m, nil
			}
			if strings.HasPrefix(text, ":delete ") {
				parts := strings.Fields(text)
				if len(parts) < 2 {
					m.messages = append(m.messages, shared.Message{Sender: "System", Content: "Usage: :delete <message_id>", CreatedAt: time.Now(), Type: shared.TextMessage})
				} else {
					id, err := strconv.ParseInt(parts[1], 10, 64)
					if err != nil {
						m.messages = append(m.messages, shared.Message{Sender: "System", Content: "Invalid message ID", CreatedAt: time.Now(), Type: shared.TextMessage})
					} else {
						delMsg := shared.Message{Type: shared.DeleteMessage, MessageID: id, Sender: m.cfg.Username}
						if m.conn != nil {
							_ = m.conn.WriteJSON(delMsg)
						}
					}
				}
				m.refreshTranscript()
				m.viewport.GotoBottom()
				m.textarea.SetValue("")
				return m, nil
			}
			// :dmhide and :dms before :dm — ":dmhide" and ":dms" have prefix ":dm" and would otherwise be parsed as :dm.
			if strings.HasPrefix(text, ":dmhide") {
				parts := strings.Fields(text)
				target := ""
				if len(parts) >= 2 {
					target = parts[1]
				} else {
					target = m.activeDMThread
				}
				if strings.TrimSpace(target) == "" {
					m.banner = "[ERROR] Usage: :dmhide <user> (or run while viewing a DM thread)"
				} else {
					if m.dmHidden == nil {
						m.dmHidden = make(map[string]bool)
					}
					key := normalizeDMUser(target)
					m.dmHidden[key] = true
					if strings.EqualFold(m.activeDMThread, target) {
						m.activeDMThread = ""
						m.dmRecipient = ""
					}
					m.saveDMUIState()
					m.banner = fmt.Sprintf("DM thread hidden: %s", target)
				}
				m.textarea.SetValue("")
				m.refreshTranscript()
				m.viewport.GotoBottom()
				return m, nil
			}
			if text == ":dms" {
				threads := m.sidebarDMThreads()
				if len(threads) == 0 {
					m.messages = append(m.messages, shared.Message{
						Sender:    "System",
						Content:   "No DM conversations yet.",
						CreatedAt: time.Now(),
						Type:      shared.TextMessage,
					})
				} else {
					lines := make([]string, 0, len(threads))
					for _, thread := range threads {
						if thread.Unread > 0 {
							lines = append(lines, fmt.Sprintf("%s (%d unread)", thread.User, thread.Unread))
						} else {
							lines = append(lines, thread.User)
						}
					}
					m.messages = append(m.messages, shared.Message{
						Sender:    "System",
						Content:   "DM conversations: " + strings.Join(lines, ", "),
						CreatedAt: time.Now(),
						Type:      shared.TextMessage,
					})
				}
				m.textarea.SetValue("")
				m.refreshTranscript()
				m.viewport.GotoBottom()
				return m, nil
			}
			if strings.HasPrefix(text, ":dm") {
				parts := strings.Fields(text)
				if len(parts) == 1 {
					m.dmRecipient = ""
					m.activeDMThread = ""
					m.banner = "DM mode disabled, switched to global chat"
				} else if len(parts) == 2 {
					target := parts[1]
					if strings.EqualFold(target, "off") || strings.EqualFold(target, "exit") || strings.EqualFold(target, "general") {
						m.dmRecipient = ""
						m.activeDMThread = ""
						m.banner = "DM mode disabled, switched to global chat"
						m.textarea.SetValue("")
						m.refreshTranscript()
						m.viewport.GotoBottom()
						return m, nil
					}
					if m.dmRecipient == target {
						m.dmRecipient = ""
						m.activeDMThread = ""
						m.banner = "DM mode disabled, switched to global chat"
					} else {
						m.dmRecipient = target
						m.activeDMThread = target
						if m.dmHidden != nil {
							delete(m.dmHidden, normalizeDMUser(target))
						}
						m.markDMThreadRead(target)
						m.banner = fmt.Sprintf("DM mode: conversation with %s", target)
					}
				} else {
					target := parts[1]
					content := strings.Join(parts[2:], " ")
					dmMsg := shared.Message{Type: shared.DirectMessage, Sender: m.cfg.Username, Recipient: target, Content: content}
					if m.conn != nil {
						_ = m.conn.WriteJSON(dmMsg)
					}
					m.dmRecipient = target
					m.activeDMThread = target
					if m.dmHidden != nil {
						delete(m.dmHidden, normalizeDMUser(target))
					}
					m.markDMThreadRead(target)
				}
				m.textarea.SetValue("")
				m.rebuildDMUnreadCounts()
				m.refreshTranscript()
				m.viewport.GotoBottom()
				return m, nil
			}
			if strings.HasPrefix(text, ":search ") {
				query := strings.TrimSpace(strings.TrimPrefix(text, ":search "))
				if query != "" {
					searchMsg := shared.Message{Type: shared.SearchMessage, Sender: m.cfg.Username, Content: query}
					if m.conn != nil {
						_ = m.conn.WriteJSON(searchMsg)
					}
				} else {
					m.messages = append(m.messages, shared.Message{Sender: "System", Content: "Usage: :search <query>", CreatedAt: time.Now(), Type: shared.TextMessage})
					m.refreshTranscript()
					m.viewport.GotoBottom()
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if strings.HasPrefix(text, ":react ") {
				parts := strings.Fields(text)
				if len(parts) < 3 {
					m.messages = append(m.messages, shared.Message{Sender: "System", Content: "Usage: :react <message_id> <emoji>", CreatedAt: time.Now(), Type: shared.TextMessage})
					m.refreshTranscript()
					m.viewport.GotoBottom()
				} else {
					id, err := strconv.ParseInt(parts[1], 10, 64)
					if err != nil {
						m.messages = append(m.messages, shared.Message{Sender: "System", Content: "Invalid message ID", CreatedAt: time.Now(), Type: shared.TextMessage})
						m.refreshTranscript()
						m.viewport.GotoBottom()
					} else {
						emoji := resolveReactionEmoji(parts[2])
						reactMsg := shared.Message{
							Type:   shared.ReactionMessage,
							Sender: m.cfg.Username,
							Reaction: &shared.ReactionMeta{
								Emoji:    emoji,
								TargetID: id,
							},
						}
						if m.conn != nil {
							_ = m.conn.WriteJSON(reactMsg)
						}
					}
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if strings.HasPrefix(text, ":pin ") {
				parts := strings.Fields(text)
				if len(parts) < 2 {
					m.messages = append(m.messages, shared.Message{Sender: "System", Content: "Usage: :pin <message_id>", CreatedAt: time.Now(), Type: shared.TextMessage})
					m.refreshTranscript()
					m.viewport.GotoBottom()
				} else {
					id, err := strconv.ParseInt(parts[1], 10, 64)
					if err != nil {
						m.messages = append(m.messages, shared.Message{Sender: "System", Content: "Invalid message ID", CreatedAt: time.Now(), Type: shared.TextMessage})
						m.refreshTranscript()
						m.viewport.GotoBottom()
					} else {
						pinMsg := shared.Message{Type: shared.PinMessage, MessageID: id, Sender: m.cfg.Username}
						if m.conn != nil {
							_ = m.conn.WriteJSON(pinMsg)
						}
					}
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":pinned" {
				pinMsg := shared.Message{Type: shared.PinMessage, Sender: m.cfg.Username, Content: "list"}
				if m.conn != nil {
					_ = m.conn.WriteJSON(pinMsg)
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":channels" {
				if m.conn != nil {
					_ = m.conn.WriteJSON(shared.Message{
						Type:   shared.ListChannelsType,
						Sender: m.cfg.Username,
					})
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if strings.HasPrefix(text, ":join ") {
				channel := strings.TrimSpace(strings.TrimPrefix(text, ":join "))
				if channel != "" {
					m.currentChannel = strings.ToLower(channel)
					joinMsg := shared.Message{
						Type:    shared.JoinChannelType,
						Sender:  m.cfg.Username,
						Channel: channel,
					}
					_ = m.conn.WriteJSON(joinMsg)
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":leave" {
				m.currentChannel = "general"
				leaveMsg := shared.Message{
					Type:   shared.LeaveChannelType,
					Sender: m.cfg.Username,
				}
				_ = m.conn.WriteJSON(leaveMsg)
				m.textarea.SetValue("")
				return m, nil
			}
			if text == ":export" || strings.HasPrefix(text, ":export ") {
				filename := "marchat-export.txt"
				if strings.HasPrefix(text, ":export ") {
					filename = strings.TrimSpace(strings.TrimPrefix(text, ":export "))
				}
				var sb strings.Builder
				for _, msg := range m.messages {
					ts := msg.CreatedAt.Format("2006-01-02 15:04:05")
					sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", ts, msg.Sender, msg.Content))
				}
				if err := os.WriteFile(filename, []byte(sb.String()), 0644); err != nil {
					m.messages = append(m.messages, shared.Message{
						Sender: "System", Content: "Export failed: " + err.Error(),
						CreatedAt: time.Now(), Type: shared.TextMessage,
					})
				} else {
					m.messages = append(m.messages, shared.Message{
						Sender: "System", Content: "History exported to " + filename,
						CreatedAt: time.Now(), Type: shared.TextMessage,
					})
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if text != "" {
				m.sending = true
				if m.conn != nil {
					// Check if this is a server-side command (admin/plugin) that should bypass encryption
					// Client-side commands are handled above and never reach this point
					clientOnlyCommands := []string{":theme", ":time", ":msginfo", ":clear", ":bell", ":bell-mention", ":code", ":sendfile", ":savefile", ":q", ":edit", ":delete", ":dm", ":dms", ":dmhide", ":search", ":react", ":pin", ":pinned", ":join", ":leave", ":channels", ":export"}
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
					// All users can send server commands unencrypted; server will check permissions
					isServerCommand := strings.HasPrefix(text, ":") && !isClientCommand

					if isServerCommand {
						// Send as admin command type to bypass encryption
						// Server will check permissions for plugin commands and built-in admin commands
						msg := shared.Message{
							Sender:  m.cfg.Username,
							Content: text,
							Type:    shared.AdminCommandType,
						}
						exthook.FireSend(msg)
						err := m.conn.WriteJSON(msg)
						if err != nil {
							m.banner = "[ERROR] Failed to send command (connection lost)"
							m.sending = false
							return m, m.listenWebSocket()
						}
						m.banner = ""
						// Do not wait for a chat echo: unknown or silent server paths used to leave
						// m.sending true forever because only shared.Message clears it.
						m.sending = false
					} else if m.dmRecipient != "" {
						dmMsg := shared.Message{
							Type:      shared.DirectMessage,
							Sender:    m.cfg.Username,
							Recipient: m.dmRecipient,
							Content:   text,
						}
						exthook.FireSend(dmMsg)
						if err := m.conn.WriteJSON(dmMsg); err != nil {
							m.banner = "[ERROR] Failed to send DM (connection lost)"
							m.sending = false
							return m, m.listenWebSocket()
						}
						m.banner = ""
						m.sending = false
					} else if m.useE2E {
						log.Printf("DEBUG: Attempting to send global encrypted message: '%s'", text)

						// Validate keystore is unlocked
						if err := verifyKeystoreUnlocked(m.keystore); err != nil {
							m.banner = fmt.Sprintf("[ERROR] Keystore not unlocked: %v", err)
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
						exthook.FireSend(shared.Message{
							Sender:  m.cfg.Username,
							Content: text,
							Type:    shared.TextMessage,
						})
						if err := debugEncryptAndSend(recipients, text, m.conn, m.keystore, m.cfg.Username); err != nil {
							m.banner = fmt.Sprintf("[ERROR] Global encryption failed: %v", err)
							m.sending = false
							m.textarea.SetValue("")
							return m, nil
						}

						log.Printf("DEBUG: Global encrypted message sent successfully")
						m.banner = ""
						m.sending = false
					} else {
						// Send plain text message
						msg := shared.Message{Sender: m.cfg.Username, Content: text}
						exthook.FireSend(msg)
						if err := debugWebSocketWrite(m.conn, msg); err != nil {
							m.banner = "[ERROR] Failed to send (connection lost)"
							m.sending = false
							return m, m.listenWebSocket()
						}
						m.banner = ""
						m.sending = false
					}
				}
				m.textarea.SetValue("")
				return m, m.listenWebSocket()
			}
			return m, nil
		case key.Matches(v, key.NewBinding(key.WithKeys("alt+enter", "ctrl+j"))):
			current := m.textarea.Value()
			m.textarea.SetValue(current + "\n")
			m.textarea.CursorEnd()
			return m, nil
		case key.Matches(v, key.NewBinding(key.WithKeys("tab"))):
			text := m.textarea.Value()
			words := strings.Fields(text)
			if len(words) > 0 {
				last := words[len(words)-1]
				if strings.HasPrefix(last, "@") && len(last) > 1 {
					partial := strings.ToLower(last[1:])
					for _, user := range m.users {
						if strings.HasPrefix(strings.ToLower(user), partial) {
							words[len(words)-1] = "@" + user + " "
							m.textarea.SetValue(strings.Join(words, " "))
							m.textarea.CursorEnd()
							break
						}
					}
				}
			}
			return m, nil
		default:
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

			if time.Since(m.lastTypingSent) > 2*time.Second && m.conn != nil {
				m.lastTypingSent = time.Now()
				typingMsg := shared.Message{Type: shared.TypingMessage, Sender: m.cfg.Username}
				if r := strings.TrimSpace(m.dmRecipient); r != "" {
					typingMsg.Recipient = r
				} else {
					typingMsg.Channel = normalizeChannel(m.currentChannel)
				}
				go func(c *websocket.Conn) {
					_ = c.WriteJSON(typingMsg)
				}(m.conn)
			}

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

		m.refreshTranscript()
		m.viewport.GotoBottom()
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
							m.banner = "[ERROR] Failed to open URL: " + err.Error()
						} else {
							m.banner = "[OK] Opening URL: " + clickedURL
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

func (m *model) View() string {
	// Header with version
	headerText := fmt.Sprintf(" marchat %s ", shared.ClientVersion)
	header := m.styles.Header.Width(m.viewport.Width + userListWidth + 4).Render(headerText)

	footerText := buildStatusFooter(m.connected, m.showHelp, m.unreadCount, m.useE2E, m.currentChannel, m.activeDMThread)
	footer := m.styles.Footer.Width(m.viewport.Width + userListWidth + 4).Render(footerText)

	// Banner
	var bannerBox string
	if m.banner != "" || m.sending {
		bannerText := m.banner
		if m.sending {
			if bannerText != "" {
				bannerText += " [Sending...]"
			} else {
				bannerText = "[Sending...]"
			}
		}
		kind := stripKindForBanner(bannerText)
		fullW := chromeFullWidth(m.viewport.Width)
		bannerShown := layoutBannerForStrip(bannerText, fullW)
		bannerBox = m.styles.BannerStrip(kind).
			Width(fullW).
			PaddingLeft(1).
			Render(bannerShown)
	}

	// Chat and user list layout
	chatBoxStyle := m.styles.Box
	chatPanel := chatBoxStyle.Width(m.viewport.Width).Render(m.viewport.View())
	userPanel := m.userListViewport.View()
	row := lipgloss.JoinHorizontal(lipgloss.Top, userPanel, chatPanel)

	// Typing indicator
	var typingLine string
	now := time.Now()
	var activeTypers []string
	for user, lastTyped := range m.typingUsers {
		if now.Sub(lastTyped) >= m.typingTimeout {
			continue
		}
		isDMScope := m.typingScopeDM != nil && m.typingScopeDM[user]
		if isDMScope && !strings.EqualFold(strings.TrimSpace(m.activeDMThread), user) {
			continue
		}
		if !isDMScope && strings.TrimSpace(m.activeDMThread) != "" {
			continue
		}
		if !isDMScope {
			typingChannel := "general"
			if m.typingChannel != nil {
				typingChannel = normalizeChannel(m.typingChannel[user])
			}
			if typingChannel != normalizeChannel(m.currentChannel) {
				continue
			}
		}
		if strings.EqualFold(strings.TrimSpace(user), strings.TrimSpace(m.cfg.Username)) {
			continue
		}
		activeTypers = append(activeTypers, user)
	}
	if len(activeTypers) > 0 {
		sort.Strings(activeTypers)
		if len(activeTypers) == 1 {
			typingLine = activeTypers[0] + " is typing..."
		} else {
			typingLine = strings.Join(activeTypers, ", ") + " are typing..."
		}
	}
	typingIndicator := lipgloss.NewStyle().Faint(true).Italic(true).Width(m.viewport.Width).Render(typingLine)

	// DM mode indicator
	var dmIndicator string
	if m.dmRecipient != "" {
		dmIndicator = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5F5F")).Render(fmt.Sprintf("[DM: %s] ", m.dmRecipient))
	}

	// Input
	inputContent := m.textarea.View()
	if dmIndicator != "" {
		inputContent = dmIndicator + inputContent
	}
	inputPanel := m.styles.Input.Width(m.viewport.Width).Render(inputContent)

	// Compose layout
	ui := lipgloss.JoinVertical(lipgloss.Left,
		header,
		bannerBox,
		row,
		typingIndicator,
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

func main() {
	flag.Parse()

	if *runDoctorJSON {
		if err := doctor.RunClient(doctor.Options{JSON: true}); err != nil {
			fmt.Fprintf(os.Stderr, "doctor: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *runDoctor {
		if err := doctor.RunClient(doctor.Options{}); err != nil {
			fmt.Fprintf(os.Stderr, "doctor: %v\n", err)
			os.Exit(1)
		}
		return
	}

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

	// Skip profile picker when CLI gives enough to connect: server, username, and admin key if --admin.
	// If --e2e without --keystore-passphrase, prompt once on the terminal (unless --non-interactive).
	if *nonInteractive || directConnectFromFlags(*serverURL, *username, *isAdmin, *adminKey) {
		// Use traditional flag-based configuration
		cfg, err = loadConfigFromFlags(*configPath, *serverURL, *username, *theme, *isAdmin, *useE2E, *skipTLSVerify)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		keystorePass := *keystorePassphrase
		if cfg.UseE2E && keystorePass == "" {
			if *nonInteractive {
				fmt.Fprintln(os.Stderr, "Error: --e2e requires --keystore-passphrase when using --non-interactive")
				os.Exit(1)
			}
			var readErr error
			keystorePass, readErr = readKeystorePassphraseFromTerminal()
			if readErr != nil {
				fmt.Fprintf(os.Stderr, "Error reading keystore passphrase: %v\n", readErr)
				os.Exit(1)
			}
		}

		if err := validateFlags(*isAdmin, *adminKey, cfg.UseE2E, keystorePass); err != nil {
			fmt.Printf("Error: %v\n", err)
			flag.Usage()
			os.Exit(1)
		}

		initializeClient(cfg, *adminKey, keystorePass)

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
			cliWelcomeLine("Welcome to marchat! Let's get you set up...")

			configResult, keystorePass, err := config.RunInteractiveConfig()
			if err != nil {
				cliPrintErr(fmt.Sprintf("Configuration error: %v", err))
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
				cliPrintWarn(fmt.Sprintf("Warning: Could not save profile: %v", err))
			}
			cliPrintOK("[OK] Configuration saved! Next time you can use --auto or --quick-start for faster connections.")

		} else {
			// Existing user - show profile selection with option to create new
			cliPrintMuted("Select a connection profile or create a new one...")

			// Sort profiles by last used (most recent first)
			sort.Slice(profiles.Profiles, func(i, j int) bool {
				return profiles.Profiles[i].LastUsed > profiles.Profiles[j].LastUsed
			})

			selectedProfile, isCreateNew, err := config.RunProfileSelectionWithNew(profiles.Profiles, loader)
			if err != nil {
				cliPrintErr(fmt.Sprintf("Profile selection error: %v", err))
				os.Exit(1)
			}

			if isCreateNew {
				// User chose to create a new profile
				cliPrintMuted("Creating a new connection profile...")

				configResult, keystorePass, err := config.RunInteractiveConfig()
				if err != nil {
					cliPrintErr(fmt.Sprintf("Configuration error: %v", err))
					os.Exit(1)
				}
				cfg = configResult
				adminKeyFromConfig = cfg.AdminKey
				keystorePassFromConfig = keystorePass

				// Save as a new profile
				profileName := config.NextDefaultProfileName(profiles.Profiles)
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
					cliPrintWarn(fmt.Sprintf("Warning: Could not save profile: %v", err))
				}
				cliPrintOK(fmt.Sprintf("[OK] Configuration saved as '%s'! You can use --auto or --quick-start for faster connections.", profileName))

			} else {
				// We now have the actual profile object, not just an index!
				// Reload profiles in case they were modified during selection
				profiles, err = loader.LoadProfiles()
				if err != nil {
					cliPrintErr(fmt.Sprintf("Error reloading profiles: %v", err))
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
					cliPrintErr("Error: Selected profile no longer exists")
					os.Exit(1)
				}

				// User selected an existing profile
				profile := &profiles.Profiles[profileIndex]
				cliSelectedProfile(profile.Name)

				// Update last used timestamp
				profile.LastUsed = time.Now().Unix()
				if err := loader.SaveProfiles(profiles); err != nil {
					// Log error but don't fail the connection
					cliPrintWarn(fmt.Sprintf("Warning: Could not update profile usage timestamp: %v", err))
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
					cliPrintErr(fmt.Sprintf("Error getting sensitive data: %v", err))
					os.Exit(1)
				}
			}
		}

		// Continue with existing client initialization...
		initializeClient(cfg, adminKeyFromConfig, keystorePassFromConfig)
	}
}

// directConnectFromFlags is true when the user supplied enough CLI args to skip the profile menu.
// Keystore passphrase may still be prompted when --e2e is set (see main).
func directConnectFromFlags(serverURL, username string, isAdmin bool, adminKey string) bool {
	if serverURL == "" || username == "" {
		return false
	}
	if isAdmin && adminKey == "" {
		return false
	}
	return true
}

func readKeystorePassphraseFromTerminal() (string, error) {
	fmt.Fprint(os.Stderr, "Keystore passphrase: ")
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
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
	cliPrintConnecting(cfg.ServerURL, cfg.Username)

	// Termux clipboard availability notice
	if isTermux() {
		cliPrintWarn("[WARN] Termux environment detected")
		if !checkClipboardSupport() {
			cliPrintWarn("[WARN] Clipboard operations may be unavailable - text will be shown in banner")
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
			cliPrintErr(fmt.Sprintf("Error getting keystore path: %v", err))
			os.Exit(1)
		}
		keystore = crypto.NewKeyStore(keystorePath)

		if err := keystore.Initialize(keystorePassphraseParam); err != nil {
			cliPrintErr(fmt.Sprintf("Error initializing keystore: %v", err))
			os.Exit(1)
		}

		cliPrintOK("E2E encryption enabled")
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
	userListVp.SetContent(renderUserList([]string{cfg.Username}, cfg.Username, getThemeStyles(cfg.Theme), 18, cfg.IsAdmin, -1, nil))

	helpVp := viewport.New(70, 20) // initial size, will be adjusted on resize

	// Initialize admin menu viewports
	dbMenuVp := viewport.New(60, 15)

	// Additional keystore initialization if E2E is enabled
	if cfg.UseE2E && keystore != nil {
		// Check environment variable status
		if envKey := os.Getenv("MARCHAT_GLOBAL_E2E_KEY"); envKey != "" {
			cliPrintAccent("Using global E2E key from environment variable")
		} else {
			cliPrintMuted("No MARCHAT_GLOBAL_E2E_KEY environment variable found")
		}

		// Verify keystore is properly unlocked
		if err := verifyKeystoreUnlocked(keystore); err != nil {
			cliPrintErr(fmt.Sprintf("Keystore unlock verification failed: %v", err))
			os.Exit(1)
		}

		// Display global key info
		if globalKey := keystore.GetGlobalKey(); globalKey != nil {
			cliPrintGlobalKeyID(globalKey.KeyID)
		} else {
			cliPrintErr("Global key not available")
			os.Exit(1)
		}

		// Test encryption roundtrip (non-blocking for production use)
		if err := validateEncryptionRoundtrip(keystore, cfg.Username); err != nil {
			cliPrintWarn(fmt.Sprintf("Encryption validation failed: %v", err))
			cliPrintWarn("E2E encryption will continue but may have issues")
			log.Printf("WARNING: Encryption validation failed: %v", err)
		} else {
			cliPrintOK("Encryption validation passed")
		}

		keystorePath, _ := config.GetKeystorePath()
		cliPrintKeystorePath(keystorePath)
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
		cfg:                 *cfg,
		configFilePath:      configFilePath,
		profileName:         getCurrentProfileName(cfg),
		textarea:            ta,
		viewport:            vp,
		styles:              getThemeStyles(cfg.Theme),
		users:               []string{cfg.Username},
		userListViewport:    userListVp,
		helpViewport:        helpVp,
		dbMenuViewport:      dbMenuVp,
		twentyFourHour:      cfg.TwentyFourHour,
		showMessageMetadata: true,
		keystore:            keystore,
		useE2E:              cfg.UseE2E,
		keys:                newKeyMap(),
		selectedUserIndex:   -1,
		typingUsers:         make(map[string]time.Time),
		typingScopeDM:       make(map[string]bool),
		typingChannel:       make(map[string]string),
		typingTimeout:       3 * time.Second,
		dmUnread:            make(map[string]int),
		dmLastSeenID:        make(map[string]int64),
		dmHidden:            make(map[string]bool),
		dmStatePath:         filepath.Join(getClientConfigDir(), "dm_state.json"),
		reactions:           make(map[int64]map[string]map[string]bool),
	}
	m.loadDMUIState()
	m.rebuildDMUnreadCounts()
	m.updateSidebar()

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
