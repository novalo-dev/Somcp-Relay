package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// DownloadPage is Ollama's official download page: the relay installs
// Ollama only from there, or through the system's package manager.
const DownloadPage = "https://ollama.com/download"

// OllamaRoot is Ollama's own API, beside the OpenAI-compatible one: the
// address the user typed without "/v1".
func OllamaRoot(base string) string {
	return strings.TrimSuffix(base, "/v1")
}

// OllamaAnswers tells whether Ollama runs at the typed address.
func OllamaAnswers(ctx context.Context, base string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, OllamaRoot(base)+"/api/version", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// Pull downloads a model into Ollama, reporting the progress in percent.
func Pull(ctx context.Context, base, model string, progress func(percent int)) error {
	if !ValidModelName(model) {
		return fmt.Errorf("invalid model name %q", model)
	}
	body, _ := json.Marshal(map[string]any{"model": model, "stream": true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, OllamaRoot(base)+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, MaxBody))
	last := -1
	for scanner.Scan() {
		var line struct {
			Status    string `json:"status"`
			Error     string `json:"error"`
			Total     int64  `json:"total"`
			Completed int64  `json:"completed"`
		}
		if json.Unmarshal(scanner.Bytes(), &line) != nil {
			continue
		}
		if line.Error != "" {
			return errors.New(line.Error)
		}
		if line.Total > 0 {
			percent := int(line.Completed * 100 / line.Total)
			if percent != last && progress != nil {
				progress(percent)
				last = percent
			}
		}
		if line.Status == "success" {
			if progress != nil && last != 100 {
				progress(100)
			}
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return errors.New("the download ended without success")
}

// ValidModelName keeps a model name to what Ollama names look like, since
// it travels to a command line on some systems.
func ValidModelName(name string) bool {
	if name == "" || len(name) > 120 {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(".:-_/", r)) {
			return false
		}
	}
	return true
}

// Size is one model the setup offers: Qwen2.5 calls tools well at every
// size, so the merchant only weighs answers against memory. Memory is
// roughly what the model takes once loaded, in GB.
type Size struct {
	Name   string `json:"name"`
	Memory int    `json:"memory"`
}

// Sizes are the models the setup offers, smallest first.
var Sizes = []Size{{"1.5b", 1}, {"3b", 2}, {"7b", 5}, {"14b", 9}}

// ValidSize says whether a size is one the setup offers.
func ValidSize(name string) bool {
	for _, s := range Sizes {
		if s.Name == name {
			return true
		}
	}
	return false
}

// RecommendSize picks the size that fits the memory with room left for the
// rest of the computer. Only a recommendation: the merchant may choose
// another.
func RecommendSize(memoryGB int) string {
	switch {
	case memoryGB >= 16:
		return "7b"
	case memoryGB >= 8:
		return "3b"
	default:
		return "1.5b"
	}
}

// Recommend is the recommended model, as Ollama names it.
func Recommend(memoryGB int) string {
	return OllamaModel(RecommendSize(memoryGB))
}

// OllamaModel is a size as Ollama names it.
func OllamaModel(size string) string {
	return "qwen2.5:" + size
}

// MemoryGB reads the computer's memory, 0 when it cannot be read.
func MemoryGB() int {
	var bytesTotal int64
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, _ := strconv.ParseInt(fields[1], 10, 64)
					bytesTotal = kb * 1024
				}
			}
		}
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 0
		}
		bytesTotal, _ = strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	case "windows":
		out, err := exec.Command("powershell", "-NoProfile", "-Command", "(Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory").Output()
		if err != nil {
			return 0
		}
		bytesTotal, _ = strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	}
	return int((bytesTotal + (1 << 29)) >> 30)
}

// OpenBrowser opens an http(s) address in the default browser.
func OpenBrowser(address string) error {
	u, err := url.Parse(address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("not a web address: %q", address)
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u.String()).Start()
	case "darwin":
		return exec.Command("open", u.String()).Start()
	default:
		return exec.Command("xdg-open", u.String()).Start()
	}
}
