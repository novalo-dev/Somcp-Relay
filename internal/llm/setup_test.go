package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// An installer is run right after it is downloaded: neither a plain http
// address nor a redirect to one may be followed.
func TestDownloadHTTPSOnly(t *testing.T) {
	reached := false
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		_, _ = w.Write([]byte("swapped"))
	}))
	defer plain.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			_, _ = w.Write([]byte("installer"))
			return
		}
		http.Redirect(w, r, plain.URL+"/setup.exe", http.StatusFound)
	}))
	defer secure.Close()

	saved := downloadClient.Transport
	downloadClient.Transport = secure.Client().Transport
	defer func() { downloadClient.Transport = saved }()

	dir := t.TempDir()
	report := func(Progress) {}
	if err := download(context.Background(), secure.URL+"/ok", filepath.Join(dir, "a"), "AI", report); err != nil {
		t.Fatalf("https download refused: %v", err)
	}
	if err := download(context.Background(), secure.URL+"/moved", filepath.Join(dir, "b"), "AI", report); err == nil || reached {
		t.Fatalf("redirect to http followed (err %v, reached %v)", err, reached)
	}
	if err := download(context.Background(), plain.URL+"/setup.exe", filepath.Join(dir, "c"), "AI", report); err == nil || reached {
		t.Fatalf("plain http address downloaded (err %v, reached %v)", err, reached)
	}
}
