package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// Test fileItem methods
func TestFileItem(t *testing.T) {
	item := fileItem{
		title:       "test.txt",
		description: "Test file",
		path:        "/path/to/test.txt",
		isDir:       false,
	}

	if item.Title() != "test.txt" {
		t.Errorf("Expected title 'test.txt', got '%s'", item.Title())
	}

	if item.Description() != "Test file" {
		t.Errorf("Expected description 'Test file', got '%s'", item.Description())
	}

	if item.FilterValue() != "test.txt" {
		t.Errorf("Expected filter value 'test.txt', got '%s'", item.FilterValue())
	}
}

// Test isAllowedFileType function
func TestIsAllowedFileType(t *testing.T) {
	testCases := []struct {
		filename string
		expected bool
	}{
		// Allowed file types
		{"test.txt", true},
		{"README.md", true},
		{"config.json", true},
		{"data.yaml", true},
		{"script.py", true},
		{"app.go", true},
		{"style.css", true},
		{"image.png", true},
		{"document.pdf", true},
		{"archive.zip", true},
		{"test.TXT", true}, // Case insensitive
		{"test.PNG", true},
		// Disallowed file types
		{"test.exe", false},
		{"script.sh", false},
		{"binary", false},
		{"noextension", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := isAllowedFileType(tc.filename)
		if result != tc.expected {
			t.Errorf("isAllowedFileType(%s) = %v, expected %v", tc.filename, result, tc.expected)
		}
	}
}

// Test formatFileSize function
func TestFormatFileSize(t *testing.T) {
	testCases := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tc := range testCases {
		result := formatFileSize(tc.bytes)
		if result != tc.expected {
			t.Errorf("formatFileSize(%d) = '%s', expected '%s'", tc.bytes, result, tc.expected)
		}
	}
}

// Test newFilePickerModel initialization
func TestNewFilePickerModel(t *testing.T) {
	onSend := func(path string) { /* callback for testing */ }
	onCancel := func() { /* callback for testing */ }

	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, onSend, onCancel)

	// Test initial state
	if model.state != stateSelectFile {
		t.Errorf("Expected initial state to be stateSelectFile, got %d", model.state)
	}

	if model.selectedFile != "" {
		t.Errorf("Expected empty selectedFile, got '%s'", model.selectedFile)
	}

	if model.fileSize != 0 {
		t.Errorf("Expected fileSize to be 0, got %d", model.fileSize)
	}

	if model.err != nil {
		t.Errorf("Expected no error, got %v", model.err)
	}

	if !model.initialized {
		t.Error("Expected model to be initialized")
	}

	if model.width != 80 {
		t.Errorf("Expected width to be 80, got %d", model.width)
	}

	if model.height != 24 {
		t.Errorf("Expected height to be 24, got %d", model.height)
	}

	if model.onSend == nil {
		t.Error("Expected onSend callback to be set")
	}

	if model.onCancel == nil {
		t.Error("Expected onCancel callback to be set")
	}
}

// Test filePickerModel Init method
func TestFilePickerModelInit(t *testing.T) {
	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})

	cmd := model.Init()
	if cmd != nil {
		t.Error("Expected Init to return nil command")
	}
}

// Test filePickerModel Update with clearErrorMsg
func TestFilePickerModelClearError(t *testing.T) {
	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})

	// Set an error
	model.err = fmt.Errorf("test error")

	// Send clear error message
	updatedModel, _ := model.Update(clearErrorMsg{})
	fpModel := updatedModel.(filePickerModel)

	if fpModel.err != nil {
		t.Errorf("Expected error to be cleared, got %v", fpModel.err)
	}
}

// Test filePickerModel Update with ESC key in select state
func TestFilePickerModelEscInSelectState(t *testing.T) {
	var onCancelCalled bool
	onCancel := func() { onCancelCalled = true }

	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, onCancel)

	// Send ESC key
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := model.Update(escMsg)
	_ = updatedModel.(filePickerModel)

	if !onCancelCalled {
		t.Error("Expected onCancel to be called")
	}

	// Test Ctrl+C
	onCancelCalled = false
	model = newFilePickerModel(styles, 80, 24, func(string) {}, onCancel)
	ctrlCMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, _ = model.Update(ctrlCMsg)
	_ = updatedModel.(filePickerModel)

	if !onCancelCalled {
		t.Error("Expected onCancel to be called with Ctrl+C")
	}
}

// Test filePickerModel Update with ESC key in confirm state
func TestFilePickerModelEscInConfirmState(t *testing.T) {
	var onCancelCalled bool
	onCancel := func() { onCancelCalled = true }

	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, onCancel)
	model.state = stateConfirmFileSend
	model.selectedFile = "/path/to/file.txt"

	// Send ESC key
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := model.Update(escMsg)
	_ = updatedModel.(filePickerModel)

	if !onCancelCalled {
		t.Error("Expected onCancel to be called in confirm state")
	}
}

// Test filePickerModel Update with Enter key in confirm state
func TestFilePickerModelEnterInConfirmState(t *testing.T) {
	var onSendCalled bool
	var sentPath string
	onSend := func(path string) {
		onSendCalled = true
		sentPath = path
	}
	var onCancelCalled bool
	onCancel := func() { onCancelCalled = true }

	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, onSend, onCancel)
	model.state = stateConfirmFileSend
	model.selectedFile = "/path/to/file.txt"

	// Send Enter key
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	_ = updatedModel.(filePickerModel)

	if !onSendCalled {
		t.Error("Expected onSend to be called")
	}

	if sentPath != "/path/to/file.txt" {
		t.Errorf("Expected sent path to be '/path/to/file.txt', got '%s'", sentPath)
	}

	if !onCancelCalled {
		t.Error("Expected onCancel to be called after sending")
	}
}

// Test filePickerModel Update with 'r' key in confirm state
func TestFilePickerModelRInConfirmState(t *testing.T) {
	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})
	model.state = stateConfirmFileSend
	model.selectedFile = "/path/to/file.txt"
	model.fileSize = 1024

	// Send 'r' key
	rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updatedModel, _ := model.Update(rMsg)
	fpModel := updatedModel.(filePickerModel)

	if fpModel.state != stateSelectFile {
		t.Errorf("Expected state to be stateSelectFile, got %d", fpModel.state)
	}

	if fpModel.selectedFile != "" {
		t.Errorf("Expected selectedFile to be cleared, got '%s'", fpModel.selectedFile)
	}

	if fpModel.fileSize != 0 {
		t.Errorf("Expected fileSize to be 0, got %d", fpModel.fileSize)
	}
}

// Test filePickerModel View in select state
func TestFilePickerModelViewSelectState(t *testing.T) {
	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})

	view := model.View()

	// Check for title
	if !contains(view, "Select File to Send") {
		t.Error("Expected view to contain 'Select File to Send'")
	}

	// Check for help text
	if !contains(view, "Navigate with arrow keys") {
		t.Error("Expected view to contain navigation help")
	}
}

// Test filePickerModel View with error
func TestFilePickerModelViewWithError(t *testing.T) {
	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})
	model.err = fmt.Errorf("test error")

	view := model.View()

	// Check for error message
	if !contains(view, "[ERROR] test error") {
		t.Error("Expected view to contain error message")
	}
}

// Test filePickerModel View in confirm state
func TestFilePickerModelViewConfirmState(t *testing.T) {
	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})
	model.state = stateConfirmFileSend
	model.selectedFile = "/path/to/test.txt"
	model.fileSize = 1024

	view := model.View()

	// Check for confirm title
	if !contains(view, "Confirm File Send") {
		t.Error("Expected view to contain 'Confirm File Send'")
	}

	// Check for file information
	if !contains(view, "test.txt") {
		t.Error("Expected view to contain filename")
	}

	if !contains(view, "/path/to/test.txt") {
		t.Error("Expected view to contain file path")
	}

	if !contains(view, "1.0 KB") {
		t.Error("Expected view to contain formatted file size")
	}

	// Check for help text
	if !contains(view, "Press Enter to send") {
		t.Error("Expected view to contain send instructions")
	}
}

// Test filePickerModel View in unknown state
func TestFilePickerModelViewUnknownState(t *testing.T) {
	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})
	model.state = 999 // Unknown state

	view := model.View()

	if view != "Unknown state" {
		t.Errorf("Expected 'Unknown state', got '%s'", view)
	}
}

// Test createFileListItems with non-existent directory
func TestCreateFileListItemsNonExistentDir(t *testing.T) {
	items := createFileListItems("/non/existent/directory")

	// Should return empty list or just parent directory
	if len(items) > 1 {
		t.Errorf("Expected empty or minimal list for non-existent directory, got %d items", len(items))
	}
}

// Test createFileListItems with current directory
func TestCreateFileListItemsCurrentDir(t *testing.T) {
	// Create a temporary directory with test files
	tempDir := t.TempDir()

	// Create test files
	testFiles := []string{"test.txt", "README.md", "script.py", "image.png"}
	for _, file := range testFiles {
		f, err := os.Create(filepath.Join(tempDir, file))
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
		f.Close()
	}

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	items := createFileListItems(tempDir)

	// Should have parent directory, subdirectory, and test files
	if len(items) < len(testFiles)+2 { // +2 for parent and subdir
		t.Errorf("Expected at least %d items, got %d", len(testFiles)+2, len(items))
	}

	// Check for parent directory
	foundParent := false
	for _, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.title == ".. (Parent Directory)" {
				foundParent = true
				break
			}
		}
	}
	if !foundParent {
		t.Error("Expected to find parent directory item")
	}

	// Check for subdirectory
	foundSubdir := false
	for _, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.title == "[DIR] subdir" {
				foundSubdir = true
				break
			}
		}
	}
	if !foundSubdir {
		t.Error("Expected to find subdirectory item")
	}
}

// Test createFileListItems with root directory
func TestCreateFileListItemsRootDir(t *testing.T) {
	items := createFileListItems("/")

	// Should not have parent directory for root
	for _, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.title == ".. (Parent Directory)" {
				t.Error("Should not have parent directory for root")
				break
			}
		}
	}
}

// Test createFileListItems with current directory (.)
func TestCreateFileListItemsCurrentDirDot(t *testing.T) {
	items := createFileListItems(".")

	// Should not have parent directory for current directory
	for _, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.title == ".. (Parent Directory)" {
				t.Error("Should not have parent directory for current directory")
				break
			}
		}
	}
}

// Test filePickerModel Update with Enter on directory
func TestFilePickerModelEnterOnDirectory(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a file in subdirectory
	testFile := filepath.Join(subDir, "test.txt")
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	f.Close()

	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})

	// Set the list to show the temp directory
	items := createFileListItems(tempDir)
	model.list.SetItems(items)

	// Find the subdirectory item
	var subdirItem fileItem
	for _, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.title == "[DIR] subdir" {
				subdirItem = fileItem
				break
			}
		}
	}

	if subdirItem.path == "" {
		t.Fatal("Could not find subdirectory item")
	}

	// Select the subdirectory item
	model.list.SetItems(items)
	for i, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.path == subdirItem.path {
				model.list.Select(i)
				break
			}
		}
	}

	// Send Enter key
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	fpModel := updatedModel.(filePickerModel)

	// Should still be in select state and list should be updated
	if fpModel.state != stateSelectFile {
		t.Errorf("Expected state to remain stateSelectFile, got %d", fpModel.state)
	}

	// Check if list was updated with subdirectory contents
	listItems := fpModel.list.Items()
	foundTestFile := false
	for _, item := range listItems {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.title == "[FILE] test.txt" {
				foundTestFile = true
				break
			}
		}
	}
	if !foundTestFile {
		t.Error("Expected to find test.txt in subdirectory listing")
	}
}

// Test filePickerModel Update with Enter on file (size check)
func TestFilePickerModelEnterOnFile(t *testing.T) {
	// Create a temporary file
	tempFile := filepath.Join(t.TempDir(), "test.txt")
	f, err := os.Create(tempFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	_, err = f.WriteString("test content")
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	f.Close()

	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})

	// Set the list to show the temp directory
	items := createFileListItems(filepath.Dir(tempFile))
	model.list.SetItems(items)

	// Find the test file item
	var testFileItem fileItem
	for _, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.path == tempFile {
				testFileItem = fileItem
				break
			}
		}
	}

	if testFileItem.path == "" {
		t.Fatal("Could not find test file item")
	}

	// Select the test file item
	for i, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.path == tempFile {
				model.list.Select(i)
				break
			}
		}
	}

	// Send Enter key
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	fpModel := updatedModel.(filePickerModel)

	// Should move to confirm state
	if fpModel.state != stateConfirmFileSend {
		t.Errorf("Expected state to be stateConfirmFileSend, got %d", fpModel.state)
	}

	if fpModel.selectedFile != tempFile {
		t.Errorf("Expected selectedFile to be '%s', got '%s'", tempFile, fpModel.selectedFile)
	}

	if fpModel.fileSize <= 0 {
		t.Errorf("Expected fileSize to be > 0, got %d", fpModel.fileSize)
	}
}

// Test filePickerModel Update with Enter on non-existent file
func TestFilePickerModelEnterOnNonExistentFile(t *testing.T) {
	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})

	// Create a fake file item
	fakeItem := fileItem{
		title: "[FILE] nonexistent.txt",
		path:  "/nonexistent/path/file.txt",
		isDir: false,
	}

	// Set the list with the fake item
	model.list.SetItems([]list.Item{fakeItem})
	model.list.Select(0)

	// Send Enter key
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	fpModel := updatedModel.(filePickerModel)

	// Should remain in select state with error
	if fpModel.state != stateSelectFile {
		t.Errorf("Expected state to remain stateSelectFile, got %d", fpModel.state)
	}

	if fpModel.err == nil {
		t.Error("Expected error for non-existent file")
	}

	if !contains(fpModel.err.Error(), "failed to read file info") {
		t.Errorf("Expected 'failed to read file info' error, got '%s'", fpModel.err.Error())
	}
}

// Test filePickerModel Update with file size limit
func TestFilePickerModelFileSizeLimit(t *testing.T) {
	// Set environment variable for file size limit
	os.Setenv("MARCHAT_MAX_FILE_BYTES", "100")
	defer os.Unsetenv("MARCHAT_MAX_FILE_BYTES")

	// Create a temporary file larger than the limit
	tempFile := filepath.Join(t.TempDir(), "large.txt")
	f, err := os.Create(tempFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	_, err = f.WriteString("This is a test file that is larger than 100 bytes to test the file size limit functionality. This text should make the file exceed 100 bytes.")
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	f.Close()

	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})

	// Set the list to show the temp directory
	items := createFileListItems(filepath.Dir(tempFile))
	model.list.SetItems(items)

	// Find the large file item
	var largeFileItem fileItem
	for _, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.path == tempFile {
				largeFileItem = fileItem
				break
			}
		}
	}

	if largeFileItem.path == "" {
		t.Fatal("Could not find large file item")
	}

	// Select the large file item
	for i, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.path == tempFile {
				model.list.Select(i)
				break
			}
		}
	}

	// Send Enter key
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	fpModel := updatedModel.(filePickerModel)

	// Should remain in select state with error
	if fpModel.state != stateSelectFile {
		t.Errorf("Expected state to remain stateSelectFile, got %d", fpModel.state)
	}

	if fpModel.err == nil {
		t.Error("Expected error for file too large")
		return // Exit early to avoid nil pointer dereference
	}

	if !contains(fpModel.err.Error(), "file too large") {
		t.Errorf("Expected 'file too large' error, got '%s'", fpModel.err.Error())
	}

	if !contains(fpModel.err.Error(), "max 100 bytes") {
		t.Errorf("Expected 'max 100 bytes' in error, got '%s'", fpModel.err.Error())
	}
}

// Test filePickerModel Update with MB file size limit
func TestFilePickerModelFileSizeLimitMB(t *testing.T) {
	// Set environment variable for file size limit in MB
	os.Setenv("MARCHAT_MAX_FILE_MB", "1")
	defer os.Unsetenv("MARCHAT_MAX_FILE_MB")

	// Create a temporary file larger than 1MB
	tempFile := filepath.Join(t.TempDir(), "large.txt")
	f, err := os.Create(tempFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Write 2MB of data
	data := make([]byte, 2*1024*1024)
	for i := range data {
		data[i] = 'a'
	}
	_, err = f.Write(data)
	if err != nil {
		t.Fatalf("Failed to write to test file: %v", err)
	}
	f.Close()

	styles := getMockThemeStyles()
	model := newFilePickerModel(styles, 80, 24, func(string) {}, func() {})

	// Set the list to show the temp directory
	items := createFileListItems(filepath.Dir(tempFile))
	model.list.SetItems(items)

	// Find the large file item
	var largeFileItem fileItem
	for _, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.path == tempFile {
				largeFileItem = fileItem
				break
			}
		}
	}

	if largeFileItem.path == "" {
		t.Fatal("Could not find large file item")
	}

	// Select the large file item
	for i, item := range items {
		if fileItem, ok := item.(fileItem); ok {
			if fileItem.path == tempFile {
				model.list.Select(i)
				break
			}
		}
	}

	// Send Enter key
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	fpModel := updatedModel.(filePickerModel)

	// Should remain in select state with error
	if fpModel.state != stateSelectFile {
		t.Errorf("Expected state to remain stateSelectFile, got %d", fpModel.state)
	}

	if fpModel.err == nil {
		t.Error("Expected error for file too large")
		return // Exit early to avoid nil pointer dereference
	}

	if !contains(fpModel.err.Error(), "file too large") {
		t.Errorf("Expected 'file too large' error, got '%s'", fpModel.err.Error())
	}

	if !contains(fpModel.err.Error(), "max 1MB") {
		t.Errorf("Expected 'max 1MB' in error, got '%s'", fpModel.err.Error())
	}
}

// Test clearErrorAfter function
func TestClearErrorAfter(t *testing.T) {
	cmd := clearErrorAfter(100 * time.Millisecond)

	if cmd == nil {
		t.Error("Expected clearErrorAfter to return a command")
	}

	// Test that the command produces a clearErrorMsg
	// This is a bit tricky to test directly, but we can verify the type
	// by checking that it's a tea.Cmd
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
