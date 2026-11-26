package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ProfileTokens holds one household member's OAuth1 user tokens.
type ProfileTokens struct {
	AccessToken       string `json:"accessToken,omitempty"`
	AccessTokenSecret string `json:"accessTokenSecret,omitempty"`
}

// Config is the persistent configuration: ~/.fatsecret-mcp/config.json with env overrides.
type Config struct {
	ClientID          string `json:"clientId,omitempty"`
	ClientSecret      string `json:"clientSecret,omitempty"`
	ConsumerSecret    string `json:"consumerSecret,omitempty"`
	AccessToken       string `json:"accessToken,omitempty"`
	AccessTokenSecret string `json:"accessTokenSecret,omitempty"`
	// Profiles holds extra users' tokens keyed by X-MCP-User header value.
	// The empty user ("") always falls back to AccessToken/AccessTokenSecret.
	Profiles map[string]ProfileTokens `json:"profiles,omitempty"`
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

// loadRawFile reads the config file into raw messages so Save preserves
// keys the typed Config doesn't know about (including nested objects).
func loadRawFile() map[string]json.RawMessage {
	m := map[string]json.RawMessage{}
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

// saveRaw writes the merged raw map back to the config file.
func saveRaw(m map[string]json.RawMessage) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	p, _ := Path()
	data, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(p, data, 0o600)
}

// saveConfig merges updates into the config file, dropping empty values.
func Save(updates map[string]string) error {
	m := loadRawFile()
	for k, v := range updates {
		if v == "" {
			delete(m, k)
		} else {
			b, _ := json.Marshal(v)
			m[k] = b
		}
	}
	return saveRaw(m)
}

// SaveProfileTokens persists one user's OAuth1 tokens: the default user ("")
// uses the top-level keys, named users live under the "profiles" object.
func SaveProfileTokens(user string, tok ProfileTokens) error {
	m := loadRawFile()
	if user == "" {
		setOrDelete(m, "accessToken", tok.AccessToken)
		setOrDelete(m, "accessTokenSecret", tok.AccessTokenSecret)
		return saveRaw(m)
	}
	var profiles map[string]ProfileTokens
	if raw, ok := m["profiles"]; ok {
		_ = json.Unmarshal(raw, &profiles)
	}
	if profiles == nil {
		profiles = map[string]ProfileTokens{}
	}
	if tok.AccessToken == "" {
		delete(profiles, user)
	} else {
		profiles[user] = tok
	}
	if len(profiles) == 0 {
		delete(m, "profiles")
	} else {
		b, _ := json.Marshal(profiles)
		m["profiles"] = b
	}
	return saveRaw(m)
}

func setOrDelete(m map[string]json.RawMessage, k, v string) {
	if v == "" {
		delete(m, k)
		return
	}
	b, _ := json.Marshal(v)
	m[k] = b
}

// ClearAllTokens drops every stored user token (default + profiles),
// e.g. when fresh API keys invalidate all authorizations.
func ClearAllTokens() error {
	m := loadRawFile()
	delete(m, "accessToken")
	delete(m, "accessTokenSecret")
	delete(m, "profiles")
	return saveRaw(m)
}
