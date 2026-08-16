package tailscale

import (
	"context"

	"tailscale.com/ipn"
)

type EventKind string

const (
	EventChanged       EventKind = "changed"
	EventLoginURL      EventKind = "login-url"
	EventLoginFinished EventKind = "login-finished"
	EventError         EventKind = "error"
)

type Event struct {
	Kind EventKind
	URL  string
	Err  error
}

const preferredWatchMask = ipn.NotifyInitialState |
	ipn.NotifyInitialPrefs |
	ipn.NotifyInitialStatus |
	ipn.NotifyPeerPatches |
	ipn.NotifyNoNetMap |
	ipn.NotifyRateLimit

func (a *Adapter) Watch(ctx context.Context, handle func(Event)) error {
	watcher, err := a.openWatcher(ctx)
	if err != nil {
		return err
	}
	defer watcher.Close()

	for {
		notify, err := watcher.Next()
		if err != nil {
			return classifyError(err)
		}
		if notify.BrowseToURL != nil && *notify.BrowseToURL != "" {
			loginURL, validationErr := ValidateLoginURL(*notify.BrowseToURL)
			if validationErr != nil {
				handle(Event{Kind: EventError, Err: validationErr})
			} else {
				handle(Event{Kind: EventLoginURL, URL: loginURL})
			}
		}
		if notify.LoginFinished != nil {
			handle(Event{Kind: EventLoginFinished})
		}
		if notify.ErrMessage != nil && *notify.ErrMessage != "" {
			handle(Event{Kind: EventError, Err: classifyError(&daemonMessageError{message: *notify.ErrMessage})})
		}
		if notify.State != nil || notify.Prefs != nil || notify.InitialStatus != nil ||
			notify.SelfChange != nil || len(notify.PeersChanged) > 0 ||
			len(notify.PeersRemoved) > 0 || len(notify.PeerChangedPatch) > 0 {
			handle(Event{Kind: EventChanged})
		}
	}
}

func (a *Adapter) openWatcher(ctx context.Context) (bus, error) {
	watcher, err := a.daemon.WatchIPNBus(ctx, preferredWatchMask)
	if err != nil {
		watcher, err = a.daemon.WatchIPNBus(ctx, 0)
	}
	if err != nil {
		return nil, classifyError(err)
	}
	return watcher, nil
}

type daemonMessageError struct {
	message string
}

func (e *daemonMessageError) Error() string { return e.message }
