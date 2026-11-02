//go:build windows

package pathenv

import (
	"encoding/base64"
	"errors"
	"os/exec"
	"strings"
	"unicode/utf16"
)

// add appends the folder to the user's Path, the one Windows keeps for this
// account, through PowerShell: Go's standard library has no registry access.
// The script is sent encoded, so a folder name needs no quoting on the
// command line.
func add(dir string) (bool, error) {
	quoted := "'" + strings.ReplaceAll(dir, "'", "''") + "'"
	script := "$d = " + quoted + "; " +
		"$p = [Environment]::GetEnvironmentVariable('Path', 'User'); " +
		"if ((\"$p\" -split ';') -contains $d) { exit 3 }; " +
		"if ($p) { $p = $p.TrimEnd(';') + ';' + $d } else { $p = $d }; " +
		"[Environment]::SetEnvironmentVariable('Path', $p, 'User')"
	units := utf16.Encode([]rune(script))
	raw := make([]byte, len(units)*2)
	for i, u := range units {
		raw[i*2], raw[i*2+1] = byte(u), byte(u>>8)
	}
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(raw))
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 3 {
		return true, nil
	}
	if err != nil {
		return false, errors.New(strings.TrimSpace(string(out)) + " " + err.Error())
	}
	return false, nil
}
