// Package relay is the loop that carries the shop's chat requests to the
// local AI and their answers back.
package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/novalo-dev/somcp-relay/internal/config"
	"github.com/novalo-dev/somcp-relay/internal/llm"
	"github.com/novalo-dev/somcp-relay/internal/shop"
)

// MaxTested is how many models the tool test runs on at start: each test
// is a real answer from the model, and a big library would delay the relay.
const MaxTested = 6

// State is what the running relay writes for the status command and the
// interface, which run in other processes.
type State struct {
	PID         int         `json:"pid"`
	Shop        string      `json:"shop"`
	AI          string      `json:"ai"`
	Software    string      `json:"software"`
	Models      []llm.Model `json:"models"`
	LastContact time.Time   `json:"last_contact"`
	Answered    int         `json:"answered"`
	LastAnswer  string      `json:"last_answer,omitempty"`
}

// StatePath is the state file beside the settings.
func StatePath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// ReadState reads what the running relay last wrote.
func ReadState() (State, error) {
	var s State
	path, err := StatePath()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

func writeState(s State) {
	path, err := StatePath()
	if err != nil {
		return
	}
	data, err := json.Marshal(s)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}

// Alive tells whether a relay wrote its state recently: a running one
// touches it at least every poll.
func (s State) Alive() bool {
	return !s.LastContact.IsZero() && time.Since(s.LastContact) < 2*time.Minute
}

// Logger receives one line per event, already translated by the caller.
type Logger interface {
	Started(shopName, ai string)
	Paused()
	Answered(d time.Duration)
	Retry(err error, wait time.Duration)
}

// Software names the AI from what answers at its address.
func Software(ctx context.Context, base string) string {
	if llm.OllamaAnswers(ctx, base) {
		return "Ollama"
	}
	return "OpenAI-compatible"
}

// Discover lists the AI's models and runs the tool test on the first
// MaxTested of them.
func Discover(ctx context.Context, ai *llm.Client) ([]llm.Model, error) {
	models, err := ai.Models(ctx)
	if err != nil {
		return nil, err
	}
	for i := range models {
		if i >= MaxTested {
			break
		}
		ok, err := ai.TestTools(ctx, models[i].ID)
		if err != nil {
			ok = false
		}
		models[i].Tools = &ok
	}
	return models, nil
}

// Run carries jobs until the context ends or the shop revokes the relay.
// The settings are read again on every turn, so a pause or an unlink made
// in the interface takes effect without a restart.
func Run(ctx context.Context, log Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !cfg.Linked() {
		return errors.New("not linked")
	}
	ai, err := llm.New(cfg.AIURL)
	if err != nil {
		return err
	}
	client, err := shop.New(cfg.ShopURL, cfg.Token)
	if err != nil {
		return err
	}

	state := State{PID: os.Getpid(), Shop: cfg.ShopName, AI: ai.Base()}
	state.Software = Software(ctx, ai.Base())
	state.Models, err = Discover(ctx, ai)
	if err != nil {
		return fmt.Errorf("local AI: %w", err)
	}
	if info, err := client.Hello(ctx, state.Software, state.Models); err == nil && info.Name != "" {
		state.Shop = info.Name
	} else if errors.Is(err, shop.ErrRevoked) {
		return err
	}
	state.LastContact = time.Now()
	writeState(state)
	log.Started(state.Shop, ai.Base())

	wait := 5 * time.Second
	wasPaused := false
	for ctx.Err() == nil {
		cfg, _ = config.Load()
		if !cfg.Linked() {
			return shop.ErrRevoked
		}
		if cfg.Paused {
			if !wasPaused {
				log.Paused()
			}
			wasPaused = true
			state.LastContact = time.Now()
			writeState(state)
			sleep(ctx, 5*time.Second)
			continue
		}
		wasPaused = false

		job, err := client.Next(ctx)
		if errors.Is(err, shop.ErrRevoked) {
			return err
		}
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Retry(err, wait)
			sleep(ctx, wait)
			if wait < time.Minute {
				wait *= 2
			}
			continue
		}
		wait = 5 * time.Second
		state.LastContact = time.Now()
		writeState(state)
		if job == nil {
			continue
		}

		started := time.Now()
		answerJob(ctx, client, ai, state.Models, job)
		state.Answered++
		state.LastAnswer = time.Since(started).Round(100 * time.Millisecond).String()
		state.LastContact = time.Now()
		writeState(state)
		log.Answered(time.Since(started))
	}
	return nil
}

func answerJob(ctx context.Context, client *shop.Client, ai *llm.Client, models []llm.Model, job *shop.Job) {
	request, err := Check(job.Request, models)
	if err != nil {
		_ = client.Result(ctx, job.ID, http.StatusBadRequest, nil, err.Error())
		return
	}
	status, answer, err := ai.Chat(ctx, request)
	if err != nil {
		_ = client.Result(ctx, job.ID, http.StatusBadGateway, nil, err.Error())
		return
	}
	_ = client.Result(ctx, job.ID, status, answer, "")
}

// Check accepts a chat request only for a model the relay reported as able
// to call tools, and without streaming, which the shop does not read. The
// address is never taken from the request: the relay only calls the AI the
// user typed.
func Check(raw json.RawMessage, models []llm.Model) ([]byte, error) {
	var request map[string]any
	if err := json.Unmarshal(raw, &request); err != nil || request == nil {
		return nil, errors.New("the request is not a JSON object")
	}
	model, _ := request["model"].(string)
	allowed := false
	for _, m := range models {
		if m.ID == model && m.Tools != nil && *m.Tools {
			allowed = true
		}
	}
	if !allowed {
		return nil, fmt.Errorf("model %q is not offered by this relay", model)
	}
	if _, ok := request["messages"].([]any); !ok {
		return nil, errors.New("the request has no messages")
	}
	request["stream"] = false
	delete(request, "stream_options")
	return json.Marshal(request)
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// OSName is sent to the shop at pairing, for its list of relays.
func OSName() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
