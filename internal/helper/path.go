package helper

import (
	"log"
	"os"
	"path/filepath"
)

const dataFileName = "flashcards.json"

func DataFile(appID string, legacyIDs ...string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	current := filepath.Join(dir, appID)
	if exists(current) {
		return filepath.Join(current, dataFileName), nil
	}

	for _, id := range legacyIDs {
		legacy := filepath.Join(dir, id)
		if !exists(legacy) {
			continue
		}
		if err := os.Rename(legacy, current); err != nil {
			log.Printf("cannot move data from %s to %s, keep using the old folder: %v", legacy, current, err)
			return filepath.Join(legacy, dataFileName), nil
		}
		return filepath.Join(current, dataFileName), nil
	}

	return filepath.Join(current, dataFileName), nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
