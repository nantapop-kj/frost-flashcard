package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

func LoadLibrary(path string) (*model.Library, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return model.NewLibrary(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read flashcard file %s: %w", path, err)
	}

	var probe struct {
		Decks json.RawMessage `json:"decks"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("flashcard file %s is not valid JSON: %w", path, err)
	}

	if probe.Decks == nil {
		var d model.Deck
		if err := json.Unmarshal(data, &d); err != nil {
			return nil, fmt.Errorf("flashcard file %s looks like the old single-deck format but could not be read: %w", path, err)
		}
		return model.LibraryFromDeck(d), nil
	}

	var lib model.Library
	if err := json.Unmarshal(data, &lib); err != nil {
		return nil, fmt.Errorf("flashcard file %s has decks in an unexpected shape: %w", path, err)
	}
	if len(lib.Decks) == 0 {
		return model.NewLibrary(), nil
	}

	return &lib, nil
}

func SaveLibrary(lib *model.Library, path string) error {
	return writeJSON(path, lib)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("convert flashcards to JSON: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create data folder for %s: %w", path, err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temporary file %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace %s with the new data (old data is unchanged): %w", path, err)
	}

	return nil
}
