package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Cod-e-Codes/marchat/config"
	"github.com/Cod-e-Codes/marchat/server"
	"github.com/Cod-e-Codes/marchat/shared"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

// Multi-admin support
// Usage: --admin Cody --admin Alice --admin-key changeme
//
// Remove old --admin-username logic

type multiFlag []string

func (m *multiFlag) String() string       { return strings.Join(*m, ",") }
func (m *multiFlag) Set(val string) error { *m = append(*m, val); return nil }

var adminUsers multiFlag
var adminKey = flag.String("admin-key", "", "Admin key for privileged commands (deprecated, use MARCHAT_ADMIN_KEY)")
var port = flag.Int("port", 0, "Port to listen on (deprecated, use MARCHAT_PORT)")
var configPath = flag.String("config", "", "Path to server config file (JSON, deprecated)")
var configDir = flag.String("config-dir", "", "Configuration directory (default: ./config in dev, $XDG_CONFIG_HOME/marchat in prod)")
var enableAdminPanel = flag.Bool("admin-panel", false, "Enable the built-in admin panel TUI")
var enableWebPanel = flag.Bool("web-panel", false, "Enable the built-in web admin panel (served at /admin)")
var nonInteractiveFlag = flag.Bool("non-interactive", false, "Disable interactive setup; exit if required config is missing (or set MARCHAT_NON_INTERACTIVE=1)")

// getEnvBool returns true if the specified env var is set to a truthy value
func getEnvBool(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func printBanner(addr string, admins []string, scheme string, tlsEnabled bool) {
	fmt.Println(`
⢀⠀⠀⠀⠀⠀⠀⠀⢀⣠⣤⣶⣶⣶⣶⣶⣶⣶⣶⣶⣦⡀⠀⠀⠀⠀⠀⠀⣀⣀⣀⣀⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀  
⣿⣷⠀⠀⣀⣤⣴⣾⣿⡿⣿⣧⣿⣶⣿⣿⣿⣽⣿⣽⣿⣷⣤⣤⣴⣶⣾⣿⣿⡿⠿⠛⠛⠿⣷⡀⢀⣀⣀⣀⣀⡀⠀⠀⠀⠀  
⠈⣿⣶⣿⣿⣛⣿⣶⣿⣿⣿⣿⣛⣿⣭⣿⣽⣿⣹⣿⣻⣿⡿⠿⠛⠛⠋⠉⠀⢀⣀⣀⣀⣀⣈⣿⠿⠿⠟⠻⠿⢿⡇⠀⠀⠀  
⠀⢹⣿⣿⣿⣿⡟⣿⣯⣿⣿⣿⣿⢿⣿⢻⣟⣻⡟⢿⠿⣿⣇⣄⣠⣤⣴⣶⣾⣿⡿⠿⠿⠻⠿⠇⠀⣀⣀⣀⣀⣸⡇⠀⠀⠀  
⠀⠀⢻⣿⣿⣿⣿⡿⣿⡛⣿⣥⣿⣿⣿⣿⢿⡿⣿⣿⣷⣿⣿⢿⣿⠿⠟⠋⠉⠀⢀⣀⣀⣀⣀⡘⣿⣿⠿⠿⠿⢿⡇⠀⠀⠀  
⠀⠀⠈⣿⡏⢿⣿⣷⣾⣿⡟⢿⣋⣿⣴⣿⣾⣷⣿⣷⣾⣾⣿⡆⠀⣀⣤⣤⣶⣾⣿⠿⠟⠛⠻⠿⣇⣀⣠⣤⣤⣼⡇⠀⠀⠀  
⠀⠀⠀⠸⣿⡀⢻⣿⣯⣸⣷⣾⡿⠟⠋⠉⠀⠀⠀⠀⠀⠀⠀⠘⣿⣿⠿⠟⠛⠉⢀⣠⣤⣤⣤⣥⣽⡿⠿⠿⠿⠿⡇⠀⠀⠀  
⠀⠀⠀⠀⢻⣷⠀⢻⣿⠟⠋⠁⠀⢀⣠⣤⣴⣶⣶⣶⣶⣾⣶⣾⡁⣀⣀⣤⣴⣾⣿⠿⠛⠋⠉⠉⢳⣤⣤⣤⣤⣤⣷⠀⠀⠀  
⠀⠀⠀⠀⠈⣿⣇⠀⣿⣀⣠⣴⣾⣿⡿⠟⠋⠉⠉⠀⠀⠀⠈⠉⣿⠿⠟⠛⠋⠁⢀⣤⣶⣶⣶⣶⣾⠟⠛⠋⠉⠉⢿⠀⠀⠀  
⠀⠀⠀⠀⠀⠘⣿⡆⠸⣿⡿⠟⠋⠁⢀⣀⣤⣴⣶⣶⣶⣶⣶⣾⣇⣀⣤⣤⣶⣾⡿⠛⠋⠉⠉⠉⠉⣤⣴⣶⣶⠶⢿⣇⠀⠀  
⠀⠀⠀⠀⠀⠀⢹⣿⡀⣿⡄⣀⣤⣾⡿⠟⠋⠉⠁⠀⠀⠀⠀⡿⠛⠛⠛⠉⠁⣀⣴⣾⣿⣿⣿⣿⡟⠉⠀⠀⣀⣀⣀⣿⡀⠀  
⠀⠀⠀⠀⠀⠀⠀⢿⣧⢸⣿⠟⠋⠁⣀⣠⣤⣴⣶⣾⣿⣿⣿⣿⣧⣤⣤⣶⠾⠟⠛⠉⢁⣀⣀⣀⣀⢰⣶⣿⠿⠿⠿⠿⣧⠀  
⠀⠀⠀⠀⠀⠀⠀⠈⣿⡆⣿⣠⣴⣿⣿⣿⠿⠟⠛⠉⠉⠀⠀⢠⡟⠋⠉⣀⣠⣴⣾⠿⠿⠟⠛⠛⠛⣿⠏⠀⠀⣀⣀⣀⣿⡄  
⠀⠀⠀⠀⠀⠀⠀⠀⠸⣿⣿⡿⠟⠋⠁⠀⠀⠀⠀⠀⠀⠀⠀⠸⣧⣶⠿⠛⠋⠁⠀⠀⠀⠀⠀⠀⠘⣿⣴⣾⠿⠛⠋⠉⠉⠁  
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢻⣿⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠁⠀⠀⠀⠀⠀⠀⠀  
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⣿⣆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀  
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠸⣿⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀  
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀  

░███     ░███                                ░██                      ░██    
░████   ░████                                ░██                      ░██    
░██░██ ░██░██  ░██████   ░██░████  ░███████  ░████████   ░██████   ░████████ 
░██ ░████ ░██       ░██  ░███     ░██    ░██ ░██    ░██       ░██     ░██    
░██  ░██  ░██  ░███████  ░██      ░██        ░██    ░██  ░███████     ░██    
░██       ░██ ░██   ░██  ░██      ░██    ░██ ░██    ░██ ░██   ░██     ░██    
░██       ░██  ░█████░██ ░██       ░███████  ░██    ░██  ░█████░██     ░████ `)
	fmt.Println()
	fmt.Printf("\U0001F310 WebSocket: %s://%s/ws\n", scheme, addr)
	fmt.Printf("\U0001F511 Admins: %s\n", strings.Join(admins, ", "))
	fmt.Printf("\U0001F4E6 Version: %s\n", shared.GetServerVersionInfo())
	if tlsEnabled {
		fmt.Println("\U0001F512 TLS: Enabled")
	} else {
		fmt.Println("\U0001F513 TLS: Disabled")
	}
	fmt.Println("\U0001F4A1 Tip: Use --username <admin> --admin --admin-key <key> to connect as admin")
}

func main() {
	flag.Var(&adminUsers, "admin", "[DEPRECATED] Admin username (use MARCHAT_USERS env var instead)")
	flag.Parse()

	// Check for MARCHAT_CONFIG_DIR environment variable first
	configDir := *configDir
	if envConfigDir := os.Getenv("MARCHAT_CONFIG_DIR"); envConfigDir != "" {
		configDir = envConfigDir
	}

	// Load configuration from environment variables and .env files (without validation)
	cfg, err := config.LoadConfigWithoutValidation(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nConfiguration options:\n")
		fmt.Fprintf(os.Stderr, "  Environment variables:\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_PORT=8080 (default: 8080)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_ADMIN_KEY=your-secret-key (required)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_USERS=user1,user2,user3 (comma-separated, required)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_DB_PATH=/path/to/db (default: $CONFIG_DIR/marchat.db)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_LOG_LEVEL=info (default: info)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_JWT_SECRET=your-jwt-secret (default: auto-generated)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_TLS_CERT_FILE=/path/to/cert.pem (optional)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_TLS_KEY_FILE=/path/to/key.pem (optional)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_CONFIG_DIR=/path/to/config (optional)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_BAN_HISTORY_GAPS=true (optional, default: true)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_PLUGIN_REGISTRY_URL=url (optional, default: GitHub registry)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_GLOBAL_E2E_KEY=base64-key (optional, for global E2E encryption)\n")
		fmt.Fprintf(os.Stderr, "  .env file: Create %s/.env with the above variables\n", configDir)
		fmt.Fprintf(os.Stderr, "  Config directory: Use --config-dir or MARCHAT_CONFIG_DIR to specify custom location\n")
		fmt.Fprintf(os.Stderr, "  Interactive setup: Run without required env vars for guided configuration\n")
		os.Exit(1)
	}

	// Check if required settings are missing and offer interactive configuration
	needsInteractiveConfig := false
	missingRequired := false
	missingE2E := false

	if cfg.AdminKey == "" {
		needsInteractiveConfig = true
		missingRequired = true
	}
	if len(cfg.Admins) == 0 {
		needsInteractiveConfig = true
		missingRequired = true
	}

	// Check if E2E configuration is missing when other settings are present
	// Only prompt for E2E if this is a completely fresh install (no .env file exists)
	if !missingRequired && cfg.GlobalE2EKey == "" {
		configDir := cfg.ConfigDir
		if configDir == "" {
			configDir = "./config"
		}
		envPath := filepath.Join(configDir, ".env")
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			// No .env file exists, this is a fresh install - offer E2E configuration
			needsInteractiveConfig = true
			missingE2E = true
		}
	}

	// Determine if we should skip interactive setup
	isTTY := term.IsTerminal(os.Stdin.Fd())
	nonInteractive := *nonInteractiveFlag || getEnvBool("MARCHAT_NON_INTERACTIVE") || !isTTY

	if needsInteractiveConfig {
		if nonInteractive {
			// Print clear non-interactive error and exit
			fmt.Fprintln(os.Stderr, "Missing required configuration and interactive setup is disabled.")
			if missingRequired {
				fmt.Fprintln(os.Stderr, "Set MARCHAT_ADMIN_KEY and MARCHAT_USERS (comma-separated) to proceed.")
			} else if missingE2E {
				fmt.Fprintln(os.Stderr, "Set MARCHAT_GLOBAL_E2E_KEY or allow the server to generate one on first run in interactive mode.")
			}
			fmt.Fprintln(os.Stderr, "Tip: Remove --non-interactive, unset MARCHAT_NON_INTERACTIVE, or run in a TTY to use guided setup.")
			os.Exit(2)
		}
		fmt.Println("🚀 Welcome to marchat server setup!")
		if missingRequired {
			fmt.Println("Some required configuration is missing. Let's set it up interactively.")
		} else if missingE2E {
			fmt.Println("E2E encryption configuration is missing. Let's configure it interactively.")
		}
		fmt.Println()

		serverConfig, err := server.RunServerConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
			os.Exit(1)
		}

		// Apply the interactive configuration
		cfg.AdminKey = serverConfig.AdminKey
		cfg.Admins = strings.Split(serverConfig.AdminUsers, ",")

		// Parse port as integer
		if port, err := strconv.Atoi(serverConfig.Port); err == nil {
			cfg.Port = port
		} else {
			fmt.Fprintf(os.Stderr, "Invalid port: %s\n", serverConfig.Port)
			os.Exit(1)
		}

		// Handle E2E configuration
		if serverConfig.EnableE2E {
			if serverConfig.GlobalE2EKey != "" {
				cfg.GlobalE2EKey = serverConfig.GlobalE2EKey
			}
			// If no key provided, we'll let the server generate one
		}

		// Clean up admin usernames (trim whitespace)
		for i, admin := range cfg.Admins {
			cfg.Admins[i] = strings.TrimSpace(admin)
		}

		fmt.Println()
		fmt.Println("✅ Configuration saved! You can now start the server.")
		fmt.Println("💡 Tip: Set environment variables to avoid this setup next time.")
		fmt.Println()
	}

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	// Warn about deprecated flags
	if len(adminUsers) > 0 {
		fmt.Fprintln(os.Stderr, "[WARNING] --admin flag is deprecated. Use MARCHAT_USERS environment variable.")
	}
	if *adminKey != "" {
		fmt.Fprintln(os.Stderr, "[WARNING] --admin-key flag is deprecated. Use MARCHAT_ADMIN_KEY environment variable.")
	}
	if *port != 0 {
		fmt.Fprintln(os.Stderr, "[WARNING] --port flag is deprecated. Use MARCHAT_PORT environment variable.")
	}
	if *configPath != "" {
		fmt.Fprintln(os.Stderr, "[WARNING] --config flag is deprecated. Use environment variables or .env file.")
	}

	// Override with deprecated flags if provided (for backward compatibility)
	admins := cfg.Admins
	key := cfg.AdminKey
	listenPort := cfg.Port

	if len(adminUsers) > 0 {
		admins = make([]string, len(adminUsers))
		copy(admins, adminUsers)
	}
	if *adminKey != "" {
		key = *adminKey
	}
	if *port != 0 {
		listenPort = *port
	}

	// Final validation
	if len(admins) == 0 {
		log.Fatal("At least one admin username is required (set MARCHAT_USERS or use --admin flag).")
	}
	if key == "" {
		log.Fatal("admin_key is required (set MARCHAT_ADMIN_KEY or use --admin-key flag).")
	}
	if listenPort < 1 || listenPort > 65535 {
		log.Fatal("Port must be between 1 and 65535.")
	}

	// Normalize admin usernames to lowercase and check for duplicates
	adminSet := make(map[string]struct{})
	for _, u := range admins {
		lu := strings.ToLower(u)
		if _, exists := adminSet[lu]; exists {
			log.Fatalf("Duplicate admin username (case-insensitive): %s", u)
		}
		adminSet[lu] = struct{}{}
	}
	admins = make([]string, 0, len(adminSet))
	for u := range adminSet {
		admins = append(admins, u)
	}

	// Initialize database with the configured path
	db := server.InitDB(cfg.DBPath)
	server.CreateSchema(db)

	// Set up plugin directories
	pluginDir := cfg.ConfigDir + "/plugins"
	dataDir := cfg.ConfigDir + "/data"

	// Get registry URL from configuration
	registryURL := cfg.PluginRegistryURL

	// Create plugin directories if they don't exist
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		log.Printf("Warning: Failed to create plugin directory %s: %v", pluginDir, err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Warning: Failed to create data directory %s: %v", dataDir, err)
	}

	hub := server.NewHub(pluginDir, dataDir, registryURL, db)
	go hub.Run()

	// Initialize admin panel if enabled (but don't start it yet)
	var adminPanelReady bool
	if *enableAdminPanel {
		adminPanelReady = true
	}

	http.HandleFunc("/ws", server.ServeWs(hub, db, admins, key, cfg.BanGapsHistory, cfg.MaxFileBytes))

	// Web admin panel routes (optional)
	if *enableWebPanel {
		web := server.NewWebAdminServer(hub, db, cfg)
		mux := http.DefaultServeMux
		web.RegisterRoutes(mux)
		log.Printf("Web admin panel enabled at /admin (use X-Admin-Key header)")
	}

	// Initialize health checker
	healthChecker := server.NewHealthChecker(hub, db, shared.GetServerVersionInfo())
	http.HandleFunc("/health", healthChecker.HealthCheckHandler)
	http.HandleFunc("/health/simple", healthChecker.SimpleHealthHandler)

	addr := fmt.Sprintf(":%d", listenPort)
	serverAddr := fmt.Sprintf("localhost:%d", listenPort)
	scheme := cfg.GetWebSocketScheme()

	// Print configuration info
	log.Printf("Configuration directory: %s", cfg.ConfigDir)
	log.Printf("Database path: %s", cfg.DBPath)

	// Print banner
	printBanner(serverAddr, admins, scheme, cfg.IsTLSEnabled())
	if adminPanelReady {
		fmt.Println("\U0001F4BB Admin Panel: Press Ctrl+A to open admin panel, Ctrl+C to shutdown")
	}

	// Create a custom server instance
	srv := &http.Server{Addr: addr}

	// Channel to listen for OS signals (Ctrl+C, etc.)
	stop := make(chan os.Signal, 1)
	adminShutdown := make(chan bool, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run server in a goroutine
	go func() {
		var err error
		if cfg.IsTLSEnabled() {
			err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Start admin panel hotkey listener
	if adminPanelReady {
		go func() {
			// Check if stdin is a terminal
			fd := os.Stdin.Fd()
			if !term.IsTerminal(fd) {
				log.Printf("Warning: stdin is not a terminal, admin panel hotkeys disabled")
				return
			}

			// Set terminal to raw mode for character-by-character input
			oldState, err := term.MakeRaw(fd)
			if err != nil {
				log.Printf("Warning: Could not set terminal to raw mode: %v", err)
				return
			}

			// Ensure we restore terminal state when the goroutine exits
			defer func() {
				if err := term.Restore(fd, oldState); err != nil {
					log.Printf("Warning: Could not restore terminal state: %v", err)
				}
			}()

			log.Printf("Admin panel ready - press Ctrl+A to open")

			// Read input character by character
			buf := make([]byte, 1)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil {
					log.Printf("Error reading from stdin: %v", err)
					break
				}
				if n == 0 {
					continue
				}

				// Check for Ctrl+A (ASCII 1) or Ctrl+C (ASCII 3)
				if buf[0] == 1 {
					log.Printf("Admin panel hotkey detected (Ctrl+A)")

					// Temporarily restore terminal state
					if err := term.Restore(fd, oldState); err != nil {
						log.Printf("Warning: Could not restore terminal state: %v", err)
					}

					// Launch admin panel
					pluginManager := hub.GetPluginManager()
					panel := server.NewAdminPanel(hub, db, pluginManager, cfg)
					p := tea.NewProgram(panel, tea.WithAltScreen())
					if _, err := p.Run(); err != nil {
						log.Printf("Admin panel error: %v", err)
					}

					// Set terminal back to raw mode
					oldState, err = term.MakeRaw(fd)
					if err != nil {
						log.Printf("Warning: Could not reset terminal to raw mode: %v", err)
						break
					}
					log.Printf("Admin panel ready - press Ctrl+A to open")
				} else if buf[0] == 3 { // Ctrl+C (ASCII 3)
					log.Printf("Ctrl+C detected, shutting down server...")

					// Restore terminal state before shutdown
					if err := term.Restore(fd, oldState); err != nil {
						log.Printf("Warning: Could not restore terminal state: %v", err)
					}

					// Signal shutdown via our channel
					adminShutdown <- true
					return
				}
			}
		}()
	}

	// Block until we receive SIGINT (Ctrl+C) or admin shutdown
	select {
	case <-stop:
		log.Println("Shutting down server gracefully...")
	case <-adminShutdown:
		log.Println("Shutting down server gracefully...")
	}

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v", err)
	}

	log.Println("Server shut down cleanly")
}
