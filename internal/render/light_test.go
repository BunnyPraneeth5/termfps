package render

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Without darkness or lamps the light grid is off and every cell is 1, so
// maps look as before M4.
func TestLightGridOff(t *testing.T) {
	m, err := world.Parse([]byte(room))
	if err != nil {
		t.Fatal(err)
	}
	var g lightGrid
	g.prepare(m, 0)
	if g.on || g.at(2, 2) != 1 || g.smooth(2.3, 2.7) != 1 {
		t.Errorf("light grid on %v, at %v, smooth %v; want off and 1", g.on, g.at(2, 2), g.smooth(2.3, 2.7))
	}
}

// Darkness lowers the base light; a lamp adds its light near it, and the
// smooth lookup matches the cell value at a cell's center.
func TestLightGridDarknessAndLamp(t *testing.T) {
	m, err := world.Parse([]byte("; darkness: 0.6\n#########\n#P.....l#\n#.......#\n#########\n"))
	if err != nil {
		t.Fatal(err)
	}
	var g lightGrid
	g.prepare(m, 0)
	if math.Abs(g.at(1, 2)-0.4) > 1e-9 {
		t.Errorf("far from the lamp: %v, want the ambient 0.4", g.at(1, 2))
	}
	if g.at(6, 1) <= 0.8 {
		t.Errorf("next to the lamp: %v, want much brighter than 0.4", g.at(6, 1))
	}
	if math.Abs(g.smooth(6.5, 1.5)-g.at(6, 1)) > 1e-9 {
		t.Error("smooth at a cell center isn't the cell's value")
	}
	if g.at(-1, 0) != 0.4 {
		t.Error("outside the map isn't the ambient light")
	}
}

// Flickering stays between 0.25 and 1 and changes over time.
func TestFlicker(t *testing.T) {
	seen := map[float64]bool{}
	for i := range 400 {
		v := flicker(float64(i)/30, 2)
		if v < 0.25 || v > 1 {
			t.Fatalf("flicker = %v at frame %d, want 0.25-1", v, i)
		}
		seen[math.Round(v*100)] = true
	}
	if len(seen) < 10 {
		t.Errorf("flicker took only %d values over 400 frames, want it to vary", len(seen))
	}
}

// A dark map draws walls darker; a map's own fog header makes far walls
// fade faster (M4).
func TestDarknessAndFogDarkenWalls(t *testing.T) {
	gray := func(header string) uint8 {
		m, err := world.Parse([]byte(header + room))
		if err != nil {
			t.Fatal(err)
		}
		s := game.NewState(m)
		s.Player.X, s.Player.Y = 2.0, 3.5 // facing the east wall 2 cells away
		f := newFrame(120, 40)
		DrawFrame(s, f)
		return f.Pix[39*f.W+60].Gray // the middle of the wall
	}
	normal, dark, foggy := gray(""), gray("; darkness: 0.5\n"), gray("; fog: 0.8\n")
	if math.Abs(float64(dark)-float64(normal)/2) > 1 {
		t.Errorf("darkness 0.5: wall %d, want about half of %d", dark, normal)
	}
	if foggy >= normal {
		t.Errorf("fog 0.8: wall %d, want darker than the default fog's %d", foggy, normal)
	}
}
