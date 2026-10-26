package render

import (
	"github.com/BunnyPraneeth5/termfps/internal/assets"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Wall textures, parsed from the built-in assets at startup.
var (
	brickTexture = mustParseImage("brick", assets.Brick)
	stoneTexture = mustParseImage("stone", assets.Stone)
	metalTexture = mustParseImage("metal", assets.Metal)

	floorTexture   = mustParseImage("floor", assets.Floor)
	ceilingTexture = mustParseImage("ceiling", assets.Ceiling)
)

// texMin is how much of a wall's shade its darkest texture pixel keeps:
// textures add detail without turning walls black.
const texMin = 0.45

// wallTexture returns the texture for a wall cell, or nil (flat color) for
// anything else, like red target blocks.
func wallTexture(c world.Cell) *Image {
	switch c {
	case world.Wall:
		return brickTexture
	case world.Stone:
		return stoneTexture
	case world.Metal:
		return metalTexture
	}
	return nil
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
