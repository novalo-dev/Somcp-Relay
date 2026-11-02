package relay

import (
	"encoding/json"
	"time"
)

// Event is one line of the journal the interface shows: what happened to a
// request of the shop, or to the relay. Key names a text of the interface,
// so each person reads it in their own language.
type Event struct {
	At    time.Time `json:"at"`
	Kind  string    `json:"kind"` // ok, slow, refused, error, info
	Key   string    `json:"key"`
	Args  []any     `json:"args,omitempty"`
	Model string    `json:"model,omitempty"`
	Ms    int64     `json:"ms,omitempty"`
}

// MaxEvents is how many lines the journal keeps.
const MaxEvents = 40

// slowAnswer marks an answer as slow in the journal: usually the model
// being loaded, or one too big for the computer.
const slowAnswer = 15 * time.Second

// note adds an event at the top of the journal.
func (s *State) note(e Event) {
	e.At = time.Now()
	s.Events = append([]Event{e}, s.Events...)
	if len(s.Events) > MaxEvents {
		s.Events = s.Events[:MaxEvents]
	}
}

// count adds an answer to the figures of the day, which start again each
// day.
func (s *State) count(d time.Duration) {
	day := time.Now().Format("2006-01-02")
	if s.Day != day {
		s.Day, s.Today, s.TodayMs = day, 0, 0
	}
	s.Today++
	s.TodayMs += d.Milliseconds()
}

// toolCalls counts the tools the AI called in its answer, for the journal.
func toolCalls(answer []byte) int {
	var parsed struct {
		Choices []struct {
			Message struct {
				ToolCalls []json.RawMessage `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(answer, &parsed) != nil || len(parsed.Choices) == 0 {
		return 0
	}
	return len(parsed.Choices[0].Message.ToolCalls)
}

// requestModel reads the model a request asks for, for the journal.
func requestModel(raw json.RawMessage) string {
	var r struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(raw, &r)
	return r.Model
}
