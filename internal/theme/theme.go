// Package theme defines the visual theming system for pdfify PDFs.
// Themes control colors, fonts, heading styles, callout colors, and mermaid chart
// appearance. Two built-in themes are provided: "default" (clean professional) and
// "party" (colorful and bold).
package theme

import "fmt"

// Theme holds all visual parameters for PDF generation.
type Theme struct {
	Name        string
	Description string

	// Colors (HTML hex without #)
	Accent        string
	AccentDark    string
	CodeBg        string
	CodeBorder    string
	HeadRuleColor string
	TitleBg       string
	TableRowGray  string

	// Callout colors: bg, bar, fg for each type
	InfoBg, InfoBar, InfoFg          string
	TipBg, TipBar, TipFg             string
	WarningBg, WarningBar, WarningFg string
	DangerBg, DangerBar, DangerFg    string
	ExampleBg, ExampleBar, ExampleFg string
	QuoteBg, QuoteBar, QuoteFg       string

	// Fonts
	MainFont    string
	MonoFont    string
	HeadingFont string

	// Heading style
	H1LetterSpace int
	H1SmallCaps   bool

	// Link color (same as accent by default)
	LinkColor string

	// Mermaid config overrides (JSON partial)
	MermaidTheme         string
	MermaidPrimaryColor  string
	MermaidPrimaryBorder string
	MermaidPrimaryText   string
	MermaidLineColor     string
	MermaidPalette       string
}

// MermaidConfigJSON returns a mermaid-cli compatible JSON config string for this theme.
func (t *Theme) MermaidConfigJSON() string {
	return fmt.Sprintf(`{"maxTextSize": 90000, "flowchart": {"useMaxWidth": true}, "theme": "%s", "themeVariables": {"primaryColor": "%s", "primaryBorderColor": "%s", "primaryTextColor": "%s", "lineColor": "%s", "xyChart": {"backgroundColor": "transparent", "plotColorPalette": "%s"}}}`,
		t.MermaidTheme,
		t.MermaidPrimaryColor,
		t.MermaidPrimaryBorder,
		t.MermaidPrimaryText,
		t.MermaidLineColor,
		t.MermaidPalette,
	)
}

// Get returns a theme by name. Returns nil if not found.
func Get(name string) *Theme {
	switch name {
	case "default", "":
		return DefaultTheme()
	case "party":
		return PartyTheme()
	default:
		return nil
	}
}

// Names returns all available theme names.
func Names() []string {
	return []string{"default", "party"}
}

// DefaultTheme returns the clean, professional default theme.
// Roboto fonts, muted grays and blues, suitable for business documents.
func DefaultTheme() *Theme {
	return &Theme{
		Name:        "default",
		Description: "Clean and professional — Roboto, muted grays and blues",

		Accent:        "374151",
		AccentDark:    "111827",
		CodeBg:        "F8F9FA",
		CodeBorder:    "E2E8F0",
		HeadRuleColor: "E2E8F0",
		TitleBg:       "E5E7EB",
		TableRowGray:  "F3F4F6",

		InfoBg: "EFF6FF", InfoBar: "3B82F6", InfoFg: "1E40AF",
		TipBg: "F0FDF4", TipBar: "22C55E", TipFg: "166534",
		WarningBg: "FFFBEB", WarningBar: "F59E0B", WarningFg: "92400E",
		DangerBg: "FEF2F2", DangerBar: "EF4444", DangerFg: "991B1B",
		ExampleBg: "F5F3FF", ExampleBar: "8B5CF6", ExampleFg: "5B21B6",
		QuoteBg: "F8F9FA", QuoteBar: "6B7280", QuoteFg: "374151",

		MainFont:    "Roboto",
		MonoFont:    "Roboto Mono",
		HeadingFont: "Roboto",

		H1LetterSpace: 5,
		H1SmallCaps:   true,
		LinkColor:     "374151",

		MermaidTheme:         "default",
		MermaidPrimaryColor:  "#3B82F6",
		MermaidPrimaryBorder: "#1E40AF",
		MermaidPrimaryText:   "#1E293B",
		MermaidLineColor:     "#475569",
		MermaidPalette:       "#2563EB,#DC2626,#16A34A,#D97706,#9333EA,#0891B2",
	}
}

// PartyTheme returns a colorful, bold theme with playful aesthetics.
// Bright gradients, fun fonts, high saturation for headings and callouts.
func PartyTheme() *Theme {
	return &Theme{
		Name:        "party",
		Description: "Bold and colorful — vibrant headings, playful callouts, maximum fun",

		Accent:        "7C3AED",
		AccentDark:    "4C1D95",
		CodeBg:        "FFF7ED",
		CodeBorder:    "FDBA74",
		HeadRuleColor: "A78BFA",
		TitleBg:       "EDE9FE",
		TableRowGray:  "FDF4FF",

		InfoBg: "DBEAFE", InfoBar: "2563EB", InfoFg: "1E40AF",
		TipBg: "D1FAE5", TipBar: "059669", TipFg: "065F46",
		WarningBg: "FEF3C7", WarningBar: "D97706", WarningFg: "92400E",
		DangerBg: "FEE2E2", DangerBar: "DC2626", DangerFg: "991B1B",
		ExampleBg: "EDE9FE", ExampleBar: "7C3AED", ExampleFg: "5B21B6",
		QuoteBg: "FCE7F3", QuoteBar: "EC4899", QuoteFg: "9D174D",

		MainFont:    "Roboto",
		MonoFont:    "Roboto Mono",
		HeadingFont: "Roboto",

		H1LetterSpace: 3,
		H1SmallCaps:   false,
		LinkColor:     "7C3AED",

		MermaidTheme:         "base",
		MermaidPrimaryColor:  "#7C3AED",
		MermaidPrimaryBorder: "#4C1D95",
		MermaidPrimaryText:   "#1E293B",
		MermaidLineColor:     "#A78BFA",
		MermaidPalette:       "#7C3AED,#EC4899,#F59E0B,#10B981,#3B82F6,#EF4444",
	}
}
