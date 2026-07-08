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

func TestPaletteMatcherHue(t *testing.T) {
	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}
	matcher := newPaletteMatcher(palette.Colors, PaletteMapConfig{Mode: "hue"})

	red := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	blue := color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.NRGBA{R: 0, G: 0, B: 0, A: 255}

	tests := []struct {
		name string
		in   color.NRGBA
		want color.NRGBA
	}{
		{"pure red", red, red},
		{"pure blue", blue, blue},
		{"pink", color.NRGBA{R: 255, G: 192, B: 203, A: 255}, red},
		{"light blue", color.NRGBA{R: 173, G: 216, B: 230, A: 255}, blue},
		{"sky blue", color.NRGBA{R: 135, G: 206, B: 235, A: 255}, blue},
		{"near white", color.NRGBA{R: 250, G: 250, B: 250, A: 255}, white},
		{"dark gray", color.NRGBA{R: 30, G: 30, B: 30, A: 255}, black},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.match(tt.in.R, tt.in.G, tt.in.B)
			gr, gg, gb := colorToRGBA8(got)
			wr, wg, wb := colorToRGBA8(tt.want)
			if gr != wr || gg != wg || gb != wb {
				t.Errorf("match(%v) = (%d,%d,%d), want (%d,%d,%d)", tt.in, gr, gg, gb, wr, wg, wb)
			}
		})
	}
}

func TestPaletteMatcherRGB(t *testing.T) {
	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}
	matcher := newPaletteMatcher(palette.Colors, PaletteMapConfig{Mode: "rgb"})

	got := matcher.match(128, 128, 128)
	gr, gg, gb := colorToRGBA8(got)
	if gr != 0 || gg != 0 || gb != 0 {
		t.Errorf("rgb mode gray mapped to (%d,%d,%d), want black", gr, gg, gb)
	}
}

func TestMapPalette(t *testing.T) {
	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}

	src := newImage(2, 1, red, blue)
	out := MapPalette(src, palette, PaletteMapConfig{})

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
	out := MapPalette(src, palette, PaletteMapConfig{Dither: true})

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

	black := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	if out.At(0, 0) != black || out.At(1, 0) != white {
		t.Errorf("mapToBinary returned unexpected colors")
	}
}
