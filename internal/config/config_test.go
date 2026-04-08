package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.TocLevel != 3 {
		t.Errorf("expected TocLevel=3, got %d", cfg.TocLevel)
	}
	if cfg.PaperSize != "letter" {
		t.Errorf("expected PaperSize=letter, got %s", cfg.PaperSize)
	}
	if cfg.Theme != "default" {
		t.Errorf("expected Theme=default, got %s", cfg.Theme)
	}
	if !cfg.IsNumbered() {
		t.Error("expected NumberSections=true by default")
	}
	if !cfg.IsPageBreak() {
		t.Error("expected PageBreak=true by default")
	}
	if cfg.NumberFrom != 2 {
		t.Errorf("expected NumberFrom=2, got %d", cfg.NumberFrom)
	}
}

func TestMerge(t *testing.T) {
	base := DefaultConfig()
	overlay := &Config{
		Title:     "Test Title",
		PaperSize: "a4",
		Theme:     "party",
	}

	result := Merge(base, overlay)

	if result.Title != "Test Title" {
		t.Errorf("expected Title='Test Title', got %q", result.Title)
	}
	if result.PaperSize != "a4" {
		t.Errorf("expected PaperSize='a4', got %q", result.PaperSize)
	}
	if result.Theme != "party" {
		t.Errorf("expected Theme='party', got %q", result.Theme)
	}
	// Unset fields should keep base values
	if result.TocLevel != 3 {
		t.Errorf("expected TocLevel=3 (from base), got %d", result.TocLevel)
	}
}

func TestMerge_BoolOverride(t *testing.T) {
	base := DefaultConfig()
	f := false
	overlay := &Config{
		NumberSections: &f,
	}

	result := Merge(base, overlay)
	if result.IsNumbered() {
		t.Error("expected NumberSections to be overridden to false")
	}
}

func TestMerge_EmptyOverlay(t *testing.T) {
	base := DefaultConfig()
	overlay := &Config{}

	result := Merge(base, overlay)
	if result.Title != base.Title {
		t.Error("empty overlay should not change base values")
	}
	if result.PaperSize != base.PaperSize {
		t.Error("empty overlay should not change base PaperSize")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should not error: %v", err)
	}
}

func TestValidate_InvalidPaperSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PaperSize = "tabloid"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid paper size")
	}
}

func TestValidate_InvalidTheme(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Theme = "neon"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid theme")
	}
}

func TestValidate_InvalidTocLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TocLevel = 5
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for toc-level > 4")
	}
}

func TestValidate_InvalidNumberFrom(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NumberFrom = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for number-from < 1")
	}
}

func TestValidate_PaperSizeNormalization(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PaperSize = "A4"
	if err := cfg.Validate(); err != nil {
		t.Errorf("A4 should be valid (case-insensitive): %v", err)
	}
	if cfg.PaperSize != "a4" {
		t.Errorf("expected normalized paper size 'a4', got %q", cfg.PaperSize)
	}
}

func TestGeometryString(t *testing.T) {
	tests := []struct {
		paper    string
		expected string
	}{
		{"letter", "letterpaper,margin=0.5in,includehead,includefoot"},
		{"a4", "a4paper,margin=0.5in,includehead,includefoot"},
		{"legal", "legalpaper,margin=0.5in,includehead,includefoot"},
	}

	for _, tt := range tests {
		cfg := DefaultConfig()
		cfg.PaperSize = tt.paper
		got := cfg.GeometryString()
		if got != tt.expected {
			t.Errorf("GeometryString(%q) = %q, want %q", tt.paper, got, tt.expected)
		}
	}
}

func TestValidPaperSizes(t *testing.T) {
	sizes := ValidPaperSizes()
	if len(sizes) == 0 {
		t.Error("expected at least one valid paper size")
	}
	// Check sorted
	for i := 1; i < len(sizes); i++ {
		if sizes[i] < sizes[i-1] {
			t.Errorf("paper sizes not sorted: %s < %s", sizes[i], sizes[i-1])
		}
	}
}
