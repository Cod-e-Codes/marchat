package doctor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	clientcfg "github.com/Cod-e-Codes/marchat/client/config"
	appconfig "github.com/Cod-e-Codes/marchat/config"
	"github.com/Cod-e-Codes/marchat/shared"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Text report styling (aligned with server pre-TUI banner). Lipgloss disables color
// for NO_COLOR, non-TTY, and dumb terminals automatically when rendering.
var (
	docTitle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#4DD0E1")).Bold(true)
	docLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("#90A4AE")).Bold(true)
	docVal     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ECEFF1"))
	docSection = lipgloss.NewStyle().Foreground(lipgloss.Color("#90A4AE")).Bold(true)
	docKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF59D"))
	docDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("#78909C"))
	docOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("#81C784")).Bold(true)
	docWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB74D")).Bold(true)
	docErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("#E57373")).Bold(true)
)

// Options configures doctor output and networking.
type Options struct {
	Out io.Writer

	// JSON emits machine-readable JSON instead of text lines.
	JSON bool

	// HTTPClient is used for the GitHub latest-release check (optional).
	HTTPClient *http.Client

	// ServerConfigDirFlag is the value of the server's -config-dir flag (may be empty).
	ServerConfigDirFlag string
}

func (o Options) out() io.Writer {
	if o.Out != nil {
		return o.Out
	}
	return os.Stdout
}

// Check is one diagnostic line in the report.
type Check struct {
	ID      string `json:"id"`
	Status  string `json:"status"` // ok, warn, error
	Message string `json:"message"`
}

// UpdateInfo describes a remote version comparison (if available).
type UpdateInfo struct {
	Current    string `json:"current"`
	Latest     string `json:"latest,omitempty"`
	UpToDate   bool   `json:"up_to_date"`
	Skipped    bool   `json:"skipped"`
	SkipReason string `json:"skip_reason,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Report is the full doctor payload for JSON output.
type Report struct {
	Role          string     `json:"role"`
	Version       string     `json:"version"`
	VersionDetail string     `json:"version_detail"`
	GoVersion     string     `json:"go_version"`
	GOOS          string     `json:"goos"`
	GOARCH        string     `json:"goarch"`
	StdoutTTY     bool       `json:"stdout_tty"`
	ConfigDir     string     `json:"config_dir,omitempty"`
	Environment   []envLine  `json:"environment"`
	Checks        []Check    `json:"checks"`
	Update        UpdateInfo `json:"update"`
}

func appendCheck(checks *[]Check, id, status, msg string) {
	*checks = append(*checks, Check{ID: id, Status: status, Message: msg})
}

func doctorNoNetwork() bool {
	v := strings.TrimSpace(os.Getenv("MARCHAT_DOCTOR_NO_NETWORK"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func checkUpdate(o Options, current string) UpdateInfo {
	info := UpdateInfo{Current: current}
	if doctorNoNetwork() {
		info.Skipped = true
		info.SkipReason = "MARCHAT_DOCTOR_NO_NETWORK is set"
		return info
	}
	if current == "" || current == "dev" {
		info.Skipped = true
		info.SkipReason = "embedded version is dev or empty"
		return info
	}
	latest, err := fetchLatestReleaseTag(o.HTTPClient)
	if err != nil {
		info.Error = "could not check for updates"
		return info
	}
	info.Latest = latest
	cmp := shared.CompareVersions(current, latest)
	if cmp >= 0 {
		info.UpToDate = true
	} else {
		info.UpToDate = false
	}
	return info
}

// RunServer writes a server-focused doctor report.
func RunServer(o Options) error {
	dir := ResolveServerConfigDir(o.ServerConfigDirFlag)
	absDir, absErr := filepath.Abs(dir)
	if absErr != nil {
		absDir = dir
	}
	checks := make([]Check, 0, 16)

	appendCheck(&checks, "config_dir", "ok", fmt.Sprintf("resolved config directory (absolute): %s", absDir))

	envPath := filepath.Join(dir, ".env")
	if st, err := os.Stat(envPath); err != nil {
		appendCheck(&checks, "dotenv", "warn", fmt.Sprintf(".env not found or unreadable at %s (%v)", envPath, err))
	} else if st.IsDir() {
		appendCheck(&checks, "dotenv", "error", fmt.Sprintf(".env path is a directory: %s", envPath))
	} else {
		appendCheck(&checks, "dotenv", "ok", fmt.Sprintf(".env present (%d bytes)", st.Size()))
	}

	cfg, err := appconfig.LoadConfigWithoutValidation(dir)
	if err != nil {
		appendCheck(&checks, "config_load", "error", fmt.Sprintf("load config: %v", err))
	} else {
		appendCheck(&checks, "config_load", "ok", "configuration loaded from env and .env")
		if valErr := cfg.Validate(); valErr != nil {
			appendCheck(&checks, "config_validate", "error", valErr.Error())
		} else {
			appendCheck(&checks, "config_validate", "ok", "required server settings are valid")
		}

		dbPath := cfg.DBPath
		appendCheck(&checks, "db_path", "ok", fmt.Sprintf("database path: %s", dbPath))

		target := detectDBTarget(dbPath)
		appendCheck(&checks, "db_dialect", "ok", fmt.Sprintf("detected DB dialect: %s (driver: %s)", target.dialect, target.driver))

		if err := validateConnectionString(target); err != nil {
			appendCheck(&checks, "db_connection_string", "error", err.Error())
		} else {
			appendCheck(&checks, "db_connection_string", "ok", "connection string format is valid")
		}

		if target.dialect == "sqlite" {
			parent := filepath.Dir(dbPath)
			if fi, err := os.Stat(parent); err != nil {
				appendCheck(&checks, "db_parent", "warn", fmt.Sprintf("database parent dir: %v", err))
			} else if !fi.IsDir() {
				appendCheck(&checks, "db_parent", "error", "database parent path is not a directory")
			} else {
				f, err := os.CreateTemp(parent, "marchat-doctor-*.tmp")
				if err != nil {
					appendCheck(&checks, "db_parent_writable", "warn", fmt.Sprintf("cannot create temp file in DB parent: %v", err))
				} else {
					_ = f.Close()
					_ = os.Remove(f.Name())
					appendCheck(&checks, "db_parent_writable", "ok", "database parent directory is writable")
				}
			}
		}

		if cfg.IsTLSEnabled() {
			if _, err := os.Stat(cfg.TLSCertFile); err != nil {
				appendCheck(&checks, "tls_cert", "error", fmt.Sprintf("TLS cert not readable: %v", err))
			} else {
				appendCheck(&checks, "tls_cert", "ok", fmt.Sprintf("TLS cert file: %s", cfg.TLSCertFile))
			}
			if _, err := os.Stat(cfg.TLSKeyFile); err != nil {
				appendCheck(&checks, "tls_key", "error", fmt.Sprintf("TLS key not readable: %v", err))
			} else {
				appendCheck(&checks, "tls_key", "ok", fmt.Sprintf("TLS key file: %s", cfg.TLSKeyFile))
			}
		} else {
			if os.Getenv("MARCHAT_TLS_CERT_FILE") != "" || os.Getenv("MARCHAT_TLS_KEY_FILE") != "" {
				appendCheck(&checks, "tls", "warn", "only one of TLS cert/key is set; TLS disabled until both are configured")
			} else {
				appendCheck(&checks, "tls", "ok", "TLS not configured (plain WS)")
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		db, err := sql.Open(target.driver, target.dsn)
		if err != nil {
			appendCheck(&checks, "db_ping", "warn", fmt.Sprintf("open %s: %v", target.dialect, err))
		} else {
			defer db.Close()
			if err := db.PingContext(ctx); err != nil {
				appendCheck(&checks, "db_ping", "warn", fmt.Sprintf("%s ping: %v", target.dialect, err))
			} else {
				appendCheck(&checks, "db_ping", "ok", fmt.Sprintf("%s database reachable", target.dialect))
			}
		}
	}

	up := checkUpdate(o, shared.ServerVersion)
	if up.Error != "" {
		appendCheck(&checks, "update", "warn", up.Error)
	} else if up.Skipped {
		appendCheck(&checks, "update", "ok", fmt.Sprintf("skipped: %s", up.SkipReason))
	} else if up.UpToDate {
		appendCheck(&checks, "update", "ok", fmt.Sprintf("up to date (latest %s)", up.Latest))
	} else {
		appendCheck(&checks, "update", "warn", fmt.Sprintf("newer release available: %s (running %s)", up.Latest, up.Current))
	}

	// Snapshot MARCHAT_* after config load so .env (godotenv.Overload) is reflected.
	envLines := buildEnvLines("server")

	rep := Report{
		Role:          "server",
		Version:       shared.ServerVersion,
		VersionDetail: shared.GetServerVersionInfo(),
		GoVersion:     runtime.Version(),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		StdoutTTY:     term.IsTerminal(os.Stdout.Fd()),
		ConfigDir:     absDir,
		Environment:   envLines,
		Checks:        checks,
		Update:        up,
	}

	if o.JSON {
		enc := json.NewEncoder(o.out())
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}
	writeTextReport(o.out(), rep)
	return nil
}

func isTermux() bool {
	return os.Getenv("TERMUX_VERSION") != "" ||
		os.Getenv("PREFIX") == "/data/data/com.termux/files/usr" ||
		(os.Getenv("ANDROID_DATA") != "" && os.Getenv("ANDROID_ROOT") != "")
}

func clipboardOK() bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- clipboard.WriteAll("marchat-doctor") }()
	select {
	case err := <-done:
		return err == nil
	case <-ctx.Done():
		return false
	}
}

// RunClient writes a client-focused doctor report.
func RunClient(o Options) error {
	if err := clientcfg.EnsureClientConfigDir(); err != nil {
		return fmt.Errorf("client config directory: %w", err)
	}
	dir := clientcfg.ResolveClientConfigDir()
	absDir, absErr := filepath.Abs(dir)
	if absErr != nil {
		absDir = dir
	}
	envLines := buildEnvLines("client")
	checks := make([]Check, 0, 24)

	appendCheck(&checks, "config_dir", "ok", fmt.Sprintf("client config directory (absolute): %s", absDir))

	appendClientHookChecks(&checks)

	if os.Getenv("MARCHAT_CONFIG_DIR") != "" {
		appendCheck(&checks, "config_layout", "ok", "MARCHAT_CONFIG_DIR overrides the default per-user client directory")
	} else {
		appendCheck(&checks, "config_layout", "ok", "client uses the per-user application data directory (Windows: %APPDATA%\\marchat; Linux: ~/.config/marchat), not the repo ./config folder (that is for the server .env and database)")
		if _, err := os.Stat("go.mod"); err == nil {
			appendCheck(&checks, "repo_note", "ok", "go.mod in cwd: server doctor still uses ./config for .env; client paths are unchanged")
		}
	}

	cfgPath := filepath.Join(dir, "config.json")
	if _, err := os.Stat(cfgPath); err != nil {
		appendCheck(&checks, "config_json", "warn", fmt.Sprintf("config.json missing or unreadable (%v)", err))
	} else {
		if _, err := clientcfg.LoadConfig(cfgPath); err != nil {
			appendCheck(&checks, "config_json", "warn", fmt.Sprintf("config.json present but invalid: %v", err))
		} else {
			appendCheck(&checks, "config_json", "ok", fmt.Sprintf("config.json OK: %s", cfgPath))
		}
	}

	profPath := filepath.Join(dir, "profiles.json")
	if _, err := os.Stat(profPath); err != nil {
		appendCheck(&checks, "profiles_json", "warn", fmt.Sprintf("profiles.json missing or unreadable (%v)", err))
	} else {
		data, err := os.ReadFile(profPath)
		if err != nil {
			appendCheck(&checks, "profiles_json", "warn", fmt.Sprintf("read profiles: %v", err))
		} else {
			var profs clientcfg.Profiles
			if err := json.Unmarshal(data, &profs); err != nil {
				appendCheck(&checks, "profiles_json", "warn", fmt.Sprintf("profiles.json invalid JSON: %v", err))
			} else {
				appendCheck(&checks, "profiles_json", "ok", fmt.Sprintf("%d saved profile(s)", len(profs.Profiles)))
			}
		}
	}

	ksPath, err := clientcfg.GetKeystorePath()
	if err != nil {
		appendCheck(&checks, "keystore", "warn", fmt.Sprintf("keystore path: %v", err))
	} else {
		primaryKs, _ := filepath.Abs(filepath.Join(absDir, "keystore.dat"))
		if _, err := os.Stat(ksPath); err != nil {
			appendCheck(&checks, "keystore", "ok", fmt.Sprintf("no keystore file yet (expected at %s)", primaryKs))
		} else {
			appendCheck(&checks, "keystore", "ok", fmt.Sprintf("keystore file present: %s", ksPath))
			if primaryKs != "" && filepath.Clean(ksPath) != filepath.Clean(primaryKs) {
				appendCheck(&checks, "keystore_note", "ok", fmt.Sprintf("keystore is outside the config directory; new runs use %s first, with fallback to older per-user paths", primaryKs))
			}
		}
	}

	if isTermux() {
		appendCheck(&checks, "termux", "ok", "Termux-like environment detected")
	} else {
		appendCheck(&checks, "termux", "ok", "not Termux")
	}

	if clipboardOK() {
		appendCheck(&checks, "clipboard", "ok", "clipboard write test succeeded")
	} else {
		appendCheck(&checks, "clipboard", "warn", "clipboard write failed or timed out (may be headless/SSH)")
	}

	if term.IsTerminal(os.Stdout.Fd()) {
		appendCheck(&checks, "stdout_tty", "ok", "stdout is a terminal")
	} else {
		appendCheck(&checks, "stdout_tty", "warn", "stdout is not a terminal")
	}

	up := checkUpdate(o, shared.ClientVersion)
	if up.Error != "" {
		appendCheck(&checks, "update", "warn", up.Error)
	} else if up.Skipped {
		appendCheck(&checks, "update", "ok", fmt.Sprintf("skipped: %s", up.SkipReason))
	} else if up.UpToDate {
		appendCheck(&checks, "update", "ok", fmt.Sprintf("up to date (latest %s)", up.Latest))
	} else {
		appendCheck(&checks, "update", "warn", fmt.Sprintf("newer release available: %s (running %s)", up.Latest, up.Current))
	}

	rep := Report{
		Role:          "client",
		Version:       shared.ClientVersion,
		VersionDetail: shared.GetVersionInfo(),
		GoVersion:     runtime.Version(),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		StdoutTTY:     term.IsTerminal(os.Stdout.Fd()),
		ConfigDir:     absDir,
		Environment:   envLines,
		Checks:        checks,
		Update:        up,
	}

	if o.JSON {
		enc := json.NewEncoder(o.out())
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}
	writeTextReport(o.out(), rep)
	return nil
}

func doctorColorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TERM")), "dumb") {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(f.Fd())
}

func checkStatusStyle(status string) lipgloss.Style {
	switch status {
	case "error":
		return docErr
	case "warn":
		return docWarn
	default:
		return docOK
	}
}

func writeTextReport(w io.Writer, rep Report) {
	if !doctorColorEnabled(w) {
		writeTextReportPlain(w, rep)
		return
	}
	writeTextReportColor(w, rep)
}

func writeTextReportPlain(w io.Writer, rep Report) {
	fmt.Fprintf(w, "marchat doctor (%s)\n", rep.Role)
	fmt.Fprintf(w, "  version: %s\n", rep.VersionDetail)
	fmt.Fprintf(w, "  go: %s  OS/Arch: %s/%s  stdout TTY: %v\n", rep.GoVersion, rep.GOOS, rep.GOARCH, rep.StdoutTTY)
	if rep.ConfigDir != "" {
		fmt.Fprintf(w, "  config dir: %s\n", rep.ConfigDir)
	}
	fmt.Fprintln(w, "\nEnvironment (MARCHAT_*, secrets masked):")
	for _, e := range rep.Environment {
		fmt.Fprintf(w, "  %s=%s\n", e.Key, e.Display)
	}
	fmt.Fprintln(w, "\nChecks:")
	for _, c := range rep.Checks {
		fmt.Fprintf(w, "  [%s] %s: %s\n", c.Status, c.ID, c.Message)
	}
	fmt.Fprintln(w, "\nUpdates:")
	writeUpdatesPlain(w, rep.Update)
}

func writeUpdatesPlain(w io.Writer, u UpdateInfo) {
	switch {
	case u.Error != "":
		fmt.Fprintf(w, "  [warn] %s\n", u.Error)
	case u.Skipped:
		fmt.Fprintf(w, "  [ok] skipped: %s\n", u.SkipReason)
	case u.UpToDate:
		fmt.Fprintf(w, "  [ok] up to date (latest %s)\n", u.Latest)
	default:
		fmt.Fprintf(w, "  [warn] newer release %s available (running %s)\n", u.Latest, u.Current)
	}
}

func writeTextReportColor(w io.Writer, rep Report) {
	fmt.Fprintln(w, docTitle.Render(fmt.Sprintf("marchat doctor (%s)", rep.Role)))
	fmt.Fprintln(w, docLabel.Render("  version: ")+docVal.Render(rep.VersionDetail))
	fmt.Fprintln(w, docLabel.Render("  go: ")+docVal.Render(fmt.Sprintf("%s  OS/Arch: %s/%s  stdout TTY: %v", rep.GoVersion, rep.GOOS, rep.GOARCH, rep.StdoutTTY)))
	if rep.ConfigDir != "" {
		fmt.Fprintln(w, docLabel.Render("  config dir: ")+docVal.Render(rep.ConfigDir))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, docSection.Render("Environment (MARCHAT_*, secrets masked):"))
	for _, e := range rep.Environment {
		fmt.Fprintln(w, "  "+docKey.Render(e.Key+"=")+docVal.Render(e.Display))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, docSection.Render("Checks:"))
	for _, c := range rep.Checks {
		tag := checkStatusStyle(c.Status).Render("[" + c.Status + "]")
		line := docDim.Render(c.ID+": ") + docVal.Render(c.Message)
		fmt.Fprintln(w, "  "+tag+" "+line)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, docSection.Render("Updates:"))
	writeUpdatesColor(w, rep.Update)
}

func writeUpdatesColor(w io.Writer, u UpdateInfo) {
	switch {
	case u.Error != "":
		fmt.Fprintln(w, "  "+docWarn.Render("[warn]")+" "+docVal.Render(u.Error))
	case u.Skipped:
		fmt.Fprintln(w, "  "+docOK.Render("[ok]")+" "+docVal.Render("skipped: "+u.SkipReason))
	case u.UpToDate:
		fmt.Fprintln(w, "  "+docOK.Render("[ok]")+" "+docVal.Render(fmt.Sprintf("up to date (latest %s)", u.Latest)))
	default:
		fmt.Fprintln(w, "  "+docWarn.Render("[warn]")+" "+docVal.Render(fmt.Sprintf("newer release %s available (running %s)", u.Latest, u.Current)))
	}
}
