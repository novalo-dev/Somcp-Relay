# Changelog

## 0.3.6 - 2026-11-03

### Security

**Built with the latest maintained Go.** The programs were built with Go 1.22,
which no longer receives security fixes. They are now built with the current
stable Go, whose standard library - the part that speaks https to your shop
and downloads the AIs - carries every published fix.

## 0.3.5 - 2026-11-02

### Added

**`oso update`.** One command installs the latest version: it downloads it
over https, checks it against the release's checksums and puts it in place of
the program, then restarts the relay if it starts with the computer. Versions
before 0.3.5 do not have it: install 0.3.5 once with the line of the README.

### Changed

**Twenty models to choose from, for Ollama and for LM Studio.** Each list
goes from the lightest model to the most capable, with its download size and
the memory it needs; all of them can act on your shop. The one suited to your
computer is marked recommended and selected, and a model bigger than your
computer's memory is flagged but can still be installed. On the command line,
`oso llm setup` takes any model name, for example `oso llm setup ollama
qwen3:14b`.

## 0.3.4 - 2026-11-02

### Security

**Ollama and LM Studio are only downloaded over https.** The one-click setup
now refuses an installer address, or a redirect, that is not https: what it
downloads is run right after, so nothing on the way can swap it.

## 0.3.3 - 2026-11-02

### Security

**The relay's key only travels encrypted.** Plain http is now accepted only
for an address that really is on this computer or the local network, and the
relay no longer follows a shop that sends it from https to http: it says so
and asks for the shop's https address.

## 0.3.2 - 2026-11-02

### Added

**Install in one line.** One command in PowerShell, or in Terminal on macOS
and Linux, downloads the latest version for your computer, checks it and
installs `oso` for your account: no archive to extract, no folder to find.
The README gives both lines.

**Choose the model when installing an AI.** The relay's interface now lists
the models it can install, from the lightest to the most capable, with the
memory each one takes. The one suited to your computer is marked recommended
and selected, and you can pick another. On the command line:
`oso llm setup ollama 14b`.

## 0.3.1 - 2026-11-02

### Fixed

**Linking a shop no longer fails with a bare "HTTP 503" on some hosts.** The
relay called the shop without naming itself, and many hosts' anti-bot
protections refuse such calls before they reach the shop. The relay now
introduces itself as O'So AI Relay. When the host still refuses it, the
relay says that the host refused it, names what answered (LiteSpeed,
cloudflare...) and tells you to allow it in the hosting panel or ask your
host. An address where the module does not answer, and a shop that fails,
get their own messages too.

## 0.3.0 - 2026-11-02

### Added

**A new interface.** Setting up shows the steps on the left - link the shop,
your AI, running - with one screen each: paste the back office command, then
test your AI or install Ollama or LM Studio in one click. Once running, the
screen shows the relay's state, the shop, the AI, the model and the memory it
takes, the requests of the day and their average time, and a **live journal**
of every request: answered, slow, refused, and why. The buttons to free the
memory, pause or stop everything sit at the top; when stopped, **Start
everything** brings it all back.

**The memory is freed after 15 minutes without a request**, so an idle
assistant no longer slows the computer down; the next question loads the
model again. It can be turned off in the settings.

**Uninstall from the relay.** The settings uninstall an AI the relay installed
itself, with its models, and O'So AI Relay itself: off the startup and the
PATH, its copy and settings deleted, and the AIs it installed too if you
leave the box ticked. From the command line: `oso llm remove ollama`,
`oso remove`.

### Changed

**Licence and sponsor.** The relay is published under the Academic Free
License 3.0, like the O'So MCP Server module, and the repository has its
Sponsor button. The README links to the module and shows its badges.

## 0.2.4 - 2026-11-02

### Added

**Free memory, Stop everything, Start everything.** A local model held in
memory can slow the whole computer down. Three buttons on the relay card, for
Ollama and LM Studio: **Free memory** unloads the model and leaves the AI
running, the next question loading it again; **Stop everything** also closes
the AI and pauses the relay; **Start everything** opens the AI again, waits
until it answers and resumes the relay. The same from the command line:
`oso ai free`, `oso ai stop`, `oso ai start`.

## 0.2.3 - 2026-11-02

### Changed

**`oso addtopath` installs the relay for your account.** It used to put the
extracted folder on your PATH, so deleting the download broke the `oso`
command. It now copies `oso` into a fixed folder of your account -
`%LOCALAPPDATA%\Programs\oso` on Windows, `~/.local/bin` on macOS and Linux -
and puts that folder on your PATH: the archive and the extracted folder can
be deleted. A relay already starting with your computer is moved to the new
copy. The README now starts with this command, then `oso ui`.

## 0.2.2 - 2026-11-02

### Added

**`oso addtopath`.** Run it once from the folder where you extracted the relay
and `oso` works from any new terminal: on Windows it adds the folder to your
account's PATH, on macOS and Linux to the startup file of your shell. No
administrator rights.

### Changed

**A short README**: what the relay is for, how to install it on Windows, macOS
and Linux, the three steps on screen, and the commands.

## 0.2.1 - 2026-11-02

### Changed

**The relay rests when nobody uses the shop's chat.** It used to keep one
request open on your shop at all times, which held one of the shop's PHP
processes for as long as the relay ran. When the chat has not been used for
a few minutes, the shop now answers at once and the relay comes back every 15
seconds; as soon as someone opens the chat, it goes back to answering without
delay.

## 0.2.0 - 2026-11-02

### Changed

**The relay is now O'So AI Relay, and its command is `oso`.** Every command
is typed with the short name - `oso pair`, `oso ai`, `oso llm setup ollama` -
and the interface carries the new name. Each release file is now an archive,
`oso-ai-relay-<system>.zip` (`.tar.gz` on Linux), holding a single program
called `oso`. Your settings and the start with the computer set up by an
earlier version are kept and found as they are.

The README now explains every step of the program on its own: download,
the interface, the one-click install of a local AI, the command line,
starting with the computer, where the settings live, and what to do when
something does not work.

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
