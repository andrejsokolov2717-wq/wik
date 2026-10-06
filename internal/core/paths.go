package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DesktopPath возвращает путь к рабочему столу пользователя.
// Учитывает XDG user-dirs на Linux (в т.ч. локализованные «Рабочий стол»).
func DesktopPath() string {
	if runtime.GOOS == "linux" {
		if p := xdgUserDir("DESKTOP"); p != "" {
			return p
		}
	}
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "Desktop")
	if _, err := os.Stat(p); err != nil {
		if ru := filepath.Join(home, "Рабочий стол"); dirExists(ru) {
			return ru
		}
	}
	return p
}

// DownloadsPath возвращает путь к папке загрузок пользователя.
func DownloadsPath() string {
	if runtime.GOOS == "linux" {
		if p := xdgUserDir("DOWNLOAD"); p != "" {
			return p
		}
	}
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "Downloads")
	if _, err := os.Stat(p); err != nil {
		if ru := filepath.Join(home, "Загрузки"); dirExists(ru) {
			return ru
		}
	}
	return p
}

// DataDir — папка для журнала и статистики Sweepy.
func DataDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "sweepy")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sweepy")
}

// xdgUserDir читает ~/.config/user-dirs.dirs (формат XDG_DESKTOP_DIR="$HOME/Desktop").
func xdgUserDir(key string) string {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".config", "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	prefix := "XDG_" + key + "_DIR=\""
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		p := strings.TrimSuffix(strings.TrimPrefix(line, prefix), "\"")
		p = strings.ReplaceAll(p, "$HOME", home)
		if dirExists(p) {
			return p
		}
	}
	return ""
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
