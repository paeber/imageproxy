// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"image/color"
	"math"
)

const (
	defaultPaletteSatMin = 15
	defaultVividFactor   = 1.35
)

// PaletteMapConfig controls how source colors are matched to a fixed palette.
type PaletteMapConfig struct {
	Dither     bool
	DitherEdge bool // attenuate error diffusion across strong edges
	Mode       string
	SatMin     int
	Vivid      bool

	Structure        *StructureContext
	StructureOverlay bool
	EdgeThreshold    float64 // 0-1, for interior detection and dither attenuation
	DitherSmooth     int     // activity threshold 1-100
	FillFlat         bool
	Outline          bool
	DitherAmount     float64 // 0-1 scale
	Saliency         bool
}

func paletteMapConfigFromOptions(opt Options, structure *StructureContext) PaletteMapConfig {
	dither := opt.Dither || opt.DitherEdge
	overlay := opt.StructureOverlay || opt.CoverPreset

	edgeThreshold := 0.0
	if opt.StructureEdge > 0 {
		edgeThreshold = float64(opt.StructureEdge) / 100.0
	} else if opt.CoverPreset {
		edgeThreshold = float64(defaultCoverEdge) / 100.0
	}

	return PaletteMapConfig{
		Dither:           dither,
		DitherEdge:       opt.DitherEdge || opt.CoverPreset || opt.CoverPMPreset,
		Mode:             opt.PaletteMode,
		SatMin:           opt.PaletteSatMin,
		Vivid:            opt.PaletteVivid,
		Structure:        structure,
		StructureOverlay: overlay,
		EdgeThreshold:    edgeThreshold,
		DitherSmooth:     opt.DitherSmooth,
		FillFlat:         opt.FillFlat,
		Outline:          opt.Outline,
		DitherAmount:     opt.ditherAmountScale(),
		Saliency:         opt.Saliency,
	}
}

type paletteEntry struct {
	c          color.NRGBA
	h, s, v    float64
	L, a, b    float64
	achromatic bool
}

type paletteMatcher struct {
	chromatic   []paletteEntry
	achromatic  []paletteEntry
	mode        string
	satMin      float64
	vividFactor float64
}

func newPaletteMatcher(colors []color.Color, cfg PaletteMapConfig) *paletteMatcher {
	mode := cfg.Mode
	if mode == "" {
		mode = "hue"
	}
	satMin := float64(defaultPaletteSatMin) / 100
	if cfg.SatMin > 0 {
		satMin = float64(cfg.SatMin) / 100
	}
	vividFactor := 1.0
	if cfg.Vivid {
		vividFactor = defaultVividFactor
	}

	m := &paletteMatcher{
		mode:        mode,
		satMin:      satMin,
		vividFactor: vividFactor,
	}
	for _, c := range colors {
		r, g, b := colorToRGBA8(c)
		h, s, v := rgbToHSV(r, g, b)
		L, aLab, bLab := rgbToLab(r, g, b)
		entry := paletteEntry{
			c:          color.NRGBA{R: r, G: g, B: b, A: 255},
			h:          h,
			s:          s,
			v:          v,
			L:          L,
			a:          aLab,
			b:          bLab,
			achromatic: s < 0.12,
		}
		if entry.achromatic {
			m.achromatic = append(m.achromatic, entry)
		} else {
			m.chromatic = append(m.chromatic, entry)
		}
	}
	return m
}

func (m *paletteMatcher) match(r, g, b uint8) color.Color {
	r, g, b = m.prepareRGB(r, g, b)
	switch m.mode {
	case "rgb":
		return nearestColorRGB(r, g, b, m.allColors())
	case "lab":
		return m.matchLab(r, g, b)
	default:
		return m.matchHue(r, g, b)
	}
}

func (m *paletteMatcher) allColors() []color.NRGBA {
	out := make([]color.NRGBA, 0, len(m.chromatic)+len(m.achromatic))
	for _, e := range m.chromatic {
		out = append(out, e.c)
	}
	for _, e := range m.achromatic {
		out = append(out, e.c)
	}
	return out
}

func (m *paletteMatcher) prepareRGB(r, g, b uint8) (uint8, uint8, uint8) {
	if m.vividFactor <= 1.0 {
		return r, g, b
	}
	return adjustSaturation(r, g, b, m.vividFactor)
}

func (m *paletteMatcher) matchAchromatic(v float64) color.Color {
	if len(m.achromatic) == 0 {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if v > 0.92 {
		return m.lightestAchromatic()
	}
	if v < 0.08 {
		return m.darkestAchromatic()
	}
	if v >= 0.5 {
		return m.lightestAchromatic()
	}
	return m.darkestAchromatic()
}

func (m *paletteMatcher) lightestAchromatic() color.Color {
	best := m.achromatic[0]
	for _, e := range m.achromatic[1:] {
		if e.v > best.v {
			best = e
		}
	}
	return best.c
}

func (m *paletteMatcher) darkestAchromatic() color.Color {
	best := m.achromatic[0]
	for _, e := range m.achromatic[1:] {
		if e.v < best.v {
			best = e
		}
	}
	return best.c
}

func (m *paletteMatcher) matchHue(r, g, b uint8) color.Color {
	h, s, v := rgbToHSV(r, g, b)
	if s < m.satMin {
		return m.matchAchromatic(v)
	}
	if len(m.chromatic) == 0 {
		return m.matchAchromatic(v)
	}

	best := m.chromatic[0]
	bestScore := math.MaxFloat64
	for _, pe := range m.chromatic {
		hd := hueDistanceDeg(h, pe.h) / 180.0
		sd := math.Abs(s - pe.s)
		vd := math.Abs(v - pe.v)
		score := hd*5.0 + sd*1.5 + vd*1.0
		if score < bestScore {
			bestScore = score
			best = pe
		}
	}
	return best.c
}

func (m *paletteMatcher) matchLab(r, g, b uint8) color.Color {
	L, a, bLab := rgbToLab(r, g, b)
	_, s, v := rgbToHSV(r, g, b)
	if s < m.satMin {
		return m.matchAchromatic(v)
	}
	if len(m.chromatic) == 0 {
		return m.matchAchromatic(v)
	}

	best := m.chromatic[0].c
	bestDist := math.MaxFloat64
	for _, pe := range m.chromatic {
		d := deltaE76(L, a, bLab, pe.L, pe.a, pe.b)
		if d < bestDist {
			bestDist = d
			best = pe.c
		}
	}
	return best
}

func nearestColorRGB(r, g, b uint8, palette []color.NRGBA) color.Color {
	best := palette[0]
	bestDist := -1
	for _, p := range palette {
		dr := int(r) - int(p.R)
		dg := int(g) - int(p.G)
		db := int(b) - int(p.B)
		dist := dr*dr*2 + dg*dg*4 + db*db*3
		if bestDist < 0 || dist < bestDist {
			bestDist = dist
			best = p
		}
	}
	return best
}

func rgbToHSV(r, g, b uint8) (h, s, v float64) {
	rf := float64(r) / 255
	gf := float64(g) / 255
	bf := float64(b) / 255
	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	v = max
	delta := max - min
	if max == 0 {
		return 0, 0, 0
	}
	s = delta / max
	if delta == 0 {
		return 0, s, v
	}
	switch max {
	case rf:
		h = 60 * math.Mod((gf-bf)/delta, 6)
	case gf:
		h = 60 * (((bf-rf)/delta) + 2)
	default:
		h = 60 * (((rf-gf)/delta) + 4)
	}
	if h < 0 {
		h += 360
	}
	return h, s, v
}

func hsvToRGB(h, s, v float64) (r, g, b uint8) {
	if s == 0 {
		return clampByte(v * 255), clampByte(v * 255), clampByte(v * 255)
	}
	h = math.Mod(h, 360)
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var rf, gf, bf float64
	switch {
	case h < 60:
		rf, gf, bf = c, x, 0
	case h < 120:
		rf, gf, bf = x, c, 0
	case h < 180:
		rf, gf, bf = 0, c, x
	case h < 240:
		rf, gf, bf = 0, x, c
	case h < 300:
		rf, gf, bf = x, 0, c
	default:
		rf, gf, bf = c, 0, x
	}
	return clampByte((rf + m) * 255), clampByte((gf + m) * 255), clampByte((bf + m) * 255)
}

func adjustSaturation(r, g, b uint8, factor float64) (uint8, uint8, uint8) {
	h, s, v := rgbToHSV(r, g, b)
	s = math.Min(1, s*factor)
	return hsvToRGB(h, s, v)
}

func hueDistanceDeg(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func rgbToLab(r, g, b uint8) (L, a, bLab float64) {
	rf := srgbToLinear(float64(r) / 255)
	gf := srgbToLinear(float64(g) / 255)
	bf := srgbToLinear(float64(b) / 255)

	x := rf*0.4124564 + gf*0.3575761 + bf*0.1804375
	y := rf*0.2126729 + gf*0.7151522 + bf*0.0721750
	z := rf*0.0193339 + gf*0.1191920 + bf*0.9503041

	x /= 0.95047
	z /= 1.08883

	fx := labF(x)
	fy := labF(y)
	fz := labF(z)

	L = 116*fy - 16
	a = 500 * (fx - fy)
	bLab = 200 * (fy - fz)
	return L, a, bLab
}

func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func labF(t float64) float64 {
	const delta = 6.0 / 29.0
	if t > delta*delta*delta {
		return math.Cbrt(t)
	}
	return t/(3*delta*delta) + 4.0/29.0
}

func deltaE76(L1, a1, b1, L2, a2, b2 float64) float64 {
	dL := L1 - L2
	da := a1 - a2
	db := b1 - b2
	return math.Sqrt(dL*dL + da*da + db*db)
}
