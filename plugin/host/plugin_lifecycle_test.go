package host

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Cod-e-Codes/marchat/plugin/sdk"
)

// minimalPluginMain is a tiny stdin/stdout JSON plugin that answers init and shutdown.
const minimalPluginMain = `package main

import (
	"encoding/json"
	"os"
)

func main() {
	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for {
		var req map[string]interface{}
		if err := dec.Decode(&req); err != nil {
			return
		}
		t, _ := req["type"].(string)
		switch t {
		case "init":
			_ = enc.Encode(map[string]interface{}{"type": "response", "success": true})
		case "shutdown":
			return
		default:
			_ = enc.Encode(map[string]interface{}{"type": "response", "success": true})
		}
	}
}
`

func buildMinimalPluginBinary(t *testing.T, pluginName, pluginRoot string) {
	t.Helper()
	srcDir := filepath.Join(pluginRoot, "_build_"+pluginName)
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	mainPath := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(minimalPluginMain), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
	// go build requires a module boundary when GO111MODULE=on (default).
	if err := os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module marchat_test_minimal_plugin\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	pluginPath := filepath.Join(pluginRoot, pluginName)
	if err := os.MkdirAll(pluginPath, 0o755); err != nil {
		t.Fatalf("mkdir plugin: %v", err)
	}

	outName := pluginName
	if runtime.GOOS == "windows" {
		outName += ".exe"
	}
	outPath := filepath.Join(pluginPath, outName)

	cmd := exec.Command("go", "build", "-o", outPath, ".")
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build plugin: %v\n%s", err, out)
	}

	manifest := sdk.PluginManifest{
		Name:        pluginName,
		Version:     "1.0.0",
		Description: "minimal test plugin",
		Author:      "test",
		License:     "MIT",
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginPath, "plugin.json"), data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestPluginHostStartStopLifecycle(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	const name = "lifecycleplug"
	buildMinimalPluginBinary(t, name, pluginDir)

	h := NewPluginHost(pluginDir, dataDir)
	if err := h.LoadPlugin(name); err != nil {
		t.Fatalf("LoadPlugin: %v", err)
	}
	if err := h.StartPlugin(name); err != nil {
		t.Fatalf("StartPlugin: %v", err)
	}
	inst := h.GetPlugin(name)
	if inst == nil || inst.Process == nil {
		t.Fatal("expected running process after StartPlugin")
	}
	if err := h.StopPlugin(name); err != nil {
		t.Fatalf("StopPlugin: %v", err)
	}
	if inst := h.GetPlugin(name); inst == nil || inst.Process != nil {
		t.Fatal("expected Process cleared after StopPlugin")
	}
}

func TestPluginHostStartPluginAlreadyRunning(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	const name = "doublestart"
	buildMinimalPluginBinary(t, name, pluginDir)

	h := NewPluginHost(pluginDir, dataDir)
	if err := h.LoadPlugin(name); err != nil {
		t.Fatalf("LoadPlugin: %v", err)
	}
	if err := h.StartPlugin(name); err != nil {
		t.Fatalf("StartPlugin: %v", err)
	}
	defer func() { _ = h.StopPlugin(name) }()

	if err := h.StartPlugin(name); err == nil {
		t.Fatal("expected error when StartPlugin on already running plugin")
	}
}

func TestPluginHostStopPluginIdempotent(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	const name = "stopnoop"
	buildMinimalPluginBinary(t, name, pluginDir)

	h := NewPluginHost(pluginDir, dataDir)
	if err := h.LoadPlugin(name); err != nil {
		t.Fatalf("LoadPlugin: %v", err)
	}
	if err := h.StopPlugin(name); err != nil {
		t.Fatalf("StopPlugin on non-started: %v", err)
	}
}

func TestPluginHostExecuteCommandWhenRunning(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	const name = "cmdplug"
	buildMinimalPluginBinary(t, name, pluginDir)

	h := NewPluginHost(pluginDir, dataDir)
	if err := h.LoadPlugin(name); err != nil {
		t.Fatalf("LoadPlugin: %v", err)
	}
	if err := h.StartPlugin(name); err != nil {
		t.Fatalf("StartPlugin: %v", err)
	}
	defer func() { _ = h.StopPlugin(name) }()

	if err := h.ExecuteCommand(name, "ping", nil); err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}
}
