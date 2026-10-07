package core

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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

// partialExt — расширения недописанных загрузок (браузеры, торренты,
// временные файлы Office). Sweepy их никогда не трогает: файл ещё пишется.
var partialExt = map[string]bool{
	".part": true, ".crdownload": true, ".download": true, ".tmp": true,
	".dmp": true, ".opdownload": true, ".folderdownload": true, ".td": true,
	".arj": true, // uTorrent
	".~df": true, // LibreOffice lock
}

// isPartialDownload сообщает, что файл — незавершённая загрузка или
// временный артефакт редактора («~file.docx», «file.pdf.crdownload»).
func isPartialDownload(name string) bool {
	if strings.HasPrefix(name, "~") || strings.HasPrefix(name, "~$") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	if partialExt[ext] {
		return true
	}
	// двойные расширения вида "archive.tar.gz.tmp" уже покрыты .tmp;
	// дополнительно отсекаем "*.partial"
	return ext == ".partial"
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
		if isPartialDownload(name) {
			continue // недописанные загрузки — не трогаем
		}
		if filepath.Ext(name) == ".lnk" || filepath.Ext(name) == ".url" {
			continue // ярлыки Windows остаются на месте
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		// на Windows ярлыки — это reparse point, а не regular file;
		// ловим их и здесь, чтобы не зависеть от типа записи каталога
		if runtime.GOOS == "windows" && (strings.EqualFold(filepath.Ext(name), ".lnk") ||
			strings.EqualFold(filepath.Ext(name), ".url")) {
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
