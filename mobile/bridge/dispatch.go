// Package bridge is the allowlisted JSON boundary shared by Android and tests.
package bridge

import (
	"encoding/json"
	"errors"

	"github.com/headscaleclient/headscaleclient/internal/application"
	"github.com/headscaleclient/headscaleclient/internal/domain"
)

// Call never accepts a URL, LocalAPI path, command or reflected method name.
func Call(service *application.Service, method, arguments string) (any, error) {
	var args []json.RawMessage
	if len(arguments) > 64*1024 {
		return nil, errors.New("request too large")
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, err
	}
	counts := map[string]int{"GetSnapshot": 0, "EnsureDaemon": 0, "SetConnection": 1,
		"SetPreference": 2, "SetExitNode": 1, "PingDevice": 1, "SaveEndpoint": 1,
		"DeleteEndpoint": 1, "SwitchProfile": 1, "Logout": 0, "BeginLogin": 1,
		"SetTheme": 1, "SetLanguage": 1}
	n, ok := counts[method]
	if !ok {
		return nil, errors.New("unsupported Android operation")
	}
	if len(args) != n {
		return nil, errors.New("invalid argument count")
	}
	var str string
	var enabled bool
	if n > 0 && method != "SetConnection" && method != "SetExitNode" && method != "SaveEndpoint" {
		if err := json.Unmarshal(args[0], &str); err != nil {
			return nil, err
		}
	}
	if method == "SetConnection" || method == "SetPreference" {
		if string(args[n-1]) == "null" {
			return nil, errors.New("boolean required")
		}
		if err := json.Unmarshal(args[n-1], &enabled); err != nil {
			return nil, err
		}
	}
	switch method {
	case "GetSnapshot", "EnsureDaemon":
		return service.GetSnapshot()
	case "SetConnection":
		return service.SetConnection(enabled)
	case "SetPreference":
		return service.SetPreference(application.PreferenceKey(str), enabled)
	case "SetExitNode":
		var id *string
		if err := json.Unmarshal(args[0], &id); err != nil {
			return nil, err
		}
		return service.SetExitNode(id)
	case "PingDevice":
		return service.PingDevice(str)
	case "SaveEndpoint":
		var input application.EndpointInput
		if err := json.Unmarshal(args[0], &input); err != nil {
			return nil, err
		}
		return service.SaveEndpoint(input)
	case "DeleteEndpoint":
		return service.DeleteEndpoint(str)
	case "SwitchProfile":
		return service.SwitchProfile(str)
	case "Logout":
		return service.Logout()
	case "BeginLogin":
		return service.BeginLogin(str)
	case "SetTheme":
		return service.SetTheme(domain.Theme(str))
	case "SetLanguage":
		return service.SetLanguage(domain.Language(str))
	}
	return nil, errors.New("unsupported operation")
}
