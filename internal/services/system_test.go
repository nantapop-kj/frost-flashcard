package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nantapop-kj/frost-flashcard/internal/dto"
	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

const (
	testAppID    = "frostflashcard.app"
	testLegacyID = "dev.frostflashcard.app"
)

type fakePlatform struct {
	mu          sync.Mutex
	quits       int
	autostartOn bool
	disableErr  error
}

func (p *fakePlatform) Quit() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.quits++
}

func (p *fakePlatform) DisableAutostart() error {
	if p.disableErr != nil {
		return p.disableErr
	}
	p.autostartOn = false
	return nil
}

func (p *fakePlatform) quitCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.quits
}

type call struct {
	name string
	args []string
}

type fakeRunner struct {
	runs, starts []call
	runOut       []byte
	runErr       error
	startErr     error
}

func newTestSystem(t *testing.T, goos, exe string) (*System, *fakePlatform, *fakeRunner, string) {
	t.Helper()
	dataDir := filepath.Join(t.TempDir(), testAppID)
	path := filepath.Join(dataDir, "flashcards.json")
	cards := New(model.NewLibrary(), path, &fakePicker{})
	cards.today = func() model.Date { return model.NewDate(2026, 1, 1) }

	p := &fakePlatform{autostartOn: true}
	r := &fakeRunner{}
	s := NewSystem(cards, "1.2.3", p, testAppID, testLegacyID)
	s.goos = goos
	s.exe = func() (string, error) { return exe, nil }
	s.run = func(name string, args ...string) ([]byte, error) {
		r.runs = append(r.runs, call{name, args})
		return r.runOut, r.runErr
	}
	s.start = func(name string, args ...string) error {
		r.starts = append(r.starts, call{name, args})
		return r.startErr
	}
	s.uninstaller = func() (string, error) { return "", errors.New("not registered") }
	s.quitDelay = 0
	s.appsDir = filepath.Join(t.TempDir(), "Applications")
	return s, p, r, dataDir
}

func writeBundle(t *testing.T, bundle, id string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(bundle, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := "<plist><dict>\n\t<key>CFBundleIdentifier</key>\n\t<string>" + id + "</string>\n</dict></plist>"
	if err := os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
}

func waitForQuit(t *testing.T, p *fakePlatform) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for p.quitCount() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("app did not quit")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestAppInfo(t *testing.T) {
	tests := []struct {
		goos, exe  string
		canInstall bool
	}{
		{"darwin", "/Users/me/Downloads/Frost Flashcard.app/Contents/MacOS/frost-flashcard", true},
		{"darwin", "APPS/Frost Flashcard.app/Contents/MacOS/frost-flashcard", false},
		{"darwin", "/tmp/go-build/exe/frost-flashcard", false},
		{"windows", `C:\Program Files\Frost Flashcard\frost-flashcard.exe`, false},
	}
	for _, tt := range tests {
		s, _, _, _ := newTestSystem(t, tt.goos, "")
		exe := strings.Replace(tt.exe, "APPS", s.appsDir, 1)
		s.exe = func() (string, error) { return exe, nil }
		want := AppInfo{Version: "1.2.3", Platform: tt.goos, CanInstall: tt.canInstall}
		if got := s.AppInfo(); got != want {
			t.Errorf("%s: AppInfo = %+v, want %+v", tt.exe, got, want)
		}
	}
}

func TestAppInfoWhenExecutableUnknown(t *testing.T) {
	s, _, _, _ := newTestSystem(t, "darwin", "")
	s.exe = func() (string, error) { return "", errors.New("no exe") }
	if s.AppInfo().CanInstall {
		t.Error("CanInstall should be false when the executable path is unknown")
	}
}

func TestBundleFromExecutable(t *testing.T) {
	tests := map[string]string{
		"/Users/me/Downloads/Frost Flashcard.app/Contents/MacOS/frost-flashcard": "/Users/me/Downloads/Frost Flashcard.app",
		"/a/Outer.app/Contents/Resources/Inner.app/Contents/MacOS/x":             "/a/Outer.app/Contents/Resources/Inner.app",
		"/usr/local/bin/frost-flashcard":                                         "",
	}
	for exe, want := range tests {
		if got := bundleFromExecutable(exe); got != want {
			t.Errorf("bundleFromExecutable(%q) = %q, want %q", exe, got, want)
		}
	}
}

func TestExecutableResolvesPath(t *testing.T) {
	exe, err := executable()
	if err != nil || !filepath.IsAbs(exe) {
		t.Errorf("executable = %q, %v", exe, err)
	}
}

func TestInstallAppRejectsNonBundle(t *testing.T) {
	for _, tt := range []struct{ goos, exe string }{
		{"darwin", "/usr/local/bin/frost-flashcard"},
		{"windows", `C:\x\Frost.app\Contents\MacOS\x`},
	} {
		s, p, r, _ := newTestSystem(t, tt.goos, tt.exe)
		wantCode(t, s.InstallApp(), dto.CodeBadInput)
		if len(r.runs)+len(r.starts) != 0 || p.quitCount() != 0 {
			t.Errorf("%s: should not run commands or quit", tt.goos)
		}
	}
}

func TestInstallAppCopiesOpensAndQuits(t *testing.T) {
	s, p, r, _ := newTestSystem(t, "darwin", "/Volumes/Frost Flashcard/Frost Flashcard.app/Contents/MacOS/frost-flashcard")
	dst := filepath.Join(s.appsDir, "Frost Flashcard.app")
	writeBundle(t, dst, testAppID)

	if err := s.InstallApp(); err != nil {
		t.Fatal(err)
	}
	if len(r.runs) != 1 || r.runs[0].name != "ditto" || r.runs[0].args[1] != dst {
		t.Errorf("runs = %+v", r.runs)
	}
	if len(r.starts) != 1 || r.starts[0].name != "open" || strings.Join(r.starts[0].args, " ") != "-n "+dst {
		t.Errorf("starts = %+v", r.starts)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("old copy should be removed before ditto: %v", err)
	}
	waitForQuit(t, p)
}

func TestInstallAppRemoveOldCopyFails(t *testing.T) {
	s, _, r, _ := newTestSystem(t, "darwin", "/Volumes/X/Frost Flashcard.app/Contents/MacOS/x")
	writeBundle(t, filepath.Join(s.appsDir, "Frost Flashcard.app"), testAppID)
	os.Chmod(s.appsDir, 0o500)
	t.Cleanup(func() { os.Chmod(s.appsDir, 0o755) })

	wantCode(t, s.InstallApp(), dto.CodeWriteFailed)
	if len(r.runs) != 0 {
		t.Error("should not copy when the old version cannot be removed")
	}
}

func TestInstallAppRefusesToReplaceAnotherApp(t *testing.T) {
	for name, setup := range map[string]func(dst string){
		"different bundle id": func(dst string) { writeBundle(t, dst, "com.apple.Notes") },
		"no Info.plist":       func(dst string) { os.MkdirAll(filepath.Join(dst, "Contents"), 0o755) },
		"symlink":             func(dst string) { os.Symlink(t.TempDir(), dst) },
	} {
		s, p, r, _ := newTestSystem(t, "darwin", "/Volumes/X/Notes.app/Contents/MacOS/x")
		os.MkdirAll(s.appsDir, 0o755)
		dst := filepath.Join(s.appsDir, "Notes.app")
		setup(dst)

		wantCode(t, s.InstallApp(), dto.CodeBadInput)
		if _, err := os.Lstat(dst); err != nil {
			t.Errorf("%s: existing app was removed: %v", name, err)
		}
		if len(r.runs)+len(r.starts) != 0 || p.quitCount() != 0 {
			t.Errorf("%s: should not copy, open or quit", name)
		}
	}
}

func TestInstallAppRefusesWhenAlreadyInApplications(t *testing.T) {
	s, _, r, _ := newTestSystem(t, "darwin", "")
	bundle := filepath.Join(s.appsDir, "Frost Flashcard.app")
	writeBundle(t, bundle, testAppID)
	s.exe = func() (string, error) { return filepath.Join(bundle, "Contents", "MacOS", "x"), nil }

	wantCode(t, s.InstallApp(), dto.CodeBadInput)
	if _, err := os.Stat(bundle); err != nil {
		t.Errorf("running copy was removed: %v", err)
	}
	if len(r.runs) != 0 {
		t.Error("should not copy onto itself")
	}
}

func TestInstallAppCopyFails(t *testing.T) {
	s, p, r, _ := newTestSystem(t, "darwin", "/Volumes/X/Missing Test App.app/Contents/MacOS/x")
	r.runOut, r.runErr = []byte("ditto: no space\n"), errors.New("exit 1")

	err := s.InstallApp()
	wantCode(t, err, dto.CodeWriteFailed)
	if err.(*dto.Error).Detail != "ditto: no space" {
		t.Errorf("detail = %q", err.(*dto.Error).Detail)
	}
	if len(r.starts) != 0 || p.quitCount() != 0 {
		t.Error("should stop after a failed copy")
	}
}

func TestInstallAppOpenFails(t *testing.T) {
	s, p, r, _ := newTestSystem(t, "darwin", "/Volumes/X/Missing Test App.app/Contents/MacOS/x")
	r.startErr = errors.New("cannot open")

	wantCode(t, s.InstallApp(), dto.CodeWriteFailed)
	if p.quitCount() != 0 {
		t.Error("should not quit when the new copy did not open")
	}
}

func TestUninstallMacDeletesDataAndAppAndQuits(t *testing.T) {
	bundle := filepath.Join(t.TempDir(), "Frost Flashcard.app")
	exe := filepath.Join(bundle, "Contents", "MacOS", "frost-flashcard")
	writeBundle(t, bundle, testAppID)
	os.WriteFile(exe, nil, 0o755)
	s, p, r, dataDir := newTestSystem(t, "darwin", exe)
	if _, err := s.cards.AddCard("a", "b", ""); err != nil {
		t.Fatal(err)
	}

	if err := s.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Errorf("data folder still exists: %v", err)
	}
	if p.autostartOn {
		t.Error("launch at login should be turned off")
	}
	if _, err := os.Stat(bundle); !os.IsNotExist(err) {
		t.Errorf("app bundle still exists: %v", err)
	}
	if len(r.starts) != 0 {
		t.Errorf("starts = %+v, want none", r.starts)
	}
	waitForQuit(t, p)

	if _, err := s.cards.AddCard("c", "d", ""); !dto.IsCode(err, dto.CodeSaveFailed) {
		t.Errorf("saving after uninstall: err = %v", err)
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Error("saving after uninstall recreated the data folder")
	}
}

func TestUninstallMacRevealsAppWhenItCannotBeRemoved(t *testing.T) {
	parent := t.TempDir()
	bundle := filepath.Join(parent, "Frost Flashcard.app")
	exe := filepath.Join(bundle, "Contents", "MacOS", "frost-flashcard")
	writeBundle(t, bundle, testAppID)
	os.WriteFile(exe, nil, 0o755)
	os.Chmod(parent, 0o500)
	t.Cleanup(func() { os.Chmod(parent, 0o755) })

	s, p, r, _ := newTestSystem(t, "darwin", exe)
	if err := s.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if len(r.starts) != 1 || strings.Join(r.starts[0].args, " ") != "-R "+bundle {
		t.Errorf("starts = %+v", r.starts)
	}
	waitForQuit(t, p)
}

func TestUninstallMacKeepsBundleOfAnotherApp(t *testing.T) {
	bundle := filepath.Join(t.TempDir(), "Other.app")
	exe := filepath.Join(bundle, "Contents", "MacOS", "frost-flashcard")
	writeBundle(t, bundle, "com.example.other")
	os.WriteFile(exe, nil, 0o755)

	s, p, r, dataDir := newTestSystem(t, "darwin", exe)
	if err := s.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bundle); err != nil {
		t.Errorf("another app's bundle was removed: %v", err)
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Errorf("our data folder should still be removed: %v", err)
	}
	if len(r.starts) != 0 {
		t.Errorf("starts = %+v, want none", r.starts)
	}
	waitForQuit(t, p)
}

func TestUninstallMacWithoutBundle(t *testing.T) {
	s, p, r, _ := newTestSystem(t, "darwin", "/tmp/go-build/frost-flashcard")
	if err := s.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if len(r.starts) != 0 {
		t.Errorf("starts = %+v, want none", r.starts)
	}
	waitForQuit(t, p)
}

func TestUninstallWindowsRunsRegisteredUninstaller(t *testing.T) {
	appDir := t.TempDir()
	uninstaller := filepath.Join(appDir, "unins000.exe")
	os.WriteFile(uninstaller, nil, 0o644)

	s, p, r, _ := newTestSystem(t, "windows", filepath.Join(appDir, "frost-flashcard.exe"))
	s.uninstaller = func() (string, error) { return `"` + uninstaller + `"`, nil }
	if err := s.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if len(r.starts) != 1 || r.starts[0].name != uninstaller ||
		!strings.Contains(strings.Join(r.starts[0].args, " "), "/VERYSILENT") {
		t.Errorf("starts = %+v", r.starts)
	}
	waitForQuit(t, p)
}

func TestUninstallWindowsSkipsUninstallersThatAreNotOurs(t *testing.T) {
	appDir := t.TempDir()
	otherDir := t.TempDir()
	ownUninstaller := filepath.Join(appDir, "unins000.exe")
	otherUninstaller := filepath.Join(otherDir, "unins000.exe")
	os.WriteFile(ownUninstaller, nil, 0o644)
	os.WriteFile(otherUninstaller, nil, 0o644)
	os.WriteFile(filepath.Join(appDir, "evil.exe"), nil, 0o644)

	tests := map[string]struct {
		registered string
		err        error
	}{
		"not registered":          {err: errors.New("not registered")},
		"registered elsewhere":    {registered: `"` + otherUninstaller + `"`},
		"not an inno uninstaller": {registered: filepath.Join(appDir, "evil.exe")},
		"registered file missing": {registered: filepath.Join(appDir, "unins001.exe")},
		"unterminated quote":      {registered: `"` + ownUninstaller},
	}
	for name, tt := range tests {
		s, p, r, _ := newTestSystem(t, "windows", filepath.Join(appDir, "frost-flashcard.exe"))
		s.uninstaller = func() (string, error) { return tt.registered, tt.err }
		if err := s.Uninstall(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(r.starts) != 0 {
			t.Errorf("%s: starts = %+v, want none", name, r.starts)
		}
		waitForQuit(t, p)
	}
}

func TestUninstallWindowsWhenExecutableUnknown(t *testing.T) {
	s, p, r, _ := newTestSystem(t, "windows", "")
	s.exe = func() (string, error) { return "", errors.New("no exe") }
	s.uninstaller = func() (string, error) { return `C:\x\unins000.exe`, nil }
	if err := s.Uninstall(); err != nil || len(r.starts) != 0 {
		t.Errorf("err = %v starts = %+v", err, r.starts)
	}
	waitForQuit(t, p)
}

func TestOwnUninstallerMatchesWindowsPaths(t *testing.T) {
	if got := windowsDir(`C:\Program Files\Frost Flashcard\unins000.exe`); got != `C:\Program Files\Frost Flashcard` {
		t.Errorf("windowsDir = %q", got)
	}
	if got := windowsBase(`C:\Program Files\Frost Flashcard\UNINS000.EXE`); !innoUninstallerName.MatchString(got) {
		t.Errorf("windowsBase = %q should match the Inno name", got)
	}
	for _, bad := range []string{"unins.exe", "unins000.exe.bat", "xunins000.exe", "unins0000.exe"} {
		if innoUninstallerName.MatchString(bad) {
			t.Errorf("%q should not match", bad)
		}
	}
}

func TestUninstallRefusesUnexpectedFolder(t *testing.T) {
	s, p, _, _ := newTestSystem(t, "darwin", "")
	other := filepath.Join(t.TempDir(), "Documents")
	os.MkdirAll(other, 0o755)
	s.cards.path = filepath.Join(other, "flashcards.json")

	wantCode(t, s.Uninstall(), dto.CodeBadInput)
	if _, err := os.Stat(other); err != nil {
		t.Errorf("folder was touched: %v", err)
	}
	if !p.autostartOn || p.quitCount() != 0 {
		t.Error("nothing should change when the folder is unexpected")
	}
}

func TestUninstallKeepsDataWhenAutostartCannotBeDisabled(t *testing.T) {
	s, p, _, dataDir := newTestSystem(t, "darwin", "")
	s.cards.AddCard("a", "b", "")
	p.disableErr = errors.New("denied")

	wantCode(t, s.Uninstall(), dto.CodeWriteFailed)
	if _, err := os.Stat(dataDir); err != nil {
		t.Errorf("data folder removed: %v", err)
	}
	if _, err := s.cards.AddCard("c", "d", ""); err != nil {
		t.Errorf("saving should work again: %v", err)
	}
}

func TestUninstallReportsRemoveFailure(t *testing.T) {
	s, _, _, dataDir := newTestSystem(t, "darwin", "")
	s.cards.AddCard("a", "b", "")
	os.Chmod(filepath.Dir(dataDir), 0o500)
	t.Cleanup(func() { os.Chmod(filepath.Dir(dataDir), 0o755) })

	wantCode(t, s.Uninstall(), dto.CodeWriteFailed)
	if _, err := s.cards.AddCard("c", "d", ""); err != nil {
		t.Errorf("saving should work again: %v", err)
	}
}

func TestNewSystemDefaults(t *testing.T) {
	s := NewSystem(nil, "dev", &fakePlatform{}, testAppID)
	if s.quitDelay <= 0 || s.exe == nil || s.appsDir != "/Applications" {
		t.Errorf("defaults not set: %+v", s)
	}
	if out, err := s.run("true"); err != nil {
		t.Errorf("run: %q %v", out, err)
	}
	if err := s.start("true"); err != nil {
		t.Errorf("start: %v", err)
	}
}

func TestUninstallAcceptsLegacyDataFolder(t *testing.T) {
	s, p, _, _ := newTestSystem(t, "darwin", "")
	legacy := filepath.Join(t.TempDir(), testLegacyID)
	s.cards.path = filepath.Join(legacy, "flashcards.json")
	s.cards.AddCard("a", "b", "")

	if err := s.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy folder still exists: %v", err)
	}
	waitForQuit(t, p)
}

func TestExecutableReportsError(t *testing.T) {
	want := errors.New("no exe")
	orig := osExecutable
	osExecutable = func() (string, error) { return "", want }
	t.Cleanup(func() { osExecutable = orig })

	if exe, err := executable(); !errors.Is(err, want) || exe != "" {
		t.Errorf("executable = %q, %v; want error %v", exe, err, want)
	}
}

func TestIsOurBundleWithoutBundleID(t *testing.T) {
	bundle := filepath.Join(t.TempDir(), "Frost Flashcard.app")
	writeBundle(t, bundle, "")
	s := &System{}
	if s.isOurBundle(bundle) {
		t.Error("a System without a bundle ID should not claim any bundle")
	}
}

func TestWindowsDirWithoutSeparator(t *testing.T) {
	if got := windowsDir("unins000.exe"); got != "" {
		t.Errorf("windowsDir = %q, want empty", got)
	}
}
