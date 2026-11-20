package helper

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestReopenWithoutTrayDoesNothing(t *testing.T) {
	(&Dialogs{}).reopen()
}

func TestReopenWithTrayWithoutWindow(t *testing.T) {
	app := application.New(application.Options{Name: "test"})
	d := &Dialogs{App: app, Tray: app.SystemTray.New()}
	d.reopen()
}
