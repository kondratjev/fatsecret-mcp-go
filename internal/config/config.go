package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the persistent configuration: ~/.fatsecret-mcp/config.json with env overrides.
type Config struct {
	ClientID          string `json:"clientId,omitempty"`
	ClientSecret      string `json:"clientSecret,omitempty"`
	ConsumerSecret    string `json:"consumerSecret,omitempty"`
	AccessToken       string `json:"accessToken,omitempty"`
	AccessTokenSecret string `json:"accessTokenSecret,omitempty"`
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fatsecret-mcp"), nil
}

func Path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func loadConfigFile() Config {
	var cfg Config
	p, err := Path()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

// resolveConfig applies env vars over the config file.
func Resolve() Config {
	file := loadConfigFile()
	cfg := file
	if v := os.Getenv("FATSECRET_CLIENT_ID"); v != "" {
		cfg.ClientID = v
	}
	if v := os.Getenv("FATSECRET_CLIENT_SECRET"); v != "" {
		cfg.ClientSecret = v
	}
	if v := os.Getenv("FATSECRET_CONSUMER_SECRET"); v != "" {
		cfg.ConsumerSecret = v
	}
	return cfg
}

// loadRawFile reads the config file straight into a map so Save preserves
// keys the typed Config doesn't know about.
func loadRawFile() map[string]string {
	m := map[string]string{}
	p, err := Path()
	if err != nil {
		return m
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	return m
}

// saveConfig merges updates into the config file, dropping empty values.
func Save(updates map[string]string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	m := loadRawFile()
	for k, v := range updates {
		if v == "" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	p, _ := Path()
	data, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(p, data, 0o600)
}
