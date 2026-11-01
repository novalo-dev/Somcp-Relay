//go:build windows

package llm

import (
	"os/exec"
	"syscall"
)

// createNoWindow keeps a console program from flashing a window.
const createNoWindow = 0x08000000

// hide runs a command without a console window: the merchant sees the
// progress in the relay, not a black box popping up.
func hide(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// detach starts a server that outlives the relay, without a window.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP}
}
