// Package config keeps the relay's settings in one file of the user's
// configuration folder. The shop token sits there too, so the file is
// written readable by its owner only.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config is everything the relay remembers between runs.
type Config struct {
	// ShopURL is the shop's address as the employee typed it.
	ShopURL string `json:"shop_url"`
	// ShopName is what the shop answered when it was linked, for display.
	ShopName string `json:"shop_name,omitempty"`
	// Token is the secret the shop handed out at pairing. Anyone holding it
	// can answer the shop's chat requests, hence the file mode.
	Token string `json:"token,omitempty"`
	// AIURL is the address of the local AI, always typed by the user: the
	// port of Ollama or LM Studio may well be taken and moved.
	AIURL string `json:"ai_url,omitempty"`
	// Lang is one of the interface languages, empty for the system's.
	Lang string `json:"lang,omitempty"`
	// Paused stops answering the shop without unlinking it.
	Paused bool `json:"paused,omitempty"`
}

// Dir is the relay's folder in the user's configuration folder.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	// The folder keeps the name of the first versions, so the shop linked
	// with one of them survives the update.
	return filepath.Join(base, "somcp-relay"), nil
}

// Path is the settings file.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the settings, or returns empty ones on a first run.
func Load() (Config, error) {
	var c Config
	path, err := Path()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(data, &c)
	return c, err
}

// Save writes the settings through a temporary file, so a crash mid-write
// never leaves a half file that loses the token.
func Save(c Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Linked tells whether a shop is paired.
func (c Config) Linked() bool {
	return c.ShopURL != "" && c.Token != ""
}
