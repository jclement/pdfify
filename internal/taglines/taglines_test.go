package taglines

import (
	"testing"
)

func TestAll_HasEntries(t *testing.T) {
	if len(All) < 50 {
		t.Errorf("expected at least 50 taglines, got %d", len(All))
	}
}

func TestRandom_ReturnsNonEmpty(t *testing.T) {
	for i := 0; i < 20; i++ {
		tag := Random()
		if tag == "" {
			t.Error("Random() returned empty string")
		}
	}
}

func TestAll_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, tag := range All {
		if seen[tag] {
			t.Errorf("duplicate tagline: %q", tag)
		}
		seen[tag] = true
	}
}
