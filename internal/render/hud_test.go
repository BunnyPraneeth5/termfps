package render

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

func sampleHUD() HUD {
	return HUD{
		FPS:     30,
		Player:  game.Player{X: 12.34, Y: 5.67, Angle: 3 * math.Pi / 2, Stamina: 1, HP: 76, Armor: 50, Guns: game.StartGuns()}, // facing north
		Draw:    70 * time.Microsecond,
		Present: 1100 * time.Microsecond,
		Show:    2 * time.Millisecond,
		Keys:    "keys async (focused)",
		Tier:    1,
	}
}

// hudLines draws the HUD on a cols x 24 screen and returns its two rows.
func hudLines(t *testing.T, cols int, h HUD) (line1, line2 string) {
	t.Helper()
	s := term.NewFake(cols, 24)
	DrawHUD(s, h)
	return strings.TrimRight(s.Row(22), " "), strings.TrimRight(s.Row(23), " ")
}

// The two lines at 80 columns: bars with numbers and the kills; the gun,
// its magazine strip, the gun numbers, the tier and the time.
func TestHUDTwoLines(t *testing.T) {
	h := sampleHUD()
	h.Kills, h.Tier, h.Time = 23, 2, 187.9
	h.Player.Guns[game.Pistol].Mag = 18
	l1, l2 := hudLines(t, 80, h)
	if want := " ♥ ▰▰▰▰▰▰▰▰▱▱ 76    ◆ ▰▰▰▰▰▱▱▱▱▱ 50    » ▰▰▰▰▰▰▰▰▰▰"; !strings.HasPrefix(l1, want) {
		t.Errorf("line 1 = %q, want it to start %q", l1, want)
	}
	if !strings.HasSuffix(l1, "KILLS 23") || len([]rune(l1)) != 79 {
		t.Errorf("line 1 = %q, want KILLS 23 one column in from the right", l1)
	}
	if want := " PISTOL 18/50  " + strings.Repeat("❙", 25) + "  1 2 3"; !strings.HasPrefix(l2, want) {
		t.Errorf("line 2 = %q, want it to start %q", l2, want)
	}
	if !strings.HasSuffix(l2, "TIER 2  TIME 3:07") {
		t.Errorf("line 2 = %q, want it to end TIER 2  TIME 3:07", l2)
	}
}

// The bars and the numbers have their colors; used rounds and guns not
// found yet are dark; the gun in hand is bright.
func TestHUDColors(t *testing.T) {
	s := term.NewFake(80, 24)
	h := sampleHUD()
	h.Player.Guns[game.Pistol].Mag = 18
	h.Player.Guns[game.Shotgun].Owned = true
	p := DrawHUD(s, h)
	fg := func(x, y int) term.Color { _, st := s.Get(x, y); return st.Fg }
	bar := p.Health + 2
	if fg(bar, 22) != ui.ColRed || fg(bar+9, 22) != ui.ColBorder {
		t.Errorf("health bar: first cell %v, last cell %v; want red, then the empty color", fg(bar, 22), fg(bar+9, 22))
	}
	if fg(p.Armor+2, 22) != ui.ColBlue || fg(p.Stamina+2, 22) != ui.ColYellow {
		t.Error("armor isn't blue or stamina isn't yellow")
	}
	if fg(p.Mag+17, 23) != ui.ColAmber || fg(p.Mag+18, 23) != ui.ColBorder {
		t.Error("the magazine strip doesn't show 18 rounds bright and the used ones dark")
	}
	if fg(p.Guns, 23) != ui.ColAmber || fg(p.Guns+2, 23) != ui.ColDim || fg(p.Guns+4, 23) != ui.ColBorder {
		t.Errorf("gun numbers: %v %v %v; want the pistol bright, the shotgun dim, the rifle dark", fg(p.Guns, 23), fg(p.Guns+2, 23), fg(p.Guns+4, 23))
	}
	if _, st := s.Get(0, 22); st.Bg != ui.ColPanel {
		t.Errorf("HUD background %v, want the panel color", st.Bg)
	}
	h.Player.HP = 20
	s2 := term.NewFake(80, 24)
	p = DrawHUD(s2, h)
	if _, st := s2.Get(p.Health+13, 22); st.Fg != ui.ColRed {
		t.Error("at 20 HP the number isn't red")
	}
}

func TestClock(t *testing.T) {
	for in, want := range map[float64]string{0: "0:00", 59.9: "0:59", 60: "1:00", 607: "10:07"} {
		if got := Clock(in); got != want {
			t.Errorf("Clock(%v) = %q, want %q", in, got, want)
		}
	}
}

// Extras go in the gap on the second line, whole or not at all: at 80
// columns the FPS; more as the window gets wider.
func TestHUDExtras(t *testing.T) {
	_, l2 := hudLines(t, 80, sampleHUD())
	if !strings.Contains(l2, "FPS 30.0") || strings.Contains(l2, "draw") || strings.Contains(l2, "270") {
		t.Errorf("line 2 at 80 = %q, want the FPS and no cut-off extras", l2)
	}
	_, l2 = hudLines(t, 110, sampleHUD())
	if !strings.Contains(l2, "FPS 30.0 · half-block · 270° N") || strings.Contains(l2, "draw") {
		t.Errorf("line 2 at 110 = %q, want the mode and the facing too", l2)
	}
	_, l2 = hudLines(t, 190, sampleHUD())
	if !strings.Contains(l2, "keys async (focused)") || !strings.Contains(l2, "draw 0.07 present 1.10 show 2.00 ms") || !strings.Contains(l2, "x 12.34 y  5.67") {
		t.Errorf("line 2 at 190 = %q, want every extra", l2)
	}
	h := sampleHUD()
	h.Mode = game.ModeASCII
	h.MinimapHidden = true
	if _, l2 = hudLines(t, 130, h); !strings.Contains(l2, "minimap hidden: screen too small · FPS 30.0 · ascii") {
		t.Errorf("line 2 = %q, want the minimap note first and the ascii mode", l2)
	}
	h = sampleHUD()
	h.Player.Angle = 2*math.Pi - 0.001 // 359.94°, rounds to 360 -> 0
	if _, l2 = hudLines(t, 110, h); !strings.Contains(l2, "  0° E") {
		t.Errorf("line 2 = %q, want 0° E", l2)
	}
}

// The HUD only writes its two rows, so it can't overlap the minimap.
func TestHUDOnlyItsRows(t *testing.T) {
	s := term.NewFake(80, 24)
	DrawHUD(s, sampleHUD())
	for row := 0; row < 24-HUDRows; row++ {
		if txt := strings.TrimSpace(s.Row(row)); txt != "" {
			t.Fatalf("row %d has %q, want empty", row, txt)
		}
	}
}

// The HUD names the gun in hand (M11); a reload shows instead of the
// magazine strip.
func TestHUDGunAndReload(t *testing.T) {
	h := sampleHUD()
	h.Player.Guns[game.Shotgun] = game.Weapon{Kind: game.Shotgun, Owned: true, Mag: 4, Reserve: 12}
	h.Player.Gun = game.Shotgun
	if _, l2 := hudLines(t, 80, h); !strings.HasPrefix(l2, " SHOTGUN 4/12  ❙❙❙❙❙❙  1 2 3") {
		t.Errorf("line 2 = %q, want SHOTGUN 4/12 with a six-round strip", l2)
	}
	// A real reload: fire the last round of a state's gun.
	m, err := world.Parse([]byte("#####\n#P..#\n#####\n"))
	if err != nil {
		t.Fatal(err)
	}
	st := game.NewState(m)
	st.Player.Weapon().Mag = 1
	game.Update(st, game.Input{Fire: true}, 0.01)
	h.Player = st.Player
	if _, l2 := hudLines(t, 80, h); !strings.Contains(l2, "PISTOL 0/50  RELOADING") {
		t.Errorf("line 2 = %q, want RELOADING in place of the strip", l2)
	}
}

// Waves (M15): the HUD shows the wave instead of the tier, and the break's
// countdown; it still fits 80 columns with the pistol's long strip.
func TestHUDWaves(t *testing.T) {
	h := sampleHUD()
	h.Wave, h.Time = 3, 187
	if _, l2 := hudLines(t, 80, h); !strings.HasSuffix(l2, "WAVE 3/10  TIME 3:07") || strings.Contains(l2, "TIER") {
		t.Errorf("line 2 = %q, want WAVE 3/10 and no tier", l2)
	}
	h.Wave, h.Break, h.Time = 10, 11.2, 607
	_, l2 := hudLines(t, 80, h)
	if !strings.Contains(l2, "WAVE 10/10  BREAK 0:12") || !strings.HasSuffix(l2, "10:07") || !strings.Contains(l2, "1 2 3  ") {
		t.Errorf("line 2 = %q, want the wave, the break and the time, clear of the gun numbers", l2)
	}
}

// The simple symbol set draws the HUD in plain ASCII.
func TestHUDSimple(t *testing.T) {
	ui.UseSimple(true)
	defer ui.UseSimple(false)
	h := sampleHUD()
	h.Player.Guns[game.Pistol].Mag = 18
	l1, l2 := hudLines(t, 80, h)
	if want := " HP [########--] 76    AR [#####-----] 50    ST [##########]"; !strings.HasPrefix(l1, want) {
		t.Errorf("line 1 = %q, want it to start %q", l1, want)
	}
	if want := " PISTOL 18/50  ||||||||||||||||||.......  [1] 2 3"; !strings.HasPrefix(l2, want) {
		t.Errorf("line 2 = %q, want it to start %q", l2, want)
	}
	for _, r := range l1 + l2 {
		if r > 126 {
			t.Fatalf("the simple HUD has %q", r)
		}
	}
}

// DrawHUDAt reports where its parts start, in order on each line.
func TestHUDParts(t *testing.T) {
	s := term.NewFake(80, 24)
	p := DrawHUDAt(s, sampleHUD(), 3, 5, 74)
	if p.Health != 4 || !(p.Health < p.Armor && p.Armor < p.Stamina && p.Stamina < p.Kills) {
		t.Errorf("line 1 parts out of order: %+v", p)
	}
	if p.Gun != 4 || !(p.Gun < p.Mag && p.Mag < p.Guns && p.Guns < p.Stage && p.Stage < p.Time) || p.Time >= 77 {
		t.Errorf("line 2 parts out of order or outside the 74 columns: %+v", p)
	}
	if str, _ := s.Get(p.Kills, 5); str != "K" {
		t.Errorf("the kills part starts with %q", str)
	}
}
