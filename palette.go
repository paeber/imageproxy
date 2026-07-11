// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"fmt"
	"image"
	"image/color"
	"math"
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

func colorToRGBA8(c color.Color) (uint8, uint8, uint8) {
	cr, cg, cb, _ := c.RGBA()
	return uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8)
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
// When cfg.Dither is true, Floyd-Steinberg error diffusion is applied.
// Structure-aware options use region colors in flat areas, edge-aware dither
// at boundaries, and an optional structure overlay for high-contrast edges.
func MapPalette(img image.Image, palette Palette, cfg PaletteMapConfig) image.Image {
	matcher := newPaletteMatcher(palette.Colors, cfg)
	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)
	w := bounds.Dx()
	h := bounds.Dy()

	var regionColors map[int]color.Color
	if cfg.Structure != nil && cfg.Structure.RegionIDs != nil {
		regionColors = regionDominantColors(img, bounds, cfg.Structure.RegionIDs, matcher)
	}

	interiorThreshold := cfg.EdgeThreshold * 0.5
	if interiorThreshold <= 0 {
		interiorThreshold = 0.15
	}

	black := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	applyOutline := func() {
		if !cfg.Outline || cfg.Structure == nil || cfg.Structure.OutlineMask == nil {
			return
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if cfg.Structure.OutlineMask[y*w+x] {
					out.Set(bounds.Min.X+x, bounds.Min.Y+y, black)
				}
			}
		}
	}

	if !cfg.Dither {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if cfg.StructureOverlay && cfg.Structure != nil && cfg.Structure.EdgeMask != nil && cfg.Structure.EdgeMask[y*w+x] {
					applyStructureOverlayAt(img, bounds, out, x, y, black, white)
					continue
				}
				if regionColors != nil && isRegionInterior(cfg.Structure, x, y, interiorThreshold) {
					out.Set(bounds.Min.X+x, bounds.Min.Y+y, regionColors[cfg.Structure.RegionIDs[y*w+x]])
					continue
				}
				if cfg.FillFlat && cfg.Structure != nil && cfg.Structure.RegionIDs != nil && !isInBoundaryBand(cfg.Structure, x, y) {
					rid := cfg.Structure.RegionIDs[y*w+x]
					if c, ok := regionColors[rid]; ok {
						out.Set(bounds.Min.X+x, bounds.Min.Y+y, c)
						continue
					}
				}
				if cfg.DitherSmooth > 0 && isHighActivity(cfg.Structure, x, y, float64(cfg.DitherSmooth)/100.0) {
					r, g, b := colorToRGBA8(img.At(bounds.Min.X+x, bounds.Min.Y+y))
					out.Set(bounds.Min.X+x, bounds.Min.Y+y, matcher.match(r, g, b))
					continue
				}
				r, g, b := colorToRGBA8(img.At(bounds.Min.X+x, bounds.Min.Y+y))
				out.Set(bounds.Min.X+x, bounds.Min.Y+y, matcher.match(r, g, b))
			}
		}
		applyOutline()
		return out
	}

	buf := make([][3]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if cfg.StructureOverlay && cfg.Structure != nil && cfg.Structure.EdgeMask != nil && cfg.Structure.EdgeMask[i] {
				applyStructureOverlayAt(img, bounds, out, x, y, black, white)
				continue
			}
			if regionColors != nil && isRegionInterior(cfg.Structure, x, y, interiorThreshold) {
				c := regionColors[cfg.Structure.RegionIDs[i]]
				out.Set(bounds.Min.X+x, bounds.Min.Y+y, c)
				nr, ng, nb := colorToRGBA8(c)
				px := img.At(bounds.Min.X+x, bounds.Min.Y+y)
				r, g, b := colorToRGBA8(px)
				oldR := float64(r) + buf[i][0]
				oldG := float64(g) + buf[i][1]
				oldB := float64(b) + buf[i][2]
				errR := oldR - float64(nr)
				errG := oldG - float64(ng)
				errB := oldB - float64(nb)
				diffuseFloydSteinberg(buf, w, h, x, y, errR, errG, errB, cfg)
				continue
			}

			if cfg.FillFlat && cfg.Structure != nil && cfg.Structure.RegionIDs != nil && !isInBoundaryBand(cfg.Structure, x, y) {
				rid := cfg.Structure.RegionIDs[i]
				if c, ok := regionColors[rid]; ok {
					out.Set(bounds.Min.X+x, bounds.Min.Y+y, c)
					nr, ng, nb := colorToRGBA8(c)
					px := img.At(bounds.Min.X+x, bounds.Min.Y+y)
					r, g, b := colorToRGBA8(px)
					oldR := float64(r) + buf[i][0]
					oldG := float64(g) + buf[i][1]
					oldB := float64(b) + buf[i][2]
					errR := oldR - float64(nr)
					errG := oldG - float64(ng)
					errB := oldB - float64(nb)
					diffuseFloydSteinberg(buf, w, h, x, y, errR, errG, errB, cfg)
					continue
				}
			}

			if !shouldDitherAt(cfg.Structure, x, y, cfg.DitherSmooth) {
				px := img.At(bounds.Min.X+x, bounds.Min.Y+y)
				r, g, b := colorToRGBA8(px)
				oldR := float64(r) + buf[i][0]
				oldG := float64(g) + buf[i][1]
				oldB := float64(b) + buf[i][2]
				nearest := matcher.match(clampByte(oldR), clampByte(oldG), clampByte(oldB))
				out.Set(bounds.Min.X+x, bounds.Min.Y+y, nearest)
				continue
			}

			px := img.At(bounds.Min.X+x, bounds.Min.Y+y)
			r, g, b := colorToRGBA8(px)
			oldR := float64(r) + buf[i][0]
			oldG := float64(g) + buf[i][1]
			oldB := float64(b) + buf[i][2]
			nearest := matcher.match(clampByte(oldR), clampByte(oldG), clampByte(oldB))
			nr, ng, nb := colorToRGBA8(nearest)
			out.Set(bounds.Min.X+x, bounds.Min.Y+y, nearest)

			errR := oldR - float64(nr)
			errG := oldG - float64(ng)
			errB := oldB - float64(nb)
			diffuseFloydSteinberg(buf, w, h, x, y, errR, errG, errB, cfg)
		}
	}
	applyOutline()
	return out
}

func diffuseFloydSteinberg(buf [][3]float64, w, h, x, y int, errR, errG, errB float64, cfg PaletteMapConfig) {
	scale := cfg.DitherAmount
	if cfg.Saliency {
		scale *= ditherModulationAt(cfg.Structure, x, y)
	}
	if scale <= 0 {
		return
	}
	errR *= scale
	errG *= scale
	errB *= scale
	edgeAt := func(nx, ny int) float64 {
		if cfg.Structure == nil || cfg.Structure.EdgeMap == nil {
			return 0
		}
		if nx < 0 || nx >= w || ny < 0 || ny >= h {
			return 0
		}
		return cfg.Structure.EdgeMap[ny*w+nx]
	}

	srcEdge := edgeAt(x, y)
	attenuate := func(nx, ny int, factor float64) float64 {
		if cfg.Structure != nil && cfg.Structure.BoundaryBand != nil {
			ni := ny*w + nx
			if ni >= 0 && ni < len(cfg.Structure.BoundaryBand) && cfg.Structure.BoundaryBand[ni] {
				return 0
			}
		}
		if !cfg.DitherEdge {
			return factor
		}
		neighborEdge := edgeAt(nx, ny)
		strength := math.Max(srcEdge, neighborEdge)
		if cfg.EdgeThreshold > 0 && strength >= cfg.EdgeThreshold {
			return 0
		}
		return factor * (1.0 - strength)
	}

	add := func(nx, ny int, factor float64) {
		if nx < 0 || nx >= w || ny < 0 || ny >= h {
			return
		}
		f := attenuate(nx, ny, factor)
		if f == 0 {
			return
		}
		i := ny*w + nx
		buf[i][0] += errR * f
		buf[i][1] += errG * f
		buf[i][2] += errB * f
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
			r, g, b := colorToRGBA8(img.At(x, y))
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
