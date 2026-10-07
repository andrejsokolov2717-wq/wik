package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Stats — накопительная статистика: «разобрано файлов», «спасено ГБ», серия дней.
// Безопасна для конкурентного доступа из UI и фонового сторожа.
type Stats struct {
	mu          sync.Mutex
	path        string
	FilesMoved  int            `json:"files_moved"`
	BytesSorted int64          `json:"bytes_sorted"`
	Days        map[string]int `json:"days"` // "2006-01-02" -> число файлов

	// RecoveredBackup — путь к карантиновому файлу битой статистики.
	RecoveredBackup string `json:"-"`
}

// Snapshot — неизменяемый снимок статистики для UI.
type Snapshot struct {
	FilesMoved  int
	BytesSorted int64
	Streak      int
	ActiveDays  int
	Days        map[string]int
}

// OpenStats загружает статистику из path; если файла нет — создаёт пустую.
// Битый JSON не роняет приложение: файл уходит в карантин, статистика
// начинается заново (RecoveryNote сообщает об этом UI).
func OpenStats(path string) (*Stats, error) {
	s := &Stats{path: path, Days: map[string]int{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		bak := quarantine(path, "bad")
		_ = bak
		fresh := &Stats{path: path, Days: map[string]int{}, RecoveredBackup: bak}
		return fresh, nil
	}
	s.path = path
	if s.Days == nil {
		s.Days = map[string]int{}
	}
	return s, nil
}

// Record учитывает уборку: n файлов и bytes байт за сегодня.
func (s *Stats) Record(n int, bytes int64) {
	if n <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FilesMoved += n
	s.BytesSorted += bytes
	key := time.Now().Format("2006-01-02")
	s.Days[key] += n
}

// Streak — сколько дней подряд (включая сегодня или вчера) была уборка.
func (s *Stats) Streak() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.streakLocked(time.Now())
}

func (s *Stats) streakLocked(now time.Time) int {
	day := now
	// серия может начинаться сегодня или со вчера
	if _, ok := s.Days[day.Format("2006-01-02")]; !ok {
		day = day.AddDate(0, 0, -1)
		if _, ok := s.Days[day.Format("2006-01-02")]; !ok {
			return 0
		}
	}
	streak := 0
	for {
		if _, ok := s.Days[day.Format("2006-01-02")]; !ok {
			break
		}
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak
}

// ActiveDays — всего дней с уборками.
func (s *Stats) ActiveDays() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Days)
}

// Snapshot возвращает консистентный снимок для отрисовки.
func (s *Stats) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	days := make(map[string]int, len(s.Days))
	for k, v := range s.Days {
		days[k] = v
	}
	return Snapshot{
		FilesMoved:  s.FilesMoved,
		BytesSorted: s.BytesSorted,
		Streak:      s.streakLocked(time.Now()),
		ActiveDays:  len(s.Days),
		Days:        days,
	}
}

// Save сохраняет статистику на диск.
func (s *Stats) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// FormatBytes форматирует байты человекочитаемо: «1.4 ГБ».
func FormatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f ГБ", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f МБ", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.0f КБ", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d Б", b)
	}
}
