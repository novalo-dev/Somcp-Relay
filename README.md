# O'So AI Relay

For PrestaShop shops using the **O'So MCP Server** module: O'So AI Relay links
the shop's built-in assistant to an AI running on your own computer (Ollama,
LM Studio or any OpenAI-compatible software). Nothing to open on your network,
no API key to pay for.

## Install

1. Download the archive for your system from the
   [latest release](https://github.com/novalo-dev/Somcp-Relay/releases/latest).
2. Extract it, then open a terminal in the extracted folder and start the
   interface.

**Windows** - `oso-ai-relay-windows-x64.zip` (`windows-arm64` for ARM PCs).
Right-click the archive, **Extract All**, then in PowerShell:

```
cd $HOME\Downloads\oso-ai-relay-windows-x64
.\oso.exe ui
```

If SmartScreen warns: **More info**, then **Run anyway**.

**macOS** - `oso-ai-relay-macos-apple-silicon.zip` (M1 and later) or
`oso-ai-relay-macos-intel.zip`. Double-click the archive, then in Terminal:

```
cd ~/Downloads
xattr -d com.apple.quarantine oso
./oso ui
```

**Linux** - `oso-ai-relay-linux-x64.tar.gz` (`linux-arm64` for ARM):

```
mkdir oso && tar -xzf oso-ai-relay-linux-x64.tar.gz -C oso
cd oso
./oso ui
```

To type plain `oso` from any folder afterwards, run `.\oso.exe addtopath`
(Windows) or `./oso addtopath` (macOS, Linux), then open a new terminal.

## Then follow the steps on screen

The interface opens in your browser:

1. **Shop** - paste the command your shop's back office shows: the shop is linked.
2. **Local AI** - type your AI's address and click **Test**, or click
   **Install everything** under Ollama or LM Studio if you have none.
3. **Relay** - it runs on its own. Turn on **Start with the computer** to keep
   it running after you close the window.

## Command line

```
oso ui                         open the interface
oso pair <shop> <code>         link a shop
oso ai <address>               set your AI's address and test it
oso llm setup ollama|lmstudio  install a local AI and a model
oso llm list                   list your AI's models
oso llm test [model]           check that a model can call tools
oso llm install <model>        download a model into Ollama
oso run                        run the relay in this window
oso install / uninstall        start, or stop starting, with the computer
oso start / stop               start or stop the background relay
oso status                     show the shop, the AI and the relay
oso addtopath                  type oso from any folder
oso version | help
```

Add `--lang en|fr|es|de` to any command.
