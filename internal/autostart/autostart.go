// Package autostart управляет запуском Sweepy вместе с системой.
// Реализации по ОС: реестр Windows, XDG autostart в Linux, LaunchAgent в macOS.
package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

// exePath возвращает путь к текущему исполняемому файлу.
func exePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// разрешаем симлинки — в автозапуск пишем реальный путь
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return exe, nil
}

// Enable включает автозапуск.
func Enable() error {
	exe, err := exePath()
	if err != nil {
		return fmt.Errorf("не удалось найти путь к программе: %w", err)
	}
	return enable(exe)
}

// Disable выключает автозапуск.
func Disable() error { return disable() }

// IsEnabled сообщает, включён ли автозапуск.
func IsEnabled() bool { return isEnabled() }
