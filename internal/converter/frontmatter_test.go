package converter

import (
	"strings"
	"testing"

	"github.com/jclement/pdfify/internal/config"
)

func TestExtractFrontmatter_WithFrontmatter(t *testing.T) {
	content := `---
title: "Test Document"
author: "Test Author"
toc-level: 2
numbersections: false
---
# Hello World

Body content here.
`

	cfg, body := ExtractFrontmatter(content)

	if cfg.Title != "Test Document" {
		t.Errorf("expected title 'Test Document', got %q", cfg.Title)
	}
	if cfg.Author != "Test Author" {
		t.Errorf("expected author 'Test Author', got %q", cfg.Author)
	}
	if cfg.TocLevel != 2 {
		t.Errorf("expected toc-level 2, got %d", cfg.TocLevel)
	}
	if cfg.NumberSections == nil || *cfg.NumberSections != false {
		t.Error("expected numbersections=false")
	}
	if !strings.HasPrefix(body, "# Hello World") {
		t.Errorf("body should start with '# Hello World', got %q", body[:30])
	}
}

func TestExtractFrontmatter_NoFrontmatter(t *testing.T) {
	content := "# Hello World\n\nBody content here.\n"

	cfg, body := ExtractFrontmatter(content)

	if cfg.Title != "" {
		t.Errorf("expected empty title, got %q", cfg.Title)
	}
	if body != content {
		t.Error("body should be unchanged when no frontmatter")
	}
}

func TestExtractFrontmatter_EmptyFrontmatter(t *testing.T) {
	content := "---\n---\n# Hello\n"

	cfg, body := ExtractFrontmatter(content)

	if cfg.Title != "" {
		t.Errorf("expected empty title, got %q", cfg.Title)
	}
	if !strings.HasPrefix(body, "# Hello") {
		t.Errorf("body should start with '# Hello', got %q", body)
	}
}

func TestExtractFrontmatter_AllFields(t *testing.T) {
	content := `---
title: "Full Test"
subtitle: "All Fields"
author: "Author"
date: "2024-01-01"
header: "Header Text"
footer: "Footer Text"
watermark: "DRAFT"
toc-level: 3
numbersections: true
numberfrom: 1
pagebreak: false
papersize: a4
theme: party
---
Body
`

	cfg, _ := ExtractFrontmatter(content)

	if cfg.Title != "Full Test" {
		t.Errorf("title = %q", cfg.Title)
	}
	if cfg.Subtitle != "All Fields" {
		t.Errorf("subtitle = %q", cfg.Subtitle)
	}
	if cfg.PaperSize != "a4" {
		t.Errorf("papersize = %q", cfg.PaperSize)
	}
	if cfg.Theme != "party" {
		t.Errorf("theme = %q", cfg.Theme)
	}
	if cfg.Watermark != "DRAFT" {
		t.Errorf("watermark = %q", cfg.Watermark)
	}
	if cfg.PageBreak == nil || *cfg.PageBreak != false {
		t.Error("expected pagebreak=false")
	}
	if cfg.NumberFrom != 1 {
		t.Errorf("numberfrom = %d", cfg.NumberFrom)
	}
}

func TestExtractFrontmatter_MalformedYAML(t *testing.T) {
	content := "---\ntitle: [invalid yaml\n---\nBody\n"

	cfg, body := ExtractFrontmatter(content)

	if cfg.Title != "" {
		t.Error("malformed YAML should return empty config")
	}
	if body != content {
		t.Error("malformed YAML should return original content")
	}
}

func TestExtractFrontmatter_NoClosingDelimiter(t *testing.T) {
	content := "---\ntitle: Test\nBody without closing\n"

	cfg, body := ExtractFrontmatter(content)

	if cfg.Title != "" {
		t.Error("missing closing --- should return empty config")
	}
	if body != content {
		t.Error("missing closing --- should return original content")
	}
}

func TestInjectFrontmatter_NewFile(t *testing.T) {
	content := "# Hello\n\nSome content.\n"
	trueVal := true
	cfg := &config.Config{
		Title:          "Injected Title",
		Author:         "Test Author",
		TocLevel:       3,
		NumberSections: &trueVal,
	}

	result := InjectFrontmatter(content, cfg)

	if !strings.HasPrefix(result, "---\n") {
		t.Error("result should start with frontmatter")
	}
	if !strings.Contains(result, `title: "Injected Title"`) {
		t.Error("result should contain title")
	}
	if !strings.Contains(result, `author: "Test Author"`) {
		t.Error("result should contain author")
	}
	if !strings.Contains(result, "# Hello") {
		t.Error("result should contain original body")
	}
}

func TestInjectFrontmatter_ReplacesExisting(t *testing.T) {
	content := "---\ntitle: Old Title\n---\n# Hello\n"
	cfg := &config.Config{
		Title: "New Title",
	}

	result := InjectFrontmatter(content, cfg)

	if strings.Contains(result, "Old Title") {
		t.Error("should not contain old title")
	}
	if !strings.Contains(result, "New Title") {
		t.Error("should contain new title")
	}
	if !strings.Contains(result, "# Hello") {
		t.Error("should contain body")
	}
}

func TestAnalyzeDocument(t *testing.T) {
	content := `# Main Title

## Section One

Some text with a table:

| Col1 | Col2 |
|------|------|
| a    | b    |

> [!info] Note
> This is an info callout.

> [!warning] Watch Out
> Be careful!

` + "```go" + `
func main() {}
` + "```" + `

` + "```mermaid" + `
graph TD
    A --> B
` + "```" + `

## Section Two

![Image](test.png)
`

	info := AnalyzeDocument(content)

	if info.H1Count != 1 {
		t.Errorf("expected 1 H1, got %d", info.H1Count)
	}
	if info.FirstH1Text != "Main Title" {
		t.Errorf("expected FirstH1Text='Main Title', got %q", info.FirstH1Text)
	}
	if info.CalloutCount != 2 {
		t.Errorf("expected 2 callouts, got %d", info.CalloutCount)
	}
	if info.MermaidCount != 1 {
		t.Errorf("expected 1 mermaid, got %d", info.MermaidCount)
	}
	if info.ImageCount != 1 {
		t.Errorf("expected 1 image, got %d", info.ImageCount)
	}
	if info.TableRows < 2 {
		t.Errorf("expected >= 2 table rows, got %d", info.TableRows)
	}
}

func TestAnalyzeDocument_MultipleH1(t *testing.T) {
	content := "# First\n\n# Second\n\n# Third\n"

	info := AnalyzeDocument(content)

	if info.H1Count != 3 {
		t.Errorf("expected 3 H1s, got %d", info.H1Count)
	}
	if info.FirstH1Text != "First" {
		t.Errorf("expected FirstH1Text='First', got %q", info.FirstH1Text)
	}
}

func TestAnalyzeDocument_NoContent(t *testing.T) {
	info := AnalyzeDocument("")
	if info.H1Count != 0 || info.CalloutCount != 0 || info.MermaidCount != 0 {
		t.Error("empty document should have zero counts")
	}
}

func TestAnalyzeDocument_HeadingsInCodeBlocks(t *testing.T) {
	content := "```\n# Not a heading\n```\n# Real heading\n"

	info := AnalyzeDocument(content)

	if info.H1Count != 1 {
		t.Errorf("expected 1 H1 (not counting code block), got %d", info.H1Count)
	}
}
