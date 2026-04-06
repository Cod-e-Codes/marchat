package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Cod-e-Codes/marchat/internal/doctor"
)

func marchatModuleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above cmd/server")
		}
		dir = parent
	}
}

func TestSubprocessDoctorPlain(t *testing.T) {
	root := marchatModuleRoot(t)
	cfgDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/server", "-doctor", "-config-dir", cfgDir)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"MARCHAT_DOCTOR_NO_NETWORK=1",
		"NO_COLOR=1",
		"CGO_ENABLED=0",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run -doctor: %v\n%s", err, out)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		t.Fatal("expected non-empty doctor output")
	}
}

func TestSubprocessDoctorJSON(t *testing.T) {
	root := marchatModuleRoot(t)
	cfgDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/server", "-doctor-json", "-config-dir", cfgDir)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"MARCHAT_DOCTOR_NO_NETWORK=1",
		"NO_COLOR=1",
		"CGO_ENABLED=0",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run -doctor-json: %v\n%s", err, out)
	}

	trim := bytes.TrimSpace(out)
	var rep doctor.Report
	if err := json.Unmarshal(trim, &rep); err != nil {
		t.Fatalf("invalid JSON doctor output: %v\n%s", err, string(trim))
	}
	if rep.Role != "server" {
		t.Fatalf("report.role = %q, want server", rep.Role)
	}
	if !strings.HasPrefix(rep.GoVersion, "go") {
		t.Fatalf("expected go_version, got %q", rep.GoVersion)
	}
}
