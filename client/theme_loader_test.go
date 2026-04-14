package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestGetCustomThemeNamesSorted(t *testing.T) {
	saved := customThemes
	t.Cleanup(func() { customThemes = saved })

	customThemes = ThemeFile{
		"zebra": {Name: "Z"},
		"alpha": {Name: "A"},
		"moon":  {Name: "M"},
	}
	got := GetCustomThemeNames()
	want := []string{"alpha", "moon", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetCustomThemeNames() = %v, want %v", got, want)
	}

	all := ListAllThemes()
	wantAll := []string{"system", "patriot", "retro", "modern", "alpha", "moon", "zebra"}
	if !reflect.DeepEqual(all, wantAll) {
		t.Fatalf("ListAllThemes() = %v, want %v", all, wantAll)
	}
}

func TestCustomThemeBannerStripsFooterFallback(t *testing.T) {
	c := ThemeColors{
		FooterBg: "#181C24",
		FooterFg: "#4F8EF7",
	}
	_, _, info := customThemeBannerStrips(c)
	out := info.Render("status")
	if !strings.Contains(out, "status") {
		t.Fatalf("expected render to contain text, got %q", out)
	}
}

func TestCustomThemeBannerStripsExplicitColors(t *testing.T) {
	c := ThemeColors{
		FooterBg:      "#181C24",
		FooterFg:      "#4F8EF7",
		BannerErrorBg: "#010101",
		BannerErrorFg: "#FEFEFE",
	}
	errS, _, _ := customThemeBannerStrips(c)
	out := errS.Render("e")
	if !strings.Contains(out, "e") {
		t.Fatalf("expected render to contain text, got %q", out)
	}
}
