package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigPath_ReturnsNonEmpty(t *testing.T) {
	p := ConfigPath()
	if p == "" {
		t.Fatal("ConfigPath() returned empty string")
	}

	expected := filepath.Join(appName, "config.yaml")
	if !hasSuffix(p, expected) {
		t.Errorf("ConfigPath() = %q, want suffix %q", p, expected)
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func TestConfigPathContainsAppName(t *testing.T) {
	p := ConfigPath()
	dir := filepath.Dir(p)
	base := filepath.Base(dir)
	if base != appName {
		t.Errorf("ConfigPath() parent dir = %q, want %q", base, appName)
	}

	if filepath.Base(p) != "config.yaml" {
		t.Errorf("ConfigPath() filename = %q, want %q", filepath.Base(p), "config.yaml")
	}
}

func TestAppNameConstant(t *testing.T) {
	if appName != "trbooksearch" {
		t.Errorf("appName = %q, want %q", appName, "trbooksearch")
	}
}

func TestFirecrawlConfigZeroValue(t *testing.T) {
	var cfg Config
	if cfg.Firecrawl.APIKey != "" {
		t.Errorf("zero value APIKey = %q, want empty", cfg.Firecrawl.APIKey)
	}
	if cfg.Firecrawl.APIURL != "" {
		t.Errorf("zero value APIURL = %q, want empty", cfg.Firecrawl.APIURL)
	}
}

func TestConfigYAMLRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "empty config", cfg: Config{}},
		{
			name: "api key only",
			cfg:  Config{Firecrawl: FirecrawlConfig{APIKey: "fc-test-key-123"}},
		},
		{
			name: "full config",
			cfg: Config{Firecrawl: FirecrawlConfig{
				APIKey: "fc-prod-key-456",
				APIURL: "https://custom.firecrawl.dev",
			}},
		},
		{
			name: "custom url without key",
			cfg:  Config{Firecrawl: FirecrawlConfig{APIURL: "http://localhost:3000"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.cfg)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var got Config
			if err := yaml.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			if got.Firecrawl.APIKey != tt.cfg.Firecrawl.APIKey {
				t.Errorf("APIKey = %q, want %q", got.Firecrawl.APIKey, tt.cfg.Firecrawl.APIKey)
			}
			if got.Firecrawl.APIURL != tt.cfg.Firecrawl.APIURL {
				t.Errorf("APIURL = %q, want %q", got.Firecrawl.APIURL, tt.cfg.Firecrawl.APIURL)
			}
		})
	}
}

func TestConfigYAMLFieldNames(t *testing.T) {
	cfg := Config{
		Firecrawl: FirecrawlConfig{
			APIKey: "the-key",
			APIURL: "the-url",
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]map[string]string
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	fc, ok := raw["firecrawl"]
	if !ok {
		t.Fatalf("missing top-level key 'firecrawl' in YAML: %s", data)
	}

	if v, ok := fc["api_key"]; !ok || v != "the-key" {
		t.Errorf("api_key = %q (present=%v), want %q", v, ok, "the-key")
	}
	if v, ok := fc["api_url"]; !ok || v != "the-url" {
		t.Errorf("api_url = %q (present=%v), want %q", v, ok, "the-url")
	}
}

func TestConfigUnmarshalFromRawYAML(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		wantKey   string
		wantURL   string
		wantError bool
	}{
		{
			name:    "complete config",
			yaml:    "firecrawl:\n  api_key: mykey\n  api_url: https://example.com\n",
			wantKey: "mykey",
			wantURL: "https://example.com",
		},
		{
			name:    "key only",
			yaml:    "firecrawl:\n  api_key: onlykey\n",
			wantKey: "onlykey",
			wantURL: "",
		},
		{
			name: "empty yaml",
			yaml: "",
		},
		{
			name: "empty firecrawl section",
			yaml: "firecrawl:\n",
		},
		{
			name:      "invalid yaml",
			yaml:      "firecrawl: [broken: yaml: {{{\n",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := yaml.Unmarshal([]byte(tt.yaml), &cfg)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if cfg.Firecrawl.APIKey != tt.wantKey {
				t.Errorf("APIKey = %q, want %q", cfg.Firecrawl.APIKey, tt.wantKey)
			}
			if cfg.Firecrawl.APIURL != tt.wantURL {
				t.Errorf("APIURL = %q, want %q", cfg.Firecrawl.APIURL, tt.wantURL)
			}
		})
	}
}

// Tests for loadFrom and saveTo (internal helpers that accept a config dir).

func TestLoadFrom_NoFileReturnsEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, err := loadFrom(tmpDir)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}
	if cfg.Firecrawl.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.Firecrawl.APIKey)
	}
	// When no file exists, no default URL is applied (zero Config returned)
	if cfg.Firecrawl.APIURL != "" {
		t.Errorf("APIURL = %q, want empty for missing file", cfg.Firecrawl.APIURL)
	}
}

func TestLoadFrom_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, appName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	content := "firecrawl:\n  api_key: test-key\n  api_url: https://custom.dev\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadFrom(tmpDir)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}
	if cfg.Firecrawl.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.Firecrawl.APIKey, "test-key")
	}
	if cfg.Firecrawl.APIURL != "https://custom.dev" {
		t.Errorf("APIURL = %q, want %q", cfg.Firecrawl.APIURL, "https://custom.dev")
	}
}

func TestLoadFrom_DefaultAPIURL(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, appName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	content := "firecrawl:\n  api_key: k\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadFrom(tmpDir)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}
	if cfg.Firecrawl.APIURL != "https://api.firecrawl.dev" {
		t.Errorf("APIURL = %q, want default", cfg.Firecrawl.APIURL)
	}
}

func TestLoadFrom_CorruptYAML(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, appName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("[[[invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadFrom(tmpDir)
	if err == nil {
		t.Fatal("expected error for corrupt YAML, got nil")
	}
}

func TestLoadFrom_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission test not meaningful")
	}

	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, appName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("firecrawl:\n"), 0000); err != nil {
		t.Fatal(err)
	}

	_, err := loadFrom(tmpDir)
	if err == nil {
		t.Fatal("expected error for unreadable file, got nil")
	}
}

func TestSaveTo_CreatesDirectoryAndFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{
		Firecrawl: FirecrawlConfig{
			APIKey: "save-test-key",
			APIURL: "https://save.dev",
		},
	}

	if err := saveTo(cfg, tmpDir); err != nil {
		t.Fatalf("saveTo: %v", err)
	}

	path := filepath.Join(tmpDir, appName, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if loaded.Firecrawl.APIKey != cfg.Firecrawl.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.Firecrawl.APIKey, cfg.Firecrawl.APIKey)
	}
	if loaded.Firecrawl.APIURL != cfg.Firecrawl.APIURL {
		t.Errorf("APIURL = %q, want %q", loaded.Firecrawl.APIURL, cfg.Firecrawl.APIURL)
	}
}

func TestSaveTo_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{Firecrawl: FirecrawlConfig{APIKey: "secret"}}

	if err := saveTo(cfg, tmpDir); err != nil {
		t.Fatalf("saveTo: %v", err)
	}

	path := filepath.Join(tmpDir, appName, "config.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestSaveToAndLoadFrom_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	original := Config{
		Firecrawl: FirecrawlConfig{
			APIKey: "round-trip-key",
			APIURL: "https://round-trip.dev",
		},
	}

	if err := saveTo(original, tmpDir); err != nil {
		t.Fatalf("saveTo: %v", err)
	}

	loaded, err := loadFrom(tmpDir)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}

	if loaded.Firecrawl.APIKey != original.Firecrawl.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.Firecrawl.APIKey, original.Firecrawl.APIKey)
	}
	if loaded.Firecrawl.APIURL != original.Firecrawl.APIURL {
		t.Errorf("APIURL = %q, want %q", loaded.Firecrawl.APIURL, original.Firecrawl.APIURL)
	}
}

func TestSaveToAndLoadFrom_EmptyURLGetsDefault(t *testing.T) {
	tmpDir := t.TempDir()
	original := Config{
		Firecrawl: FirecrawlConfig{APIKey: "nourl-key"},
	}

	if err := saveTo(original, tmpDir); err != nil {
		t.Fatalf("saveTo: %v", err)
	}

	loaded, err := loadFrom(tmpDir)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}

	if loaded.Firecrawl.APIKey != "nourl-key" {
		t.Errorf("APIKey = %q, want %q", loaded.Firecrawl.APIKey, "nourl-key")
	}
	if loaded.Firecrawl.APIURL != "https://api.firecrawl.dev" {
		t.Errorf("APIURL = %q, want default", loaded.Firecrawl.APIURL)
	}
}

func TestSaveTo_OverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()

	first := Config{Firecrawl: FirecrawlConfig{APIKey: "first"}}
	if err := saveTo(first, tmpDir); err != nil {
		t.Fatalf("saveTo first: %v", err)
	}

	second := Config{Firecrawl: FirecrawlConfig{APIKey: "second"}}
	if err := saveTo(second, tmpDir); err != nil {
		t.Fatalf("saveTo second: %v", err)
	}

	loaded, err := loadFrom(tmpDir)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}

	if loaded.Firecrawl.APIKey != "second" {
		t.Errorf("APIKey = %q, want %q", loaded.Firecrawl.APIKey, "second")
	}
}

func TestSaveTo_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Config{}

	if err := saveTo(cfg, tmpDir); err != nil {
		t.Fatalf("saveTo: %v", err)
	}

	loaded, err := loadFrom(tmpDir)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}

	// Empty URL should get default on load
	if loaded.Firecrawl.APIURL != "https://api.firecrawl.dev" {
		t.Errorf("APIURL = %q, want default", loaded.Firecrawl.APIURL)
	}
}

// Tests for the public Load/Save that go through os.UserConfigDir().
// These don't write to the real config since we just verify they don't crash.
func TestLoad_DoesNotCrash(t *testing.T) {
	// Load from the real config directory - should either work or return zero config
	_, err := Load()
	if err != nil {
		// This can fail if the config file is corrupt, which is fine for unit tests
		t.Logf("Load returned error (may be expected): %v", err)
	}
}

func TestConfigPath_NonEmpty(t *testing.T) {
	p := ConfigPath()
	if p == "" {
		t.Fatal("ConfigPath() returned empty")
	}
}
