// Package i18n holds the relay's texts in English, French, Spanish and
// German. English is the default; every key must exist in all four, which
// the package test checks, so a text never falls back silently.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

//go:embed locales/*.json
var files embed.FS

// Languages are the codes the relay speaks, English first.
var Languages = []string{"en", "fr", "es", "de"}

var (
	mu       sync.RWMutex
	current  = "en"
	catalogs = map[string]map[string]string{}
)

func init() {
	for _, lang := range Languages {
		data, err := files.ReadFile("locales/" + lang + ".json")
		if err != nil {
			panic(err)
		}
		texts := map[string]string{}
		if err := json.Unmarshal(data, &texts); err != nil {
			panic(fmt.Sprintf("locales/%s.json: %v", lang, err))
		}
		catalogs[lang] = texts
	}
}

// Supported tells whether a code is one of the four languages.
func Supported(lang string) bool {
	_, ok := catalogs[lang]
	return ok
}

// Set picks the language: the one given if spoken, else the system's if
// spoken, else English.
func Set(lang string) {
	mu.Lock()
	defer mu.Unlock()
	switch {
	case Supported(lang):
		current = lang
	case Supported(System()):
		current = System()
	default:
		current = "en"
	}
}

// Current is the language in use.
func Current() string {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// System reads the language of the system from the usual variables; on
// Windows, where they are rarely set, the interface falls back to English
// unless the user picks another.
func System() string {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		value := strings.ToLower(os.Getenv(name))
		if len(value) >= 2 {
			return value[:2]
		}
	}
	return ""
}

// T returns the text of a key in the current language, with fmt verbs
// filled from args. A missing key shows as itself, so it is seen.
func T(key string, args ...any) string {
	mu.RLock()
	text, ok := catalogs[current][key]
	mu.RUnlock()
	if !ok {
		text, ok = catalogs["en"][key]
	}
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}

// All returns the whole catalogue of a language, for the interface.
func All(lang string) map[string]string {
	if !Supported(lang) {
		lang = "en"
	}
	return catalogs[lang]
}

// Catalog exposes one catalogue to the package test.
func Catalog(lang string) map[string]string {
	return catalogs[lang]
}
