package config

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

func TestStoreLoadMissingReturnsDefaultsWithoutWriting(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "config.json")
	now := time.Date(2026, 8, 15, 1, 2, 3, 0, time.FixedZone("test", 8*60*60))
	store := newTestStore(t, path, func() time.Time { return now })

	configuration, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if configuration.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("schema version = %d, want %d", configuration.SchemaVersion, CurrentSchemaVersion)
	}
	if len(configuration.Endpoints) != 1 {
		t.Fatalf("endpoint count = %d, want 1", len(configuration.Endpoints))
	}
	official := configuration.Endpoints[0]
	if official.ID != OfficialEndpointID || official.BaseURL != OfficialEndpointURL || !official.BuiltIn {
		t.Fatalf("unexpected official endpoint: %+v", official)
	}
	if official.CreatedAt != "2026-08-14T17:02:03Z" {
		t.Fatalf("official timestamp = %q", official.CreatedAt)
	}
	if configuration.Settings != domain.DefaultAppSettings() {
		t.Fatalf("settings = %+v, want defaults %+v", configuration.Settings, domain.DefaultAppSettings())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Load() should not create the file, stat error: %v", err)
	}
}

func TestStoreUsesInstalledDefaultLanguageOnlyWhenLanguageIsMissing(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	store, err := NewStore(
		WithPath(path),
		WithDefaultLanguage(domain.LanguageEnglish),
	)
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}

	configuration, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(missing) error: %v", err)
	}
	if configuration.Settings.Language != domain.LanguageEnglish {
		t.Fatalf("missing configuration language = %q, want %q", configuration.Settings.Language, domain.LanguageEnglish)
	}

	legacy := `{
  "settings": {
    "theme": "system",
    "closeToTray": true,
    "notificationsEnabled": true,
    "checkForUpdates": true,
    "updateChannel": "stable"
  }
}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	configuration, err = store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(legacy) error: %v", err)
	}
	if configuration.Settings.Language != domain.LanguageEnglish {
		t.Fatalf("legacy configuration language = %q, want %q", configuration.Settings.Language, domain.LanguageEnglish)
	}

	configuration.Settings.Language = domain.LanguageChinese
	if _, err := store.SaveSettings(context.Background(), configuration.Settings); err != nil {
		t.Fatalf("SaveSettings() error: %v", err)
	}
	configuration, err = store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(explicit) error: %v", err)
	}
	if configuration.Settings.Language != domain.LanguageChinese {
		t.Fatalf("explicit language = %q, want %q", configuration.Settings.Language, domain.LanguageChinese)
	}
}

func TestStoreRejectsInvalidDefaultLanguage(t *testing.T) {
	t.Parallel()

	_, err := NewStore(WithDefaultLanguage(domain.Language("fr-FR")))
	assertErrorCode(t, err, domain.ErrorInvalidArgument)
}

func TestStoreEndpointLifecyclePersistsNormalizedConfiguration(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	now := time.Date(2026, 8, 15, 2, 0, 0, 0, time.UTC)
	store := newTestStore(t, path, func() time.Time { return now })

	created, err := store.SaveEndpoint(context.Background(), domain.ControlEndpointInput{
		Name:             "  Personal Headscale  ",
		BaseURL:          "HTTPS://HS.Example:443/control/",
		DaemonProfileIDs: []string{" profile-a ", "profile-a", "profile-b"},
	})
	if err != nil {
		t.Fatalf("SaveEndpoint(create) error: %v", err)
	}
	if !validUUID(created.ID) || created.ID == OfficialEndpointID {
		t.Fatalf("created endpoint has invalid ID %q", created.ID)
	}
	if created.Name != "Personal Headscale" || created.BaseURL != "https://hs.example/control" {
		t.Fatalf("endpoint was not normalized: %+v", created)
	}
	if created.Provider != domain.ProviderAuto {
		t.Fatalf("provider = %q, want auto", created.Provider)
	}
	if got := created.DaemonProfileIDs; len(got) != 2 || got[0] != "profile-a" || got[1] != "profile-b" {
		t.Fatalf("profile IDs = %#v", got)
	}
	if created.CreatedAt != now.Format(time.RFC3339Nano) || created.UpdatedAt != created.CreatedAt {
		t.Fatalf("unexpected timestamps: %+v", created)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	var disk Configuration
	if err := json.Unmarshal(data, &disk); err != nil {
		t.Fatalf("saved file is invalid JSON: %v", err)
	}
	if disk.SchemaVersion != CurrentSchemaVersion || len(disk.Endpoints) != 2 {
		t.Fatalf("unexpected saved configuration: %+v", disk)
	}

	now = now.Add(time.Hour)
	updated, err := store.SaveEndpoint(context.Background(), domain.ControlEndpointInput{
		ID: created.ID, Name: "Work Headscale", BaseURL: "https://work.example",
		Provider: domain.ProviderHeadscale,
	})
	if err != nil {
		t.Fatalf("SaveEndpoint(update) error: %v", err)
	}
	if updated.CreatedAt != created.CreatedAt || updated.UpdatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("update timestamps are wrong: created=%q updated=%q", updated.CreatedAt, updated.UpdatedAt)
	}
	if updated.Provider != domain.ProviderHeadscale {
		t.Fatalf("updated provider = %q", updated.Provider)
	}

	reloaded := newTestStore(t, path, func() time.Time { return now })
	endpoints, err := reloaded.ListEndpoints(context.Background())
	if err != nil {
		t.Fatalf("ListEndpoints() error: %v", err)
	}
	if len(endpoints) != 2 || endpoints[0].ID != OfficialEndpointID || endpoints[1].Name != "Work Headscale" {
		t.Fatalf("unexpected reloaded endpoints: %+v", endpoints)
	}
	if err := reloaded.DeleteEndpoint(context.Background(), created.ID); err != nil {
		t.Fatalf("DeleteEndpoint() error: %v", err)
	}
	endpoints, err = reloaded.ListEndpoints(context.Background())
	if err != nil {
		t.Fatalf("ListEndpoints() after delete error: %v", err)
	}
	if len(endpoints) != 1 || endpoints[0].ID != OfficialEndpointID {
		t.Fatalf("unexpected endpoints after delete: %+v", endpoints)
	}
}

func TestStoreRejectsBuiltInMutation(t *testing.T) {
	t.Parallel()

	store := newTestStore(t, filepath.Join(t.TempDir(), "config.json"), time.Now)
	_, err := store.SaveEndpoint(context.Background(), domain.ControlEndpointInput{
		ID: OfficialEndpointID, Name: "Changed", BaseURL: "https://example.com",
		Provider: domain.ProviderCompatible,
	})
	assertErrorCode(t, err, domain.ErrorPreconditionFailed)

	err = store.DeleteEndpoint(context.Background(), OfficialEndpointID)
	assertErrorCode(t, err, domain.ErrorPreconditionFailed)
}

func TestStoreValidationDoesNotCreateFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	store := newTestStore(t, path, time.Now)
	_, err := store.SaveEndpoint(context.Background(), domain.ControlEndpointInput{
		Name: "Unsafe", BaseURL: "http://headscale.example",
	})
	assertErrorCode(t, err, domain.ErrorInvalidArgument)
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("invalid save should not create config, stat error: %v", statErr)
	}
}

func TestStoreLoadsAndMigratesSchemaZero(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	legacy := `{
  "endpoints": [{
    "id": "11111111-1111-4111-8111-111111111111",
    "name": "Legacy",
    "baseURL": "https://LEGACY.example/",
    "provider": "auto"
  }],
  "settings": {"closeToTray": false}
}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	store := newTestStore(t, path, func() time.Time {
		return time.Date(2026, 8, 15, 3, 0, 0, 0, time.UTC)
	})
	configuration, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if configuration.SchemaVersion != CurrentSchemaVersion || len(configuration.Endpoints) != 2 {
		t.Fatalf("migration failed: %+v", configuration)
	}
	if configuration.Endpoints[1].BaseURL != "https://legacy.example" {
		t.Fatalf("legacy URL not normalized: %q", configuration.Endpoints[1].BaseURL)
	}
	if configuration.Settings.Theme != domain.ThemeSystem || configuration.Settings.Language != domain.LanguageChinese || configuration.Settings.UpdateChannel != domain.UpdateStable {
		t.Fatalf("legacy settings defaults not applied: %+v", configuration.Settings)
	}
	if configuration.Settings.CloseToTray {
		t.Fatal("explicit legacy closeToTray=false was not preserved")
	}
	if !configuration.Settings.NotificationsEnabled || !configuration.Settings.CheckForUpdates {
		t.Fatalf("omitted legacy booleans did not retain defaults: %+v", configuration.Settings)
	}
	migratedData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(migrated) error: %v", err)
	}
	var migrated Configuration
	if err := json.Unmarshal(migratedData, &migrated); err != nil {
		t.Fatalf("migrated file is invalid JSON: %v", err)
	}
	if migrated.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("persisted schema version = %d, want %d", migrated.SchemaVersion, CurrentSchemaVersion)
	}
}

func TestStoreRejectsUnsupportedAndMalformedConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		code domain.ErrorCode
	}{
		{name: "future schema", body: `{"schemaVersion": 99}`, code: domain.ErrorConfigurationUnsupported},
		{name: "malformed JSON", body: `{`, code: domain.ErrorConfigurationInvalid},
		{name: "duplicate IDs", body: `{
  "schemaVersion": 1,
  "endpoints": [
    {"id":"11111111-1111-4111-8111-111111111111","name":"One","baseURL":"https://one.example","provider":"auto"},
    {"id":"11111111-1111-4111-8111-111111111111","name":"Two","baseURL":"https://two.example","provider":"auto"}
  ],
  "settings":{"theme":"system","updateChannel":"stable"}
}`, code: domain.ErrorConfigurationInvalid},
		{name: "invalid timestamp", body: `{
  "schemaVersion": 1,
  "endpoints": [
    {"id":"11111111-1111-4111-8111-111111111111","name":"One","baseURL":"https://one.example","provider":"auto","createdAt":"yesterday"}
  ],
  "settings":{"theme":"system","updateChannel":"stable"}
}`, code: domain.ErrorConfigurationInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tt.body), 0o600); err != nil {
				t.Fatalf("WriteFile() error: %v", err)
			}
			store := newTestStore(t, path, time.Now)
			_, err := store.Load(context.Background())
			assertErrorCode(t, err, tt.code)
		})
	}
}

func TestStoreRepeatedAtomicWritesLeaveValidFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	store := newTestStore(t, path, time.Now)
	for i := 0; i < 5; i++ {
		settings := domain.DefaultAppSettings()
		settings.CloseToTray = i%2 == 0
		if _, err := store.SaveSettings(context.Background(), settings); err != nil {
			t.Fatalf("SaveSettings(%d) error: %v", i, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%d) error: %v", i, err)
		}
		var configuration Configuration
		if err := json.Unmarshal(data, &configuration); err != nil {
			t.Fatalf("write %d produced invalid JSON: %v", i, err)
		}
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(directory, ".config-*.tmp"))
	if err != nil {
		t.Fatalf("Glob() error: %v", err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary files remain: %v", temporaryFiles)
	}
}

func TestStoreHonorsCancelledContext(t *testing.T) {
	t.Parallel()

	store := newTestStore(t, filepath.Join(t.TempDir(), "config.json"), time.Now)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.Load(ctx)
	assertErrorCode(t, err, domain.ErrorCancelled)
}

func newTestStore(t *testing.T, path string, now func() time.Time) *Store {
	t.Helper()
	store, err := NewStore(WithPath(path), WithClock(now))
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}
	return store
}
