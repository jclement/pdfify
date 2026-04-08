// pdfify converts Markdown documents to beautifully styled PDFs.
// It uses Docker to run pandoc + XeLaTeX + mermaid-cli in a container,
// with support for themes, callouts, mermaid diagrams, tables, and more.
//
// Usage: pdfify <file.md> [flags]
package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jclement/pdfify/internal/config"
	"github.com/jclement/pdfify/internal/converter"
	"github.com/jclement/pdfify/internal/docker"
	"github.com/jclement/pdfify/internal/server"
	"github.com/jclement/pdfify/internal/taglines"
	"github.com/jclement/pdfify/internal/theme"
	"github.com/jclement/pdfify/internal/ui"
	"github.com/jclement/pdfify/internal/updater"
	"github.com/spf13/cobra"
)

// Set by goreleaser ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	w := ui.Stderr()

	root := &cobra.Command{
		Use:   "pdfify [file.md ...] [flags]",
		Short: "Convert Markdown to beautiful PDFs",
		Long: fmt.Sprintf(`pdfify — Markdown to beautiful PDF via Docker

  %s

Converts Markdown files to professionally styled PDFs using pandoc + XeLaTeX
inside a Docker container. Supports mermaid diagrams, callouts, tables,
code blocks with syntax highlighting, themes, and more.

Every setting can be configured via CLI flags, YAML frontmatter, or
preferences file (run --init to create).
Priority: CLI > frontmatter > preferences > defaults.

Examples:
  pdfify document.md                     Convert to PDF
  pdfify document.md --theme party       Convert with party theme
  pdfify document.md --watch             Watch + browser auto-reload
  pdfify document.md --edit              Browser editor + live preview
  pdfify document.md --headerify         Inject/update frontmatter
  pdfify --doctor                        Check Docker, image, config
  pdfify --example                       Print example markdown
  pdfify --themes                        List available themes
  pdfify --update                        Self-update to latest
  pdfify --init                          Generate preferences file
  pdfify --clean                         Remove Docker image`, taglines.Random()),
		Args:              cobra.ArbitraryArgs,
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
		RunE:              run,
		Version:           fmt.Sprintf("%s (%s, %s)\n  %s", version, commit, date, taglines.Random()),
	}

	// --- Utility flags (no file required) ---
	root.Flags().Bool("doctor", false, "Check that Docker, image, and config are set up correctly")
	root.Flags().Bool("example", false, "Print an example markdown document with all supported features")
	root.Flags().Bool("themes", false, "List available themes")
	root.Flags().Bool("update", false, "Self-update pdfify to the latest version")
	root.Flags().Bool("clean", false, "Remove the pdfify Docker image")
	root.Flags().Bool("init", false, "Generate or update the preferences file with documentation")

	// --- File operation flags (require a file) ---
	root.Flags().Bool("headerify", false, "Add or update YAML frontmatter in the input file(s)")
	root.Flags().Bool("watch", false, "Watch file and auto-rebuild (opens browser preview)")
	root.Flags().Bool("edit", false, "Open split-pane editor with live preview in browser")
	root.Flags().Bool("preview", false, "Render to temp file and open in browser")
	root.Flags().Bool("open", false, "Open PDF after generation")
	root.Flags().Bool("rebuild", false, "Force rebuild the Docker image before converting")

	// --- Conversion settings ---
	root.Flags().StringP("output", "o", "", "Output PDF file path")
	root.Flags().String("title", "", "Document title")
	root.Flags().String("subtitle", "", "Document subtitle")
	root.Flags().String("author", "", "Document author")
	root.Flags().String("date", "", "Document date (use 'none' to suppress)")
	root.Flags().String("header", "", "Page header text")
	root.Flags().String("footer", "", "Page footer text")
	root.Flags().String("watermark", "", "Diagonal watermark text")
	root.Flags().Int("toc-level", 0, "TOC depth: 0=none, 1-4 (default: 3)")
	root.Flags().Bool("numbers", false, "Enable section numbering")
	root.Flags().Bool("no-numbers", false, "Disable section numbering")
	root.Flags().Int("number-from", 0, "Start numbering at heading level N (1-4)")
	root.Flags().Bool("page-break", false, "Enable page breaks before top sections")
	root.Flags().Bool("no-page-break", false, "Disable page breaks before top sections")
	root.Flags().String("paper", "", "Paper size: letter, a4, a5, legal, executive")
	root.Flags().String("theme", "", "Visual theme: default, party")

	if err := root.Execute(); err != nil {
		w.Error(err.Error())
		os.Exit(1)
	}
}

// run is the single entry point. It dispatches based on which flags are set.
func run(cmd *cobra.Command, args []string) error {
	w := ui.Stderr()

	// --- Utility flags (no file needed) ---

	if doctor, _ := cmd.Flags().GetBool("doctor"); doctor {
		return runDoctor(w)
	}
	if example, _ := cmd.Flags().GetBool("example"); example {
		fmt.Print(exampleDocument)
		return nil
	}
	if themes, _ := cmd.Flags().GetBool("themes"); themes {
		return runThemes(w)
	}
	if update, _ := cmd.Flags().GetBool("update"); update {
		return runUpdate(w)
	}
	if clean, _ := cmd.Flags().GetBool("clean"); clean {
		return runClean(w)
	}
	if init, _ := cmd.Flags().GetBool("init"); init {
		return runInit(w)
	}

	// --- Everything below needs at least one file ---

	if len(args) == 0 {
		return cmd.Help()
	}

	// --- Headerify mode (modifies file, no conversion) ---

	if headerify, _ := cmd.Flags().GetBool("headerify"); headerify {
		return runHeaderify(args, w)
	}

	// --- Conversion modes ---

	cfg := buildConfigFromFlags(cmd)

	output, _ := cmd.Flags().GetString("output")
	if output != "" && len(args) > 1 {
		return fmt.Errorf("--output cannot be used with multiple input files")
	}
	cfg.Output = output

	w.Header(fmt.Sprintf("pdfify v%s", version))

	// Ensure Docker image is ready
	if err := ensureDockerImage(w, cfg.Rebuild); err != nil {
		return err
	}

	// Edit mode
	if edit, _ := cmd.Flags().GetBool("edit"); edit {
		if len(args) != 1 {
			return fmt.Errorf("--edit supports a single file")
		}
		return runEditMode(args[0], cfg, w)
	}

	// Watch mode
	if watch, _ := cmd.Flags().GetBool("watch"); watch {
		if len(args) != 1 {
			return fmt.Errorf("--watch supports a single file")
		}
		return runWatchMode(args[0], cfg, w)
	}

	// Standard conversion
	failed := 0
	for _, inputFile := range args {
		if err := convertFile(inputFile, cfg, w); err != nil {
			w.Error(fmt.Sprintf("Failed: %s — %s", inputFile, err))
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d file(s) failed", failed, len(args))
	}

	w.Header(fmt.Sprintf("Complete! (%d file(s))", len(args)))

	// Non-blocking update check
	go func() {
		if result := updater.Check(version); result != nil && result.UpdateAvailable {
			w.Warn(fmt.Sprintf("Update available: %s → %s (run: pdfify --update)", version, result.LatestVersion))
		}
	}()

	return nil
}

// ---------------------------------------------------------------------------
// Utility operations
// ---------------------------------------------------------------------------

func runDoctor(w *ui.Writer) error {
	w.Header("pdfify doctor")
	allOk := true

	w.Info("Checking Docker...")
	if docker.IsDockerAvailable() {
		w.Success("Docker is available")
	} else {
		w.Error("Docker is not available")
		w.Detail("Install Docker: https://docs.docker.com/get-docker/")
		allOk = false
	}

	w.Info("Checking Docker image...")
	status, detail, _ := docker.Inspect(version)
	switch status {
	case docker.StatusReady:
		w.Success(fmt.Sprintf("Docker image ready (%s)", detail))
	case docker.StatusOutdated:
		w.Warn(fmt.Sprintf("Docker image outdated: %s", detail))
		w.Detail("It will rebuild automatically on next conversion")
	case docker.StatusMissing:
		w.Warn("Docker image not built yet")
		w.Detail("It will be built automatically on first run")
	}

	w.Info("Checking preferences...")
	prefsPath := config.PreferencesPath()
	if _, err := os.Stat(prefsPath); err == nil {
		w.Success(fmt.Sprintf("Preferences file: %s", prefsPath))
	} else {
		w.Detail(fmt.Sprintf("No preferences file (optional): %s", prefsPath))
	}

	w.Info("Checking for updates...")
	if result := updater.Check(version); result != nil {
		if result.UpdateAvailable {
			w.Warn(fmt.Sprintf("Update available: %s → %s", version, result.LatestVersion))
		} else {
			w.Success(fmt.Sprintf("Up to date: v%s", version))
		}
	} else {
		w.Detail("Could not check for updates (network issue or dev build)")
	}

	w.Blank()
	if allOk {
		w.Success("Everything looks good!")
	} else {
		w.Error("Some issues found — see above")
	}
	return nil
}

func runThemes(w *ui.Writer) error {
	w.Header("Available Themes")
	for _, name := range theme.Names() {
		t := theme.Get(name)
		if t != nil {
			w.Info(fmt.Sprintf("%s — %s", t.Name, t.Description))
		}
	}
	w.Blank()
	w.Detail("Set theme via: --theme <name>, frontmatter 'theme:', or preferences.yaml")
	return nil
}

func runUpdate(w *ui.Writer) error {
	w.Info("Checking for updates...")

	result := updater.Check(version)
	if result == nil {
		w.Success("Could not check for updates")
		return nil
	}
	if !result.UpdateAvailable {
		w.Success(fmt.Sprintf("Already up to date: v%s", version))
		return nil
	}

	w.Info(fmt.Sprintf("Updating: v%s → %s", version, result.LatestVersion))
	if err := updater.SelfUpdate(result); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	w.Success(fmt.Sprintf("Updated to %s!", result.LatestVersion))
	w.Detail("Restart pdfify to use the new version")
	return nil
}

func runClean(w *ui.Writer) error {
	w.Info("Removing Docker image...")
	if err := docker.Remove(); err != nil {
		w.Detail("Image not found or already removed")
	} else {
		w.Success("Docker image removed")
	}
	return nil
}

func runInit(w *ui.Writer) error {
	path := config.PreferencesPath()

	// Load existing preferences (empty Config if file doesn't exist)
	existing, _ := config.LoadPreferences()

	content := config.GeneratePreferencesFile(existing)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing preferences file: %w", err)
	}

	if existing.Theme != "" || existing.PaperSize != "" || existing.Author != "" ||
		existing.TocLevel != 0 || existing.NumberSections != nil {
		w.Success(fmt.Sprintf("Preferences updated: %s", path))
		w.Detail("Existing values preserved, documentation refreshed")
	} else {
		w.Success(fmt.Sprintf("Preferences created: %s", path))
		w.Detail("Edit the file to customize your defaults")
	}
	return nil
}

func runHeaderify(files []string, w *ui.Writer) error {
	for _, inputPath := range files {
		content, err := os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", inputPath, err)
		}

		existing, _ := converter.ExtractFrontmatter(string(content))
		defaults := config.DefaultConfig()
		merged := config.Merge(defaults, existing)

		result := converter.InjectFrontmatter(string(content), merged)

		if err := os.WriteFile(inputPath, []byte(result), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", inputPath, err)
		}

		w.Success(fmt.Sprintf("Frontmatter updated: %s", inputPath))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Conversion
// ---------------------------------------------------------------------------

func convertFile(inputPath string, cfg *config.Config, w *ui.Writer) error {
	w.Blank()
	w.Info(fmt.Sprintf("Input:  %s", inputPath))

	result, err := converter.Convert(inputPath, cfg, func(step string, pct float64) {
		w.Progress(pct, step)
	})
	if err != nil {
		return err
	}

	w.Blank()
	w.Success(fmt.Sprintf("PDF created: %s (%s)", result.OutputPath, ui.FormatSize(result.FileSize)))

	if cfg.Open || cfg.Preview {
		openFile(result.OutputPath)
	}

	return nil
}

func runWatchMode(inputPath string, cfg *config.Config, w *ui.Writer) error {
	// Initial conversion
	if err := convertFile(inputPath, cfg, w); err != nil {
		return err
	}

	absInput, _ := filepath.Abs(inputPath)
	outputPath := strings.TrimSuffix(absInput, filepath.Ext(absInput)) + ".pdf"
	if cfg.Output != "" {
		outputPath, _ = filepath.Abs(cfg.Output)
	}

	srv := server.New(absInput, outputPath, "watch", func() error {
		return convertFile(inputPath, cfg, w)
	})
	url, err := srv.Start()
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	w.Blank()
	w.Success(fmt.Sprintf("Watching %s", filepath.Base(inputPath)))
	w.Info(fmt.Sprintf("Preview: %s", url))
	w.Detail("Press Ctrl+C to stop")
	w.Blank()

	openURL(url)

	lastHash := fileHash(absInput)
	for {
		time.Sleep(1 * time.Second)
		newHash := fileHash(absInput)
		if newHash != lastHash {
			lastHash = newHash
			w.Info("Change detected — rebuilding...")
			if err := convertFile(inputPath, cfg, w); err != nil {
				w.Error(err.Error())
			} else {
				srv.NotifyReload()
			}
		}
	}
}

func runEditMode(inputPath string, cfg *config.Config, w *ui.Writer) error {
	// Initial conversion
	if err := convertFile(inputPath, cfg, w); err != nil {
		return err
	}

	absInput, _ := filepath.Abs(inputPath)
	outputPath := strings.TrimSuffix(absInput, filepath.Ext(absInput)) + ".pdf"
	if cfg.Output != "" {
		outputPath, _ = filepath.Abs(cfg.Output)
	}

	srv := server.New(absInput, outputPath, "edit", func() error {
		result, err := converter.Convert(inputPath, cfg, nil)
		if err != nil {
			return err
		}
		w.Success(fmt.Sprintf("Rebuilt: %s", ui.FormatSize(result.FileSize)))
		return nil
	})
	url, err := srv.Start()
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	w.Blank()
	w.Success(fmt.Sprintf("Editing %s", filepath.Base(inputPath)))
	w.Info(fmt.Sprintf("Editor: %s", url))
	w.Detail("Press Ctrl+C to stop")
	w.Blank()

	openURL(url)

	// Block until Ctrl+C
	select {}
}

// ---------------------------------------------------------------------------
// Docker image management
// ---------------------------------------------------------------------------

func ensureDockerImage(w *ui.Writer, forceRebuild bool) error {
	if forceRebuild {
		w.Info("Removing existing Docker image...")
		docker.Remove()
	}

	status, detail, err := docker.Inspect(version)
	if err != nil {
		return fmt.Errorf("Docker is required but not available: %w", err)
	}

	switch status {
	case docker.StatusReady:
		w.Success(fmt.Sprintf("Docker image ready (%s)", detail))
		return nil

	case docker.StatusOutdated:
		w.Warn(fmt.Sprintf("Docker image outdated: %s", detail))
		w.Info("Rebuilding Docker image...")
		fallthrough

	case docker.StatusMissing:
		if status == docker.StatusMissing {
			w.Info("Building Docker image (first run, takes 2-3 minutes)...")
		}
		w.Detail("Installing: pandoc, XeLaTeX, mermaid-cli, Chromium, fonts")
		w.Blank()

		if err := docker.Build(func(line string) {
			if strings.HasPrefix(line, "Step ") || strings.Contains(line, "Successfully") {
				w.Detail(line)
			}
		}); err != nil {
			return fmt.Errorf("Docker build failed: %w", err)
		}
		w.Success("Docker image built")
		return nil
	}

	return nil
}

// ---------------------------------------------------------------------------
// Config from flags
// ---------------------------------------------------------------------------

func buildConfigFromFlags(cmd *cobra.Command) *config.Config {
	cfg := &config.Config{}

	cfg.Title, _ = cmd.Flags().GetString("title")
	cfg.Subtitle, _ = cmd.Flags().GetString("subtitle")
	cfg.Author, _ = cmd.Flags().GetString("author")
	cfg.Date, _ = cmd.Flags().GetString("date")
	cfg.Header, _ = cmd.Flags().GetString("header")
	cfg.Footer, _ = cmd.Flags().GetString("footer")
	cfg.Watermark, _ = cmd.Flags().GetString("watermark")
	cfg.PaperSize, _ = cmd.Flags().GetString("paper")
	cfg.Theme, _ = cmd.Flags().GetString("theme")

	if v, _ := cmd.Flags().GetInt("toc-level"); v > 0 {
		cfg.TocLevel = v
	}
	if v, _ := cmd.Flags().GetInt("number-from"); v > 0 {
		cfg.NumberFrom = v
	}
	if v, _ := cmd.Flags().GetBool("numbers"); v {
		t := true
		cfg.NumberSections = &t
	}
	if v, _ := cmd.Flags().GetBool("no-numbers"); v {
		f := false
		cfg.NumberSections = &f
	}
	if v, _ := cmd.Flags().GetBool("page-break"); v {
		t := true
		cfg.PageBreak = &t
	}
	if v, _ := cmd.Flags().GetBool("no-page-break"); v {
		f := false
		cfg.PageBreak = &f
	}

	cfg.Preview, _ = cmd.Flags().GetBool("preview")
	cfg.Open, _ = cmd.Flags().GetBool("open")
	cfg.Rebuild, _ = cmd.Flags().GetBool("rebuild")

	if cfg.Preview {
		cfg.Open = true
	}

	return cfg
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func openFile(path string) {
	var cmd *exec.Cmd
	switch {
	case commandExists("open"):
		cmd = exec.Command("open", path)
	case commandExists("xdg-open"):
		cmd = exec.Command("xdg-open", path)
	case commandExists("start"):
		cmd = exec.Command("cmd", "/c", "start", path)
	default:
		return
	}
	cmd.Start()
}

func openURL(url string) {
	openFile(url)
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func fileHash(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// exampleDocument is the example markdown shown by `pdfify --example`.
const exampleDocument = `---
title: "My Document"
subtitle: "A comprehensive example"
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
---

# My Document

This is an example document demonstrating all pdfify features.

## Text Formatting

This is **bold text**, *italic text*, and ` + "`inline code`" + `.

## Code Blocks

` + "```go" + `
func main() {
    fmt.Println("Hello, pdfify!")
}
` + "```" + `

## Tables

| Feature | Status | Notes |
|---------|--------|-------|
| Markdown | Supported | Full CommonMark |
| Mermaid | Supported | Rendered to PNG |
| Callouts | Supported | Obsidian-style |
| Themes | Supported | default, party |

## Mermaid Diagrams

` + "```mermaid" + `
graph TD
    A[Markdown] -->|pdfify| B[PDF]
    A --> C[Docker Container]
    C --> D[pandoc]
    C --> E[XeLaTeX]
    C --> F[mermaid-cli]
    D --> B
    E --> B
    F --> B
` + "```" + `

## Callouts

> [!info] Information
> This is an informational callout.

> [!tip] Pro Tip
> You can use callouts for important notes.

> [!warning] Be Careful
> This is a warning callout.

> [!danger] Critical
> This is a danger/error callout.

> [!example] Example
> This is an example callout.

> [!quote] Famous Quote
> The best way to predict the future is to invent it.

## Lists

- First item
- Second item
  - Nested item
  - Another nested item
- Third item

1. Numbered first
2. Numbered second
3. Numbered third

## Blockquotes

> This is a standard blockquote. It will be styled with a blue left border
> and light background.

## Images

Images referenced in markdown will be included in the PDF.
Use relative paths: ` + "`![Alt text](images/photo.png)`" + `

## Horizontal Rule

---

## Links

Visit [pdfify on GitHub](https://github.com/jclement/pdfify) for more info.
`
