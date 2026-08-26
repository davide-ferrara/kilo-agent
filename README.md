<p align="center">
  <img src="kilo-agent.png" alt="Kilo-Agent" width="80%" />
</p>

<h1 align="center">Kilo-Agent</h1>

<p align="center">
  <b>A personal agent in a few lines of Go. No dependencies, no bloaaaaat.</b>
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.x-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img alt="License" src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" />
  <img alt="Dependencies" src="https://img.shields.io/badge/Dependencies-zero-brightgreen?style=flat-square" />
  <img alt="Status" src="https://img.shields.io/badge/Status-WIP-orange?style=flat-square" />
  <img alt="Lang" src="https://img.shields.io/badge/Language-Golang-blue?style=flat-square&logo=go&logoColor=white" />
  <img alt="No deps" src="https://img.shields.io/badge/No%20deps-100%25-brightgreen?style=flat-square" />
  <img alt="Platform" src="https://img.shields.io/badge/Platform-Cross--platform-lightgrey?style=flat-square" />
  <img alt="Made for" src="https://img.shields.io/badge/Made%20for-Learning-purple?style=flat-square" />
  <img alt="Code style" src="https://img.shields.io/badge/Code-Simple%20&%20Readable-ff69b4?style=flat-square" />
  <img alt="PRs" src="https://img.shields.io/badge/PRs-Welcome-brightgreen?style=flat-square" />
</p>

---

## What is it?

**Kilo-Agent** is a tiny proof-of-concept that shows you can build your own
personal agent in **just a few lines of Go** — with **zero dependencies**.

It's a blueprint, not a framework:

- **Learn** — read the code, understand the pieces
- **Extend** — add your own tools and skills
- **Have fun** with it!

## Build & Run

```bash
make build
./kilo-agent
```

That's it. No `go mod vendor`, no download bombs — just your machine and Go.

## Project Layout

```
agent.go        # the core agent loop
tools.go        # built-in tool definitions
input.go        # input handling
types.go        # shared types
tui.go          # terminal UI
main.go         # entry point
```

## Tests

```bash
go test ./...
```

## Disclaimer

> [!DANGER]
> This is a **work in progress** agent blueprint. Use it at your own risk!
> It's more of a conversation starter than a production tool.
