# O'So AI Relay

For PrestaShop shops using the **O'So MCP Server** module: O'So AI Relay links
the shop's built-in assistant to an AI running on your own computer (Ollama,
LM Studio or any OpenAI-compatible software). Nothing to open on your network,
no API key to pay for.

## Install

1. Download the archive for your system from the
   [latest release](https://github.com/novalo-dev/Somcp-Relay/releases/latest):
   `windows-x64` (or `windows-arm64`), `macos-apple-silicon` (or
   `macos-intel`), `linux-x64` (or `linux-arm64`).
2. Extract the archive.
3. Open a command window: Command Prompt or PowerShell on Windows, Terminal on
   macOS and Linux.
4. Go to the extracted folder and install `oso` for your account:

   ```
   cd <path to the extracted folder>
   .\oso.exe addtopath    (Windows)
   ./oso addtopath        (macOS, Linux)
   ```

   On macOS, run `xattr -d com.apple.quarantine oso` first. On Windows, if
   SmartScreen warns: **More info**, then **Run anyway**.
5. Open a new command window and run:

   ```
   oso ui
   ```

The downloaded files can then be deleted.

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
oso addtopath                  install oso for your account, on your PATH
oso version | help
```

Add `--lang en|fr|es|de` to any command.
