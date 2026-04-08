// Package converter implements the markdown-to-PDF conversion pipeline.
// This file handles YAML frontmatter extraction from markdown files.
package converter

import (
	"fmt"
	"strings"

	"github.com/jclement/pdfify/internal/config"
	"gopkg.in/yaml.v3"
)

// FrontmatterRaw mirrors the YAML keys used in markdown frontmatter.
// Keys use the original bash script's naming convention for compatibility.
type FrontmatterRaw struct {
	Title          string `yaml:"title"`
	Subtitle       string `yaml:"subtitle"`
	Author         string `yaml:"author"`
	Date           string `yaml:"date"`
	Header         string `yaml:"header"`
	Footer         string `yaml:"footer"`
	TocLevel       *int   `yaml:"toc-level"`
	NumberSections *bool  `yaml:"numbersections"`
	NumberFrom     *int   `yaml:"numberfrom"`
	Watermark      string `yaml:"watermark"`
	PageBreak      *bool  `yaml:"pagebreak"`
	PaperSize      string `yaml:"papersize"`
	Theme          string `yaml:"theme"`
}

// ExtractFrontmatter parses YAML frontmatter from markdown content.
// Returns the frontmatter as a Config overlay and the body (without frontmatter).
func ExtractFrontmatter(content string) (*config.Config, string) {
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return &config.Config{}, content
	}

	// Find closing ---
	rest := content[4:] // skip opening ---\n
	if strings.HasPrefix(content, "---\r\n") {
		rest = content[5:]
	}

	// Handle empty frontmatter (closing --- immediately follows opening)
	var fmBlock, body string
	if strings.HasPrefix(rest, "---\n") || strings.HasPrefix(rest, "---\r\n") {
		fmBlock = ""
		if strings.HasPrefix(rest, "---\r\n") {
			body = rest[5:]
		} else {
			body = rest[4:]
		}
	} else {
		endIdx := strings.Index(rest, "\n---")
		if endIdx < 0 {
			return &config.Config{}, content
		}
		fmBlock = rest[:endIdx]
		body = rest[endIdx+4:] // skip \n---
	}
	// Skip trailing newline after closing ---
	if strings.HasPrefix(body, "\n") {
		body = body[1:]
	} else if strings.HasPrefix(body, "\r\n") {
		body = body[2:]
	}

	var raw FrontmatterRaw
	if err := yaml.Unmarshal([]byte(fmBlock), &raw); err != nil {
		// Malformed frontmatter — return content as-is
		return &config.Config{}, content
	}

	cfg := &config.Config{
		Title:     raw.Title,
		Subtitle:  raw.Subtitle,
		Author:    raw.Author,
		Date:      raw.Date,
		Header:    raw.Header,
		Footer:    raw.Footer,
		Watermark: raw.Watermark,
		PaperSize: raw.PaperSize,
		Theme:     raw.Theme,
	}

	if raw.TocLevel != nil {
		cfg.TocLevel = *raw.TocLevel
	}
	if raw.NumberSections != nil {
		cfg.NumberSections = raw.NumberSections
	}
	if raw.NumberFrom != nil {
		cfg.NumberFrom = *raw.NumberFrom
	}
	if raw.PageBreak != nil {
		cfg.PageBreak = raw.PageBreak
	}

	return cfg, body
}

// InjectFrontmatter adds or updates frontmatter in a markdown file.
// If frontmatter exists, it merges with existing values. If not, prepends a new block.
func InjectFrontmatter(content string, cfg *config.Config) string {
	_, body := ExtractFrontmatter(content)

	var fm strings.Builder
	fm.WriteString("---\n")
	if cfg.Title != "" {
		fm.WriteString(fmt.Sprintf("title: %q\n", cfg.Title))
	}
	if cfg.Subtitle != "" {
		fm.WriteString(fmt.Sprintf("subtitle: %q\n", cfg.Subtitle))
	}
	if cfg.Author != "" {
		fm.WriteString(fmt.Sprintf("author: %q\n", cfg.Author))
	}
	if cfg.Date != "" {
		fm.WriteString(fmt.Sprintf("date: %q\n", cfg.Date))
	}
	if cfg.Header != "" {
		fm.WriteString(fmt.Sprintf("header: %q\n", cfg.Header))
	}
	if cfg.Footer != "" {
		fm.WriteString(fmt.Sprintf("footer: %q\n", cfg.Footer))
	}
	if cfg.TocLevel != 0 {
		fm.WriteString(fmt.Sprintf("toc-level: %d\n", cfg.TocLevel))
	}
	if cfg.NumberSections != nil {
		fm.WriteString(fmt.Sprintf("numbersections: %t\n", *cfg.NumberSections))
	}
	if cfg.NumberFrom != 0 {
		fm.WriteString(fmt.Sprintf("numberfrom: %d\n", cfg.NumberFrom))
	}
	if cfg.Watermark != "" {
		fm.WriteString(fmt.Sprintf("watermark: %q\n", cfg.Watermark))
	}
	if cfg.PageBreak != nil {
		fm.WriteString(fmt.Sprintf("pagebreak: %t\n", *cfg.PageBreak))
	}
	if cfg.PaperSize != "" {
		fm.WriteString(fmt.Sprintf("papersize: %s\n", cfg.PaperSize))
	}
	if cfg.Theme != "" {
		fm.WriteString(fmt.Sprintf("theme: %s\n", cfg.Theme))
	}
	fm.WriteString("---\n")

	return fm.String() + body
}
