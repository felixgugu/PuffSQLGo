package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"puffsql/internal/model"
)

func TestStoreRoundTripsProfilesAtomically(t *testing.T) {
	store := NewAt(t.TempDir())
	profiles := []model.ConnectionProfile{{
		ConnectionConfig: model.ConnectionConfig{
			Name: "Local", Host: "localhost", Port: 1433, Database: "master", Username: "sa",
		},
		ID: "one", Engine: "mssql", CreatedAt: "created", UpdatedAt: "updated",
	}}
	if err := store.SaveProfiles(profiles); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].ID != "one" || loaded[0].Host != "localhost" {
		t.Fatalf("loaded profiles = %#v", loaded)
	}
	info, err := os.Stat(store.ConnectionsPath())
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("connection file should be private, mode = %v", info.Mode())
	}
}

func TestStoreSurvivesMalformedProfileFile(t *testing.T) {
	dir := t.TempDir()
	store := NewAt(dir)
	if err := os.WriteFile(store.ConnectionsPath(), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	profiles, err := store.LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if profiles == nil || len(profiles) != 0 {
		t.Fatalf("want empty profiles, got %#v", profiles)
	}
}

func TestLegacyDirectoryMigration(t *testing.T) {
	base := t.TempDir()
	legacy := filepath.Join(base, legacyDirName)
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(legacy, "connections.json")
	if err := os.WriteFile(expected, []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := migrateLegacyDir(base); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, appDirName, "connections.json")); err != nil {
		t.Fatalf("legacy data was not moved: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy directory still exists")
	}
}

func TestTemplateFileIsCreatedAndEditable(t *testing.T) {
	store := NewAt(t.TempDir())
	path, err := store.EnsureTemplateFile()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != store.Dir() {
		t.Fatalf("NewAt must stay inside its isolated directory; got %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("sample template file is empty")
	}
	if err := os.WriteFile(path, []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	again, err := store.EnsureTemplateFile()
	if err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(again)
	if string(data) != "[]" {
		t.Fatalf("existing template file was overwritten: %q", data)
	}
}
