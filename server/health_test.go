package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestHealthChecker(t *testing.T) (*HealthChecker, *sql.DB, func()) {
	t.Helper()

	// Create temporary database
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	CreateSchema(db)

	// Create hub with correct parameters
	hub := NewHub(tdir, tdir, "http://localhost:8080", db)
	go hub.Run()

	// Create health checker
	hc := NewHealthChecker(hub, db, "test-version")

	cleanup := func() {
		db.Close()
	}

	return hc, db, cleanup
}

func TestNewHealthChecker(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	if hc == nil {
		t.Fatal("HealthChecker should not be nil")
	}

	if hc.startTime.IsZero() {
		t.Error("Start time should be set")
	}

	if hc.version != "test-version" {
		t.Errorf("Expected version 'test-version', got '%s'", hc.version)
	}

	// Check that components are initialized
	expectedComponents := []string{"database", "websocket", "memory"}
	for _, component := range expectedComponents {
		if _, exists := hc.components[component]; !exists {
			t.Errorf("Component '%s' should be initialized", component)
		}
	}
}

func TestHealthChecker_CheckHealth(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	// Perform health check
	health := hc.CheckHealth()

	if health == nil {
		t.Fatal("Health check should not return nil")
	}

	if health.Version != "test-version" {
		t.Errorf("Expected version 'test-version', got '%s'", health.Version)
	}

	if health.Uptime == "" {
		t.Error("Uptime should not be empty")
	}

	// Check that all components are present
	expectedComponents := []string{"database", "websocket", "memory"}
	for _, component := range expectedComponents {
		if _, exists := health.Components[component]; !exists {
			t.Errorf("Component '%s' should be present in health check", component)
		}
	}

	// Verify timestamp is recent
	if time.Since(health.Timestamp) > time.Second {
		t.Error("Health check timestamp should be recent")
	}
}

func TestHealthChecker_CheckDatabaseHealth(t *testing.T) {
	hc, db, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	// Test healthy database
	health := hc.checkDatabaseHealth()

	if health.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}

	if health.Message == "" {
		t.Error("Message should not be empty")
	}

	if time.Since(health.LastCheck) > time.Second {
		t.Error("Last check time should be recent")
	}

	// Test with closed database (should be unhealthy)
	db.Close()
	health = hc.checkDatabaseHealth()

	if health.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status for closed database, got %s", health.Status)
	}
}

func TestHealthChecker_CheckWebSocketHealth(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	// Test with hub
	health := hc.checkWebSocketHealth()

	if health.Status != HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}

	if health.Message == "" {
		t.Error("Message should not be empty")
	}

	// Test with nil hub
	hc.hub = nil
	health = hc.checkWebSocketHealth()

	if health.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status for nil hub, got %s", health.Status)
	}

	if health.Message != "Hub not initialized" {
		t.Errorf("Expected 'Hub not initialized' message, got '%s'", health.Message)
	}
}

func TestHealthChecker_CheckMemoryHealth(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	health := hc.checkMemoryHealth()

	if health.Status != HealthStatusHealthy && health.Status != HealthStatusDegraded {
		t.Errorf("Expected healthy or degraded status, got %s", health.Status)
	}

	if health.Message == "" {
		t.Error("Message should not be empty")
	}

	if time.Since(health.LastCheck) > time.Second {
		t.Error("Last check time should be recent")
	}
}

func TestHealthChecker_DetermineOverallStatus(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	// Test all healthy
	for _, component := range hc.components {
		component.Status = HealthStatusHealthy
	}

	status := hc.determineOverallStatus()
	if status != HealthStatusHealthy {
		t.Errorf("Expected healthy overall status, got %s", status)
	}

	// Test one degraded
	hc.components["database"].Status = HealthStatusDegraded
	status = hc.determineOverallStatus()
	if status != HealthStatusDegraded {
		t.Errorf("Expected degraded overall status, got %s", status)
	}

	// Test one unhealthy
	hc.components["database"].Status = HealthStatusUnhealthy
	status = hc.determineOverallStatus()
	if status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy overall status, got %s", status)
	}
}

func TestHealthChecker_GetSystemMetrics(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	metrics := hc.getSystemMetrics()

	if metrics.MemoryUsage < 0 {
		t.Error("Memory usage should be non-negative")
	}

	if metrics.Goroutines < 0 {
		t.Error("Goroutines should be non-negative")
	}

	if metrics.ActiveUsers < 0 {
		t.Error("Active users should be non-negative")
	}

	if metrics.TotalMessages < 0 {
		t.Error("Total messages should be non-negative")
	}

	if metrics.DatabaseStatus == "" {
		t.Error("Database status should not be empty")
	}
}

func TestHealthChecker_GetComponentsMap(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	components := hc.getComponentsMap()

	if len(components) == 0 {
		t.Error("Components map should not be empty")
	}

	// Verify it's a copy, not a reference
	originalComponent := hc.components["database"]
	originalComponent.Status = HealthStatusUnhealthy

	copiedComponent := components["database"]
	if copiedComponent.Status == HealthStatusUnhealthy {
		t.Error("Components map should be a copy, not a reference")
	}
}

func TestHealthStatus_String(t *testing.T) {
	testCases := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusHealthy, "healthy"},
		{HealthStatusDegraded, "degraded"},
		{HealthStatusUnhealthy, "unhealthy"},
	}

	for _, tc := range testCases {
		result := tc.status.String()
		if result != tc.expected {
			t.Errorf("Expected '%s', got '%s'", tc.expected, result)
		}
	}
}

func TestHealthChecker_HealthCheckHandler(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	// Test healthy status
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	hc.HealthCheckHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected content-type application/json, got %s", w.Header().Get("Content-Type"))
	}

	// Parse response
	var health HealthCheck
	err := json.NewDecoder(w.Body).Decode(&health)
	if err != nil {
		t.Fatalf("Failed to decode health check response: %v", err)
	}

	if health.Version != "test-version" {
		t.Errorf("Expected version 'test-version', got '%s'", health.Version)
	}

	// Test unhealthy status - close the database to make it unhealthy
	hc.db.Close()
	w = httptest.NewRecorder()
	hc.HealthCheckHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for unhealthy, got %d", w.Code)
	}
}

func TestHealthChecker_SimpleHealthHandler(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/health/simple", nil)
	w := httptest.NewRecorder()

	hc.SimpleHealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", body)
	}

	// Test unhealthy status - close the database to make it unhealthy
	hc.db.Close()
	w = httptest.NewRecorder()
	hc.SimpleHealthHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 for unhealthy, got %d", w.Code)
	}

	body = w.Body.String()
	if body != "UNHEALTHY" {
		t.Errorf("Expected body 'UNHEALTHY', got '%s'", body)
	}
}

func TestHealthChecker_JSONEncoding(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	health := hc.CheckHealth()

	// Test that health check can be JSON encoded
	jsonData, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("Failed to marshal health check to JSON: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("JSON data should not be empty")
	}

	// Test that it can be unmarshaled back
	var decodedHealth HealthCheck
	err = json.Unmarshal(jsonData, &decodedHealth)
	if err != nil {
		t.Fatalf("Failed to unmarshal health check from JSON: %v", err)
	}

	if decodedHealth.Version != health.Version {
		t.Errorf("Version mismatch after JSON roundtrip")
	}

	if decodedHealth.Status != health.Status {
		t.Errorf("Status mismatch after JSON roundtrip")
	}
}

func TestHealthChecker_ConcurrentAccess(t *testing.T) {
	hc, _, cleanup := setupTestHealthChecker(t)
	defer cleanup()

	// Test concurrent access to health checker
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			health := hc.CheckHealth()
			if health == nil {
				t.Error("Health check should not return nil")
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
