package helper

import "github.com/wailsapp/wails/v3/pkg/application"

type Dialogs struct {
	App  *application.App
	Tray *application.SystemTray
}

func (d *Dialogs) PickImportFile(title string) (string, error) {
	defer d.reopen()
	return d.App.Dialog.OpenFile().
		SetTitle(title).
		CanChooseFiles(true).
		AddFilter("Excel / CSV", "*.xlsx;*.xlsm;*.csv").
		PromptForSingleSelection()
}

func (d *Dialogs) PickExportFile(title, filename string) (string, error) {
	defer d.reopen()
	return d.App.Dialog.SaveFile().
		SetMessage(title).
		SetFilename(filename).
		AddFilter("CSV", "*.csv").
		PromptForSingleSelection()
}

func (d *Dialogs) reopen() {
	if d.Tray != nil {
		d.Tray.ShowWindow()
	}
}
