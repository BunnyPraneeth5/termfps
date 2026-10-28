package render

import (
	"fmt"
	"math"
	"slices"

	"github.com/BunnyPraneeth5/termfps/internal/assets"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Enemy and pickup images, parsed from the built-in assets at startup.
var (
	ZombieImage  = mustParseImage("zombie", assets.Zombie)
	ShooterImage = mustParseImage("shooter", assets.Shooter)
	AmmoImage    = mustParseImage("ammo", assets.Ammo)

	barrelImage = mustParseImage("barrel", assets.Barrel)
	crateImage  = mustParseImage("crate", assets.Crate)
	pillarImage = mustParseImage("pillar", assets.Pillar)
	lampImage   = mustParseImage("lamp", assets.Lamp)
	debrisImage = mustParseImage("debris", assets.Debris)
	bodyImage   = mustParseImage("body", assets.Body)
)

// PropSprite returns the sprite for a prop (M2) standing at (x, y): its
// picture and height in cells. It panics for a cell that isn't a prop: the
// caller only passes Map.Props, so that would be a bug.
func PropSprite(kind world.Cell, x, y float64) Sprite {
	s := Sprite{X: x, Y: y}
	switch kind {
	case world.Barrel:
		s.Image, s.Height = barrelImage, 0.55
	case world.Crate:
		s.Image, s.Height = crateImage, 0.6
	case world.Pillar:
		s.Image, s.Height = pillarImage, 1.0
	case world.Lamp, world.FlickerLamp:
		s.Image, s.Height = lampImage, 0.95
	case world.Debris:
		s.Image, s.Height = debrisImage, 0.12
	case world.Body:
		s.Image, s.Height = bodyImage, 0.18
	default:
		panic(fmt.Sprintf("render: no sprite for prop %q", kind))
	}
	return s
}

// Sprite is a flat picture standing in the world (an enemy, a pickup). It
// always faces the camera, like the enemies in Doom and Wolfenstein.
type Sprite struct {
	X, Y   float64 // where it stands, in map cells
	Image  *Image
	Height float64 // height in map cells (1 = as tall as a wall)
	Flash  bool    // draw every opaque pixel white (just got shot)

	dist float64 // distance from the camera, for sorting (set by DrawSprites)
}

// minSpriteDepth skips sprites behind the camera or touching it.
const minSpriteDepth = 0.1

// DrawSprites draws sprites into f as seen by p, far ones first so near
// ones cover them, and skips every column where the wall in the depth
// buffer (filled by DrawFrame) is closer. Call it after DrawFrame and
// before the weapon. It reorders the sprites slice.
func DrawSprites(f *Frame, p game.Player, sprites []Sprite) {
	if f.W <= 0 || f.H <= 0 || len(f.ZBuf) != f.W {
		return
	}
	for i := range sprites {
		dx, dy := sprites[i].X-p.X, sprites[i].Y-p.Y
		sprites[i].dist = dx*dx + dy*dy // squared is enough for ordering
	}
	// slices.SortFunc sorts in place using the compare function: a positive
	// result puts a after b, so this orders far to near.
	slices.SortFunc(sprites, func(a, b Sprite) int {
		switch {
		case a.dist > b.dist:
			return -1
		case a.dist < b.dist:
			return 1
		}
		return 0
	})
	for _, s := range sprites {
		drawSprite(f, p, s)
	}
}

// project returns where a world point appears: depth along the view
// direction and the screen column of its center. It solves
// point = depth*Dir + side*Plane, the same camera DrawFrame uses, so
// sprites line up with walls. ok is false behind (or touching) the camera.
func project(f *Frame, p game.Player, x, y float64) (depth, screenX float64, ok bool) {
	dirX, dirY := p.Dir()
	planeX, planeY := p.Plane()
	rx, ry := x-p.X, y-p.Y
	det := dirX*planeY - planeX*dirY
	depth = (rx*planeY - planeX*ry) / det
	side := (dirX*ry - rx*dirY) / det
	if depth < minSpriteDepth {
		return 0, 0, false
	}
	return depth, float64(f.W) / 2 * (1 + side/depth), true
}

// drawSprite draws one sprite, standing on the floor at its depth.
func drawSprite(f *Frame, p game.Player, s Sprite) {
	depth, centerX, ok := project(f, p, s.X, s.Y)
	if !ok || s.Image == nil {
		return
	}
	focal := float64(f.W) / (2 * game.PlaneLen) // pixels per cell at depth 1, sideways
	wallH := focal / (f.PixelAspect * depth)    // a full wall's height here, in pixels
	pixH := wallH * s.Height
	pixW := focal / depth * s.Height * float64(s.Image.W) / float64(s.Image.H)
	if pixW < 1 || pixH < 1 {
		return
	}
	floorY := float64(f.H)/2 + wallH/2 // the floor line at this depth
	top := floorY - pixH
	left := centerX - pixW/2

	// The same distance fog as walls, and the light of the sprite's cell
	// (M4). No side darkening: sprites always face the camera.
	fog := 1 / (1 + f.fogK()*depth) * f.light.at(int(math.Floor(s.X)), int(math.Floor(s.Y)))

	x0 := max(int(math.Floor(left)), 0)
	x1 := min(int(math.Ceil(left+pixW)), f.W)
	y0 := max(int(math.Floor(top)), 0)
	y1 := min(int(math.Ceil(floorY)), f.H)
	for x := x0; x < x1; x++ {
		if depth >= f.ZBuf[x] {
			continue // a wall in this column is closer
		}
		tx := int((float64(x) - left) / pixW * float64(s.Image.W))
		if tx < 0 || tx >= s.Image.W {
			continue
		}
		for y := y0; y < y1; y++ {
			ty := int((float64(y) - top) / pixH * float64(s.Image.H))
			if ty < 0 || ty >= s.Image.H {
				continue
			}
			i := ty*s.Image.W + tx
			if !s.Image.Opaque[i] {
				continue
			}
			px := s.Image.Pix[i]
			if s.Flash {
				px = Pixel{Gray: 255}
			} else {
				px.Gray = toGray(float64(px.Gray) * fog)
			}
			f.Pix[y*f.W+x] = px
		}
	}
}
