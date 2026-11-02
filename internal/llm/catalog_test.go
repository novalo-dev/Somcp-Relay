package llm

import "testing"

// Twenty models per AI, weakest first, each one usable as it is listed.
func TestCatalogs(t *testing.T) {
	for _, kind := range []string{KindOllama, KindLMStudio} {
		list := Catalog(kind)
		if len(list) != 20 {
			t.Errorf("%s: %d models, want 20", kind, len(list))
		}
		seen := map[string]bool{}
		for i, m := range list {
			if seen[m.ID] {
				t.Errorf("%s: %s listed twice", kind, m.ID)
			}
			seen[m.ID] = true
			if !ValidModel(kind, m.ID) {
				t.Errorf("%s: %s is not a valid name", kind, m.ID)
			}
			if i > 0 && m.Memory < list[i-1].Memory {
				t.Errorf("%s: %s is out of order", kind, m.ID)
			}
		}
	}
}

func TestRecommended(t *testing.T) {
	for _, c := range []struct {
		kind   string
		memory int
		want   string
	}{
		{KindOllama, 0, "qwen3:4b"},
		{KindOllama, 4, "qwen3:4b"},
		{KindOllama, 8, "qwen3:4b"},
		{KindOllama, 16, "qwen3:8b"},
		{KindOllama, 32, "qwen3.8:27b-q4_K_M"},
		{KindOllama, 128, "gpt-oss:120b"},
		{KindLMStudio, 8, "qwen/qwen3-8b"},
		{KindLMStudio, 32, "qwen/qwen3.8-27b"},
	} {
		if got := Recommended(c.kind, c.memory); got != c.want {
			t.Errorf("Recommended(%s, %d) = %s, want %s", c.kind, c.memory, got, c.want)
		}
	}
}

// What reaches `lms get` as an argument must never read as an option.
func TestValidModel(t *testing.T) {
	for _, bad := range []string{"", "-rf", "--yes", "a b", "x;y", "http://huggingface.co/a/b", "https://evil.example/a/b"} {
		if ValidModel(KindLMStudio, bad) || ValidModel(KindOllama, bad) {
			t.Errorf("%q accepted", bad)
		}
	}
	for _, good := range []string{"qwen3:32b", "hf.co/org/repo:Q4_K_M", "qwen/qwen3-32b"} {
		if !ValidModel(KindOllama, good) {
			t.Errorf("%q refused", good)
		}
	}
	if !ValidModel(KindLMStudio, "https://huggingface.co/lmstudio-community/Qwen3-32B-GGUF") {
		t.Error("Hugging Face address refused")
	}
}
