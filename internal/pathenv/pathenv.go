// Package pathenv makes the relay typeable as plain "oso" from any
// terminal: it copies the program into a fixed folder of the user's
// account and puts that folder on the user's PATH, so the downloaded
// archive and the extracted folder can be deleted afterwards. It changes
// only the current user's settings: no administrator rights.
package pathenv

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Result says what Add did.
type Result struct {
	// Program is where oso now lives.
	Program string
	// Folder is the folder put on the PATH.
	Folder string
	// Already is true when the folder was on the PATH before.
	Already bool
}

// Home is the fixed folder the program is copied to: the place each system
// keeps programs installed for one user.
func Home() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			dir, err := os.UserConfigDir()
			if err != nil {
				return "", err
			}
			base = dir
		}
		return filepath.Join(base, "Programs", "oso"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin"), nil
}

// Add copies the running program into Home and puts Home on the PATH.
// Terminals opened afterwards find it.
func Add() (Result, error) {
	var res Result
	exe, err := os.Executable()
	if err != nil {
		return res, err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	dir, err := Home()
	if err != nil {
		return res, err
	}
	name := "oso"
	if runtime.GOOS == "windows" {
		name = "oso.exe"
	}
	target := filepath.Join(dir, name)
	if !samePath(exe, target) {
		if err := install(exe, target); err != nil {
			return res, err
		}
	}
	res.Program, res.Folder = target, dir
	res.Already, err = add(dir)
	return res, err
}

// install copies the program over any older copy. A copy in use - the
// relay started with the computer - cannot be overwritten on Windows but
// can be renamed, so it is moved aside first and removed when possible.
func install(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	old := dst + ".old"
	_ = os.Remove(old)
	if _, err := os.Stat(dst); err == nil {
		if err := os.Rename(dst, old); err != nil {
			return err
		}
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		return err
	}
	_ = os.Remove(old)
	return nil
}

// Restart tells the copy to set up the startup again, so a relay already
// starting with the computer runs the new copy rather than the downloaded
// file about to be deleted.
func Restart(program string) error {
	return exec.Command(program, "install").Run()
}

func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// Remove takes oso off the PATH and deletes the copy addtopath made. A
// running program cannot be deleted on Windows: it is then deleted by a
// command left to run once the relay has exited.
func Remove() error {
	dir, err := Home()
	if err != nil {
		return err
	}
	pathErr := remove(dir)
	name := "oso"
	if runtime.GOOS == "windows" {
		name = "oso.exe"
	}
	program := filepath.Join(dir, name)
	if _, err := os.Stat(program); err != nil {
		return pathErr
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/C", "ping -n 4 127.0.0.1 >NUL & del /F /Q \""+program+"\" & rmdir \""+dir+"\"")
		hideAndDetach(cmd)
		if err := cmd.Start(); err != nil {
			return err
		}
		return pathErr
	}
	if err := os.Remove(program); err != nil {
		return err
	}
	return pathErr
}
