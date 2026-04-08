# pdfify — Claude Code Guidelines

## Project Overview

Go CLI tool that converts Markdown to beautiful PDFs via Docker (pandoc + XeLaTeX + mermaid-cli).

## Development

- **Build:** `CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=dev" -o bin/pdfify ./cmd/pdfify`
- **Test:** `go test -v -race ./...`
- **Vet:** `go vet ./...`
- **Format:** `go fmt ./...`
- **Run:** `./bin/pdfify examples/basic.md`

## Architecture

- `cmd/pdfify/main.go` — Cobra CLI, all subcommands
- `internal/config/` — Three-tier config (CLI > frontmatter > preferences)
- `internal/theme/` — Theme definitions (default, party)
- `internal/converter/` — Markdown preprocessing + LaTeX preamble generation
- `internal/docker/` — Docker image lifecycle (build, inspect, run)
- `internal/server/` — HTTP server for --watch and --edit modes
- `internal/updater/` — Self-update via GitHub Releases
- `internal/ui/` — Terminal output formatting
- `internal/taglines/` — Fun one-liners

## Rules

- CGO_ENABLED=0 always
- Run tests after every change
- Update README.md and DESIGN.md with code changes
- All exported functions need doc comments
