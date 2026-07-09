// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"image"
	"image/color"
	"testing"
)

func TestSobelEdgeMapDetectsEdge(t *testing.T) {
	// Left half black, right half white — vertical edge in the middle.
	src := newImage(10, 10, color.NRGBA{0, 0, 0, 255})
	for x := 5; x < 10; x++ {
		for y := 0; y < 10; y++ {
			src.(*image.NRGBA).Set(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}

	edges := sobelEdgeMap(src, src.Bounds())
	w := src.Bounds().Dx()

	var edgeSum, flatSum float64
	var edgeCount, flatCount int
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			v := edges[y*w+x]
			if x == 4 || x == 5 {
				edgeSum += v
				edgeCount++
			} else if x <= 2 || x >= 7 {
				flatSum += v
				flatCount++
			}
		}
	}
	if edgeCount == 0 || flatCount == 0 {
		t.Fatal("expected edge and flat samples")
	}
	if edgeSum/float64(edgeCount) <= flatSum/float64(flatCount) {
		t.Errorf("edge magnitude %v should exceed flat %v", edgeSum/float64(edgeCount), flatSum/float64(flatCount))
	}
}

func TestMorphDilateExpandsMask(t *testing.T) {
	mask := make([]bool, 9)
	mask[4] = true // center of 3x3
	out := morphDilate(mask, 3, 3, 1)
	if !out[1] || !out[3] || !out[5] || !out[7] {
		t.Errorf("dilate should expand to neighbors, got %v", out)
	}
}

func TestMorphErodeShrinksMask(t *testing.T) {
	mask := make([]bool, 9)
	for i := range mask {
		mask[i] = true
	}
	mask[0] = false
	mask[2] = false
	mask[6] = false
	mask[8] = false
	out := morphErode(mask, 3, 3, 1)
	if out[4] {
		t.Errorf("center should erode when surrounded by partial mask")
	}
}

func TestLabelConnectedComponents(t *testing.T) {
	// Two separate red regions with different bucket IDs.
	buckets := []int{
		0, 0, 1, 1,
		0, 0, 1, 1,
	}
	labels := labelConnectedComponents(buckets, 4, 2)
	if labels[0] == labels[2] {
		t.Errorf("separate components should have different labels: %v", labels)
	}
	if labels[0] != labels[1] || labels[2] != labels[3] {
		t.Errorf("connected pixels should share labels: %v", labels)
	}
}

func TestMedianCutBucketsUniformColors(t *testing.T) {
	src := newImage(4, 2,
		color.NRGBA{200, 0, 0, 255}, color.NRGBA{200, 0, 0, 255},
		color.NRGBA{0, 0, 200, 255}, color.NRGBA{0, 0, 200, 255},
		color.NRGBA{200, 0, 0, 255}, color.NRGBA{200, 0, 0, 255},
		color.NRGBA{0, 0, 200, 255}, color.NRGBA{0, 0, 200, 255},
	)
	buckets := medianCutBuckets(src, src.Bounds(), 4)
	if buckets[0] == buckets[2] {
		t.Errorf("red and blue areas should land in different buckets: %v", buckets)
	}
	if buckets[0] != buckets[1] {
		t.Errorf("same-color neighbors should share bucket: %v", buckets)
	}
}

func TestBuildStructureContextCover(t *testing.T) {
	src := newImage(20, 20, color.NRGBA{100, 120, 140, 255})
	opt := ParseOptions("cover")
	ctx := buildStructureContext(src, opt)
	if ctx == nil {
		t.Fatal("cover preset should enable structure context")
	}
	if len(ctx.EdgeMap) != 400 {
		t.Errorf("edge map length = %d, want 400", len(ctx.EdgeMap))
	}
	if len(ctx.RegionIDs) != 400 {
		t.Errorf("region IDs length = %d, want 400", len(ctx.RegionIDs))
	}
}

func TestIsRegionInteriorFlatRegion(t *testing.T) {
	src := newImage(10, 10, color.NRGBA{200, 50, 50, 255})
	opt := ParseOptions("regions8")
	ctx := buildStructureContext(src, opt)
	if ctx == nil || ctx.RegionIDs == nil {
		t.Fatal("expected region context")
	}
	interior := 0
	for y := 2; y < 8; y++ {
		for x := 2; x < 8; x++ {
			if isRegionInterior(ctx, x, y, 0.15) {
				interior++
			}
		}
	}
	if interior == 0 {
		t.Error("uniform image should have interior region pixels")
	}
}

func TestRegionDominantColorsUniformRegion(t *testing.T) {
	palette, err := LookupPalette("E1002")
	if err != nil {
		t.Fatal(err)
	}
	matcher := newPaletteMatcher(palette.Colors, PaletteMapConfig{Mode: "hue"})
	src := newImage(10, 10, color.NRGBA{255, 0, 0, 255})
	opt := ParseOptions("regions8")
	ctx := buildStructureContext(src, opt)
	colors := regionDominantColors(src, src.Bounds(), ctx.RegionIDs, matcher)
	if len(colors) == 0 {
		t.Fatal("expected at least one region color")
	}
	for _, c := range colors {
		r, g, b := colorToRGBA8(c)
		if r != 255 || g != 0 || b != 0 {
			t.Errorf("red region should map to red, got (%d,%d,%d)", r, g, b)
		}
	}
}
