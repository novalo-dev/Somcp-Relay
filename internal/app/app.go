// Package app holds the actions the command line and the interface share,
// so both pair a shop and set the AI through the same code.
package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/novalo-dev/somcp-relay/internal/config"
	"github.com/novalo-dev/somcp-relay/internal/llm"
	"github.com/novalo-dev/somcp-relay/internal/pathenv"
	"github.com/novalo-dev/somcp-relay/internal/relay"
	"github.com/novalo-dev/somcp-relay/internal/service"
	"github.com/novalo-dev/somcp-relay/internal/shop"
)

// Version is set at build time with -ldflags "-X ...app.Version=x.y.z".
var Version = "dev"

// Pair links a shop with the code its back office shows, and keeps the
// token. Linking again replaces the shop linked before.
func Pair(ctx context.Context, shopURL, code string) (shop.Info, error) {
	client, err := shop.New(shopURL, "")
	if err != nil {
		return shop.Info{}, err
	}
	token, info, err := client.Pair(ctx, code, Version, relay.OSName())
	if err != nil {
		return info, err
	}
	cfg, err := config.Load()
	if err != nil {
		return info, err
	}
	cfg.ShopURL = shopURL
	cfg.ShopName = info.Name
	cfg.Token = token
	cfg.Paused = false
	return info, config.Save(cfg)
}

// Unlink forgets the shop and its token; a running relay stops on its
// next turn.
func Unlink() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.ShopURL, cfg.ShopName, cfg.Token = "", "", ""
	return config.Save(cfg)
}

// SetAI tests the typed address - it answers, it lists models, which of
// them call tools - and keeps it only when it answers. A wrong address is
// never saved, so the relay never runs half set up.
func SetAI(ctx context.Context, address string) (string, []llm.Model, error) {
	ai, err := llm.New(address)
	if err != nil {
		return "", nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	models, err := relay.Discover(ctx, ai)
	if err != nil {
		return "", nil, err
	}
	if len(models) == 0 {
		return "", nil, errors.New("no model")
	}
	cfg, err := config.Load()
	if err != nil {
		return "", nil, err
	}
	cfg.AIURL = address
	if err := config.Save(cfg); err != nil {
		return "", nil, err
	}
	return relay.Software(ctx, ai.Base()), models, nil
}

// Power frees, stops or starts the local AI: "free" unloads its models,
// "stop" also pauses the relay and closes the AI, "start" opens the AI again
// and resumes the relay. It returns the text key of the outcome.
func Power(ctx context.Context, action string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.AIURL == "" {
		return "", errors.New("no local AI address yet")
	}
	switch action {
	case "free":
		return "power.freed", llm.Free(ctx, cfg.AIURL)
	case "stop":
		// Paused first: a request arriving meanwhile waits in the shop
		// instead of failing against an AI being closed.
		if err := SetPaused(true); err != nil {
			return "", err
		}
		return "power.stopped", llm.Stop(ctx, cfg.AIURL)
	case "start":
		if err := llm.Start(ctx, cfg.AIURL); err != nil {
			return "", err
		}
		return "power.started", SetPaused(false)
	}
	return "", errors.New("unknown action")
}

// SetPaused pauses or resumes the relay without unlinking it.
func SetPaused(paused bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Paused = paused
	return config.Save(cfg)
}

// SetLang keeps the interface language.
func SetLang(lang string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Lang = lang
	return config.Save(cfg)
}

// RememberInstalled records an AI the relay installed, the only kind it
// will offer to uninstall.
func RememberInstalled(kind string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	for _, k := range cfg.Installed {
		if k == kind {
			return nil
		}
	}
	cfg.Installed = append(cfg.Installed, kind)
	return config.Save(cfg)
}

// SetKeepLoaded keeps the model in memory between requests, or lets the
// relay free it after a while without one.
func SetKeepLoaded(keep bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.KeepLoaded = keep
	return config.Save(cfg)
}

// ErrNotOurs refuses to uninstall an AI the relay did not install.
var ErrNotOurs = errors.New("this AI was not installed by the relay")

// RemoveAI uninstalls an AI the relay installed, with its models. An AI
// installed another way is refused: it is not the relay's to remove.
func RemoveAI(ctx context.Context, kind string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	kept := cfg.Installed[:0]
	found := false
	for _, k := range cfg.Installed {
		if k == kind {
			found = true
			continue
		}
		kept = append(kept, k)
	}
	if !found {
		return ErrNotOurs
	}
	if err := llm.Uninstall(ctx, kind); err != nil {
		return err
	}
	cfg.Installed = kept
	// The address pointed at the AI just removed: nothing answers there now.
	if llm.Kind(ctx, cfg.AIURL) == kind || llm.Kind(ctx, cfg.AIURL) == "" {
		cfg.AIURL = ""
	}
	return config.Save(cfg)
}

// RemoveAll uninstalls the relay from this computer: off the startup, the
// background relay stopped, off the PATH, its copy and its settings
// deleted - and, when asked, the AIs it installed. The shop keeps a relay
// that no longer calls; removing the local assistant there unlinks it.
func RemoveAll(ctx context.Context, withAI bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	_ = service.Uninstall()
	_ = service.Stop()
	var problems []string
	keepSetup := false
	for _, kind := range cfg.Installed {
		if !withAI {
			// Ollama on Linux lives in the relay's folder: kept with it.
			keepSetup = keepSetup || kind == llm.KindOllama
			continue
		}
		if err := llm.Uninstall(ctx, kind); err != nil {
			problems = append(problems, kind+": "+err.Error())
		}
	}
	if err := pathenv.Remove(); err != nil {
		problems = append(problems, err.Error())
	}
	dir, err := config.Dir()
	if err == nil {
		if keepSetup {
			_ = os.Remove(filepath.Join(dir, "config.json"))
			_ = os.Remove(filepath.Join(dir, "state.json"))
		} else if err := os.RemoveAll(dir); err != nil {
			problems = append(problems, err.Error())
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}
