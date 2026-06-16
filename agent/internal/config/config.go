package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	ServerURL       string `json:"server_url"`
	Name            string `json:"name"`
	DeviceID        string `json:"device_id"`
	DeviceSecret    string `json:"device_secret"`
	ProtocolVersion string `json:"protocol_version"`
	AllowReboot     bool   `json:"allow_reboot"`
}

type Prefs struct {
	ServerURL string `json:"server_url"`
	Name      string `json:"name"`
}

func DefaultPath() string {
	if runtime.GOOS == "windows" {
		base := os.Getenv("PROGRAMDATA")
		if base == "" {
			base = "."
		}
		return filepath.Join(base, "OpsRelayAgent", "config.json")
	}
	return "/etc/opsrelay-agent/config.json"
}

func DefaultPrefsPath() string {
	return filepath.Join(filepath.Dir(DefaultPath()), "prefs.json")
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.ServerURL == "" || cfg.DeviceID == "" || cfg.DeviceSecret == "" {
		return Config{}, errors.New("config is missing server_url, device_id or device_secret")
	}
	return cfg, nil
}

func LoadPrefs(path string) (Prefs, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Prefs{}, err
	}
	var prefs Prefs
	if err := json.Unmarshal(raw, &prefs); err != nil {
		return Prefs{}, err
	}
	return prefs, nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

func SavePrefs(path string, prefs Prefs) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}
