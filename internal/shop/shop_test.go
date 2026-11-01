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
			_, _ = w.Write([]byte(`{"job":{"id":"j1","request":{"model":"m","messages":[]}}}`))
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
	job, err := client.Next(ctx)
	if err != nil || job == nil || job.ID != "j1" {
		t.Fatalf("next: %v %v", job, err)
	}
	if err := client.Result(ctx, "j1", 200, []byte(`{"choices":[]}`), ""); err != nil {
		t.Fatal(err)
	}
	if result["id"] != "j1" || result["response"] == nil {
		t.Errorf("result: %v", result)
	}
	revoked = true
	if _, err := client.Next(ctx); err != ErrRevoked {
		t.Errorf("revoked: %v", err)
	}
}
