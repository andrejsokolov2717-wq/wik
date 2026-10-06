package core

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Mover выполняет и откатывает планы.
// Жёсткое правило безопасности: здесь НЕТ ни одного вызова удаления
// пользовательских файлов — только os.Rename (перемещение).
// Исключения: недописанная копия при междисковом переносе и опустевшие
// папки категорий после отката.
// Методы сериализованы мьютексом: UI и фоновый сторож не могут
// перемещать файлы одновременно.
type Mover struct {
	mu      sync.Mutex
	Journal *Journal
	Stats   *Stats
}

// NewMover создаёт движок с журналом и статистикой.
func NewMover(j *Journal, st *Stats) *Mover {
	return &Mover{Journal: j, Stats: st}
}

// Execute применяет план: перемещает файлы и пишет сессию в журнал.
// При ошибке на N-м файле уже перемещённые файлы остаются в журнале
// и доступны для отката — частичная уборка не теряется.
func (m *Mover) Execute(plan *Plan) (done int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(plan.Moves) == 0 {
		return 0, nil
	}
	sess := Session{
		ID:        newID(),
		StartedAt: time.Now(),
		Root:      plan.Root,
	}
	for _, mv := range plan.Moves {
		if err := os.MkdirAll(filepath.Dir(mv.To), 0o755); err != nil {
			return done, fmt.Errorf("не удалось создать %s: %w", filepath.Dir(mv.To), err)
		}
		if err := moveFile(mv.From, mv.To); err != nil {
			// сохраняем то, что успели
			if len(sess.Moves) > 0 {
				_ = m.Journal.Add(sess)
			}
			return done, fmt.Errorf("%s: %w", mv.From, err)
		}
		sess.Moves = append(sess.Moves, mv)
		done++
	}
	if err := m.Journal.Add(sess); err != nil {
		return done, err
	}
	if m.Stats != nil {
		m.Stats.Record(len(sess.Moves), plan.TotalBytes())
		_ = m.Stats.Save()
	}
	return done, nil
}

// Undo откатывает сессию: возвращает каждый файл на исходное место.
// Файлы, которые пользователь уже переименовал/убрал вручную, пропускаются.
func (m *Mover) Undo(sess *Session) (restored, missing int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// состояние отката смотрим в журнале — переданная сессия может быть
	// устаревшей копией
	if fresh := m.Journal.byID(sess.ID); fresh != nil {
		sess = fresh
	}
	if sess.Undone {
		return 0, 0, errors.New("эта уборка уже отменена")
	}
	// идём в обратном порядке, чтобы не столкнуться с коллизиями суффиксов
	for i := len(sess.Moves) - 1; i >= 0; i-- {
		mv := sess.Moves[i]
		if _, statErr := os.Stat(mv.To); statErr != nil {
			missing++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(mv.From), 0o755); err != nil {
			return restored, missing, err
		}
		// если на исходном месте уже появился другой файл с тем же именем —
		// не перезаписываем его, возвращаем рядом с суффиксом
		to := mv.From
		if _, statErr := os.Stat(to); statErr == nil {
			to = uniquePath(filepath.Dir(mv.From), filepath.Base(mv.From), map[string]bool{})
		}
		if err := moveFile(mv.To, to); err != nil {
			return restored, missing, fmt.Errorf("откат %s: %w", mv.To, err)
		}
		restored++
	}
	// убираем опустевшие категорийные папки
	removeEmptyDirs(sess)
	if err := m.Journal.MarkUndone(sess.ID); err != nil {
		return restored, missing, err
	}
	return restored, missing, nil
}

// moveFile — перемещение с фолбэком на копирование.
// os.Rename не работает между разными файловыми системами (EXDEV),
// поэтому при любой ошибке Rename пробуем «скопировать + убрать оригинал».
func moveFile(from, to string) error {
	renameErr := os.Rename(from, to)
	if renameErr == nil {
		return nil
	}
	// копируем, проверяем размер, сохраняем время модификации, убираем оригинал
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	n, err := io.Copy(dst, src)
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil || n != info.Size() {
		os.Remove(to) // убираем недописанную копию
		if err == nil {
			err = fmt.Errorf("скопировано %d из %d байт", n, info.Size())
		}
		return renameErr
	}
	_ = os.Chtimes(to, info.ModTime(), info.ModTime())
	return os.Remove(from)
}

// removeEmptyDirs удаляет опустевшие папки категорий после отката.
func removeEmptyDirs(sess *Session) {
	seen := map[string]bool{}
	for _, mv := range sess.Moves {
		dir := filepath.Dir(mv.To)
		if seen[dir] {
			continue
		}
		seen[dir] = true
		if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
			_ = os.Remove(dir) // пустая папка — безопасно
		}
	}
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%d", b, time.Now().Unix())
}
