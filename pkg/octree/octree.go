package octree

import (
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/pechorka/simple-raytracer/pkg/utils"
)

const uint8MSB = 7

type levelsCache [8][]*node

type Root struct {
	root   *node
	levels levelsCache
}

type node struct {
	id        int
	parent    *node
	children  [8]*node
	pixelIdxs []int
	sumR      uint32
	sumG      uint32
	sumB      uint32
}

func NewRoot() *Root {
	return &Root{}
}

func (r *Root) Insert(pixel uint32, pixelIdx int) error {
	var path [uint8MSB]int

	msb := uint8MSB
	r8, g, b, _ := utils.RGBAFromPixel(pixel)
	for msb > 0 {
		// R: 11001000
		// G: 01100100 -> 100
		// B: 00110010
		path[uint8MSB-msb] =
			((int(((r8 & (1 << msb)) >> msb)) << 2) |
				(int(((g & (1 << msb)) >> msb)) << 1) |
				(int(((b & (1 << msb)) >> msb)) << 0))
		msb--
	}
	if r.root == nil {
		newNode := &node{}
		r.root = newNode
		r.levels[0] = append(r.levels[0], newNode)
	}

	return r.root.insert(pixel, pixelIdx, path[:], &r.levels)
}

func (r *Root) reduce(to int) error {
	nodeCount := 0
	for _, lvl := range r.levels {
		nodeCount += len(lvl)
	}

	if nodeCount <= to {
		return nil
	}

	reduceNodes := nodeCount - to
	for i := 7; i > 0; i-- {
		level := r.levels[i]
		slices.SortFunc(level, func(n1, n2 *node) int {
			return len(n1.pixelIdxs) - len(n2.pixelIdxs)
		})

		j := 0
		for ; j < len(level) && reduceNodes > 0; j++ {
			n := level[j]
			if n.parent == nil {
				return fmt.Errorf("node %d has nil parent while reducing octree", n.id)
			}
			n.parent.sumR += n.sumR
			n.parent.sumG += n.sumG
			n.parent.sumB += n.sumB
			n.parent.pixelIdxs = append(n.parent.pixelIdxs, n.pixelIdxs...)
			n.parent.children[n.id] = nil
			reduceNodes--
		}
		r.levels[i] = r.levels[i][j:]
	}

	return nil
}

func (r *Root) ToImage(frameRect image.Rectangle) (*image.Paletted, error) {
	if r == nil || r.root == nil {
		return nil, fmt.Errorf("octree root is empty")
	}

	res := image.NewPaletted(frameRect, nil)

	const maxPaletteLen = 256
	if err := r.reduce(maxPaletteLen); err != nil {
		return nil, err
	}
	palette := make(color.Palette, 0, maxPaletteLen)
	for _, lvl := range r.levels {
		for _, n := range lvl {
			if len(n.pixelIdxs) == 0 {
				continue
			}

			for _, pixelIdx := range n.pixelIdxs {
				res.Pix[pixelIdx] = uint8(len(palette))
			}

			palette = append(palette, color.RGBA{
				R: uint8(n.sumR / uint32(len(n.pixelIdxs))),
				G: uint8(n.sumG / uint32(len(n.pixelIdxs))),
				B: uint8(n.sumB / uint32(len(n.pixelIdxs))),
			})

			if len(palette) >= maxPaletteLen {
				return nil, fmt.Errorf("palette should contain at most %d colors", maxPaletteLen)
			}
		}
	}

	if len(palette) == 0 {
		return nil, fmt.Errorf("octree has no colors")
	}

	res.Palette = palette
	return res, nil
}

func (n *node) insert(pixel uint32, pixelIdx int, path []int, levels *levelsCache) error {
	if len(path) == 0 {
		r, g, b, _ := utils.RGBAFromPixel(pixel)
		n.sumR += uint32(r)
		n.sumG += uint32(g)
		n.sumB += uint32(b)
		n.pixelIdxs = append(n.pixelIdxs, pixelIdx)
		return nil
	}

	ci := path[0]
	if ci >= len(n.children) {
		return fmt.Errorf("path element %d is larger than max child index", ci)
	}
	if n.children[ci] == nil {
		newNode := &node{id: ci, parent: n}
		n.children[ci] = newNode
		depth := uint8MSB - len(path) + 1
		(*levels)[depth] = append((*levels)[depth], newNode)
	}

	return n.children[ci].insert(pixel, pixelIdx, path[1:], levels)
}
