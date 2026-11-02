package update

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"0.3.5", "0.3.4", true},
		{"0.10.0", "0.9.9", true},
		{"0.3.4", "0.3.4", false},
		{"0.3.3", "0.3.4", false},
		{"0.3.4", "dev", true},
		{"dev", "0.3.4", false},
	} {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%s, %s) = %v", c.a, c.b, got)
		}
	}
}

// The program is taken from a checked archive, and only the program.
func TestChecksumAndExtract(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("archive layout checked on unix")
	}
	dir := t.TempDir()
	archive := filepath.Join(dir, "oso-ai-relay-linux-x64.zip")
	f, _ := os.Create(archive)
	z := zip.NewWriter(f)
	w, _ := z.Create("oso")
	_, _ = w.Write([]byte("new program"))
	w, _ = z.Create("other")
	_, _ = w.Write([]byte("ignored"))
	_ = z.Close()
	_ = f.Close()

	data, _ := os.ReadFile(archive)
	sum := sha256.Sum256(data)
	sums := filepath.Join(dir, "SHA256SUMS.txt")
	_ = os.WriteFile(sums, []byte(hex.EncodeToString(sum[:])+"  oso-ai-relay-linux-x64.zip\n"), 0o600)
	if err := checkSum(archive, sums, "oso-ai-relay-linux-x64.zip"); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(sums, []byte("00  oso-ai-relay-linux-x64.zip\n"), 0o600)
	if checkSum(archive, sums, "oso-ai-relay-linux-x64.zip") == nil {
		t.Fatal("a wrong checksum was accepted")
	}

	program := filepath.Join(dir, "oso")
	_ = os.WriteFile(program, []byte("old program"), 0o755)
	if err := extract(archive, program+".new"); err != nil {
		t.Fatal(err)
	}
	if err := replace(program, program+".new"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(program); string(got) != "new program" {
		t.Fatalf("program is %q", got)
	}
}
