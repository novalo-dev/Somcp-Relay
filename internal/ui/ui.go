// Package ui serves the relay's interface on this computer only. Every
// call needs a secret token the relay puts in the address it opens, and
// the Host header must name the relay itself: another web page in the
// browser can neither drive it nor read it, even by DNS rebinding.
package ui

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/novalo-dev/somcp-relay/internal/app"
	"github.com/novalo-dev/somcp-relay/internal/config"
	"github.com/novalo-dev/somcp-relay/internal/i18n"
	"github.com/novalo-dev/somcp-relay/internal/llm"
	"github.com/novalo-dev/somcp-relay/internal/pathenv"
	"github.com/novalo-dev/somcp-relay/internal/relay"
	"github.com/novalo-dev/somcp-relay/internal/service"
)

//go:embed index.html
var page []byte

// Server is the interface's web server.
type Server struct {
	token string
	host  string
	mu    sync.Mutex
	setup map[string]any
	run   runner
}

// Start listens on a free port of 127.0.0.1 and returns the address to
// open, token included.
func Start(ctx context.Context) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	secret := make([]byte, 24)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	s := &Server{token: hex.EncodeToString(secret), host: listener.Addr().String(), setup: map[string]any{}}
	server := &http.Server{Handler: s.routes(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	go func() { _ = server.Serve(listener) }()
	go s.supervise(ctx)
	return "http://" + s.host + "/#" + s.token, nil
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	api := map[string]func(*http.Request) (any, error){
		"/api/state":    s.state,
		"/api/lang":     s.lang,
		"/api/pair":     s.pair,
		"/api/ai":       s.ai,
		"/api/test":     s.test,
		"/api/pause":    s.pause,
		"/api/unlink":   func(*http.Request) (any, error) { return nil, app.Unlink() },
		"/api/startup":  s.startup,
		"/api/setup":    s.startSetup,
		"/api/power":    s.power,
		"/api/keep":     s.keep,
		"/api/removeai": s.removeAI,
		"/api/remove":   s.remove,
	}
	for path, handler := range api {
		handler := handler
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if !s.allowed(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			out, err := handler(r)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(out)
		})
	}
	return mux
}

// allowed checks the Host and the token. The page is served to anyone on
// this computer; the data and the actions are not.
func (s *Server) allowed(r *http.Request) bool {
	if r.Host != s.host {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		return false
	}
	given := r.Header.Get("X-Relay-Token")
	return subtle.ConstantTimeCompare([]byte(given), []byte(s.token)) == 1
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.Host != s.host || r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'")
	w.Header().Set("X-Frame-Options", "DENY")
	_, _ = w.Write(page)
}

func decode(r *http.Request, into any) error {
	if r.Method != http.MethodPost {
		return errors.New("POST expected")
	}
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16)).Decode(into)
}

func (s *Server) state(r *http.Request) (any, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	st, _ := relay.ReadState()
	memory := llm.MemoryGB()
	s.mu.Lock()
	setup := s.setup
	s.mu.Unlock()
	carrier, problem, revoked := s.carrier()
	var used int64
	usedKnown := false
	if cfg.AIURL != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		used, usedKnown = llm.Loaded(ctx, cfg.AIURL)
		cancel()
	}
	home, _ := pathenv.Home()
	return map[string]any{
		"carrier":   carrier,
		"problem":   problem,
		"revoked":   revoked,
		"version":   app.Version,
		"lang":      i18n.Current(),
		"languages": i18n.Languages,
		"texts":     i18n.All(i18n.Current()),
		"shop":      map[string]any{"url": cfg.ShopURL, "name": cfg.ShopName, "linked": cfg.Linked()},
		"ai":        cfg.AIURL,
		"paused":    cfg.Paused,
		"running":   st.Alive(),
		"relay":     st,
		"startup":   service.Installed(),
		"memory":    memory,
		"recommend": llm.Recommend(memory),
		"installed": cfg.Installed,
		"keep":      cfg.KeepLoaded,
		"used":      used,
		"usedKnown": usedKnown,
		"home":      home,
		"lmstudio":  lmModel(memory),
		"setup":     setup,
	}, nil
}

func (s *Server) lang(r *http.Request) (any, error) {
	var in struct {
		Lang string `json:"lang"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	if !i18n.Supported(in.Lang) {
		return nil, errors.New("unsupported language")
	}
	i18n.Set(in.Lang)
	return nil, app.SetLang(in.Lang)
}

func (s *Server) pair(r *http.Request) (any, error) {
	var in struct {
		Shop string `json:"shop"`
		Code string `json:"code"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	return app.Pair(r.Context(), in.Shop, in.Code)
}

func (s *Server) ai(r *http.Request) (any, error) {
	var in struct {
		Address string `json:"address"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	software, models, err := app.SetAI(r.Context(), in.Address)
	if err != nil {
		return nil, err
	}
	return map[string]any{"software": software, "models": models}, nil
}

func (s *Server) test(r *http.Request) (any, error) {
	var in struct {
		Model string `json:"model"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	cfg, _ := config.Load()
	ai, err := llm.New(cfg.AIURL)
	if err != nil {
		return nil, err
	}
	ok, err := ai.TestTools(r.Context(), in.Model)
	return map[string]bool{"tools": ok}, err
}

func (s *Server) pause(r *http.Request) (any, error) {
	var in struct {
		Paused bool `json:"paused"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	return nil, app.SetPaused(in.Paused)
}

func (s *Server) startup(r *http.Request) (any, error) {
	var in struct {
		On bool `json:"on"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	if in.On {
		return nil, service.Install()
	}
	return nil, service.Uninstall()
}

// startSetup installs a local AI in the background - Ollama or LM Studio,
// then a model - and tests it; the page follows the steps through the
// state. One setup at a time.
func (s *Server) startSetup(r *http.Request) (any, error) {
	var in struct {
		Kind string `json:"kind"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	if in.Kind != llm.KindOllama && in.Kind != llm.KindLMStudio {
		return nil, errors.New("unknown AI")
	}
	s.mu.Lock()
	if running, _ := s.setup["running"].(bool); running {
		s.mu.Unlock()
		return nil, errors.New("a setup is already running")
	}
	s.setup = map[string]any{"kind": in.Kind, "running": true, "key": "ui.working", "args": []any{}, "percent": -1}
	s.mu.Unlock()

	go func() {
		ctx := context.Background()
		report := func(p llm.Progress) {
			s.mu.Lock()
			s.setup = map[string]any{"kind": in.Kind, "running": true, "key": p.Key, "args": p.Args, "percent": p.Percent}
			s.mu.Unlock()
		}
		address, installed, err := llm.Setup(ctx, in.Kind, llm.MemoryGB(), report)
		if installed {
			_ = app.RememberInstalled(in.Kind)
		}
		var models []llm.Model
		software := ""
		if err == nil {
			report(llm.Progress{Key: "setup.test", Args: []any{}, Percent: -1})
			// Saved like a typed address: the relay starts on it.
			software, models, err = app.SetAI(ctx, address)
		}
		done := map[string]any{"kind": in.Kind, "running": false, "done": err == nil, "address": address, "models": models, "software": software}
		switch {
		case errors.Is(err, llm.ErrNeedsZstd):
			done["key"], done["args"] = "setup.zstd", []any{}
		case err != nil:
			done["key"], done["args"] = "err.generic", []any{err.Error()}
		default:
			done["key"], done["args"] = "setup.done", []any{address}
		}
		s.mu.Lock()
		s.setup = done
		s.mu.Unlock()
	}()
	return nil, nil
}

func lmModel(memory int) string {
	name, _ := llm.RecommendLMStudio(memory)
	return name
}

// power frees, stops or starts the local AI, and says how it went in the
// page's language.
func (s *Server) power(r *http.Request) (any, error) {
	var in struct {
		Action string `json:"action"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	key, err := app.Power(r.Context(), in.Action)
	if errors.Is(err, llm.ErrUnknownAI) {
		return nil, errors.New(i18n.T("power.unknown"))
	}
	if err != nil {
		return nil, err
	}
	return map[string]string{"message": i18n.T(key)}, nil
}

// keep keeps the model loaded between requests, or lets the relay free it.
func (s *Server) keep(r *http.Request) (any, error) {
	var in struct {
		Keep bool `json:"keep"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	return nil, app.SetKeepLoaded(in.Keep)
}

// removeAI uninstalls an AI the relay installed.
func (s *Server) removeAI(r *http.Request) (any, error) {
	var in struct {
		Kind string `json:"kind"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	err := app.RemoveAI(r.Context(), in.Kind)
	if errors.Is(err, app.ErrNotOurs) {
		return nil, errors.New(i18n.T("removeai.not_ours"))
	}
	return nil, err
}

// remove uninstalls the relay itself. The answer goes out first; then the
// relay removes itself and exits, since nothing is left to serve the page.
func (s *Server) remove(r *http.Request) (any, error) {
	var in struct {
		AI bool `json:"ai"`
	}
	if err := decode(r, &in); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.run.cancel != nil {
		s.run.cancel()
	}
	s.mu.Unlock()
	go func() {
		time.Sleep(500 * time.Millisecond)
		_ = app.RemoveAll(context.Background(), in.AI)
		os.Exit(0)
	}()
	return nil, nil
}
