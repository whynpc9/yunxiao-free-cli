package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultDomain  = "openapi-rdc.aliyuncs.com"
	appDir         = "yunxiao-free-cli"
	EnvToken       = "YUNXIAO_TOKEN"
	LegacyEnvToken = "YX_TOKEN"
)

// Config is the local persisted user settings.
type Config struct {
	Token                   string `json:"token"`
	Domain                  string `json:"domain"`
	DefaultOrganizationID   string `json:"defaultOrganizationId,omitempty"`
	DefaultOrganizationName string `json:"defaultOrganizationName,omitempty"`
	UpdatedAt               string `json:"updatedAt,omitempty"`
}

func Default() Config {
	return Config{Domain: defaultDomain}
}

func ConfigFilePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}
	return filepath.Join(base, appDir, "config.json"), nil
}

func Load() (Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := Default()
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Domain == "" {
		cfg.Domain = defaultDomain
	}
	return cfg, nil
}

func Save(cfg Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	cfg.UpdatedAt = time.Now().Format(time.RFC3339)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func MaskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func TokenFromEnv() (string, string) {
	for _, name := range []string{EnvToken, LegacyEnvToken} {
		value := strings.TrimSpace(os.Getenv(name))
		if value != "" {
			return value, name
		}
	}
	return "", ""
}

func EffectiveToken(cfg Config) (string, string) {
	if token, envName := TokenFromEnv(); token != "" {
		return token, "env:" + envName
	}
	return strings.TrimSpace(cfg.Token), "config"
}
