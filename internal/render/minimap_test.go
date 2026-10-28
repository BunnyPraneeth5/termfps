package render

import (
	"math"
	"strings"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

func TestArrowFor(t *testing.T) {
	cases := []struct {
		deg  float64
		want string
	}{
		{0, "→"}, {45, "↘"}, {90, "↓"}, {135, "↙"}, {180, "←"},
		{225, "↖"}, {270, "↑"}, {315, "↗"}, {359, "→"}, {20, "→"}, {25, "↘"},
	}
	for _, c := range cases {
		if got := arrowFor(c.deg * math.Pi / 180); got != c.want {
			t.Errorf("arrowFor(%v°) = %q, want %q", c.deg, got, c.want)
		}
	}
}

// room (5 wide, 7 tall) with the player at (1, 3). On an 80x24 screen the
// minimap is 10 columns wide and starts at x = 80 - 10 - 1 = 69, inside a
// border (M1): map rows start at row 1.
func TestDrawMinimapLayout(t *testing.T) {
	scr := term.NewFake(80, 24)
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	st := game.NewState(m)
	st.Player.Angle = math.Pi / 2 // facing south

	if !DrawMinimap(scr, st, false) {
		t.Fatal("DrawMinimap returned false on 80x24")
	}
	const x0 = 69
	// Player arrow at cell (1, 3) -> column 69 + 2 = 71, row 1 + 3.
	if str, _ := scr.Get(x0+2, 4); str != "↓" {
		t.Errorf("player cell = %q, want ↓ (facing south)", str)
	}
	// Top-left map cell is a wall: both columns use the wall style.
	for _, x := range []int{x0, x0 + 1} {
		if _, style := scr.Get(x, 1); style != minimapWall {
			t.Errorf("(%d, 1) is not wall-styled", x)
		}
	}
	// Cell (2, 1) is floor.
	if _, style := scr.Get(x0+4, 2); style != minimapFloor {
		t.Error("cell (2, 1) is not floor-styled")
	}
	// The border goes around the 10x7 map area: rows 0 and 8, columns 68 and 79.
	for _, c := range []struct {
		x, y int
		want string
	}{{x0 - 1, 0, "┌"}, {x0 + 10, 0, "┐"}, {x0 - 1, 8, "└"}, {x0 + 10, 8, "┘"}, {x0 + 3, 0, "─"}, {x0 - 1, 4, "│"}} {
		if str, style := scr.Get(c.x, c.y); str != c.want || style != minimapBorder {
			t.Errorf("border (%d, %d) = %q, want %q", c.x, c.y, str, c.want)
		}
	}
	// Nothing left of the border.
	if str, _ := scr.Get(x0-2, 1); str != " " {
		t.Errorf("minimap drew left of its border: %q", str)
	}
}

func TestDrawMinimapTooSmall(t *testing.T) {
	scr := term.NewFake(10, 7) // map and border need 12 columns and 10 rows
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if DrawMinimap(scr, game.NewState(m), false) {
		t.Error("DrawMinimap returned true on a screen too small for it")
	}
}

// Ammo boxes show as yellow dots on the minimap.
func TestDrawMinimapPickup(t *testing.T) {
	scr := term.NewFake(80, 24)
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	st := game.NewState(m)
	st.Pickups = []game.Pickup{{X: 2.5, Y: 4.5, Amount: 25}}
	DrawMinimap(scr, st, false)
	const x0 = 69 // 80 - 5*2 - 1
	if str, style := scr.Get(x0+2*2, 1+4); str != "●" || style != minimapAmmo {
		t.Errorf("box cell = %q, want a yellow dot", str)
	}
}

func TestMinimapWindow(t *testing.T) {
	cases := []struct {
		name           string
		px, py, ox, oy int
	}{
		{"centered", 30, 20, 22, 12},
		{"left/top edge", 2, 1, 0, 0},
		{"right/bottom edge", 63, 31, 48, 16},
	}
	for _, c := range cases {
		// 64x32 map, 16x16 window.
		if ox, oy := minimapWindow(64, 32, 16, 16, c.px, c.py); ox != c.ox || oy != c.oy {
			t.Errorf("%s: window at (%d, %d), want (%d, %d)", c.name, ox, oy, c.ox, c.oy)
		}
	}
	if ox, oy := minimapWindow(16, 16, 16, 16, 5, 9); ox != 0 || oy != 0 {
		t.Errorf("map that fits: window at (%d, %d), want (0, 0)", ox, oy)
	}
}

// On a map bigger than the minimap's space, it shows a 16x16 window around
// the player (M1), and the player's arrow is inside it.
func TestDrawMinimapScrolls(t *testing.T) {
	scr := term.NewFake(80, 24)
	var b strings.Builder
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			switch {
			case x == 0 || y == 0 || x == 63 || y == 31:
				b.WriteByte('#')
			case x == 40 && y == 20:
				b.WriteByte('P')
			default:
				b.WriteByte('.')
			}
		}
		b.WriteByte('\n')
	}
	m, err := world.Parse([]byte(b.String()))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	st := game.NewState(m) // facing east
	if !DrawMinimap(scr, st, false) {
		t.Fatal("DrawMinimap returned false for a big map on 80x24")
	}
	// Window origin: x 40-8 = 32, y 20-8 = 12. Player at view cell (8, 8).
	const x0 = 80 - 16*2 - 1
	if str, _ := scr.Get(x0+8*2, 1+8); str != "→" {
		t.Errorf("view cell (8, 8) = %q, want the player's arrow →", str)
	}
	// Row 17 is the bottom border; below it is the 3D view.
	if str, _ := scr.Get(x0, 17); str != "─" {
		t.Errorf("row 17 = %q, want the bottom border", str)
	}
	if _, style := scr.Get(x0, 18); style == minimapFloor || style == minimapWall {
		t.Error("minimap drew more than 16 rows")
	}
	if _, style := scr.Get(x0-2, 1); style == minimapFloor || style == minimapWall {
		t.Error("minimap drew more than 16 cells across")
	}
}

// With fog of war (M8) unseen cells are blank, and so is an ammo box on
// one; seen cells look as usual.
func TestDrawMinimapFog(t *testing.T) {
	scr := term.NewFake(80, 24)
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	st := game.NewState(m)
	st.Seen[1*m.W+1] = true // only cell (1, 1)
	st.Pickups = []game.Pickup{{X: 2.5, Y: 4.5, Amount: 25}}
	DrawMinimap(scr, st, true)
	const x0 = 69
	if _, style := scr.Get(x0+2, 1+1); style != minimapFloor {
		t.Error("seen floor cell (1, 1) isn't floor-styled")
	}
	if _, style := scr.Get(x0+4, 1+1); style != minimapUnseen {
		t.Error("unseen cell (2, 1) isn't blank")
	}
	if str, _ := scr.Get(x0+2*2, 1+4); str == "●" {
		t.Error("the ammo box on an unseen cell is shown")
	}
	if str, _ := scr.Get(x0+2, 1+3); str != "→" {
		t.Errorf("player cell = %q, want the arrow even unseen", str)
	}
}
