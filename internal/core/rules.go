package core

import (
	"path/filepath"
	"strings"
)

// Category описывает целевую папку и правила отбора файлов в неё.
type Category struct {
	Name       string   `json:"name"`       // имя подпапки, например "Документы"
	Extensions []string `json:"extensions"` // расширения с точкой: ".pdf"
	Patterns   []string `json:"patterns"`   // подстроки имени (в нижнем регистре): "screenshot"
}

// Matches сообщает, подходит ли файл под категорию.
// Скриншот-паттерны проверяются до расширений вызывающим кодом —
// для этого категории должны быть отсортированы по приоритету.
func (c Category) Matches(fileName string) bool {
	lower := strings.ToLower(fileName)
	for _, p := range c.Patterns {
		if p != "" && strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		return false
	}
	for _, e := range c.Extensions {
		if strings.EqualFold(strings.TrimSpace(e), ext) {
			return true
		}
	}
	return false
}

// DefaultCategories возвращает пресет «из коробки».
// Порядок важен: «Скриншоты» стоят раньше «Изображений»,
// чтобы screen-файлы не уезжали в общую папку картинок.
func DefaultCategories() []Category {
	return []Category{
		{Name: "Скриншоты",
			Extensions: []string{},
			Patterns:   []string{"screenshot", "screen shot", "снимок экрана", "скриншот"}},
		{Name: "Документы",
			Extensions: []string{".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".xls", ".xlsx", ".csv", ".ppt", ".pptx", ".md", ".epub", ".djvu"},
			Patterns:   []string{}},
		{Name: "Изображения",
			Extensions: []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico", ".tiff", ".heic", ".raw"},
			Patterns:   []string{}},
		{Name: "Архивы",
			Extensions: []string{".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz", ".iso"},
			Patterns:   []string{}},
		{Name: "Аудио",
			Extensions: []string{".mp3", ".wav", ".flac", ".ogg", ".m4a", ".aac", ".wma"},
			Patterns:   []string{}},
		{Name: "Видео",
			Extensions: []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".webm", ".m4v", ".flv"},
			Patterns:   []string{}},
		{Name: "Программы",
			Extensions: []string{".exe", ".msi", ".dmg", ".pkg", ".deb", ".rpm", ".appimage", ".apk"},
			Patterns:   []string{"setup", "installer"}},
		{Name: "Прочее",
			Extensions: []string{},
			Patterns:   []string{}},
	}
}

// MatchCategory возвращает категорию для файла.
// Если ни одно правило не сработало — возвращается «Прочее».
// Если и её нет в списке — nil (файл не трогаем).
func MatchCategory(fileName string, cats []Category) *Category {
	for i := range cats {
		if len(cats[i].Extensions) == 0 && len(cats[i].Patterns) == 0 {
			continue // «Прочее» проверяем в конце
		}
		if cats[i].Matches(fileName) {
			return &cats[i]
		}
	}
	for i := range cats {
		if len(cats[i].Extensions) == 0 && len(cats[i].Patterns) == 0 {
			return &cats[i]
		}
	}
	return nil
}
