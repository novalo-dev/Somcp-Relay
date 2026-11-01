# Changelog

## 0.1.1 - 2026-11-02

### Added

**A local AI installed in one click.** Under **No local AI yet?**, pick
**Ollama** or **LM Studio** and click **Install everything**: the relay
downloads it from its official site, installs it for your account only -
no administrator rights -, starts it, downloads a model suited to your
computer's memory, tests it with the shop's tools and keeps its address. A
progress bar follows each step, and there is nothing else to type. The same
setup runs from the command line with `somcp-relay llm setup ollama` or
`somcp-relay llm setup lmstudio`. On Linux, Ollama needs the zstd tool to
unpack; the relay says so and suggests LM Studio when it is missing.

### Fixed

**Downloading a model before any AI was set up failed** with "empty
address": the download looked for the address of an AI that was not there
yet. It is now part of the one-click install, which knows the address.

## 0.1.0 - 2026-11-02

### Added

**First version of the relay.** One program for Windows, macOS and Linux
links your shop's built-in assistant to an AI running on your computer -
Ollama, LM Studio or any OpenAI-compatible software. Its interface, in
English, French, Spanish and German, links the shop with the code shown in
the back office, tests the AI address you type and lists the models that can
call tools, installs Ollama from its official site and downloads a model
suited to your computer's memory, and shows the relay's status. Everything
also works from the command line, and the relay can start with your computer
without administrator rights. The relay calls your shop, never the other way
round, and only ever calls the AI address you typed.

The interface walks you through three steps with a progress bar: link the
shop - pasting the command shown in the back office fills the address and
the code and links at once -, test your AI, then the relay runs. While the
interface is open it carries the shop's requests itself, so nothing else has
to be started; with **Start with the computer** on, the relay keeps running
after you close it. When you remove the local assistant in the back office,
the relay forgets the shop and asks for a new code.
