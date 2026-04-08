// Package ui provides terminal output formatting for pdfify.
// Handles progress bars, status messages, and styled output.
// Respects TTY detection — when piped, falls back to plain text.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// Colors for terminal output.
const (
	Red     = "\033[0;31m"
	Green   = "\033[0;32m"
	Yellow  = "\033[0;33m"
	Blue    = "\033[0;34m"
	Magenta = "\033[0;35m"
	Cyan    = "\033[0;36m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Reset   = "\033[0m"
)

// Writer handles formatted output to a writer (usually os.Stderr).
type Writer struct {
	out   io.Writer
	isTTY bool
}

// NewWriter creates a Writer that auto-detects TTY for the given file.
func NewWriter(f *os.File) *Writer {
	return &Writer{
		out:   f,
		isTTY: term.IsTerminal(int(f.Fd())),
	}
}

// Stderr returns a Writer connected to os.Stderr.
func Stderr() *Writer {
	return NewWriter(os.Stderr)
}

// Info prints an informational message.
func (w *Writer) Info(msg string) {
	if w.isTTY {
		fmt.Fprintf(w.out, "%s::%s %s%s%s\n", Blue, Reset, Bold, msg, Reset)
	} else {
		fmt.Fprintf(w.out, ":: %s\n", msg)
	}
}

// Success prints a success message.
func (w *Writer) Success(msg string) {
	if w.isTTY {
		fmt.Fprintf(w.out, "%s✓%s %s\n", Green, Reset, msg)
	} else {
		fmt.Fprintf(w.out, "[OK] %s\n", msg)
	}
}

// Warn prints a warning message.
func (w *Writer) Warn(msg string) {
	if w.isTTY {
		fmt.Fprintf(w.out, "%s⚠%s %s\n", Yellow, Reset, msg)
	} else {
		fmt.Fprintf(w.out, "[WARN] %s\n", msg)
	}
}

// Error prints an error message.
func (w *Writer) Error(msg string) {
	if w.isTTY {
		fmt.Fprintf(w.out, "%s✗%s %s\n", Red, Reset, msg)
	} else {
		fmt.Fprintf(w.out, "[ERROR] %s\n", msg)
	}
}

// Detail prints an indented detail line.
func (w *Writer) Detail(msg string) {
	if w.isTTY {
		fmt.Fprintf(w.out, "  %s→%s %s\n", Dim, Reset, msg)
	} else {
		fmt.Fprintf(w.out, "  → %s\n", msg)
	}
}

// Header prints a section header with decorative lines.
func (w *Writer) Header(msg string) {
	if w.isTTY {
		line := Magenta + strings.Repeat("━", 50) + Reset
		fmt.Fprintf(w.out, "\n%s\n%s  %s%s%s\n%s\n\n", line, Magenta, Bold, msg, Reset, line)
	} else {
		fmt.Fprintf(w.out, "\n=== %s ===\n\n", msg)
	}
}

// Progress renders a progress bar with status text.
// pct is 0.0 to 1.0.
func (w *Writer) Progress(pct float64, status string) {
	if !w.isTTY {
		fmt.Fprintf(w.out, "[%3.0f%%] %s\n", pct*100, status)
		return
	}

	width := 30
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := Green + strings.Repeat("█", filled) + Dim + strings.Repeat("░", empty) + Reset

	// Clear line and write progress
	fmt.Fprintf(w.out, "\r%s[%s]%s %s%-40s%s", Dim, bar, Reset, Cyan, status, Reset)

	if pct >= 1.0 {
		fmt.Fprintln(w.out)
	}
}

// Table prints a simple key-value table.
func (w *Writer) Table(rows [][]string) {
	maxKey := 0
	for _, row := range rows {
		if len(row) > 0 && len(row[0]) > maxKey {
			maxKey = len(row[0])
		}
	}

	for _, row := range rows {
		if len(row) >= 2 {
			if w.isTTY {
				fmt.Fprintf(w.out, "  %s%-*s%s  %s%s%s\n", Dim, maxKey, row[0], Reset, Cyan, row[1], Reset)
			} else {
				fmt.Fprintf(w.out, "  %-*s  %s\n", maxKey, row[0], row[1])
			}
		}
	}
}

// Blank prints an empty line.
func (w *Writer) Blank() {
	fmt.Fprintln(w.out)
}

// IsTTY returns whether the writer is connected to a terminal.
func (w *Writer) IsTTY() bool {
	return w.isTTY
}

// StatusFlair returns a random sarcastic/fun status word to spice up messages.
var statusFlair = []string{
	"Summoning LaTeX demons",
	"Persuading pandoc",
	"Bribing the typesetter",
	"Herding mermaid diagrams",
	"Arguing with margins",
	"Convincing fonts to cooperate",
	"Wrangling Docker containers",
	"Whispering to XeLaTeX",
	"Compiling your hopes and dreams",
	"Making pixels presentable",
	"Teaching tables manners",
	"Negotiating page breaks",
	"Taming the callout beasts",
	"Optimizing whitespace",
	"Polishing headings to a shine",
	"Consulting the typographic oracle",
	"Aligning the stars (and margins)",
	"Converting caffeine to PDFs",
	"Performing document alchemy",
	"Feeding the rendering hamsters",
}

// RandomFlair returns a random sarcastic status message.
func RandomFlair() string {
	return statusFlair[time.Now().UnixNano()%int64(len(statusFlair))]
}

// FormatSize formats a byte count as a human-readable string.
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
