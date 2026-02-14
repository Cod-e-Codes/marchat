package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check response
type HealthCheck struct {
	Status     HealthStatus               `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Version    string                     `json:"version"`
	Uptime     string                     `json:"uptime"`
	Components map[string]ComponentHealth `json:"components"`
	Metrics    SystemMetrics              `json:"metrics"`
}

// ComponentHealth represents the health of a specific component
type ComponentHealth struct {
	Status    HealthStatus `json:"status"`
	Message   string       `json:"message,omitempty"`
	LastCheck time.Time    `json:"last_check"`
}

// SystemMetrics represents system performance metrics
type SystemMetrics struct {
	MemoryUsage    float64 `json:"memory_usage_mb"`
	Goroutines     int     `json:"goroutines"`
	ActiveUsers    int     `json:"active_users"`
	TotalMessages  int     `json:"total_messages"`
	DatabaseStatus string  `json:"database_status"`
}

// HealthChecker manages health check functionality
type HealthChecker struct {
	startTime  time.Time
	hub        *Hub
	db         *sql.DB
	version    string
	components map[string]*ComponentHealth
	mutex      sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(hub *Hub, db *sql.DB, version string) *HealthChecker {
	hc := &HealthChecker{
		startTime:  time.Now(),
		hub:        hub,
		db:         db,
		version:    version,
		components: make(map[string]*ComponentHealth),
	}

	// Initialize component health
	hc.components["database"] = &ComponentHealth{
		Status:    HealthStatusHealthy,
		LastCheck: time.Now(),
	}
	hc.components["websocket"] = &ComponentHealth{
		Status:    HealthStatusHealthy,
		LastCheck: time.Now(),
	}
	hc.components["memory"] = &ComponentHealth{
		Status:    HealthStatusHealthy,
		LastCheck: time.Now(),
	}

	return hc
}

// CheckHealth performs a comprehensive health check
func (hc *HealthChecker) CheckHealth() *HealthCheck {
	now := time.Now()
	uptime := now.Sub(hc.startTime)

	// Check database health
	dbHealth := hc.checkDatabaseHealth()

	// Check websocket health
	wsHealth := hc.checkWebSocketHealth()

	// Check memory health
	memHealth := hc.checkMemoryHealth()

	// Update components with write lock
	hc.mutex.Lock()
	hc.components["database"] = dbHealth
	hc.components["websocket"] = wsHealth
	hc.components["memory"] = memHealth
	hc.mutex.Unlock()

	// Determine overall status
	overallStatus := hc.determineOverallStatus()

	// Get system metrics
	metrics := hc.getSystemMetrics()

	return &HealthCheck{
		Status:     overallStatus,
		Timestamp:  now,
		Version:    hc.version,
		Uptime:     uptime.Round(time.Second).String(),
		Components: hc.getComponentsMap(),
		Metrics:    metrics,
	}
}

// checkDatabaseHealth checks the database connection and performance
func (hc *HealthChecker) checkDatabaseHealth() *ComponentHealth {
	start := time.Now()

	// Test database connection with a simple query
	var count int
	err := hc.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)

	responseTime := time.Since(start)

	health := &ComponentHealth{
		LastCheck: time.Now(),
	}

	if err != nil {
		health.Status = HealthStatusUnhealthy
		health.Message = fmt.Sprintf("Database error: %v", err)
		SecurityLogger.Error("Database health check failed", err)
	} else if responseTime > 5*time.Second {
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("Slow response: %v", responseTime)
	} else {
		health.Status = HealthStatusHealthy
		health.Message = fmt.Sprintf("Response time: %v", responseTime)
	}

	return health
}

// checkWebSocketHealth checks websocket connectivity
func (hc *HealthChecker) checkWebSocketHealth() *ComponentHealth {
	health := &ComponentHealth{
		LastCheck: time.Now(),
	}

	if hc.hub == nil {
		health.Status = HealthStatusUnhealthy
		health.Message = "Hub not initialized"
		return health
	}

	// Check if hub is responsive
	hc.hub.clientsMutex.RLock()
	clientCount := len(hc.hub.clients)
	hc.hub.clientsMutex.RUnlock()
	if clientCount >= 1000 { // Arbitrary limit
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("High client count: %d", clientCount)
	} else {
		health.Status = HealthStatusHealthy
		health.Message = fmt.Sprintf("Active clients: %d", clientCount)
	}

	return health
}

// checkMemoryHealth checks memory usage
func (hc *HealthChecker) checkMemoryHealth() *ComponentHealth {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	health := &ComponentHealth{
		LastCheck: time.Now(),
	}

	memoryMB := float64(m.Alloc) / 1024 / 1024

	if memoryMB > 1000 { // 1GB limit
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("High memory usage: %.1f MB", memoryMB)
	} else if memoryMB > 500 { // 500MB warning
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("Moderate memory usage: %.1f MB", memoryMB)
	} else {
		health.Status = HealthStatusHealthy
		health.Message = fmt.Sprintf("Memory usage: %.1f MB", memoryMB)
	}

	return health
}

// determineOverallStatus determines the overall system health status
func (hc *HealthChecker) determineOverallStatus() HealthStatus {
	hasUnhealthy := false
	hasDegraded := false

	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	for _, component := range hc.components {
		switch component.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return HealthStatusUnhealthy
	} else if hasDegraded {
		return HealthStatusDegraded
	}

	return HealthStatusHealthy
}

// getSystemMetrics collects system performance metrics
func (hc *HealthChecker) getSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	activeUsers := 0
	if hc.hub != nil {
		hc.hub.clientsMutex.RLock()
		activeUsers = len(hc.hub.clients)
		hc.hub.clientsMutex.RUnlock()
	}

	totalMessages := 0
	if hc.db != nil {
		_ = hc.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&totalMessages)
	}

	hc.mutex.RLock()
	databaseStatus := hc.components["database"].Status.String()
	hc.mutex.RUnlock()

	return SystemMetrics{
		MemoryUsage:    float64(m.Alloc) / 1024 / 1024,
		Goroutines:     runtime.NumGoroutine(),
		ActiveUsers:    activeUsers,
		TotalMessages:  totalMessages,
		DatabaseStatus: databaseStatus,
	}
}

// getComponentsMap returns a copy of the components map
func (hc *HealthChecker) getComponentsMap() map[string]ComponentHealth {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	components := make(map[string]ComponentHealth)
	for name, health := range hc.components {
		components[name] = *health
	}
	return components
}

// String returns the string representation of HealthStatus
func (hs HealthStatus) String() string {
	return string(hs)
}

// HealthCheckHandler handles HTTP health check requests
func (hc *HealthChecker) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	health := hc.CheckHealth()

	// Set appropriate HTTP status code
	switch health.Status {
	case HealthStatusHealthy:
		w.WriteHeader(http.StatusOK)
	case HealthStatusDegraded:
		w.WriteHeader(http.StatusOK) // Still OK, but with warnings
	case HealthStatusUnhealthy:
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Encode and send response
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(health); err != nil {
		ServerLogger.Error("Failed to encode health check response", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// SimpleHealthHandler provides a simple health check endpoint
func (hc *HealthChecker) SimpleHealthHandler(w http.ResponseWriter, r *http.Request) {
	health := hc.CheckHealth()

	// Simple response for load balancers
	if health.Status == HealthStatusHealthy {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("UNHEALTHY"))
	}
}
