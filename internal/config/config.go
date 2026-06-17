// Package config handles configuration loading for the Flotio CLI.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// FlagConfigPath is set by the --config persistent flag.
var FlagConfigPath string

// FlagHost is set by the --host persistent flag.
var FlagHost string

// Config holds all CLI configuration values.
type Config struct {
	// Host is the Flotio API host (may include scheme, e.g. "http://localhost:8080").
	Host string `yaml:"host"`

	// Logging controls log level and format.
	Logging LoggingConfig `yaml:"logging"`
}

// LoggingConfig controls log output.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Host: "api.flotio.ovh",
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// ResolveHost returns the effective API host from (in precedence):
// 1. --host flag, 2. config file, 3. FLOTIO_HOST env, 4. default.
// The returned value is a full base URL (scheme://host).
func (c *Config) ResolveHost() string {
	raw := c.resolveRaw()
	return normalizeHost(raw)
}

// ResolveHostOnly returns just the hostname (without scheme) for the
// go-swagger generated client, which manages schemes separately.
func (c *Config) ResolveHostOnly() string {
	raw := c.resolveRaw()
	_, host := parseHost(raw)
	return host
}

func (c *Config) resolveRaw() string {
	if FlagHost != "" {
		return FlagHost
	}
	if c.Host != "" {
		return c.Host
	}
	if env := os.Getenv("FLOTIO_HOST"); env != "" {
		return env
	}
	return "api.flotio.ovh"
}

// normalizeHost ensures the host string has a scheme.
// If no scheme is present, "https://" is prepended.
func normalizeHost(raw string) string {
	if strings.Contains(raw, "://") {
		return raw
	}
	return "https://" + raw
}

// parseHost splits "scheme://host:port" into (scheme, host).
// If no scheme is present, defaults to "https".
func parseHost(raw string) (scheme, host string) {
	if idx := strings.Index(raw, "://"); idx >= 0 {
		return raw[:idx], raw[idx+3:]
	}
	return "https", raw
}

func configPath() (string, error) {
	if FlagConfigPath != "" {
		return FlagConfigPath, nil
	}
	if env := os.Getenv("FLOTIO_CONFIG"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".flotio", "config.yaml"), nil
}

// Load reads and parses the configuration file.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return cfg, nil
}
