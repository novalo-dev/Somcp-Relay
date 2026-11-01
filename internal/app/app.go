// Package app holds the actions the command line and the interface share,
// so both pair a shop and set the AI through the same code.
package app

import (
	"context"
	"errors"
	"time"

	"github.com/novalo-dev/somcp-relay/internal/config"
	"github.com/novalo-dev/somcp-relay/internal/llm"
	"github.com/novalo-dev/somcp-relay/internal/relay"
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
