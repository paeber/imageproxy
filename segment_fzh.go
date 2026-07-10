// Copyright 2013 The imageproxy authors.
// SPDX-License-Identifier: Apache-2.0

package imageproxy

import (
	"image"
	"math"
	"sort"
)

type fzhEdge struct {
	a, b int
	w    float64
}

type fzhUnionFind struct {
	parent []int
	rank   []int
	size   []int
}

func newFZHUnionFind(n int) *fzhUnionFind {
	uf := &fzhUnionFind{
		parent: make([]int, n),
		rank:   make([]int, n),
		size:   make([]int, n),
	}
	for i := range uf.parent {
		uf.parent[i] = i
		uf.size[i] = 1
	}
	return uf
}

func (uf *fzhUnionFind) find(x int) int {
	for uf.parent[x] != x {
		uf.parent[x] = uf.parent[uf.parent[x]]
		x = uf.parent[x]
	}
	return x
}

func (uf *fzhUnionFind) union(a, b int) {
	ra, rb := uf.find(a), uf.find(b)
	if ra == rb {
		return
	}
	if uf.rank[ra] < uf.rank[rb] {
		ra, rb = rb, ra
	}
	uf.parent[rb] = ra
	uf.size[ra] += uf.size[rb]
	if uf.rank[ra] == uf.rank[rb] {
		uf.rank[ra]++
	}
}

func fzhThreshold(size int, k float64) float64 {
	return k / float64(size)
}

// felzenszwalbSegment labels pixels using graph-based segmentation (FZH).
// k controls coarseness (higher = fewer, larger regions).
func felzenszwalbSegment(img image.Image, bounds image.Rectangle, k int) []int {
	w, h := bounds.Dx(), bounds.Dy()
	n := w * h
	if n == 0 {
		return nil
	}

	rgb := make([][3]float64, n)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b := colorToRGBA8(img.At(bounds.Min.X+x, bounds.Min.Y+y))
			i := y*w + x
			rgb[i] = [3]float64{float64(r), float64(g), float64(b)}
		}
	}

	edges := make([]fzhEdge, 0, n*4)
	addEdge := func(a, b int) {
		dr := rgb[a][0] - rgb[b][0]
		dg := rgb[a][1] - rgb[b][1]
		db := rgb[a][2] - rgb[b][2]
		wt := math.Sqrt(dr*dr + dg*dg + db*db)
		edges = append(edges, fzhEdge{a, b, wt})
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			if x+1 < w {
				addEdge(i, i+1)
			}
			if y+1 < h {
				addEdge(i, i+w)
			}
		}
	}

	sort.Slice(edges, func(i, j int) bool { return edges[i].w < edges[j].w })

	uf := newFZHUnionFind(n)
	kf := float64(k)
	for _, e := range edges {
		a := uf.find(e.a)
		b := uf.find(e.b)
		if a == b {
			continue
		}
		if e.w <= fzhThreshold(uf.size[a], kf) || e.w <= fzhThreshold(uf.size[b], kf) {
			uf.union(a, b)
		}
	}

	labels := make([]int, n)
	for i := range labels {
		labels[i] = uf.find(i)
	}

	// Renumber labels to contiguous IDs.
	remap := make(map[int]int)
	next := 0
	for i := range labels {
		root := labels[i]
		id, ok := remap[root]
		if !ok {
			id = next
			remap[root] = id
			next++
		}
		labels[i] = id
	}
	return labels
}
