// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"math"
	"testing"
)

func TestRGBToHSVRoundTrip(t *testing.T) {
	r, g, b := uint8(135), uint8(206), uint8(235)
	h, s, v := rgbToHSV(r, g, b)
	rr, gg, bb := hsvToRGB(h, s, v)
	if math.Abs(float64(rr)-float64(r)) > 2 || math.Abs(float64(gg)-float64(g)) > 2 || math.Abs(float64(bb)-float64(b)) > 2 {
		t.Fatalf("hsv round trip drift: got (%d,%d,%d) want (%d,%d,%d)", rr, gg, bb, r, g, b)
	}
}

func TestAdjustSaturation(t *testing.T) {
	r, g, b := adjustSaturation(200, 200, 220, 1.5)
	_, s, _ := rgbToHSV(r, g, b)
	_, s0, _ := rgbToHSV(200, 200, 220)
	if s <= s0 {
		t.Errorf("expected higher saturation after boost: before=%f after=%f", s0, s)
	}
}
