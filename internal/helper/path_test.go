package helper

import (
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
