package main

import (
	"log"
	"runtime"

	"github.com/nantapop-kj/frost-flashcard/assets"
	"github.com/nantapop-kj/frost-flashcard/frontend"
	"github.com/nantapop-kj/frost-flashcard/internal/services"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	app := application.New(application.Options{
		Name:        "Frost Flashcard",
		Description: "A flashcard app that lives in your macOS menu bar",
		Icon:        assets.AppIcon,
		Services:    []application.Service{application.NewService(&services.Service{})},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(frontend.Assets)},
		Mac:         application.MacOptions{ActivationPolicy: application.ActivationPolicyAccessory},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            "main",
		Title:           "Frost Flashcard",
		Width:           400,
		Height:          300,
		URL:             "/",
		Hidden:          true,
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		HideOnFocusLost: true,
		HideOnEscape:    true,
	})

	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(assets.TrayTemplateIcon)
	} else {
		tray.SetIcon(assets.TrayColorIcon)
	}
	tray.SetTooltip("Frost Flashcard")

	menu := app.NewMenu()
	menu.Add("Quit Frost Flashcard").OnClick(func(*application.Context) {
		app.Quit()
	})
	tray.SetMenu(menu)

	tray.AttachWindow(window).WindowOffset(5)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
