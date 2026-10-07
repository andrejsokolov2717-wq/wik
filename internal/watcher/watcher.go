// Package watcher следит за папкой и раскладывает новые файлы на лету.
package watcher

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"sweepy/internal/core"
)

// sizeStamp — снимок размера файла для проверки «дописался ли».
type sizeStamp struct {
	size int64
	when time.Time
}

// Watcher — фоновый сторож «Загрузок»: новый файл раскладывается
// через quiet-период после последнего изменения (дебаунс записи).
// Дополнительно убеждается, что размер файла не меняется (загрузка
// завершена), и пропускает временные файлы браузеров (.part и т.п.).
type Watcher struct {
	Dir     string
	Quiet   time.Duration // пауза после последней записи, по умолчанию 5 сек
	Mover   *core.Mover
	Scanner *core.Scanner
	Cats    func() []core.Category // актуальные категории из настроек
	OnTidy  func(m core.Move)      // колбэк для UI/лога

	fsw      *fsnotify.Watcher
	mu       sync.Mutex
	pending  map[string]time.Time
	lastSize map[string]sizeStamp
	stop     chan struct{}
	done     chan struct{}
	running  bool
}

// New создаёт сторож для каталога dir.
func New(dir string, mover *core.Mover, scanner *core.Scanner, cats func() []core.Category) *Watcher {
	return &Watcher{
		Dir:      dir,
		Quiet:    5 * time.Second,
		Mover:    mover,
		Scanner:  scanner,
		Cats:     cats,
		pending:  map[string]time.Time{},
		lastSize: map[string]sizeStamp{},
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start запускает сторож в фоне. Повторный вызов безопасен.
func (w *Watcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(w.Dir, 0o755); err != nil {
		fsw.Close()
		return err
	}
	if err := fsw.Add(w.Dir); err != nil {
		fsw.Close()
		return err
	}

	w.mu.Lock()
	w.fsw = fsw
	w.running = true
	w.mu.Unlock()
	go w.loop()
	return nil
}

// Stop останавливает сторож. Повторный вызов безопасен.
func (w *Watcher) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	stop := w.stop
	fsw := w.fsw
	done := w.done
	w.mu.Unlock()

	close(stop)
	<-done // ждём выхода цикла loop()
	if fsw != nil {
		fsw.Close() // закрываем после выхода цикла: safe для pending-обработок
	}

	w.mu.Lock()
	// сброс каналов и карт для возможного перезапуска
	w.stop = make(chan struct{})
	w.done = make(chan struct{})
	w.pending = map[string]time.Time{}
	w.lastSize = map[string]sizeStamp{}
	w.fsw = nil
	w.mu.Unlock()
}

// Running сообщает, активен ли сторож.
func (w *Watcher) Running() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

func (w *Watcher) loop() {
	defer close(w.done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}
			w.mu.Lock()
			w.pending[ev.Name] = time.Now()
			w.mu.Unlock()
		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
		case <-ticker.C:
			w.flush()
		}
	}
}

// flush раскладывает все файлы, чьё «затишье» превысило Quiet, одним планом.
// Файл считается готовым, если за Quiet-период к нему не было записей
// И его размер не меняется между двумя проверками (загрузка завершилась).
func (w *Watcher) flush() {
	w.mu.Lock()
	now := time.Now()
	var readySet map[string]bool
	for name, t := range w.pending {
		if now.Sub(t) >= w.Quiet {
			if readySet == nil {
				readySet = map[string]bool{}
			}
			readySet[name] = true
			delete(w.pending, name)
		}
	}
	last := w.lastSize
	w.mu.Unlock()

	if len(readySet) == 0 {
		return
	}
	// проверка стабильности размера: файл, который всё ещё растёт,
	// возвращается в pending и получит ещё один quiet-период
	for name := range readySet {
		st, err := os.Stat(name)
		if err != nil || !st.Mode().IsRegular() {
			delete(readySet, name) // исчез / стал папкой — снимаем с контроля
			w.mu.Lock()
			delete(last, name)
			w.mu.Unlock()
			continue
		}
		w.mu.Lock()
		prev, seen := last[name]
		last[name] = sizeStamp{size: st.Size(), when: now}
		w.mu.Unlock()
		if seen && prev.size == st.Size() {
			continue // размер не изменился — файл дописан
		}
		delete(readySet, name) // ещё пишется — ждём дальше
		w.mu.Lock()
		w.pending[name] = now
		w.mu.Unlock()
	}
	if len(readySet) == 0 {
		return
	}

	files, err := w.Scanner.Scan(w.Dir)
	if err != nil {
		log.Printf("sweepy watcher: %v", err)
		return
	}
	// раскладываем только те, что «отстоялись»
	var fresh []core.FileInfo
	for _, f := range files {
		if readySet[f.Path] {
			fresh = append(fresh, f)
		}
	}
	if len(fresh) == 0 {
		return
	}
	plan := core.BuildPlan(w.Dir, fresh, w.Cats())
	if len(plan.Moves) == 0 {
		return
	}
	if _, err := w.Mover.Execute(plan); err != nil {
		log.Printf("sweepy watcher: %v", err)
		return
	}
	// после переноса снимаем файлы из наблюдения по размеру
	w.mu.Lock()
	for name := range readySet {
		delete(last, name)
	}
	w.mu.Unlock()
	for _, mv := range plan.Moves {
		if w.OnTidy != nil {
			w.OnTidy(mv)
		}
	}
}
