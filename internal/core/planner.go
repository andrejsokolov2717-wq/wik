package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Move — одна атомарная операция перемещения.
type Move struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Category string `json:"category"`
	Size     int64  `json:"size"`
}

// Plan — набор перемещений, показанный пользователю до применения.
type Plan struct {
	Root  string // целевая папка (Рабочий стол / Загрузки)
	Moves []Move
}

// TotalBytes — суммарный размер файлов в плане.
func (p *Plan) TotalBytes() int64 {
	var n int64
	for _, m := range p.Moves {
		n += m.Size
	}
	return n
}

// BuildPlan строит план: каждый файл → подпапка категории внутри root.
// Коллизии имён разрешаются суффиксом " (1)", " (2)" и т.д. —
// существующие файлы никогда не перезаписываются.
func BuildPlan(root string, files []FileInfo, cats []Category) *Plan {
	plan := &Plan{Root: root}
	taken := map[string]bool{} // занятые целевые пути (на диске + в плане)

	for _, f := range files {
		cat := MatchCategory(f.Name, cats)
		if cat == nil {
			continue
		}
		destDir := filepath.Join(root, cat.Name)
		to := uniquePath(destDir, f.Name, taken)
		taken[strings.ToLower(to)] = true
		plan.Moves = append(plan.Moves, Move{
			From:     f.Path,
			To:       to,
			Category: cat.Name,
			Size:     f.Size,
		})
	}
	return plan
}

// uniquePath подбирает свободное имя в destDir.
func uniquePath(destDir, name string, taken map[string]bool) string {
	candidate := filepath.Join(destDir, name)
	if !occupied(candidate, taken) {
		return candidate
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		candidate = filepath.Join(destDir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if !occupied(candidate, taken) {
			return candidate
		}
	}
}

func occupied(path string, taken map[string]bool) bool {
	if taken[strings.ToLower(path)] {
		return true
	}
	_, err := os.Stat(path)
	return err == nil
}
