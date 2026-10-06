// Package watcher следит за папкой и раскладывает новые файлы на лету.
package watcher

import (
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"sweepy/internal/core"
)

// Watcher — фоновый сторож «Загрузок»: новый файл раскладывается
// через quiet-период после последнего изменения (дебаунс записи).
type Watcher struct {
	Dir     string
	Quiet   time.Duration // пауза после последней записи, по умолчанию 5 сек
	Mover   *core.Mover
	Scanner *core.Scanner
	Cats    func() []core.Category // актуальные категории из настроек
	OnTidy  func(m core.Move)      // колбэк для UI/лога

	fsw     *fsnotify.Watcher
	mu      sync.Mutex
	pending map[string]time.Time
	stop    chan struct{}
	done    chan struct{}
	running bool
}

// New создаёт сторож для каталога dir.
func New(dir string, mover *core.Mover, scanner *core.Scanner, cats func() []core.Category) *Watcher {
	return &Watcher{
		Dir:     dir,
		Quiet:   5 * time.Second,
		Mover:   mover,
		Scanner: scanner,
		Cats:    cats,
		pending: map[string]time.Time{},
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
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
	if err := fsw.Add(w.Dir); err != nil {
		fsw.Close()
		return err
	}
	w.fsw = fsw
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()
	go w.loop()
	return nil
}

// Stop останавливает сторож.
func (w *Watcher) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()
	close(w.stop)
	<-w.done
	w.fsw.Close()
	// сброс каналов для возможного перезапуска
	w.stop = make(chan struct{})
	w.done = make(chan struct{})
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
	w.mu.Unlock()

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
	for _, mv := range plan.Moves {
		if w.OnTidy != nil {
			w.OnTidy(mv)
		}
	}
}
