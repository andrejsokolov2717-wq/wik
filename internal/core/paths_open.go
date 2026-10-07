package core

import (
	"os"
	"os/exec"
	"runtime"
)

// OpenFolder открывает папку в системном файловом менеджере
// (Finder / Explorer / xdg-open). Возвращает ошибку, если папки нет
// или запуск не удался. Дочерний процесс detachment-стиля: не ждём.
func OpenFolder(path string) error {
	if path == "" {
		return os.ErrNotExist
	}
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		return os.ErrNotExist
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default: // linux, *bsd
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
