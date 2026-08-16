package tailscale

import (
	"github.com/headscaleclient/headscaleclient/internal/domain"
	"tailscale.com/ipn"
	"tailscale.com/tailcfg"
)

func maskedPrefs(patch domain.PreferencePatch) *ipn.MaskedPrefs {
	result := &ipn.MaskedPrefs{}
	if patch.WantRunning != nil {
		result.WantRunning = *patch.WantRunning
		result.WantRunningSet = true
	}
	if patch.CorpDNS != nil {
		result.CorpDNS = *patch.CorpDNS
		result.CorpDNSSet = true
	}
	if patch.AcceptRoutes != nil {
		result.RouteAll = *patch.AcceptRoutes
		result.RouteAllSet = true
	}
	if patch.ShieldsUp != nil {
		result.ShieldsUp = *patch.ShieldsUp
		result.ShieldsUpSet = true
	}
	if patch.ExitNodeID != nil {
		result.ExitNodeID = tailcfg.StableNodeID(*patch.ExitNodeID)
		result.ExitNodeIDSet = true
	}
	if patch.ExitNodeAllowLANAccess != nil {
		result.ExitNodeAllowLANAccess = *patch.ExitNodeAllowLANAccess
		result.ExitNodeAllowLANAccessSet = true
	}
	return result
}

func mapPreferences(prefs *ipn.Prefs) domain.Preferences {
	if prefs == nil {
		return domain.Preferences{}
	}
	return domain.Preferences{
		WantRunning:            prefs.WantRunning,
		CorpDNS:                prefs.CorpDNS,
		AcceptRoutes:           prefs.RouteAll,
		ShieldsUp:              prefs.ShieldsUp,
		ExitNodeID:             string(prefs.ExitNodeID),
		ExitNodeAllowLANAccess: prefs.ExitNodeAllowLANAccess,
	}
}
