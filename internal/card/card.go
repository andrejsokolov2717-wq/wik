// Package card рисует PNG-карточку «до/после», которой пользователь
// может поделиться: сколько файлов разобрано, куда они уехали, сколько
// места наведено в порядок. Текст рендерится встроенным шрифтом
// Noto Sans (SIL OFL) — внешние шрифты не нужны.
package card

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sort"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts
var fontsFS embed.FS

const (
	Width  = 1200
	Height = 675
)

// CategoryStat — строка разбивки «категория × N».
type CategoryStat struct {
	Name  string
	Count int
}

// Data — содержимое карточки. Все строки уже локализованы вызывающим кодом.
type Data struct {
	Title      string         // «До → После»
	FilesText  string         // «47 файлов разобрано»
	BytesText  string         // «1.4 ГБ»
	Date       string         // «07.10.2026»
	Categories []CategoryStat // топ категорий уборки
	Footer     string         // водяной знак
}

var (
	bgColor     = color.RGBA{0x16, 0x18, 0x1d, 0xff}
	panelColor  = color.RGBA{0x1f, 0x22, 0x29, 0xff}
	titleColor  = color.RGBA{0xf2, 0xf3, 0xf5, 0xff}
	mutedColor  = color.RGBA{0x9a, 0xa0, 0xaa, 0xff}
	accentColor = color.RGBA{0x6e, 0xa8, 0xfe, 0xff}
	chipPalette = []color.RGBA{
		{0x37, 0x42, 0x5a, 0xff},
		{0x2f, 0x4a, 0x3d, 0xff},
		{0x4a, 0x3d, 0x2f, 0xff},
		{0x43, 0x33, 0x52, 0xff},
		{0x52, 0x33, 0x38, 0xff},
		{0x2f, 0x45, 0x52, 0xff},
	}
)

type facePair struct {
	regular, bold font.Face
}

func mustFont(name string) *opentype.Font {
	data, err := fontsFS.ReadFile("fonts/" + name)
	if err != nil {
		panic(err)
	}
	f, err := opentype.Parse(data)
	if err != nil {
		panic(err)
	}
	return f
}

func newFace(f *opentype.Font, size float64) font.Face {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size: size, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}
	return face
}

// RenderPNG рисует карточку и возвращает её в виде PNG.
func RenderPNG(d Data) ([]byte, error) {
	reg := mustFont("NotoSans-Regular.ttf")
	bold := mustFont("NotoSans-Bold.ttf")

	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// левая акцентная полоса
	fillRect(img, 0, 0, 14, Height, accentColor)

	drawText(img, newFace(bold, 56), 64, 96, d.Title, titleColor)
	drawText(img, newFace(reg, 26), 64, 138, d.Date, mutedColor)

	// панель с главными цифрами
	fillRect(img, 64, 180, Width-64, 340, panelColor)
	drawText(img, newFace(bold, 54), 96, 262, d.FilesText, accentColor)
	drawText(img, newFace(reg, 40), 96, 318, d.BytesText, titleColor)

	// чипы категорий
	x, y := 64, 390
	chipFace := newFace(reg, 24)
	for i, cat := range d.Categories {
		if i >= 8 {
			break
		}
		label := cat.Name + " × " + itoa(cat.Count)
		w := textWidth(chipFace, label) + 40
		if x+w > Width-64 {
			x = 64
			y += 64
		}
		if y+48 > 596 {
			break
		}
		fillRect(img, x, y, x+w, y+48, chipPalette[i%len(chipPalette)])
		drawText(img, chipFace, x+20, y+33, label, titleColor)
		x += w + 16
	}

	drawText(img, newFace(reg, 22), 64, Height-32, d.Footer, mutedColor)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// TopCategories группирует перемещения уборки по категориям
// и возвращает топ-n по числу файлов.
func TopCategories(moves map[string]int, n int) []CategoryStat {
	out := make([]CategoryStat, 0, len(moves))
	for name, count := range moves {
		out = append(out, CategoryStat{Name: name, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{c}, image.Point{}, draw.Src)
}

func drawText(img *image.RGBA, face font.Face, x, y int, s string, c color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{c},
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

func textWidth(face font.Face, s string) int {
	d := &font.Drawer{Face: face}
	return d.MeasureString(s).Ceil()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
