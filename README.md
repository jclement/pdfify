# pdfify

Convert Markdown to beautifully styled PDFs via Docker.

> Making documentation look like it had a graphic designer

## Quick Start

1. [Install Go](https://go.dev/dl/) and [Docker](https://docs.docker.com/get-docker/)
2. Install pdfify:
   ```bash
   go install github.com/jclement/pdfify/cmd/pdfify@latest
   ```
   Or download a binary from [Releases](https://github.com/jclement/pdfify/releases).
3. Convert a markdown file:
   ```bash
   pdfify document.md
   ```
4. The Docker image builds automatically on first run (~2-3 minutes, cached after).

## Features

- **Markdown to PDF** — pandoc + XeLaTeX in a Docker container
- **Mermaid diagrams** — rendered to PNG automatically
- **Obsidian callouts** — info, tip, warning, danger, example, quote
- **Themes** — `default` (clean professional) and `party` (bold colorful)
- **Paper sizes** — letter, a4, a5, legal, executive
- **Auto-detection** — smart heading numbering based on document structure
- **Watch mode** — `--watch` opens browser with auto-reload on file changes
- **Edit mode** — `--edit` opens split-pane editor with live PDF preview
- **Self-update** — `pdfify update` downloads latest release
- **Docker image** — `docker run ghcr.io/jclement/pdfify` works standalone
- **Three-tier config** — CLI flags > frontmatter > preferences file

## Usage

```bash
# Basic conversion
pdfify document.md

# With options
pdfify document.md --theme party --paper a4 --watermark DRAFT

# Preview (renders to temp, opens in browser)
pdfify document.md --preview

# Watch mode (browser auto-reloads on save)
pdfify document.md --watch

# Edit mode (browser-based editor + live preview)
pdfify document.md --edit

# Multiple files
pdfify chapter1.md chapter2.md chapter3.md

# Custom output path
pdfify document.md -o output/report.pdf

# Show example document
pdfify --example

# Check setup
pdfify --doctor

# Add frontmatter to a file
pdfify document.md --headerify
```

## Configuration

Every setting can be set via CLI flag, YAML frontmatter, or preferences file. Priority: **CLI > frontmatter > preferences > defaults**.

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-o, --output` | Output PDF file path | `<input>.pdf` |
| `--title` | Document title | *(auto-detected from H1)* |
| `--subtitle` | Document subtitle | *(none)* |
| `--author` | Document author | *(none)* |
| `--date` | Document date (`none` to suppress) | *(none)* |
| `--header` | Page header text | *(none)* |
| `--footer` | Page footer text | *(none)* |
| `--watermark` | Diagonal watermark text | *(none)* |
| `--toc-level` | TOC depth: 0=none, 1-4 | `3` |
| `--numbers` / `--no-numbers` | Section numbering | `true` |
| `--number-from` | Start numbering at heading level (1-4) | `2` |
| `--page-break` / `--no-page-break` | Page break before top sections | `true` |
| `--paper` | Paper size | `letter` |
| `--theme` | Visual theme | `default` |
| `--watch` | Watch + browser auto-reload | `false` |
| `--edit` | Browser editor + live preview | `false` |
| `--preview` | Render to temp + open | `false` |
| `--open` | Open PDF after generation | `false` |
| `--rebuild` | Force rebuild Docker image | `false` |

### Frontmatter

```yaml
---
title: "My Document"
subtitle: "A subtitle"
author: "Your Name"
date: "2026-04-08"
header: "CONFIDENTIAL"
footer: "Acme Corp"
toc-level: 3
numbersections: true
numberfrom: 2
pagebreak: true
papersize: letter
theme: default
watermark: "DRAFT"
---
```

### Preferences File

`~/.config/pdfify/preferences.yaml`:

```yaml
author: "Jeff Clement"
paper_size: letter
theme: default
toc_level: 3
number_sections: true
number_from: 2
```

## Themes

| Theme | Description |
|-------|-------------|
| `default` | Clean and professional — Roboto, muted grays and blues |
| `party` | Bold and colorful — vibrant headings, playful callouts |

Set via `--theme`, frontmatter `theme:`, or preferences.

## Paper Sizes

`letter`, `a4`, `a5`, `legal`, `executive`

## Docker Usage

Run pdfify directly via Docker (no Go installation needed):

```bash
docker run --rm -v $(pwd):/workspace ghcr.io/jclement/pdfify:latest /workspace/document.md
```

## Utility Flags

| Flag | Description |
|------|-------------|
| `--doctor` | Check that Docker, image, and config are set up |
| `--example` | Print an example markdown document |
| `--headerify` | Add/update YAML frontmatter in the input file(s) |
| `--update` | Self-update to latest release |
| `--clean` | Remove the Docker image |
| `--themes` | List available themes |
| `--version` | Show version info |

## Development

| Command | What it does |
|---------|-------------|
| `mise run dev` | Build and run pdfify |
| `mise run test` | Run all tests |
| `mise run lint` | Lint and vet |
| `mise run build` | Production build |
| `mise run fmt` | Format all code |
| `mise run release` | Tag and push a release |

## Architecture

See [DESIGN.md](DESIGN.md) for architecture details.

The conversion pipeline:

```
Markdown --> [Parse Config] --> [Analyze Structure] --> [Docker Container]
                                                             |
                                                   [Callout Processing]
                                                   [Mermaid Rendering]
                                                   [Page Break Injection]
                                                   [pandoc + XeLaTeX]
                                                             |
                                                           [PDF]
```

## License

(C) 2026 Jeff Clement
