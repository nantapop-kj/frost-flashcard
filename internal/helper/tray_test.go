package helper

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type fakeTray struct{ template, color []byte }

func (f *fakeTray) SetTemplateIcon(b []byte) *application.SystemTray { f.template = b; return nil }
func (f *fakeTray) SetIcon(b []byte) *application.SystemTray         { f.color = b; return nil }

func TestSetTrayIcon(t *testing.T) {
	template, color := []byte("template"), []byte("color")
	tests := []struct {
		goos         string
		wantTemplate bool
	}{
		{"darwin", true},
		{"windows", false},
		{"linux", false},
	}
	for _, tt := range tests {
		f := &fakeTray{}
		SetTrayIcon(f, tt.goos, template, color)
		if tt.wantTemplate && (!bytes.Equal(f.template, template) || f.color != nil) {
			t.Errorf("%s: want template icon only, got %+v", tt.goos, f)
		}
		if !tt.wantTemplate && (!bytes.Equal(f.color, color) || f.template != nil) {
			t.Errorf("%s: want color icon only, got %+v", tt.goos, f)
		}
	}
}

func TestOnClickRunsAction(t *testing.T) {
	calls := 0
	OnClick(func() { calls++ })(nil)
	if calls != 1 {
		t.Errorf("action ran %d times, want 1", calls)
	}
}

func TestSetupTrayBuildsWindowAndTray(t *testing.T) {
	app := application.New(application.Options{Name: "test"})
	tray := SetupTray(app, t.TempDir())

	if tray == nil {
		t.Error("SetupTray should return the tray")
	}
	if _, ok := app.Window.GetByName("main"); !ok {
		t.Error("SetupTray should create the main popup window")
	}
}

type fakeAutostart struct {
	enabled, disabled int
	err               error
}

func (f *fakeAutostart) Enable() error  { f.enabled++; return f.err }
func (f *fakeAutostart) Disable() error { f.disabled++; return f.err }

func TestSetAutostart(t *testing.T) {
	f := &fakeAutostart{}
	_ = SetAutostart(f, true)
	_ = SetAutostart(f, false)
	_ = SetAutostart(f, false)
	if f.enabled != 1 || f.disabled != 2 {
		t.Errorf("got enable=%d disable=%d, want 1 and 2", f.enabled, f.disabled)
	}
}

type fakeCheckItem struct{ checked bool }

func (f *fakeCheckItem) Checked() bool { return f.checked }
func (f *fakeCheckItem) SetChecked(on bool) *application.MenuItem {
	f.checked = on
	return nil
}

func TestToggleAutostart(t *testing.T) {
	a := &fakeAutostart{}
	item := &fakeCheckItem{checked: true}
	if err := ToggleAutostart(a, item); err != nil || a.enabled != 1 || !item.checked {
		t.Errorf("enable: err=%v %+v item=%+v", err, a, item)
	}

	item.checked = false
	if err := ToggleAutostart(a, item); err != nil || a.disabled != 1 || item.checked {
		t.Errorf("disable: err=%v %+v item=%+v", err, a, item)
	}
}

func TestToggleAutostartRevertsCheckboxOnError(t *testing.T) {
	a := &fakeAutostart{err: errors.New("denied")}
	item := &fakeCheckItem{checked: true}
	if err := ToggleAutostart(a, item); err == nil || item.checked {
		t.Errorf("err=%v checked=%v, want error and unchecked", err, item.checked)
	}
}

func TestOpenDataFolderCreatesThenOpens(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b")
	var opened string
	if err := OpenDataFolder(dir, func(p string) error { opened = p; return nil }); err != nil {
		t.Fatal(err)
	}
	if opened != dir {
		t.Errorf("opened %q, want %q", opened, dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("folder not created: %v", err)
	}
}

func TestOpenDataFolderErrors(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	called := false
	if err := OpenDataFolder(filepath.Join(file, "sub"), func(string) error { called = true; return nil }); err == nil || called {
		t.Errorf("err=%v called=%v, want mkdir error and no open", err, called)
	}

	want := errors.New("no finder")
	if err := OpenDataFolder(t.TempDir(), func(string) error { return want }); !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
}

type fakeHider struct{ hidden int }

func (f *fakeHider) Hide() application.Window { f.hidden++; return nil }

func TestHideOnCloseHidesAndCancels(t *testing.T) {
	w := &fakeHider{}
	e := application.NewWindowEvent()
	hideOnClose(w)(e)
	if w.hidden != 1 || !e.IsCancelled() {
		t.Errorf("hidden=%d cancelled=%v, want 1 and true", w.hidden, e.IsCancelled())
	}
}

type emitted struct {
	name string
	data []any
}

type fakeEmitter struct{ events []emitted }

func (f *fakeEmitter) emit(name string, data ...any) bool {
	f.events = append(f.events, emitted{name, data})
	return false
}

func TestEmitOnShowSendsPopupShown(t *testing.T) {
	f := &fakeEmitter{}
	emitOnShow(f.emit)(application.NewWindowEvent())
	if len(f.events) != 1 || f.events[0].name != PopupShownEvent {
		t.Errorf("events = %+v, want one %q", f.events, PopupShownEvent)
	}
}

type fakeChecker struct {
	enabled bool
	err     error
}

func (f fakeChecker) IsEnabled() (bool, error) { return f.enabled, f.err }

func TestAutostartEnabled(t *testing.T) {
	if !autostartEnabled(fakeChecker{enabled: true}) {
		t.Error("want enabled")
	}
	if autostartEnabled(fakeChecker{err: errors.New("denied")}) {
		t.Error("an error should read as disabled")
	}
}

func TestTrayActionsOpenViewNavigatesThenShows(t *testing.T) {
	f := &fakeEmitter{}
	var order []string
	a := &trayActions{
		emit: func(name string, data ...any) bool {
			order = append(order, "emit")
			return f.emit(name, data...)
		},
		show: func() { order = append(order, "show") },
	}

	a.openView("review")()
	if len(f.events) != 1 || f.events[0].name != NavigateEvent || !slices.Equal(f.events[0].data, []any{"review"}) {
		t.Errorf("events = %+v, want navigate to review", f.events)
	}
	if !slices.Equal(order, []string{"emit", "show"}) {
		t.Errorf("order = %v, want emit then show", order)
	}
}

func TestTrayActionsToggleAutostart(t *testing.T) {
	a := &trayActions{autostart: &fakeAutostart{}, autostartItem: &fakeCheckItem{checked: true}}
	a.toggleAutostart()
	if a.autostart.(*fakeAutostart).enabled != 1 {
		t.Errorf("autostart = %+v, want enabled once", a.autostart)
	}

	item := &fakeCheckItem{checked: true}
	a = &trayActions{autostart: &fakeAutostart{err: errors.New("denied")}, autostartItem: item}
	a.toggleAutostart()
	if item.checked {
		t.Error("checkbox should revert when autostart cannot be changed")
	}
}

func TestTrayActionsOpenDataFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	var opened string
	a := &trayActions{dataDir: dir, openFile: func(p string) error { opened = p; return nil }}
	a.openDataFolder()
	if opened != dir {
		t.Errorf("opened %q, want %q", opened, dir)
	}

	a.openFile = func(string) error { return errors.New("no finder") }
	a.openDataFolder()
}
