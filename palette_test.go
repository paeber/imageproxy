// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"image/color"
	"testing"
)

func TestLookupPalette(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
		colors  int
	}{
		{"E1002", false, 6},
		{"e1002", false, 6},
		{"BW", false, 2},
		{"unknown", true, 0},
	}

	for _, tt := range tests {
		p, err := LookupPalette(tt.id)
		if tt.wantErr {
			if err == nil {
				t.Errorf("LookupPalette(%q) expected error", tt.id)
			}
			continue
		}
		if err != nil {
			t.Errorf("LookupPalette(%q) returned error: %v", tt.id, err)
			continue
		}
		if len(p.Colors) != tt.colors {
			t.Errorf("LookupPalette(%q) returned %d colors, want %d", tt.id, len(p.Colors), tt.colors)
		}
	}
}

func TestNearestColor(t *testing.T) {
	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		in   color.NRGBA
		want color.NRGBA
	}{
		{{255, 255, 255, 255}, {255, 255, 255, 255}},
		{{0, 0, 0, 255}, {0, 0, 0, 255}},
		{{255, 0, 0, 255}, {255, 0, 0, 255}},
		{{0, 0, 255, 255}, {0, 0, 255, 255}},
		{{128, 128, 128, 255}, {0, 0, 0, 255}},
	}

	for _, tt := range tests {
		got := nearestColor(tt.in, palette.Colors)
		gr, gg, gb, ga := got.RGBA()
		wr, wg, wb, wa := tt.want.RGBA()
		if gr != wr || gg != wg || gb != wb || ga != wa {
			t.Errorf("nearestColor(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestMapPalette(t *testing.T) {
	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}

	src := newImage(2, 1, red, blue)
	out := MapPalette(src, palette, false)

	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := out.At(x, y)
			if !paletteContains(c, palette.Colors) {
				t.Errorf("pixel (%d,%d) = %v not in palette", x, y, c)
			}
		}
	}
}

func TestMapPaletteDither(t *testing.T) {
	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}

	src := newImage(8, 8, red)
	out := MapPalette(src, palette, true)

	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := out.At(x, y)
			if !paletteContains(c, palette.Colors) {
				t.Errorf("pixel (%d,%d) = %v not in palette", x, y, c)
			}
		}
	}
}

func TestMapToBinary(t *testing.T) {
	src := newImage(2, 2, color.NRGBA{0, 0, 0, 255}, color.NRGBA{255, 255, 255, 255}, color.NRGBA{0, 0, 0, 255}, color.NRGBA{255, 255, 255, 255})
	out := mapToBinary(src, 128)

	black := color.NRGBA{0, 0, 0, 255}
	white := color.NRGBA{255, 255, 255, 255}
	if out.At(0, 0) != black || out.At(1, 0) != white {
		t.Errorf("mapToBinary returned unexpected colors")
	}
}
