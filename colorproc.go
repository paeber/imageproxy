// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"image"
	"log"

	"github.com/disintegration/imaging"
)

const defaultBinaryThreshold = 128

// applyColorProcessing applies grayscale, binary, or palette transforms to m.
func applyColorProcessing(m image.Image, opt Options) image.Image {
	if opt.Palette != "" {
		palette, err := LookupPalette(opt.Palette)
		if err != nil {
			log.Printf("palette error: %v", err)
			return m
		}
		var structure *StructureContext
		if opt.structureEnabled() {
			structure = buildStructureContext(m, opt)
		}
		src := m
		if opt.SmoothPasses > 0 {
			edgeThreshold := 0.0
			if opt.StructureEdge > 0 {
				edgeThreshold = float64(opt.StructureEdge) / 100.0
			} else if opt.CoverPreset || opt.CoverPMPreset {
				edgeThreshold = float64(defaultCoverEdge) / 100.0
			}
			var edgeMap []float64
			if structure != nil {
				edgeMap = structure.EdgeMap
			} else {
				edgeMap = sobelEdgeMap(m, m.Bounds())
			}
			src = smoothImage(m, opt.SmoothPasses, edgeMap, edgeThreshold)
			if opt.structureEnabled() {
				structure = buildStructureContext(src, opt)
			}
		}
		return MapPalette(src, palette, paletteMapConfigFromOptions(opt, structure))
	}
	if opt.Binary {
		gray := imaging.Grayscale(m)
		threshold := opt.BinaryThreshold
		if threshold == 0 {
			threshold = defaultBinaryThreshold
		}
		return mapToBinary(gray, threshold)
	}
	if opt.Grayscale {
		return imaging.Grayscale(m)
	}
	return m
}
