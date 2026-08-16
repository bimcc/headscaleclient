package config

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

const (
	CurrentSchemaVersion = 1
	OfficialEndpointID   = "00000000-0000-4000-8000-000000000001"
	OfficialEndpointURL  = "https://controlplane.tailscale.com"
	configDirectoryName  = "HeadscaleClient"
	configFileName       = "config.json"
	maxEndpointName      = 100
	maxCustomCARef       = 512
	maxProfileIDs        = 256
	maxProfileIDLength   = 256
)

type Configuration struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Endpoints     []domain.ControlEndpoint `json:"endpoints"`
	Settings      domain.AppSettings       `json:"settings"`
}

type Store struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

type Option func(*Store) error

func WithPath(path string) Option {
	return func(store *Store) error {
		path = strings.TrimSpace(path)
		if path == "" {
			return domain.NewError(domain.ErrorInvalidArgument, "Configuration path cannot be empty.").WithDetail("path")
		}
		store.path = filepath.Clean(path)
		return nil
	}
}

func WithClock(now func() time.Time) Option {
	return func(store *Store) error {
		if now == nil {
			return domain.NewError(domain.ErrorInvalidArgument, "Configuration clock cannot be nil.").WithDetail("clock")
		}
		store.now = now
		return nil
	}
}

func NewStore(options ...Option) (*Store, error) {
	store := &Store{now: time.Now}
	for _, option := range options {
		if option == nil {
			return nil, domain.NewError(domain.ErrorInvalidArgument, "Configuration option cannot be nil.")
		}
		if err := option(store); err != nil {
			return nil, err
		}
	}
	if store.path == "" {
		path, err := DefaultPath()
		if err != nil {
			return nil, err
		}
		store.path = path
	}
	return store, nil
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", domain.WrapError(domain.ErrorConfigurationReadFailed, "Could not locate the user configuration directory.", err)
	}
	return filepath.Join(dir, configDirectoryName, configFileName), nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Load(ctx context.Context) (Configuration, error) {
	if err := contextError(ctx); err != nil {
		return Configuration{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked(ctx)
}

func (s *Store) ListEndpoints(ctx context.Context) ([]domain.ControlEndpoint, error) {
	configuration, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	return append([]domain.ControlEndpoint(nil), configuration.Endpoints...), nil
}

func (s *Store) SaveEndpoint(ctx context.Context, input domain.ControlEndpointInput) (domain.ControlEndpoint, error) {
	if err := contextError(ctx); err != nil {
		return domain.ControlEndpoint{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	configuration, err := s.loadLocked(ctx)
	if err != nil {
		return domain.ControlEndpoint{}, err
	}
	normalized, err := normalizeEndpointInput(input)
	if err != nil {
		return domain.ControlEndpoint{}, err
	}

	now := s.now().UTC().Format(time.RFC3339Nano)
	if normalized.ID == "" {
		normalized.ID, err = newUUID()
		if err != nil {
			return domain.ControlEndpoint{}, domain.WrapError(domain.ErrorInternal, "Could not create an endpoint identifier.", err)
		}
		endpoint := domain.ControlEndpoint{
			ID:               normalized.ID,
			Name:             normalized.Name,
			BaseURL:          normalized.BaseURL,
			Provider:         normalized.Provider,
			CustomCARef:      normalized.CustomCARef,
			DaemonProfileIDs: normalized.DaemonProfileIDs,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		configuration.Endpoints = append(configuration.Endpoints, endpoint)
		if err := s.writeLocked(ctx, configuration); err != nil {
			return domain.ControlEndpoint{}, err
		}
		return endpoint, nil
	}

	for i := range configuration.Endpoints {
		endpoint := &configuration.Endpoints[i]
		if endpoint.ID != normalized.ID {
			continue
		}
		if endpoint.BuiltIn {
			return domain.ControlEndpoint{}, domain.NewError(domain.ErrorPreconditionFailed, "The built-in Tailscale endpoint cannot be edited.")
		}
		endpoint.Name = normalized.Name
		endpoint.BaseURL = normalized.BaseURL
		endpoint.Provider = normalized.Provider
		endpoint.CustomCARef = normalized.CustomCARef
		endpoint.DaemonProfileIDs = normalized.DaemonProfileIDs
		endpoint.UpdatedAt = now
		if err := s.writeLocked(ctx, configuration); err != nil {
			return domain.ControlEndpoint{}, err
		}
		return *endpoint, nil
	}
	return domain.ControlEndpoint{}, domain.NewError(domain.ErrorNotFound, "Control server was not found.")
}

func (s *Store) DeleteEndpoint(ctx context.Context, endpointID string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	endpointID = strings.ToLower(strings.TrimSpace(endpointID))
	if !validUUID(endpointID) {
		return domain.NewError(domain.ErrorInvalidArgument, "Endpoint identifier is invalid.").WithDetail("endpointID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	configuration, err := s.loadLocked(ctx)
	if err != nil {
		return err
	}
	for i, endpoint := range configuration.Endpoints {
		if endpoint.ID != endpointID {
			continue
		}
		if endpoint.BuiltIn {
			return domain.NewError(domain.ErrorPreconditionFailed, "The built-in Tailscale endpoint cannot be removed.")
		}
		configuration.Endpoints = append(configuration.Endpoints[:i], configuration.Endpoints[i+1:]...)
		return s.writeLocked(ctx, configuration)
	}
	return domain.NewError(domain.ErrorNotFound, "Control server was not found.")
}

func (s *Store) GetSettings(ctx context.Context) (domain.AppSettings, error) {
	configuration, err := s.Load(ctx)
	if err != nil {
		return domain.AppSettings{}, err
	}
	return configuration.Settings, nil
}

func (s *Store) SaveSettings(ctx context.Context, settings domain.AppSettings) (domain.AppSettings, error) {
	if err := contextError(ctx); err != nil {
		return domain.AppSettings{}, err
	}
	if !settings.Valid() {
		return domain.AppSettings{}, domain.NewError(domain.ErrorInvalidArgument, "Application settings are invalid.").WithDetail("settings")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	configuration, err := s.loadLocked(ctx)
	if err != nil {
		return domain.AppSettings{}, err
	}
	configuration.Settings = settings
	if err := s.writeLocked(ctx, configuration); err != nil {
		return domain.AppSettings{}, err
	}
	return settings, nil
}

func (s *Store) loadLocked(ctx context.Context) (Configuration, error) {
	if err := contextError(ctx); err != nil {
		return Configuration{}, err
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.defaultConfiguration(), nil
	}
	if err != nil {
		return Configuration{}, domain.WrapError(domain.ErrorConfigurationReadFailed, "Could not read application configuration.", err).WithRetryable(true)
	}
	if len(data) == 0 {
		return Configuration{}, domain.NewError(domain.ErrorConfigurationInvalid, "Application configuration is empty.")
	}

	configuration := Configuration{Settings: domain.DefaultAppSettings()}
	if err := json.Unmarshal(data, &configuration); err != nil {
		return Configuration{}, domain.WrapError(domain.ErrorConfigurationInvalid, "Application configuration is not valid JSON.", err)
	}
	originalSchemaVersion := configuration.SchemaVersion
	if originalSchemaVersion > CurrentSchemaVersion {
		return Configuration{}, domain.NewError(domain.ErrorConfigurationUnsupported, "Application configuration was created by a newer version.")
	}
	if configuration.SchemaVersion < 0 {
		return Configuration{}, domain.NewError(domain.ErrorConfigurationInvalid, "Application configuration schema is invalid.")
	}

	configuration.SchemaVersion = CurrentSchemaVersion
	if configuration.Settings.Theme == "" {
		configuration.Settings.Theme = domain.ThemeSystem
	}
	if configuration.Settings.Language == "" {
		configuration.Settings.Language = domain.LanguageChinese
	}
	if configuration.Settings.UpdateChannel == "" {
		configuration.Settings.UpdateChannel = domain.UpdateStable
	}
	if !configuration.Settings.Valid() {
		return Configuration{}, domain.NewError(domain.ErrorConfigurationInvalid, "Application settings in the configuration are invalid.")
	}
	if err := s.normalizeLoadedEndpoints(&configuration); err != nil {
		return Configuration{}, err
	}
	if originalSchemaVersion < CurrentSchemaVersion {
		if err := s.writeLocked(ctx, configuration); err != nil {
			return Configuration{}, err
		}
	}
	return configuration, nil
}

func (s *Store) writeLocked(ctx context.Context, configuration Configuration) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	configuration.SchemaVersion = CurrentSchemaVersion
	if !configuration.Settings.Valid() {
		return domain.NewError(domain.ErrorConfigurationInvalid, "Application settings are invalid.")
	}
	if err := s.normalizeLoadedEndpoints(&configuration); err != nil {
		return err
	}
	data, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return domain.WrapError(domain.ErrorConfigurationWriteFailed, "Could not encode application configuration.", err)
	}
	data = append(data, '\n')
	if err := writeFileAtomic(s.path, data, 0o600); err != nil {
		return domain.WrapError(domain.ErrorConfigurationWriteFailed, "Could not save application configuration.", err).WithRetryable(true)
	}
	return nil
}

func (s *Store) defaultConfiguration() Configuration {
	return Configuration{
		SchemaVersion: CurrentSchemaVersion,
		Endpoints:     []domain.ControlEndpoint{officialEndpoint(s.now().UTC().Format(time.RFC3339Nano))},
		Settings:      domain.DefaultAppSettings(),
	}
}

func (s *Store) normalizeLoadedEndpoints(configuration *Configuration) error {
	seen := make(map[string]struct{}, len(configuration.Endpoints)+1)
	normalized := make([]domain.ControlEndpoint, 0, len(configuration.Endpoints)+1)
	var official *domain.ControlEndpoint
	now := s.now().UTC().Format(time.RFC3339Nano)

	for _, endpoint := range configuration.Endpoints {
		endpoint.ID = strings.ToLower(strings.TrimSpace(endpoint.ID))
		if !validUUID(endpoint.ID) {
			return domain.NewError(domain.ErrorConfigurationInvalid, "Configuration contains an invalid endpoint identifier.")
		}
		if _, duplicate := seen[endpoint.ID]; duplicate {
			return domain.NewError(domain.ErrorConfigurationInvalid, "Configuration contains duplicate endpoint identifiers.")
		}
		seen[endpoint.ID] = struct{}{}
		if endpoint.ID == OfficialEndpointID {
			canonical := officialEndpoint(now)
			profileIDs, err := normalizeProfileIDs(endpoint.DaemonProfileIDs)
			if err != nil {
				return domain.WrapError(domain.ErrorConfigurationInvalid, "Configuration contains invalid daemon profile associations.", err)
			}
			canonical.DaemonProfileIDs = profileIDs
			if endpoint.CreatedAt != "" {
				if !validTimestamp(endpoint.CreatedAt) {
					return domain.NewError(domain.ErrorConfigurationInvalid, "Configuration contains an invalid endpoint timestamp.")
				}
				canonical.CreatedAt = endpoint.CreatedAt
			}
			if endpoint.UpdatedAt != "" {
				if !validTimestamp(endpoint.UpdatedAt) {
					return domain.NewError(domain.ErrorConfigurationInvalid, "Configuration contains an invalid endpoint timestamp.")
				}
				canonical.UpdatedAt = endpoint.UpdatedAt
			}
			official = &canonical
			continue
		}

		input, err := normalizeEndpointInput(domain.ControlEndpointInput{
			ID: endpoint.ID, Name: endpoint.Name, BaseURL: endpoint.BaseURL,
			Provider: endpoint.Provider, CustomCARef: endpoint.CustomCARef,
			DaemonProfileIDs: endpoint.DaemonProfileIDs,
		})
		if err != nil {
			return domain.WrapError(domain.ErrorConfigurationInvalid, "Configuration contains an invalid control server.", err)
		}
		endpoint.Name = input.Name
		endpoint.BaseURL = input.BaseURL
		endpoint.Provider = input.Provider
		endpoint.CustomCARef = input.CustomCARef
		endpoint.DaemonProfileIDs = input.DaemonProfileIDs
		endpoint.BuiltIn = false
		if endpoint.CreatedAt == "" {
			endpoint.CreatedAt = now
		} else if !validTimestamp(endpoint.CreatedAt) {
			return domain.NewError(domain.ErrorConfigurationInvalid, "Configuration contains an invalid endpoint timestamp.")
		}
		if endpoint.UpdatedAt == "" {
			endpoint.UpdatedAt = endpoint.CreatedAt
		} else if !validTimestamp(endpoint.UpdatedAt) {
			return domain.NewError(domain.ErrorConfigurationInvalid, "Configuration contains an invalid endpoint timestamp.")
		}
		normalized = append(normalized, endpoint)
	}

	if official == nil {
		value := officialEndpoint(now)
		official = &value
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return strings.ToLower(normalized[i].Name) < strings.ToLower(normalized[j].Name)
	})
	configuration.Endpoints = append([]domain.ControlEndpoint{*official}, normalized...)
	return nil
}

func officialEndpoint(timestamp string) domain.ControlEndpoint {
	return domain.ControlEndpoint{
		ID: OfficialEndpointID, Name: "Tailscale", BaseURL: OfficialEndpointURL,
		Provider: domain.ProviderTailscale, DaemonProfileIDs: []string{},
		BuiltIn: true, CreatedAt: timestamp, UpdatedAt: timestamp,
	}
}

func normalizeEndpointInput(input domain.ControlEndpointInput) (domain.ControlEndpointInput, error) {
	input.ID = strings.ToLower(strings.TrimSpace(input.ID))
	if input.ID != "" && !validUUID(input.ID) {
		return domain.ControlEndpointInput{}, domain.NewError(domain.ErrorInvalidArgument, "Endpoint identifier is invalid.").WithDetail("id")
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return domain.ControlEndpointInput{}, domain.NewError(domain.ErrorInvalidArgument, "Control server name is required.").WithDetail("name")
	}
	if len(input.Name) > maxEndpointName {
		return domain.ControlEndpointInput{}, domain.NewError(domain.ErrorInvalidArgument, "Control server name is too long.").WithDetail("name")
	}
	if containsControlCharacter(input.Name) {
		return domain.ControlEndpointInput{}, domain.NewError(domain.ErrorInvalidArgument, "Control server name contains invalid characters.").WithDetail("name")
	}
	if input.Provider == "" {
		input.Provider = domain.ProviderAuto
	}
	if !input.Provider.Valid() {
		return domain.ControlEndpointInput{}, domain.NewError(domain.ErrorInvalidArgument, "Control server provider is invalid.").WithDetail("provider")
	}
	baseURL, err := NormalizeControlURL(input.BaseURL)
	if err != nil {
		return domain.ControlEndpointInput{}, err
	}
	input.BaseURL = baseURL
	input.CustomCARef = strings.TrimSpace(input.CustomCARef)
	if len(input.CustomCARef) > maxCustomCARef {
		return domain.ControlEndpointInput{}, domain.NewError(domain.ErrorInvalidArgument, "Custom CA reference is too long.").WithDetail("customCARef")
	}
	if containsControlCharacter(input.CustomCARef) {
		return domain.ControlEndpointInput{}, domain.NewError(domain.ErrorInvalidArgument, "Custom CA reference contains invalid characters.").WithDetail("customCARef")
	}
	input.DaemonProfileIDs, err = normalizeProfileIDs(input.DaemonProfileIDs)
	if err != nil {
		return domain.ControlEndpointInput{}, err
	}
	return input, nil
}

func normalizeProfileIDs(ids []string) ([]string, error) {
	if len(ids) > maxProfileIDs {
		return nil, domain.NewError(domain.ErrorInvalidArgument, "Too many daemon profiles are associated with the control server.").WithDetail("daemonProfileIDs")
	}
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || len(id) > maxProfileIDLength || containsControlCharacter(id) {
			return nil, domain.NewError(domain.ErrorInvalidArgument, "Daemon profile identifier is invalid.").WithDetail("daemonProfileIDs")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	if result == nil {
		result = []string{}
	}
	return result, nil
}

func validUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				return false
			}
		}
	}
	return true
}

func newUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

func validTimestamp(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}

func containsControlCharacter(value string) bool {
	return strings.IndexFunc(value, unicode.IsControl) >= 0
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return domain.NewError(domain.ErrorInvalidArgument, "Context cannot be nil.")
	}
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return domain.WrapError(domain.ErrorTimeout, "The configuration operation timed out.", ctx.Err()).WithRetryable(true)
		}
		return domain.WrapError(domain.ErrorCancelled, "The configuration operation was cancelled.", ctx.Err())
	default:
		return nil
	}
}
