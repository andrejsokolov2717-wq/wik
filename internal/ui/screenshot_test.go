package ui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

// TestScreenshots рендерит вкладки приложения в PNG, когда задан
// SWEEPY_SHOT_DIR. Используется для подготовки релизных материалов:
//
//	SWEEPY_SHOT_DIR=./shots go test ./internal/ui -run TestScreenshots
func TestScreenshots(t *testing.T) {
	dir := os.Getenv("SWEEPY_SHOT_DIR")
	if dir == "" {
		t.Skip("SWEEPY_SHOT_DIR не задан")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	a := test.NewApp()
	defer a.Quit()
	w := a.NewWindow("Sweepy")

	ctl, err := NewController(a, w)
	if err != nil {
		t.Fatal(err)
	}

	tabs := container.NewAppTabs(
		container.NewTabItem("Порядок", homeTab(ctl)),
		container.NewTabItem("Статистика", statsTab(ctl)),
		container.NewTabItem("Настройки", settingsTab(ctl)),
	)
	names := []string{"home", "stats", "settings"}
	for i, name := range names {
		tabs.SelectIndex(i)
		w.SetContent(tabs)
		w.Resize(fyne.NewSize(1000, 700))
		w.Show()
		img := w.Canvas().Capture()
		f, err := os.Create(filepath.Join(dir, "sweepy-"+name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
	}
}
