//go:build linux

package autostart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnableDisable(t *testing.T) {
	// изолируемся в временный XDG_CONFIG_HOME
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	if IsEnabled() {
		t.Fatal("автозапуск не должен быть включён изначально")
	}
	if err := Enable(); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if !IsEnabled() {
		t.Fatal("после Enable автозапуск должен быть включён")
	}

	p := filepath.Join(tmp, "autostart", "sweepy.desktop")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("desktop-файл не создан: %v", err)
	}
	if !strings.Contains(string(data), "Exec=") || !strings.Contains(string(data), "Type=Application") {
		t.Errorf("desktop-файл выглядит неправильно:\n%s", data)
	}

	if err := Disable(); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if IsEnabled() {
		t.Fatal("после Disable автозапуск должен быть выключен")
	}
	// повторный Disable не ошибка
	if err := Disable(); err != nil {
		t.Fatalf("повторный Disable: %v", err)
	}
}
