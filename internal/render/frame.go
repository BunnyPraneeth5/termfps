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
	// Scale is how many screen cells across and down each Frame cell fills
	// (V6): 1 (or 0) draws the view at full resolution; 2 at half, each cell
	// shown as a 2x2 block, so neighboring cells share colors and the
	// terminal gets fewer color codes. Size the Frame with ScaledFrameSize.
	Scale int

	// Set by DrawFrame for the map being drawn (M4), and used by the
	// sprites after it: the fog factor and the light grid.
	fog   float64
	light lightGrid
	rows  rowCache
	// acid is the map being drawn when it has acid floor (M14), else nil.
	acid *world.Map
}

// rowCache holds, for every pixel row, what only depends on the row (V8),
// worked out once per frame instead of once per pixel: whether it's above
// the middle (ceiling), the flat background shade, and for the cast floor
// its distance, fog shade and texture level. Reused between frames.
type rowCache struct {
	ceiling []bool
	flat    []uint8
	dist    []float64
	shade   []float64
	tex     []*Image
}

// prepareRows fills f.rows for the current size, fog and light.
func (f *Frame) prepareRows() {
	r := &f.rows
	if cap(r.flat) < f.H {
		*r = rowCache{ceiling: make([]bool, f.H), flat: make([]uint8, f.H), dist: make([]float64, f.H),
			shade: make([]float64, f.H), tex: make([]*Image, f.H)}
	}
	r.ceiling, r.flat, r.dist, r.shade, r.tex = r.ceiling[:f.H], r.flat[:f.H], r.dist[:f.H], r.shade[:f.H], r.tex[:f.H]
	// Texture pixels one screen pixel covers (V3): across a row, neighbor
	// columns' rays are 2*PlaneLen/W apart, so their points are that times
	// d apart; down a column, the distance changes by d*d/K per row (K as
	// in castDist: d = K/offset). Far away the second is much bigger.
	k := float64(f.W) / (4 * game.PlaneLen * f.PixelAspect)
	for y := range f.H {
		d, ceiling := castDist(f, y)
		r.ceiling[y], r.dist[y] = ceiling, d
		r.flat[y] = backgroundShade(y, f.H)
		if !ceiling && f.light.on {
			// The flat floor has no world position: dark maps just dim it.
			r.flat[y] = toGray(float64(r.flat[y]) * f.light.ambient)
		}
		r.shade[y] = floorBase / (1 + f.fogK()*d)
		r.tex[y] = floorMips.level(math.Max(2*game.PlaneLen*d/float64(f.W), d*d/k) * float64(floorMips[0].W))
	}
}

// fogK returns the Frame's fog factor: the map's "; fog:" header, or the
// default fogK for maps without one (and Frames DrawFrame hasn't drawn).
func (f *Frame) fogK() float64 {
	if f.fog > 0 {
		return f.fog
	}
	return fogK
}

// AutoScaleCols is the widest the 3D view is drawn with the "auto" render
// scale: wider terminals get a bigger Scale (V6).
const AutoScaleCols = 160

// AutoScale is the render scale "auto" picks for a terminal cols wide:
// 1 up to AutoScaleCols, then 2, 3, ... so the view stays at most that
// many columns of detail.
func AutoScale(cols int) int {
	return max(1, (cols+AutoScaleCols-1)/AutoScaleCols) // rounds up
}

// ScaledFrameSize is FrameSize for a Frame drawn at scale: each Frame cell
// covers scale x scale screen cells, rounded up so the view is covered.
func ScaledFrameSize(cols, rows, scale int) (w, h int) {
	scale = max(scale, 1)
	viewRows := (rows - 1 + scale - 1) / scale
	return (cols + scale - 1) / scale, 2 * viewRows
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
	f.fog = s.Map.Fog
	f.acid = nil
	if s.Map.HasAcid() {
		f.acid = s.Map
	}
	f.light.prepare(s.Map, s.Time)
	f.prepareRows()
	p := s.Player
	dirX, dirY := p.Dir()
	planeX, planeY := p.Plane()
	var prev world.Hit // the previous column's hit, for texture filtering
	// colShade holds, for the current column, the shaded gray of each texture
	// row (V8), so wall pixels look it up instead of redoing the math.
	// Textures are at most 32 tall; 64 leaves room.
	var colShade [64]uint8

	for x := 0; x < f.W; x++ {
		// cameraX goes from -1 (left edge) to almost +1 (right edge).
		cameraX := 2*float64(x)/float64(f.W) - 1
		rayX, rayY := dirX+planeX*cameraX, dirY+planeY*cameraX
		hit := world.CastRay(s.Map, p.X, p.Y, rayX, rayY)

		f.ZBuf[x] = hit.PerpDist
		top, height := wallLine(f, hit.PerpDist)
		bottom := top + height
		wall := Pixel{Gray: wallShadeFog(hit.PerpDist, hit.Side, f.fogK())}
		if f.light.on && !math.IsInf(hit.PerpDist, 1) {
			// A wall gets the light of the cell in front of it (M4).
			back := hit.PerpDist - 0.01
			l := f.light.at(int(math.Floor(p.X+rayX*back)), int(math.Floor(p.Y+rayY*back)))
			wall.Gray = toGray(float64(wall.Gray) * l)
		}
		if hit.Cell == world.Target {
			wall.Tint = TintRed
		}
		var tex *Image
		tx := 0
		if f.Textures {
			if mips := wallTexture(hit.Cell); mips != nil {
				tex = mips.level(wallTexels(mips[0], height, hit, prev, x > 0))
				tx = textureColumn(tex, hit, rayX, rayY)
			}
		}
		prev = hit

		// The wall touches pixel rows yTop to yEnd-1: the first and last
		// only partly (V2 blends those). Clamp in float before converting:
		// up close the wall is huge, and an out-of-range float -> int
		// conversion gives a garbage value.
		yTop := int(clamp(math.Floor(top), 0, float64(f.H)))
		yEnd := int(clamp(math.Ceil(bottom), 0, float64(f.H)))
		var texPerRow, texStart float64
		if tex != nil {
			for t := range tex.H {
				colShade[t] = texturedGray(wall.Gray, tex.Pix[t*tex.W+tx].Gray)
			}
			// The texture row of pixel row y is int((y + 0.5 - top) / height * tex.H).
			texPerRow = float64(tex.H) / height
			texStart = (0.5 - top) * texPerRow
		}
		for y := 0; y < yTop; y++ {
			f.Pix[y*f.W+x] = backgroundPixel(f, p, rayX, rayY, y) // ceiling, or floor
		}
		for y := yTop; y < yEnd; y++ {
			px := wall
			if tex != nil {
				ty := min(max(int(float64(y)*texPerRow+texStart), 0), tex.H-1)
				px.Gray = colShade[ty]
			}
			if y == yTop || y == yEnd-1 {
				// An edge pixel: mix the wall with the floor or ceiling behind
				// it by how much of the row the wall covers, so wall edges
				// don't step from pixel to pixel.
				if cov := coverage(y, top, bottom); cov < 1 {
					px = blendPixel(backgroundPixel(f, p, rayX, rayY, y), px, cov)
				}
			}
			f.Pix[y*f.W+x] = px // sets the tint too, so no color is left from last frame
		}
		for y := yEnd; y < f.H; y++ {
			f.Pix[y*f.W+x] = backgroundPixel(f, p, rayX, rayY, y) // floor
		}
	}
}

// backgroundPixel is the ceiling (above the middle) or floor (below) in
// row y of the column whose ray is (rayX, rayY). With textures on, the
// floor is cast and textured; the ceiling is always flat near-black (the
// user's choice after V1: more contrast for the walls, fewer color codes).
func backgroundPixel(f *Frame, p game.Player, rayX, rayY float64, y int) Pixel {
	if f.rows.ceiling[y] {
		return Pixel{Gray: f.rows.flat[y]}
	}
	px := Pixel{Gray: f.rows.flat[y]}
	if f.Textures {
		px = castPixel(f, p, rayX, rayY, y)
	}
	if f.acid != nil {
		d := f.rows.dist[y]
		if f.acid.At(int(math.Floor(p.X+rayX*d)), int(math.Floor(p.Y+rayY*d))) == world.Acid {
			// Acid (M14) glows a little, so it shows on dark maps too.
			px = Pixel{Gray: toGray(math.Max(float64(px.Gray), acidGlow*f.rows.shade[y]/floorBase)), Tint: TintGreen}
		}
	}
	return px
}

// acidGlow is the least brightness of acid floor up close (M14).
const acidGlow = 120

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

// castPixel returns the floor pixel in row y (below the middle) of the
// column whose ray is (rayX, rayY), for player p (V1, Lodev's floor
// casting).
//
// A floor point at perpendicular distance d is where the bottom of a wall
// at distance d would be drawn. wallLine makes that wall W/(2*PlaneLen*
// aspect*d) pixels tall around the middle row, so a pixel row offset rows
// below the middle sees the floor at d = W/(4*PlaneLen*aspect*offset). The
// ray's direction has length 1 along the view direction, so the point is
// player + ray*d.
func castPixel(f *Frame, p game.Player, rayX, rayY float64, y int) Pixel {
	d, tex := f.rows.dist[y], f.rows.tex[y] // worked out once per row (V8)
	wx, wy := p.X+rayX*d, p.Y+rayY*d
	// The fraction of the world position inside its cell picks the texel.
	tx := min(int((wx-math.Floor(wx))*float64(tex.W)), tex.W-1)
	ty := min(int((wy-math.Floor(wy))*float64(tex.H)), tex.H-1)
	shade := toGray(f.rows.shade[y] * f.light.smooth(wx, wy)) // fog (per row) and lamp light (M4)
	return Pixel{Gray: texturedGray(shade, tex.Pix[ty*tex.W+tx].Gray)}
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
