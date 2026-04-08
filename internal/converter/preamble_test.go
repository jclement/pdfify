package converter

import (
	"strings"
	"testing"

	"github.com/jclement/pdfify/internal/config"
	"github.com/jclement/pdfify/internal/theme"
)

func TestGeneratePreamble_ContainsThemeColors(t *testing.T) {
	cfg := config.DefaultConfig()
	th := theme.DefaultTheme()

	preamble := GeneratePreamble(cfg, th)

	if !strings.Contains(preamble, th.Accent) {
		t.Error("preamble should contain accent color")
	}
	if !strings.Contains(preamble, th.InfoBar) {
		t.Error("preamble should contain info bar color")
	}
	if !strings.Contains(preamble, "\\usepackage{xcolor}") {
		t.Error("preamble should include xcolor package")
	}
}

func TestGeneratePreamble_PartyTheme(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Theme = "party"
	th := theme.PartyTheme()

	preamble := GeneratePreamble(cfg, th)

	if !strings.Contains(preamble, th.Accent) {
		t.Error("party preamble should contain party accent color")
	}
	if !strings.Contains(preamble, "party") {
		t.Error("preamble should reference party theme name")
	}
}

func TestGeneratePreamble_WithWatermark(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Watermark = "DRAFT"
	th := theme.DefaultTheme()

	preamble := GeneratePreamble(cfg, th)

	if !strings.Contains(preamble, "DRAFT") {
		t.Error("preamble should contain watermark text")
	}
	if !strings.Contains(preamble, "eso-pic") {
		t.Error("preamble should include eso-pic package for watermark")
	}
}

func TestGeneratePreamble_WithoutWatermark(t *testing.T) {
	cfg := config.DefaultConfig()
	th := theme.DefaultTheme()

	preamble := GeneratePreamble(cfg, th)

	if strings.Contains(preamble, "eso-pic") {
		t.Error("preamble should not include eso-pic when no watermark")
	}
}

func TestGeneratePreamble_WithHeaderFooter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Header = "CONFIDENTIAL"
	cfg.Footer = "Acme Corp"
	th := theme.DefaultTheme()

	preamble := GeneratePreamble(cfg, th)

	if !strings.Contains(preamble, "CONFIDENTIAL") {
		t.Error("preamble should contain header text")
	}
	if !strings.Contains(preamble, "Acme Corp") {
		t.Error("preamble should contain footer text")
	}
}

func TestGeneratePreamble_NumberedSections(t *testing.T) {
	cfg := config.DefaultConfig()
	th := theme.DefaultTheme()

	preamble := GeneratePreamble(cfg, th)

	if !strings.Contains(preamble, "secnumdepth") {
		t.Error("numbered sections preamble should set secnumdepth")
	}
}

func TestGeneratePreamble_UnnumberedSections(t *testing.T) {
	f := false
	cfg := config.DefaultConfig()
	cfg.NumberSections = &f
	th := theme.DefaultTheme()

	preamble := GeneratePreamble(cfg, th)

	if strings.Contains(preamble, "secnumdepth") {
		t.Error("unnumbered sections should not set secnumdepth")
	}
}

func TestGeneratePreamble_PaperSize(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.PaperSize = "a4"
	_ = theme.DefaultTheme()

	geo := cfg.GeometryString()
	if !strings.Contains(geo, "a4paper") {
		t.Errorf("geometry should contain a4paper, got %q", geo)
	}
}

func TestGenerateTitleBanner_WithTitle(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Title = "Test Document"
	cfg.Subtitle = "A Test"
	cfg.Author = "Test Author"
	cfg.Date = "2024-01-01"

	banner := GenerateTitleBanner(cfg)

	if !strings.Contains(banner, "Test Document") {
		t.Error("banner should contain title")
	}
	if !strings.Contains(banner, "A Test") {
		t.Error("banner should contain subtitle")
	}
	if !strings.Contains(banner, "Test Author") {
		t.Error("banner should contain author")
	}
	if !strings.Contains(banner, "maketitle") {
		t.Error("banner should define maketitle")
	}
}

func TestGenerateTitleBanner_NoTitle(t *testing.T) {
	cfg := &config.Config{}

	banner := GenerateTitleBanner(cfg)

	if !strings.Contains(banner, "\\renewcommand{\\maketitle}{}") {
		t.Error("no title should produce empty maketitle")
	}
}

func TestLatexEscape(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"hello & world", `\&`},
		{"100% done", `\%`},
		{"$price", `\$`},
		{"section #1", `\#`},
		{"under_score", `\_`},
		{"normal text", "normal text"},
	}

	for _, tt := range tests {
		result := latexEscape(tt.input)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("latexEscape(%q) should contain %q, got %q", tt.input, tt.contains, result)
		}
	}
}
