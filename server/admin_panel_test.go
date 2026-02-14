package server

import (
	"path/filepath"
	"testing"

	appcfg "github.com/Cod-e-Codes/marchat/config"
)

func setupPanelEnv(t *testing.T) (*AdminPanel, func()) {
	t.Helper()
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	CreateSchema(db)
	pluginDir := filepath.Join(tdir, "plugins")
	dataDir := filepath.Join(tdir, "data")
	hub := NewHub(pluginDir, dataDir, "", db)
	cfg := &appcfg.Config{Port: 8080, AdminKey: "k", Admins: []string{"a"}, DBPath: dbPath, ConfigDir: tdir}
	panel := NewAdminPanel(hub, db, hub.GetPluginManager(), cfg)
	return panel, func() { _ = db.Close() }
}

func TestAdminPanel_InitAndRefresh(t *testing.T) {
	panel, cleanup := setupPanelEnv(t)
	defer cleanup()

	if panel == nil {
		t.Fatalf("panel is nil")
	}
	// basic invariants after NewAdminPanel -> refreshData called
	if panel.systemInfo.ServerStatus == "" {
		t.Errorf("expected server status set")
	}
	// call refresh again to ensure it doesn't panic and updates tables
	panel.refreshData()
	// userTable rows should be set (possibly empty) and not nil
	if panel.userTable.Rows() == nil {
		t.Errorf("expected user table rows initialized")
	}
}
