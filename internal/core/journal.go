package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// maxSessions — сколько сессий храним в журнале.
// Старые откаченные сессии вычищаются, чтобы файл не рос бесконечно.
const maxSessions = 200

// Session — одна уборка: набор перемещений, которые можно откатить целиком.
type Session struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"started_at"`
	Root      string    `json:"root"`
	Moves     []Move    `json:"moves"`
	Undone    bool      `json:"undone"`
}

// Journal — постоянный журнал сессий (JSON-файл в конфиге пользователя).
// Безопасен для конкурентного доступа из UI и фонового сторожа.
type Journal struct {
	mu       sync.Mutex
	path     string
	Sessions []Session `json:"sessions"`
}

// OpenJournal загружает журнал из path; если файла нет — создаёт пустой.
func OpenJournal(path string) (*Journal, error) {
	j := &Journal{path: path}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return j, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, j); err != nil {
		return nil, fmt.Errorf("журнал повреждён: %w", err)
	}
	j.path = path
	return j, nil
}

// Add записывает сессию в журнал и сохраняет его на диск.
func (j *Journal) Add(s Session) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Sessions = append(j.Sessions, s)
	j.pruneLocked()
	return j.saveLocked()
}

// pruneLocked отбрасывает самые старые откаченные сессии сверх лимита.
// Активные (неоткаченные) сессии не выбрасываются никогда — их ещё можно откатить.
func (j *Journal) pruneLocked() {
	for len(j.Sessions) > maxSessions {
		idx := -1
		for i := range j.Sessions {
			if j.Sessions[i].Undone {
				idx = i
				break
			}
		}
		if idx < 0 {
			return // все сессии активны — не трогаем
		}
		j.Sessions = append(j.Sessions[:idx], j.Sessions[idx+1:]...)
	}
}

// MarkUndone помечает сессию откаченной.
func (j *Journal) MarkUndone(id string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i := range j.Sessions {
		if j.Sessions[i].ID == id {
			j.Sessions[i].Undone = true
			return j.saveLocked()
		}
	}
	return fmt.Errorf("сессия %s не найдена", id)
}

// LastActive возвращает копию последней неоткаченной сессии или nil.
func (j *Journal) LastActive() *Session {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i := len(j.Sessions) - 1; i >= 0; i-- {
		if !j.Sessions[i].Undone {
			cp := j.Sessions[i]
			return &cp
		}
	}
	return nil
}

// byID возвращает актуальную копию сессии из журнала (или nil).
// Mover пользуется ею, чтобы состояние отката бралось из источника истины.
func (j *Journal) byID(id string) *Session {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i := range j.Sessions {
		if j.Sessions[i].ID == id {
			cp := j.Sessions[i]
			return &cp
		}
	}
	return nil
}

// ActiveCount — число сессий, которые ещё можно откатить.
func (j *Journal) ActiveCount() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	n := 0
	for i := range j.Sessions {
		if !j.Sessions[i].Undone {
			n++
		}
	}
	return n
}

func (j *Journal) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(j.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	tmp := j.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	// атомарная замена, чтобы журнал не побился при внезапном выходе
	return os.Rename(tmp, j.path)
}
