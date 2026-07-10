// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package coverreport

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"time"

	_ "golang.org/x/image/webp"
)

// TransformFunc applies imageproxy options to encoded image bytes.
type TransformFunc func(in []byte, options string) (out []byte, width, height int, err error)

// Result holds one generated scenario output.
type Result struct {
	Scenario     Scenario
	Options      string
	SamplePath   string
	OriginalRel  string
	OutputRel    string
	OutputBytes  []byte
	OutputWidth  int
	OutputHeight int
}

// Report is the full generation output passed to the HTML writer.
type Report struct {
	GeneratedAt time.Time
	SampleDir   string
	ReportDir   string
	Results     []Result
}

// Generate runs all scenarios and writes PNG outputs plus originals into reportDir.
func Generate(sampleDir, reportDir string, transform TransformFunc) (*Report, error) {
	if err := os.MkdirAll(filepath.Join(reportDir, "output"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(reportDir, "originals"), 0o755); err != nil {
		return nil, err
	}

	report := &Report{
		GeneratedAt: time.Now().UTC(),
		SampleDir:   sampleDir,
		ReportDir:   reportDir,
	}

	seenOriginals := make(map[string]string)

	for _, sc := range Scenarios() {
		samplePath := filepath.Join(sampleDir, sc.Sample)
		in, err := os.ReadFile(samplePath)
		if err != nil {
			return nil, fmt.Errorf("read sample %q: %w", sc.Sample, err)
		}

		opts := sc.FullOptions()
		out, w, h, err := transform(in, opts)
		if err != nil {
			return nil, fmt.Errorf("transform %q (%s): %w", sc.ID, opts, err)
		}

		origRel, ok := seenOriginals[sc.Sample]
		if !ok {
			origRel = filepath.ToSlash(filepath.Join("originals", sc.Sample))
			if err := os.WriteFile(filepath.Join(reportDir, origRel), in, 0o644); err != nil {
				return nil, fmt.Errorf("write original %q: %w", sc.Sample, err)
			}
			seenOriginals[sc.Sample] = origRel
		}

		outRel := filepath.ToSlash(filepath.Join("output", sc.ID+".png"))
		if err := os.WriteFile(filepath.Join(reportDir, outRel), out, 0o644); err != nil {
			return nil, fmt.Errorf("write output %q: %w", sc.ID, err)
		}

		report.Results = append(report.Results, Result{
			Scenario:     sc,
			Options:      opts,
			SamplePath:   samplePath,
			OriginalRel:  origRel,
			OutputRel:    outRel,
			OutputBytes:  out,
			OutputWidth:  w,
			OutputHeight: h,
		})
	}

	return report, nil
}

// DecodeOutputConfig returns image dimensions from encoded bytes.
func DecodeOutputConfig(out []byte) (width, height int, err error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
