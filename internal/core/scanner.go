package core

import (
	"os"
	"path/filepath"
	"sort"
)

// FileInfo — один найденный файл верхнего уровня целевой папки.
type FileInfo struct {
	Path string // полный путь
	Name string // имя файла
	Size int64  // размер в байтах
}

// Scanner сканирует каталоги, не заходя в подпапки.
type Scanner struct {
	// SkipNames — имена, которые никогда не трогаем (ярлыки, системные файлы).
	SkipNames map[string]bool
}

// NewScanner возвращает сканер с безопасным списком исключений.
func NewScanner() *Scanner {
	return &Scanner{SkipNames: map[string]bool{
		"desktop.ini": true,
		".ds_store":   true,
		"thumbs.db":   true,
		".localized":  true,
		"sweepy.exe":  true,
		"sweepy":      true,
		"sweepy.app":  true,
	}}
}

// Scan возвращает файлы верхнего уровня каталога dir (без подпапок).
// Категориальные папки (Документы, Изображения и т.д.) пропускаются —
// Sweepy никогда не заходит внутрь уже разложенного.
func (s *Scanner) Scan(dir string) ([]FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []FileInfo
	for _, e := range entries {
		if e.IsDir() || !e.Type().IsRegular() {
			continue
		}
		name := e.Name()
		if s.SkipNames[name] || len(name) == 0 || name[0] == '.' {
			continue // скрытые и системные — не трогаем
		}
		if filepath.Ext(name) == ".lnk" || filepath.Ext(name) == ".url" {
			continue // ярлыки Windows остаются на месте
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, FileInfo{
			Path: filepath.Join(dir, name),
			Name: name,
			Size: info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
