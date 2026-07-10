// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkCoverPipeline_pmrgb_dither(b *testing.B) {
	benchmarkCoverPipeline(b, "palE1002,pmrgb,dither,png")
}

func BenchmarkCoverPipeline_coverpm(b *testing.B) {
	benchmarkCoverPipeline(b, "palE1002,coverpm,png")
}

func BenchmarkCoverPipeline_full(b *testing.B) {
	benchmarkCoverPipeline(b, "palE1002,pmrgb,dither,ditheredge,dithersmooth25,fillflat,outline,png")
}

func benchmarkCoverPipeline(b *testing.B, options string) {
	sample := filepath.Join("test", "sample_images", "beatles.jpg")
	in, err := os.ReadFile(sample)
	if err != nil {
		b.Skip("sample image not available")
	}
	opt := ParseOptions("800x480," + options)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := Transform(in, opt)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := png.Decode(bytes.NewReader(out)); err != nil {
			b.Fatal(err)
		}
	}
}
