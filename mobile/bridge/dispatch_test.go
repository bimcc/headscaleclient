package bridge

import "testing"

func TestBoundaryRejectsInvalidRequestsWithoutCallingService(t *testing.T) {
	for _, tc := range []struct{ method, args string }{
		{"http://example.com", "[]"}, {"CallLocalAPI", `["prefs"]`}, {"SetAppSetting", `["launchAtLogin",true]`},
		{"GetSnapshot", "[1]"}, {"SetConnection", "[null]"}, {"SetConnection", `["true"]`},
		{"GetSnapshot", "null"}, {"SetTheme", "[null]"}, {"SaveEndpoint", "[null]"},
		{"SetPreference", `["acceptRoutes",null]`}, {"SwitchProfile", `[{}]`}, {"GetSnapshot", "{"},
	} {
		t.Run(tc.method+tc.args, func(t *testing.T) {
			if _, err := Call(nil, tc.method, tc.args); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
}
