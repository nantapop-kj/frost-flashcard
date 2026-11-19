package helper

import (
	"log"
	"os"
	"runtime"

	"github.com/nantapop-kj/frost-flashcard/assets"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const NavigateEvent = "navigate"

func SetupTray(app *application.App, dataDir string) {
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

	openView := func(view string) func() {
		return func() {
			app.Event.Emit(NavigateEvent, view)
			tray.ShowWindow()
		}
	}

	menu := app.NewMenu()
	menu.Add("Review Cards").SetAccelerator("CmdOrCtrl+R").OnClick(OnClick(openView("review")))
	menu.Add("Add Word…").SetAccelerator("CmdOrCtrl+N").OnClick(OnClick(openView("add")))
	menu.AddSeparator()

	enabled, err := app.Autostart.IsEnabled()
	if err != nil {
		log.Printf("cannot read launch-at-login state: %v", err)
	}
	menu.AddCheckbox("Launch at Login", enabled).OnClick(func(ctx *application.Context) {
		item := ctx.ClickedMenuItem()
		if err := SetAutostart(app.Autostart, item.Checked()); err != nil {
			log.Printf("cannot change launch-at-login: %v", err)
			item.SetChecked(!item.Checked())
		}
	})
	menu.Add("Open Data Folder").OnClick(OnClick(func() {
		if err := os.MkdirAll(dataDir, 0o755); err != nil {
			log.Printf("cannot create data folder: %v", err)
			return
		}
		if err := app.Browser.OpenFile(dataDir); err != nil {
			log.Printf("cannot open data folder: %v", err)
		}
	}))
	menu.AddSeparator()

	menu.Add("About Frost Flashcard").OnClick(OnClick(app.Menu.ShowAbout))
	menu.Add("Quit Frost Flashcard").SetAccelerator("CmdOrCtrl+Q").OnClick(OnClick(app.Quit))
	tray.SetMenu(menu)

	tray.AttachWindow(window).WindowOffset(5)
}

type Autostarter interface {
	Enable() error
	Disable() error
}

func SetAutostart(a Autostarter, on bool) error {
	if on {
		return a.Enable()
	}
	return a.Disable()
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
