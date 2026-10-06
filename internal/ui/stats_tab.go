package ui

import (
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"sweepy/internal/core"
	"sweepy/internal/i18n"
)

// statCard — большая цифра с подписью; возвращает значение-лейбл для обновления.
func statCard(caption string) (*widget.Label, fyne.CanvasObject) {
	v := widget.NewLabelWithStyle("0", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	c := widget.NewLabelWithStyle(caption, fyne.TextAlignCenter, fyne.TextStyle{})
	return v, container.NewVBox(v, c)
}

// statsTab — экран накопительной статистики.
func statsTab(c *Controller) fyne.CanvasObject {
	movedVal, movedCard := statCard(i18n.T("stats.filesMoved"))
	savedVal, savedCard := statCard(i18n.T("stats.bytesSaved"))
	streakVal, streakCard := statCard(i18n.T("stats.streak"))
	daysVal, daysCard := statCard(i18n.T("stats.days"))

	history := widget.NewLabel("")
	history.Wrapping = fyne.TextWrapWord

	refreshStats := func() {
		snap := c.Stats.Snapshot()
		movedVal.SetText(itoa(snap.FilesMoved))
		savedVal.SetText(core.FormatBytes(snap.BytesSorted))
		streakVal.SetText(itoa(snap.Streak))
		daysVal.SetText(itoa(snap.ActiveDays))
		history.SetText(historyText(snap))
	}
	refreshStats()

	refreshBtn := widget.NewButtonWithIcon(i18n.T("stats.refresh"), theme.ViewRefreshIcon(), refreshStats)

	return container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("stats.title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, movedCard, savedCard),
		container.NewGridWithColumns(2, streakCard, daysCard),
		widget.NewSeparator(),
		widget.NewLabelWithStyle(i18n.T("stats.history"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		history,
		refreshBtn,
	)
}

// historyText — последние 14 дней уборок.
func historyText(snap core.Snapshot) string {
	if len(snap.Days) == 0 {
		return i18n.T("stats.empty")
	}
	days := make([]string, 0, len(snap.Days))
	for d := range snap.Days {
		days = append(days, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(days)))
	if len(days) > 14 {
		days = days[:14]
	}
	out := ""
	for _, d := range days {
		n := snap.Days[d]
		out += i18n.T("stats.dayLine", d, n, i18n.Plural(n, "file")) + "\n"
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
