// Package ui — графический интерфейс Sweepy на Fyne.
package ui

import (
	"encoding/json"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"

	"sweepy/internal/core"
	"sweepy/internal/i18n"
	"sweepy/internal/watcher"
)

// Controller связывает ядро, сторож и экраны приложения.
type Controller struct {
	App     fyne.App
	Window  fyne.Window
	Scanner *core.Scanner
	Journal *core.Journal
	Stats   *core.Stats
	Mover   *core.Mover
	Watcher *watcher.Watcher

	mu   sync.RWMutex
	cats []core.Category
}

// NewController собирает сервисы приложения.
func NewController(a fyne.App, w fyne.Window) (*Controller, error) {
	// язык интерфейса из настроек; при первом запуске — по локали ОС
	if lang := a.Preferences().String("lang"); lang != "" {
		i18n.Set(i18n.Lang(lang))
	}

	dataDir := core.DataDir()
	journal, err := core.OpenJournal(filepath.Join(dataDir, "journal.json"))
	if err != nil {
		return nil, err
	}
	stats, err := core.OpenStats(filepath.Join(dataDir, "stats.json"))
	if err != nil {
		return nil, err
	}
	c := &Controller{
		App:     a,
		Window:  w,
		Scanner: core.NewScanner(),
		Journal: journal,
		Stats:   stats,
	}
	c.Mover = core.NewMover(journal, stats)
	c.cats = c.loadCategories()
	return c, nil
}

// Categories возвращает актуальные категории.
func (c *Controller) Categories() []core.Category {
	c.mu.RLock()
	cats := c.cats
	c.mu.RUnlock()
	if len(cats) == 0 {
		return core.DefaultCategories()
	}
	out := make([]core.Category, len(cats))
	copy(out, cats)
	return out
}

// SetCategories сохраняет категории в настройки.
func (c *Controller) SetCategories(cats []core.Category) {
	c.mu.Lock()
	c.cats = cats
	c.mu.Unlock()
	if data, err := json.Marshal(cats); err == nil {
		c.App.Preferences().SetString("categories", string(data))
	}
}

func (c *Controller) loadCategories() []core.Category {
	raw := c.App.Preferences().String("categories")
	if raw == "" {
		return core.DefaultCategories()
	}
	var cats []core.Category
	if err := json.Unmarshal([]byte(raw), &cats); err != nil || len(cats) == 0 {
		return core.DefaultCategories()
	}
	return cats
}

// Language — текущий язык интерфейса из настроек.
func (c *Controller) Language() i18n.Lang {
	return i18n.Current()
}

// SetLanguage сохраняет и применяет язык.
func (c *Controller) SetLanguage(l i18n.Lang) {
	i18n.Set(l)
	c.App.Preferences().SetString("lang", string(l))
}

// TargetDir — текущая целевая папка из настроек.
func (c *Controller) TargetDir() string {
	pref := c.App.Preferences()
	switch pref.StringWithFallback("target", "downloads") {
	case "desktop":
		return core.DesktopPath()
	case "custom":
		return pref.StringWithFallback("custom_path", core.DownloadsPath())
	default:
		return core.DownloadsPath()
	}
}

// SetTarget сохраняет выбор целевой папки.
func (c *Controller) SetTarget(mode, customPath string) {
	c.App.Preferences().SetString("target", mode)
	if customPath != "" {
		c.App.Preferences().SetString("custom_path", customPath)
	}
}

// StartWatcher / StopWatcher — управление фоновым сторожем целевой папки.
func (c *Controller) StartWatcher() error {
	dir := c.TargetDir()
	if c.Watcher != nil && c.Watcher.Running() {
		c.Watcher.Stop()
	}
	c.Watcher = watcher.New(dir, c.Mover, c.Scanner, c.Categories)
	if err := c.Watcher.Start(); err != nil {
		c.App.Preferences().SetBool("watch_enabled", false)
		return err
	}
	c.App.Preferences().SetBool("watch_enabled", true)
	return nil
}

func (c *Controller) StopWatcher() {
	if c.Watcher != nil {
		c.Watcher.Stop()
	}
	c.App.Preferences().SetBool("watch_enabled", false)
}

// WatcherEnabled — сохранённая настройка сторожа.
func (c *Controller) WatcherEnabled() bool {
	return c.App.Preferences().Bool("watch_enabled")
}
