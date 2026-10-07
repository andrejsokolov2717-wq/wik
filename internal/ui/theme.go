package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Тёмная «морская» палитра Sweepy: глубокий тёмно-синий фон,
// голубые акценты, приглушённые сине-серые поверхности.
// Без зелёного — только стихия моря.
var (
	// cBackground — почти чёрный navy: основа экрана.
	cBackground = color.NRGBA{R: 0x0A, G: 0x12, B: 0x20, A: 0xFF}
	// cSurface — карточки и панели: тёмно-синий на порядок светлее фона.
	cSurface = color.NRGBA{R: 0x12, G: 0x1D, B: 0x2E, A: 0xFF}
	// cOverlay — всплывающие слои (меню, диалоги).
	cOverlay = color.NRGBA{R: 0x16, G: 0x24, B: 0x38, A: 0xFF}
	// cPrimary — акцент: яркий морской голубой (кнопки, выделения, фокус).
	cPrimary = color.NRGBA{R: 0x3B, G: 0x9E, B: 0xF5, A: 0xFF}
	// cText — основной текст: холодный белый с синевой.
	cText = color.NRGBA{R: 0xEA, G: 0xF1, B: 0xFA, A: 0xFF}
	// cDisabled — вторичный текст: серо-голубые подписи.
	cDisabled = color.NRGBA{R: 0x7C, G: 0x8C, B: 0xA5, A: 0xFF}
	// cError — ошибка: тёплый коралловый, единственный «тёплый» сигнал.
	cError = color.NRGBA{R: 0xF2, G: 0x6B, B: 0x6B, A: 0xFF}
	// cSeparator — разделители: едва заметные.
	cSeparator = color.NRGBA{R: 0x1E, G: 0x2E, B: 0x45, A: 0x80}
)

// oceanTheme — кастомная тема Fyne в морской стилистике.
type oceanTheme struct{}

var _ fyne.Theme = (*oceanTheme)(nil)

func (t *oceanTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return cBackground
	case theme.ColorNameButton, theme.ColorNameInputBackground, theme.ColorNameMenuBackground,
		theme.ColorNameHeaderBackground:
		return cSurface
	case theme.ColorNameOverlayBackground:
		return cOverlay
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink,
		theme.ColorNameSelection, theme.ColorNamePlaceHolder:
		return cPrimary
	case theme.ColorNameForeground:
		return cText
	case theme.ColorNameDisabled:
		return cDisabled
	case theme.ColorNameError:
		return cError
	case theme.ColorNameSeparator:
		return cSeparator
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x3B, G: 0x9E, B: 0xF5, A: 0x22}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 0x2A, G: 0x3F, B: 0x5C, A: 0xCC}
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (t *oceanTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *oceanTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *oceanTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInputRadius, theme.SizeNameSelectionRadius:
		return 8
	case theme.SizeNameSeparatorThickness:
		return 1
	default:
		return theme.DefaultTheme().Size(name)
	}
}
