// Package pathenv puts the folder of the relay on the user's PATH, so the
// program can be typed as plain "oso" from any terminal. It changes only
// the current user's settings: no administrator rights.
package pathenv

import (
	"os"
	"path/filepath"
)

// Folder is the folder holding the running program.
func Folder() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return filepath.Dir(exe), nil
}

// Add puts the program's folder on the user's PATH. It returns the folder,
// and whether it was there already. Terminals opened afterwards see it.
func Add() (string, bool, error) {
	dir, err := Folder()
	if err != nil {
		return "", false, err
	}
	already, err := add(dir)
	return dir, already, err
}
