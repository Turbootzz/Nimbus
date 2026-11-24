package models

import (
	"strings"
	"testing"
)

func TestIsValidBoardSlug(t *testing.T) {
	valid := []string{"home", "media-room", "a", "wall-2", strings.Repeat("a", MaxBoardSlugLen)}
	invalid := []string{
		"", "Home", "media room", "media_room", "-media", "media-", "media--room", "a/b", "a?kiosk=1", "../etc", "möbel",
		strings.Repeat("a", MaxBoardSlugLen+1),
		// Reserved for pages under /board, and what a URL would read as an ID
		"new", "edit", "settings", "default", "shared", "all", "kiosk",
		"b0000000-0000-0000-0000-000000000001", "b0000000000000000000000000000001",
	}
	for _, slug := range valid {
		if !IsValidBoardSlug(slug) {
			t.Errorf("%q should be a valid slug", slug)
		}
	}
	for _, slug := range invalid {
		if IsValidBoardSlug(slug) {
			t.Errorf("%q should not be a valid slug", slug)
		}
	}
}

func TestBoardSlug(t *testing.T) {
	for name, want := range map[string]string{
		"Media Room":                     "media-room",
		"  Kids' stuff! ":                "kids-stuff",
		"Wall #2":                        "wall-2",
		"Ünïcödé":                        "n-c-d",
		"日本語":                            "board",
		"New":                            "board",
		"":                               "board",
		strings.Repeat("Long name ", 10): "long-name-long-name-long-name-long-n",
	} {
		got := BoardSlug(name)
		if got != want {
			t.Errorf("BoardSlug(%q) = %q, want %q", name, got, want)
		}
		if !IsValidBoardSlug(got) {
			t.Errorf("BoardSlug(%q) = %q is not a valid slug", name, got)
		}
	}
}
