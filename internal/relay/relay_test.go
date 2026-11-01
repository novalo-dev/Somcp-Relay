package relay

import (
	"encoding/json"
	"testing"

	"github.com/novalo-dev/somcp-relay/internal/llm"
)

// The relay passes on only requests for a model it reported as able to
// call tools, and never streams.
func TestCheck(t *testing.T) {
	yes, no := true, false
	models := []llm.Model{{ID: "good", Tools: &yes}, {ID: "prose", Tools: &no}, {ID: "untested"}}

	out, err := Check(json.RawMessage(`{"model":"good","messages":[{"role":"user","content":"hi"}],"stream":true,"stream_options":{}}`), models)
	if err != nil {
		t.Fatal(err)
	}
	var sent map[string]any
	_ = json.Unmarshal(out, &sent)
	if sent["stream"] != false {
		t.Error("stream not forced off")
	}
	if _, ok := sent["stream_options"]; ok {
		t.Error("stream_options kept")
	}

	for _, bad := range []string{
		`{"model":"prose","messages":[]}`,
		`{"model":"untested","messages":[]}`,
		`{"model":"other","messages":[]}`,
		`{"model":"good"}`,
		`[1,2]`,
		`not json`,
	} {
		if _, err := Check(json.RawMessage(bad), models); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}
