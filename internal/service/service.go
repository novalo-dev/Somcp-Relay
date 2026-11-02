// Package service makes the relay start with the computer, for the user
// signed in and without administrator rights: a systemd user unit on
// Linux, a launchd agent on macOS, a logon task on Windows.
package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Name is the service's name on every system. It keeps the name of the
// first versions, so a relay installed with one of them is still found,
// stopped and removed by a newer one.
const Name = "somcp-relay"

const label = "fr.novalo.somcp-relay"

// ErrUnsupported is returned on a system none of the three cover.
var ErrUnsupported = errors.New("unsupported system")

func executable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(path)
}

// Install registers the relay to start with the computer, and starts it.
func Install() error {
	exe, err := executable()
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		path, err := unitPath()
		if err != nil {
			return err
		}
		unit := fmt.Sprintf("[Unit]\nDescription=O'So AI Relay\nAfter=network-online.target\n\n[Service]\nExecStart=%q run\nRestart=on-failure\nRestartSec=10\n\n[Install]\nWantedBy=default.target\n", exe)
		if err := writeFile(path, unit); err != nil {
			return err
		}
		if err := cmd("systemctl", "--user", "daemon-reload"); err != nil {
			return err
		}
		return cmd("systemctl", "--user", "enable", "--now", Name)
	case "darwin":
		path, err := plistPath()
		if err != nil {
			return err
		}
		plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>%s</string>
<key>ProgramArguments</key><array><string>%s</string><string>run</string></array>
<key>RunAtLoad</key><true/>
<key>KeepAlive</key><true/>
</dict></plist>
`, label, xmlEscape(exe))
		if err := writeFile(path, plist); err != nil {
			return err
		}
		return cmd("launchctl", "load", "-w", path)
	case "windows":
		if err := cmd("schtasks", "/Create", "/TN", Name, "/TR", `"`+exe+`" run`, "/SC", "ONLOGON", "/RL", "LIMITED", "/F"); err != nil {
			return err
		}
		return Start()
	}
	return ErrUnsupported
}

// Uninstall stops the relay and removes it from the startup.
func Uninstall() error {
	switch runtime.GOOS {
	case "linux":
		_ = cmd("systemctl", "--user", "disable", "--now", Name)
		path, err := unitPath()
		if err != nil {
			return err
		}
		_ = os.Remove(path)
		return cmd("systemctl", "--user", "daemon-reload")
	case "darwin":
		path, err := plistPath()
		if err != nil {
			return err
		}
		_ = cmd("launchctl", "unload", "-w", path)
		return os.Remove(path)
	case "windows":
		_ = Stop()
		return cmd("schtasks", "/Delete", "/TN", Name, "/F")
	}
	return ErrUnsupported
}

// Start runs the installed relay in the background.
func Start() error {
	switch runtime.GOOS {
	case "linux":
		return cmd("systemctl", "--user", "start", Name)
	case "darwin":
		return cmd("launchctl", "start", label)
	case "windows":
		return cmd("schtasks", "/Run", "/TN", Name)
	}
	return ErrUnsupported
}

// Stop stops the background relay.
func Stop() error {
	switch runtime.GOOS {
	case "linux":
		return cmd("systemctl", "--user", "stop", Name)
	case "darwin":
		return cmd("launchctl", "stop", label)
	case "windows":
		return cmd("schtasks", "/End", "/TN", Name)
	}
	return ErrUnsupported
}

// Installed tells whether the relay is set to start with the computer.
func Installed() bool {
	switch runtime.GOOS {
	case "linux":
		path, err := unitPath()
		return err == nil && exists(path)
	case "darwin":
		path, err := plistPath()
		return err == nil && exists(path)
	case "windows":
		return exec.Command("schtasks", "/Query", "/TN", Name).Run() == nil
	}
	return false
}

func unitPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "systemd", "user", Name+".service"), nil
}

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func cmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
