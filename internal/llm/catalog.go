package llm

import (
	"regexp"
	"strings"
)

// Offer is one model the setup offers. Every one of them calls tools - a
// model that cannot is useless to the shop - and the merchant only weighs
// answers against memory: nothing is refused, even a model far bigger than
// the computer, which is merely flagged.
type Offer struct {
	// ID is what Ollama pulls or what lms gets.
	ID string `json:"id"`
	// Name is what the merchant reads.
	Name   string `json:"name"`
	Params string `json:"params"`
	// Download is the size of the download, in GB, as the catalog shows it.
	Download string `json:"download"`
	// Memory is the memory the model needs once loaded, in GB, rounded up
	// from the download with room for the context.
	Memory int `json:"memory"`
	// Pick marks the model recommended for a computer that has at least
	// Memory: the best tool user of its tier.
	Pick bool `json:"pick"`
}

// The two catalogs, weakest first. Each identifier was checked against
// ollama.com/library and lmstudio.ai/models, with the tools capability, on
// 2026-11-02. Explicit q4_K_M tags where Ollama's bare tag covers several
// sizes, so the size shown is the size downloaded.
var (
	OllamaCatalog = []Offer{
		{"llama3.2:1b", "Llama 3.2 1B", "1B", "1.3", 2, false},
		{"qwen3:1.7b", "Qwen3 1.7B", "1.7B", "1.4", 2, false},
		{"qwen3.5:2b-q4_K_M", "Qwen3.5 2B", "2B", "1.9", 3, false},
		{"granite4:3b", "Granite 4 3B", "3B", "2.1", 3, false},
		{"qwen3:4b", "Qwen3 4B", "4B", "2.5", 4, true},
		{"ministral-3:3b", "Ministral 3 3B", "3B", "3.0", 4, false},
		{"qwen3.5:4b-q4_K_M", "Qwen3.5 4B", "4B", "3.3", 5, false},
		{"qwen3:8b", "Qwen3 8B", "8B", "5.2", 7, true},
		{"ministral-3:8b", "Ministral 3 8B", "8B", "6.0", 8, false},
		{"qwen3.5:9b-q4_K_M", "Qwen3.5 9B", "9B", "6.6", 9, false},
		{"gemma4:12b-it-q4_K_M", "Gemma 4 12B", "12B", "8.0", 11, false},
		{"ministral-3:14b", "Ministral 3 14B", "14B", "9.1", 12, false},
		{"qwen3:14b", "Qwen3 14B", "14B", "9.3", 13, true},
		{"gpt-oss:20b", "gpt-oss 20B", "20B", "14", 19, false},
		{"qwen3.8:27b-q4_K_M", "Qwen3.8 27B", "27B", "18", 24, true},
		{"gemma4:31b-it-q4_K_M", "Gemma 4 31B", "31B", "20", 26, false},
		{"qwen3.6:35b-a3b-q4_K_M", "Qwen3.6 35B-A3B", "35B (3B active)", "24", 32, false},
		{"llama3.3:70b", "Llama 3.3 70B", "70B", "43", 56, false},
		{"gpt-oss:120b", "gpt-oss 120B", "120B", "65", 85, true},
		{"qwen3.5:122b-a10b-q4_K_M", "Qwen3.5 122B-A10B", "122B (10B active)", "81", 106, false},
	}
	LMStudioCatalog = []Offer{
		{"liquid/lfm3-vl-2b", "LFM3 VL 2B", "2B", "1.1", 2, false},
		{"qwen/qwen3.5-2b", "Qwen3.5 2B", "2B", "2.0", 3, false},
		{"ibm/granite-4.1-3b", "Granite 4.1 3B", "3B", "2.1", 3, false},
		{"mistralai/ministral-3-3b", "Ministral 3 3B", "3B", "2.0", 3, false},
		{"qwen/qwen3-4b", "Qwen3 4B", "4B", "2.2", 3, true},
		{"qwen/qwen3.5-4b", "Qwen3.5 4B", "4B", "3.75", 5, false},
		{"qwen/qwen3-8b", "Qwen3 8B", "8B", "4.6", 6, true},
		{"ibm/granite-4.1-8b", "Granite 4.1 8B", "8B", "5.3", 7, false},
		{"google/gemma-4-e4b", "Gemma 4 E4B", "8B (4.5B effective)", "5.9", 8, false},
		{"mistralai/ministral-3-8b", "Ministral 3 8B", "8B", "6.5", 9, false},
		{"qwen/qwen3.5-9b", "Qwen3.5 9B", "9B", "7.0", 10, false},
		{"google/gemma-4-12b", "Gemma 4 12B", "12B", "7.4", 10, false},
		{"qwen/qwen3-14b", "Qwen3 14B", "14B", "8.4", 11, false},
		{"mistralai/ministral-3-14b", "Ministral 3 14B", "14B", "9.5", 13, false},
		{"openai/gpt-oss-20b", "gpt-oss 20B", "20B", "12", 16, true},
		{"qwen/qwen3.8-27b", "Qwen3.8 27B", "27B", "16.1", 21, true},
		{"google/gemma-4-31b", "Gemma 4 31B", "31B", "19", 25, false},
		{"qwen/qwen3.6-35b-a3b", "Qwen3.6 35B-A3B", "35B (3B active)", "20.4", 27, false},
		{"openai/gpt-oss-120b", "gpt-oss 120B", "120B", "65", 85, true},
		{"nvidia/nemotron-3-super", "Nemotron 3 Super", "120B (12B active)", "83", 108, false},
	}
)

// Catalog is the list offered for one kind of AI.
func Catalog(kind string) []Offer {
	switch kind {
	case KindOllama:
		return OllamaCatalog
	case KindLMStudio:
		return LMStudioCatalog
	}
	return nil
}

// Recommended is the model suggested for this computer: the strongest pick
// that leaves a quarter of the memory to the rest of the computer, or the
// lightest pick when the memory is unknown or too small for any. The picks
// are listed by growing memory.
func Recommended(kind string, memoryGB int) string {
	best := ""
	for _, m := range Catalog(kind) {
		if !m.Pick {
			continue
		}
		if best == "" || (memoryGB > 0 && m.Memory <= memoryGB*3/4) {
			best = m.ID
		}
	}
	return best
}

// InCatalog says whether a model is one the setup lists.
func InCatalog(kind, id string) bool {
	for _, m := range Catalog(kind) {
		if m.ID == id {
			return true
		}
	}
	return false
}

// A model typed on the command line goes to `ollama pull` through the API,
// or to `lms get` as a program argument: only the characters of a model
// name, never a leading dash that lms would read as an option, or a Hugging
// Face address over https.
var (
	modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
	hfAddress = regexp.MustCompile(`^https://huggingface\.co/[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)
)

// ValidModel says whether a model can be passed to the setup: any model of
// the catalog, or another one named properly.
func ValidModel(kind, id string) bool {
	if InCatalog(kind, id) {
		return true
	}
	if kind == KindLMStudio && hfAddress.MatchString(id) {
		return true
	}
	// An address other than Hugging Face's would point lms at any server.
	return !strings.Contains(id, "://") && modelName.MatchString(id)
}
