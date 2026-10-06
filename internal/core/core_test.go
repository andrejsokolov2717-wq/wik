package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- правила ---

func TestMatchCategory_Basic(t *testing.T) {
	cats := DefaultCategories()
	cases := map[string]string{
		"отчёт.pdf":              "Документы",
		"photo.JPG":              "Изображения",
		"Screenshot 2026-01.png": "Скриншоты", // приоритет паттерна над расширением
		"Снимок экрана (12).png": "Скриншоты",
		"archive.tar.gz":         "Архивы",
		"song.flac":              "Аудио",
		"movie.mkv":              "Видео",
		"setup-1.0.exe":          "Программы",
		"weird.xyz":              "Прочее",
		"no_extension":           "Прочее",
	}
	for name, want := range cases {
		got := MatchCategory(name, cats)
		if got == nil || got.Name != want {
			t.Errorf("%q: ожидалось %q, получено %v", name, want, got)
		}
	}
}

// --- планировщик ---

func TestBuildPlan_CollisionSuffix(t *testing.T) {
	dir := t.TempDir()
	// существующий файл с тем же именем в целевой подпапке
	os.MkdirAll(filepath.Join(dir, "Документы"), 0o755)
	existing := filepath.Join(dir, "Документы", "a.pdf")
	os.WriteFile(existing, []byte("old"), 0o644)

	files := []FileInfo{
		{Path: filepath.Join(dir, "a.pdf"), Name: "a.pdf", Size: 3},
		{Path: filepath.Join(dir, "b.pdf"), Name: "b.pdf", Size: 3},
	}
	plan := BuildPlan(dir, files, DefaultCategories())
	if len(plan.Moves) != 2 {
		t.Fatalf("ожидалось 2 перемещения, получено %d", len(plan.Moves))
	}
	if plan.Moves[0].To == existing {
		t.Errorf("план перезаписал бы существующий файл: %s", plan.Moves[0].To)
	}
	if !strings.Contains(plan.Moves[0].To, "a (1).pdf") {
		t.Errorf("ожидался суффикс (1), получено %s", plan.Moves[0].To)
	}
}

// --- выполнение и откат ---

func TestExecuteAndUndo_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()

	// создаём «захламлённую» папку
	names := []string{"отчёт.pdf", "фото.png", "архив.zip", "песня.mp3", "фильм.mp4", "setup.exe"}
	contents := map[string]string{}
	for _, n := range names {
		contents[n] = "данные файла " + n
		os.WriteFile(filepath.Join(dir, n), []byte(contents[n]), 0o644)
	}

	journal, err := OpenJournal(filepath.Join(dataDir, "journal.json"))
	if err != nil {
		t.Fatal(err)
	}
	stats, _ := OpenStats(filepath.Join(dataDir, "stats.json"))
	mover := NewMover(journal, stats)

	scanner := NewScanner()
	files, err := scanner.Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	plan := BuildPlan(dir, files, DefaultCategories())
	if len(plan.Moves) != len(names) {
		t.Fatalf("ожидалось %d перемещений, получено %d", len(names), len(plan.Moves))
	}

	done, err := mover.Execute(plan)
	if err != nil || done != len(names) {
		t.Fatalf("Execute: done=%d err=%v", done, err)
	}

	// корень должен быть пуст от исходных файлов
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); !os.IsNotExist(err) {
			t.Errorf("файл %s остался на месте после уборки", n)
		}
	}
	// статистика записалась
	if stats.FilesMoved != len(names) || stats.BytesSorted <= 0 {
		t.Errorf("статистика не обновилась: %+v", stats)
	}
	if stats.Streak() != 1 {
		t.Errorf("ожидалась серия 1 день, получено %d", stats.Streak())
	}

	// откат возвращает всё как было
	sess := journal.LastActive()
	if sess == nil {
		t.Fatal("журнал пуст после уборки")
	}
	restored, missing, err := mover.Undo(sess)
	if err != nil {
		t.Fatal(err)
	}
	if restored != len(names) || missing != 0 {
		t.Errorf("Undo: restored=%d missing=%d", restored, missing)
	}
	for _, n := range names {
		got, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			t.Errorf("файл %s не вернулся: %v", n, err)
			continue
		}
		if string(got) != contents[n] {
			t.Errorf("файл %s повреждён при откате", n)
		}
	}
	// повторный откат запрещён
	if _, _, err := mover.Undo(sess); err == nil {
		t.Error("повторный откат должен возвращать ошибку")
	}
}

// TestExecute_PartialFailureJournal: при ошибке посередине частичный
// прогресс сохраняется в журнале и доступен для отката.
func TestExecute_PartialFailureJournal(t *testing.T) {
	dir := t.TempDir()
	dataDir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "a.pdf"), []byte("a"), 0o644)
	// файл-призрак: есть в плане, но исчезнет к моменту выполнения
	ghost := filepath.Join(dir, "b.pdf")
	os.WriteFile(ghost, []byte("b"), 0o644)

	journal, _ := OpenJournal(filepath.Join(dataDir, "journal.json"))
	mover := NewMover(journal, nil)

	scanner := NewScanner()
	files, _ := scanner.Scan(dir)
	plan := BuildPlan(dir, files, DefaultCategories())

	// «пользователь» удалил файл между сканированием и применением
	os.Remove(ghost)

	done, err := mover.Execute(plan)
	if err == nil {
		t.Fatal("ожидалась ошибка на исчезнувшем файле")
	}
	if done != 1 {
		t.Errorf("ожидался 1 успешный перенос, получено %d", done)
	}
	sess := journal.LastActive()
	if sess == nil || len(sess.Moves) != 1 {
		t.Fatalf("частичная сессия не записана в журнал: %+v", sess)
	}
	// частичную уборку можно откатить
	restored, _, err := mover.Undo(sess)
	if err != nil || restored != 1 {
		t.Errorf("откат частичной уборки: restored=%d err=%v", restored, err)
	}
}

// --- журнал ---

func TestJournal_Persistence(t *testing.T) {
	dataDir := t.TempDir()
	path := filepath.Join(dataDir, "journal.json")

	j1, _ := OpenJournal(path)
	j1.Add(Session{ID: "s1", Moves: []Move{{From: "/a", To: "/b"}}})

	j2, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(j2.Sessions) != 1 || j2.Sessions[0].ID != "s1" {
		t.Fatalf("журнал не сохранился: %+v", j2.Sessions)
	}
	if s := j2.LastActive(); s == nil || s.ID != "s1" {
		t.Error("LastActive не нашёл сессию")
	}
	j2.MarkUndone("s1")
	if j2.LastActive() != nil {
		t.Error("после MarkUndone активных сессий быть не должно")
	}
}

// --- статистика ---

func TestStats_Streak(t *testing.T) {
	s := &Stats{Days: map[string]int{}}
	if s.Streak() != 0 {
		t.Error("пустая статистика: серия должна быть 0")
	}
	s.Record(3, 1024)
	s.Record(2, 2048)
	if s.FilesMoved != 5 || s.BytesSorted != 3072 {
		t.Errorf("Record: %+v", s)
	}
	if s.Streak() != 1 {
		t.Errorf("ожидалась серия 1, получено %d", s.Streak())
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		500:             "500 Б",
		2048:            "2 КБ",
		5 * 1024 * 1024: "5.0 МБ",
	}
	for in, want := range cases {
		if got := FormatBytes(in); got != want {
			t.Errorf("FormatBytes(%d): ожидалось %q, получено %q", in, want, got)
		}
	}
}
