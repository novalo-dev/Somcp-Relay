//go:build !windows

package pathenv

import (
	"os"
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
