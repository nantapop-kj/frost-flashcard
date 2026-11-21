package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func setHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("AppData", "")
	if home != "" {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		t.Setenv("AppData", filepath.Join(home, "AppData"))
	}
}

func TestDataFileIsInsideUserConfigDir(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)

	got, err := DataFile("my.app")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "flashcards.json" || filepath.Base(filepath.Dir(got)) != "my.app" {
		t.Errorf("DataFile = %q, want .../my.app/flashcards.json", got)
	}
	if rel, err := filepath.Rel(home, got); err != nil || rel == got || rel[:2] == ".." {
		t.Errorf("DataFile = %q, want it under %q", got, home)
	}
}

func TestDataFileFailsWithoutHome(t *testing.T) {
	setHome(t, "")
	if got, err := DataFile("my.app"); err == nil {
		t.Errorf("DataFile = %q, want an error when no config folder can be found", got)
	}
}

func writeData(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flashcards.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readData(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

func TestDataFileMovesLegacyFolder(t *testing.T) {
	setHome(t, t.TempDir())
	config, _ := os.UserConfigDir()
	legacy := filepath.Join(config, "old.app")
	writeData(t, legacy, "old data")

	got, err := DataFile("new.app", "missing.app", "old.app")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(config, "new.app", "flashcards.json") {
		t.Errorf("DataFile = %q", got)
	}
	if readData(t, got) != "old data" {
		t.Error("data was not moved to the new folder")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy folder still exists: %v", err)
	}
}

func TestDataFilePrefersCurrentFolder(t *testing.T) {
	setHome(t, t.TempDir())
	config, _ := os.UserConfigDir()
	writeData(t, filepath.Join(config, "new.app"), "new data")
	writeData(t, filepath.Join(config, "old.app"), "old data")

	got, err := DataFile("new.app", "old.app")
	if err != nil {
		t.Fatal(err)
	}
	if readData(t, got) != "new data" {
		t.Error("should use the current folder when it exists")
	}
	if readData(t, filepath.Join(config, "old.app", "flashcards.json")) != "old data" {
		t.Error("legacy folder should be left alone")
	}
}

func TestDataFileKeepsLegacyFolderWhenMoveFails(t *testing.T) {
	setHome(t, t.TempDir())
	config, _ := os.UserConfigDir()
	legacy := filepath.Join(config, "old.app")
	writeData(t, legacy, "old data")
	os.Chmod(config, 0o500)
	t.Cleanup(func() { os.Chmod(config, 0o755) })

	got, err := DataFile("new.app", "old.app")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(legacy, "flashcards.json") || readData(t, got) != "old data" {
		t.Errorf("DataFile = %q, want the legacy file", got)
	}
}
