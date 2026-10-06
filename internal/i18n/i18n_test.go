package i18n

import "testing"

func TestT_BothLangs(t *testing.T) {
	Set(RU)
	if got := T("app.tabTidy"); got != "Порядок" {
		t.Errorf("RU: получено %q", got)
	}
	Set(EN)
	if got := T("app.tabTidy"); got != "Tidy" {
		t.Errorf("EN: получено %q", got)
	}
	// форматирование
	if got := T("home.confirmMoveTitle", 3, "files"); got != "Move 3 files?" {
		t.Errorf("EN fmt: получено %q", got)
	}
	// неизвестный ключ возвращается как есть
	if got := T("no.such.key"); got != "no.such.key" {
		t.Errorf("fallback: получено %q", got)
	}
	Set(RU)
}

func TestT_AllKeysExistInBothLangs(t *testing.T) {
	for key := range catalog[EN] {
		if _, ok := catalog[RU][key]; !ok {
			t.Errorf("ключ %q есть в EN, но нет в RU", key)
		}
	}
	for key := range catalog[RU] {
		if _, ok := catalog[EN][key]; !ok {
			t.Errorf("ключ %q есть в RU, но нет в EN", key)
		}
	}
}

func TestPlural_RU(t *testing.T) {
	Set(RU)
	defer Set(RU)
	cases := map[int]string{
		1: "файл", 2: "файла", 3: "файла", 5: "файлов",
		11: "файлов", 12: "файлов", 21: "файл", 22: "файла",
		25: "файлов", 101: "файл", 111: "файлов",
	}
	for n, want := range cases {
		if got := Plural(n, "file"); got != want {
			t.Errorf("Plural(%d): ожидалось %q, получено %q", n, want, got)
		}
	}
}

func TestPlural_EN(t *testing.T) {
	Set(EN)
	defer Set(RU)
	if got := Plural(1, "file"); got != "file" {
		t.Errorf("Plural(1) EN: %q", got)
	}
	if got := Plural(5, "file"); got != "files" {
		t.Errorf("Plural(5) EN: %q", got)
	}
}
