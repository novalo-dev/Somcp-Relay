//go:build !windows

package pathenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// marker ends the line the relay adds, so it is recognised later.
const marker = "# added by O'So AI Relay"

// add writes the folder into the startup file of the user's shell, which
// every new terminal reads: zsh, bash or fish, and ~/.profile otherwise.
func add(dir string) (bool, error) {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == dir {
			return true, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	quoted := "'" + strings.ReplaceAll(dir, "'", `'\''`) + "'"
	file, line := filepath.Join(home, ".profile"), "export PATH=\"$PATH:\""+quoted+" "+marker
	switch filepath.Base(os.Getenv("SHELL")) {
	case "zsh":
		file = filepath.Join(home, ".zshrc")
	case "bash":
		file = filepath.Join(home, ".bashrc")
		if runtime.GOOS == "darwin" {
			// Terminal on macOS opens login shells, which read this one.
			file = filepath.Join(home, ".bash_profile")
		}
	case "fish":
		file = filepath.Join(home, ".config", "fish", "config.fish")
		line = "fish_add_path " + quoted + " " + marker
	}
	if current, err := os.ReadFile(file); err == nil && strings.Contains(string(current), line) {
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return false, err
	}
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.WriteString("\n" + line + "\n")
	return false, err
}

// remove takes out the lines addtopath wrote, recognised by their marker,
// from every startup file it may have written to.
func remove(string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	for _, name := range []string{".profile", ".zshrc", ".bashrc", ".bash_profile", filepath.Join(".config", "fish", "config.fish")} {
		file := filepath.Join(home, name)
		data, err := os.ReadFile(file)
		if err != nil || !strings.Contains(string(data), marker) {
			continue
		}
		var kept []string
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(line, marker) {
				kept = append(kept, line)
			}
		}
		if err := os.WriteFile(file, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// hideAndDetach is only needed on Windows.
func hideAndDetach(*exec.Cmd) {}
