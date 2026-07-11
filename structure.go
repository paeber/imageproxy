// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// StructureContext holds precomputed structure maps for palette mapping.
type StructureContext struct {
	EdgeMap      []float64 // normalized Sobel magnitude per pixel, length w*h
	EdgeMask     []bool    // binary edge mask after threshold and morphology
	ActivityMap  []float64 // local luminance variance 0-1
	SaliencyMap  []float64 // simple saliency proxy 0-1
	BoundaryBand []bool    // dilated edge band for protected dither/fillflat
	OutlineMask  []bool    // morphological gradient for outline mode
	RegionIDs    []int     // per-pixel region ID, -1 when regions disabled
	Width        int
	Height       int
}

// structureConfigFromOptions builds structure processing parameters from Options.
func structureConfigFromOptions(opt Options) (edgeThreshold int, dilate, erode, regions int, overlay bool) {
	edgeThreshold = opt.StructureEdge
	dilate = opt.StructureDilate
	erode = opt.StructureErode
	regions = opt.StructureRegions
	overlay = opt.StructureOverlay
	if opt.CoverPreset {
		if edgeThreshold == 0 {
			edgeThreshold = defaultCoverEdge
		}
		if dilate == 0 {
			dilate = defaultCoverDilate
		}
		if regions == 0 {
			regions = defaultCoverRegions
		}
		overlay = true
	}
	if opt.CoverPMPreset {
		if edgeThreshold == 0 {
			edgeThreshold = defaultCoverEdge
		}
		if dilate == 0 {
			dilate = defaultCoverDilate
		}
		if regions == 0 {
			regions = defaultCoverRegions
		}
	}
	return edgeThreshold, dilate, erode, regions, overlay
}

const (
	defaultCoverEdge    = 30
	defaultCoverDilate  = 1
	defaultCoverRegions = 12
)

// buildStructureContext computes edge and region maps for structure-aware palette mapping.
func buildStructureContext(img image.Image, opt Options) *StructureContext {
	if !opt.structureEnabled() {
		return nil
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return nil
	}

	edgeThreshold, dilate, erode, regions, _ := structureConfigFromOptions(opt)

	ctx := &StructureContext{
		Width:  w,
		Height: h,
	}

	if edgeThreshold > 0 || opt.DitherEdge || opt.StructureOverlay || opt.CoverPreset || opt.CoverPMPreset || opt.Outline || opt.FillFlat || opt.ProtectEdge > 0 {
		ctx.EdgeMap = sobelEdgeMap(img, bounds)
		threshold := 0.0
		if edgeThreshold > 0 {
			threshold = float64(edgeThreshold) / 100.0
		} else if opt.Outline || opt.FillFlat || opt.ProtectEdge > 0 {
			threshold = float64(defaultCoverEdge) / 100.0
		}
		if threshold > 0 {
			mask := thresholdEdgeMask(ctx.EdgeMap, threshold)
			if erode > 0 {
				mask = morphErode(mask, w, h, erode)
			}
			if dilate > 0 {
				mask = morphDilate(mask, w, h, dilate)
			}
			ctx.EdgeMask = mask
		}
	}

	if opt.DitherSmooth > 0 || opt.FillFlat {
		ctx.ActivityMap = localVarianceMap(img, bounds)
	}

	if opt.Saliency {
		ctx.SaliencyMap = saliencyMap(bounds, ctx.EdgeMap)
	}

	protectRadius := opt.ProtectEdge
	if protectRadius == 0 && opt.FillFlat {
		protectRadius = 2
	}
	if protectRadius > 0 && ctx.EdgeMask != nil {
		ctx.BoundaryBand = morphDilate(ctx.EdgeMask, w, h, protectRadius)
	} else if protectRadius > 0 && ctx.EdgeMap != nil {
		mask := thresholdEdgeMask(ctx.EdgeMap, 0.15)
		ctx.BoundaryBand = morphDilate(mask, w, h, protectRadius)
	}

	if opt.Outline && ctx.EdgeMask != nil {
		ctx.OutlineMask = morphGradient(ctx.EdgeMask, w, h)
	}

	if opt.SegFZH > 0 {
		ctx.RegionIDs = felzenszwalbSegment(img, bounds, opt.SegFZH)
	} else if regions > 0 {
		buckets := medianCutBuckets(img, bounds, regions)
		ctx.RegionIDs = labelConnectedComponents(buckets, w, h)
	}

	return ctx
}

func pixelLuminance(r, g, b uint8) float64 {
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
}

func sobelEdgeMap(img image.Image, bounds image.Rectangle) []float64 {
	w, h := bounds.Dx(), bounds.Dy()
	lum := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b := colorToRGBA8(img.At(bounds.Min.X+x, bounds.Min.Y+y))
			lum[y*w+x] = pixelLuminance(r, g, b)
		}
	}

	edges := make([]float64, w*h)
	var maxMag float64
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			gx := -lum[(y-1)*w+(x-1)] - 2*lum[y*w+(x-1)] - lum[(y+1)*w+(x-1)] +
				lum[(y-1)*w+(x+1)] + 2*lum[y*w+(x+1)] + lum[(y+1)*w+(x+1)]
			gy := -lum[(y-1)*w+(x-1)] - 2*lum[(y-1)*w+x] - lum[(y-1)*w+(x+1)] +
				lum[(y+1)*w+(x-1)] + 2*lum[(y+1)*w+x] + lum[(y+1)*w+(x+1)]
			mag := math.Hypot(gx, gy)
			edges[y*w+x] = mag
			if mag > maxMag {
				maxMag = mag
			}
		}
	}
	if maxMag > 0 {
		for i := range edges {
			edges[i] /= maxMag
		}
	}
	return edges
}

func thresholdEdgeMask(edges []float64, threshold float64) []bool {
	mask := make([]bool, len(edges))
	for i, e := range edges {
		mask[i] = e >= threshold
	}
	return mask
}

func morphDilate(mask []bool, w, h, radius int) []bool {
	if radius <= 0 {
		return mask
	}
	out := make([]bool, len(mask))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if mask[y*w+x] {
				for dy := -radius; dy <= radius; dy++ {
					for dx := -radius; dx <= radius; dx++ {
						nx, ny := x+dx, y+dy
						if nx >= 0 && nx < w && ny >= 0 && ny < h {
							out[ny*w+nx] = true
						}
					}
				}
			}
		}
	}
	return out
}

func morphErode(mask []bool, w, h, radius int) []bool {
	if radius <= 0 {
		return mask
	}
	out := make([]bool, len(mask))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !mask[y*w+x] {
				continue
			}
			keep := true
			for dy := -radius; dy <= radius && keep; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					nx, ny := x+dx, y+dy
					if nx < 0 || nx >= w || ny < 0 || ny >= h || !mask[ny*w+nx] {
						keep = false
						break
					}
				}
			}
			out[y*w+x] = keep
		}
	}
	return out
}

func morphGradient(mask []bool, w, h int) []bool {
	if mask == nil {
		return nil
	}
	dilated := morphDilate(mask, w, h, 1)
	eroded := morphErode(mask, w, h, 1)
	out := make([]bool, len(mask))
	for i := range out {
		out[i] = dilated[i] && !eroded[i]
	}
	return out
}

func localVarianceMap(img image.Image, bounds image.Rectangle) []float64 {
	w, h := bounds.Dx(), bounds.Dy()
	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var vals []float64
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := x+dx, y+dy
					if nx < 0 || nx >= w || ny < 0 || ny >= h {
						continue
					}
					r, g, b := colorToRGBA8(img.At(bounds.Min.X+nx, bounds.Min.Y+ny))
					vals = append(vals, pixelLuminance(r, g, b))
				}
			}
			if len(vals) == 0 {
				continue
			}
			var mean float64
			for _, v := range vals {
				mean += v
			}
			mean /= float64(len(vals))
			var varSum float64
			for _, v := range vals {
				d := v - mean
				varSum += d * d
			}
			out[y*w+x] = varSum / float64(len(vals))
		}
	}
	var maxVar float64
	for _, v := range out {
		if v > maxVar {
			maxVar = v
		}
	}
	if maxVar > 0 {
		for i := range out {
			out[i] /= maxVar
		}
	}
	return out
}

func saliencyMap(bounds image.Rectangle, edgeMap []float64) []float64 {
	w, h := bounds.Dx(), bounds.Dy()
	out := make([]float64, w*h)
	cx, cy := float64(w-1)/2, float64(h-1)/2
	maxDist := math.Hypot(cx, cy)
	if maxDist == 0 {
		maxDist = 1
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			centerBias := 1.0 - math.Hypot(float64(x)-cx, float64(y)-cy)/maxDist
			edge := 0.0
			if edgeMap != nil {
				edge = edgeMap[i]
			}
			out[i] = 0.5*centerBias + 0.5*edge
		}
	}
	return out
}

func isHighActivity(ctx *StructureContext, x, y int, threshold float64) bool {
	if ctx == nil || ctx.ActivityMap == nil {
		return false
	}
	i := y*ctx.Width + x
	return ctx.ActivityMap[i] > threshold
}

func isInBoundaryBand(ctx *StructureContext, x, y int) bool {
	if ctx == nil || ctx.BoundaryBand == nil {
		return false
	}
	i := y*ctx.Width + x
	return ctx.BoundaryBand[i]
}

func shouldDitherAt(ctx *StructureContext, x, y int, ditherSmooth int) bool {
	if ditherSmooth <= 0 {
		return true
	}
	threshold := float64(ditherSmooth) / 100.0
	return !isHighActivity(ctx, x, y, threshold)
}

func ditherModulationAt(ctx *StructureContext, x, y int) float64 {
	if ctx == nil || ctx.SaliencyMap == nil {
		return 1.0
	}
	i := y*ctx.Width + x
	return 0.35 + 0.65*ctx.SaliencyMap[i]
}

type rgbPixel struct {
	r, g, b uint8
	idx     int
}

type colorBox struct {
	pixels []rgbPixel
}

func medianCutBuckets(img image.Image, bounds image.Rectangle, n int) []int {
	w, h := bounds.Dx(), bounds.Dy()
	pixels := make([]rgbPixel, 0, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b := colorToRGBA8(img.At(bounds.Min.X+x, bounds.Min.Y+y))
			pixels = append(pixels, rgbPixel{r, g, b, y*w + x})
		}
	}
	if n < 1 {
		n = 1
	}
	if n > 32 {
		n = 32
	}

	boxes := []colorBox{{pixels: pixels}}
	for len(boxes) < n {
		bestIdx := -1
		bestRange := -1
		for i, box := range boxes {
			if len(box.pixels) < 2 {
				continue
			}
			rMin, rMax := 255, 0
			gMin, gMax := 255, 0
			bMin, bMax := 255, 0
			for _, p := range box.pixels {
				if int(p.r) < rMin {
					rMin = int(p.r)
				}
				if int(p.r) > rMax {
					rMax = int(p.r)
				}
				if int(p.g) < gMin {
					gMin = int(p.g)
				}
				if int(p.g) > gMax {
					gMax = int(p.g)
				}
				if int(p.b) < bMin {
					bMin = int(p.b)
				}
				if int(p.b) > bMax {
					bMax = int(p.b)
				}
			}
			rRange := rMax - rMin
			gRange := gMax - gMin
			bRange := bMax - bMin
			maxRange := rRange
			channel := 0
			if gRange > maxRange {
				maxRange = gRange
				channel = 1
			}
			if bRange > maxRange {
				maxRange = bRange
				channel = 2
			}
			if maxRange > bestRange {
				bestRange = maxRange
				bestIdx = i
				_ = channel
			}
		}
		if bestIdx < 0 || bestRange <= 0 {
			break
		}
		box := boxes[bestIdx]
		rMin, rMax := 255, 0
		gMin, gMax := 255, 0
		bMin, bMax := 255, 0
		for _, p := range box.pixels {
			if int(p.r) < rMin {
				rMin = int(p.r)
			}
			if int(p.r) > rMax {
				rMax = int(p.r)
			}
			if int(p.g) < gMin {
				gMin = int(p.g)
			}
			if int(p.g) > gMax {
				gMax = int(p.g)
			}
			if int(p.b) < bMin {
				bMin = int(p.b)
			}
			if int(p.b) > bMax {
				bMax = int(p.b)
			}
		}
		rRange := rMax - rMin
		gRange := gMax - gMin
		bRange := bMax - bMin
		channel := 0
		maxRange := rRange
		if gRange > maxRange {
			maxRange = gRange
			channel = 1
		}
		if bRange > maxRange {
			channel = 2
		}

		sorted := append([]rgbPixel(nil), box.pixels...)
		switch channel {
		case 0:
			sort.Slice(sorted, func(i, j int) bool { return sorted[i].r < sorted[j].r })
		case 1:
			sort.Slice(sorted, func(i, j int) bool { return sorted[i].g < sorted[j].g })
		default:
			sort.Slice(sorted, func(i, j int) bool { return sorted[i].b < sorted[j].b })
		}
		mid := len(sorted) / 2
		boxes[bestIdx] = colorBox{pixels: sorted[:mid]}
		boxes = append(boxes, colorBox{pixels: sorted[mid:]})
	}

	buckets := make([]int, w*h)
	for bucketID, box := range boxes {
		for _, p := range box.pixels {
			buckets[p.idx] = bucketID
		}
	}
	return buckets
}

func labelConnectedComponents(buckets []int, w, h int) []int {
	labels := make([]int, len(buckets))
	for i := range labels {
		labels[i] = -1
	}
	currentLabel := 0
	dirs := [4][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if labels[i] >= 0 {
				continue
			}
			bucket := buckets[i]
			stack := [][2]int{{x, y}}
			labels[i] = currentLabel
			for len(stack) > 0 {
				px, py := stack[len(stack)-1][0], stack[len(stack)-1][1]
				stack = stack[:len(stack)-1]
				for _, d := range dirs {
					nx, ny := px+d[0], py+d[1]
					if nx < 0 || nx >= w || ny < 0 || ny >= h {
						continue
					}
					ni := ny*w + nx
					if labels[ni] >= 0 || buckets[ni] != bucket {
						continue
					}
					labels[ni] = currentLabel
					stack = append(stack, [2]int{nx, ny})
				}
			}
			currentLabel++
		}
	}
	return labels
}

func regionDominantColors(img image.Image, bounds image.Rectangle, regionIDs []int, matcher *paletteMatcher) map[int]color.Color {
	counts := make(map[int]map[color.Color]int)
	w := bounds.Dx()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			rid := regionIDs[i]
			if rid < 0 {
				continue
			}
			r, g, b := colorToRGBA8(img.At(bounds.Min.X+x, bounds.Min.Y+y))
			c := matcher.match(r, g, b)
			if counts[rid] == nil {
				counts[rid] = make(map[color.Color]int)
			}
			counts[rid][c]++
		}
	}

	out := make(map[int]color.Color, len(counts))
	for rid, colorCounts := range counts {
		best := color.NRGBA{A: 255}
		bestCount := -1
		for c, n := range colorCounts {
			if n > bestCount {
				bestCount = n
				best = colorToNRGBA(c)
			}
		}
		out[rid] = best
	}
	return out
}

func colorToNRGBA(c color.Color) color.NRGBA {
	r, g, b := colorToRGBA8(c)
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}

func isRegionInterior(ctx *StructureContext, x, y int, edgeInteriorThreshold float64) bool {
	if ctx == nil || ctx.RegionIDs == nil {
		return false
	}
	i := y*ctx.Width + x
	rid := ctx.RegionIDs[i]
	if rid < 0 {
		return false
	}
	if isInBoundaryBand(ctx, x, y) {
		return false
	}
	if ctx.EdgeMap != nil && ctx.EdgeMap[i] > edgeInteriorThreshold {
		return false
	}
	if ctx.EdgeMask != nil && ctx.EdgeMask[i] {
		return false
	}
	return true
}

func localMeanLuminance(img image.Image, bounds image.Rectangle, x, y, radius int) float64 {
	var sum float64
	var count int
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			nx, ny := bounds.Min.X+x+dx, bounds.Min.Y+y+dy
			if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y {
				continue
			}
			r, g, b := colorToRGBA8(img.At(nx, ny))
			sum += pixelLuminance(r, g, b)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func applyStructureOverlayAt(img image.Image, bounds image.Rectangle, out *image.NRGBA, x, y int, black, white color.NRGBA) {
	r, g, b := colorToRGBA8(img.At(bounds.Min.X+x, bounds.Min.Y+y))
	lum := pixelLuminance(r, g, b)
	mean := localMeanLuminance(img, bounds, x, y, 2)
	if lum < mean {
		out.Set(bounds.Min.X+x, bounds.Min.Y+y, black)
	} else {
		out.Set(bounds.Min.X+x, bounds.Min.Y+y, white)
	}
}
