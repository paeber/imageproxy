// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

// Palette is a named set of colors for pixel mapping.
type Palette struct {
	ID     string
	Colors []color.Color
}

// Spectra 6 ePaper colors (reTerminal E1002 / E1004).
var spectra6Colors = []color.Color{
	color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // white
	color.NRGBA{R: 0, G: 0, B: 0, A: 255},       // black
	color.NRGBA{R: 0, G: 255, B: 0, A: 255},     // green
	color.NRGBA{R: 255, G: 0, B: 0, A: 255},     // red
	color.NRGBA{R: 255, G: 255, B: 0, A: 255},   // yellow
	color.NRGBA{R: 0, G: 0, B: 255, A: 255},     // blue
}

var bwColors = []color.Color{
	color.NRGBA{R: 0, G: 0, B: 0, A: 255},
	color.NRGBA{R: 255, G: 255, B: 255, A: 255},
}

var palettes = map[string]Palette{
	"E1002": {ID: "E1002", Colors: spectra6Colors},
	"BW":    {ID: "BW", Colors: bwColors},
}

// LookupPalette returns a predefined palette by ID (case-insensitive).
func LookupPalette(id string) (Palette, error) {
	p, ok := palettes[strings.ToUpper(id)]
	if !ok {
		return Palette{}, fmt.Errorf("unknown palette %q", id)
	}
	return p, nil
}

// nearestColor returns the palette color closest to c using weighted RGB distance.
func nearestColor(c color.Color, palette []color.Color) color.Color {
	cr, cg, cb, _ := c.RGBA()
	// RGBA returns 16-bit values; scale to 8-bit for distance calculation.
	r8 := int(cr >> 8)
	g8 := int(cg >> 8)
	b8 := int(cb >> 8)

	best := palette[0]
	bestDist := -1
	for _, p := range palette {
		pr, pg, pb, _ := p.RGBA()
		dr := r8 - int(pr>>8)
		dg := g8 - int(pg>>8)
		db := b8 - int(pb>>8)
		// Weight green channel slightly higher for perceptual match on ePaper.
		dist := dr*dr*2 + dg*dg*4 + db*db*3
		if bestDist < 0 || dist < bestDist {
			bestDist = dist
			best = p
		}
	}
	return best
}

func colorToRGBA8(c color.Color) (r, g, b, a uint8) {
	cr, cg, cb, ca := c.RGBA()
	return uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8), uint8(ca >> 8)
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// MapPalette maps each pixel in img to the nearest palette color.
// When dither is true, Floyd-Steinberg error diffusion is applied.
func MapPalette(img image.Image, palette Palette, dither bool) image.Image {
	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	if !dither {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				out.Set(x, y, nearestColor(img.At(x, y), palette.Colors))
			}
		}
		return out
	}

	// Working buffer with float error accumulation per channel.
	w := bounds.Dx()
	h := bounds.Dy()
	buf := make([][3]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			px := img.At(bounds.Min.X+x, bounds.Min.Y+y)
			r, g, b, _ := colorToRGBA8(px)
			i := y*w + x
			oldR := float64(r) + buf[i][0]
			oldG := float64(g) + buf[i][1]
			oldB := float64(b) + buf[i][2]
			nearest := nearestColor(color.NRGBA{
				R: clampByte(oldR),
				G: clampByte(oldG),
				B: clampByte(oldB),
				A: 255,
			}, palette.Colors)
			nr, ng, nb, _ := colorToRGBA8(nearest)
			out.Set(bounds.Min.X+x, bounds.Min.Y+y, nearest)

			errR := oldR - float64(nr)
			errG := oldG - float64(ng)
			errB := oldB - float64(nb)
			diffuseFloydSteinberg(buf, w, h, x, y, errR, errG, errB)
		}
	}
	return out
}

func diffuseFloydSteinberg(buf [][3]float64, w, h, x, y int, errR, errG, errB float64) {
	add := func(nx, ny int, factor float64) {
		if nx < 0 || nx >= w || ny < 0 || ny >= h {
			return
		}
		i := ny*w + nx
		buf[i][0] += errR * factor
		buf[i][1] += errG * factor
		buf[i][2] += errB * factor
	}
	add(x+1, y, 7.0/16.0)
	add(x-1, y+1, 3.0/16.0)
	add(x, y+1, 5.0/16.0)
	add(x+1, y+1, 1.0/16.0)
}

// paletteContains reports whether c matches any color in the palette.
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

// mapToBinary converts img to black and white using the given threshold.
func mapToBinary(img image.Image, threshold int) image.Image {
	if threshold <= 0 {
		threshold = 128
	}
	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	black := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := colorToRGBA8(img.At(x, y))
			lum := int(r)*299 + int(g)*587 + int(b)*114
			if lum >= threshold*1000 {
				out.Set(x, y, white)
			} else {
				out.Set(x, y, black)
			}
		}
	}
	return out
}
