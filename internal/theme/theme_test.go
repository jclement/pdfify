package theme

import (
	"strings"
	"testing"
)

func TestGet_Default(t *testing.T) {
	th := Get("default")
	if th == nil {
		t.Fatal("Get('default') returned nil")
	}
	if th.Name != "default" {
		t.Errorf("expected name 'default', got %q", th.Name)
	}
}

func TestGet_Party(t *testing.T) {
	th := Get("party")
	if th == nil {
		t.Fatal("Get('party') returned nil")
	}
	if th.Name != "party" {
		t.Errorf("expected name 'party', got %q", th.Name)
	}
}

func TestGet_Empty(t *testing.T) {
	th := Get("")
	if th == nil {
		t.Fatal("Get('') should return default theme")
	}
	if th.Name != "default" {
		t.Errorf("empty name should return default, got %q", th.Name)
	}
}

func TestGet_Unknown(t *testing.T) {
	th := Get("nonexistent")
	if th != nil {
		t.Error("Get('nonexistent') should return nil")
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) < 2 {
		t.Errorf("expected at least 2 themes, got %d", len(names))
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["default"] {
		t.Error("missing 'default' theme")
	}
	if !found["party"] {
		t.Error("missing 'party' theme")
	}
}

func TestDefaultTheme_HasAllColors(t *testing.T) {
	th := DefaultTheme()

	// Verify critical color fields are non-empty
	colors := []struct {
		name  string
		value string
	}{
		{"Accent", th.Accent},
		{"AccentDark", th.AccentDark},
		{"CodeBg", th.CodeBg},
		{"InfoBg", th.InfoBg},
		{"InfoBar", th.InfoBar},
		{"TipBg", th.TipBg},
		{"WarningBar", th.WarningBar},
		{"DangerBar", th.DangerBar},
		{"ExampleBar", th.ExampleBar},
		{"QuoteBar", th.QuoteBar},
	}

	for _, c := range colors {
		if c.value == "" {
			t.Errorf("default theme missing color: %s", c.name)
		}
		if len(c.value) != 6 {
			t.Errorf("color %s should be 6 hex chars, got %q", c.name, c.value)
		}
	}
}

func TestPartyTheme_DiffersFromDefault(t *testing.T) {
	def := DefaultTheme()
	party := PartyTheme()

	if def.Accent == party.Accent {
		t.Error("party theme should have different accent color than default")
	}
	if def.TitleBg == party.TitleBg {
		t.Error("party theme should have different title background than default")
	}
}

func TestMermaidConfigJSON(t *testing.T) {
	th := DefaultTheme()
	json := th.MermaidConfigJSON()

	if !strings.Contains(json, `"theme"`) {
		t.Error("mermaid config should contain theme key")
	}
	if !strings.Contains(json, th.MermaidPrimaryColor) {
		t.Error("mermaid config should contain primary color")
	}
	if !strings.HasPrefix(json, "{") || !strings.HasSuffix(json, "}") {
		t.Error("mermaid config should be valid JSON object")
	}
}

func TestTheme_HasFonts(t *testing.T) {
	for _, name := range Names() {
		th := Get(name)
		if th.MainFont == "" {
			t.Errorf("theme %q missing MainFont", name)
		}
		if th.MonoFont == "" {
			t.Errorf("theme %q missing MonoFont", name)
		}
		if th.HeadingFont == "" {
			t.Errorf("theme %q missing HeadingFont", name)
		}
	}
}
