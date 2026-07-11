// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"image"
	"image/color"
	"testing"
)

func TestLocalVarianceMapUniform(t *testing.T) {
	src := newImage(20, 20, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	bounds := src.Bounds()
	varMap := localVarianceMap(src, bounds)
	for _, v := range varMap {
		if v > 0.01 {
			t.Errorf("uniform image should have near-zero variance, got %f", v)
		}
	}
}

func TestLocalVarianceMapStripes(t *testing.T) {
	src := newImage(20, 20, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	m := src.(*image.NRGBA)
	for x := 0; x < 20; x++ {
		for y := 0; y < 20; y++ {
			if x%2 == 0 {
				m.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}
	varMap := localVarianceMap(src, src.Bounds())
	high := 0
	for _, v := range varMap {
		if v > 0.3 {
			high++
		}
	}
	if high == 0 {
		t.Error("striped image should have high-variance pixels")
	}
}

func TestFelzenszwalbSegmentContiguous(t *testing.T) {
	src := newImage(40, 40, color.NRGBA{R: 200, G: 50, B: 50, A: 255})
	m := src.(*image.NRGBA)
	for y := 0; y < 40; y++ {
		for x := 20; x < 40; x++ {
			m.Set(x, y, color.NRGBA{R: 50, G: 50, B: 200, A: 255})
		}
	}
	labels := felzenszwalbSegment(src, src.Bounds(), 500)
	if len(labels) != 40*40 {
		t.Fatalf("expected %d labels, got %d", 40*40, len(labels))
	}
	if len(uniqueInts(labels)) < 2 {
		t.Error("segmentation should produce multiple regions on two-color image")
	}
}

func TestMorphGradient(t *testing.T) {
	mask := make([]bool, 9)
	mask[4] = true
	grad := morphGradient(mask, 3, 3)
	if !grad[4] {
		t.Error("gradient should include edge pixel")
	}
}

func TestCoverPMPresetParse(t *testing.T) {
	opt := ParseOptions("palE1002,coverpm")
	if opt.PaletteMode != "rgb" || !opt.Dither || !opt.DitherEdge {
		t.Errorf("coverpm preset mismatch: %#v", opt)
	}
	if opt.StructureRegions != 12 || opt.StructureEdge != 30 {
		t.Errorf("coverpm structure defaults mismatch: %#v", opt)
	}
}

func uniqueInts(in []int) map[int]bool {
	m := make(map[int]bool)
	for _, v := range in {
		m[v] = true
	}
	return m
}
