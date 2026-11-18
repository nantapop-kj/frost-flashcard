package helper

import (
	"runtime"

	"github.com/nantapop-kj/frost-flashcard/assets"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func SetupTray(app *application.App) {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            "main",
		Title:           "Frost Flashcard",
		Width:           360,
		Height:          460,
		URL:             "/",
		Hidden:          true,
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		HideOnFocusLost: true,
		HideOnEscape:    true,
	})

	tray := app.SystemTray.New()
	SetTrayIcon(tray, runtime.GOOS, assets.TrayTemplateIcon, assets.TrayColorIcon)
	tray.SetTooltip("Frost Flashcard")

	menu := app.NewMenu()
	menu.Add("Quit Frost Flashcard").OnClick(OnClick(app.Quit))
	tray.SetMenu(menu)

	tray.AttachWindow(window).WindowOffset(5)
}

type TrayIconSetter interface {
	SetTemplateIcon([]byte) *application.SystemTray
	SetIcon([]byte) *application.SystemTray
}

func SetTrayIcon(tray TrayIconSetter, goos string, templateIcon, colorIcon []byte) {
	if goos == "darwin" {
		tray.SetTemplateIcon(templateIcon)
	} else {
		tray.SetIcon(colorIcon)
	}
}

func OnClick(action func()) func(*application.Context) {
	return func(*application.Context) { action() }
}
