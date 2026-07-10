// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

//go:build coverreport

package imageproxy_test

import (
	"os"
	"path/filepath"
	"testing"

	"willnorris.com/go/imageproxy"
	"willnorris.com/go/imageproxy/test/coverreport"
)

func TestCoverReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cover report in short mode")
	}

	sampleDir := filepath.Join("test", "sample_images")
	if !hasSampleImages(sampleDir) {
		t.Skip("no sample images in test/sample_images")
	}

	reportDir := filepath.Join("test", "report")
	report, err := coverreport.Generate(sampleDir, reportDir, transformCover)
	if err != nil {
		t.Fatal(err)
	}

	if err := coverreport.WriteHTML(report); err != nil {
		t.Fatal(err)
	}

	for _, r := range report.Results {
		if err := coverreport.ValidateDecode(r); err != nil {
			t.Errorf("scenario %q: %v", r.Scenario.ID, err)
		}
		if err := validatePaletteOutput(r); err != nil {
			t.Errorf("scenario %q: %v", r.Scenario.ID, err)
		}
	}

	missing, err := coverreport.HTMLContainsOutputs(reportDir, report.Results)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) > 0 {
		t.Errorf("HTML report missing assets: %v", missing)
	}

	t.Logf("cover report written to %s", filepath.Join(reportDir, "index.html"))
}

func transformCover(in []byte, options string) ([]byte, int, int, error) {
	opt := imageproxy.ParseOptions(options)
	out, err := imageproxy.Transform(in, opt)
	if err != nil {
		return nil, 0, 0, err
	}
	w, h, err := coverreport.DecodeOutputConfig(out)
	if err != nil {
		return nil, 0, 0, err
	}
	return out, w, h, nil
}

func validatePaletteOutput(r coverreport.Result) error {
	opt := imageproxy.ParseOptions(r.Options)
	if opt.Palette == "" {
		return nil
	}
	palette, err := imageproxy.LookupPalette(opt.Palette)
	if err != nil {
		return err
	}
	// Validation uses image decode in coverreport; palette check here via test helper
	return validatePixelsInPalette(r.OutputBytes, palette.Colors)
}

func hasSampleImages(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == ".gitignore" {
			continue
		}
		return true
	}
	return false
}
