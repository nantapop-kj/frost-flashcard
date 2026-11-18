package helper

import (
	"bytes"
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
	SetupTray(app)

	if _, ok := app.Window.GetByName("main"); !ok {
		t.Error("SetupTray should create the main popup window")
	}
}
