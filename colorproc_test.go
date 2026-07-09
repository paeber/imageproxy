// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"image/color"
	"testing"
)

func TestApplyColorProcessingGrayscale(t *testing.T) {
	src := newImage(2, 1, red, blue)
	out := applyColorProcessing(src, Options{Grayscale: true})

	r, g, b, _ := out.At(0, 0).RGBA()
	if r != g || g != b {
		t.Errorf("grayscale pixel not uniform: r=%d g=%d b=%d", r>>8, g>>8, b>>8)
	}
}

func TestApplyColorProcessingBinary(t *testing.T) {
	src := newImage(2, 1, color.NRGBA{0, 0, 0, 255}, color.NRGBA{255, 255, 255, 255})
	out := applyColorProcessing(src, Options{Binary: true, BinaryThreshold: 128})

	black := color.NRGBA{0, 0, 0, 255}
	white := color.NRGBA{255, 255, 255, 255}
	if out.At(0, 0) != black || out.At(1, 0) != white {
		t.Errorf("binary conversion returned unexpected colors")
	}
}

func TestApplyColorProcessingPalette(t *testing.T) {
	src := newImage(2, 1, red, blue)
	out := applyColorProcessing(src, Options{Palette: "E1002"})

	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			if !paletteContains(out.At(x, y), palette.Colors) {
				t.Errorf("pixel (%d,%d) not in palette", x, y)
			}
		}
	}
}

func TestApplyColorProcessingPalettePrecedence(t *testing.T) {
	src := newImage(1, 1, color.NRGBA{128, 128, 128, 255})
	out := applyColorProcessing(src, Options{Binary: true, Palette: "E1002"})

	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}
	if !paletteContains(out.At(0, 0), palette.Colors) {
		t.Errorf("palette should take precedence over binary")
	}
}

func TestApplyColorProcessingCover(t *testing.T) {
	src := newImage(40, 40, color.NRGBA{180, 100, 100, 255})
	out := applyColorProcessing(src, Options{Palette: "E1002", CoverPreset: true, PaletteVivid: true, PaletteSatMin: 10, StructureRegions: 12, DitherEdge: true, Dither: true, StructureEdge: 30, StructureDilate: 1, StructureOverlay: true})

	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			if !paletteContains(out.At(x, y), palette.Colors) {
				t.Errorf("pixel (%d,%d) not in palette", x, y)
			}
		}
	}
}

func TestOptionsTransformColor(t *testing.T) {
	tests := []struct {
		name string
		opt  Options
	}{
		{"palette", Options{Palette: "E1002"}},
		{"grayscale", Options{Grayscale: true}},
		{"binary", Options{Binary: true}},
	}
	for _, tt := range tests {
		if !tt.opt.transform() {
			t.Errorf("%s options should trigger transform", tt.name)
		}
	}
}
