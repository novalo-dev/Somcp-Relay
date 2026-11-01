# somcp-relay

Links a PrestaShop shop running the [O'So MCP Server](https://github.com/novalo-dev/Prestashop-MCP-Server)
module to an AI running on your own computer: Ollama, LM Studio, llama.cpp,
LocalAI, Jan, vLLM - anything that speaks the OpenAI-compatible API.

The module's built-in assistant then answers with your local model: your
shop's data and your conversations go to your computer, not to an AI
company.

## How it works

The relay calls your shop, the way a browser does, asks it for chat requests,
passes them to the AI on your computer and sends the answers back. Your shop
never connects to your computer, so nothing is opened on your network, and it
works whether the shop is hosted online or runs next to you.

The relay only ever calls the AI address you typed, never an address sent by
the shop, and passes on only requests for a model that passed its tool test.
The details are in [PROTOCOL.md](PROTOCOL.md).

## Install

Download the file for your system from the releases, then run it. With no
command it opens its interface in your browser:

1. **Shop** - type your shop's address and the pairing code shown in the back
   office under **Built-in assistant > Add a local assistant**.
2. **Local AI** - type the address where your AI runs (`http://localhost:11434`
   for Ollama, `http://localhost:1234` for LM Studio, or your own port) and
   click **Test**. Only the models that can call tools are offered in the shop.
   No local AI yet? The same screen installs Ollama from its official site and
   downloads a model suited to your computer's memory.
3. **Status** - pause the relay, make it start with your computer, or unlink
   the shop.

The interface speaks English, French, Spanish and German; it follows your
system's language when it is one of those, English otherwise.

## Command line

```
somcp-relay ui                     open the interface
somcp-relay pair <shop> <code>     link a shop
somcp-relay ai <address>           set the local AI and test it
somcp-relay run                    run the relay in this window
somcp-relay install / uninstall    start, or stop starting, with the computer
somcp-relay start / stop           start or stop the background relay
somcp-relay status                 show the shop, the AI and the relay
somcp-relay llm list               list the models of the local AI
somcp-relay llm test [model]       check that a model can call tools
somcp-relay llm install <model>    download a model into Ollama
somcp-relay version | help
```

Add `--lang en|fr|es|de` to any command.

Starting with the computer needs no administrator rights: a systemd user unit
on Linux, a launchd agent on macOS, a logon task on Windows.

## Unsigned builds

The files are not signed yet. Windows SmartScreen may warn on the first run
(**More info > Run anyway**), and macOS blocks it by default (right-click the
file, **Open**, then confirm).

## Build

Go 1.22 or later, standard library only:

```
go build -ldflags "-X github.com/novalo-dev/somcp-relay/internal/app.Version=0.1.0" .
go test ./...
```

Every text lives in `internal/i18n/locales/` in the four languages; the tests
fail when a key is missing in one of them.
