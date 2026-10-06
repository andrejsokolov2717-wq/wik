package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sweepy/internal/core"
)

// TestWatcher_TidiesNewFile: сторож подхватывает новый файл
// и раскладывает его после quiet-периода.
func TestWatcher_TidiesNewFile(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()

	journal, err := core.OpenJournal(filepath.Join(dataDir, "journal.json"))
	if err != nil {
		t.Fatal(err)
	}
	stats, _ := core.OpenStats(filepath.Join(dataDir, "stats.json"))
	mover := core.NewMover(journal, stats)

	tidied := make(chan core.Move, 4)
	w := New(dir, mover, core.NewScanner(), core.DefaultCategories)
	w.Quiet = 300 * time.Millisecond
	w.OnTidy = func(m core.Move) { tidied <- m }

	if err := w.Start(); err != nil {
		t.Skipf("fsnotify недоступен в этом окружении: %v", err)
	}
	defer w.Stop()

	// «скачали» файл
	if err := os.WriteFile(filepath.Join(dir, "отчёт.pdf"), []byte("данные"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case mv := <-tidied:
		if mv.Category != "Документы" {
			t.Errorf("ожидалась категория «Документы», получена %q", mv.Category)
		}
		if _, err := os.Stat(mv.To); err != nil {
			t.Errorf("файл не на месте после авторазбора: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("сторож не разобрал файл за 5 секунд")
	}

	// авторазбор откатывается через общий журнал
	sess := journal.LastActive()
	if sess == nil {
		t.Fatal("сессия сторожа не записана в журнал")
	}
}

// TestWatcher_Restart: повторный Start/Stop безопасен.
func TestWatcher_Restart(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()
	journal, _ := core.OpenJournal(filepath.Join(dataDir, "journal.json"))
	mover := core.NewMover(journal, nil)

	w := New(dir, mover, core.NewScanner(), core.DefaultCategories)
	if err := w.Start(); err != nil {
		t.Skipf("fsnotify недоступен: %v", err)
	}
	if !w.Running() {
		t.Error("ожидался running=true")
	}
	w.Stop()
	if w.Running() {
		t.Error("ожидался running=false после Stop")
	}
	if err := w.Start(); err != nil {
		t.Fatalf("перезапуск: %v", err)
	}
	w.Stop()
}
