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

	// RecoveredBackup — путь к карантиновому файлу, в который переехал
	// повреждённый журнал при загрузке (пусто, если всё было в порядке).
	RecoveredBackup string `json:"-"`
}

// OpenJournal загружает журнал из path; если файла нет — создаёт пустой.
// Битый JSON не роняет приложение: повреждённый файл сохраняется рядом
// (journal.json.bad-<ts>) и стартует чистый журнал — RecoveredBackup
// сообщает UI, что нужен диалог восстановления.
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
		bak := quarantine(path, "bad")
		_ = bak // даже если сохранить бэкап не удалось — стартуем с пустым журналом
		fresh := &Journal{path: path, RecoveredBackup: bak}
		return fresh, nil
	}
	j.path = path
	return j, nil
}

// quarantine переименовывает повреждённый файл в "<path>.<tag>-<unix>".
// Возвращает путь карантина (или "" при неудаче).
func quarantine(path, tag string) string {
	bak := fmt.Sprintf("%s.%s-%d", path, tag, time.Now().Unix())
	if err := os.Rename(path, bak); err != nil {
		return ""
	}
	return bak
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

// RecentActive возвращает до n последних неоткаченных сессий, новые первыми.
func (j *Journal) RecentActive(n int) []*Session {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []*Session
	for i := len(j.Sessions) - 1; i >= 0 && len(out) < n; i-- {
		if !j.Sessions[i].Undone {
			cp := j.Sessions[i]
			out = append(out, &cp)
		}
	}
	return out
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
