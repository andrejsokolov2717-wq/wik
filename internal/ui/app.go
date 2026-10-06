package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"

	"sweepy/internal/assets"
	"sweepy/internal/i18n"
)

// Run запускает приложение Sweepy.
func Run() {
	a := app.NewWithID("com.sweepy.app")
	a.Settings().SetTheme(theme.DefaultTheme())
	a.SetIcon(assets.Icon)

	w := a.NewWindow(i18n.T("app.windowTitle"))
	w.Resize(fyne.NewSize(900, 660))
	w.SetMaster()
	w.SetIcon(assets.Icon)

	ctl, err := NewController(a, w)
	if err != nil {
		w.SetContent(fyne.NewContainerWithoutLayout())
		w.ShowAndRun()
		return
	}

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon(i18n.T("app.tabTidy"), theme.HomeIcon(), homeTab(ctl)),
		container.NewTabItemWithIcon(i18n.T("app.tabStats"), theme.StorageIcon(), statsTab(ctl)),
		container.NewTabItemWithIcon(i18n.T("app.tabSettings"), theme.SettingsIcon(), settingsTab(ctl)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	w.SetContent(tabs)

	// --- системный трей ---
	if desk, ok := a.(desktop.App); ok {
		menu := fyne.NewMenu("Sweepy",
			fyne.NewMenuItem(i18n.T("app.trayOpen"), func() { w.Show() }),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(i18n.T("app.trayQuit"), func() { a.Quit() }),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(assets.Icon)
	}

	// закрытие окна — в трей, а не в выход
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	// восстанавливаем сторож, если он был включён
	if ctl.WatcherEnabled() {
		_ = ctl.StartWatcher()
	}

	w.ShowAndRun()
}
