package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"sweepy/internal/autostart"
	"sweepy/internal/build"
	"sweepy/internal/core"
	"sweepy/internal/i18n"
)

// settingsTab — настройки: язык, автозапуск, сторож папки, редактор категорий.
func settingsTab(c *Controller) fyne.CanvasObject {
	// --- язык ---
	langSelect := widget.NewSelect([]string{"Русский", "English"}, nil)
	if c.Language() == i18n.EN {
		langSelect.SetSelected("English")
	} else {
		langSelect.SetSelected("Русский")
	}
	// колбэк назначаем после SetSelected, чтобы не всплывал диалог при открытии
	langSelect.OnChanged = func(s string) {
		want := i18n.RU
		if s == "English" {
			want = i18n.EN
		}
		if want == c.Language() {
			return
		}
		c.SetLanguage(want)
		dialog.ShowInformation(i18n.T("settings.language"),
			i18n.T("settings.langRestart"), c.Window)
	}

	// --- автозапуск ---
	autostartCheck := widget.NewCheck(i18n.T("settings.autostart"), nil)
	autostartCheck.SetChecked(autostart.IsEnabled())
	autostartCheck.OnChanged = func(on bool) {
		var err error
		if on {
			err = autostart.Enable()
		} else {
			err = autostart.Disable()
		}
		if err != nil {
			dialog.ShowError(fmt.Errorf("%s", i18n.T("settings.autostartError", err)), c.Window)
			autostartCheck.SetChecked(!on)
		}
	}

	// --- фоновый сторож ---
	watchCheck := widget.NewCheck(i18n.T("settings.watch"), nil)
	watchCheck.SetChecked(c.WatcherEnabled())
	watchCheck.OnChanged = func(on bool) {
		if on {
			if err := c.StartWatcher(); err != nil {
				dialog.ShowError(fmt.Errorf("%s", i18n.T("settings.watchError", err)), c.Window)
				watchCheck.SetChecked(false)
			}
		} else {
			c.StopWatcher()
		}
	}

	// --- о программе ---
	aboutBtn := widget.NewButtonWithIcon(i18n.T("settings.about"), theme.InfoIcon(), func() {
		dialog.ShowInformation(i18n.T("settings.aboutTitle"),
			i18n.T("settings.aboutBody", build.Version), c.Window)
	})

	// --- редактор категорий ---
	rows := container.NewVBox()
	cats := append([]core.Category(nil), c.Categories()...)

	var rebuild func()
	rebuild = func() {
		rows.Objects = nil
		for i := range cats {
			idx := i
			name := widget.NewEntry()
			name.SetText(cats[idx].Name)
			name.OnChanged = func(s string) { cats[idx].Name = s }

			ext := widget.NewEntry()
			ext.SetText(strings.Join(cats[idx].Extensions, ", "))
			ext.SetPlaceHolder(i18n.T("settings.extPlaceholder"))
			ext.OnChanged = func(s string) { cats[idx].Extensions = splitList(s) }

			pat := widget.NewEntry()
			pat.SetText(strings.Join(cats[idx].Patterns, ", "))
			pat.SetPlaceHolder(i18n.T("settings.patPlaceholder"))
			pat.OnChanged = func(s string) { cats[idx].Patterns = splitList(s) }

			del := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				if len(cats) <= 1 {
					return
				}
				cats = append(cats[:idx], cats[idx+1:]...)
				rebuild()
			})

			row := container.NewGridWithColumns(4, name, ext, pat, del)
			rows.Add(row)
		}
		rows.Refresh()
	}
	rebuild()

	addBtn := widget.NewButtonWithIcon(i18n.T("settings.addCategory"), theme.ContentAddIcon(), func() {
		cats = append(cats, core.Category{Name: i18n.T("settings.newCategory")})
		rebuild()
	})

	saveBtn := widget.NewButtonWithIcon(i18n.T("settings.save"), theme.DocumentSaveIcon(), func() {
		// отбрасываем пустые категории
		var clean []core.Category
		for _, cat := range cats {
			if strings.TrimSpace(cat.Name) == "" {
				continue
			}
			clean = append(clean, cat)
		}
		if len(clean) == 0 {
			dialog.ShowInformation(i18n.T("settings.title"), i18n.T("settings.needCategory"), c.Window)
			return
		}
		c.SetCategories(clean)
		dialog.ShowInformation(i18n.T("settings.title"), i18n.T("settings.saved"), c.Window)
	})

	resetBtn := widget.NewButtonWithIcon(i18n.T("settings.reset"), theme.ViewRestoreIcon(), func() {
		dialog.ShowConfirm(i18n.T("settings.resetTitle"), i18n.T("settings.resetBody"), func(ok bool) {
			if !ok {
				return
			}
			cats = core.DefaultCategories()
			c.SetCategories(cats)
			rebuild()
		}, c.Window)
	})

	header := container.NewGridWithColumns(4,
		widget.NewLabelWithStyle(i18n.T("settings.colFolder"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle(i18n.T("settings.colExt"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle(i18n.T("settings.colWords"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(""),
	)

	return container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("settings.title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, widget.NewLabel(i18n.T("settings.language")), aboutBtn, langSelect),
		widget.NewSeparator(),
		widget.NewLabelWithStyle(i18n.T("settings.automation"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		autostartCheck,
		watchCheck,
		widget.NewSeparator(),
		widget.NewLabelWithStyle(i18n.T("settings.categories"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		header,
		rows,
		container.NewHBox(addBtn, saveBtn, resetBtn),
	))
}

// splitList разбивает строку «a, b, c» в список.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
