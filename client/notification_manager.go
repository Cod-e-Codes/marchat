package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// NotificationLevel defines the priority/type of notification
type NotificationLevel int

const (
	NotificationLevelInfo NotificationLevel = iota
	NotificationLevelMention
	NotificationLevelDM
	NotificationLevelUrgent
)

// NotificationMode defines how notifications should be delivered
type NotificationMode int

const (
	NotificationModeNone NotificationMode = iota
	NotificationModeBell
	NotificationModeDesktop
	NotificationModeBoth
)

// NotificationConfig holds user preferences for notifications
type NotificationConfig struct {
	// Global notification mode
	Mode NotificationMode

	// Bell settings
	BellEnabled     bool
	BellOnMention   bool
	BellOnDM        bool
	BellMinInterval time.Duration

	// Desktop notification settings
	DesktopEnabled   bool
	DesktopOnMention bool
	DesktopOnDM      bool
	DesktopOnAll     bool
	DesktopTitle     string

	// Quiet hours (optional)
	QuietHoursEnabled bool
	QuietHoursStart   int // Hour in 24h format (e.g., 22 for 10 PM)
	QuietHoursEnd     int // Hour in 24h format (e.g., 8 for 8 AM)

	// Focus mode (temporary mute)
	FocusModeEnabled bool
	FocusModeUntil   time.Time
}

// DefaultNotificationConfig returns sensible defaults
func DefaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		Mode:              NotificationModeBell,
		BellEnabled:       true,
		BellOnMention:     false, // Bell for all messages by default
		BellOnDM:          true,
		BellMinInterval:   500 * time.Millisecond,
		DesktopEnabled:    false, // Disabled by default (requires setup)
		DesktopOnMention:  true,
		DesktopOnDM:       true,
		DesktopOnAll:      false,
		DesktopTitle:      "marchat",
		QuietHoursEnabled: false,
		QuietHoursStart:   22,
		QuietHoursEnd:     8,
		FocusModeEnabled:  false,
	}
}

// NotificationManager handles all notification delivery
type NotificationManager struct {
	config      NotificationConfig
	lastBell    time.Time
	lastDesktop time.Time
	mu          sync.Mutex

	// Platform-specific notification support
	desktopSupported bool
	notifyCommand    string
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(config NotificationConfig) *NotificationManager {
	nm := &NotificationManager{
		config: config,
	}

	// Detect desktop notification support
	nm.detectDesktopSupport()

	return nm
}

// detectDesktopSupport checks if desktop notifications are available
func (nm *NotificationManager) detectDesktopSupport() {
	switch runtime.GOOS {
	case "darwin":
		// macOS: osascript
		if _, err := exec.LookPath("osascript"); err == nil {
			nm.desktopSupported = true
			nm.notifyCommand = "osascript"
		}
	case "linux":
		// Linux: notify-send
		if _, err := exec.LookPath("notify-send"); err == nil {
			nm.desktopSupported = true
			nm.notifyCommand = "notify-send"
		}
	case "windows":
		// Windows: PowerShell toast notifications
		if _, err := exec.LookPath("powershell"); err == nil {
			nm.desktopSupported = true
			nm.notifyCommand = "powershell"
		}
	}
}

// Notify sends a notification based on the message and configuration
func (nm *NotificationManager) Notify(sender, content string, level NotificationLevel) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Check if we're in quiet hours
	if nm.isQuietHours() {
		return
	}

	// Check focus mode
	if nm.config.FocusModeEnabled && time.Now().Before(nm.config.FocusModeUntil) {
		return
	}

	// Determine if we should notify based on level and config
	shouldBell := nm.shouldNotifyBell(level)
	shouldDesktop := nm.shouldNotifyDesktop(level)

	// Send notifications
	if shouldBell {
		nm.playBell()
	}

	if shouldDesktop {
		nm.sendDesktopNotification(sender, content)
	}
}

// shouldNotifyBell determines if a bell should be played
func (nm *NotificationManager) shouldNotifyBell(level NotificationLevel) bool {
	if !nm.config.BellEnabled {
		return false
	}

	// Check rate limiting
	now := time.Now()
	if now.Sub(nm.lastBell) < nm.config.BellMinInterval {
		return false
	}

	switch level {
	case NotificationLevelDM:
		return nm.config.BellOnDM
	case NotificationLevelMention:
		return true
	case NotificationLevelUrgent:
		return true
	case NotificationLevelInfo:
		return !nm.config.BellOnMention
	}

	return false
}

// shouldNotifyDesktop determines if a desktop notification should be sent
func (nm *NotificationManager) shouldNotifyDesktop(level NotificationLevel) bool {
	if !nm.config.DesktopEnabled || !nm.desktopSupported {
		return false
	}

	// Check rate limiting (2 seconds for desktop)
	now := time.Now()
	if now.Sub(nm.lastDesktop) < 2*time.Second {
		return false
	}

	// Check level-based rules
	switch level {
	case NotificationLevelDM:
		return nm.config.DesktopOnDM
	case NotificationLevelMention:
		return nm.config.DesktopOnMention
	case NotificationLevelUrgent:
		return true
	case NotificationLevelInfo:
		return nm.config.DesktopOnAll
	}

	return false
}

// playBell plays the ASCII bell sound
func (nm *NotificationManager) playBell() {
	nm.lastBell = time.Now()
	fmt.Print("\a")
}

// sendDesktopNotification sends a platform-specific desktop notification
func (nm *NotificationManager) sendDesktopNotification(title, message string) {
	nm.lastDesktop = time.Now()

	// Truncate long messages
	if len(message) > 100 {
		message = message[:97] + "..."
	}

	go func() {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			// macOS osascript
			script := fmt.Sprintf(`display notification "%s" with title "%s"`,
				escapeForAppleScript(message),
				escapeForAppleScript(title))
			cmd = exec.Command("osascript", "-e", script)

		case "linux":
			// Linux notify-send
			cmd = exec.Command("notify-send",
				title,
				message,
				"-u", "normal",
				"-t", "5000") // 5 second timeout

		case "windows":
			// Windows PowerShell toast
			script := fmt.Sprintf(`
				[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
				[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
				$template = @"
<toast>
	<visual>
		<binding template="ToastText02">
			<text id="1">%s</text>
			<text id="2">%s</text>
		</binding>
	</visual>
</toast>
"@
				$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
				$xml.LoadXml($template)
				$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
				[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("%s").Show($toast)
			`, escapeForPowerShell(title),
				escapeForPowerShell(message),
				escapeForPowerShell(nm.config.DesktopTitle))
			cmd = exec.Command("powershell", "-Command", script)
		}

		if cmd != nil {
			_ = cmd.Run() // Ignore errors for notifications
		}
	}()
}

// isQuietHours checks if we're currently in quiet hours
func (nm *NotificationManager) isQuietHours() bool {
	if !nm.config.QuietHoursEnabled {
		return false
	}

	now := time.Now()
	currentHour := now.Hour()

	// Handle overnight quiet hours (e.g., 22:00 to 08:00)
	if nm.config.QuietHoursStart > nm.config.QuietHoursEnd {
		return currentHour >= nm.config.QuietHoursStart || currentHour < nm.config.QuietHoursEnd
	}

	// Handle same-day quiet hours (e.g., 13:00 to 14:00)
	return currentHour >= nm.config.QuietHoursStart && currentHour < nm.config.QuietHoursEnd
}

// UpdateConfig updates the notification configuration
func (nm *NotificationManager) UpdateConfig(config NotificationConfig) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config = config
}

// GetConfig returns the current configuration
func (nm *NotificationManager) GetConfig() NotificationConfig {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	return nm.config
}

// EnableFocusMode temporarily disables notifications
func (nm *NotificationManager) EnableFocusMode(duration time.Duration) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.FocusModeEnabled = true
	nm.config.FocusModeUntil = time.Now().Add(duration)
}

// DisableFocusMode re-enables notifications
func (nm *NotificationManager) DisableFocusMode() {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.FocusModeEnabled = false
}

// IsDesktopSupported returns whether desktop notifications are available
func (nm *NotificationManager) IsDesktopSupported() bool {
	return nm.desktopSupported
}

// ToggleBell toggles bell notifications on/off
func (nm *NotificationManager) ToggleBell() bool {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.BellEnabled = !nm.config.BellEnabled
	return nm.config.BellEnabled
}

// ToggleBellOnMention toggles mention-only bell mode
func (nm *NotificationManager) ToggleBellOnMention() bool {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.BellOnMention = !nm.config.BellOnMention
	return nm.config.BellOnMention
}

// ToggleDesktop toggles desktop notifications on/off
func (nm *NotificationManager) ToggleDesktop() bool {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.DesktopEnabled = !nm.config.DesktopEnabled
	return nm.config.DesktopEnabled
}

// SetMode sets the notification mode
func (nm *NotificationManager) SetMode(mode NotificationMode) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.Mode = mode

	// Update individual settings based on mode
	switch mode {
	case NotificationModeNone:
		nm.config.BellEnabled = false
		nm.config.DesktopEnabled = false
	case NotificationModeBell:
		nm.config.BellEnabled = true
		nm.config.DesktopEnabled = false
	case NotificationModeDesktop:
		nm.config.BellEnabled = false
		nm.config.DesktopEnabled = true
		nm.config.DesktopOnAll = true // Enable notifications for all messages
	case NotificationModeBoth:
		nm.config.BellEnabled = true
		nm.config.DesktopEnabled = true
		nm.config.DesktopOnAll = true // Enable notifications for all messages
	}
}

// SetQuietHours enables or disables quiet hours
func (nm *NotificationManager) SetQuietHours(enabled bool, start, end int) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.config.QuietHoursEnabled = enabled
	nm.config.QuietHoursStart = start
	nm.config.QuietHoursEnd = end
}

// Helper functions for escaping strings

func escapeForAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func escapeForPowerShell(s string) string {
	s = strings.ReplaceAll(s, `"`, `""`)
	return s
}
