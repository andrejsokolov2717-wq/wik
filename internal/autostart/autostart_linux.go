//go:build linux

package autostart

import (
	"os"
	"path/filepath"
	"strings"
)

func desktopFilePath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "autostart", "sweepy.desktop"), nil
}

func enable(exe string) error {
	p, err := desktopFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	// путь с пробелами экранируем двойными кавычками по спецификации Desktop Entry
	quoted := exe
	if strings.ContainsAny(exe, " \t") {
		quoted = `"` + exe + `"`
	}
	content := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=Sweepy\n" +
		"Comment=Tidy up Downloads and Desktop in one click\n" +
		"Exec=" + quoted + "\n" +
		"Terminal=false\n" +
		"X-GNOME-Autostart-enabled=true\n"
	return os.WriteFile(p, []byte(content), 0o644)
}

func disable() error {
	p, err := desktopFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func isEnabled() bool {
	p, err := desktopFilePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}
