// Package shop speaks to the somcp module of a PrestaShop shop. The relay
// always calls out: it asks the shop for work and posts the answers back,
// so the shop never has to reach into the merchant's network. PROTOCOL.md
// describes every call.
package shop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/novalo-dev/somcp-relay/internal/i18n"
	"github.com/novalo-dev/somcp-relay/internal/llm"
)

// UserAgent names the relay to the shop's host. Without it Go sends
// "Go-http-client", which many hosts' anti-bot rules block outright.
var UserAgent = "OSo-AI-Relay (+https://github.com/novalo-dev/Somcp-Relay)"

// ErrRevoked means the shop no longer knows the token: it was revoked in
// the back office, or the relay was paired again elsewhere.
var ErrRevoked = errors.New("the shop revoked this relay")

// Client calls one shop.
type Client struct {
	endpoint string
	token    string
	http     *http.Client
}

// Info is what the shop says about itself.
type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Job is one chat request the shop wants answered.
type Job struct {
	ID string `json:"id"`
	// Request is an OpenAI-compatible chat completion body, written by the
	// module. Its model must be one the relay reported.
	Request json.RawMessage `json:"request"`
}

// Endpoint builds the module's relay address from the shop's address.
// The plain index.php form works with friendly URLs on or off.
func Endpoint(shopURL string) (string, error) {
	shopURL = strings.TrimSpace(shopURL)
	if !strings.Contains(shopURL, "://") {
		shopURL = "https://" + shopURL
	}
	u, err := url.Parse(shopURL)
	if err != nil {
		return "", err
	}
	// Plain http is accepted only on this computer or a test shop on the
	// local network: over the internet the token would travel in clear.
	if u.Scheme != "https" && !(u.Scheme == "http" && localHost(u.Hostname())) {
		return "", errors.New("the shop address must start with https://")
	}
	if u.Host == "" {
		return "", errors.New("no host in the shop address")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/index.php"
	u.RawQuery = "fc=module&module=somcp&controller=relay"
	u.Fragment = ""
	return u.String(), nil
}

// localHost says whether plain http can carry the token to this host: this
// computer or the local network only. The host must be a real IP address in
// those ranges, or localhost, or an mDNS .local name - a prefix test would
// let a public name such as 10.example.com through.
func localHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".local") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

// errDowngrade stops a redirect from https to plain http: the token would
// follow the request in clear.
var errDowngrade = errors.New("redirect from https to http")

// keepHTTPS is the client's redirect policy. Go already drops the token
// when a redirect leaves for another host, but not when the same host
// sends it to http.
func keepHTTPS(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if via[0].URL.Scheme == "https" && req.URL.Scheme != "https" {
		return errDowngrade
	}
	return nil
}

// New returns a client for a shop, with the token when already paired.
func New(shopURL, token string) (*Client, error) {
	endpoint, err := Endpoint(shopURL)
	if err != nil {
		return nil, err
	}
	// Longer than the shop's own wait on next, so a quiet poll ends on the
	// shop's side and not as a client timeout.
	return &Client{endpoint: endpoint, token: token, http: &http.Client{Timeout: 60 * time.Second, CheckRedirect: keepHTTPS}}, nil
}

// Pair trades a pairing code for the relay's token.
func (c *Client) Pair(ctx context.Context, code, version, osName string) (string, Info, error) {
	var answer struct {
		Token string `json:"token"`
		Shop  Info   `json:"shop"`
	}
	err := c.call(ctx, http.MethodPost, "pair", map[string]any{
		"code":  strings.ToUpper(strings.TrimSpace(code)),
		"relay": map[string]string{"version": version, "os": osName},
	}, &answer, false)
	if err == nil && answer.Token == "" {
		err = errors.New("the shop sent no token")
	}
	return answer.Token, answer.Shop, err
}

// Hello tells the shop which AI answers and which models it offers, with
// the result of their tool test, so the back office offers only those.
func (c *Client) Hello(ctx context.Context, software string, models []llm.Model) (Info, error) {
	var answer struct {
		Shop Info `json:"shop"`
	}
	err := c.call(ctx, http.MethodPost, "hello", map[string]any{
		"ai": map[string]any{"software": software, "models": models},
	}, &answer, true)
	return answer.Shop, err
}

// MaxIdle caps the pause a shop may ask for, whatever it sends.
const MaxIdle = 2 * time.Minute

// Next waits for a job; nil when the shop had none before its wait ended.
// While nobody uses the shop's chat, the shop answers at once instead of
// holding the call, and says how long to wait before asking again: idle is
// that pause, zero otherwise. wait=1 tells the shop this relay understands
// it - an older one, told nothing, would ask again at once.
func (c *Client) Next(ctx context.Context) (*Job, time.Duration, error) {
	var answer struct {
		Job  *Job `json:"job"`
		Wait int  `json:"wait"`
	}
	if err := c.call(ctx, http.MethodGet, "next&wait=1", nil, &answer, true); err != nil {
		return nil, 0, err
	}
	idle := time.Duration(answer.Wait) * time.Second
	if idle < 0 {
		idle = 0
	}
	if idle > MaxIdle {
		idle = MaxIdle
	}
	return answer.Job, idle, nil
}

// Result posts the AI's answer to a job, or the reason there is none.
func (c *Client) Result(ctx context.Context, id string, status int, response []byte, failure string) error {
	body := map[string]any{"id": id, "status": status}
	if failure != "" {
		body["error"] = failure
	} else {
		body["response"] = json.RawMessage(response)
	}
	return c.call(ctx, http.MethodPost, "result", body, nil, true)
}

func (c *Client) call(ctx context.Context, method, action string, body any, out any, auth bool) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+"&action="+action, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if errors.Is(err, errDowngrade) {
		return errors.New(i18n.T("err.shop_downgrade"))
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, llm.MaxBody))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized && auth {
		return ErrRevoked
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		var problem struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &problem) == nil && problem.Error != "" {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, problem.Error)
		}
		return notTheModule(resp)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

// notTheModule explains an error the module did not write: it always
// answers in JSON, so a bare status came from the web server or something
// in front of it, and "HTTP 503" alone sends the merchant looking in the
// wrong place.
func notTheModule(resp *http.Response) error {
	status := resp.StatusCode
	switch status {
	case http.StatusNotFound:
		return errors.New(i18n.T("err.shop_no_module", status))
	case http.StatusForbidden, http.StatusNotAcceptable, http.StatusTooManyRequests, http.StatusServiceUnavailable:
		// The Server header names what answered (LiteSpeed, cloudflare...),
		// which tells the merchant where the rule to change lives.
		who := strings.TrimSpace(resp.Header.Get("Server"))
		if who == "" {
			return errors.New(i18n.T("err.shop_blocked", status))
		}
		return errors.New(i18n.T("err.shop_blocked_by", status, who))
	}
	if status >= 500 {
		return errors.New(i18n.T("err.shop_failed", status))
	}
	return fmt.Errorf("HTTP %d", status)
}
