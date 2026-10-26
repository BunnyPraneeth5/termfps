// Package render turns game state into pixels (DrawFrame, no screen) and
// pixels into terminal cells (presenters, which draw on a term.Screen).
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
	// ZBuf is the depth buffer: for each column, the distance to the wall
	// DrawFrame drew there. Sprites behind that wall are skipped.
	ZBuf []float64
	// Textures turns wall textures on (P1-10). Off gives flat gray walls.
	Textures bool
	// MixedBackground is how bright a Mixed-mode cell's background is, as a
	// fraction (0-1) of its character's color (S3). Usually
	// DefaultMixedBackground; the Settings menu changes it.
	MixedBackground float64
	// ColorThreshold lets the truecolor presenters (half-block, mixed) reuse
	// the color of the cell to the left when no red, green or blue channel
	// differs by more than this (0 = off), so fewer color codes go to the
	// terminal (S3, from S0ra's libasciic).
	ColorThreshold uint8
}

// DefaultMixedBackground is the Mixed background brightness (20%, P1-15).
const DefaultMixedBackground = 0.2

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
	cols := max(w, 0)
	if cap(f.ZBuf) >= cols {
		f.ZBuf = f.ZBuf[:cols]
	} else {
		f.ZBuf = make([]float64, cols)
	}
	f.W, f.H = w, h
}

const (
	// minPerpDist keeps wall height finite when the camera touches a wall.
	minPerpDist = 0.0001
)

// DrawFrame casts one ray per column and fills f with ceiling, wall and floor.
// It doesn't touch the screen. A Frame with no pixels is left as is.
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
		rayX, rayY := dirX+planeX*cameraX, dirY+planeY*cameraX
		hit := world.CastRay(s.Map, p.X, p.Y, rayX, rayY)

		f.ZBuf[x] = hit.PerpDist
		top, height := wallLine(f, hit.PerpDist)
		bottom := top + height
		wall := Pixel{Gray: wallShade(hit.PerpDist, hit.Side)}
		if hit.Cell == world.Target {
			wall.Tint = TintRed
		}
		var tex *Image
		tx := 0
		if f.Textures {
			if tex = wallTexture(hit.Cell); tex != nil {
				tx = textureColumn(tex, hit, rayX, rayY)
			}
		}

		for y := 0; y < f.H; y++ {
			// cov is how much of this pixel row the wall covers: 1 inside the
			// wall, 0 above or below it, in between on its top and bottom
			// edge (V2).
			cov := coverage(y, top, bottom)
			if cov <= 0 {
				f.Pix[y*f.W+x] = backgroundPixel(f, p, rayX, rayY, y)
				continue
			}
			px := wall
			if tex != nil {
				// Where this pixel falls on the wall, 0 (top) to 1 (bottom).
				v := (float64(y) + 0.5 - top) / height
				ty := min(max(int(v*float64(tex.H)), 0), tex.H-1)
				px.Gray = texturedGray(wall.Gray, tex.Pix[ty*tex.W+tx].Gray)
			}
			if cov < 1 {
				// An edge pixel: mix the wall with the floor or ceiling behind
				// it, so wall edges don't step from pixel to pixel.
				px = blendPixel(backgroundPixel(f, p, rayX, rayY, y), px, cov)
			}
			f.Pix[y*f.W+x] = px // sets the tint too, so no color is left from last frame
		}
	}
}

// backgroundPixel is the ceiling (above the middle) or floor (below) in
// row y of the column whose ray is (rayX, rayY): cast and textured with
// textures on, the flat gradient with them off.
func backgroundPixel(f *Frame, p game.Player, rayX, rayY float64, y int) Pixel {
	if f.Textures {
		return castPixel(f, p, rayX, rayY, y)
	}
	return Pixel{Gray: backgroundShade(y, f.H)}
}

// coverage returns how much of pixel row y (from y to y+1) lies between
// top and bottom, from 0 to 1.
func coverage(y int, top, bottom float64) float64 {
	fy := float64(y)
	return clamp(math.Min(fy+1, bottom)-math.Max(fy, top), 0, 1)
}

// blendPixel mixes front over back: cov = 1 is all front, 0 all back. The
// tint goes with whichever covers most of the pixel.
func blendPixel(back, front Pixel, cov float64) Pixel {
	out := Pixel{Gray: toGray(float64(back.Gray)*(1-cov) + float64(front.Gray)*cov), Tint: back.Tint}
	if cov >= 0.5 {
		out.Tint = front.Tint
	}
	return out
}

// castPixel returns the floor (below the middle) or ceiling (above) pixel
// in row y of the column whose ray is (rayX, rayY), for player p (V1,
// Lodev's floor casting).
//
// A floor point at perpendicular distance d is where the bottom of a wall
// at distance d would be drawn. wallLine makes that wall W/(2*PlaneLen*
// aspect*d) pixels tall around the middle row, so a pixel row offset rows
// below the middle sees the floor at d = W/(4*PlaneLen*aspect*offset). The
// ray's direction has length 1 along the view direction, so the point is
// player + ray*d. The ceiling is the mirror image above the middle.
func castPixel(f *Frame, p game.Player, rayX, rayY float64, y int) Pixel {
	d, ceiling := castDist(f, y)
	tex, base := floorTexture, floorBase
	if ceiling {
		tex, base = ceilingTexture, ceilingBase
	}
	wx, wy := p.X+rayX*d, p.Y+rayY*d
	// The fraction of the world position inside its cell picks the texel.
	tx := min(int((wx-math.Floor(wx))*float64(tex.W)), tex.W-1)
	ty := min(int((wy-math.Floor(wy))*float64(tex.H)), tex.H-1)
	return Pixel{Gray: texturedGray(castShade(base, d), tex.Pix[ty*tex.W+tx].Gray)}
}

// castDist returns the perpendicular distance of the floor (or, above the
// middle row, the ceiling) seen in pixel row y. See castPixel.
func castDist(f *Frame, y int) (d float64, ceiling bool) {
	offset := float64(y) + 0.5 - float64(f.H)/2
	if offset < 0 {
		offset, ceiling = -offset, true
	}
	return float64(f.W) / (4 * game.PlaneLen * f.PixelAspect * offset), ceiling
}

// wallLine returns where a wall at perpendicular distance dist starts
// (top, in pixel rows, may be negative up close) and how tall it is, before
// clipping to the Frame.
func wallLine(f *Frame, dist float64) (top, height float64) {
	dist = math.Max(dist, minPerpDist)
	height = float64(f.W) / (2 * game.PlaneLen * f.PixelAspect * dist)
	return float64(f.H)/2 - height/2, height
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(math.Max(v, lo), hi)
}

// Dim darkens every pixel to a third, for the paused game behind a menu.
func Dim(f *Frame) {
	for i := range f.Pix {
		f.Pix[i].Gray /= 3
	}
}
