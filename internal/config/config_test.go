package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Host != "api.flotio.ovh" {
		t.Errorf("expected host api.flotio.ovh, got %s", cfg.Host)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected log level info, got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("expected log format text, got %s", cfg.Logging.Format)
	}
}

func TestResolveHost(t *testing.T) {
	origHost := os.Getenv("FLOTIO_HOST")
	origFlag := FlagHost
	defer func() {
		os.Setenv("FLOTIO_HOST", origHost)
		FlagHost = origFlag
	}()

	// Default (Host set via config file)
	os.Unsetenv("FLOTIO_HOST")
	FlagHost = ""
	cfg := DefaultConfig()
	if got := cfg.ResolveHost(); got != "https://api.flotio.ovh" {
		t.Errorf("default: expected https://api.flotio.ovh, got %s", got)
	}

	// Env overrides config file
	os.Setenv("FLOTIO_HOST", "custom.example.com")
	cfg.Host = "" // clear config value so env kicks in
	if got := cfg.ResolveHost(); got != "https://custom.example.com" {
		t.Errorf("env: expected https://custom.example.com, got %s", got)
	}

	// Env with explicit scheme
	os.Setenv("FLOTIO_HOST", "http://localhost:8080")
	if got := cfg.ResolveHost(); got != "http://localhost:8080" {
		t.Errorf("env+scheme: expected http://localhost:8080, got %s", got)
	}

	// Flag overrides everything
	FlagHost = "flag.example.com"
	if got := cfg.ResolveHost(); got != "https://flag.example.com" {
		t.Errorf("flag: expected https://flag.example.com, got %s", got)
	}
}

func TestResolveHostFromConfigFileValue(t *testing.T) {
	origFlag := FlagHost
	origHost := os.Getenv("FLOTIO_HOST")
	defer func() {
		FlagHost = origFlag
		os.Setenv("FLOTIO_HOST", origHost)
	}()
	FlagHost = ""
	os.Unsetenv("FLOTIO_HOST")

	cfg := &Config{Host: "config.example.com"}
	if got := cfg.ResolveHost(); got != "https://config.example.com" {
		t.Errorf("expected https://config.example.com, got %s", got)
	}

	// Config with explicit scheme
	cfg = &Config{Host: "http://config.example.com:8080"}
	if got := cfg.ResolveHost(); got != "http://config.example.com:8080" {
		t.Errorf("with scheme: expected http://config.example.com:8080, got %s", got)
	}
}

func TestResolveHostOnly(t *testing.T) {
	origFlag := FlagHost
	origHost := os.Getenv("FLOTIO_HOST")
	defer func() {
		FlagHost = origFlag
		os.Setenv("FLOTIO_HOST", origHost)
	}()
	FlagHost = ""
	os.Unsetenv("FLOTIO_HOST")

	// Plain hostname
	cfg := DefaultConfig()
	if got := cfg.ResolveHostOnly(); got != "api.flotio.ovh" {
		t.Errorf("expected api.flotio.ovh, got %s", got)
	}

	// Full URL should extract hostname
	cfg.Host = "http://localhost:8080"
	if got := cfg.ResolveHostOnly(); got != "localhost:8080" {
		t.Errorf("expected localhost:8080, got %s", got)
	}
}

func TestFindProject(t *testing.T) {
	dir := t.TempDir()

	// Create .flotio.yaml
	if err := SaveProject(dir, &ProjectConfig{ProjectID: 42}); err != nil {
		t.Fatal(err)
	}

	pc, found := FindProject(dir)
	if pc == nil {
		t.Fatal("expected project config")
	}
	if pc.ProjectID != 42 {
		t.Errorf("expected project ID 42, got %d", pc.ProjectID)
	}
	if found != dir {
		t.Errorf("expected dir %s, got %s", dir, found)
	}

	// Walking up from subdirectory
	sub := filepath.Join(dir, "sub", "deep")
	os.MkdirAll(sub, 0755)
	pc, found = FindProject(sub)
	if pc == nil || pc.ProjectID != 42 {
		t.Error("expected to find project from subdirectory")
	}
}

func TestFindProjectInvalid(t *testing.T) {
	dir := t.TempDir()

	// Invalid YAML in current dir should be skipped.
	// FindProject may still find a valid one from a parent dir,
	// so we only check that the invalid file itself is not returned.
	os.WriteFile(filepath.Join(dir, ".flotio.yaml"), []byte("garbage"), 0644)
	pc, found := FindProject(dir)
	if pc != nil && found == dir {
		t.Error("invalid yaml in current dir should be skipped")
	}

	// Missing project_id should also be skipped
	os.WriteFile(filepath.Join(dir, ".flotio.yaml"), []byte("name: test"), 0644)
	pc, found = FindProject(dir)
	if pc != nil && found == dir {
		t.Error("missing project_id in current dir should be skipped")
	}
}

func TestResolveProjectID(t *testing.T) {
	dir := t.TempDir()

	// Without .flotio.yaml in this specific dir → error
	// (may find one from parent if it exists, so skip if found)
	_, err := ResolveProjectID(0)
	if err == nil {
		t.Skip("skipping: .flotio.yaml found in parent directory")
	}

	// With explicit flag → always works
	id, err := ResolveProjectID(99)
	if err != nil {
		t.Fatal(err)
	}
	if id != 99 {
		t.Errorf("expected 99, got %d", id)
	}

	// Create .flotio.yaml locally
	SaveProject(dir, &ProjectConfig{ProjectID: 42})
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	id, err = ResolveProjectID(0)
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Errorf("expected 42 from .flotio.yaml, got %d", id)
	}

	// Flag overrides .flotio.yaml
	id, err = ResolveProjectID(7)
	if err != nil {
		t.Fatal(err)
	}
	if id != 7 {
		t.Errorf("flag should override, expected 7, got %d", id)
	}
}
