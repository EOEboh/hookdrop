// Package config manages the CLI's local configuration file, which holds
// the API token — permissions are locked down and writes are atomic.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// DefaultAPIURL is the hosted hookdrop backend.
	DefaultAPIURL = "https://api.hookdrop.app"
	// DefaultFrontendURL is the hosted web app, used for browser login.
	DefaultFrontendURL = "https://hookdrop.app"
)

type Config struct {
	APIURL      string `json:"api_url,omitempty"`
	FrontendURL string `json:"frontend_url,omitempty"`
	Token       string `json:"token,omitempty"`
}

// Path returns the config file location:
// macOS ~/Library/Application Support/hookdrop, Linux ~/.config/hookdrop,
// Windows %AppData%\hookdrop.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate config dir: %w", err)
	}
	return filepath.Join(dir, "hookdrop", "config.json"), nil
}

// Load reads the config file, applying defaults and environment overrides
// (HOOKDROP_TOKEN, HOOKDROP_API_URL, HOOKDROP_FRONTEND_URL). A missing file
// is not an error; a corrupt one is, with a message pointing at the path.
// Loose file permissions produce a warning on stderr but do not fail.
func Load() (*Config, error) {
	cfg := &Config{}

	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// fresh install — fall through to defaults
	case err != nil:
		return nil, fmt.Errorf("read config %s: %w", path, err)
	default:
		if jsonErr := json.Unmarshal(data, cfg); jsonErr != nil {
			return nil, fmt.Errorf("config file %s is corrupt (%v) — run 'hookdrop login' to recreate it", path, jsonErr)
		}
		warnLoosePermissions(path)
	}

	if v := os.Getenv("HOOKDROP_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("HOOKDROP_API_URL"); v != "" {
		cfg.APIURL = v
	}
	if v := os.Getenv("HOOKDROP_FRONTEND_URL"); v != "" {
		cfg.FrontendURL = v
	}
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPIURL
	}
	if cfg.FrontendURL == "" {
		cfg.FrontendURL = DefaultFrontendURL
	}
	return cfg, nil
}

// Save writes the config atomically (temp file + rename) with 0600 perms.
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "config-*.json")
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	defer os.Remove(tmp.Name())

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return os.Rename(tmp.Name(), path)
}

func warnLoosePermissions(path string) {
	if runtime.GOOS == "windows" {
		return // unix permission bits are meaningless there
	}
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr,
			"warning: %s is readable by other users (mode %o) — consider: chmod 600 %s\n",
			path, info.Mode().Perm(), path)
	}
}
