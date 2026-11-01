package ui

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/novalo-dev/somcp-relay/internal/app"
	"github.com/novalo-dev/somcp-relay/internal/config"
	"github.com/novalo-dev/somcp-relay/internal/relay"
	"github.com/novalo-dev/somcp-relay/internal/shop"
)

// The interface carries the shop's requests itself while it is open.
// Without that, a merchant who linked the shop and tested the AI in the
// interface still had nothing running: the back office kept waiting for
// models that no relay had reported.
//
// It steps aside when another process - the relay started with the
// computer - already carries them, so the shop never has two relays
// asking for the same requests.

// otherAlive is how recent another relay's state must be for the
// interface to leave the work to it: that relay writes it at least every
// wait on the shop.
const otherAlive = 45 * time.Second

// retryAfter is how long the interface waits before starting a relay that
// stopped on an error again, so a stopped AI is not hammered with tests.
const retryAfter = 15 * time.Second

type runner struct {
	cancel  context.CancelFunc
	key     string
	err     string
	retryAt time.Time
	revoked bool
}

func (s *Server) supervise(ctx context.Context) {
	for {
		s.tick(ctx)
		select {
		case <-ctx.Done():
			s.mu.Lock()
			if s.run.cancel != nil {
				s.run.cancel()
			}
			s.mu.Unlock()
			return
		case <-time.After(3 * time.Second):
		}
	}
}

// tick starts, keeps or stops the relay to match the settings: a new shop,
// a new token or a new AI address restarts it, so the shop hears about the
// AI's models again.
func (s *Server) tick(ctx context.Context) {
	cfg, err := config.Load()
	if err != nil {
		return
	}
	key := ""
	if cfg.Linked() && cfg.AIURL != "" {
		key = cfg.ShopURL + "\n" + cfg.Token + "\n" + cfg.AIURL
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if key != "" && key == s.run.key && s.run.cancel != nil {
		return
	}
	if s.run.cancel != nil {
		s.run.cancel()
		s.run.cancel = nil
	}
	if key != s.run.key {
		// Other settings: the last error was about the old ones.
		s.run.err, s.run.retryAt = "", time.Time{}
	}
	s.run.key = ""
	if key == "" || time.Now().Before(s.run.retryAt) || otherRelay() {
		s.run.key = key
		return
	}

	runCtx, cancel := context.WithCancel(ctx)
	s.run.cancel, s.run.key, s.run.revoked = cancel, key, false
	go func() {
		err := relay.Run(runCtx, pageLog{s})
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.run.key != key || runCtx.Err() != nil {
			return
		}
		s.run.cancel = nil
		if errors.Is(err, shop.ErrRevoked) {
			// The shop no longer knows this relay: the merchant removed
			// the local assistant. The token is useless, so it is
			// forgotten and the interface asks for a new code.
			_ = app.Unlink()
			s.run.revoked = true
			return
		}
		if err != nil {
			s.run.err = err.Error()
		}
		s.run.retryAt = time.Now().Add(retryAfter)
	}()
}

// otherRelay tells whether another process carries the requests.
func otherRelay() bool {
	st, err := relay.ReadState()
	return err == nil && st.PID != os.Getpid() && !st.LastContact.IsZero() && time.Since(st.LastContact) < otherAlive
}

// carrier says who carries the requests, for the page: here, the
// background relay, or nobody yet.
func (s *Server) carrier() (string, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.run.cancel != nil:
		return "here", s.run.err, s.run.revoked
	case otherRelay():
		return "service", "", s.run.revoked
	default:
		return "", s.run.err, s.run.revoked
	}
}

// pageLog keeps what the page needs of the relay's console lines: the
// last problem, cleared once the relay works again.
type pageLog struct{ s *Server }

func (l pageLog) note(text string) {
	l.s.mu.Lock()
	l.s.run.err = text
	l.s.mu.Unlock()
}

func (l pageLog) Started(string, string)           { l.note("") }
func (l pageLog) Paused()                          {}
func (l pageLog) Answered(time.Duration)           { l.note("") }
func (l pageLog) Retry(err error, _ time.Duration) { l.note(err.Error()) }
