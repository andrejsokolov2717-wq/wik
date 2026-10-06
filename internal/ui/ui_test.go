package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

// TestBuildsSmoke: все три вкладки строятся без паники (headless-драйвер).
func TestBuildsSmoke(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	w := a.NewWindow("test")

	ctl, err := NewController(a, w)
	if err != nil {
		t.Fatal(err)
	}
	tabs := container.NewAppTabs(
		container.NewTabItem("Порядок", homeTab(ctl)),
		container.NewTabItem("Статистика", statsTab(ctl)),
		container.NewTabItem("Настройки", settingsTab(ctl)),
	)
	w.SetContent(tabs)
	w.Resize(fyne.NewSize(860, 640))

	// категории по умолчанию подхватились
	if len(ctl.Categories()) < 5 {
		t.Errorf("ожидалось >=5 категорий, получено %d", len(ctl.Categories()))
	}
	// целевая папка по умолчанию — Загрузки
	if ctl.TargetDir() == "" {
		t.Error("TargetDir пуст")
	}
}
