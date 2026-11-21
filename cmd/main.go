package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/nantapop-kj/frost-flashcard/assets"
	"github.com/nantapop-kj/frost-flashcard/frontend"
	"github.com/nantapop-kj/frost-flashcard/internal/helper"
	"github.com/nantapop-kj/frost-flashcard/internal/services"
	"github.com/nantapop-kj/frost-flashcard/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	appID       = "frostflashcard.app"
	legacyAppID = "dev.frostflashcard.app"
)

var version = "dev"

var (
	runApp = (*application.App).Run
	fatal  = log.Fatal
)

func main() {
	if err := run(); err != nil {
		fatal(err)
	}
}

func run() error {
	dataPath, err := helper.DataFile(appID, legacyAppID)
	if err != nil {
		return fmt.Errorf("cannot find a folder to store flashcards: %w", err)
	}
	lib, err := storage.LoadLibrary(dataPath)
	if err != nil {
		return fmt.Errorf("cannot load flashcards; fix or move the file and restart: %w", err)
	}

	desktop := &helper.Desktop{}
	cards := services.New(lib, dataPath, desktop)
	system := services.NewSystem(cards, version, desktop, appID, legacyAppID)
	app := application.New(application.Options{
		Name:        "Frost Flashcard",
		Description: "A flashcard app that lives in your macOS menu bar",
		Icon:        assets.AppIcon,
		Services: []application.Service{
			application.NewService(cards),
			application.NewService(system),
		},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(frontend.Assets)},
		Mac:    application.MacOptions{ActivationPolicy: application.ActivationPolicyAccessory},
	})

	desktop.App = app
	desktop.Tray = helper.SetupTray(app, filepath.Dir(dataPath))

	if err := runApp(app); err != nil {
		return fmt.Errorf("app stopped unexpectedly: %w", err)
	}

	return nil
}
