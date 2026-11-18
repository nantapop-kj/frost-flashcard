package main

import (
	"fmt"
	"log"

	"github.com/nantapop-kj/frost-flashcard/assets"
	"github.com/nantapop-kj/frost-flashcard/frontend"
	"github.com/nantapop-kj/frost-flashcard/internal/helper"
	"github.com/nantapop-kj/frost-flashcard/internal/services"
	"github.com/nantapop-kj/frost-flashcard/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const appID = "dev.frostflashcard.app"

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
	dataPath, err := helper.DataFile(appID)
	if err != nil {
		return fmt.Errorf("cannot find a folder to store flashcards: %w", err)
	}
	lib, err := storage.LoadLibrary(dataPath)
	if err != nil {
		return fmt.Errorf("cannot load flashcards; fix or move the file and restart: %w", err)
	}

	app := application.New(application.Options{
		Name:        "Frost Flashcard",
		Description: "A flashcard app that lives in your macOS menu bar",
		Icon:        assets.AppIcon,
		Services:    []application.Service{application.NewService(services.New(lib, dataPath))},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(frontend.Assets)},
		Mac:         application.MacOptions{ActivationPolicy: application.ActivationPolicyAccessory},
	})

	helper.SetupTray(app)

	if err := runApp(app); err != nil {
		return fmt.Errorf("app stopped unexpectedly: %w", err)
	}

	return nil
}
