# O'So AI Relay

O'So AI Relay links your PrestaShop shop to an AI running on your own
computer - Ollama, LM Studio, llama.cpp, LocalAI, Jan, vLLM, or any software
that speaks the OpenAI-compatible API. Your shop's assistant then answers with
that AI: your conversations and your shop's data go to your computer, not to
an AI company.

The program is called `oso`. It runs on Windows, macOS and Linux, needs no
administrator rights, and speaks English, French, Spanish and German.

## How it works

The relay calls your shop, the way a browser does: it asks for the chat
requests waiting there, passes them to the AI on your computer and sends the
answers back. Your shop never connects to your computer, so nothing has to be
opened on your router or your network, and it works whether your shop is
hosted online or runs next to you.

The relay only ever calls the AI address you typed on your computer, never an
address sent by the shop, and passes on only requests for a model that passed
its tool test. The details are in [PROTOCOL.md](PROTOCOL.md).

## Download

Take the file for your system from the
[latest release](https://github.com/novalo-dev/Somcp-Relay/releases/latest):

| System | File |
| --- | --- |
| Windows (most PCs) | `oso-ai-relay-windows-x64.zip` |
| Windows on ARM | `oso-ai-relay-windows-arm64.zip` |
| Mac with an Apple chip (M1 and later) | `oso-ai-relay-macos-apple-silicon.zip` |
| Mac with an Intel chip | `oso-ai-relay-macos-intel.zip` |
| Linux, PC | `oso-ai-relay-linux-x64.tar.gz` |
| Linux, ARM (Raspberry Pi...) | `oso-ai-relay-linux-arm64.tar.gz` |

Unpack it: you get one program, `oso` (`oso.exe` on Windows). There is
nothing to install. `SHA256SUMS.txt` lets you check the file you downloaded.

The files are not signed yet:

- **Windows** may show SmartScreen on the first run: click **More info**,
  then **Run anyway**.
- **macOS** blocks it by default: right-click `oso`, choose **Open**, then
  confirm. Once is enough.
- **Linux**: make it executable with `chmod +x oso`.

To type `oso` from any folder, move it to a folder on your PATH - for example
`~/.local/bin` on Linux and macOS, or add its folder to the PATH on Windows.
Otherwise, run it from its folder: `.\oso.exe` on Windows, `./oso` elsewhere.

## First use, with the interface

Double-click `oso`, or run it with no command: its interface opens in your
browser, on this computer only. A progress bar follows three steps, and each
step folds into a one-line summary once done.

### 1. Link your shop

Your shop gives you a pairing code, single use and valid for 10 minutes, and a
ready-made command such as `oso pair https://myshop.com K7Q2-9XD4`.

Paste that whole command into either field of the interface: the address and
the code fill in and the shop is linked at once. You can also type the shop
address and the code by hand and click **Link**.

### 2. Your local AI

Type the address where your AI answers and click **Test**. The buttons
**Ollama · 11434** and **LM Studio · 1234** fill in their usual address and
test it; the port may differ on your computer, which is why the address is
always yours to type.

The relay lists the models of your AI and checks which of them can call
tools - a model that cannot is of no use to your shop's assistant. Only the
models marked **tools OK** are offered to your shop.

**No local AI yet?** Open that section, pick **Ollama** or **LM Studio** and
click **Install everything**. The relay then, on its own:

1. downloads it from its official site and installs it for your account only;
2. starts it;
3. downloads a model suited to your computer's memory (Qwen 2.5, 7B with
   16 GB or more, 3B with 8 GB, 1.5B below);
4. tests the model and keeps its address.

A progress bar shows each step. It downloads several GB in all: count a few
minutes on a good connection. On Linux, Ollama needs the `zstd` tool to unpack
(`sudo apt install zstd` on Debian and Ubuntu); the relay says so if it is
missing, and LM Studio needs nothing more.

### 3. Relay

The relay starts by itself once the shop is linked and the AI is ready, and
carries the shop's requests **while the interface is open**. The card shows
the last contact with the shop, the requests answered and the last problem, if
any.

- **Start with the computer** keeps the relay running after you close the
  interface, and starts it again each time you sign in.
- **Pause** keeps the link but lets the shop's requests wait.
- **Unlink the shop** forgets the shop; the assistant stops in the shop until
  you link again with a new code.

If the shop stops knowing the relay, the relay forgets it too and asks for a
new code.

## Command line

Every step also works from a terminal:

```
oso                          open the interface (same as "oso ui")
oso pair <shop> <code>       link a shop with its pairing code
oso ai <address>             set the address of your AI and test its models
oso run                      run the relay in this window (Ctrl+C to stop)
oso install                  start the relay with the computer, and start it now
oso uninstall                stop starting with the computer
oso start / oso stop         start or stop the background relay
oso status                   show the shop, the AI and whether the relay runs
oso llm list                 list the models of your AI
oso llm test [model]         check that a model can call tools (all if none given)
oso llm install <model>      download a model into Ollama
oso llm setup ollama         install Ollama and a model, then test them
oso llm setup lmstudio       install LM Studio and a model, then test them
oso version                  show the version
oso help                     show this list
```

Add `--lang en`, `fr`, `es` or `de` to any command; without it the relay
follows your system's language, English otherwise.

A typical setup on a server without a screen:

```
oso llm setup ollama
oso pair https://myshop.com K7Q2-9XD4
oso install
oso status
```

## Starting with the computer

`oso install` and the **Start with the computer** switch use what each system
offers to a user without administrator rights: a systemd user unit on Linux, a
launchd agent on macOS, a task run at sign-in on Windows. `oso uninstall`
removes it.

Only one relay carries the requests at a time: an interface opened while the
background relay runs shows its state and leaves the work to it.

## Where it keeps its settings

The shop address, the relay's token, the AI address and the language are kept
in one file readable by your account only:

- Windows: `%AppData%\somcp-relay\config.json`
- macOS: `~/Library/Application Support/somcp-relay/config.json`
- Linux: `~/.config/somcp-relay/config.json`

Deleting that folder resets the relay. The folder keeps the name of the first
versions so an update finds your settings.

## Troubleshooting

- **The shop says the relay is offline** - the relay is not running: open the
  interface, or turn on **Start with the computer**, or run `oso status`.
- **"The AI does not answer"** - your AI is not started, or listens on another
  port: start it and check its address in its own settings.
- **No model marked tools OK** - your models cannot call tools: install one
  that can, for example with **Install everything**, then click **Test again**.
- **"The shop does not answer"** - check the shop address, starting with
  `https://`; plain `http://` is accepted only for a shop on this computer or
  your local network.
- **The code is refused** - it was already used or is older than 10 minutes:
  ask your shop for a new one.

## Build

Go 1.22 or later, standard library only:

```
go build -o oso -ldflags "-X github.com/novalo-dev/somcp-relay/internal/app.Version=0.2.0" .
go test ./...
```

Every text lives in `internal/i18n/locales/` in the four languages; the tests
fail when a key is missing in one of them.

Releases are published by `.github/workflows/release.yml` when the newest
section of [CHANGELOG.md](CHANGELOG.md) names a new version.
