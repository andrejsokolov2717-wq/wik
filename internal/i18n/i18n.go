// Package i18n — минимальная локализация Sweepy (русский и английский).
// Строки живут в каталоге ниже; язык хранится в настройках приложения,
// по умолчанию определяется по локали ОС.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// Lang — код языка интерфейса.
type Lang string

const (
	RU Lang = "ru"
	EN Lang = "en"
)

var current atomic.Value // Lang

func init() {
	current.Store(Detect())
}

// Detect определяет язык по переменным окружения ОС.
func Detect() Lang {
	for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(os.Getenv(env))
		if strings.HasPrefix(v, "ru") {
			return RU
		}
		if v != "" {
			return EN
		}
	}
	return EN
}

// Set переключает текущий язык.
func Set(l Lang) {
	if l != RU && l != EN {
		l = EN
	}
	current.Store(l)
}

// Current возвращает текущий язык.
func Current() Lang { return current.Load().(Lang) }

// T возвращает локализованную строку по ключу.
// Если в строке есть глаголы формата — подставляет args.
func T(key string, args ...any) string {
	l := Current()
	if s, ok := catalog[l][key]; ok {
		if len(args) == 0 {
			return s
		}
		return fmt.Sprintf(s, args...)
	}
	if s, ok := catalog[EN][key]; ok {
		if len(args) == 0 {
			return s
		}
		return fmt.Sprintf(s, args...)
	}
	return key
}

// Plural возвращает правильную форму существительного для числа n.
// key — ключ из pluralCatalog ("file" и т.п.).
func Plural(n int, key string) string {
	byLang := pluralCatalog[key]
	if byLang == nil {
		return ""
	}
	forms := byLang[Current()]
	if Current() == EN {
		if n == 1 {
			return forms[0]
		}
		return forms[1]
	}
	// русская плюрализация: 1 файл, 2-4 файла, 5-20 файлов, 21 файл…
	n10 := n % 10
	n100 := n % 100
	switch {
	case n10 == 1 && n100 != 11:
		return forms[0]
	case n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14):
		return forms[1]
	default:
		return forms[2]
	}
}

var pluralCatalog = map[string]map[Lang][3]string{
	"file": {
		RU: {"файл", "файла", "файлов"},
		EN: {"file", "files", "files"},
	},
}

var catalog = map[Lang]map[string]string{
	RU: {
		"app.windowTitle":    "Sweepy — порядок в файлах в один клик",
		"app.tabTidy":        "Порядок",
		"app.tabStats":       "Статистика",
		"app.tabSettings":    "Настройки",
		"app.trayOpen":       "Открыть Sweepy",
		"app.trayQuit":       "Выход",
		"app.recoveredTitle": "Журнал восстановлен",
		"app.recoveredBody":  "Файл журнала был повреждён и сохранён как %s. Начат с чистого журнала — ваши файлы в безопасности.",

		"home.subtitle":         "Чистый рабочий стол за 10 секунд — без подписки, без облака, без AI",
		"home.whatToTidy":       "Что разбираем:",
		"home.downloads":        "Загрузки",
		"home.desktop":          "Рабочий стол",
		"home.custom":           "Своя папка",
		"home.browse":           "Выбрать…",
		"home.pickFolder":       "Выберите папку для разбора",
		"home.noFolder":         "папка не выбрана",
		"home.previewHeader":    "Предпросмотр: что куда поедет",
		"home.statusStart":      "Выберите папку и нажмите «Сканировать»",
		"home.scan":             "Сканировать",
		"home.scanning":         "Просматриваю папку…",
		"home.tidy":             "Навести порядок",
		"home.undo":             "Отменить последнюю уборку",
		"home.undoPick":         "Отменить…",
		"home.undoPickTitle":    "Выберите уборку для отмены",
		"home.undoEntry":        "%s — %d %s",
		"home.cancel":           "Отмена",
		"home.openFolder":       "Открыть папку",
		"home.openFolderError":  "Не удалось открыть папку: %v",
		"home.shareCard":        "Карточка «до/после»",
		"home.confirmMoveTitle": "Переместить %d %s?",
		"home.confirmMoveBody":  "Файлы будут разложены по подпапкам. Ничего не удаляется — всегда можно отменить.",
		"home.working":          "Раскладываю файлы…",
		"home.done":             "Готово: разложено %d %s (%s). Хаос побеждён ✨",
		"home.alreadyClean":     "В папке %s уже порядок — разбирать нечего 👌",
		"home.found":            "Найдено %d %s (%s). Проверьте предпросмотр и жмите «Навести порядок».",
		"home.scanError":        "Не удалось прочитать %s: %v",
		"home.undoConfirmTitle": "Отменить уборку?",
		"home.undoConfirmBody":  "Вернуть %d %s из сессии %s на прежние места?",
		"home.restored":         "Возвращено %d %s",
		"home.restoredMissing":  " (не найдено: %d — вы, видимо, уже разобрали их сами)",
		"home.nothingToUndo":    "Нечего отменять — журнал пуст.",
		"home.saveCard":         "Сохранить карточку «до/после»",
		"home.cardSaved":        "Карточка сохранена: %s",
		"home.cardError":        "Не удалось создать карточку: %v",

		"stats.title":      "Ваша статистика порядка",
		"stats.filesMoved": "файлов разобрано",
		"stats.bytesSaved": "наведено порядка",
		"stats.streak":     "дней подряд 🔥",
		"stats.days":       "чистых дней всего",
		"stats.history":    "История по дням",
		"stats.empty":      "Пока пусто — нажмите «Навести порядок» на главном экране.",
		"stats.dayLine":    "%s — %d %s",
		"stats.refresh":    "Обновить",

		"settings.title":          "Настройки",
		"settings.automation":     "Автоматизация",
		"settings.watch":          "Следить за выбранной папкой и раскладывать новые файлы в фоне",
		"settings.watchError":     "Не удалось запустить сторож: %v",
		"settings.autostart":      "Запускать Sweepy вместе с системой",
		"settings.autostartError": "Не удалось изменить автозапуск: %v",
		"settings.language":       "Язык / Language",
		"settings.langRestart":    "Интерфейс переведён мгновенно — перезапуск не нужен.",
		"settings.categories":     "Категории и правила",
		"settings.colFolder":      "Папка",
		"settings.colExt":         "Расширения",
		"settings.colWords":       "Слова в имени",
		"settings.extPlaceholder": ".pdf, .docx",
		"settings.patPlaceholder": "screenshot, setup",
		"settings.addCategory":    "Добавить категорию",
		"settings.save":           "Сохранить настройки",
		"settings.reset":          "Сбросить к пресетам",
		"settings.needCategory":   "Нужна хотя бы одна категория.",
		"settings.saved":          "Категории сохранены.",
		"settings.resetTitle":     "Сброс",
		"settings.resetBody":      "Вернуть категории по умолчанию?",
		"settings.newCategory":    "Новая категория",
		"settings.about":          "О программе",
		"settings.aboutTitle":     "О программе",
		"settings.aboutBody":      "Sweepy %s\n\nЧистый рабочий стол в один клик.\nТолько перемещение — никогда удаление.\nБез подписки, без облака, без аккаунта, без телеметрии.\n\nЛицензия: MIT",

		"card.title":  "До » После",
		"card.files":  "%d %s разобрано",
		"card.footer": "сделано в Sweepy — $0.99, без подписки",
	},
	EN: {
		"app.windowTitle":    "Sweepy — a tidy desktop in one click",
		"app.tabTidy":        "Tidy",
		"app.tabStats":       "Stats",
		"app.tabSettings":    "Settings",
		"app.trayOpen":       "Open Sweepy",
		"app.trayQuit":       "Quit",
		"app.recoveredTitle": "Journal recovered",
		"app.recoveredBody":  "The journal file was corrupted and saved as %s. Started with a fresh journal — your files are safe.",

		"home.subtitle":         "A clean desktop in 10 seconds — no subscription, no cloud, no AI",
		"home.whatToTidy":       "What to tidy:",
		"home.downloads":        "Downloads",
		"home.desktop":          "Desktop",
		"home.custom":           "Custom folder",
		"home.browse":           "Browse…",
		"home.pickFolder":       "Choose a folder to tidy",
		"home.noFolder":         "no folder selected",
		"home.previewHeader":    "Preview: what goes where",
		"home.statusStart":      "Pick a folder and press “Scan”",
		"home.scan":             "Scan",
		"home.scanning":         "Looking through the folder…",
		"home.tidy":             "Tidy up",
		"home.undo":             "Undo last tidy-up",
		"home.undoPick":         "Undo…",
		"home.undoPickTitle":    "Choose a tidy-up to undo",
		"home.undoEntry":        "%s — %d %s",
		"home.cancel":           "Cancel",
		"home.openFolder":       "Open folder",
		"home.openFolderError":  "Could not open the folder: %v",
		"home.shareCard":        "Before/after card",
		"home.confirmMoveTitle": "Move %d %s?",
		"home.confirmMoveBody":  "Files will be moved into subfolders. Nothing is deleted — you can always undo.",
		"home.working":          "Tidying up…",
		"home.done":             "Done: %d %s tidied (%s). Chaos defeated ✨",
		"home.alreadyClean":     "%s is already tidy — nothing to do 👌",
		"home.found":            "Found %d %s (%s). Check the preview and press “Tidy up”.",
		"home.scanError":        "Cannot read %s: %v",
		"home.undoConfirmTitle": "Undo tidy-up?",
		"home.undoConfirmBody":  "Restore %d %s from the %s session to their original places?",
		"home.restored":         "Restored %d %s",
		"home.restoredMissing":  " (missing: %d — you have probably moved them yourself)",
		"home.nothingToUndo":    "Nothing to undo — the journal is empty.",
		"home.saveCard":         "Save before/after card",
		"home.cardSaved":        "Card saved: %s",
		"home.cardError":        "Could not create the card: %v",

		"stats.title":      "Your tidiness stats",
		"stats.filesMoved": "files tidied",
		"stats.bytesSaved": "tidied up",
		"stats.streak":     "day streak 🔥",
		"stats.days":       "tidy days total",
		"stats.history":    "History by day",
		"stats.empty":      "Empty so far — press “Tidy up” on the main screen.",
		"stats.dayLine":    "%s — %d %s",
		"stats.refresh":    "Refresh",

		"settings.title":          "Settings",
		"settings.automation":     "Automation",
		"settings.watch":          "Watch the selected folder and tidy new files automatically",
		"settings.watchError":     "Could not start the watcher: %v",
		"settings.autostart":      "Launch Sweepy at login",
		"settings.autostartError": "Could not change autostart: %v",
		"settings.language":       "Language / Язык",
		"settings.langRestart":    "The interface switched instantly — no restart needed.",
		"settings.categories":     "Categories and rules",
		"settings.colFolder":      "Folder",
		"settings.colExt":         "Extensions",
		"settings.colWords":       "Name keywords",
		"settings.extPlaceholder": ".pdf, .docx",
		"settings.patPlaceholder": "screenshot, setup",
		"settings.addCategory":    "Add category",
		"settings.save":           "Save settings",
		"settings.reset":          "Reset to defaults",
		"settings.needCategory":   "At least one category is required.",
		"settings.saved":          "Categories saved.",
		"settings.resetTitle":     "Reset",
		"settings.resetBody":      "Restore default categories?",
		"settings.newCategory":    "New category",
		"settings.about":          "About",
		"settings.aboutTitle":     "About Sweepy",
		"settings.aboutBody":      "Sweepy %s\n\nA tidy desktop in one click.\nMoves files — never deletes.\nNo subscription, no cloud, no account, no telemetry.\n\nLicense: MIT",

		"card.title":  "Before » After",
		"card.files":  "%d %s tidied",
		"card.footer": "made with Sweepy — $0.99, no subscription",
	},
}
