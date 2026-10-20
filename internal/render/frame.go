// Package render turns game state into pixels (DrawFrame, no tcell) and
// pixels into terminal cells (presenters, which use tcell).
package render

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Pixel is one Frame pixel.
type Pixel struct {
	Gray uint8 // brightness 0-255
	Tint Tint  // color on top of the gray (TintNone = plain gray)
}

// Frame is the pixel buffer for the 3D view. Each terminal cell shows two
// pixels stacked (half-block), so H is twice the number of view rows.
type Frame struct {
	W, H int
	Pix  []Pixel // row-major, index = y*W + x
	// PixelAspect is a pixel's height divided by its width; must be > 0.
	// Usually DefaultPixelAspect; the -aspect flag overrides it for tuning.
	PixelAspect float64
}

// DefaultPixelAspect was measured in T11 from a Windows Terminal screenshot:
// cells are about 14x28 screen pixels, so each half-block pixel is 14x14.
const DefaultPixelAspect = 1.0

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
	// minPerpDist keeps wall height finite when the camera touches a wall.
	minPerpDist = 0.0001
)

// DrawFrame casts one ray per column and fills f with ceiling, wall and floor.
// It doesn't use tcell. A Frame with no pixels is left as is.
func DrawFrame(s *game.State, f *Frame) {
	if f.W <= 0 || f.H <= 0 {
		return
	}
	if f.PixelAspect <= 0 {
		panic("render: Frame.PixelAspect must be > 0") // a bug in the caller, not bad input
	}
	p := s.Player
	dirX, dirY := p.Dir()
	planeX, planeY := p.Plane()

	for x := 0; x < f.W; x++ {
		// cameraX goes from -1 (left edge) to almost +1 (right edge).
		cameraX := 2*float64(x)/float64(f.W) - 1
		hit := world.CastRay(s.Map, p.X, p.Y, dirX+planeX*cameraX, dirY+planeY*cameraX)

		start, end := wallSpan(f, hit.PerpDist)
		wall := Pixel{Gray: wallShade(hit.PerpDist, hit.Side)}
		if hit.Cell == world.Target {
			wall.Tint = TintRed
		}

		for y := 0; y < f.H; y++ {
			px := wall
			if y < start || y >= end {
				px = Pixel{Gray: backgroundShade(y, f.H)} // ceiling above, floor below
			}
			f.Pix[y*f.W+x] = px // sets the tint too, so no color is left from last frame
		}
	}
}

// wallSpan returns the pixel rows [start, end) a wall at perpendicular
// distance dist covers in f, centered vertically.
func wallSpan(f *Frame, dist float64) (start, end int) {
	dist = math.Max(dist, minPerpDist)
	lineHeight := float64(f.W) / (2 * game.PlaneLen * f.PixelAspect * dist)
	center := float64(f.H) / 2
	// Clamp in float before converting: up close lineHeight is huge, and
	// converting an out-of-range float to int gives a garbage value.
	start = int(clamp(math.Round(center-lineHeight/2), 0, float64(f.H)))
	end = int(clamp(math.Round(center+lineHeight/2), 0, float64(f.H)))
	return start, end
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}
