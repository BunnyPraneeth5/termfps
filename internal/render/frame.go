// Package render turns game state into pixels (DrawFrame, no tcell) and
// pixels into terminal cells (presenters, which use tcell).
package render

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Pixel is one Frame pixel. T20 (stretch) adds a Color field.
type Pixel struct {
	Gray uint8 // brightness 0-255
}

// Frame is the pixel buffer for the 3D view. Each terminal cell shows two
// pixels stacked (half-block), so H is twice the number of view rows.
type Frame struct {
	W, H int
	Pix  []Pixel // row-major, index = y*W + x
}

// FrameSize returns the Frame size for a terminal of cols x rows.
// The last row is the HUD, so the view gets rows-1 cells = 2*(rows-1) pixels.
func FrameSize(cols, rows int) (w, h int) {
	return cols, 2 * (rows - 1)
}

// Resize sets the Frame to w x h. It reuses the existing pixel slice when it
// is big enough, so resizing smaller (or to the same size) never allocates.
func (f *Frame) Resize(w, h int) {
	n := max(w*h, 0)
	if cap(f.Pix) >= n {
		f.Pix = f.Pix[:n] // re-slice: same backing array, new length
	} else {
		f.Pix = make([]Pixel, n)
	}
	f.W, f.H = w, h
}

const (
	// pixelAspect is a Frame pixel's height divided by its width. Two pixels
	// per terminal cell makes them about square. Tuned by eye in T11.
	pixelAspect = 1.0
	// minPerpDist keeps wall height finite when the camera touches a wall.
	minPerpDist = 0.0001
	// wallGray is the flat wall color until shading (T12).
	wallGray uint8 = 200
	// floorGray and ceilingGray are flat until T13.
	floorGray   uint8 = 0
	ceilingGray uint8 = 0
)

// DrawFrame casts one ray per column and fills f with ceiling, wall and floor.
// It doesn't use tcell. A Frame with no pixels is left as is.
func DrawFrame(s *game.State, f *Frame) {
	if f.W <= 0 || f.H <= 0 {
		return
	}
	p := s.Player
	dirX, dirY := p.Dir()
	planeX, planeY := p.Plane()
	center := float64(f.H) / 2

	for x := 0; x < f.W; x++ {
		// cameraX goes from -1 (left edge) to almost +1 (right edge).
		cameraX := 2*float64(x)/float64(f.W) - 1
		hit := world.CastRay(s.Map, p.X, p.Y, dirX+planeX*cameraX, dirY+planeY*cameraX)

		dist := math.Max(hit.PerpDist, minPerpDist)
		lineHeight := float64(f.W) / (2 * game.PlaneLen * pixelAspect * dist)

		// Clamp in float before converting: up close lineHeight is huge, and
		// converting an out-of-range float to int gives a garbage value.
		start := int(clamp(math.Round(center-lineHeight/2), 0, float64(f.H)))
		end := int(clamp(math.Round(center+lineHeight/2), 0, float64(f.H)))

		for y := 0; y < f.H; y++ {
			g := ceilingGray
			switch {
			case y >= end:
				g = floorGray
			case y >= start:
				g = wallGray
			}
			f.Pix[y*f.W+x].Gray = g
		}
	}
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}
