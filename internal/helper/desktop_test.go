package helper

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func newTestDesktop(t *testing.T) *Desktop {
	t.Helper()
	app := application.New(application.Options{Name: "test"})
	return &Desktop{App: app, Tray: app.SystemTray.New()}
}

func TestReopenWithoutTrayDoesNothing(t *testing.T) {
	(&Desktop{}).reopen()
}

func TestReopenWithTrayWithoutWindow(t *testing.T) {
	newTestDesktop(t).reopen()
}

func TestPickFilesBeforeAppStartsReturnsNothing(t *testing.T) {
	d := newTestDesktop(t)

	if path, err := d.PickImportFile("Import"); path != "" || err != nil {
		t.Errorf("PickImportFile = %q, %v; want empty and nil", path, err)
	}
	if path, err := d.PickExportFile("Export", "deck.csv"); path != "" || err != nil {
		t.Errorf("PickExportFile = %q, %v; want empty and nil", path, err)
	}
}

func TestQuitBeforeAppStartsDoesNothing(t *testing.T) {
	newTestDesktop(t).Quit()
}

func TestDisableAutostartWhenNotRegistered(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	if err := newTestDesktop(t).DisableAutostart(); err != nil {
		t.Errorf("DisableAutostart = %v, want nil when nothing is registered", err)
	}
}
