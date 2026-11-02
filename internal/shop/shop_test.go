package shop

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/novalo-dev/somcp-relay/internal/llm"
)

func TestEndpoint(t *testing.T) {
	got, err := Endpoint("ma-boutique.fr/shop/")
	if err != nil || got != "https://ma-boutique.fr/shop/index.php?fc=module&module=somcp&controller=relay" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := Endpoint("http://ma-boutique.fr"); err == nil {
		t.Error("plain http over the internet accepted")
	}
	if _, err := Endpoint("http://localhost:8080"); err != nil {
		t.Errorf("local http refused: %v", err)
	}
}

// One full round: pairing, hello, a job, its result, then a revoked token.
func TestProtocol(t *testing.T) {
	revoked := false
	var result map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		if r.URL.Query().Get("controller") != "relay" {
			t.Errorf("wrong controller: %s", r.URL)
		}
		if action != "pair" && (r.Header.Get("Authorization") != "Bearer tok" || revoked) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		switch action {
		case "pair":
			if !strings.Contains(string(body), `"code":"K7Q2-9XD4"`) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"bad code"}`))
				return
			}
			_, _ = w.Write([]byte(`{"token":"tok","shop":{"name":"MyShop","version":"9.0.0"}}`))
		case "hello":
			_, _ = w.Write([]byte(`{"shop":{"name":"MyShop"}}`))
		case "next":
			// The relay says it understands the idle pause; the shop asks
			// for one far above the cap, which the relay must clamp.
			if r.URL.Query().Get("wait") != "1" {
				t.Error("next without wait=1")
			}
			_, _ = w.Write([]byte(`{"job":{"id":"j1","request":{"model":"m","messages":[]}},"wait":9999}`))
		case "result":
			_ = json.Unmarshal(body, &result)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	client, err := New(strings.Replace(server.URL, "127.0.0.1", "localhost", 1), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Pair(ctx, "wrong", "dev", "test"); err == nil || !strings.Contains(err.Error(), "bad code") {
		t.Errorf("wrong code: %v", err)
	}
	token, info, err := client.Pair(ctx, " k7q2-9xd4 ", "dev", "test")
	if err != nil || token != "tok" || info.Name != "MyShop" {
		t.Fatalf("pair: %q %v %v", token, info, err)
	}
	client.token = token
	if _, err := client.Hello(ctx, "Ollama", []llm.Model{{ID: "m"}}); err != nil {
		t.Fatal(err)
	}
	job, idle, err := client.Next(ctx)
	if err != nil || job == nil || job.ID != "j1" {
		t.Fatalf("next: %v %v", job, err)
	}
	if idle != MaxIdle {
		t.Errorf("idle pause not capped: %v", idle)
	}
	if err := client.Result(ctx, "j1", 200, []byte(`{"choices":[]}`), ""); err != nil {
		t.Fatal(err)
	}
	if result["id"] != "j1" || result["response"] == nil {
		t.Errorf("result: %v", result)
	}
	revoked = true
	if _, _, err := client.Next(ctx); err != ErrRevoked {
		t.Errorf("revoked: %v", err)
	}
}

// A host's anti-bot answers before the module: the relay must name itself,
// and say who refused it rather than a bare status.
func TestBlockedByHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "OSo-AI-Relay") {
			t.Errorf("User-Agent %q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Server", "LiteSpeed")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("<html>blocked</html>"))
	}))
	defer server.Close()
	client, err := New(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.Pair(context.Background(), "K7Q2-9XD4", "test", "linux")
	if err == nil || !strings.Contains(err.Error(), "503") || !strings.Contains(err.Error(), "LiteSpeed") {
		t.Fatalf("got %v", err)
	}
}

// Plain http only reaches this computer or the local network: a public
// name that merely starts like a private address does not count.
func TestLocalHost(t *testing.T) {
	for host, want := range map[string]bool{
		"localhost": true, "127.0.0.1": true, "::1": true, "10.0.0.5": true,
		"172.20.1.1": true, "192.168.1.10": true, "shop.local": true,
		"10.example.com": false, "192.168.evil.com": false, "8.8.8.8": false,
		"172.32.0.1": false, "example.com": false,
	} {
		if got := localHost(host); got != want {
			t.Errorf("localHost(%q) = %v, want %v", host, got, want)
		}
	}
}

// A shop that redirects from https to http must not receive the token in
// clear.
func TestNoDowngrade(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("the token reached plain http")
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer plain.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+r.URL.RequestURI(), http.StatusFound)
	}))
	defer secure.Close()
	client, err := New(secure.URL, "tok")
	if err != nil {
		t.Fatal(err)
	}
	client.http.Transport = secure.Client().Transport
	if err := client.Result(context.Background(), "j1", 200, []byte(`{}`), ""); err == nil {
		t.Fatal("the redirect to http was followed")
	}
}
