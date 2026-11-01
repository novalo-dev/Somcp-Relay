//go:build !windows

package llm

import (
	"os/exec"
	"syscall"
)

// hide has nothing to hide outside Windows.
func hide(*exec.Cmd) {}

// detach puts a server in its own process group, so the Ctrl+C that stops
// the relay does not stop the AI with it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
