// Package config handles the three-tier configuration system for pdfify.
// Settings cascade: CLI flags > frontmatter > preferences file (~/.config/pdfify/preferences.yaml).
// This package defines the unified Config struct and handles loading/merging from all sources.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all settings for a pdfify conversion run.
// Zero values mean "not set" — the merge logic uses this to layer sources.
type Config struct {
	// Document metadata
	Title    string `yaml:"title" json:"title"`
	Subtitle string `yaml:"subtitle" json:"subtitle"`
	Author   string `yaml:"author" json:"author"`
	Date     string `yaml:"date" json:"date"`

	// Page layout
	Header    string `yaml:"header" json:"header"`
	Footer    string `yaml:"footer" json:"footer"`
	Watermark string `yaml:"watermark" json:"watermark"`

	// TOC and numbering
	TocLevel       int    `yaml:"toc_level" json:"toc_level"`
	NumberSections *bool  `yaml:"number_sections" json:"number_sections"`
	NumberFrom     int    `yaml:"number_from" json:"number_from"`
	PageBreak      *bool  `yaml:"page_break" json:"page_break"`

	// Paper and output
	PaperSize string `yaml:"paper_size" json:"paper_size"`
	Theme     string `yaml:"theme" json:"theme"`
	Output    string `yaml:"output" json:"output"`

	// Behavior
	Watch   bool `yaml:"-" json:"-"`
	Open    bool `yaml:"-" json:"-"`
	Preview bool `yaml:"-" json:"-"`
	Rebuild bool `yaml:"-" json:"-"`
}

// PaperDimensions maps paper size names to LaTeX geometry strings.
var PaperDimensions = map[string]string{
	"letter":    "letterpaper",
	"a4":        "a4paper",
	"a5":        "a5paper",
	"legal":     "legalpaper",
	"executive": "executivepaper",
}

// ValidPaperSizes returns a sorted list of supported paper sizes.
func ValidPaperSizes() []string {
	return []string{"a4", "a5", "executive", "legal", "letter"}
}

// DefaultConfig returns sensible defaults for all settings.
func DefaultConfig() *Config {
	t := true
	return &Config{
		TocLevel:       3,
		NumberSections: &t,
		NumberFrom:     2,
		PageBreak:      &t,
		PaperSize:      "letter",
		Theme:          "default",
	}
}

// PreferencesPath returns the path to the user's preferences file.
func PreferencesPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(configDir, "pdfify", "preferences.yaml")
}

// LoadPreferences reads the preferences file if it exists.
// Returns an empty Config (not nil) if the file doesn't exist.
func LoadPreferences() (*Config, error) {
	path := PreferencesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading preferences %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing preferences %s: %w", path, err)
	}
	return &cfg, nil
}

// Merge creates a new Config by layering sources: base < overlay.
// Non-zero values in overlay win. This is used to implement the cascade:
// defaults < preferences < frontmatter < CLI flags.
func Merge(base, overlay *Config) *Config {
	result := *base

	if overlay.Title != "" {
		result.Title = overlay.Title
	}
	if overlay.Subtitle != "" {
		result.Subtitle = overlay.Subtitle
	}
	if overlay.Author != "" {
		result.Author = overlay.Author
	}
	if overlay.Date != "" {
		result.Date = overlay.Date
	}
	if overlay.Header != "" {
		result.Header = overlay.Header
	}
	if overlay.Footer != "" {
		result.Footer = overlay.Footer
	}
	if overlay.Watermark != "" {
		result.Watermark = overlay.Watermark
	}
	if overlay.TocLevel != 0 {
		result.TocLevel = overlay.TocLevel
	}
	if overlay.NumberSections != nil {
		result.NumberSections = overlay.NumberSections
	}
	if overlay.NumberFrom != 0 {
		result.NumberFrom = overlay.NumberFrom
	}
	if overlay.PageBreak != nil {
		result.PageBreak = overlay.PageBreak
	}
	if overlay.PaperSize != "" {
		result.PaperSize = overlay.PaperSize
	}
	if overlay.Theme != "" {
		result.Theme = overlay.Theme
	}
	if overlay.Output != "" {
		result.Output = overlay.Output
	}

	// Behavioral flags always come from overlay (CLI)
	if overlay.Watch {
		result.Watch = true
	}
	if overlay.Open {
		result.Open = true
	}
	if overlay.Preview {
		result.Preview = true
	}
	if overlay.Rebuild {
		result.Rebuild = true
	}

	return &result
}

// Validate checks that all config values are valid.
func (c *Config) Validate() error {
	if c.PaperSize != "" {
		lower := strings.ToLower(c.PaperSize)
		if _, ok := PaperDimensions[lower]; !ok {
			return fmt.Errorf("invalid paper size %q (valid: %s)", c.PaperSize, strings.Join(ValidPaperSizes(), ", "))
		}
		c.PaperSize = lower
	}
	if c.TocLevel < 0 || c.TocLevel > 4 {
		return fmt.Errorf("toc-level must be 0-4, got %d", c.TocLevel)
	}
	if c.NumberFrom < 1 || c.NumberFrom > 4 {
		return fmt.Errorf("number-from must be 1-4, got %d", c.NumberFrom)
	}
	if c.Theme != "" {
		validThemes := []string{"default", "party"}
		found := false
		for _, t := range validThemes {
			if c.Theme == t {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid theme %q (valid: %s)", c.Theme, strings.Join(validThemes, ", "))
		}
	}
	return nil
}

// IsNumbered returns whether section numbering is enabled.
func (c *Config) IsNumbered() bool {
	return c.NumberSections != nil && *c.NumberSections
}

// IsPageBreak returns whether page breaks before top-level sections are enabled.
func (c *Config) IsPageBreak() bool {
	return c.PageBreak != nil && *c.PageBreak
}

// GeneratePreferencesFile produces a documented YAML preferences template.
// If existing is non-nil, any non-zero values are emitted uncommented;
// all other settings are shown commented out with their defaults.
func GeneratePreferencesFile(existing *Config) string {
	if existing == nil {
		existing = &Config{}
	}
	defaults := DefaultConfig()

	var b strings.Builder

	b.WriteString(`# pdfify preferences
# https://github.com/jclement/pdfify
#
# These settings apply to all conversions unless overridden by
# document frontmatter or CLI flags.
#
# Priority: CLI flags > frontmatter > preferences > defaults
#
# Uncomment and modify any setting below.

`)

	// --- Document metadata ---
	b.WriteString(`# ─── Document metadata ─────────────────────────────────────────────
# Default metadata applied to every document. Most users leave these
# unset here and specify them per-document in YAML frontmatter.
#
`)
	writeStringPref(&b, "title", existing.Title, "", "Document title")
	writeStringPref(&b, "subtitle", existing.Subtitle, "", "Document subtitle")
	writeStringPref(&b, "author", existing.Author, "", "Author name")
	writeStringPref(&b, "date", existing.Date, "", `Date string, or "none" to suppress`)
	b.WriteByte('\n')

	// --- Page layout ---
	b.WriteString(`# ─── Page layout ──────────────────────────────────────────────────
`)
	writeStringPref(&b, "header", existing.Header, "", "Text centered at the top of every page")
	writeStringPref(&b, "footer", existing.Footer, "", "Text at the bottom-left of every page")
	writeStringPref(&b, "watermark", existing.Watermark, "", `Diagonal watermark (e.g. "DRAFT", "CONFIDENTIAL")`)
	b.WriteByte('\n')

	// --- Table of contents ---
	b.WriteString(`# ─── Table of contents ────────────────────────────────────────────
`)
	writeIntPref(&b, "toc_level", existing.TocLevel, defaults.TocLevel, "Heading depth for TOC (0 = disabled, 1–4)")
	b.WriteByte('\n')

	// --- Section numbering ---
	b.WriteString(`# ─── Section numbering ────────────────────────────────────────────
`)
	writeBoolPref(&b, "number_sections", existing.NumberSections, defaults.NumberSections, "Enable automatic section numbering")
	writeIntPref(&b, "number_from", existing.NumberFrom, defaults.NumberFrom, "Heading level to start numbering (1–4)")
	writeBoolPref(&b, "page_break", existing.PageBreak, defaults.PageBreak, "Insert a page break before each top-level section")
	b.WriteByte('\n')

	// --- Paper and theme ---
	b.WriteString(`# ─── Paper and theme ──────────────────────────────────────────────
`)
	writeStringPref(&b, "paper_size", existing.PaperSize, defaults.PaperSize,
		"Paper size: "+strings.Join(ValidPaperSizes(), ", "))
	writeStringPref(&b, "theme", existing.Theme, defaults.Theme, "Theme: default, party")

	return b.String()
}

// writeStringPref writes a string preference line.
// If the user has set a value, it's emitted uncommented; otherwise the default is commented out.
func writeStringPref(b *strings.Builder, key, userVal, defaultVal, comment string) {
	if userVal != "" {
		fmt.Fprintf(b, "%-18s %s", key+":", quote(userVal))
	} else if defaultVal != "" {
		fmt.Fprintf(b, "# %-16s %s", key+":", quote(defaultVal))
	} else {
		fmt.Fprintf(b, "# %-16s \"\"", key+":")
	}
	fmt.Fprintf(b, "  # %s\n", comment)
}

// writeIntPref writes an integer preference line.
func writeIntPref(b *strings.Builder, key string, userVal, defaultVal int, comment string) {
	if userVal != 0 {
		fmt.Fprintf(b, "%-18s %d", key+":", userVal)
	} else {
		fmt.Fprintf(b, "# %-16s %d", key+":", defaultVal)
	}
	fmt.Fprintf(b, "  # %s\n", comment)
}

// writeBoolPref writes a boolean pointer preference line.
func writeBoolPref(b *strings.Builder, key string, userVal, defaultVal *bool, comment string) {
	if userVal != nil {
		fmt.Fprintf(b, "%-18s %t", key+":", *userVal)
	} else if defaultVal != nil {
		fmt.Fprintf(b, "# %-16s %t", key+":", *defaultVal)
	} else {
		fmt.Fprintf(b, "# %-16s false", key+":")
	}
	fmt.Fprintf(b, "  # %s\n", comment)
}

// quote wraps a string in double quotes for YAML output.
func quote(s string) string {
	return fmt.Sprintf("%q", s)
}

// GeometryString returns the LaTeX geometry option string for this config.
func (c *Config) GeometryString() string {
	paper := PaperDimensions[c.PaperSize]
	if paper == "" {
		paper = "letterpaper"
	}
	return fmt.Sprintf("%s,margin=0.5in,includehead,includefoot", paper)
}
