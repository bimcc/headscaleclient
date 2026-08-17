package tailscale

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/types/key"
)

type pingTestDaemon struct {
	*loginTestDaemon
	status    *ipnstate.Status
	results   []*ipnstate.PingResult
	errors    []error
	pingTypes []tailcfg.PingType
	pingCalls int
}

func (d *pingTestDaemon) Status(context.Context) (*ipnstate.Status, error) {
	return d.status, nil
}

func (d *pingTestDaemon) Ping(_ context.Context, _ netip.Addr, pingType tailcfg.PingType) (*ipnstate.PingResult, error) {
	d.pingTypes = append(d.pingTypes, pingType)
	index := d.pingCalls
	d.pingCalls++
	var result *ipnstate.PingResult
	if len(d.results) > 0 {
		resultIndex := min(index, len(d.results)-1)
		result = d.results[resultIndex]
	}
	var err error
	if len(d.errors) > 0 {
		errorIndex := min(index, len(d.errors)-1)
		err = d.errors[errorIndex]
	}
	return result, err
}

func TestPingDeviceUsesDiscoAndStopsWhenDirect(t *testing.T) {
	t.Parallel()

	daemon := newPingTestDaemon(
		&ipnstate.PingResult{LatencySeconds: 0.432, DERPRegionID: 10, DERPRegionCode: "hkg"},
		&ipnstate.PingResult{LatencySeconds: 0.024, Endpoint: "203.0.113.8:41641"},
	)
	adapter := newAdapterWithDaemon(daemon)
	adapter.pingInterval = 0

	result, err := adapter.PingDevice(context.Background(), "peer-1")
	if err != nil {
		t.Fatalf("PingDevice() error: %v", err)
	}
	if result.Via != PingViaDirect || result.Endpoint != "203.0.113.8:41641" || result.LatencyMS != 24 {
		t.Fatalf("PingDevice() result = %+v", result)
	}
	if len(daemon.pingTypes) != 2 {
		t.Fatalf("ping calls = %d, want 2", len(daemon.pingTypes))
	}
	for _, pingType := range daemon.pingTypes {
		if pingType != tailcfg.PingDisco {
			t.Fatalf("ping type = %q, want %q", pingType, tailcfg.PingDisco)
		}
	}
}

func TestPingDeviceReportsRelayAfterBoundedAttempts(t *testing.T) {
	t.Parallel()

	daemon := newPingTestDaemon(&ipnstate.PingResult{
		LatencySeconds: 0.125,
		DERPRegionID:   10,
		DERPRegionCode: "hkg",
	})
	adapter := newAdapterWithDaemon(daemon)
	adapter.pingAttempts = 2
	adapter.pingInterval = 0

	result, err := adapter.PingDevice(context.Background(), "peer-1")
	if err != nil {
		t.Fatalf("PingDevice() error: %v", err)
	}
	if result.Via != PingViaRelay || result.RelayRegion != "hkg" || result.LatencyMS != 125 {
		t.Fatalf("PingDevice() result = %+v", result)
	}
	if daemon.pingCalls != 2 {
		t.Fatalf("ping calls = %d, want 2", daemon.pingCalls)
	}
}

func TestPingDeviceDoesNotAssumeMissingRouteIsDirect(t *testing.T) {
	t.Parallel()

	daemon := newPingTestDaemon(&ipnstate.PingResult{LatencySeconds: 0.05})
	adapter := newAdapterWithDaemon(daemon)
	adapter.pingAttempts = 1

	result, err := adapter.PingDevice(context.Background(), "peer-1")
	if err != nil {
		t.Fatalf("PingDevice() error: %v", err)
	}
	if result.Via != PingViaUnknown {
		t.Fatalf("route = %q, want %q", result.Via, PingViaUnknown)
	}
}

func TestPingDeviceKeepsLastRelayWhenLaterProbeFails(t *testing.T) {
	t.Parallel()

	daemon := newPingTestDaemon(
		&ipnstate.PingResult{LatencySeconds: 0.125, DERPRegionID: 10, DERPRegionCode: "hkg"},
		nil,
	)
	daemon.errors = []error{nil, errors.New("probe timed out")}
	adapter := newAdapterWithDaemon(daemon)
	adapter.pingInterval = 0

	result, err := adapter.PingDevice(context.Background(), "peer-1")
	if err != nil {
		t.Fatalf("PingDevice() error: %v", err)
	}
	if result.Via != PingViaRelay || result.RelayRegion != "hkg" || result.LatencyMS != 125 {
		t.Fatalf("PingDevice() result = %+v", result)
	}
	if daemon.pingCalls != 2 {
		t.Fatalf("ping calls = %d, want 2", daemon.pingCalls)
	}
}

func newPingTestDaemon(results ...*ipnstate.PingResult) *pingTestDaemon {
	return &pingTestDaemon{
		loginTestDaemon: &loginTestDaemon{},
		status: &ipnstate.Status{Peer: map[key.NodePublic]*ipnstate.PeerStatus{
			key.NodePublic{}: {
				ID:           tailcfg.StableNodeID("peer-1"),
				TailscaleIPs: []netip.Addr{netip.MustParseAddr("100.64.0.2")},
			},
		}},
		results: results,
	}
}
