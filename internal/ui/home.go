package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"sweepy/internal/card"
	"sweepy/internal/core"
	"sweepy/internal/i18n"
)

// homeTab — главный экран: выбор папки, предпросмотр, одна большая кнопка.
func homeTab(c *Controller) fyne.CanvasObject {
	prefs := c.App.Preferences()

	title := widget.NewLabelWithStyle("Sweepy", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel(i18n.T("home.subtitle"))
	subtitle.Alignment = fyne.TextAlignCenter

	// --- выбор целевой папки ---
	optDownloads := i18n.T("home.downloads")
	optDesktop := i18n.T("home.desktop")
	optCustom := i18n.T("home.custom")

	customLabel := widget.NewLabel("")
	customLabel.Hide()
	customLabel.Wrapping = fyne.TextWrapWord
	customLabel.Truncation = fyne.TextTruncateEllipsis

	var targetRadio *widget.RadioGroup
	browseBtn := widget.NewButtonWithIcon(i18n.T("home.browse"), theme.FolderOpenIcon(), func() {
		dlg := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			c.SetTarget("custom", uri.Path())
			customLabel.SetText(uri.Path())
		}, c.Window)
		dlg.Show()
	})
	browseBtn.Hide()

	targetRadio = widget.NewRadioGroup([]string{optDownloads, optDesktop, optCustom}, func(s string) {
		mode := "downloads"
		switch s {
		case optDesktop:
			mode = "desktop"
		case optCustom:
			mode = "custom"
		}
		c.SetTarget(mode, prefs.String("custom_path"))
		if mode == "custom" {
			customLabel.Show()
			browseBtn.Show()
			p := prefs.String("custom_path")
			if p == "" {
				customLabel.SetText(i18n.T("home.noFolder"))
			} else {
				customLabel.SetText(p)
			}
		} else {
			customLabel.Hide()
			browseBtn.Hide()
		}
	})
	targetRadio.Horizontal = true
	switch prefs.StringWithFallback("target", "downloads") {
	case "desktop":
		targetRadio.SetSelected(optDesktop)
	case "custom":
		targetRadio.SetSelected(optCustom)
	default:
		targetRadio.SetSelected(optDownloads)
	}

	// --- предпросмотр плана ---
	var plan *core.Plan
	previewHeader := widget.NewLabelWithStyle(i18n.T("home.previewHeader"),
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	previewHeader.Hide()

	shorten := func(p string) string {
		if plan == nil || plan.Root == "" {
			return p
		}
		return strings.TrimPrefix(p, plan.Root+string(filepath.Separator))
	}

	table := widget.NewTable(
		func() (int, int) {
			if plan == nil {
				return 0, 0
			}
			return len(plan.Moves), 3
		},
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			lbl := cell.(*widget.Label)
			m := plan.Moves[id.Row]
			switch id.Col {
			case 0:
				lbl.SetText(shorten(m.From))
			case 1:
				lbl.SetText("→")
			case 2:
				lbl.SetText(shorten(m.To))
			}
		},
	)
	table.SetColumnWidth(0, 300)
	table.SetColumnWidth(1, 24)
	table.SetColumnWidth(2, 340)
	table.Hide()

	status := widget.NewLabel(i18n.T("home.statusStart"))
	status.Wrapping = fyne.TextWrapWord

	// --- кнопки действий ---
	var scanBtn, tidyBtn, undoBtn, undoMoreBtn, shareBtn *widget.Button

	setUndoState := func() {
		n := c.Journal.ActiveCount()
		if n > 0 {
			undoBtn.Enable()
			undoMoreBtn.Enable()
		} else {
			undoBtn.Disable()
			undoMoreBtn.Disable()
		}
	}

	// runTidy выполняет план в фоне и обновляет статус в потоке UI.
	runTidy := func(planned *core.Plan) {
		tidyBtn.Disable()
		scanBtn.Disable()
		status.SetText(i18n.T("home.working"))
		go func() {
			done, err := c.Mover.Execute(planned)
			fyne.Do(func() {
				scanBtn.Enable()
				if err != nil {
					dialog.ShowError(err, c.Window)
				}
				if done > 0 {
					status.SetText(i18n.T("home.done", done, i18n.Plural(done, "file"),
						core.FormatBytes(planned.TotalBytes())))
					shareBtn.Enable()
				} else {
					status.SetText(i18n.T("home.statusStart"))
				}
				plan = nil
				table.Hide()
				previewHeader.Hide()
				table.Refresh()
				setUndoState()
			})
		}()
	}

	tidyBtn = widget.NewButtonWithIcon(i18n.T("home.tidy"), theme.ConfirmIcon(), func() {
		if plan == nil || len(plan.Moves) == 0 {
			return
		}
		planned := plan
		dialog.ShowConfirm(
			i18n.T("home.confirmMoveTitle", len(planned.Moves), i18n.Plural(len(planned.Moves), "file")),
			i18n.T("home.confirmMoveBody"),
			func(ok bool) {
				if ok {
					runTidy(planned)
				}
			}, c.Window)
	})
	tidyBtn.Importance = widget.HighImportance
	tidyBtn.Disable()

	// undoSession — откат конкретной сессии с обновлением статуса.
	undoSession := func(sess *core.Session) {
		go func() {
			restored, missing, err := c.Mover.Undo(sess)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, c.Window)
				}
				msg := i18n.T("home.restored", restored, i18n.Plural(restored, "file"))
				if missing > 0 {
					msg += i18n.T("home.restoredMissing", missing)
				}
				status.SetText(msg)
				setUndoState()
			})
		}()
	}

	confirmUndo := func(sess *core.Session) {
		dialog.ShowConfirm(i18n.T("home.undoConfirmTitle"),
			i18n.T("home.undoConfirmBody", len(sess.Moves), i18n.Plural(len(sess.Moves), "file"),
				sess.StartedAt.Format("02.01.2006 15:04")),
			func(ok bool) {
				if ok {
					undoSession(sess)
				}
			}, c.Window)
	}

	undoBtn = widget.NewButtonWithIcon(i18n.T("home.undo"), theme.ContentUndoIcon(), func() {
		sess := c.Journal.LastActive()
		if sess == nil {
			status.SetText(i18n.T("home.nothingToUndo"))
			return
		}
		confirmUndo(sess)
	})

	// undoMoreBtn — выбор конкретной уборки из журнала: обещание README
	// про откат любой прошлой сессии теперь доступно прямо в UI.
	undoMoreBtn = widget.NewButtonWithIcon(i18n.T("home.undoPick"), theme.ListIcon(), func() {
		sessions := c.RecentSessions(10)
		if len(sessions) == 0 {
			status.SetText(i18n.T("home.nothingToUndo"))
			return
		}
		list := widget.NewList(
			func() int { return len(sessions) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				s := sessions[id]
				obj.(*widget.Label).SetText(i18n.T("home.undoEntry",
					s.StartedAt.Format("02.01.2006 15:04"),
					len(s.Moves), i18n.Plural(len(s.Moves), "file")))
			},
		)
		d := dialog.NewCustom(i18n.T("home.undoPickTitle"),
			widget.NewButtonWithIcon(i18n.T("home.cancel"), theme.CancelIcon(), func() { d.Hide() }),
			container.NewPadded(list), c.Window)
		list.OnSelected = func(id widget.ListItemID) {
			d.Hide()
			confirmUndo(sessions[id])
		}
		d.Resize(fyne.NewSize(420, 380))
		d.Show()
	})

	openFolderBtn := widget.NewButtonWithIcon(i18n.T("home.openFolder"), theme.FolderOpenIcon(), func() {
		if err := c.OpenTarget(); err != nil {
			status.SetText(i18n.T("home.openFolderError", err))
		}
	})

	shareBtn = widget.NewButtonWithIcon(i18n.T("home.shareCard"), theme.MediaPhotoIcon(), func() {
		sess := c.Journal.LastActive()
		if sess == nil || len(sess.Moves) == 0 {
			status.SetText(i18n.T("home.nothingToUndo"))
			return
		}
		byCat := map[string]int{}
		var bytes int64
		for _, mv := range sess.Moves {
			byCat[mv.Category]++
			bytes += mv.Size
		}
		data := card.Data{
			Title:      i18n.T("card.title"),
			FilesText:  i18n.T("card.files", len(sess.Moves), i18n.Plural(len(sess.Moves), "file")),
			BytesText:  core.FormatBytes(bytes),
			Date:       sess.StartedAt.Format("02.01.2006 15:04"),
			Categories: card.TopCategories(byCat, 8),
			Footer:     i18n.T("card.footer"),
		}
		png, err := card.RenderPNG(data)
		if err != nil {
			dialog.ShowError(fmt.Errorf("%s", i18n.T("home.cardError", err)), c.Window)
			return
		}
		save := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil || wc == nil {
				return
			}
			defer wc.Close()
			if _, err := wc.Write(png); err != nil {
				dialog.ShowError(err, c.Window)
				return
			}
			status.SetText(i18n.T("home.cardSaved", wc.URI().Path()))
		}, c.Window)
		save.SetFileName("sweepy-before-after.png")
		save.SetFilter(storage.NewExtensionFileFilter([]string{".png"}))
		save.Show()
	})
	shareBtn.Disable()

	scanBtn = widget.NewButtonWithIcon(i18n.T("home.scan"), theme.ViewRefreshIcon(), func() {
		dir := c.TargetDir()
		scanBtn.Disable()
		status.SetText(i18n.T("home.scanning"))
		go func() {
			files, err := c.Scanner.Scan(dir)
			fyne.Do(func() {
				scanBtn.Enable()
				if err != nil {
					dialog.ShowError(fmt.Errorf("%s", i18n.T("home.scanError", dir, err)), c.Window)
					status.SetText(i18n.T("home.statusStart"))
					return
				}
				plan = core.BuildPlan(dir, files, c.Categories())
				table.Refresh()
				if len(plan.Moves) == 0 {
					table.Hide()
					previewHeader.Hide()
					tidyBtn.Disable()
					status.SetText(i18n.T("home.alreadyClean", dir))
					return
				}
				previewHeader.Show()
				table.Show()
				tidyBtn.Enable()
				status.SetText(i18n.T("home.found", len(plan.Moves),
					i18n.Plural(len(plan.Moves), "file"), core.FormatBytes(plan.TotalBytes())))
			})
		}()
	})

	targetBox := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("home.whatToTidy"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, openFolderBtn, targetRadio),
		container.NewBorder(nil, nil, nil, browseBtn, customLabel),
	)
	actions := container.NewHBox(scanBtn, undoBtn, undoMoreBtn, shareBtn)

	setUndoState()

	return container.NewBorder(
		container.NewPadded(container.NewVBox(title, subtitle, targetBox, actions, tidyBtn, status, previewHeader)),
		nil, nil, nil,
		container.NewPadded(table),
	)
}
