package render

import (
	"strings"
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
		if img.W != 32 || img.H != 32 { // V4; floor and ceiling stay 16x16
			t.Errorf("%s is %dx%d, want 32x32", name, img.W, img.H)
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
	if wallTexture(world.Wall)[0] != brickTexture || wallTexture(world.Stone)[0] != stoneTexture || wallTexture(world.Metal)[0] != metalTexture {
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
	if fromEast != 8 || fromWest != 23 {
		t.Errorf("columns: from east %d, from west %d; want 8 and 23", fromEast, fromWest)
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

// Each mipmap level is half the size of the one before, down to 1x1, and
// averages 2x2 pixels of it (V3).
func TestNewMipmap(t *testing.T) {
	m := newMipmap(brickTexture)
	for i, want := range []int{32, 16, 8, 4, 2, 1} {
		if i >= len(m) || m[i].W != want || m[i].H != want {
			t.Fatalf("level %d: want %dx%d, have %d levels", i, want, want, len(m))
		}
	}
	p := brickTexture.Pix
	sum := int(p[0].Gray) + int(p[1].Gray) + int(p[32].Gray) + int(p[33].Gray)
	if got := int(m[1].Pix[0].Gray); got != (sum+2)/4 {
		t.Errorf("level 1 pixel (0, 0) = %d, want the average %d", got, (sum+2)/4)
	}
}

func TestMipmapLevel(t *testing.T) {
	m := newMipmap(brickTexture)
	cases := []struct {
		texels float64
		size   int
	}{
		{0.3, 32}, {1.9, 32}, {2, 16}, {4, 8}, {7.9, 8}, {1000, 1},
	}
	for _, c := range cases {
		if got := m.level(c.texels).W; got != c.size {
			t.Errorf("level(%v) is %d wide, want %d", c.texels, got, c.size)
		}
	}
}

// Down a column the estimate is texture height / wall height; at a
// grazing angle, neighbor columns hit far apart on the same face, which
// wins. A different face doesn't count.
func TestWallTexels(t *testing.T) {
	hit := world.Hit{Side: 0, MapX: 5, MapY: 3, WallX: 0.9}
	if got := wallTexels(brickTexture, 4, hit, world.Hit{}, false); got != 8 {
		t.Errorf("32-pixel texture on a 4-pixel wall = %v texels, want 8", got)
	}
	prev := world.Hit{Side: 0, MapX: 5, MapY: 2, WallX: 0.4} // 1.5 cells along the same face
	if got := wallTexels(brickTexture, 40, hit, prev, true); got != 48 {
		t.Errorf("grazing: %v texels, want 1.5 cells * 32 = 48", got)
	}
	other := world.Hit{Side: 0, MapX: 7, MapY: 2, WallX: 0.4} // another face
	if got := wallTexels(brickTexture, 40, hit, other, true); got != 0.8 {
		t.Errorf("different face: %v texels, want only the column's 0.8", got)
	}
}

// Far across a wide hall, the floor samples a small mipmap level, so one
// row of it is an even gray instead of a flickering pattern. The hall is
// wide so the middle columns see floor there, not side walls.
func TestFarFloorIsAveraged(t *testing.T) {
	hall := "################################\n"
	for range 30 { // range over a number repeats the loop that many times
		hall += "#..............................#\n"
	}
	hall += "################################\n"
	m, err := world.Parse([]byte(strings.Replace(hall, ".", "P", 1)))
	if err != nil {
		t.Fatal(err)
	}
	s := game.NewState(m)
	s.Player.X, s.Player.Y = 1.5, 16.0 // west end, middle row, facing east
	f := newFrame(120, 40)
	f.Textures = true
	DrawFrame(s, f)
	y := 41 // just below the far east wall: floor about 18 cells away
	if d, ceiling := castDist(f, y); ceiling || d < 15 {
		t.Fatalf("row %d is at %v (ceiling %v), want far floor", y, d, ceiling)
	}
	first := f.Pix[y*f.W+50].Gray
	for x := 50; x < 70; x++ {
		if g := f.Pix[y*f.W+x].Gray; g != first {
			t.Fatalf("far floor row: column %d = %d, column 50 = %d; want one even gray", x, g, first)
		}
	}
}
