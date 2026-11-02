package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Setup installs a local AI with nothing for the merchant to do: it
// downloads the software from its official site, installs it for the
// current account only - no administrator rights -, starts it, downloads
// the model chosen - by default the one suited to the computer's memory -,
// and returns the address where
// it answers. The commands are the ones each vendor documents for its own
// silent install; nothing else is run.

// Kinds of local AI the relay can install.
const (
	KindOllama   = "ollama"
	KindLMStudio = "lmstudio"
)

// Default addresses: the ports each one listens on unless told otherwise.
const (
	OllamaAddress   = "http://localhost:11434"
	LMStudioAddress = "http://localhost:1234"
)

// Progress is one step of a setup, as a text key and its arguments, so the
// interface shows it in its own language. Percent is -1 when unknown.
type Progress struct {
	Key     string
	Args    []any
	Percent int
}

// ErrNeedsZstd means Ollama's Linux archive cannot be unpacked here.
var ErrNeedsZstd = errors.New("zstd missing")

// Setup installs and starts the AI of the given kind, then downloads the
// model of the given size, or the recommended one when size is empty. It
// returns the address of the AI, and whether it installed the AI itself -
// one already there is not the relay's to uninstall later.
func Setup(ctx context.Context, kind, size string, report func(Progress)) (string, bool, error) {
	if size == "" {
		size = RecommendSize(MemoryGB())
	}
	if !ValidSize(size) {
		return "", false, fmt.Errorf("unknown model size %q", size)
	}
	switch kind {
	case KindOllama:
		return setupOllama(ctx, size, report)
	case KindLMStudio:
		return setupLMStudio(ctx, size, report)
	}
	return "", false, fmt.Errorf("unknown AI %q", kind)
}

func setupOllama(ctx context.Context, size string, report func(Progress)) (string, bool, error) {
	base, _ := Normalize(OllamaAddress)
	installed := false
	if !OllamaAnswers(ctx, base) {
		if err := installOllama(ctx, report); err != nil {
			return "", false, err
		}
		installed = true
		report(Progress{Key: "setup.start_app", Args: []any{"Ollama"}, Percent: -1})
		if !waitFor(ctx, func() bool { return OllamaAnswers(ctx, base) }, 20*time.Second) {
			// The app did not start the server on its own: start it.
			if err := startOllama(); err != nil {
				return "", installed, err
			}
			if !waitFor(ctx, func() bool { return OllamaAnswers(ctx, base) }, 60*time.Second) {
				return "", installed, errors.New("Ollama was installed but does not answer")
			}
		}
	}

	model := OllamaModel(size)
	report(Progress{Key: "setup.download_model", Args: []any{model, 0}, Percent: 0})
	err := Pull(ctx, base, model, func(p int) {
		report(Progress{Key: "setup.download_model", Args: []any{model, p}, Percent: p})
	})
	return OllamaAddress, installed, err
}

// installOllama runs Ollama's own installer, the way its install scripts
// do: the Windows setup in silent mode for this account, the macOS app in
// the user's Applications folder, the Linux build in the relay's folder.
func installOllama(ctx context.Context, report func(Progress)) error {
	dir, err := setupDir()
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		setup := filepath.Join(dir, "OllamaSetup.exe")
		if err := download(ctx, "https://ollama.com/download/OllamaSetup.exe", setup, "Ollama", report); err != nil {
			return err
		}
		report(Progress{Key: "setup.install_app", Args: []any{"Ollama"}, Percent: -1})
		defer os.Remove(setup)
		return runHidden(ctx, setup, "/VERYSILENT", "/NORESTART", "/SUPPRESSMSGBOXES")
	case "darwin":
		archive := filepath.Join(dir, "Ollama-darwin.zip")
		if err := download(ctx, "https://ollama.com/download/Ollama-darwin.zip", archive, "Ollama", report); err != nil {
			return err
		}
		defer os.Remove(archive)
		report(Progress{Key: "setup.install_app", Args: []any{"Ollama"}, Percent: -1})
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		apps := filepath.Join(home, "Applications")
		if err := os.MkdirAll(apps, 0o755); err != nil {
			return err
		}
		_ = os.RemoveAll(filepath.Join(apps, "Ollama.app"))
		// ditto keeps the app bundle's links and permissions, which a
		// plain unzip in Go would lose.
		if err := runHidden(ctx, "ditto", "-x", "-k", archive, apps); err != nil {
			return err
		}
		return exec.Command("open", filepath.Join(apps, "Ollama.app")).Start()
	default:
		arch := runtime.GOARCH
		target := filepath.Join(dir, "ollama")
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
		// Recent builds are compressed with zstd, which Go cannot read on
		// its own: the system's zstd unpacks them. Older ones are .tgz.
		if zstd, err := exec.LookPath("zstd"); err == nil {
			archive := filepath.Join(dir, "ollama.tar.zst")
			if err := download(ctx, "https://ollama.com/download/ollama-linux-"+arch+".tar.zst", archive, "Ollama", report); err == nil {
				defer os.Remove(archive)
				report(Progress{Key: "setup.install_app", Args: []any{"Ollama"}, Percent: -1})
				return runHidden(ctx, "sh", "-c", `"$0" -d -c "$1" | tar -xf - -C "$2"`, zstd, archive, target)
			}
		}
		archive := filepath.Join(dir, "ollama.tgz")
		if err := download(ctx, "https://ollama.com/download/ollama-linux-"+arch+".tgz", archive, "Ollama", report); err != nil {
			if _, missing := exec.LookPath("zstd"); missing != nil {
				return ErrNeedsZstd
			}
			return err
		}
		defer os.Remove(archive)
		report(Progress{Key: "setup.install_app", Args: []any{"Ollama"}, Percent: -1})
		return runHidden(ctx, "tar", "-xzf", archive, "-C", target)
	}
}

// startOllama starts Ollama's server where the setup put it, detached so
// it keeps running when the relay stops.
func startOllama() error {
	var bin string
	switch runtime.GOOS {
	case "windows":
		bin = filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Ollama", "ollama.exe")
	case "darwin":
		home, _ := os.UserHomeDir()
		bin = filepath.Join(home, "Applications", "Ollama.app", "Contents", "Resources", "ollama")
	default:
		dir, err := setupDir()
		if err != nil {
			return err
		}
		bin = filepath.Join(dir, "ollama", "bin", "ollama")
	}
	if path, err := exec.LookPath("ollama"); err == nil {
		if _, err := os.Stat(bin); err != nil {
			bin = path
		}
	}
	cmd := exec.Command(bin, "serve")
	detach(cmd)
	return cmd.Start()
}

func setupLMStudio(ctx context.Context, size string, report func(Progress)) (string, bool, error) {
	lms := lmsPath()
	installed := lms == ""
	if lms == "" {
		dir, err := setupDir()
		if err != nil {
			return "", installed, err
		}
		// LM Studio's documented install of its headless core, llmster,
		// for this account only.
		if runtime.GOOS == "windows" {
			script := filepath.Join(dir, "lmstudio-install.ps1")
			if err := download(ctx, "https://lmstudio.ai/install.ps1", script, "LM Studio", report); err != nil {
				return "", installed, err
			}
			defer os.Remove(script)
			report(Progress{Key: "setup.install_app", Args: []any{"LM Studio"}, Percent: -1})
			if err := runHidden(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script); err != nil {
				return "", installed, err
			}
		} else {
			script := filepath.Join(dir, "lmstudio-install.sh")
			if err := download(ctx, "https://lmstudio.ai/install.sh", script, "LM Studio", report); err != nil {
				return "", installed, err
			}
			defer os.Remove(script)
			report(Progress{Key: "setup.install_app", Args: []any{"LM Studio"}, Percent: -1})
			if err := runHidden(ctx, "bash", script); err != nil {
				return "", installed, err
			}
		}
		if lms = lmsPath(); lms == "" {
			return "", installed, errors.New("LM Studio was installed but its lms command is missing")
		}
	}

	base, _ := Normalize(LMStudioAddress)
	ai := &Client{base: base, http: &http.Client{Timeout: 10 * time.Second}}
	if _, err := ai.Models(ctx); err != nil {
		report(Progress{Key: "setup.start_app", Args: []any{"LM Studio"}, Percent: -1})
		_ = runHidden(ctx, lms, "daemon", "up")
		if err := runHidden(ctx, lms, "server", "start", "--port", "1234"); err != nil {
			return "", installed, err
		}
		if !waitFor(ctx, func() bool { _, err := ai.Models(ctx); return err == nil }, 60*time.Second) {
			return "", installed, errors.New("LM Studio was installed but does not answer")
		}
	}

	model, source := LMStudioModel(size)
	report(Progress{Key: "setup.download_model", Args: []any{model, 0}, Percent: 0})
	// --yes takes the download variant LM Studio preselects for this
	// computer. Its progress is read from what it prints.
	percent := regexp.MustCompile(`(\d{1,3})(?:\.\d+)?\s*%`)
	err := runLines(ctx, func(line string) {
		if m := percent.FindStringSubmatch(line); m != nil {
			p, _ := strconv.Atoi(m[1])
			if p <= 100 {
				report(Progress{Key: "setup.download_model", Args: []any{model, p}, Percent: p})
			}
		}
	}, lms, "get", source, "--yes")
	return LMStudioAddress, installed, err
}

// LMStudioModel is a size from LM Studio's own repository of tested
// builds: the name shown, and what lms gets.
func LMStudioModel(size string) (string, string) {
	name := "Qwen2.5-" + strings.ToUpper(size) + "-Instruct"
	return name, "https://huggingface.co/lmstudio-community/" + name + "-GGUF"
}

// lmsPath finds LM Studio's command, installed in ~/.lmstudio/bin unless
// the user moved LM Studio's home elsewhere.
func lmsPath() string {
	name := "lms"
	if runtime.GOOS == "windows" {
		name = "lms.exe"
	}
	home, err := os.UserHomeDir()
	if err == nil {
		root := filepath.Join(home, ".lmstudio")
		if moved, err := os.ReadFile(filepath.Join(home, ".lmstudio-home-pointer")); err == nil && strings.TrimSpace(string(moved)) != "" {
			root = strings.TrimSpace(string(moved))
		}
		if path := filepath.Join(root, "bin", name); fileExists(path) {
			return path
		}
	}
	if path, err := exec.LookPath("lms"); err == nil {
		return path
	}
	return ""
}

// setupDir is where downloads and the Linux Ollama go, beside the settings.
func setupDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "somcp-relay", "setup")
	return dir, os.MkdirAll(dir, 0o700)
}

// download saves a file and reports its progress.
// downloadClient fetches the installers. What it downloads is run right
// after, so every hop must stay on https: a redirect to plain http would let
// anyone on the path swap the installer.
var downloadClient = &http.Client{CheckRedirect: httpsOnly}

func httpsOnly(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if req.URL.Scheme != "https" {
		return fmt.Errorf("refused a redirect to %s: installers are only downloaded over https", req.URL.Redacted())
	}
	return nil
}

func download(ctx context.Context, address, path, label string, report func(Progress)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	if req.URL.Scheme != "https" {
		return fmt.Errorf("refused %s: installers are only downloaded over https", req.URL.Redacted())
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", address, resp.StatusCode)
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	counter := &counting{total: resp.ContentLength, report: func(p int) {
		report(Progress{Key: "setup.download_app", Args: []any{label, p}, Percent: p})
	}}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && runtime.GOOS != "windows" {
		err = os.Chmod(path, 0o700)
	}
	return err
}

type counting struct {
	total, done int64
	last        int
	report      func(int)
}

func (c *counting) Write(p []byte) (int, error) {
	c.done += int64(len(p))
	if c.total > 0 {
		if pct := int(c.done * 100 / c.total); pct != c.last {
			c.last = pct
			c.report(pct)
		}
	}
	return len(p), nil
}

// runHidden runs a command to its end, with no console window on Windows.
func runHidden(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	hide(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(out))
		if len(text) > 300 {
			text = text[len(text)-300:]
		}
		return fmt.Errorf("%s: %v %s", filepath.Base(name), err, text)
	}
	return nil
}

// runLines runs a command and hands each line - or carriage-return
// update - of its output to read.
func runLines(ctx context.Context, read func(string), name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	hide(cmd)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}
	buf := make([]byte, 4096)
	var line []byte
	for {
		n, rerr := pipe.Read(buf)
		for _, b := range buf[:n] {
			if b == '\n' || b == '\r' {
				read(string(line))
				line = line[:0]
				continue
			}
			line = append(line, b)
		}
		if rerr != nil {
			break
		}
	}
	read(string(line))
	return cmd.Wait()
}

func waitFor(ctx context.Context, ready func() bool, limit time.Duration) bool {
	until := time.Now().Add(limit)
	for time.Now().Before(until) {
		if ready() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Second):
		}
	}
	return ready()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
