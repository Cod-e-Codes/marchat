package manager

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Cod-e-Codes/marchat/plugin/sdk"
)

// Same minimal plugin source as plugin/host/plugin_lifecycle_test.go (stdlib JSON IPC).
const minimalPluginMainForManager = `package main

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

func buildMinimalPluginForManager(t *testing.T, pluginName, pluginRoot string) {
	t.Helper()
	srcDir := filepath.Join(pluginRoot, "_build_"+pluginName)
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte(minimalPluginMainForManager), 0o644); err != nil {
		t.Fatalf("write main: %v", err)
	}
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

func TestPluginManagerDiscoverStartDisableEnable(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	const name = "mgrlifecycle"
	buildMinimalPluginForManager(t, name, pluginDir)

	// NewPluginManager runs discoverInstalledPlugins and auto-starts enabled plugins.
	m := NewPluginManager(pluginDir, dataDir, "https://example.invalid/registry.json")
	inst := m.GetPlugin(name)
	if inst == nil {
		t.Fatal("expected plugin discovered and loaded")
	}
	if inst.Process == nil {
		t.Fatal("expected auto-started plugin process")
	}

	if err := m.DisablePlugin(name); err != nil {
		t.Fatalf("DisablePlugin: %v", err)
	}
	if p := m.GetPlugin(name); p == nil || p.Process != nil {
		t.Fatal("expected plugin stopped after disable")
	}

	if err := m.EnablePlugin(name); err != nil {
		t.Fatalf("EnablePlugin: %v", err)
	}
	if p := m.GetPlugin(name); p == nil || p.Process == nil {
		t.Fatal("expected plugin running after enable")
	}

	_ = m.DisablePlugin(name)
}

func TestPluginManagerExecuteCommandDelegates(t *testing.T) {
	pluginDir := t.TempDir()
	dataDir := t.TempDir()
	const name = "mgrcmd"
	buildMinimalPluginForManager(t, name, pluginDir)

	m := NewPluginManager(pluginDir, dataDir, "https://example.invalid/registry.json")
	defer func() { _ = m.DisablePlugin(name) }()

	if err := m.ExecuteCommand(name, "hello", []string{"a"}); err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}
}
