# O'So AI Relay

[![PrestaShop](.github/badges/prestashop.svg)](https://github.com/novalo-dev/Prestashop-MCP-Server)
[![Go](.github/badges/go.svg)](https://go.dev)
![Platforms](.github/badges/platforms.svg)
[![Ollama](.github/badges/ollama.svg)](https://ollama.com)
[![LM Studio](.github/badges/lmstudio.svg)](https://lmstudio.ai)
[![License](.github/badges/license.svg)](LICENSE.txt)
[![Release](https://img.shields.io/github/v/release/novalo-dev/Somcp-Relay?style=flat&label=release)](https://github.com/novalo-dev/Somcp-Relay/releases/latest)
[![CI](https://github.com/novalo-dev/Somcp-Relay/actions/workflows/ci.yml/badge.svg?branch=develop)](https://github.com/novalo-dev/Somcp-Relay/actions/workflows/ci.yml)
[![Stars](https://img.shields.io/github/stars/novalo-dev/Somcp-Relay?style=flat&logo=github&label=Stars)](https://github.com/novalo-dev/Somcp-Relay/stargazers)

For PrestaShop shops using the
[O'So MCP Server](https://github.com/novalo-dev/Prestashop-MCP-Server) module:
O'So AI Relay links the shop's built-in assistant to an AI running on your own
computer (Ollama, LM Studio or any OpenAI-compatible software). Nothing to
open on your network, no API key to pay for.

## Install

Copy one line into a terminal: it downloads the latest release for your
computer, checks it and installs `oso` for your account. No administrator
rights needed.

**Windows** - in PowerShell:

```
irm https://github.com/novalo-dev/Somcp-Relay/releases/latest/download/install.ps1 | iex
```

**macOS and Linux** - in Terminal:

```
curl -fsSL https://github.com/novalo-dev/Somcp-Relay/releases/latest/download/install.sh | sh
```

Then run:

```
oso ui
```

On macOS and Linux, open a new terminal first.

<details>
<summary>Install by hand</summary>

1. Download the archive for your system from the
   [latest release](https://github.com/novalo-dev/Somcp-Relay/releases/latest).
2. Extract it, open a command window in the extracted folder and run
   `.\oso.exe addtopath` (Windows) or `./oso addtopath` (macOS, Linux).
   On macOS, run `xattr -d com.apple.quarantine oso` first.
3. Open a new command window and run `oso ui`.

</details>

## Then follow the steps on screen

The interface opens in your browser:

1. **Shop** - paste the command your shop's back office shows: the shop is linked.
2. **Local AI** - type your AI's address and click **Test**. If you have
   none, choose a model (the one recommended for your computer is already
   selected) and click **Install everything** under Ollama or LM Studio.
3. **Relay** - it runs on its own. Turn on **Start with the computer** to keep
   it running after you close the window.

## Command line

```
oso ui                         open the interface
oso pair <shop> <code>         link a shop
oso ai <address>               set your AI's address and test it
oso ai free|stop|start         free your AI's memory, stop or start it
oso llm setup ollama|lmstudio [1.5b|3b|7b|14b]
                               install a local AI and a model
oso llm list                   list your AI's models
oso llm test [model]           check that a model can call tools
oso llm install <model>        download a model into Ollama
oso llm remove ollama|lmstudio uninstall an AI oso installed, with its models
oso run                        run the relay in this window
oso install / uninstall        start, or stop starting, with the computer
oso start / stop               start or stop the background relay
oso status                     show the shop, the AI and the relay
oso addtopath                  install oso for your account, on your PATH
oso remove [--keep-ai]         uninstall oso (and the AIs it installed)
oso version | help
```

Add `--lang en|fr|es|de` to any command.

## License

[Academic Free License 3.0](LICENSE.txt). Made by
[Novalo](https://novalo.fr) - [sponsor the project](https://github.com/sponsors/novalo-dev).
