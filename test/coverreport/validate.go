// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package coverreport

import (
	"bytes"
	"fmt"
	"image"
)

// ValidateDecode ensures output bytes form a decodable image with expected width when set.
func ValidateDecode(r Result) error {
	img, _, err := image.Decode(bytes.NewReader(r.OutputBytes))
	if err != nil {
		return fmt.Errorf("decode output: %w", err)
	}
	if r.OutputWidth > 0 && img.Bounds().Dx() != r.OutputWidth {
		return fmt.Errorf("width mismatch: got %d want %d", img.Bounds().Dx(), r.OutputWidth)
	}
	return nil
}
