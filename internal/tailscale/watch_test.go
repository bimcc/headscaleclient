package tailscale

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"tailscale.com/ipn"
)

type watchFallbackDaemon struct {
	*loginTestDaemon
	masks []ipn.NotifyWatchOpt
}

func (d *watchFallbackDaemon) WatchIPNBus(_ context.Context, mask ipn.NotifyWatchOpt) (bus, error) {
	d.masks = append(d.masks, mask)
	if len(d.masks) == 1 {
		return nil, errors.New("preferred mask is not supported")
	}
	return d.bus, nil
}

func TestOpenWatcherFallsBackToLegacyMask(t *testing.T) {
	t.Parallel()

	wantBus := &loginTestBus{}
	daemon := &watchFallbackDaemon{loginTestDaemon: &loginTestDaemon{bus: wantBus}}
	adapter := newAdapterWithDaemon(daemon)

	gotBus, err := adapter.openWatcher(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotBus != wantBus {
		t.Fatalf("watcher = %T, want %T", gotBus, wantBus)
	}
	wantMasks := []ipn.NotifyWatchOpt{preferredWatchMask, 0}
	if !reflect.DeepEqual(daemon.masks, wantMasks) {
		t.Fatalf("watch masks = %#v, want %#v", daemon.masks, wantMasks)
	}
}
