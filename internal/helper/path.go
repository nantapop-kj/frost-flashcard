package helper

import (
	"os"
	"path/filepath"
)

func DataFile(appID string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appID, "flashcards.json"), nil
}
