package render

import (
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/assets"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

func TestTextureAssetsParse(t *testing.T) {
	for name, data := range map[string][]byte{"brick": assets.Brick, "stone": assets.Stone, "metal": assets.Metal} {
		img, err := ParseImage(data)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if img.W != 16 || img.H != 16 {
			t.Errorf("%s is %dx%d, want 16x16", name, img.W, img.H)
		}
		for i, ok := range img.Opaque {
			if !ok {
				t.Errorf("%s has a transparent pixel at %d; textures must be solid", name, i)
				break
			}
		}
	}
}

func TestWallTextureByCell(t *testing.T) {
	if wallTexture(world.Wall) != brickTexture || wallTexture(world.Stone) != stoneTexture || wallTexture(world.Metal) != metalTexture {
		t.Error("wrong texture for a wall type")
	}
	if wallTexture(world.Target) != nil {
		t.Error("target blocks should stay flat red")
	}
}

func TestTexturedGray(t *testing.T) {
	if got := texturedGray(200, 252); got != 200 {
		t.Errorf("bright texture pixel: %d, want the full shade 200", got)
	}
	if got := texturedGray(200, 0); got != 90 {
		t.Errorf("black texture pixel: %d, want %v of 200 = 90", got, texMin)
	}
}

// Seen from either side, a face's texture isn't mirrored: hitting at WallX
// 0.25 from the west gives the mirror column of hitting it from the east.
func TestTextureColumnFlip(t *testing.T) {
	hit := world.Hit{Side: 0, WallX: 0.25}
	fromWest := textureColumn(brickTexture, hit, 1, 0)  // ray going east
	fromEast := textureColumn(brickTexture, hit, -1, 0) // ray going west
	if fromEast != 4 || fromWest != 11 {
		t.Errorf("columns: from east %d, from west %d; want 4 and 11", fromEast, fromWest)
	}
}

// With textures on, a brick wall shows several grays down a column (the
// mortar lines); with them off it's one flat gray.
func TestDrawFrameTextures(t *testing.T) {
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatal(err)
	}
	s := game.NewState(m)
	s.Player.X, s.Player.Y = 2.0, 3.5 // facing a brick wall 2 cells away

	grays := func(textures bool) map[uint8]bool {
		f := newFrame(120, 40)
		f.Textures = textures
		DrawFrame(s, f)
		seen := map[uint8]bool{}
		for y := 20; y < 58; y++ { // inside the wall span (16-61)
			seen[f.Pix[y*f.W+60].Gray] = true
		}
		return seen
	}
	if flat := grays(false); len(flat) != 1 {
		t.Errorf("textures off: %d grays down the wall, want 1", len(flat))
	}
	if tex := grays(true); len(tex) < 3 {
		t.Errorf("textures on: %d grays down the wall, want the brick pattern (3+)", len(tex))
	}
}

func BenchmarkDrawFrameTextured120x40(b *testing.B) {
	s := newState(b, 2.0, 3.5)
	f := newFrame(120, 40)
	f.Textures = true
	for b.Loop() {
		DrawFrame(s, f)
	}
}
