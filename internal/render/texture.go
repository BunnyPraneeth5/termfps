package render

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/assets"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Wall textures, parsed from the built-in assets at startup.
var (
	brickTexture = mustParseImage("brick", assets.Brick)
	stoneTexture = mustParseImage("stone", assets.Stone)
	metalTexture = mustParseImage("metal", assets.Metal)

	floorTexture = mustParseImage("floor", assets.Floor)

	// Their mipmaps (V3), built once at startup.
	brickMips = newMipmap(brickTexture)
	stoneMips = newMipmap(stoneTexture)
	metalMips = newMipmap(metalTexture)
	floorMips = newMipmap(floorTexture)
)

// mipmap is a texture at full size (level 0) and at half, quarter, ...
// size down to 1 pixel (V3). Each level averages 2x2 pixels of the one
// before. Far away, one screen pixel covers many texture pixels; sampling
// just one of them makes patterns flicker as the view moves, so far
// surfaces sample a smaller, averaged level instead.
type mipmap []*Image

// newMipmap builds the levels of an opaque texture.
func newMipmap(img *Image) mipmap {
	m := mipmap{img}
	for prev := img; prev.W > 1 && prev.H > 1; {
		next := &Image{W: prev.W / 2, H: prev.H / 2}
		next.Pix = make([]Pixel, next.W*next.H)
		next.Opaque = make([]bool, len(next.Pix))
		for y := 0; y < next.H; y++ {
			for x := 0; x < next.W; x++ {
				sum := 0
				for _, i := range []int{2*y*prev.W + 2*x, 2*y*prev.W + 2*x + 1, (2*y+1)*prev.W + 2*x, (2*y+1)*prev.W + 2*x + 1} {
					sum += int(prev.Pix[i].Gray)
				}
				next.Pix[y*next.W+x] = Pixel{Gray: uint8((sum + 2) / 4)} // +2 rounds to nearest
				next.Opaque[y*next.W+x] = true
			}
		}
		m = append(m, next)
		prev = next
	}
	return m
}

// level returns the level to sample when one screen pixel covers texels
// pixels of the full-size texture: one level smaller per doubling, so a
// screen pixel covers about one pixel of the level it samples.
func (m mipmap) level(texels float64) *Image {
	k := 0
	for texels >= 2 && k < len(m)-1 {
		texels /= 2
		k++
	}
	return m[k]
}

// texMin is how much of a wall's shade its darkest texture pixel keeps:
// textures add detail without turning walls black. 0.3 since V7 (was
// 0.45), so brick lines, stone gaps and floor tiles read clearly.
const texMin = 0.3

// wallTexture returns the texture (with its mipmap levels) for a wall
// cell, or nil (flat color) for anything else, like red target blocks.
func wallTexture(c world.Cell) mipmap {
	switch c {
	case world.Wall:
		return brickMips
	case world.Stone:
		return stoneMips
	case world.Metal:
		return metalMips
	}
	return nil
}

// wallTexels estimates how many full-size texture pixels one screen pixel
// of a wall column covers: down the column, the texture's height over the
// wall's height in pixels; across, how far apart this column and the
// previous one hit the same wall face. At a grazing angle the second is
// much bigger.
func wallTexels(tex *Image, height float64, hit, prev world.Hit, havePrev bool) float64 {
	texels := float64(tex.H) / height
	if havePrev && hit.Side == prev.Side {
		// An X-side face is the line x = MapX (or MapX+1); along it the
		// position is MapY + WallX. A Y-side face the other way round.
		sameFace := (hit.Side == 0 && hit.MapX == prev.MapX) || (hit.Side == 1 && hit.MapY == prev.MapY)
		if sameFace {
			along := func(h world.Hit) float64 {
				if h.Side == 0 {
					return float64(h.MapY) + h.WallX
				}
				return float64(h.MapX) + h.WallX
			}
			texels = math.Max(texels, math.Abs(along(hit)-along(prev))*float64(tex.W))
		}
	}
	return texels
}

// textureColumn picks the texture column for a hit. Faces seen from the
// "other" side are flipped, so a pattern reads the same way from both
// sides instead of mirrored.
func textureColumn(tex *Image, hit world.Hit, rayDirX, rayDirY float64) int {
	tx := min(int(hit.WallX*float64(tex.W)), tex.W-1)
	if (hit.Side == 0 && rayDirX > 0) || (hit.Side == 1 && rayDirY < 0) {
		tx = tex.W - 1 - tx
	}
	return tx
}

// texturedGray darkens a wall's shade by a texture pixel: a bright texture
// pixel keeps the full shade, a black one keeps texMin of it.
func texturedGray(shade uint8, texGray uint8) uint8 {
	factor := texMin + (1-texMin)*float64(texGray)/252
	return toGray(float64(shade) * factor)
}
