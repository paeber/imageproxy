// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"image"
	"image/color"
)

const smoothKernelRadius = 3

// smoothImage applies separable box smoothing with optional edge-aware attenuation.
func smoothImage(img image.Image, passes int, edgeMap []float64, edgeThreshold float64) image.Image {
	if passes <= 0 {
		return img
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return img
	}

	cur := imageToFloatRGBA(img, bounds)
	for p := 0; p < passes; p++ {
		cur = separableBoxPass(cur, w, h, edgeMap, edgeThreshold, true)
		cur = separableBoxPass(cur, w, h, edgeMap, edgeThreshold, false)
	}
	return floatRGBAtoImage(cur, bounds)
}

func imageToFloatRGBA(img image.Image, bounds image.Rectangle) [][4]float64 {
	w, h := bounds.Dx(), bounds.Dy()
	out := make([][4]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			i := y*w + x
			out[i] = [4]float64{float64(r >> 8), float64(g >> 8), float64(b >> 8), float64(a >> 8)}
		}
	}
	return out
}

func floatRGBAtoImage(px [][4]float64, bounds image.Rectangle) *image.NRGBA {
	w, h := bounds.Dx(), bounds.Dy()
	out := image.NewNRGBA(bounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			p := px[y*w+x]
			out.Set(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{
				R: clampByte(p[0]),
				G: clampByte(p[1]),
				B: clampByte(p[2]),
				A: clampByte(p[3]),
			})
		}
	}
	return out
}

func separableBoxPass(px [][4]float64, w, h int, edgeMap []float64, edgeThreshold float64, horizontal bool) [][4]float64 {
	out := make([][4]float64, len(px))
	r := smoothKernelRadius

	edgeAt := func(x, y int) float64 {
		if edgeMap == nil || x < 0 || x >= w || y < 0 || y >= h {
			return 0
		}
		return edgeMap[y*w+x]
	}

	blend := func(src, dst [4]float64, x, y int) [4]float64 {
		strength := edgeAt(x, y)
		if edgeThreshold > 0 && strength >= edgeThreshold {
			return src
		}
		if edgeMap != nil && edgeThreshold > 0 {
			f := 1.0 - strength/edgeThreshold
			if f < 0 {
				f = 0
			}
			return [4]float64{
				src[0]*f + dst[0]*(1-f),
				src[1]*f + dst[1]*(1-f),
				src[2]*f + dst[2]*(1-f),
				src[3]*f + dst[3]*(1-f),
			}
		}
		return dst
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			avg, count := boxAverage(px, w, h, x, y, horizontal, r)
			if count == 0 {
				out[y*w+x] = px[y*w+x]
				continue
			}
			out[y*w+x] = blend(px[y*w+x], avg, x, y)
		}
	}
	return out
}

func boxAverage(px [][4]float64, w, h, x, y int, horizontal bool, radius int) ([4]float64, int) {
	var sum [4]float64
	count := 0
	for o := -radius; o <= radius; o++ {
		nx, ny := x, y
		if horizontal {
			nx = x + o
		} else {
			ny = y + o
		}
		if nx < 0 || nx >= w || ny < 0 || ny >= h {
			continue
		}
		p := px[ny*w+nx]
		sum[0] += p[0]
		sum[1] += p[1]
		sum[2] += p[2]
		sum[3] += p[3]
		count++
	}
	if count == 0 {
		return [4]float64{}, 0
	}
	inv := 1.0 / float64(count)
	return [4]float64{sum[0] * inv, sum[1] * inv, sum[2] * inv, sum[3] * inv}, count
}
