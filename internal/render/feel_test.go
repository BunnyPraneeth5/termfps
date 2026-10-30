package render

import (
	"math"
	"strings"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

func feelState(t *testing.T, layout string) *game.State {
	t.Helper()
	m, err := world.Parse([]byte(layout))
	if err != nil {
		t.Fatal(err)
	}
	s := game.NewState(m)
	s.Spawning = false
	return s
}

// A pattern Frame: every pixel's gray is its column, so shifts are easy to see.
func patternFrame() *Frame {
	f := newFrame(40, 20)
	for i := range f.Pix {
		f.Pix[i] = Pixel{Gray: uint8(i % f.W)}
	}
	return f
}

// A shot red barrel shakes the view: the picture moves, the edges are
// filled, nothing is left blank (M16).
func TestShakeView(t *testing.T) {
	s := feelState(t, "#######\n#P..B.#\n#######\n")
	game.Update(s, game.Input{Fire: true}, 1.0/30)
	if x, y := s.Player.Shake(); x == 0 && y == 0 {
		t.Fatal("no shake after a barrel blew up 3 cells away")
	}
	f := patternFrame()
	ShakeView(f, s.Player)
	moved := false
	for i, px := range f.Pix {
		if int(px.Gray) != i%f.W {
			moved = true
		}
	}
	sx, sy := s.Player.Shake()
	if dx, dy := math.Round(sx*shakePixels*float64(f.H)), math.Round(sy*shakePixels*float64(f.H)); (dx != 0 || dy != 0) && !moved {
		t.Error("view didn't move while shaking")
	}
	still := patternFrame()
	ShakeView(still, game.Player{})
	for i, px := range still.Pix {
		if int(px.Gray) != i%still.W {
			t.Fatal("view moved without a shake")
		}
	}
}

// A shift by a known amount: every pixel comes from dx columns to the left.
func TestShakeViewShift(t *testing.T) {
	s := feelState(t, "#######\n#P..B.#\n#######\n")
	game.Update(s, game.Input{Fire: true}, 1.0/30)
	f := patternFrame()
	sx, _ := s.Player.Shake()
	dx := int(math.Round(sx * shakePixels * float64(f.H)))
	ShakeView(f, s.Player)
	for x := 0; x < f.W; x++ {
		want := min(max(x-dx, 0), f.W-1)
		if got := int(f.Pix[(f.H/2)*f.W+x].Gray); got != want {
			t.Fatalf("shift %d: column %d shows column %d, want %d", dx, x, got, want)
		}
	}
}

// The hit marker: white ticks on a hit, red on a kill, nothing otherwise.
func TestDrawHitMarker(t *testing.T) {
	s := feelState(t, "########\n#P...Z.#\n########\n")
	f := gunFrame(s.Player)
	DrawHitMarker(f, s.Player)
	if countGray(f, 255) != 0 {
		t.Fatal("hit marker drawn without a hit")
	}
	game.Update(s, game.Input{Fire: true}, 1.0/30)
	f = gunFrame(s.Player)
	DrawHitMarker(f, s.Player)
	cx, cy := f.W/2, f.H/2
	if p := f.Pix[(cy+crosshairArm+1)*f.W+cx+crosshairArm+1]; p.Gray != 255 || p.Tint != TintNone {
		t.Errorf("tick pixel = %+v after a hit, want white", p)
	}
}

func countGray(f *Frame, g uint8) int {
	n := 0
	for _, p := range f.Pix {
		if p.Gray == g && p.Tint == TintNone {
			n++
		}
	}
	return n
}

// Hit from the player's right: the red arc is right of center, not left.
func TestDrawHitDirection(t *testing.T) {
	s := feelState(t, "#####\n#...#\n#.PZ#\n#...#\n#####\n")
	s.Player.Angle = 3 * math.Pi / 2 // facing north: the zombie (east) is to the right
	zombie := s.Enemies
	s.Enemies = nil
	f := newFrame(80, 24)
	DrawHitDirection(f, s.Player)
	if countTint(f, TintRed) != 0 {
		t.Fatal("hit direction drawn without a hit")
	}
	s.Enemies = zombie
	for i := 0; i < 60 && !s.Player.Hurting(); i++ {
		game.Update(s, game.Input{}, 1.0/30)
	}
	if !s.Player.Hurting() {
		t.Fatal("the zombie never hit")
	}
	f = newFrame(80, 24)
	DrawHitDirection(f, s.Player)
	left, right := 0, 0
	for i, p := range f.Pix {
		if p.Tint == TintRed {
			if i%f.W < f.W/2 {
				left++
			} else {
				right++
			}
		}
	}
	if right == 0 || left != 0 {
		t.Errorf("hit from the right: %d red pixels left of center, %d right; want all right", left, right)
	}
}

// Stamina shows on the HUD only while it isn't full.
func TestHUDStamina(t *testing.T) {
	h := sampleHUD()
	if line := hudLine(t, 120, h); strings.Contains(line, "STAMINA") {
		t.Errorf("HUD %q shows stamina while full", line)
	}
	h.Player.Stamina = 0.42
	if line := hudLine(t, 120, h); !strings.Contains(line, "| 0:00 | STAMINA 42% |") {
		t.Errorf("HUD %q, want STAMINA 42%% after the time", line)
	}
}
