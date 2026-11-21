package helper

import (
	"log"
	"os"
	"runtime"

	"github.com/nantapop-kj/frost-flashcard/assets"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	NavigateEvent     = "navigate"
	PopupShownEvent   = "popup-shown"
	popupCornerRadius = 14
)

func SetupTray(app *application.App, dataDir string) *application.SystemTray {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            "main",
		Title:           "Frost Flashcard",
		Width:           360,
		Height:          500,
		URL:             "/",
		Hidden:          true,
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		HideOnFocusLost: true,
		CloseButtonState: application.ButtonHidden,
		BackgroundType:   application.BackgroundTypeTranslucent,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropLiquidGlass,
			LiquidGlass: application.MacLiquidGlass{
				Style:        application.LiquidGlassStyleAutomatic,
				Material:     application.NSVisualEffectMaterialPopover,
				CornerRadius: popupCornerRadius,
			},
		},
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
			BackdropType:    application.Acrylic,
		},
	})
	window.RegisterHook(events.Common.WindowClosing, hideOnClose(window))
	window.OnWindowEvent(events.Common.WindowShow, emitOnShow(app.Event.Emit))

	tray := app.SystemTray.New()
	SetTrayIcon(tray, runtime.GOOS, assets.TrayTemplateIcon, assets.TrayColorIcon)
	tray.SetTooltip("Frost Flashcard")

	actions := &trayActions{
		emit:      app.Event.Emit,
		show:      tray.ShowWindow,
		autostart: app.Autostart,
		openFile:  app.Browser.OpenFile,
		dataDir:   dataDir,
	}

	menu := app.NewMenu()
	menu.Add("Review Cards").SetAccelerator("CmdOrCtrl+R").OnClick(OnClick(actions.openView("review")))
	menu.Add("Add Word…").SetAccelerator("CmdOrCtrl+N").OnClick(OnClick(actions.openView("add")))
	menu.AddSeparator()

	autostartItem := menu.AddCheckbox("Launch at Login", autostartEnabled(app.Autostart))
	actions.autostartItem = autostartItem
	autostartItem.OnClick(OnClick(actions.toggleAutostart))
	menu.Add("Open Data Folder").OnClick(OnClick(actions.openDataFolder))
	menu.AddSeparator()

	menu.Add("About Frost Flashcard").OnClick(OnClick(app.Menu.ShowAbout))
	menu.Add("Quit Frost Flashcard").SetAccelerator("CmdOrCtrl+Q").OnClick(OnClick(app.Quit))
	tray.SetMenu(menu)

	tray.AttachWindow(window).WindowOffset(5)

	return tray
}

type windowHider interface {
	Hide() application.Window
}

func hideOnClose(w windowHider) func(*application.WindowEvent) {
	return func(e *application.WindowEvent) {
		w.Hide()
		e.Cancel()
	}
}

type emitFunc func(name string, data ...any) bool

func emitOnShow(emit emitFunc) func(*application.WindowEvent) {
	return func(*application.WindowEvent) {
		emit(PopupShownEvent)
	}
}

type autostartChecker interface {
	IsEnabled() (bool, error)
}

func autostartEnabled(a autostartChecker) bool {
	enabled, err := a.IsEnabled()
	if err != nil {
		log.Printf("cannot read launch-at-login state: %v", err)
	}

	return enabled
}

type trayActions struct {
	emit          emitFunc
	show          func()
	autostart     Autostarter
	autostartItem CheckItem
	openFile      func(string) error
	dataDir       string
}

func (a *trayActions) openView(view string) func() {
	return func() {
		a.emit(NavigateEvent, view)
		a.show()
	}
}

func (a *trayActions) toggleAutostart() {
	if err := ToggleAutostart(a.autostart, a.autostartItem); err != nil {
		log.Printf("cannot change launch-at-login: %v", err)
	}
}

func (a *trayActions) openDataFolder() {
	if err := OpenDataFolder(a.dataDir, a.openFile); err != nil {
		log.Printf("cannot open data folder: %v", err)
	}
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

type CheckItem interface {
	Checked() bool
	SetChecked(bool) *application.MenuItem
}

func ToggleAutostart(a Autostarter, item CheckItem) error {
	err := SetAutostart(a, item.Checked())
	if err != nil {
		item.SetChecked(!item.Checked())
	}

	return err
}

func OpenDataFolder(dir string, open func(string) error) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return open(dir)
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
