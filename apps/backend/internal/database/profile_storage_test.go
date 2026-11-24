package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewProfileStorageUsesAnIsolatedDirectory(t *testing.T) {
	dataDir := t.TempDir()
	storage, err := NewProfileStorage(dataDir, DefaultProfileID)
	if err != nil {
		t.Fatal(err)
	}

	if storage.Directory() != filepath.Join(dataDir, "profiles", DefaultProfileID) {
		t.Fatalf("unexpected default profile directory: %q", storage.Directory())
	}

	other, err := NewProfileStorage(dataDir, "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	if other.Directory() == storage.Directory() {
		t.Fatal("different profiles share the same directory")
	}
	if err := other.Ensure(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(other.Directory()); err != nil || !info.IsDir() {
		t.Fatalf("profile directory was not created: info=%v err=%v", info, err)
	}
}

func TestProfileStorageRejectsInvalidPaths(t *testing.T) {
	if _, err := NewProfileStorage(t.TempDir(), "../../outside"); err == nil {
		t.Fatal("invalid profile id was accepted")
	}
	storage, err := NewProfileStorage(t.TempDir(), DefaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "../devices.json", "profiles/devices.json"} {
		if _, err := storage.File(name); err == nil {
			t.Fatalf("invalid profile file name %q was accepted", name)
		}
	}
}

func TestMigrateLegacyDataIsIdempotent(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "devices.json"), []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "simulation-logs.events"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := migrateLegacyData(dataDir); err != nil {
		t.Fatal(err)
	}
	storage, err := NewProfileStorage(dataDir, DefaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(storage.Directory(), "devices.json")); err != nil {
		t.Fatalf("legacy file was not migrated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(storage.Directory(), "simulation-logs.events")); err != nil {
		t.Fatalf("event sidecar was not migrated: %v", err)
	}
	for _, name := range []string{"devices.json", "gateways.json", "simulation-logs.json"} {
		if _, err := os.Stat(filepath.Join(storage.Directory(), name)); err != nil {
			t.Fatalf("profile data file %s was not initialized: %v", name, err)
		}
	}
	if err := migrateLegacyData(dataDir); err != nil {
		t.Fatalf("second migration failed: %v", err)
	}
}
