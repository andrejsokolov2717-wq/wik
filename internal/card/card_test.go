package card

import (
	"bytes"
	"image/png"
	"testing"
)

func TestRenderPNG(t *testing.T) {
	data := Data{
		Title:     "До → После",
		FilesText: "47 файлов разобрано",
		BytesText: "1.4 ГБ",
		Date:      "07.10.2026 15:04",
		Categories: []CategoryStat{
			{Name: "Документы", Count: 20},
			{Name: "Скриншоты", Count: 15},
			{Name: "Архивы", Count: 7},
			{Name: "Прочее", Count: 5},
		},
		Footer: "сделано в Sweepy — $0.99, без подписки",
	}
	pngBytes, err := RenderPNG(data)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("невалидный PNG: %v", err)
	}
	if img.Bounds().Dx() != Width || img.Bounds().Dy() != Height {
		t.Errorf("размер %v, ожидался %dx%d", img.Bounds(), Width, Height)
	}
}

func TestTopCategories(t *testing.T) {
	moves := map[string]int{
		"Документы": 3, "Скриншоты": 10, "Архивы": 1, "Прочее": 5,
	}
	top := TopCategories(moves, 2)
	if len(top) != 2 {
		t.Fatalf("ожидалось 2 категории, получено %d", len(top))
	}
	if top[0].Name != "Скриншоты" || top[1].Name != "Прочее" {
		t.Errorf("неверный порядок: %+v", top)
	}
}
