package render

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/assets"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// A big open room (walls far away) facing east from (2.5, 5.5).
const spriteRoom = "################\n" +
	"#..............#\n" +
	"#..............#\n" +
	"#..............#\n" +
	"#..............#\n" +
	"#.P............#\n" +
	"#..............#\n" +
	"#..............#\n" +
	"#..............#\n" +
	"#..............#\n" +
	"################\n"

func spriteState(t *testing.T, layout string) *game.State {
	t.Helper()
	m, err := world.Parse([]byte(layout))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return game.NewState(m)
}

// greenColumns returns the frame columns that have at least one green pixel.
func greenColumns(f *Frame) []int {
	var cols []int
	for x := 0; x < f.W; x++ {
		for y := 0; y < f.H; y++ {
			if f.Pix[y*f.W+x].Tint == TintGreen {
				cols = append(cols, x)
				break
			}
		}
	}
	return cols
}

func drawWith(t *testing.T, s *game.State, sprites ...Sprite) *Frame {
	t.Helper()
	f := newFrame(120, 40)
	DrawFrame(s, f)
	DrawSprites(f, s.Player, sprites)
	return f
}

func TestProjectCenterAndSides(t *testing.T) {
	s := spriteState(t, spriteRoom) // at (2.5, 5.5) facing east
	f := newFrame(120, 40)
	p := s.Player
	depth, x, ok := project(f, p, 6.5, 5.5) // 4 cells straight ahead
	if !ok || math.Abs(depth-4) > 1e-9 || math.Abs(x-60) > 1e-9 {
		t.Errorf("straight ahead: depth %v, x %v, ok %v; want 4, 60, true", depth, x, ok)
	}
	// Facing east, +y (south) is the camera's right.
	if _, x, _ := project(f, p, 6.5, 6.5); x <= 60 {
		t.Errorf("sprite to the south is at x %v, want the right half", x)
	}
	if _, x, _ := project(f, p, 6.5, 4.5); x >= 60 {
		t.Errorf("sprite to the north is at x %v, want the left half", x)
	}
	if _, _, ok := project(f, p, 1.5, 5.5); ok {
		t.Error("sprite behind the camera projected as visible")
	}
}

// A zombie straight ahead is drawn around the center, standing on the
// floor line, and DrawFrame recorded the wall distance per column.
func TestDrawSpriteVisible(t *testing.T) {
	s := spriteState(t, spriteRoom)
	f := drawWith(t, s, Sprite{X: 6.5, Y: 5.5, Image: ZombieImage, Height: 0.8})
	if math.Abs(f.ZBuf[60]-12.5) > 1e-9 {
		t.Errorf("ZBuf[60] = %v, want 12.5 (east wall face at x=15)", f.ZBuf[60])
	}
	cols := greenColumns(f)
	if len(cols) == 0 {
		t.Fatal("no green pixels: zombie not drawn")
	}
	if cols[0] > 60 || cols[len(cols)-1] < 60 {
		t.Errorf("zombie spans columns %d-%d, want it around the center 60", cols[0], cols[len(cols)-1])
	}
	// Feet near the floor line of depth 4: center 39 + wallH/2 = 39 + 11.4.
	lowest := 0
	for i, px := range f.Pix {
		if px.Tint == TintGreen {
			lowest = max(lowest, i/f.W)
		}
	}
	if lowest < 48 || lowest > 51 {
		t.Errorf("zombie's lowest pixel row = %d, want about 50 (standing on the floor)", lowest)
	}
}

// A wall between the camera and the sprite hides it completely.
func TestDrawSpriteBehindWall(t *testing.T) {
	s := spriteState(t, "################\n"+
		"#..............#\n"+
		"#..............#\n"+
		"#..............#\n"+
		"#.....#........#\n"+
		"#.P...#........#\n"+
		"#.....#........#\n"+
		"#..............#\n"+
		"#..............#\n"+
		"#..............#\n"+
		"################\n")
	f := drawWith(t, s, Sprite{X: 9.5, Y: 5.5, Image: ZombieImage, Height: 0.8})
	if cols := greenColumns(f); len(cols) != 0 {
		t.Errorf("zombie behind a wall drawn in %d columns, want 0", len(cols))
	}
}

// A pillar covering part of the view hides only the columns behind it.
func TestDrawSpritePartlyHidden(t *testing.T) {
	s := spriteState(t, "################\n"+
		"#..............#\n"+
		"#..............#\n"+
		"#..............#\n"+
		"#..............#\n"+
		"#.P..#.........#\n"+
		"#..............#\n"+
		"#..............#\n"+
		"#..............#\n"+
		"#..............#\n"+
		"################\n")
	// Seen from (2.5, 5.5), the zombie at y 3.8-4.4 lines up with the
	// pillar's front face at y 4.89-5.11; the pillar's edge (y = 5) splits it.
	full := len(greenColumns(drawWith(t, spriteState(t, spriteRoom),
		Sprite{X: 9.5, Y: 4.1, Image: ZombieImage, Height: 0.8})))
	part := len(greenColumns(drawWith(t, s,
		Sprite{X: 9.5, Y: 4.1, Image: ZombieImage, Height: 0.8})))
	if part == 0 || part >= full {
		t.Errorf("partly hidden zombie drawn in %d columns, want between 1 and %d", part, full-1)
	}
}

// Farther sprites are smaller.
func TestDrawSpriteShrinksWithDistance(t *testing.T) {
	near := len(greenColumns(drawWith(t, spriteState(t, spriteRoom),
		Sprite{X: 5.5, Y: 5.5, Image: ZombieImage, Height: 0.8})))
	far := len(greenColumns(drawWith(t, spriteState(t, spriteRoom),
		Sprite{X: 11.5, Y: 5.5, Image: ZombieImage, Height: 0.8})))
	if far == 0 || far >= near {
		t.Errorf("columns: near %d, far %d; want far smaller and not 0", near, far)
	}
}

// A near sprite covers a far one in the same spot on screen, whatever
// order they come in.
func TestDrawSpritesNearCoversFar(t *testing.T) {
	s := spriteState(t, spriteRoom)
	f := drawWith(t, s,
		Sprite{X: 5.5, Y: 5.5, Image: ZombieImage, Height: 0.8},   // near, listed first
		Sprite{X: 10.5, Y: 5.5, Image: ShooterImage, Height: 0.8}, // far
	)
	// At the screen center, the chest of the near zombie must win.
	foundGreen := false
	for y := 0; y < f.H; y++ {
		switch f.Pix[y*f.W+60].Tint {
		case TintGreen:
			foundGreen = true
		case TintRed:
			if !foundGreen {
				continue // red above the zombie's head is fine
			}
		}
	}
	if !foundGreen {
		t.Error("near zombie not visible at the center; the far shooter covered it")
	}
}

func TestEnemyAssetsParse(t *testing.T) {
	for name, data := range map[string][]byte{"zombie": assets.Zombie, "shooter": assets.Shooter} {
		if _, err := ParseImage(data); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// A flashing sprite (just shot) is drawn white instead of its colors.
func TestDrawSpriteFlash(t *testing.T) {
	s := spriteState(t, spriteRoom)
	f := drawWith(t, s, Sprite{X: 6.5, Y: 5.5, Image: ZombieImage, Height: 0.8, Flash: true})
	if cols := greenColumns(f); len(cols) != 0 {
		t.Error("flashing zombie still has green pixels")
	}
	white := 0
	for _, p := range f.Pix {
		if p.Gray == 255 && p.Tint == TintNone {
			white++
		}
	}
	if white == 0 {
		t.Error("flashing zombie has no white pixels")
	}
}
