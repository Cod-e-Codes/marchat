package main

import (
	"reflect"
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
