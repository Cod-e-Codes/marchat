package config

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Test NewConfigUI initialization
func TestNewConfigUI(t *testing.T) {
	model := NewConfigUI()

	// Test initial state
	if len(model.inputs) != 7 {
		t.Errorf("Expected 7 input fields, got %d", len(model.inputs))
	}

	if model.focusIndex != 0 {
		t.Errorf("Expected initial focus index to be 0, got %d", model.focusIndex)
	}

	if model.config == nil {
		t.Error("Expected config to be initialized")
	}

	if model.config.Theme != "system" {
		t.Errorf("Expected default theme to be 'system', got '%s'", model.config.Theme)
	}

	if model.config.TwentyFourHour != true {
		t.Error("Expected TwentyFourHour to be true")
	}

	if model.finished {
		t.Error("Expected model to not be finished initially")
	}

	if model.cancelled {
		t.Error("Expected model to not be cancelled initially")
	}

	// Test first input is focused
	if !model.inputs[0].Focused() {
		t.Error("Expected first input to be focused")
	}
}

// Test ConfigUIModel navigation
func TestConfigUIModelNavigation(t *testing.T) {
	model := NewConfigUI()

	// Test tab navigation
	tabMsg := tea.KeyPressMsg{Code: tea.KeyTab}
	updatedModel, _ := model.Update(tabMsg)
	csModel := updatedModel.(ConfigUIModel)

	if csModel.focusIndex != 1 {
		t.Errorf("Expected focus index to be 1 after tab, got %d", csModel.focusIndex)
	}

	// Test shift+tab navigation (backwards)
	shiftTabMsg := tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	updatedModel, _ = csModel.Update(shiftTabMsg)
	csModel = updatedModel.(ConfigUIModel)

	if csModel.focusIndex != 0 {
		t.Errorf("Expected focus index to be 0 after shift+tab, got %d", csModel.focusIndex)
	}

	// Test down arrow navigation
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	updatedModel, _ = csModel.Update(downMsg)
	csModel = updatedModel.(ConfigUIModel)

	if csModel.focusIndex != 1 {
		t.Errorf("Expected focus index to be 1 after down arrow, got %d", csModel.focusIndex)
	}

	// Test up arrow navigation
	upMsg := tea.KeyPressMsg{Code: tea.KeyUp}
	updatedModel, _ = csModel.Update(upMsg)
	csModel = updatedModel.(ConfigUIModel)

	if csModel.focusIndex != 0 {
		t.Errorf("Expected focus index to be 0 after up arrow, got %d", csModel.focusIndex)
	}
}

// Test ConfigUIModel conditional fields
func TestConfigUIModelConditionalFields(t *testing.T) {
	model := NewConfigUI()

	// Test admin field shows admin key field
	adminInput := &model.inputs[adminField]
	adminInput.SetValue("y")
	model.updateConditionalFields()

	if !model.showAdminKey {
		t.Error("Expected admin key field to be shown when admin is 'y'")
	}

	// Test admin field hides admin key field
	adminInput.SetValue("n")
	model.updateConditionalFields()

	if model.showAdminKey {
		t.Error("Expected admin key field to be hidden when admin is 'n'")
	}

	// Test E2E field shows keystore passphrase field
	e2eInput := &model.inputs[e2eField]
	e2eInput.SetValue("y")
	model.updateConditionalFields()

	if !model.showE2EFields {
		t.Error("Expected E2E fields to be shown when E2E is 'y'")
	}

	// Test E2E field hides keystore passphrase field
	e2eInput.SetValue("n")
	model.updateConditionalFields()

	if model.showE2EFields {
		t.Error("Expected E2E fields to be hidden when E2E is 'n'")
	}
}

// Test ConfigUIModel validation
func TestConfigUIModelValidation(t *testing.T) {
	model := NewConfigUI()

	// Test empty server URL validation
	err := model.validateAndBuildConfig()
	if err == nil {
		t.Error("Expected validation error for empty server URL")
	}
	if err.Error() != "server URL is required" {
		t.Errorf("Expected 'server URL is required' error, got '%s'", err.Error())
	}

	// Test empty username validation
	model.inputs[serverURLField].SetValue("wss://example.com/ws")
	err = model.validateAndBuildConfig()
	if err == nil {
		t.Error("Expected validation error for empty username")
	}
	if err.Error() != "username is required" {
		t.Errorf("Expected 'username is required' error, got '%s'", err.Error())
	}

	// Test admin key validation
	model.inputs[usernameField].SetValue("testuser")
	model.inputs[adminField].SetValue("y")
	err = model.validateAndBuildConfig()
	if err == nil {
		t.Error("Expected validation error for missing admin key")
	}
	if err.Error() != "admin key is required for admin users" {
		t.Errorf("Expected admin key error, got '%s'", err.Error())
	}

	// Test keystore passphrase validation
	model.inputs[adminField].SetValue("n")
	model.inputs[adminKeyField].SetValue("")
	model.inputs[e2eField].SetValue("y")
	err = model.validateAndBuildConfig()
	if err == nil {
		t.Error("Expected validation error for missing keystore passphrase")
	}
	if err.Error() != "keystore passphrase is required for E2E encryption" {
		t.Errorf("Expected keystore passphrase error, got '%s'", err.Error())
	}
}

// Test ConfigUIModel successful validation
func TestConfigUIModelSuccessfulValidation(t *testing.T) {
	model := NewConfigUI()

	// Set valid values
	model.inputs[serverURLField].SetValue("wss://example.com/ws")
	model.inputs[usernameField].SetValue("testuser")
	model.inputs[adminField].SetValue("n")
	model.inputs[e2eField].SetValue("n")
	model.inputs[themeField].SetValue("modern")

	err := model.validateAndBuildConfig()
	if err != nil {
		t.Errorf("Expected no validation error, got %v", err)
	}

	// Test config was built correctly
	config := model.GetConfig()
	if config.ServerURL != "wss://example.com/ws" {
		t.Errorf("Expected ServerURL to be 'wss://example.com/ws', got '%s'", config.ServerURL)
	}
	if config.Username != "testuser" {
		t.Errorf("Expected Username to be 'testuser', got '%s'", config.Username)
	}
	if config.IsAdmin {
		t.Error("Expected IsAdmin to be false")
	}
	if config.UseE2E {
		t.Error("Expected UseE2E to be false")
	}
	if config.Theme != "modern" {
		t.Errorf("Expected Theme to be 'modern', got '%s'", config.Theme)
	}
}

// Test ConfigUIModel cancellation
func TestConfigUIModelCancellation(t *testing.T) {
	model := NewConfigUI()

	// Test ESC key cancellation
	escMsg := tea.KeyPressMsg{Code: tea.KeyEsc}
	updatedModel, _ := model.Update(escMsg)
	csModel := updatedModel.(ConfigUIModel)

	if !csModel.cancelled {
		t.Error("Expected model to be cancelled after ESC key")
	}

	// Test Ctrl+C cancellation
	model = NewConfigUI()
	ctrlCMsg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	updatedModel, _ = model.Update(ctrlCMsg)
	csModel = updatedModel.(ConfigUIModel)

	if !csModel.cancelled {
		t.Error("Expected model to be cancelled after Ctrl+C")
	}
}

// Test ConfigUIModel getNextValidFocus
func TestConfigUIModelGetNextValidFocus(t *testing.T) {
	model := NewConfigUI()

	// Test with no conditional fields shown
	nextFocus := model.getNextValidFocus(0, false)
	if nextFocus != 0 {
		t.Errorf("Expected next focus to be 0 (serverURLField is valid), got %d", nextFocus)
	}

	// Test with admin key field hidden - should skip adminKeyField
	nextFocus = model.getNextValidFocus(3, false) // adminKeyField when not shown
	if nextFocus != 4 {                           // should skip to e2eField
		t.Errorf("Expected next focus to be 4 (e2eField after skipping adminKeyField), got %d", nextFocus)
	}

	// Test with E2E fields hidden - should skip keystorePassField
	model.showAdminKey = false
	nextFocus = model.getNextValidFocus(5, false) // keystorePassField when not shown
	if nextFocus != 6 {                           // should skip to themeField
		t.Errorf("Expected next focus to be 6 (themeField after skipping keystorePassField), got %d", nextFocus)
	}

	// Test with both conditional fields shown - should return same index
	model.showAdminKey = true
	model.showE2EFields = true
	nextFocus = model.getNextValidFocus(3, false) // adminKeyField when shown
	if nextFocus != 3 {
		t.Errorf("Expected next focus to be 3 (adminKeyField when shown), got %d", nextFocus)
	}
}

// Test NewProfileSelectionModel
func TestNewProfileSelectionModel(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "Profile1", Username: "user1", ServerURL: "wss://server1.com"},
		{Name: "Profile2", Username: "user2", ServerURL: "wss://server2.com"},
	}

	model := NewProfileSelectionModel(profiles, true)

	if len(model.profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(model.profiles))
	}

	if model.cursor != 0 {
		t.Errorf("Expected cursor to be 0, got %d", model.cursor)
	}

	if !model.showNewOption {
		t.Error("Expected showNewOption to be true")
	}

	if model.operation != ProfileOpNone {
		t.Errorf("Expected operation to be ProfileOpNone, got %v", model.operation)
	}
}

// Test ProfileSelectionModel navigation
func TestProfileSelectionModelNavigation(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "Profile1", Username: "user1", ServerURL: "wss://server1.com"},
		{Name: "Profile2", Username: "user2", ServerURL: "wss://server2.com"},
	}

	model := NewProfileSelectionModel(profiles, false)

	// Test down navigation
	downMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	updatedModel, _ := model.Update(downMsg)
	psModel := updatedModel.(ProfileSelectionModel)

	if psModel.cursor != 1 {
		t.Errorf("Expected cursor to be 1 after down, got %d", psModel.cursor)
	}

	// Test up navigation
	upMsg := tea.KeyPressMsg{Code: tea.KeyUp}
	updatedModel, _ = psModel.Update(upMsg)
	psModel = updatedModel.(ProfileSelectionModel)

	if psModel.cursor != 0 {
		t.Errorf("Expected cursor to be 0 after up, got %d", psModel.cursor)
	}

	// Test 'j' key navigation
	jMsg := tea.KeyPressMsg{Code: 'j', Text: "j"}
	updatedModel, _ = psModel.Update(jMsg)
	psModel = updatedModel.(ProfileSelectionModel)

	if psModel.cursor != 1 {
		t.Errorf("Expected cursor to be 1 after 'j', got %d", psModel.cursor)
	}

	// Test 'k' key navigation
	kMsg := tea.KeyPressMsg{Code: 'k', Text: "k"}
	updatedModel, _ = psModel.Update(kMsg)
	psModel = updatedModel.(ProfileSelectionModel)

	if psModel.cursor != 0 {
		t.Errorf("Expected cursor to be 0 after 'k', got %d", psModel.cursor)
	}
}

// Test ProfileSelectionModel selection
func TestProfileSelectionModelSelection(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "Profile1", Username: "user1", ServerURL: "wss://server1.com"},
		{Name: "Profile2", Username: "user2", ServerURL: "wss://server2.com"},
	}

	model := NewProfileSelectionModel(profiles, false)

	// Test profile selection
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	psModel := updatedModel.(ProfileSelectionModel)

	if !psModel.selected {
		t.Error("Expected profile to be selected")
	}

	if psModel.choice != 0 {
		t.Errorf("Expected choice to be 0, got %d", psModel.choice)
	}

	// Test "Create New Profile" selection
	model = NewProfileSelectionModel(profiles, true)
	model.cursor = 2 // "Create New Profile" option
	updatedModel, _ = model.Update(enterMsg)
	psModel = updatedModel.(ProfileSelectionModel)

	if !psModel.selected {
		t.Error("Expected 'Create New Profile' to be selected")
	}

	if psModel.choice != 2 {
		t.Errorf("Expected choice to be 2 (Create New), got %d", psModel.choice)
	}

	if !psModel.IsCreateNew() {
		t.Error("Expected IsCreateNew to be true")
	}
}

// Test ProfileSelectionModel operations
func TestProfileSelectionModelOperations(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "Profile1", Username: "user1", ServerURL: "wss://server1.com"},
		{Name: "Profile2", Username: "user2", ServerURL: "wss://server2.com"},
	}

	model := NewProfileSelectionModel(profiles, false)

	// Test view operation
	viewMsg := tea.KeyPressMsg{Code: 'v', Text: "v"}
	updatedModel, _ := model.Update(viewMsg)
	psModel := updatedModel.(ProfileSelectionModel)

	if psModel.operation != ProfileOpView {
		t.Errorf("Expected operation to be ProfileOpView, got %v", psModel.operation)
	}

	// Test rename operation
	model = NewEnhancedProfileSelectionModel(profiles, false, nil)
	renameMsg := tea.KeyPressMsg{Code: 'r', Text: "r"}
	updatedModel, _ = model.Update(renameMsg)
	psModel = updatedModel.(ProfileSelectionModel)

	if psModel.operation != ProfileOpRename {
		t.Errorf("Expected operation to be ProfileOpRename, got %v", psModel.operation)
	}

	// Test delete operation
	model = NewProfileSelectionModel(profiles, false)
	deleteMsg := tea.KeyPressMsg{Code: 'd', Text: "d"}
	updatedModel, _ = model.Update(deleteMsg)
	psModel = updatedModel.(ProfileSelectionModel)

	if psModel.operation != ProfileOpDelete {
		t.Errorf("Expected operation to be ProfileOpDelete, got %v", psModel.operation)
	}
}

// Test ProfileSelectionModel delete protection
func TestProfileSelectionModelDeleteProtection(t *testing.T) {
	// Test with single profile (should not allow deletion)
	profiles := []ConnectionProfile{
		{Name: "OnlyProfile", Username: "user1", ServerURL: "wss://server1.com"},
	}

	model := NewProfileSelectionModel(profiles, false)
	deleteMsg := tea.KeyPressMsg{Code: 'd', Text: "d"}
	updatedModel, _ := model.Update(deleteMsg)
	psModel := updatedModel.(ProfileSelectionModel)

	if psModel.operation != ProfileOpNone {
		t.Errorf("Expected operation to remain ProfileOpNone for single profile, got %v", psModel.operation)
	}

	if psModel.message != "Cannot delete the only profile" {
		t.Errorf("Expected deletion protection message, got '%s'", psModel.message)
	}

	if psModel.messageType != "error" {
		t.Errorf("Expected message type to be 'error', got '%s'", psModel.messageType)
	}
}

// Test ProfileSelectionModel cancellation
func TestProfileSelectionModelCancellation(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "Profile1", Username: "user1", ServerURL: "wss://server1.com"},
	}

	model := NewProfileSelectionModel(profiles, false)

	// Test ESC key cancellation
	escMsg := tea.KeyPressMsg{Code: tea.KeyEsc}
	updatedModel, _ := model.Update(escMsg)
	psModel := updatedModel.(ProfileSelectionModel)

	if !psModel.cancelled {
		t.Error("Expected model to be cancelled after ESC key")
	}

	// Test Ctrl+C cancellation
	model = NewProfileSelectionModel(profiles, false)
	ctrlCMsg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	updatedModel, _ = model.Update(ctrlCMsg)
	psModel = updatedModel.(ProfileSelectionModel)

	if !psModel.cancelled {
		t.Error("Expected model to be cancelled after Ctrl+C")
	}
}

// Test NewSensitiveDataPrompt
func TestNewSensitiveDataPrompt(t *testing.T) {
	// Test with admin only
	model := NewSensitiveDataPrompt(true, false)

	if len(model.inputs) != 1 {
		t.Errorf("Expected 1 input field for admin only, got %d", len(model.inputs))
	}

	if !model.isAdmin {
		t.Error("Expected isAdmin to be true")
	}

	if model.useE2E {
		t.Error("Expected useE2E to be false")
	}

	// Test with E2E only
	model = NewSensitiveDataPrompt(false, true)

	if len(model.inputs) != 1 {
		t.Errorf("Expected 1 input field for E2E only, got %d", len(model.inputs))
	}

	if model.isAdmin {
		t.Error("Expected isAdmin to be false")
	}

	if !model.useE2E {
		t.Error("Expected useE2E to be true")
	}

	// Test with both
	model = NewSensitiveDataPrompt(true, true)

	if len(model.inputs) != 2 {
		t.Errorf("Expected 2 input fields for both admin and E2E, got %d", len(model.inputs))
	}

	if !model.isAdmin {
		t.Error("Expected isAdmin to be true")
	}

	if !model.useE2E {
		t.Error("Expected useE2E to be true")
	}
}

// Test SensitiveDataModel validation
func TestSensitiveDataModelValidation(t *testing.T) {
	// Test admin key validation
	model := NewSensitiveDataPrompt(true, false)
	model.inputs[0].SetValue("") // Empty admin key

	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	sdModel := updatedModel.(SensitiveDataModel)

	if sdModel.finished {
		t.Error("Expected model to not be finished with empty admin key")
	}

	if sdModel.errorMessage != "Admin key is required" {
		t.Errorf("Expected admin key error message, got '%s'", sdModel.errorMessage)
	}

	// Test keystore passphrase validation
	model = NewSensitiveDataPrompt(false, true)
	model.inputs[0].SetValue("") // Empty keystore passphrase

	updatedModel, _ = model.Update(enterMsg)
	sdModel = updatedModel.(SensitiveDataModel)

	if sdModel.finished {
		t.Error("Expected model to not be finished with empty keystore passphrase")
	}

	if sdModel.errorMessage != "Keystore passphrase is required" {
		t.Errorf("Expected keystore passphrase error message, got '%s'", sdModel.errorMessage)
	}
}

// Test SensitiveDataModel successful completion
func TestSensitiveDataModelSuccessfulCompletion(t *testing.T) {
	// Test admin only
	model := NewSensitiveDataPrompt(true, false)
	model.inputs[0].SetValue("adminkey123")

	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	sdModel := updatedModel.(SensitiveDataModel)

	if !sdModel.finished {
		t.Error("Expected model to be finished")
	}

	if sdModel.adminKey != "adminkey123" {
		t.Errorf("Expected admin key to be 'adminkey123', got '%s'", sdModel.adminKey)
	}

	// Test E2E only
	model = NewSensitiveDataPrompt(false, true)
	model.inputs[0].SetValue("keystorepass123")

	updatedModel, _ = model.Update(enterMsg)
	sdModel = updatedModel.(SensitiveDataModel)

	if !sdModel.finished {
		t.Error("Expected model to be finished")
	}

	if sdModel.keystorePass != "keystorepass123" {
		t.Errorf("Expected keystore passphrase to be 'keystorepass123', got '%s'", sdModel.keystorePass)
	}

	// Test both
	model = NewSensitiveDataPrompt(true, true)

	// Simulate typing admin key
	for _, char := range "adminkey123" {
		charMsg := tea.KeyPressMsg{Code: char, Text: string(char)}
		updatedModel, _ = model.Update(charMsg)
		model = updatedModel.(SensitiveDataModel)
	}

	// Move to next field (keystore passphrase)
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(SensitiveDataModel)

	// Simulate typing keystore passphrase
	for _, char := range "keystorepass123" {
		charMsg := tea.KeyPressMsg{Code: char, Text: string(char)}
		updatedModel, _ = model.Update(charMsg)
		model = updatedModel.(SensitiveDataModel)
	}

	// Submit
	updatedModel, _ = model.Update(enterMsg)
	sdModel = updatedModel.(SensitiveDataModel)

	if !sdModel.finished {
		t.Error("Expected model to be finished")
	}

	if sdModel.adminKey != "adminkey123" {
		t.Errorf("Expected admin key to be 'adminkey123', got '%s'", sdModel.adminKey)
	}

	if sdModel.keystorePass != "keystorepass123" {
		t.Errorf("Expected keystore passphrase to be 'keystorepass123', got '%s'", sdModel.keystorePass)
	}
}

// Test SensitiveDataModel navigation
func TestSensitiveDataModelNavigation(t *testing.T) {
	model := NewSensitiveDataPrompt(true, true)

	// Test tab navigation
	tabMsg := tea.KeyPressMsg{Code: tea.KeyTab}
	updatedModel, _ := model.Update(tabMsg)
	sdModel := updatedModel.(SensitiveDataModel)

	if sdModel.focusIndex != 1 {
		t.Errorf("Expected focus index to be 1 after tab, got %d", sdModel.focusIndex)
	}

	// Test shift+tab navigation
	shiftTabMsg := tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	updatedModel, _ = sdModel.Update(shiftTabMsg)
	sdModel = updatedModel.(SensitiveDataModel)

	if sdModel.focusIndex != 0 {
		t.Errorf("Expected focus index to be 0 after shift+tab, got %d", sdModel.focusIndex)
	}
}

// Test SensitiveDataModel cancellation
func TestSensitiveDataModelCancellation(t *testing.T) {
	model := NewSensitiveDataPrompt(true, false)

	// Test ESC key cancellation
	escMsg := tea.KeyPressMsg{Code: tea.KeyEsc}
	updatedModel, _ := model.Update(escMsg)
	sdModel := updatedModel.(SensitiveDataModel)

	if !sdModel.cancelled {
		t.Error("Expected model to be cancelled after ESC key")
	}

	// Test Ctrl+C cancellation
	model = NewSensitiveDataPrompt(true, false)
	ctrlCMsg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	updatedModel, _ = model.Update(ctrlCMsg)
	sdModel = updatedModel.(SensitiveDataModel)

	if !sdModel.cancelled {
		t.Error("Expected model to be cancelled after Ctrl+C")
	}
}

// Test View methods
func TestConfigUIModelView(t *testing.T) {
	model := NewConfigUI()
	view := model.View().Content

	// Check for title
	if !contains(view, "marchat Configuration") {
		t.Error("Expected view to contain title")
	}

	// Check for help text
	if !contains(view, "Tab/Shift+Tab") {
		t.Error("Expected view to contain help text")
	}
}

// Test ProfileSelectionModel View
func TestProfileSelectionModelView(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "Profile1", Username: "user1", ServerURL: "wss://server1.com", IsAdmin: true},
		{Name: "Profile2", Username: "user2", ServerURL: "wss://server2.com", UseE2E: true},
	}

	model := NewProfileSelectionModel(profiles, true)
	view := model.View().Content

	// Check for title
	if !contains(view, "Select a connection profile") {
		t.Error("Expected view to contain title")
	}

	// Check for profile names
	if !contains(view, "Profile1") {
		t.Error("Expected view to contain Profile1")
	}

	if !contains(view, "Profile2") {
		t.Error("Expected view to contain Profile2")
	}

	// Check for "Create New Profile" option
	if !contains(view, "Create New Profile") {
		t.Error("Expected view to contain 'Create New Profile' option")
	}

	// Check for help text
	if !contains(view, "↑/↓: Navigate") {
		t.Error("Expected view to contain help text")
	}
}

// Test ProfileSelectionModel viewDetails
func TestProfileSelectionModelViewDetails(t *testing.T) {
	profiles := []ConnectionProfile{
		{
			Name:      "TestProfile",
			Username:  "testuser",
			ServerURL: "wss://test.com",
			IsAdmin:   true,
			UseE2E:    true,
			Theme:     "modern",
			LastUsed:  1640995200, // Jan 1, 2022
		},
	}

	model := NewProfileSelectionModel(profiles, false)
	model.operation = ProfileOpView
	view := model.View().Content

	// Check for profile details
	if !contains(view, "Profile Details") {
		t.Error("Expected view to contain 'Profile Details'")
	}

	if !contains(view, "TestProfile") {
		t.Error("Expected view to contain profile name")
	}

	if !contains(view, "testuser") {
		t.Error("Expected view to contain username")
	}

	if !contains(view, "wss://test.com") {
		t.Error("Expected view to contain server URL")
	}
}

// Test ProfileSelectionModel viewRename
func TestProfileSelectionModelViewRename(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "TestProfile", Username: "testuser", ServerURL: "wss://test.com"},
	}

	model := NewEnhancedProfileSelectionModel(profiles, false, nil)
	model.operation = ProfileOpRename
	view := model.View().Content

	// Check for rename interface
	if !contains(view, "Rename Profile") {
		t.Error("Expected view to contain 'Rename Profile'")
	}

	if !contains(view, "TestProfile") {
		t.Error("Expected view to contain current profile name")
	}

	if !contains(view, "New name:") {
		t.Error("Expected view to contain rename input")
	}
}

// Test ProfileSelectionModel viewDelete
func TestProfileSelectionModelViewDelete(t *testing.T) {
	profiles := []ConnectionProfile{
		{Name: "TestProfile", Username: "testuser", ServerURL: "wss://test.com"},
	}

	model := NewProfileSelectionModel(profiles, false)
	model.operation = ProfileOpDelete
	model.deleteConfirm = "TestProfile"
	view := model.View().Content

	// Check for delete interface
	if !contains(view, "Delete Profile") {
		t.Error("Expected view to contain 'Delete Profile'")
	}

	if !contains(view, "Warning") {
		t.Error("Expected view to contain warning")
	}

	if !contains(view, "TestProfile") {
		t.Error("Expected view to contain profile name being deleted")
	}

	if !contains(view, "y: Confirm Delete") {
		t.Error("Expected view to contain confirmation instructions")
	}
}

// Test SensitiveDataModel View
func TestSensitiveDataModelView(t *testing.T) {
	model := NewSensitiveDataPrompt(true, true)
	view := model.View().Content

	// Check for title
	if !contains(view, "Authentication Required") {
		t.Error("Expected view to contain title")
	}

	// Check for help text
	if !contains(view, "Tab/Enter: Next") {
		t.Error("Expected view to contain help text")
	}
}
