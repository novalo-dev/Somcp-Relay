// somcp-relay links a PrestaShop shop running the somcp module to an AI
// running on this computer. It calls the shop, never the other way round,
// so nothing is opened on the local network.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/novalo-dev/somcp-relay/internal/app"
	"github.com/novalo-dev/somcp-relay/internal/config"
	"github.com/novalo-dev/somcp-relay/internal/i18n"
	"github.com/novalo-dev/somcp-relay/internal/llm"
	"github.com/novalo-dev/somcp-relay/internal/relay"
	"github.com/novalo-dev/somcp-relay/internal/service"
	"github.com/novalo-dev/somcp-relay/internal/shop"
	"github.com/novalo-dev/somcp-relay/internal/ui"
)

func main() {
	args, lang := takeLang(os.Args[1:])
	cfg, _ := config.Load()
	if lang == "" {
		lang = cfg.Lang
	}
	i18n.Set(lang)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := dispatch(ctx, args); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("err.generic", err))
		os.Exit(1)
	}
}

// takeLang pulls --lang out of the arguments, wherever it is.
func takeLang(args []string) ([]string, string) {
	var rest []string
	lang := ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--lang" && i+1 < len(args):
			lang = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--lang="):
			lang = strings.TrimPrefix(args[i], "--lang=")
		default:
			rest = append(rest, args[i])
		}
	}
	return rest, lang
}

func dispatch(ctx context.Context, args []string) error {
	if len(args) == 0 {
		args = []string{"ui"}
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Println(i18n.T("help"))
	case "version", "--version":
		fmt.Println(i18n.T("version", app.Version))
	case "pair":
		if len(args) != 3 {
			return usage("somcp-relay pair <shop> <code>")
		}
		info, err := app.Pair(ctx, args[1], args[2])
		if err != nil {
			return err
		}
		fmt.Println(i18n.T("pair.ok", info.Name))
	case "ai":
		if len(args) != 2 {
			return usage("somcp-relay ai <address>")
		}
		software, models, err := app.SetAI(ctx, args[1])
		if err != nil {
			return errors.New(i18n.T("ai.fail", args[1], err))
		}
		fmt.Println(i18n.T("ai.ok", software, len(models)))
		printModels(models)
	case "run":
		return runRelay(ctx)
	case "ui":
		address, err := ui.Start(ctx)
		if err != nil {
			return err
		}
		fmt.Println(i18n.T("ui.opening", address))
		_ = llm.OpenBrowser(address)
		<-ctx.Done()
	case "install":
		return serviceCall(service.Install, "service.installed")
	case "uninstall":
		return serviceCall(service.Uninstall, "service.uninstalled")
	case "start":
		return serviceCall(service.Start, "service.started")
	case "stop", "close":
		return serviceCall(service.Stop, "service.stopped")
	case "status":
		status()
	case "llm":
		return llmCommand(ctx, args[1:])
	default:
		return errors.New(i18n.T("err.unknown", args[0]))
	}
	return nil
}

func usage(line string) error {
	return errors.New(i18n.T("err.usage", line))
}

func serviceCall(call func() error, done string) error {
	err := call()
	if errors.Is(err, service.ErrUnsupported) {
		return errors.New(i18n.T("service.unsupported", runtime.GOOS))
	}
	if err != nil {
		return err
	}
	fmt.Println(i18n.T(done))
	return nil
}

func status() {
	cfg, _ := config.Load()
	shopName := i18n.T("status.none")
	if cfg.Linked() {
		shopName = cfg.ShopName + " (" + cfg.ShopURL + ")"
	}
	fmt.Println(i18n.T("status.shop", shopName))
	ai := cfg.AIURL
	if ai == "" {
		ai = i18n.T("status.none")
	}
	fmt.Println(i18n.T("status.ai", ai))
	if st, err := relay.ReadState(); err == nil && st.Alive() {
		fmt.Println(i18n.T("status.running", st.LastContact.Format(time.RFC3339)))
	} else {
		fmt.Println(i18n.T("status.stopped"))
	}
}

func llmCommand(ctx context.Context, args []string) error {
	// setup needs no AI yet: it is how one gets installed.
	if len(args) >= 1 && args[0] == "setup" {
		if len(args) != 2 {
			return usage("somcp-relay llm setup ollama|lmstudio")
		}
		address, err := llm.Setup(ctx, args[1], llm.MemoryGB(), func(p llm.Progress) {
			fmt.Printf("\r%s", i18n.T(p.Key, p.Args...))
		})
		fmt.Println()
		if errors.Is(err, llm.ErrNeedsZstd) {
			return errors.New(i18n.T("setup.zstd"))
		}
		if err != nil {
			return err
		}
		fmt.Println(i18n.T("setup.test"))
		software, models, err := app.SetAI(ctx, address)
		if err != nil {
			return errors.New(i18n.T("ai.fail", address, err))
		}
		fmt.Println(i18n.T("ai.ok", software, len(models)))
		printModels(models)
		fmt.Println(i18n.T("setup.done", address))
		return nil
	}
	cfg, _ := config.Load()
	if cfg.AIURL == "" {
		return errors.New(i18n.T("err.no_ai"))
	}
	ai, err := llm.New(cfg.AIURL)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return usage("somcp-relay llm list|test [model]|install <model>")
	}
	switch args[0] {
	case "list":
		models, err := ai.Models(ctx)
		if err != nil {
			return errors.New(i18n.T("ai.fail", cfg.AIURL, err))
		}
		printModels(models)
	case "test":
		models := args[1:]
		if len(models) == 0 {
			listed, err := ai.Models(ctx)
			if err != nil {
				return errors.New(i18n.T("ai.fail", cfg.AIURL, err))
			}
			for _, m := range listed {
				models = append(models, m.ID)
			}
		}
		if len(models) == 0 {
			return errors.New(i18n.T("llm.no_model"))
		}
		for _, model := range models {
			ok, err := ai.TestTools(ctx, model)
			if ok && err == nil {
				fmt.Println(i18n.T("llm.test_ok", model))
			} else {
				fmt.Println(i18n.T("llm.test_fail", model))
			}
		}
	case "install":
		if len(args) != 2 {
			return usage("somcp-relay llm install <model>")
		}
		if !llm.OllamaAnswers(ctx, ai.Base()) {
			return errors.New(i18n.T("ollama.missing", llm.OllamaRoot(ai.Base()), llm.DownloadPage))
		}
		fmt.Println(i18n.T("llm.pulling", args[1]))
		err := llm.Pull(ctx, ai.Base(), args[1], func(p int) {
			fmt.Printf("\r%s", i18n.T("llm.progress", args[1], p))
		})
		fmt.Println()
		if err != nil {
			return err
		}
		fmt.Println(i18n.T("llm.pulled", args[1]))
	default:
		return errors.New(i18n.T("err.unknown", "llm "+args[0]))
	}
	return nil
}

func printModels(models []llm.Model) {
	for _, m := range models {
		label := i18n.T("llm.tools_untested")
		if m.Tools != nil && *m.Tools {
			label = i18n.T("llm.tools_ok")
		} else if m.Tools != nil {
			label = i18n.T("llm.tools_no")
		}
		fmt.Printf("  %-40s %s\n", m.ID, label)
	}
}

type consoleLog struct{}

func (consoleLog) Started(shopName, ai string) { fmt.Println(i18n.T("run.start", shopName, ai)) }
func (consoleLog) Paused()                     { fmt.Println(i18n.T("run.paused")) }
func (consoleLog) Answered(d time.Duration) {
	fmt.Println(i18n.T("run.answered", d.Round(100*time.Millisecond)))
}
func (consoleLog) Retry(err error, wait time.Duration) {
	fmt.Println(i18n.T("run.retry", err, wait))
}

func runRelay(ctx context.Context) error {
	cfg, _ := config.Load()
	if !cfg.Linked() {
		return errors.New(i18n.T("err.not_linked"))
	}
	if cfg.AIURL == "" {
		return errors.New(i18n.T("err.no_ai"))
	}
	err := relay.Run(ctx, consoleLog{})
	if errors.Is(err, shop.ErrRevoked) {
		return errors.New(i18n.T("run.unlinked"))
	}
	return err
}
