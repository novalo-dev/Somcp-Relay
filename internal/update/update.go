// Package update replaces the oso program with the latest published
// release: the archive for this system is downloaded over https, checked
// against the release's SHA256SUMS.txt, and its program takes the place of
// the one running.
package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/novalo-dev/somcp-relay/internal/llm"
)

const (
	latestAPI      = "https://api.github.com/repos/novalo-dev/Somcp-Relay/releases/latest"
	latestDownload = "https://github.com/novalo-dev/Somcp-Relay/releases/latest/download/"
)

// The largest program accepted from an archive: the real one is about
// 10 MB, and a bigger entry is not ours.
const maxProgram = 200 << 20

// Latest asks GitHub for the version of the latest release, without the v.
func Latest(ctx context.Context) (string, error) {
	resp, err := llm.GetHTTPS(ctx, latestAPI)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var release struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return "", err
	}
	version := strings.TrimPrefix(release.Tag, "v")
	if _, ok := parse(version); !ok {
		return "", fmt.Errorf("unexpected release tag %q", release.Tag)
	}
	return version, nil
}

// Newer says whether version a comes after b. A build without a version,
// such as "dev", is older than any release.
func Newer(a, b string) bool {
	x, okA := parse(a)
	y, okB := parse(b)
	if !okA {
		return false
	}
	if !okB {
		return true
	}
	for i := range x {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

func parse(version string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Archive is the release file for this computer, named as the release
// workflow names it.
func Archive() (string, error) {
	names := map[string]string{
		"windows/amd64": "oso-ai-relay-windows-x64.zip",
		"windows/arm64": "oso-ai-relay-windows-arm64.zip",
		"darwin/arm64":  "oso-ai-relay-macos-apple-silicon.zip",
		"darwin/amd64":  "oso-ai-relay-macos-intel.zip",
		"linux/amd64":   "oso-ai-relay-linux-x64.tar.gz",
		"linux/arm64":   "oso-ai-relay-linux-arm64.tar.gz",
	}
	name, ok := names[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return "", fmt.Errorf("no release for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return name, nil
}

// Program is the path of the oso program running, links resolved.
func Program() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// Install downloads the latest release and puts its program in place of
// program. Nothing is replaced unless the archive matches its checksum.
func Install(ctx context.Context, program string) error {
	archive, err := Archive()
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "oso-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	quiet := func(llm.Progress) {}
	local := filepath.Join(dir, archive)
	if err := llm.Download(ctx, latestDownload+archive, local, "O'So AI Relay", quiet); err != nil {
		return err
	}
	sums := filepath.Join(dir, "SHA256SUMS.txt")
	if err := llm.Download(ctx, latestDownload+"SHA256SUMS.txt", sums, "O'So AI Relay", quiet); err != nil {
		return err
	}
	if err := checkSum(local, sums, archive); err != nil {
		return err
	}

	// The new program is written beside the old one, so the final rename
	// stays on one disk.
	fresh := program + ".new"
	if err := extract(local, fresh); err != nil {
		os.Remove(fresh)
		return err
	}
	return replace(program, fresh)
}

func checkSum(path, sums, name string) error {
	file, err := os.Open(sums)
	if err != nil {
		return err
	}
	defer file.Close()
	want := ""
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		fields := strings.Fields(lines.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			want = strings.ToLower(fields[0])
		}
	}
	if want == "" {
		return fmt.Errorf("%s is not in the release checksums", name)
	}
	data, err := os.Open(path)
	if err != nil {
		return err
	}
	defer data.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, data); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != want {
		return errors.New("the download does not match the release checksum")
	}
	return nil
}

// extract writes the archive's program, and only it, to target.
func extract(archive, target string) error {
	name := "oso"
	if runtime.GOOS == "windows" {
		name = "oso.exe"
	}
	if strings.HasSuffix(archive, ".zip") {
		z, err := zip.OpenReader(archive)
		if err != nil {
			return err
		}
		defer z.Close()
		for _, f := range z.File {
			if f.Name != name {
				continue
			}
			src, err := f.Open()
			if err != nil {
				return err
			}
			defer src.Close()
			return write(src, target)
		}
		return fmt.Errorf("%s is not in the archive", name)
	}
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	entries := tar.NewReader(gz)
	for {
		header, err := entries.Next()
		if err == io.EOF {
			return fmt.Errorf("%s is not in the archive", name)
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeReg && filepath.Clean(header.Name) == name {
			return write(entries, target)
		}
	}
}

func write(src io.Reader, target string) error {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(src, maxProgram+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxProgram {
		err = errors.New("the program in the archive is too big")
	}
	return err
}

// replace swaps the program for the new one. Windows cannot overwrite a
// running program but lets it be renamed: the old one is moved aside and
// removed at the next start (see Cleanup).
func replace(program, fresh string) error {
	if runtime.GOOS == "windows" {
		old := program + ".old"
		os.Remove(old)
		if err := os.Rename(program, old); err != nil {
			os.Remove(fresh)
			return err
		}
		if err := os.Rename(fresh, program); err != nil {
			// Put the old program back rather than leave none.
			_ = os.Rename(old, program)
			os.Remove(fresh)
			return err
		}
		return nil
	}
	if err := os.Rename(fresh, program); err != nil {
		os.Remove(fresh)
		return err
	}
	return nil
}

// Cleanup removes the program a Windows update moved aside, once it no
// longer runs.
func Cleanup() {
	if runtime.GOOS != "windows" {
		return
	}
	if program, err := Program(); err == nil {
		os.Remove(program + ".old")
	}
}
