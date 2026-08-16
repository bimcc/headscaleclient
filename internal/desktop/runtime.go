package desktop

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const autostartIdentifier = "io.headscaleclient.desktop"

// Runtime is the Wails-facing implementation of application event and
// autostart ports. It is attached after application.New returns.
type Runtime struct {
	mu             sync.RWMutex
	app            *application.App
	nextListenerID uint64
	listeners      map[uint64]func(string, any)
}

func NewRuntime() *Runtime {
	return &Runtime{listeners: make(map[uint64]func(string, any))}
}

func (r *Runtime) Attach(app *application.App) {
	r.mu.Lock()
	r.app = app
	r.mu.Unlock()
}

func (r *Runtime) Emit(name string, payload any) {
	r.mu.RLock()
	app := r.app
	listeners := make([]func(string, any), 0, len(r.listeners))
	for _, listener := range r.listeners {
		listeners = append(listeners, listener)
	}
	r.mu.RUnlock()
	if app != nil {
		app.Event.Emit(name, payload)
	}
	for _, listener := range listeners {
		listener(name, payload)
	}
}

func (r *Runtime) Subscribe(listener func(string, any)) func() {
	if listener == nil {
		return func() {}
	}
	r.mu.Lock()
	r.nextListenerID++
	id := r.nextListenerID
	r.listeners[id] = listener
	r.mu.Unlock()
	return func() {
		r.mu.Lock()
		delete(r.listeners, id)
		r.mu.Unlock()
	}
}

func (r *Runtime) Enable() error {
	app := r.application()
	if app == nil {
		return nil
	}
	return app.Autostart.EnableWithOptions(application.AutostartOptions{
		Identifier: autostartIdentifier,
		Arguments:  []string{"--background"},
	})
}

func (r *Runtime) Disable() error {
	app := r.application()
	if app == nil {
		return nil
	}
	return app.Autostart.Disable()
}

func (r *Runtime) IsEnabled() (bool, error) {
	app := r.application()
	if app == nil {
		return false, nil
	}
	return app.Autostart.IsEnabled()
}

func (r *Runtime) application() *application.App {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.app
}
