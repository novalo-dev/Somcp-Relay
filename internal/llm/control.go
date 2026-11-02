package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Free, Stop and Start give the computer back when the merchant needs it:
// a model kept in memory slows everything else down. They work on Ollama
// and LM Studio, the two the relay knows how to install; any other AI is
// left to its own window.

// ErrUnknownAI means the address belongs to neither Ollama nor LM Studio.
var ErrUnknownAI = errors.New("only Ollama and LM Studio can be stopped and started from here")

// Kind tells which AI answers at an address, even when it is stopped: by
// asking it first, then by the port each one uses.
func Kind(ctx context.Context, address string) string {
	base, err := Normalize(address)
	if err != nil {
		return ""
	}
	if OllamaAnswers(ctx, base) {
		return KindOllama
	}
	port := ""
	if u, err := url.Parse(base); err == nil {
		port = u.Port()
	}
	switch {
	case port == "11434":
		return KindOllama
	case lmsPath() != "":
		return KindLMStudio
	}
	return ""
}

// Free unloads the models from memory without stopping the AI: the next
// question loads its model again, a little slower.
func Free(ctx context.Context, address string) error {
	base, err := Normalize(address)
	if err != nil {
		return err
	}
	switch Kind(ctx, address) {
	case KindOllama:
		return freeOllama(ctx, base)
	case KindLMStudio:
		return runHidden(ctx, lmsPath(), "unload", "--all")
	}
	return ErrUnknownAI
}

// Stop unloads the models and stops the AI's server.
func Stop(ctx context.Context, address string) error {
	base, err := Normalize(address)
	if err != nil {
		return err
	}
	switch Kind(ctx, address) {
	case KindOllama:
		_ = freeOllama(ctx, base)
		return stopOllama(ctx)
	case KindLMStudio:
		lms := lmsPath()
		_ = runHidden(ctx, lms, "unload", "--all")
		_ = runHidden(ctx, lms, "server", "stop")
		return runHidden(ctx, lms, "daemon", "down")
	}
	return ErrUnknownAI
}

// Start starts the AI's server again and waits until it answers.
func Start(ctx context.Context, address string) error {
	base, err := Normalize(address)
	if err != nil {
		return err
	}
	ai := &Client{base: base, http: &http.Client{Timeout: 5 * time.Second}}
	answers := func() bool { _, err := ai.Models(ctx); return err == nil }
	if answers() {
		return nil
	}
	switch Kind(ctx, address) {
	case KindOllama:
		if err := launchOllama(); err != nil {
			return err
		}
	case KindLMStudio:
		lms := lmsPath()
		port := "1234"
		if u, err := url.Parse(base); err == nil && u.Port() != "" {
			port = u.Port()
		}
		_ = runHidden(ctx, lms, "daemon", "up")
		if err := runHidden(ctx, lms, "server", "start", "--port", port); err != nil {
			return err
		}
	default:
		return ErrUnknownAI
	}
	if !waitFor(ctx, answers, 60*time.Second) {
		return errors.New("the AI was started but does not answer yet")
	}
	return nil
}

// freeOllama asks Ollama to drop every loaded model at once: keep_alive 0
// is its documented way to unload one.
func freeOllama(ctx context.Context, base string) error {
	root := OllamaRoot(base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, root+"/api/ps", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	var loaded struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	err = json.NewDecoder(resp.Body).Decode(&loaded)
	resp.Body.Close()
	if err != nil {
		return err
	}
	for _, m := range loaded.Models {
		body, _ := json.Marshal(map[string]any{"model": m.Name, "keep_alive": 0})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, root+"/api/generate", bytes.NewReader(body))
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
	}
	return nil
}

// stopOllama ends Ollama's app and server, as quitting it from its icon
// does.
func stopOllama(ctx context.Context) error {
	switch runtime.GOOS {
	case "windows":
		_ = runHidden(ctx, "taskkill", "/F", "/IM", "ollama app.exe")
		return runHidden(ctx, "taskkill", "/F", "/IM", "ollama.exe")
	case "darwin":
		_ = runHidden(ctx, "osascript", "-e", `quit app "Ollama"`)
		_ = runHidden(ctx, "pkill", "-x", "ollama")
		return nil
	default:
		return runHidden(ctx, "pkill", "-x", "ollama")
	}
}

// launchOllama starts Ollama the way its own shortcut does: the app on
// Windows and macOS, which starts the server, the server alone elsewhere.
func launchOllama() error {
	switch runtime.GOOS {
	case "windows":
		app := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Ollama", "ollama app.exe")
		if fileExists(app) {
			cmd := exec.Command(app)
			detach(cmd)
			return cmd.Start()
		}
	case "darwin":
		if err := exec.Command("open", "-a", "Ollama").Run(); err == nil {
			return nil
		}
		home, _ := os.UserHomeDir()
		if app := filepath.Join(home, "Applications", "Ollama.app"); fileExists(filepath.Join(app, "Contents", "Info.plist")) {
			return exec.Command("open", app).Run()
		}
	}
	return startOllama()
}

// Loaded says how much memory the AI's loaded models take, in bytes, when
// the AI tells: Ollama does, LM Studio's server does not, and then ok is
// false.
func Loaded(ctx context.Context, address string) (int64, bool) {
	base, err := Normalize(address)
	if err != nil || !OllamaAnswers(ctx, base) {
		return 0, false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, OllamaRoot(base)+"/api/ps", nil)
	if err != nil {
		return 0, false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	var loaded struct {
		Models []struct {
			Size int64 `json:"size"`
		} `json:"models"`
	}
	if json.NewDecoder(resp.Body).Decode(&loaded) != nil {
		return 0, false
	}
	var total int64
	for _, m := range loaded.Models {
		total += m.Size
	}
	return total, true
}

// Uninstall removes an AI the relay installed, with its downloaded models:
// the program from where Setup put it, and the folder where it keeps its
// models. It never touches an AI installed another way, which the caller
// checks against what Setup recorded.
func Uninstall(ctx context.Context, kind string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	switch kind {
	case KindOllama:
		_ = stopOllama(ctx)
		switch runtime.GOOS {
		case "windows":
			// Ollama's own uninstaller, silent, as its setup was.
			unins := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Ollama", "unins000.exe")
			if fileExists(unins) {
				if err := runHidden(ctx, unins, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART"); err != nil {
					return err
				}
			}
		case "darwin":
			if err := os.RemoveAll(filepath.Join(home, "Applications", "Ollama.app")); err != nil {
				return err
			}
		default:
			dir, err := setupDir()
			if err != nil {
				return err
			}
			if err := os.RemoveAll(filepath.Join(dir, "ollama")); err != nil {
				return err
			}
		}
		return os.RemoveAll(filepath.Join(home, ".ollama"))
	case KindLMStudio:
		if lms := lmsPath(); lms != "" {
			_ = runHidden(ctx, lms, "server", "stop")
			_ = runHidden(ctx, lms, "daemon", "down")
		}
		root := filepath.Join(home, ".lmstudio")
		pointer := filepath.Join(home, ".lmstudio-home-pointer")
		if moved, err := os.ReadFile(pointer); err == nil && len(bytes.TrimSpace(moved)) > 0 {
			root = string(bytes.TrimSpace(moved))
		}
		// A moved home is removed only when it is plainly LM Studio's
		// folder: a pointer to a broader folder must never wipe it.
		name := strings.ToLower(filepath.Base(filepath.Clean(root)))
		if filepath.Clean(root) == filepath.Clean(home) || !strings.Contains(strings.ReplaceAll(name, "-", ""), "lmstudio") {
			return errors.New("LM Studio's folder is not where the relay put it: remove it by hand")
		}
		_ = os.Remove(pointer)
		return os.RemoveAll(root)
	}
	return ErrUnknownAI
}
