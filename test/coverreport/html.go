// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package coverreport

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteHTML writes index.html for the report.
func WriteHTML(report *Report) error {
	sections := groupBySection(report.Results)
	sectionOrder := orderedSections(report.Results)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>imageproxy cover settings report</title>\n")
	b.WriteString("<style>\n")
	b.WriteString(css)
	b.WriteString("</style>\n</head>\n<body>\n")

	b.WriteString("<header><h1>Cover conversion settings</h1>\n")
	b.WriteString(fmt.Sprintf("<p class=\"meta\">Generated %s · %d scenarios · target %s · E1002 ePaper palette</p>\n",
		html.EscapeString(report.GeneratedAt.Format("2006-01-02 15:04 UTC")),
		len(report.Results), epaperSize))
	b.WriteString("<p class=\"meta\">Regenerate: <code>go test -tags coverreport -run TestCoverReport -count=1</code></p>\n")
	b.WriteString("<nav><ul>\n")
	for _, sec := range sectionOrder {
		id := sectionID(sec)
		b.WriteString(fmt.Sprintf("<li><a href=\"#%s\">%s</a></li>\n", id, html.EscapeString(sec)))
	}
	b.WriteString("</ul></nav></header>\n<main>\n")

	for _, sec := range sectionOrder {
		cards := sections[sec]
		id := sectionID(sec)
		b.WriteString(fmt.Sprintf("<section id=\"%s\"><h2>%s</h2>\n", id, html.EscapeString(sec)))
		b.WriteString(sectionIntro(sec))
		b.WriteString("<div class=\"grid\">\n")
		for _, r := range cards {
			writeCard(&b, r)
		}
		b.WriteString("</div></section>\n")
	}

	b.WriteString("</main></body></html>\n")

	outPath := filepath.Join(report.ReportDir, "index.html")
	return os.WriteFile(outPath, []byte(b.String()), 0o644)
}

func groupBySection(results []Result) map[string][]Result {
	m := make(map[string][]Result)
	for _, r := range results {
		m[r.Scenario.Section] = append(m[r.Scenario.Section], r)
	}
	return m
}

func orderedSections(results []Result) []string {
	seen := make(map[string]bool)
	var order []string
	for _, r := range results {
		sec := r.Scenario.Section
		if !seen[sec] {
			seen[sec] = true
			order = append(order, sec)
		}
	}
	return order
}

func sectionID(sec string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(sec, " ", "-"), "&", "and"))
}

func sectionIntro(sec string) string {
	intros := map[string]string{
		"Baseline":           "<p>Starting point: resize to the E1002 panel and map colors without extra processing.</p>",
		"Dithering":            "<p>Dithering simulates tones the hardware cannot display. Compare grain, color bleed, and edge behavior.</p>",
		"Palette matching":     "<p>Controls how each source RGB value is matched to the nearest palette swatch.</p>",
		"Saturation & vivid":   "<p>Tune how aggressively low-chroma pixels are treated as neutral vs chromatic.</p>",
		"Regions":              "<p>Region segmentation posterizes flat areas before palette mapping (4–32 buckets).</p>",
		"Structure & edges":    "<p>Edge detection and morphology emphasize text and logos; pairs with ditheredge in the cover preset.</p>",
		"Cover preset":         "<p>Balanced album-art pipeline; individual options can override preset defaults.</p>",
		"pmrgb + dither":       "<p>Recommended baseline for album covers on E1002 using weighted RGB matching and dithering.</p>",
		"Enhanced processing":  "<p>New structure-aware options: selective dither, flat fill, outlines, smoothing, and FZH segmentation.</p>",
		"Other color modes":    "<p>Non-palette transforms for comparison.</p>",
		"Input formats":        "<p>Decoder support for varied source formats (JPEG, PNG, WebP).</p>",
	}
	if intro, ok := intros[sec]; ok {
		return intro
	}
	return ""
}

func writeCard(b *strings.Builder, r Result) {
	b.WriteString("<article class=\"card\">\n")
	b.WriteString(fmt.Sprintf("<h3>%s</h3>\n", html.EscapeString(r.Scenario.Title)))
	b.WriteString(fmt.Sprintf("<p class=\"opts\"><code>%s</code></p>\n", html.EscapeString(r.Options)))
	b.WriteString(fmt.Sprintf("<p class=\"desc\">%s</p>\n", html.EscapeString(r.Scenario.Description)))
	b.WriteString(fmt.Sprintf("<p class=\"sample\">Source: <code>%s</code></p>\n", html.EscapeString(r.Scenario.Sample)))
	b.WriteString("<div class=\"images\">\n")
	b.WriteString("<figure><figcaption>Original</figcaption>")
	b.WriteString(fmt.Sprintf("<img src=\"%s\" alt=\"original\" loading=\"lazy\"></figure>\n", html.EscapeString(r.OriginalRel)))
	b.WriteString("<figure><figcaption>Converted")
	if r.OutputWidth > 0 && r.OutputHeight > 0 {
		b.WriteString(fmt.Sprintf(" (%d×%d)", r.OutputWidth, r.OutputHeight))
	}
	b.WriteString("</figcaption>")
	b.WriteString(fmt.Sprintf("<img src=\"%s\" alt=\"converted\" loading=\"lazy\"></figure>\n", html.EscapeString(r.OutputRel)))
	b.WriteString("</div></article>\n")
}

// HTMLContainsOutputs returns missing output paths referenced in the written HTML.
func HTMLContainsOutputs(reportDir string, results []Result) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(reportDir, "index.html"))
	if err != nil {
		return nil, err
	}
	html := string(data)
	var missing []string
	refs := make(map[string]bool)
	for _, r := range results {
		refs[r.OutputRel] = true
		refs[r.OriginalRel] = true
	}
	keys := make([]string, 0, len(refs))
	for k := range refs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, rel := range keys {
		if !strings.Contains(html, rel) {
			missing = append(missing, rel)
			continue
		}
		if _, err := os.Stat(filepath.Join(reportDir, filepath.FromSlash(rel))); err != nil {
			missing = append(missing, rel)
		}
	}
	return missing, nil
}

const css = `
:root { color-scheme: light dark; --bg: #1a1a1a; --fg: #e8e8e8; --card: #252525; --border: #444; --accent: #6cb6ff; }
@media (prefers-color-scheme: light) {
  :root { --bg: #f5f5f5; --fg: #222; --card: #fff; --border: #ddd; --accent: #0550ae; }
}
* { box-sizing: border-box; }
body { font-family: system-ui, sans-serif; background: var(--bg); color: var(--fg); margin: 0; line-height: 1.5; }
header, main { max-width: 1200px; margin: 0 auto; padding: 1rem 1.5rem; }
h1 { margin-bottom: 0.25rem; }
.meta { color: #888; font-size: 0.9rem; }
nav ul { display: flex; flex-wrap: wrap; gap: 0.5rem 1rem; padding: 0; list-style: none; }
nav a { color: var(--accent); text-decoration: none; }
section { margin: 2.5rem 0; }
section > p { max-width: 70ch; }
.grid { display: grid; gap: 1.5rem; }
@media (min-width: 900px) { .grid { grid-template-columns: 1fr; } }
.card { background: var(--card); border: 1px solid var(--border); border-radius: 8px; padding: 1rem 1.25rem; }
.card h3 { margin: 0 0 0.5rem; font-size: 1.1rem; }
.opts code { font-size: 0.85rem; word-break: break-all; }
.desc { font-size: 0.95rem; }
.sample { font-size: 0.85rem; color: #888; }
.images { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-top: 0.75rem; }
@media (max-width: 700px) { .images { grid-template-columns: 1fr; } }
figure { margin: 0; }
figcaption { font-size: 0.8rem; margin-bottom: 0.35rem; color: #888; }
img { max-width: 100%; height: auto; border: 1px solid var(--border); border-radius: 4px; display: block; }
`
