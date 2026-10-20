package render

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

func TestArrowFor(t *testing.T) {
	cases := []struct {
		deg  float64
		want rune
	}{
		{0, '→'}, {45, '↘'}, {90, '↓'}, {135, '↙'}, {180, '←'},
		{225, '↖'}, {270, '↑'}, {315, '↗'}, {359, '→'}, {20, '→'}, {25, '↘'},
	}
	for _, c := range cases {
		if got := arrowFor(c.deg * math.Pi / 180); got != c.want {
			t.Errorf("arrowFor(%v°) = %q, want %q", c.deg, got, c.want)
		}
	}
}

// room (5 wide, 7 tall) with the player at (1, 3). On an 80x24 screen the
// minimap is 10 columns wide and starts at x = 80 - 10 - 1 = 69.
func TestDrawMinimapLayout(t *testing.T) {
	scr := newSimScreen(t, 80, 24)
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	st := game.NewState(m)
	st.Player.Angle = math.Pi / 2 // facing south

	if !DrawMinimap(scr, st) {
		t.Fatal("DrawMinimap returned false on 80x24")
	}
	const x0 = 69
	// Player arrow at cell (1, 3) -> column 69 + 2 = 71, row 3.
	if str, _, _ := scr.Get(x0+2, 3); str != "↓" {
		t.Errorf("player cell = %q, want ↓ (facing south)", str)
	}
	// Top-left map cell is a wall: both columns use the wall style.
	for _, x := range []int{x0, x0 + 1} {
		if _, style, _ := scr.Get(x, 0); style != minimapWall {
			t.Errorf("(%d, 0) is not wall-styled", x)
		}
	}
	// Cell (2, 1) is floor.
	if _, style, _ := scr.Get(x0+4, 1); style != minimapFloor {
		t.Error("cell (2, 1) is not floor-styled")
	}
	// Nothing on the HUD row, nothing left of the minimap on row 0.
	if _, style, _ := scr.Get(x0-1, 0); style == minimapWall || style == minimapFloor {
		t.Error("minimap drew left of its area")
	}
}

func TestDrawMinimapTooSmall(t *testing.T) {
	scr := newSimScreen(t, 10, 7) // map needs 11 columns and 8 rows
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if DrawMinimap(scr, game.NewState(m)) {
		t.Error("DrawMinimap returned true on a screen too small for it")
	}
}

// Ammo boxes show as yellow dots on the minimap.
func TestDrawMinimapPickup(t *testing.T) {
	scr := newSimScreen(t, 80, 24)
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	st := game.NewState(m)
	st.Pickups = []game.Pickup{{X: 2.5, Y: 4.5, Amount: 25}}
	DrawMinimap(scr, st)
	const x0 = 69 // 80 - 5*2 - 1
	if str, style, _ := scr.Get(x0+2*2, 4); str != "●" || style != minimapAmmo {
		t.Errorf("box cell = %q, want a yellow dot", str)
	}
}
