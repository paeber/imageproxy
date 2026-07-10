// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package coverreport

// Scenario describes one transform to run and document in the HTML report.
type Scenario struct {
	ID          string
	Section     string
	Title       string
	Description string
	Options     string // comma-separated options (without size/format); size added by runner
	Sample      string // filename under sample_images/
}

const (
	defaultSample = "beatles.jpg"
	epaperSize    = "800x480"
	outputFormat  = "png"
)

// Scenarios returns the visual regression catalog: one sample cover per setting.
func Scenarios() []Scenario {
	return []Scenario{
		// Baseline
		{
			ID: "baseline-palette", Section: "Baseline",
			Title: "Palette only",
			Description: "Resize to 800×480 and map each pixel to the nearest E1002 color with no dithering. " +
				"Flat color areas stay smooth; gradients band into posterized blocks.",
			Options: "palE1002", Sample: defaultSample,
		},
		{
			ID: "pmrgb-dither-baseline", Section: "pmrgb + dither",
			Title: "pmrgb + dither (recommended baseline)",
			Description: "Weighted RGB palette matching with Floyd–Steinberg dithering. Often the best starting point for album covers.",
			Options: "palE1002,pmrgb,dither", Sample: defaultSample,
		},
		{
			ID: "coverpm-preset", Section: "pmrgb + dither",
			Title: "coverpm preset",
			Description: "pmrgb + dither + ditheredge + regions12 + edge30 + dilate1 without vivid/sat overlay.",
			Options: "palE1002,coverpm", Sample: defaultSample,
		},

		// Dithering
		{
			ID: "dither-classic", Section: "Dithering",
			Title: "Floyd–Steinberg dither",
			Description: "Classic error diffusion adds grain that simulates intermediate tones on the six-color panel.",
			Options: "palE1002,dither", Sample: defaultSample,
		},
		{
			ID: "dither-vivid", Section: "Dithering",
			Title: "Vivid + dither",
			Description: "Boost saturation before mapping, then dither. Useful for muted album art that needs stronger chroma.",
			Options: "palE1002,vivid,dither", Sample: defaultSample,
		},
		{
			ID: "dither-edge", Section: "Dithering",
			Title: "Edge-aware dither (ditheredge)",
			Description: "Floyd–Steinberg diffusion is attenuated across detected edges, reducing color bleed into text and logos.",
			Options: "palE1002,vivid,ditheredge", Sample: defaultSample,
		},
		{
			ID: "dither-smooth", Section: "Dithering",
			Title: "Activity-gated dither (dithersmooth30)",
			Description: "Dither only in smooth regions; high-detail areas (text, edges) stay flat.",
			Options: "palE1002,pmrgb,dither,dithersmooth30", Sample: defaultSample,
		},
		{
			ID: "dither-amount", Section: "Dithering",
			Title: "Reduced dither strength (ditheramt50)",
			Description: "Scales error diffusion to 50% for subtler grain.",
			Options: "palE1002,pmrgb,dither,ditheramt50", Sample: defaultSample,
		},

		// Palette matching
		{
			ID: "match-hue", Section: "Palette matching",
			Title: "Hue priority (pmhue, default)",
			Description: "Light blues map to blue and pinks to red instead of washing out to white. Default for album art.",
			Options: "palE1002,pmhue", Sample: defaultSample,
		},
		{
			ID: "match-lab", Section: "Palette matching",
			Title: "Perceptual Lab (pmlab)",
			Description: "Matches using CIE Lab color distance for perceptually uniform nearest-color selection.",
			Options: "palE1002,pmlab", Sample: defaultSample,
		},
		{
			ID: "match-rgb", Section: "Palette matching",
			Title: "Weighted RGB (pmrgb)",
			Description: "Legacy weighted RGB distance. Mid-tones tend to map toward black more aggressively than hue mode.",
			Options: "palE1002,pmrgb", Sample: defaultSample,
		},
		{
			ID: "palette-bw", Section: "Palette matching",
			Title: "Two-color palette (palBW)",
			Description: "Maps to black and white only using the BW predefined palette.",
			Options: "palBW", Sample: defaultSample,
		},

		// Saturation & vivid
		{
			ID: "sat-low", Section: "Saturation & vivid",
			Title: "Low chroma threshold (sat5)",
			Description: "sat{N} sets the chroma threshold (1–100, default 15). Lower values keep more pastels as chromatic colors.",
			Options: "palE1002,sat5", Sample: defaultSample,
		},
		{
			ID: "sat-default", Section: "Saturation & vivid",
			Title: "Default chroma threshold (sat15)",
			Description: "Default saturation cutoff: moderately desaturated tones may still map to white or black.",
			Options: "palE1002,sat15", Sample: defaultSample,
		},
		{
			ID: "sat-high", Section: "Saturation & vivid",
			Title: "High chroma threshold (sat30)",
			Description: "Higher threshold treats more low-chroma pixels as neutral, pushing washed-out areas to white/black.",
			Options: "palE1002,sat30", Sample: defaultSample,
		},
		{
			ID: "vivid", Section: "Saturation & vivid",
			Title: "Vivid boost",
			Description: "Increases saturation before palette mapping. Recommended for album covers and graphics.",
			Options: "palE1002,vivid", Sample: defaultSample,
		},

		// Regions
		{
			ID: "regions-few", Section: "Regions",
			Title: "Few regions (regions4)",
			Description: "regions{N} segments the image into N color buckets (4–32). Fewer regions yield larger flat color blocks.",
			Options: "palE1002,regions4", Sample: defaultSample,
		},
		{
			ID: "regions-mid", Section: "Regions",
			Title: "Medium regions (regions12)",
			Description: "Balanced segmentation between flat areas and detail retention.",
			Options: "palE1002,regions12", Sample: defaultSample,
		},
		{
			ID: "regions-many", Section: "Regions",
			Title: "Many regions (regions32)",
			Description: "Maximum region count preserves more tonal variation at the cost of less solid fill.",
			Options: "palE1002,regions32", Sample: defaultSample,
		},

		// Structure / edges
		{
			ID: "edge-low", Section: "Structure & edges",
			Title: "Sensitive edges (edge20)",
			Description: "edge{N} sets the structure detection threshold (1–100). Lower values detect finer detail for overlays and dither attenuation.",
			Options: "palE1002,edge20", Sample: defaultSample,
		},
		{
			ID: "edge-high", Section: "Structure & edges",
			Title: "Coarse edges (edge50)",
			Description: "Higher threshold limits structure emphasis to stronger edges only.",
			Options: "palE1002,edge50", Sample: defaultSample,
		},
		{
			ID: "dilate", Section: "Structure & edges",
			Title: "Dilate structure (dilate2)",
			Description: "dilate{N} (0–5) thickens the detected edge mask, making text and logos bolder.",
			Options: "palE1002,dilate2", Sample: defaultSample,
		},
		{
			ID: "erode-dilate", Section: "Structure & edges",
			Title: "Erode then dilate (erode1,dilate2)",
			Description: "erode{N} reduces edge noise before dilation for cleaner structure masks.",
			Options: "palE1002,erode1,dilate2", Sample: defaultSample,
		},

		// Cover preset
		{
			ID: "cover-preset", Section: "Cover preset",
			Title: "Album cover preset (cover)",
			Description: "Balanced preset: vivid, sat10, regions12, ditheredge, edge30, dilate1, and structure overlay. Recommended starting point for music covers.",
			Options: "palE1002,cover", Sample: defaultSample,
		},
		{
			ID: "cover-regions-override", Section: "Cover preset",
			Title: "Cover + fewer regions",
			Description: "Explicit regions8 overrides the cover preset default (regions12) for more solid color areas.",
			Options: "palE1002,cover,regions8", Sample: defaultSample,
		},
		{
			ID: "cover-edge-override", Section: "Cover preset",
			Title: "Cover + sharper edges",
			Description: "edge50 overrides the cover preset default (edge30) for stronger structure detection.",
			Options: "palE1002,cover,edge50", Sample: defaultSample,
		},

		// Enhanced processing
		{
			ID: "fillflat", Section: "Enhanced processing",
			Title: "Flat region fill (fillflat)",
			Description: "Region interiors use a single dominant palette color; dither is confined to boundary bands.",
			Options: "palE1002,pmrgb,dither,regions12,fillflat", Sample: defaultSample,
		},
		{
			ID: "outline", Section: "Enhanced processing",
			Title: "Edge outline (outline)",
			Description: "Morphological gradient edges drawn in black and white on top of the converted image.",
			Options: "palE1002,pmrgb,dither,outline", Sample: defaultSample,
		},
		{
			ID: "fillflat-outline", Section: "Enhanced processing",
			Title: "fillflat + outline",
			Description: "Combines flat region interiors with cartoon-style edge outlines.",
			Options: "palE1002,pmrgb,dither,ditheredge,fillflat,outline", Sample: defaultSample,
		},
		{
			ID: "smooth-passes", Section: "Enhanced processing",
			Title: "Pre-smooth (smooth2)",
			Description: "Two passes of edge-aware separable smoothing before palette mapping.",
			Options: "palE1002,pmrgb,dither,smooth2", Sample: defaultSample,
		},
		{
			ID: "segfzh", Section: "Enhanced processing",
			Title: "Felzenszwalb segmentation (segfzh500)",
			Description: "Graph-based superpixels replace median-cut regions for cleaner region boundaries.",
			Options: "palE1002,pmrgb,dither,segfzh500,fillflat", Sample: defaultSample,
		},
		{
			ID: "protectedge", Section: "Enhanced processing",
			Title: "Protected edge band (protectedge3)",
			Description: "Widens the zero-diffusion band around detected edges to reduce color bleed.",
			Options: "palE1002,pmrgb,dither,ditheredge,protectedge3", Sample: defaultSample,
		},
		{
			ID: "saliency-dither", Section: "Enhanced processing",
			Title: "Saliency-modulated dither",
			Description: "Reduces dither in less salient areas using a center/edge saliency proxy.",
			Options: "palE1002,pmrgb,dither,saliency", Sample: defaultSample,
		},
		{
			ID: "full-enhanced", Section: "Enhanced processing",
			Title: "Full enhanced pipeline",
			Description: "pmrgb + ditheredge + dithersmooth25 + fillflat + outline — quality preset.",
			Options: "palE1002,pmrgb,dither,ditheredge,dithersmooth25,fillflat,outline", Sample: defaultSample,
		},

		// Other color modes
		{
			ID: "grayscale", Section: "Other color modes",
			Title: "Grayscale (gray)",
			Description: "Full grayscale conversion without palette mapping.",
			Options: "gray", Sample: defaultSample,
		},
		{
			ID: "binary-default", Section: "Other color modes",
			Title: "1-bit B/W (bw)",
			Description: "Threshold at 128 converts luminance to pure black or white.",
			Options: "bw", Sample: defaultSample,
		},
		{
			ID: "binary-threshold", Section: "Other color modes",
			Title: "1-bit B/W threshold 160 (bw160)",
			Description: "bw{N} sets the binary threshold (0–255). Higher values map more pixels to black.",
			Options: "bw160", Sample: defaultSample,
		},

		// Input format
		{
			ID: "input-webp", Section: "Input formats",
			Title: "WebP source",
			Description: "WebP decodes correctly and is re-encoded as PNG. Same cover preset applied to a .webp source file.",
			Options: "palE1002,cover", Sample: "686263.webp",
		},
	}
}

// FullOptions returns the complete option string used for Transform.
func (s Scenario) FullOptions() string {
	return epaperSize + "," + s.Options + "," + outputFormat
}
