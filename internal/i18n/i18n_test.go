package i18n

import (
	"regexp"
	"testing"
)

// Every key exists in the four languages, with the same placeholders in the
// same order: a text missing in one language, or a %d that became %s, would
// show broken to the people reading that language.
func TestCatalogsMatch(t *testing.T) {
	verbs := regexp.MustCompile(`%[sdv%]`)
	english := Catalog("en")
	for _, lang := range Languages {
		catalog := Catalog(lang)
		if len(catalog) != len(english) {
			t.Errorf("%s has %d texts, en has %d", lang, len(catalog), len(english))
		}
		for key, text := range english {
			other, ok := catalog[key]
			if !ok {
				t.Errorf("%s misses %q", lang, key)
				continue
			}
			if a, b := verbs.FindAllString(text, -1), verbs.FindAllString(other, -1); len(a) != len(b) {
				t.Errorf("%s %q has placeholders %v, en has %v", lang, key, b, a)
			} else {
				for i := range a {
					if a[i] != b[i] {
						t.Errorf("%s %q has placeholders %v, en has %v", lang, key, b, a)
						break
					}
				}
			}
		}
	}
}

func TestSetFallsBackToEnglish(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")
	t.Setenv("LANGUAGE", "")
	Set("xx")
	if Current() != "en" {
		t.Fatalf("got %s", Current())
	}
	Set("de")
	if Current() != "de" || T("ui.step.shop") != "Shop" {
		t.Fatalf("German not set")
	}
	Set("en")
}
