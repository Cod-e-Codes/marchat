package doctor

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Cod-e-Codes/marchat/shared"
)

func TestFormatEnvValue(t *testing.T) {
	t.Parallel()
	if got := FormatEnvValue("MARCHAT_PORT", ""); got != "(not set)" {
		t.Fatalf("empty: got %q", got)
	}
	if got := FormatEnvValue("MARCHAT_PORT", "8080"); got != "8080" {
		t.Fatalf("plain: got %q", got)
	}
	got := FormatEnvValue("MARCHAT_ADMIN_KEY", "supersecret")
	if !strings.Contains(got, "len=11") {
		t.Fatalf("secret mask unexpected: %q", got)
	}
	if strings.Contains(got, "cret") {
		t.Fatalf("secret suffix leaked in output: %q", got)
	}
}

func TestParseLatestReleaseTag(t *testing.T) {
	t.Parallel()
	body := []byte(`{"tag_name":"v1.2.3","draft":false}`)
	tag, err := ParseLatestReleaseTag(body)
	if err != nil || tag != "v1.2.3" {
		t.Fatalf("got %q err %v", tag, err)
	}
	_, err = ParseLatestReleaseTag([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error for empty tag")
	}
}

// Not parallel: swaps package-level osEnviron; parallel tests would restore/stomp each other.
func TestBuildEnvLines_orderAndMask(t *testing.T) {
	environMu.Lock()
	old := osEnviron
	osEnviron = func() []string {
		return []string{
			"MARCHAT_PORT=9090",
			"MARCHAT_ADMIN_KEY=topsecret",
			"MARCHAT_EXTRA_CUSTOM=bar",
		}
	}
	environMu.Unlock()
	t.Cleanup(func() {
		environMu.Lock()
		osEnviron = old
		environMu.Unlock()
	})
	lines := buildEnvLines("server")
	foundPort := false
	foundExtra := false
	for _, e := range lines {
		if e.Key == "MARCHAT_PORT" && e.Display == "9090" {
			foundPort = true
		}
		if e.Key == "MARCHAT_EXTRA_CUSTOM" && e.Display == "bar" {
			foundExtra = true
		}
		if e.Key == "MARCHAT_ADMIN_KEY" && strings.Contains(e.Display, "topsecret") {
			t.Fatalf("admin key leaked: %q", e.Display)
		}
	}
	if !foundPort || !foundExtra {
		t.Fatalf("missing lines: port=%v extra=%v", foundPort, foundExtra)
	}
}

func TestCheckUpdate_fakeClient(t *testing.T) {
	t.Setenv("MARCHAT_DOCTOR_NO_NETWORK", "")
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			body := `{"tag_name":"v99.0.0"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	o := Options{HTTPClient: client}
	info := checkUpdate(o, "v0.1.0")
	if info.Latest != "v99.0.0" || info.UpToDate {
		t.Fatalf("unexpected: %+v", info)
	}
}

func TestCheckUpdate_noNetworkEnv(t *testing.T) {
	t.Setenv("MARCHAT_DOCTOR_NO_NETWORK", "1")
	info := checkUpdate(Options{}, "v1.0.0")
	if !info.Skipped || info.SkipReason == "" {
		t.Fatalf("expected skip: %+v", info)
	}
}

// TestRunServerDoctor_environmentReflectsDotenv ensures the Environment section
// reflects MARCHAT_* values after server config loads config/.env (godotenv.Overload).
// Not parallel: RunServer mutates process environment.
func TestRunServerDoctor_environmentReflectsDotenv(t *testing.T) {
	t.Setenv("MARCHAT_DOCTOR_NO_NETWORK", "1")

	prevCfgDir := os.Getenv("MARCHAT_CONFIG_DIR")
	t.Cleanup(func() {
		if prevCfgDir == "" {
			_ = os.Unsetenv("MARCHAT_CONFIG_DIR")
		} else {
			_ = os.Setenv("MARCHAT_CONFIG_DIR", prevCfgDir)
		}
	})
	_ = os.Unsetenv("MARCHAT_CONFIG_DIR")

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "doctor.sqlite")
	envBody := "MARCHAT_ADMIN_KEY=mysecretkey123456789012\nMARCHAT_USERS=alice,bob\nMARCHAT_DB_PATH=" + dbPath + "\n"
	if err := os.WriteFile(filepath.Join(tmp, ".env"), []byte(envBody), 0600); err != nil {
		t.Fatal(err)
	}

	dotenvKeys := []string{"MARCHAT_ADMIN_KEY", "MARCHAT_USERS", "MARCHAT_DB_PATH"}
	prev := make(map[string]string, len(dotenvKeys))
	for _, k := range dotenvKeys {
		prev[k] = os.Getenv(k)
	}
	t.Cleanup(func() {
		for _, k := range dotenvKeys {
			if prev[k] == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, prev[k])
			}
		}
	})

	var buf bytes.Buffer
	if err := RunServer(Options{JSON: true, Out: &buf, ServerConfigDirFlag: tmp}); err != nil {
		t.Fatalf("RunServer: %v", err)
	}
	var rep Report
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	var adminDisplay, usersDisplay string
	for _, e := range rep.Environment {
		switch e.Key {
		case "MARCHAT_ADMIN_KEY":
			adminDisplay = e.Display
		case "MARCHAT_USERS":
			usersDisplay = e.Display
		}
	}
	if strings.Contains(adminDisplay, "not set") || adminDisplay == "" {
		t.Fatalf("MARCHAT_ADMIN_KEY should reflect .env (masked), got %q", adminDisplay)
	}
	if usersDisplay != "alice,bob" {
		t.Fatalf("MARCHAT_USERS want alice,bob from .env, got %q", usersDisplay)
	}
}

// Not parallel: buildEnvLines reads osEnviron; must not run while another test's mock is installed.
func TestBuildEnvLines_clientIncludesHookVars(t *testing.T) {
	found := false
	for _, e := range buildEnvLines("client") {
		if e.Key == "MARCHAT_CLIENT_HOOK_RECEIVE" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("client doctor env should list MARCHAT_CLIENT_HOOK_RECEIVE")
	}
	for _, e := range buildEnvLines("server") {
		if e.Key == "MARCHAT_CLIENT_HOOK_RECEIVE" {
			t.Fatal("server doctor env should not list client hook vars")
		}
	}
}

// Not parallel: swaps package-level osEnviron (see TestBuildEnvLines_orderAndMask).
func TestBuildEnvLines_serverOmitsClientHookEnvEvenWhenSet(t *testing.T) {
	environMu.Lock()
	old := osEnviron
	osEnviron = func() []string {
		return []string{
			"MARCHAT_PORT=8080",
			"MARCHAT_CLIENT_HOOK_RECEIVE=C:\\temp\\marchat-hook-log.exe",
			"MARCHAT_CLIENT_HOOK_SEND=C:\\temp\\marchat-hook-log.exe",
			"MARCHAT_HOOK_LOG=C:\\Users\\x\\hook.log",
		}
	}
	environMu.Unlock()
	t.Cleanup(func() {
		environMu.Lock()
		osEnviron = old
		environMu.Unlock()
	})
	for _, e := range buildEnvLines("server") {
		switch e.Key {
		case "MARCHAT_CLIENT_HOOK_RECEIVE", "MARCHAT_CLIENT_HOOK_SEND", "MARCHAT_HOOK_LOG":
			t.Fatalf("server doctor should omit client hook env when set in process: saw %s", e.Key)
		}
	}
}

func TestAppendClientHookChecks_goodPath(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hook.exe")
	if err := os.WriteFile(p, []byte{1, 2}, 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARCHAT_CLIENT_HOOK_RECEIVE", abs)
	t.Setenv("MARCHAT_CLIENT_HOOK_SEND", "")
	t.Cleanup(func() {
		_ = os.Unsetenv("MARCHAT_CLIENT_HOOK_RECEIVE")
		_ = os.Unsetenv("MARCHAT_CLIENT_HOOK_SEND")
	})
	var checks []Check
	appendClientHookChecks(&checks)
	if len(checks) != 1 || checks[0].Status != "ok" || checks[0].ID != "client_hook_receive" {
		t.Fatalf("checks: %+v", checks)
	}
}

func TestAppendClientHookChecks_relativePathWarns(t *testing.T) {
	t.Setenv("MARCHAT_CLIENT_HOOK_RECEIVE", "relative\\hook.exe")
	t.Cleanup(func() { _ = os.Unsetenv("MARCHAT_CLIENT_HOOK_RECEIVE") })
	var checks []Check
	appendClientHookChecks(&checks)
	if len(checks) != 1 || checks[0].Status != "warn" {
		t.Fatalf("checks: %+v", checks)
	}
}

func TestWriteTextReport_nonTTYWriterIsPlainASCII(t *testing.T) {
	t.Parallel()
	rep := Report{
		Role:          "client",
		VersionDetail: "v0.0.0-test",
		GoVersion:     "go1.test",
		GOOS:          "plan9",
		GOARCH:        "amd64",
		StdoutTTY:     false,
		Checks: []Check{
			{ID: "sample", Status: "ok", Message: "all good"},
		},
		Update: UpdateInfo{Skipped: true, SkipReason: "test"},
	}
	var buf bytes.Buffer
	writeTextReport(&buf, rep)
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("expected plain output without ANSI escapes, got %q", out)
	}
	if !strings.Contains(out, "marchat doctor (client)") || !strings.Contains(out, "[ok] sample: all good") {
		t.Fatalf("unexpected plain report: %s", out)
	}
}

func TestReportJSON_roundTrip(t *testing.T) {
	t.Parallel()
	rep := Report{
		Role:          "client",
		Version:       shared.ClientVersion,
		VersionDetail: shared.GetVersionInfo(),
		Checks: []Check{
			{ID: "x", Status: "ok", Message: "y"},
		},
		Update: UpdateInfo{Current: "v1", Skipped: true, SkipReason: "test"},
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(rep); err != nil {
		t.Fatal(err)
	}
	var dec Report
	if err := json.NewDecoder(&buf).Decode(&dec); err != nil {
		t.Fatal(err)
	}
	if dec.Role != "client" || len(dec.Checks) != 1 {
		t.Fatalf("decode mismatch: %+v", dec)
	}
}

func TestRunClientDoctor_IncludesDMStateAndE2EKeySource(t *testing.T) {
	t.Setenv("MARCHAT_DOCTOR_NO_NETWORK", "1")
	tmp := t.TempDir()
	t.Setenv("MARCHAT_CONFIG_DIR", tmp)
	t.Setenv("MARCHAT_GLOBAL_E2E_KEY", "c29tZS1ub3QtcmVhbC1iYXNlNjQ=")

	if err := os.WriteFile(filepath.Join(tmp, "dm_state.json"), []byte(`{"last_seen":{"bob":10}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := RunClient(Options{JSON: true, Out: &buf}); err != nil {
		t.Fatalf("RunClient: %v", err)
	}
	var rep Report
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("decode report: %v", err)
	}

	foundDMState := false
	foundE2ESource := false
	for _, c := range rep.Checks {
		if c.ID == "dm_state" && strings.Contains(c.Message, "dm_state.json") {
			foundDMState = true
		}
		if c.ID == "e2e_key_source" && strings.Contains(c.Message, "MARCHAT_GLOBAL_E2E_KEY is set") {
			foundE2ESource = true
		}
	}
	if !foundDMState {
		t.Fatal("expected dm_state check in client doctor output")
	}
	if !foundE2ESource {
		t.Fatal("expected e2e_key_source check in client doctor output")
	}
}

func TestRunServerDoctor_IncludesDMHistoryCheck(t *testing.T) {
	t.Setenv("MARCHAT_DOCTOR_NO_NETWORK", "1")
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "doctor.sqlite")
	envBody := "MARCHAT_ADMIN_KEY=mysecretkey123456789012\nMARCHAT_USERS=alice,bob\nMARCHAT_DB_PATH=" + dbPath + "\n"
	if err := os.WriteFile(filepath.Join(tmp, ".env"), []byte(envBody), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := RunServer(Options{JSON: true, Out: &buf, ServerConfigDirFlag: tmp}); err != nil {
		t.Fatalf("RunServer: %v", err)
	}
	var rep Report
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	for _, c := range rep.Checks {
		if c.ID == "dm_history" {
			return
		}
	}
	t.Fatal("expected dm_history check in server doctor output")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
