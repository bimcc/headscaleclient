package tailscale

import (
	"testing"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

func pointer[T any](value T) *T { return &value }

func TestMaskedPrefsPreservesFalseAndEmptyValues(t *testing.T) {
	patch := domain.PreferencePatch{
		WantRunning:            pointer(false),
		CorpDNS:                pointer(false),
		AcceptRoutes:           pointer(false),
		ShieldsUp:              pointer(false),
		ExitNodeID:             pointer(""),
		ExitNodeAllowLANAccess: pointer(false),
	}
	prefs := maskedPrefs(patch)

	checks := map[string]bool{
		"WantRunningSet":            prefs.WantRunningSet,
		"CorpDNSSet":                prefs.CorpDNSSet,
		"RouteAllSet":               prefs.RouteAllSet,
		"ShieldsUpSet":              prefs.ShieldsUpSet,
		"ExitNodeIDSet":             prefs.ExitNodeIDSet,
		"ExitNodeAllowLANAccessSet": prefs.ExitNodeAllowLANAccessSet,
	}
	for name, value := range checks {
		if !value {
			t.Errorf("%s is false", name)
		}
	}
	if prefs.WantRunning || prefs.CorpDNS || prefs.RouteAll || prefs.ShieldsUp ||
		prefs.ExitNodeID != "" || prefs.ExitNodeAllowLANAccess {
		t.Fatal("false and empty values were not preserved")
	}
}

func TestMaskedPrefsLeavesOmittedFieldsUnset(t *testing.T) {
	prefs := maskedPrefs(domain.PreferencePatch{CorpDNS: pointer(true)})
	if !prefs.CorpDNSSet || !prefs.CorpDNS {
		t.Fatal("CorpDNS update was not mapped")
	}
	if prefs.WantRunningSet || prefs.RouteAllSet || prefs.ExitNodeIDSet {
		t.Fatal("omitted fields were unexpectedly marked as set")
	}
}
