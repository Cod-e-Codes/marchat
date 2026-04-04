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
	"github.com/Cod-e-Codes/marchat/internal/doctor"
	"github.com/Cod-e-Codes/marchat/server"
	"github.com/Cod-e-Codes/marchat/shared"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

// Pre-TUI / banner styling (respects NO_COLOR via lipgloss/termenv).
var (
	srvLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#90A4AE")).Bold(true)
	srvVal   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ECEFF1"))
	srvURL   = lipgloss.NewStyle().Foreground(lipgloss.Color("#4FC3F7"))
	srvOK    = lipgloss.NewStyle().Foreground(lipgloss.Color("#81C784"))
	srvWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB74D"))
	srvKey   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF59D"))
	srvDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("#78909C"))
	srvInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4DD0E1"))
)

// Multi-admin support
// Usage: --admin alice --admin bob --admin-key changeme
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
var interactiveFlag = flag.Bool("interactive", false, "Enable interactive setup when required configuration is missing")
var runDoctor = flag.Bool("doctor", false, "Print environment and configuration diagnostics, then exit")
var runDoctorJSON = flag.Bool("doctor-json", false, "Same as -doctor with JSON output (if both are set, JSON is used)")

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
	ws := fmt.Sprintf("%s://%s/ws", scheme, addr)
	fmt.Println(srvLabel.Render("WebSocket: ") + srvURL.Render(ws))
	fmt.Println(srvLabel.Render("Admins: ") + srvVal.Render(strings.Join(admins, ", ")))
	fmt.Println(srvLabel.Render("Version: ") + srvVal.Render(shared.GetServerVersionInfo()))
	if tlsEnabled {
		fmt.Println(srvLabel.Render("TLS: ") + srvOK.Render("Enabled"))
	} else {
		fmt.Println(srvLabel.Render("TLS: ") + srvWarn.Render("Disabled"))
	}
	fmt.Println(srvDim.Render("Tip: Use ") + srvKey.Render("--username <admin> --admin --admin-key <key>") + srvDim.Render(" to connect as admin"))
}

func main() {
	flag.Var(&adminUsers, "admin", "[DEPRECATED] Admin username (use MARCHAT_USERS env var instead)")
	flag.Parse()

	if *runDoctorJSON {
		if err := doctor.RunServer(doctor.Options{JSON: true, ServerConfigDirFlag: *configDir}); err != nil {
			fmt.Fprintf(os.Stderr, "doctor: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if *runDoctor {
		if err := doctor.RunServer(doctor.Options{ServerConfigDirFlag: *configDir}); err != nil {
			fmt.Fprintf(os.Stderr, "doctor: %v\n", err)
			os.Exit(1)
		}
		return
	}

	actualConfigDir := doctor.ResolveServerConfigDir(*configDir)

	// Redirect runtime logs to debug file (but keep startup logs on stdout)
	debugLogPath := filepath.Join(actualConfigDir, "marchat-debug.log")
	if err := server.LogToFile(debugLogPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to redirect logs to debug file: %v\n", err)
	}

	// Load configuration from environment variables and .env files (without validation)
	cfg, err := config.LoadConfigWithoutValidation(actualConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nConfiguration options:\n")
		fmt.Fprintf(os.Stderr, "  Environment variables:\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_PORT=8080 (default: 8080)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_ADMIN_KEY=your-secret-key (required)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_USERS=user1,user2,user3 (comma-separated, required)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_DB_PATH=/path/to/db (default: $CONFIG_DIR/marchat.db)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_LOG_LEVEL=info (default: info)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_SESSION_SECRET=your-secret (default: auto-generated)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_JWT_SECRET=your-secret (deprecated alias for SESSION_SECRET)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_TLS_CERT_FILE=/path/to/cert.pem (optional)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_TLS_KEY_FILE=/path/to/key.pem (optional)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_CONFIG_DIR=/path/to/config (optional)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_BAN_HISTORY_GAPS=true (optional, default: false)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_PLUGIN_REGISTRY_URL=url (optional, default: GitHub registry)\n")
		fmt.Fprintf(os.Stderr, "    MARCHAT_GLOBAL_E2E_KEY=base64-key (optional, for global E2E encryption)\n")
		fmt.Fprintf(os.Stderr, "  .env file: Create %s/.env with the above variables\n", actualConfigDir)
		fmt.Fprintf(os.Stderr, "  Config directory: Use --config-dir or MARCHAT_CONFIG_DIR to specify custom location\n")
		fmt.Fprintf(os.Stderr, "  Interactive setup: Use --interactive flag for guided configuration\n")
		os.Exit(1)
	}

	// Check if required settings are missing and offer interactive configuration
	needsInteractiveConfig := false

	if cfg.AdminKey == "" {
		needsInteractiveConfig = true
	}
	if len(cfg.Admins) == 0 {
		needsInteractiveConfig = true
	}

	if needsInteractiveConfig {
		if !*interactiveFlag {
			// Print clear non-interactive error and exit
			fmt.Fprintln(os.Stderr, "Missing required configuration.")
			fmt.Fprintln(os.Stderr, "Set MARCHAT_ADMIN_KEY and MARCHAT_USERS (comma-separated) to proceed.")
			fmt.Fprintln(os.Stderr, "Tip: Use --interactive flag for guided configuration setup.")
			os.Exit(2)
		}

		fmt.Println(srvInfo.Render("[INFO]") + " Welcome to marchat server setup!")
		fmt.Println(srvDim.Render("Some required configuration is missing. Let's set it up interactively."))
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

		// Clean up admin usernames (trim whitespace)
		for i, admin := range cfg.Admins {
			cfg.Admins[i] = strings.TrimSpace(admin)
		}

		fmt.Println()
		fmt.Println(srvOK.Render("[OK]") + " Configuration saved. You can now start the server.")
		fmt.Println(srvDim.Render("[TIP] Set environment variables to avoid this setup next time."))
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
	db, err := server.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	server.CreateSchema(db)

	// Set up plugin directories
	pluginDir := cfg.ConfigDir + "/plugins"
	dataDir := cfg.ConfigDir + "/data"

	// Get registry URL from configuration
	registryURL := cfg.PluginRegistryURL

	// Create plugin directories if they don't exist
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		server.ServerLogger.Warn("Failed to create plugin directory", map[string]interface{}{
			"path":  pluginDir,
			"error": err.Error(),
		})
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		server.ServerLogger.Warn("Failed to create data directory", map[string]interface{}{
			"path":  dataDir,
			"error": err.Error(),
		})
	}

	hub := server.NewHub(pluginDir, dataDir, registryURL, db)
	go hub.Run()

	// Log server startup
	server.ServerLogger.Info("Server starting", map[string]interface{}{
		"port":        listenPort,
		"admin_count": len(admins),
		"plugin_dir":  pluginDir,
		"db_path":     cfg.DBPath,
	})

	// Initialize admin panel if enabled (but don't start it yet)
	var adminPanelReady bool
	if *enableAdminPanel {
		adminPanelReady = true
	}

	http.HandleFunc("/ws", server.ServeWs(hub, db, admins, key, cfg.BanGapsHistory, cfg.MaxFileBytes, cfg.DBPath))

	// Web admin panel routes (optional)
	if *enableWebPanel {
		web := server.NewWebAdminServer(hub, db, cfg)
		mux := http.DefaultServeMux
		web.RegisterRoutes(mux)
		server.ServerLogger.Info("Web admin panel enabled", map[string]interface{}{
			"endpoint": "/admin",
		})
	}

	// Initialize health checker
	healthChecker := server.NewHealthChecker(hub, db, shared.GetServerVersionInfo())
	http.HandleFunc("/health", healthChecker.HealthCheckHandler)
	http.HandleFunc("/health/simple", healthChecker.SimpleHealthHandler)

	addr := fmt.Sprintf(":%d", listenPort)
	serverAddr := fmt.Sprintf("localhost:%d", listenPort)
	scheme := cfg.GetWebSocketScheme()

	// Log configuration info
	server.ServerLogger.Info("Configuration loaded", map[string]interface{}{
		"config_dir": cfg.ConfigDir,
		"db_path":    cfg.DBPath,
	})

	// Print banner
	printBanner(serverAddr, admins, scheme, cfg.IsTLSEnabled())
	if adminPanelReady {
		fmt.Println(srvDim.Render("Admin panel: Press ") + srvKey.Render("Ctrl+A") + srvDim.Render(" to open admin panel, ") + srvKey.Render("Ctrl+C") + srvDim.Render(" to shutdown"))
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
				server.ServerLogger.Warn("stdin is not a terminal, admin panel hotkeys disabled", nil)
				return
			}

			// Set terminal to raw mode for character-by-character input
			oldState, err := term.MakeRaw(fd)
			if err != nil {
				server.ServerLogger.Warn("Could not set terminal to raw mode", map[string]interface{}{
					"error": err.Error(),
				})
				return
			}

			// Ensure we restore terminal state when the goroutine exits
			defer func() {
				if err := term.Restore(fd, oldState); err != nil {
					server.ServerLogger.Warn("Could not restore terminal state", map[string]interface{}{
						"error": err.Error(),
					})
				}
			}()

			server.ServerLogger.Info("Admin panel ready", map[string]interface{}{
				"hotkey": "Ctrl+A",
			})

			// Read input character by character
			buf := make([]byte, 1)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil {
					server.ServerLogger.Error("Error reading from stdin", err)
					break
				}
				if n == 0 {
					continue
				}

				// Check for Ctrl+A (ASCII 1) or Ctrl+C (ASCII 3)
				if buf[0] == 1 {
					// Temporarily restore terminal state
					if err := term.Restore(fd, oldState); err != nil {
						server.ServerLogger.Warn("Could not restore terminal state", map[string]interface{}{
							"error": err.Error(),
						})
					}

					// Launch admin panel
					pluginManager := hub.GetPluginManager()
					panel := server.NewAdminPanel(hub, db, pluginManager, cfg)
					p := tea.NewProgram(panel, tea.WithAltScreen())
					if _, err := p.Run(); err != nil {
						server.ServerLogger.Error("Admin panel error", err)
					}

					// Set terminal back to raw mode
					oldState, err = term.MakeRaw(fd)
					if err != nil {
						server.ServerLogger.Warn("Could not reset terminal to raw mode", map[string]interface{}{
							"error": err.Error(),
						})
						break
					}
					server.ServerLogger.Info("Admin panel closed", nil)
				} else if buf[0] == 3 { // Ctrl+C (ASCII 3)
					// Restore terminal state before shutdown
					if err := term.Restore(fd, oldState); err != nil {
						server.ServerLogger.Warn("Could not restore terminal state", map[string]interface{}{
							"error": err.Error(),
						})
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
		server.ServerLogger.Info("Shutdown signal received", nil)
	case <-adminShutdown:
		server.ServerLogger.Info("Shutdown initiated from admin panel", nil)
	}

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		server.ServerLogger.Error("Graceful shutdown failed", err)
		log.Fatalf("Graceful shutdown failed: %v", err)
	}

	server.ServerLogger.Info("Server shut down cleanly", nil)
}
