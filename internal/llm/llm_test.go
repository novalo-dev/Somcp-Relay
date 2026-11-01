package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"localhost:11434":           "http://localhost:11434/v1",
		"http://localhost:1234/":    "http://localhost:1234/v1",
		"http://127.0.0.1:8080/v1":  "http://127.0.0.1:8080/v1",
		"https://ai.example.com/x/": "https://ai.example.com/x/v1",
	}
	for in, want := range cases {
		got, err := Normalize(in)
		if err != nil || got != want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "ftp://host", "http://"} {
		if _, err := Normalize(bad); err == nil {
			t.Errorf("Normalize(%q) accepted", bad)
		}
	}
}

func TestModelsAndToolTest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"good"},{"id":"bad"}]}`))
		case "/v1/chat/completions":
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("no JSON content type")
			}
			var body [512]byte
			n, _ := r.Body.Read(body[:])
			if contains(string(body[:n]), `"model":"good"`) {
				_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"function":{"name":"get_time"}}]}}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"It is noon."}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ai, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	models, err := ai.Models(context.Background())
	if err != nil || len(models) != 2 {
		t.Fatalf("models: %v %v", models, err)
	}
	if ok, err := ai.TestTools(context.Background(), "good"); !ok || err != nil {
		t.Errorf("good model: %v %v", ok, err)
	}
	if ok, err := ai.TestTools(context.Background(), "bad"); ok || err != nil {
		t.Errorf("bad model: %v %v", ok, err)
	}
}

func TestValidModelName(t *testing.T) {
	for _, ok := range []string{"qwen2.5:7b", "library/llama3.1:8b"} {
		if !ValidModelName(ok) {
			t.Errorf("%q refused", ok)
		}
	}
	for _, bad := range []string{"", "a b", "x;rm -rf /", "$(id)"} {
		if ValidModelName(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
