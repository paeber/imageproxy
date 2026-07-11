// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

//go:build coverreport

package imageproxy_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
)

func validatePixelsInPalette(out []byte, palette []color.Color) error {
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !paletteContains(img.At(x, y), palette) {
				return fmt.Errorf("pixel (%d,%d) not in palette", x, y)
			}
		}
	}
	return nil
}

func paletteContains(c color.Color, palette []color.Color) bool {
	cr, cg, cb, ca := c.RGBA()
	for _, p := range palette {
		pr, pg, pb, pa := p.RGBA()
		if cr == pr && cg == pg && cb == pb && ca == pa {
			return true
		}
	}
	return false
}
