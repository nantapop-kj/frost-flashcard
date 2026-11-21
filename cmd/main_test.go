package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nantapop-kj/frost-flashcard/internal/helper"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func setHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("AppData", "")
	if home != "" {
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		t.Setenv("AppData", filepath.Join(home, "AppData"))
	}
}

func stubRunApp(t *testing.T, fn func(*application.App) error) {
	t.Helper()
	orig := runApp
	runApp = fn
	t.Cleanup(func() { runApp = orig })
}

func stubFatal(t *testing.T) *[]any {
	t.Helper()
	var got []any
	orig := fatal
	fatal = func(v ...any) { got = append(got, v...) }
	t.Cleanup(func() { fatal = orig })
	return &got
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("err = %v, want it to contain %q", err, want)
	}
}

func TestRunFailsWithoutHome(t *testing.T) {
	setHome(t, "")
	assertErrorContains(t, run(), "cannot find a folder to store flashcards")
}

func TestRunFailsOnBrokenDataFile(t *testing.T) {
	setHome(t, t.TempDir())
	path, err := helper.DataFile(appID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	assertErrorContains(t, run(), "cannot load flashcards")
}

func TestRunBuildsAppAndReportsRunError(t *testing.T) {
	setHome(t, t.TempDir())
	var started *application.App
	stubRunApp(t, func(app *application.App) error {
		started = app
		return errors.New("boom")
	})

	assertErrorContains(t, run(), "app stopped unexpectedly: boom")
	if started == nil {
		t.Error("run should hand the built app to runApp")
	}

	stubRunApp(t, func(*application.App) error { return nil })
	if err := run(); err != nil {
		t.Errorf("run = %v, want nil when the app exits cleanly", err)
	}
}

func TestMainCallsFatalOnError(t *testing.T) {
	setHome(t, "")
	got := stubFatal(t)

	main()

	if len(*got) != 1 {
		t.Fatalf("fatal called with %v, want one error", *got)
	}
	if err, ok := (*got)[0].(error); !ok || !strings.Contains(err.Error(), "cannot find a folder") {
		t.Errorf("fatal got %v", (*got)[0])
	}
}

func TestMainDoesNotCallFatalOnSuccess(t *testing.T) {
	setHome(t, t.TempDir())
	stubRunApp(t, func(*application.App) error { return nil })
	got := stubFatal(t)

	main()

	if len(*got) != 0 {
		t.Errorf("fatal called with %v, want no call", *got)
	}
}
