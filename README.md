<h1 align="center">Kilo-Agent</h1>

<p align="center">
  <b>A personal agent in a few lines of Go. No dependencies, no bloaaaaat.</b>
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.x-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img alt="License" src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" />
  <img alt="Dependencies" src="https://img.shields.io/badge/Dependencies-zero-brightgreen?style=flat-square" />
  <img alt="Status" src="https://img.shields.io/badge/Status-WIP-orange?style=flat-square" />
  <img alt="Telegram Integration" src="https://img.shields.io/badge/Telegram-Integration-26A5E4?style=flat-square&logo=telegram&logoColor=white" />
  <img alt="Lang" src="https://img.shields.io/badge/Language-Golang-blue?style=flat-square&logo=go&logoColor=white" />
  <img alt="No deps" src="https://img.shields.io/badge/No%20deps-100%25-brightgreen?style=flat-square" />
  <img alt="Platform" src="https://img.shields.io/badge/Platform-Cross--platform-lightgrey?style=flat-square" />
  <img alt="Made for" src="https://img.shields.io/badge/Made%20for-Learning-purple?style=flat-square" />
  <img alt="Code style" src="https://img.shields.io/badge/Code-Simple%20&%20Readable-ff69b4?style=flat-square" />
  <img alt="PRs" src="https://img.shields.io/badge/PRs-Welcome-brightgreen?style=flat-square" />
</p>

<p align="center">
  <img src="kilo-agent.png?v=3" alt="Kilo-Agent" width="80%" />
</p>

<p align="center">
  <img src="kilo-agent-tg.png?v=2" alt="Kilo-Agent Telegram integration" width="80%" />
</p>

---

## What is it?

**Kilo-Agent** is a tiny proof-of-concept that shows you can build your own
personal agent in **just a few lines of Go** — with **zero dependencies**.

It's a blueprint, not a framework.
This aims to be a **mallable agent**:

- **Learn** — read the code, understand the pieces
- **Extend** — add your own tools and skills
- **Have fun** with it!

## Install

Build and install Kilo Agent to `~/.local/bin`:

```bash
make install
```

Make sure `~/.local/bin` is included in your `PATH`, then run it from anywhere:

```bash
kilo-agent
```

To install somewhere else, set `BINDIR`:

```bash
sudo make install BINDIR=/usr/local/bin
```

Use the same `BINDIR` value to uninstall it:

```bash
make uninstall
# Or, for a custom installation directory:
sudo make uninstall BINDIR=/usr/local/bin
```

## Build & Run From Source

```bash
make build
./kilo-agent
```

That's it. No `go mod vendor`, no download bombs — just your machine and Go.

## System Prompt

Kilo Agent includes a default system prompt inside the executable. To replace
it with your own instructions, create:

```text
~/.config/kilo-agent/SYSTEM.md
```

The contents of that file completely override the embedded `SYSTEM.md` the
next time Kilo Agent starts. If `XDG_CONFIG_HOME` is set, use
`$XDG_CONFIG_HOME/kilo-agent/SYSTEM.md` instead. Remove the override file to
return to the embedded default.

## Terminal UI

Kilo Agent includes a dependency-light, Elm-style terminal interface: terminal
events update a model, the model produces effects, and a diff renderer redraws
only changed rows. Output and the editable input wrap to the terminal width.

- Type `@` to search for a project file and attach its contents to the prompt.
- Type `/` to browse commands. `/help` shows the complete in-app reference.
- Use all four arrow keys to move through wrapped input; use `Page Up` or
  `Page Down` for conversation scrollback.
- Use `Shift+Enter` (or `Alt+Enter`) to insert a newline without sending.
- `Ctrl+A/E`, `Ctrl+B/F`, `Alt+B/F`, `Ctrl+W`, `Ctrl+U/K`, `Ctrl+Y`, and
  `Ctrl+P/N` behave like familiar Linux terminal/readline bindings.
- Drag normally to select text; terminal copy shortcuts still apply. Bracketed
  paste preserves pasted text safely, including multiple lines.
- The bottom bar reports model activity, token usage, context consumption, and
  generation speed. Fenced `diff` blocks receive add/remove/hunk styling.

The default palette combines CyanogenMod 11 cyan and green with Omarchy-style
near-black surfaces. It uses truecolor ANSI sequences and no TUI framework.

Commands: `/help`, `/clear`, `/tokens`, and `/quit`.

## Project Layout

```
agent.go        # the core agent loop
tools.go        # built-in tool definitions
tui/            # terminal input, editor, update model, and renderer
main.go         # entry point
```

## Tests

```bash
go test ./...
```

## Disclaimer

> [!CAUTION]
> This is a **work in progress** agent blueprint. Use it at your own risk!
> It's more of a conversation starter than a production tool.
