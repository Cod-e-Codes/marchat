package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/shared"
)

func TestMainFunctionExists(t *testing.T) {
	// This test ensures the main function exists and can be called
	// We can't actually call main() in tests, but we can verify the package compiles
	if testing.Short() {
		t.Skip("Skipping main function test in short mode")
	}
}

func TestFlagParsing(t *testing.T) {
	// Test that flag parsing works correctly
	// Reset flags to avoid conflicts with other tests
	flag.CommandLine = flag.NewFlagSet("test", flag.ExitOnError)

	// Re-define flags for testing
	serverURL := flag.String("server", "", "Server URL")
	username := flag.String("username", "", "Username")
	isAdmin := flag.Bool("admin", false, "Connect as admin (requires --admin-key)")
	adminKey := flag.String("admin-key", "", "Admin key for privileged commands")
	useE2E := flag.Bool("e2e", false, "Enable end-to-end encryption")
	keystorePassphrase := flag.String("keystore-passphrase", "", "Passphrase for keystore (required for E2E)")

	// Test flag parsing with various combinations
	testCases := []struct {
		name     string
		args     []string
		expected map[string]interface{}
	}{
		{
			name: "basic flags",
			args: []string{"-server", "ws://localhost:8080", "-username", "testuser"},
			expected: map[string]interface{}{
				"server":   "ws://localhost:8080",
				"username": "testuser",
				"admin":    false,
				"e2e":      false,
			},
		},
		{
			name: "admin flags",
			args: []string{"-server", "ws://localhost:8080", "-username", "admin", "-admin", "-admin-key", "secret"},
			expected: map[string]interface{}{
				"server":   "ws://localhost:8080",
				"username": "admin",
				"admin":    true,
				"adminKey": "secret",
			},
		},
		{
			name: "e2e flags",
			args: []string{"-server", "ws://localhost:8080", "-username", "user", "-e2e", "-keystore-passphrase", "pass"},
			expected: map[string]interface{}{
				"server":             "ws://localhost:8080",
				"username":           "user",
				"e2e":                true,
				"keystorePassphrase": "pass",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset flag values
			flag.CommandLine = flag.NewFlagSet("test", flag.ExitOnError)
			serverURL = flag.String("server", "", "Server URL")
			username = flag.String("username", "", "Username")
			isAdmin = flag.Bool("admin", false, "Connect as admin (requires --admin-key)")
			adminKey = flag.String("admin-key", "", "Admin key for privileged commands")
			useE2E = flag.Bool("e2e", false, "Enable end-to-end encryption")
			keystorePassphrase = flag.String("keystore-passphrase", "", "Passphrase for keystore (required for E2E)")

			err := flag.CommandLine.Parse(tc.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Verify flag values
			if expected, ok := tc.expected["server"]; ok && *serverURL != expected {
				t.Errorf("Expected server %v, got %v", expected, *serverURL)
			}
			if expected, ok := tc.expected["username"]; ok && *username != expected {
				t.Errorf("Expected username %v, got %v", expected, *username)
			}
			if expected, ok := tc.expected["admin"]; ok && *isAdmin != expected {
				t.Errorf("Expected admin %v, got %v", expected, *isAdmin)
			}
			if expected, ok := tc.expected["adminKey"]; ok && *adminKey != expected {
				t.Errorf("Expected adminKey %v, got %v", expected, *adminKey)
			}
			if expected, ok := tc.expected["e2e"]; ok && *useE2E != expected {
				t.Errorf("Expected e2e %v, got %v", expected, *useE2E)
			}
			if expected, ok := tc.expected["keystorePassphrase"]; ok && *keystorePassphrase != expected {
				t.Errorf("Expected keystorePassphrase %v, got %v", expected, *keystorePassphrase)
			}
		})
	}
}

func TestEnvironmentVariableHandling(t *testing.T) {
	// Test environment variable handling
	originalEnv := os.Getenv("MARCHAT_MAX_FILE_BYTES")
	defer func() {
		if originalEnv != "" {
			os.Setenv("MARCHAT_MAX_FILE_BYTES", originalEnv)
		} else {
			os.Unsetenv("MARCHAT_MAX_FILE_BYTES")
		}
	}()

	// Test setting environment variable
	os.Setenv("MARCHAT_MAX_FILE_BYTES", "2048")
	if os.Getenv("MARCHAT_MAX_FILE_BYTES") != "2048" {
		t.Error("Failed to set environment variable")
	}

	// Test unsetting environment variable
	os.Unsetenv("MARCHAT_MAX_FILE_BYTES")
	if os.Getenv("MARCHAT_MAX_FILE_BYTES") != "" {
		t.Error("Failed to unset environment variable")
	}
}

func TestDefaultValues(t *testing.T) {
	// Test that default values are set correctly
	if maxMessages != 100 {
		t.Errorf("Expected maxMessages to be 100, got %d", maxMessages)
	}
	if maxUsersDisplay != 20 {
		t.Errorf("Expected maxUsersDisplay to be 20, got %d", maxUsersDisplay)
	}
	if userListWidth != 18 {
		t.Errorf("Expected userListWidth to be 18, got %d", userListWidth)
	}
	if pingPeriod != 50*time.Second {
		t.Errorf("Expected pingPeriod to be 50s, got %v", pingPeriod)
	}
	if reconnectMaxDelay != 30*time.Second {
		t.Errorf("Expected reconnectMaxDelay to be 30s, got %v", reconnectMaxDelay)
	}
}

func TestMainPackageStructure(t *testing.T) {
	// Test that key variables and types are defined
	if mentionRegex == nil {
		t.Error("mentionRegex should be initialized")
	}
	if urlRegex == nil {
		t.Error("urlRegex should be initialized")
	}

	// Test that keyMap type exists and has expected methods
	km := newKeyMap()
	if km.ShortHelp() == nil {
		t.Error("keyMap.ShortHelp() should return a non-nil slice")
	}
	if km.FullHelp() == nil {
		t.Error("keyMap.FullHelp() should return a non-nil slice")
	}
	if km.GetCommandHelp(false, false) == nil {
		t.Error("keyMap.GetCommandHelp() should return a non-nil slice")
	}
}

func TestErrorHandling(t *testing.T) {
	// Test error handling functions
	if allFlagsProvided("", "", false, "", false, "") {
		t.Error("allFlagsProvided should return false for empty required flags")
	}
	if allFlagsProvided("ws://localhost", "", false, "", false, "") {
		t.Error("allFlagsProvided should return false for missing username")
	}
	if allFlagsProvided("ws://localhost", "user", true, "", false, "") {
		t.Error("allFlagsProvided should return false for admin without admin key")
	}
	if allFlagsProvided("ws://localhost", "user", false, "", true, "") {
		t.Error("allFlagsProvided should return false for e2e without passphrase")
	}
	if !allFlagsProvided("ws://localhost", "user", false, "", false, "") {
		t.Error("allFlagsProvided should return true for valid basic flags")
	}
	if !allFlagsProvided("ws://localhost", "user", true, "key", false, "") {
		t.Error("allFlagsProvided should return true for valid admin flags")
	}
	if !allFlagsProvided("ws://localhost", "user", false, "", true, "pass") {
		t.Error("allFlagsProvided should return true for valid e2e flags")
	}
}

func TestBasicFunctionality(t *testing.T) {
	// Test basic functionality without actually running the main function

	// Test sortMessagesByTimestamp function
	messages := []shared.Message{
		{
			Sender:    "user1",
			Content:   "second message",
			CreatedAt: time.Now().Add(1 * time.Second),
		},
		{
			Sender:    "user2",
			Content:   "first message",
			CreatedAt: time.Now(),
		},
		{
			Sender:    "user1",
			Content:   "third message",
			CreatedAt: time.Now().Add(2 * time.Second),
		},
	}

	// Sort messages
	sortMessagesByTimestamp(messages)

	// Verify sorting
	if messages[0].Content != "first message" {
		t.Errorf("Expected first message to be 'first message', got '%s'", messages[0].Content)
	}
	if messages[1].Content != "second message" {
		t.Errorf("Expected second message to be 'second message', got '%s'", messages[1].Content)
	}
	if messages[2].Content != "third message" {
		t.Errorf("Expected third message to be 'third message', got '%s'", messages[2].Content)
	}
}

func TestNotificationManager(t *testing.T) {
	// Test NotificationManager functionality
	config := DefaultNotificationConfig()
	nm := NewNotificationManager(config)
	if nm == nil {
		t.Fatal("NewNotificationManager() should not return nil")
	}

	// Test initial state
	cfg := nm.GetConfig()
	if !cfg.BellEnabled {
		t.Error("NotificationManager should have bell enabled by default")
	}

	// Test mode setting
	nm.SetMode(NotificationModeNone)
	cfg = nm.GetConfig()
	if cfg.Mode != NotificationModeNone {
		t.Error("Mode should be None after SetMode(NotificationModeNone)")
	}

	nm.SetMode(NotificationModeBell)
	cfg = nm.GetConfig()
	if cfg.Mode != NotificationModeBell {
		t.Error("Mode should be Bell after SetMode(NotificationModeBell)")
	}

	// Test bell toggle
	enabled := nm.ToggleBell()
	if enabled {
		t.Error("Bell should be disabled after first toggle")
	}
	enabled = nm.ToggleBell()
	if !enabled {
		t.Error("Bell should be enabled after second toggle")
	}

	// Test desktop toggle
	_ = nm.ToggleDesktop()

	// Test quiet hours
	nm.SetQuietHours(true, 22, 8)
	cfg = nm.GetConfig()
	if !cfg.QuietHoursEnabled {
		t.Error("Quiet hours should be enabled")
	}
	if cfg.QuietHoursStart != 22 || cfg.QuietHoursEnd != 8 {
		t.Error("Quiet hours should be set to 22:00-08:00")
	}

	// Test focus mode
	nm.EnableFocusMode(30 * time.Minute)
	cfg = nm.GetConfig()
	if !cfg.FocusModeEnabled {
		t.Error("Focus mode should be enabled")
	}

	nm.DisableFocusMode()
	cfg = nm.GetConfig()
	if cfg.FocusModeEnabled {
		t.Error("Focus mode should be disabled")
	}

	// Test Notify (should not panic)
	nm.Notify("TestUser", "Test message", NotificationLevelInfo)
	nm.Notify("TestUser", "Test mention", NotificationLevelMention)
	nm.Notify("TestUser", "Test urgent", NotificationLevelUrgent)

	// Test desktop support detection
	_ = nm.IsDesktopSupported()
}

func TestThemeStyles(t *testing.T) {
	// Test theme styles functions
	baseStyles := baseThemeStyles()
	// Test that we can render something with the styles (basic functionality test)
	testContent := "test"
	_ = baseStyles.User.Render(testContent)

	// Test different themes
	themes := []string{"system", "patriot", "retro", "modern", "unknown"}
	for _, theme := range themes {
		styles := getThemeStyles(theme)
		// Test that we can render something with the styles
		_ = styles.User.Render(testContent)
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test isTermux function
	isTermux := isTermux()
	// We can't easily test the actual environment, but we can test it returns a boolean
	_ = isTermux

	// Test checkClipboardSupport function
	support := checkClipboardSupport()
	// We can't easily test clipboard support, but we can test it returns a boolean
	_ = support
}

func TestRegexPatterns(t *testing.T) {
	// Test mention regex
	testMentions := []string{
		"@user1",
		"Hello @user2",
		"@user3 and @user4",
		"no mention here",
		"@user5@user6", // Should match both
	}

	for _, test := range testMentions {
		matches := mentionRegex.FindAllString(test, -1)
		if strings.Contains(test, "@") && len(matches) == 0 {
			t.Errorf("Expected to find mentions in '%s', but found none", test)
		}
	}

	// Test URL regex
	testURLs := []string{
		"https://example.com",
		"http://test.org",
		"www.example.com",
		"no url here",
		"Check out https://github.com",
	}

	for _, test := range testURLs {
		matches := urlRegex.FindAllString(test, -1)
		if strings.Contains(test, "http") && len(matches) == 0 {
			t.Errorf("Expected to find URLs in '%s', but found none", test)
		}
	}
}

func TestRenderFunctions(t *testing.T) {
	// Test renderEmojis function
	emojis := map[string]string{
		":)": "😊",
		":(": "🙁",
		":D": "😃",
		"<3": "❤️",
		":P": "😛",
	}

	for short, expected := range emojis {
		result := renderEmojis(short)
		if result != expected {
			t.Errorf("Expected %s to render as %s, got %s", short, expected, result)
		}
	}

	// Test renderHyperlinks function
	styles := baseThemeStyles()
	content := "Check out https://example.com"
	result := renderHyperlinks(content, styles)
	if !strings.Contains(result, "https://example.com") {
		t.Error("renderHyperlinks should preserve URLs")
	}

	// Test renderCodeBlocks function
	codeContent := "```go\nfunc main() {}\n```"
	result = renderCodeBlocks(codeContent)
	// The function should return something (either the original or highlighted version)
	if result == "" {
		t.Error("renderCodeBlocks should return non-empty result")
	}
}

func TestModelInitialization(t *testing.T) {
	// Test that we can create a basic model structure
	// This is a simplified test since we can't easily test the full model without dependencies

	// Test keyMap creation
	keys := newKeyMap()
	if keys.Send.Keys()[0] == "" {
		t.Error("keyMap should have non-empty key bindings")
	}

	// Test that keyMap methods work
	shortHelp := keys.ShortHelp()
	if len(shortHelp) == 0 {
		t.Error("ShortHelp should return at least one key binding")
	}

	fullHelp := keys.FullHelp()
	if len(fullHelp) == 0 {
		t.Error("FullHelp should return at least one row of key bindings")
	}

	commandHelp := keys.GetCommandHelp(false, false)
	if len(commandHelp) == 0 {
		t.Error("GetCommandHelp should return at least one row of command help")
	}

	// Test admin command help
	adminCommandHelp := keys.GetCommandHelp(true, false)
	if len(adminCommandHelp) == 0 {
		t.Error("GetCommandHelp should return admin commands for admin users")
	}
}

func TestSortMessagesByTimestamp(t *testing.T) {
	// Test the secondary and tertiary sorting logic that was untested
	now := time.Now()

	// Create messages with identical timestamps to test secondary/tertiary sorting
	messages := []shared.Message{
		{
			Sender:    "user2",
			Content:   "message b",
			CreatedAt: now,
		},
		{
			Sender:    "user1",
			Content:   "message a",
			CreatedAt: now,
		},
		{
			Sender:    "user1",
			Content:   "message c",
			CreatedAt: now,
		},
	}

	// Sort messages
	sortMessagesByTimestamp(messages)

	// Verify secondary sort by sender
	if messages[0].Sender != "user1" {
		t.Errorf("Expected first message to be from user1, got %s", messages[0].Sender)
	}
	if messages[0].Content != "message a" {
		t.Errorf("Expected first message content to be 'message a', got '%s'", messages[0].Content)
	}

	// Verify tertiary sort by content for same sender
	if messages[1].Sender != "user1" {
		t.Errorf("Expected second message to be from user1, got %s", messages[1].Sender)
	}
	if messages[1].Content != "message c" {
		t.Errorf("Expected second message content to be 'message c', got '%s'", messages[1].Content)
	}

	if messages[2].Sender != "user2" {
		t.Errorf("Expected third message to be from user2, got %s", messages[2].Sender)
	}
	if messages[2].Content != "message b" {
		t.Errorf("Expected third message content to be 'message b', got '%s'", messages[2].Content)
	}
}

func TestSafeClipboardOperation(t *testing.T) {
	// Test safeClipboardOperation with a simple operation
	err := safeClipboardOperation(func() error {
		return nil
	}, 1*time.Second)

	if err != nil {
		t.Errorf("Expected no error for successful operation, got %v", err)
	}

	// Test timeout behavior
	err = safeClipboardOperation(func() error {
		time.Sleep(2 * time.Second)
		return nil
	}, 100*time.Millisecond)

	if err == nil {
		t.Error("Expected timeout error for slow operation")
	}
}

func TestVerifyKeystoreUnlocked(t *testing.T) {
	// Test with nil keystore
	err := verifyKeystoreUnlocked(nil)
	if err == nil {
		t.Error("Expected error for nil keystore")
	}
	if !strings.Contains(err.Error(), "keystore is nil") {
		t.Errorf("Expected 'keystore is nil' error, got: %v", err)
	}

	// Note: We can't easily test the successful case without a real keystore
	// This would require mocking or integration testing
}

func TestValidateEncryptionRoundtrip(t *testing.T) {
	// Note: We can't test this function easily with a nil keystore because
	// it will panic when trying to call methods on the keystore.
	// This function requires a properly initialized keystore to test.
	// Full testing would require a real keystore with proper initialization
	// This is more of an integration test concern.

	// We can only test that the function exists and has the right signature
	_ = validateEncryptionRoundtrip
}

func TestDebugWebSocketWrite(t *testing.T) {
	// Test JSON marshaling error handling
	// We can't easily test the actual WebSocket write without a real connection
	// but we can test the JSON marshaling logic

	// Create a message that should marshal successfully
	msg := map[string]interface{}{
		"content": "test message",
		"sender":  "testuser",
	}

	// We can't test the actual WebSocket write without a real connection,
	// but we can verify the function exists and handles the message structure
	_ = msg
}

func TestInitFunction(t *testing.T) {
	// Test that the init function runs without panicking
	// The init function sets up regex patterns and logging
	// We can verify the patterns are initialized

	if mentionRegex == nil {
		t.Error("mentionRegex should be initialized in init()")
	}
	if urlRegex == nil {
		t.Error("urlRegex should be initialized in init()")
	}

	// Test that the regex patterns work
	testMention := "@user"
	if !mentionRegex.MatchString(testMention) {
		t.Error("mentionRegex should match @user pattern")
	}

	testURL := "https://example.com"
	if !urlRegex.MatchString(testURL) {
		t.Error("urlRegex should match URL pattern")
	}
}

func TestLogOutputHandling(t *testing.T) {
	// Test that the log output handling in init() works
	// This is hard to test directly since it modifies global log state
	// but we can verify the log package is working

	// Test that we can write to log
	log.Printf("Test log message")

	// The init function tries to open marchat-debug.log
	// If it fails, it falls back to stderr
	// We can't easily test this without file system manipulation
}

func TestAllFlagsProvided(t *testing.T) {
	// Test allFlagsProvided function thoroughly
	testCases := []struct {
		name                string
		serverURL, username string
		isAdmin             bool
		adminKey            string
		useE2E              bool
		keystorePassphrase  string
		expected            bool
	}{
		{
			name:               "all provided",
			serverURL:          "ws://localhost:8080",
			username:           "user",
			isAdmin:            false,
			adminKey:           "",
			useE2E:             false,
			keystorePassphrase: "",
			expected:           true,
		},
		{
			name:               "missing serverURL",
			serverURL:          "",
			username:           "user",
			isAdmin:            false,
			adminKey:           "",
			useE2E:             false,
			keystorePassphrase: "",
			expected:           false,
		},
		{
			name:               "missing username",
			serverURL:          "ws://localhost:8080",
			username:           "",
			isAdmin:            false,
			adminKey:           "",
			useE2E:             false,
			keystorePassphrase: "",
			expected:           false,
		},
		{
			name:               "admin without key",
			serverURL:          "ws://localhost:8080",
			username:           "user",
			isAdmin:            true,
			adminKey:           "",
			useE2E:             false,
			keystorePassphrase: "",
			expected:           false,
		},
		{
			name:               "admin with key",
			serverURL:          "ws://localhost:8080",
			username:           "user",
			isAdmin:            true,
			adminKey:           "secret",
			useE2E:             false,
			keystorePassphrase: "",
			expected:           true,
		},
		{
			name:               "e2e without passphrase",
			serverURL:          "ws://localhost:8080",
			username:           "user",
			isAdmin:            false,
			adminKey:           "",
			useE2E:             true,
			keystorePassphrase: "",
			expected:           false,
		},
		{
			name:               "e2e with passphrase",
			serverURL:          "ws://localhost:8080",
			username:           "user",
			isAdmin:            false,
			adminKey:           "",
			useE2E:             true,
			keystorePassphrase: "pass",
			expected:           true,
		},
		{
			name:               "both admin and e2e",
			serverURL:          "ws://localhost:8080",
			username:           "user",
			isAdmin:            true,
			adminKey:           "secret",
			useE2E:             true,
			keystorePassphrase: "pass",
			expected:           true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := allFlagsProvided(tc.serverURL, tc.username, tc.isAdmin, tc.adminKey, tc.useE2E, tc.keystorePassphrase)
			if result != tc.expected {
				t.Errorf("Expected %v for %s, got %v", tc.expected, tc.name, result)
			}
		})
	}
}

func TestValidateFlags(t *testing.T) {
	// Test validateFlags function
	testCases := []struct {
		name               string
		isAdmin            bool
		adminKey           string
		useE2E             bool
		keystorePassphrase string
		expectError        bool
	}{
		{
			name:               "valid basic",
			isAdmin:            false,
			adminKey:           "",
			useE2E:             false,
			keystorePassphrase: "",
			expectError:        false,
		},
		{
			name:               "admin without key",
			isAdmin:            true,
			adminKey:           "",
			useE2E:             false,
			keystorePassphrase: "",
			expectError:        true,
		},
		{
			name:               "admin with key",
			isAdmin:            true,
			adminKey:           "secret",
			useE2E:             false,
			keystorePassphrase: "",
			expectError:        false,
		},
		{
			name:               "e2e without passphrase",
			isAdmin:            false,
			adminKey:           "",
			useE2E:             true,
			keystorePassphrase: "",
			expectError:        true,
		},
		{
			name:               "e2e with passphrase",
			isAdmin:            false,
			adminKey:           "",
			useE2E:             true,
			keystorePassphrase: "pass",
			expectError:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFlags(tc.isAdmin, tc.adminKey, tc.useE2E, tc.keystorePassphrase)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tc.name)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tc.name, err)
			}
		})
	}
}

func TestLogOutputHandlingDetailed(t *testing.T) {
	// Test the specific else branch in init() where log.SetOutput(os.Stderr) is called
	// This is hard to test directly since it modifies global state, but we can test the logic

	// Test that we can handle file creation errors
	tempDir := t.TempDir()

	// Try to create a file in a non-existent directory to trigger the else branch
	badPath := filepath.Join(tempDir, "nonexistent", "marchat-debug.log")

	// This should trigger the else branch in init()
	f, err := os.OpenFile(badPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// This simulates the else branch in init()
		log.SetOutput(os.Stderr)
		// Test that logging still works
		log.Printf("Test message to stderr")
	} else {
		f.Close()
	}
}

func TestRenderMessages(t *testing.T) {
	// Test the renderMessages function
	now := time.Now()
	messages := []shared.Message{
		{
			Sender:    "user1",
			Content:   "Hello world",
			CreatedAt: now,
			Type:      shared.TextMessage,
		},
		{
			Sender:    "user2",
			Content:   "Hi there",
			CreatedAt: now.Add(1 * time.Second),
			Type:      shared.TextMessage,
		},
	}

	styles := baseThemeStyles()
	username := "user1"
	users := []string{"user1", "user2"}
	width := 80
	twentyFourHour := true

	// Test basic rendering
	result := renderMessages(messages, styles, username, users, width, twentyFourHour)
	if result == "" {
		t.Error("renderMessages should return non-empty result")
	}

	// Test with file message
	fileMessages := []shared.Message{
		{
			Sender:    "user1",
			Content:   "Here's a file",
			CreatedAt: now,
			Type:      shared.FileMessageType,
			File: &shared.FileMeta{
				Filename: "test.txt",
				Size:     1024,
			},
		},
	}

	fileResult := renderMessages(fileMessages, styles, username, users, width, twentyFourHour)
	if !strings.Contains(fileResult, "test.txt") {
		t.Error("renderMessages should include filename for file messages")
	}

	// Test with mentions
	mentionMessages := []shared.Message{
		{
			Sender:    "user2",
			Content:   "Hello @user1",
			CreatedAt: now,
			Type:      shared.TextMessage,
		},
	}

	mentionResult := renderMessages(mentionMessages, styles, username, users, width, twentyFourHour)
	if !strings.Contains(mentionResult, "@user1") {
		t.Error("renderMessages should preserve mentions")
	}

	// Test with hyperlinks
	linkMessages := []shared.Message{
		{
			Sender:    "user1",
			Content:   "Check out https://example.com",
			CreatedAt: now,
			Type:      shared.TextMessage,
		},
	}

	linkResult := renderMessages(linkMessages, styles, username, users, width, twentyFourHour)
	if !strings.Contains(linkResult, "https://example.com") {
		t.Error("renderMessages should preserve URLs")
	}

	// Test 12-hour format
	twelveHourResult := renderMessages(messages, styles, username, users, width, false)
	if twelveHourResult == "" {
		t.Error("renderMessages should work with 12-hour format")
	}

	// Test message limit
	tooManyMessages := make([]shared.Message, maxMessages+10)
	for i := 0; i < len(tooManyMessages); i++ {
		tooManyMessages[i] = shared.Message{
			Sender:    "user1",
			Content:   fmt.Sprintf("Message %d", i),
			CreatedAt: now.Add(time.Duration(i) * time.Second),
			Type:      shared.TextMessage,
		}
	}

	limitedResult := renderMessages(tooManyMessages, styles, username, users, width, twentyFourHour)
	if limitedResult == "" {
		t.Error("renderMessages should handle message limit")
	}
}

func TestRenderUserList(t *testing.T) {
	// Test the renderUserList function
	users := []string{"user1", "user2", "user3", "currentuser"}
	me := "currentuser"
	styles := baseThemeStyles()
	width := 18
	isAdmin := true
	selectedUserIndex := 1 // Select user2

	result := renderUserList(users, me, styles, width, isAdmin, selectedUserIndex)
	if result == "" {
		t.Error("renderUserList should return non-empty result")
	}

	if !strings.Contains(result, "Users") {
		t.Error("renderUserList should include title")
	}

	// Test with no admin
	nonAdminResult := renderUserList(users, me, styles, width, false, -1)
	if nonAdminResult == "" {
		t.Error("renderUserList should work for non-admin users")
	}

	// Test with many users (should show +X more)
	manyUsers := make([]string, maxUsersDisplay+5)
	for i := 0; i < len(manyUsers); i++ {
		manyUsers[i] = fmt.Sprintf("user%d", i)
	}

	manyUsersResult := renderUserList(manyUsers, "user0", styles, width, false, -1)
	if !strings.Contains(manyUsersResult, "more") {
		t.Error("renderUserList should show 'more' indicator for many users")
	}
}

func TestOpenURL(t *testing.T) {
	// Skip this test as openURL actually opens browsers
	// Testing this would require mocking exec.Command which is complex
	// The function is simple enough that integration testing is sufficient
	t.Skip("Skipping openURL test - function opens actual browser")
}

func TestDebugEncryptAndSend(t *testing.T) {
	// Test debugEncryptAndSend function
	// We can't test the full function without real keystore and websocket,
	// but we can test the nil keystore case

	// Test with nil keystore
	err := debugEncryptAndSend([]string{"user1"}, "test message", nil, nil, "sender")
	if err == nil {
		t.Error("Expected error for nil keystore")
	}
	if !strings.Contains(err.Error(), "keystore not initialized") {
		t.Errorf("Expected 'keystore not initialized' error, got: %v", err)
	}
}

func TestDebugWebSocketWriteDetailed(t *testing.T) {
	// Test debugWebSocketWrite function
	// We can't test the actual WebSocket write, but we can test JSON marshaling

	// Test with a message that should marshal successfully
	msg := shared.Message{
		Sender:    "testuser",
		Content:   "test message",
		CreatedAt: time.Now(),
		Type:      shared.TextMessage,
	}

	// Test JSON marshaling (part of the function logic)
	jsonData, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("JSON marshaling should succeed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON data should not be empty")
	}

	// Test parsing the JSON back
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Errorf("JSON unmarshaling should succeed: %v", err)
	}

	// Test content field extraction
	if content, exists := parsed["content"]; exists {
		if contentStr, ok := content.(string); ok {
			if len(contentStr) == 0 {
				t.Error("Content should not be empty")
			}
		}
	}
}
