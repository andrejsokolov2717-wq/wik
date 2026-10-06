// Package assets содержит встроенные ресурсы приложения.
package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconPNG []byte

// Icon — иконка приложения (окно, трей, пакет).
var Icon = fyne.NewStaticResource("icon.png", iconPNG)
