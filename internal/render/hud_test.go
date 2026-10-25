package render

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
)

func sampleHUD() HUD {
	return HUD{
		FPS:     30,
		Player:  game.Player{X: 12.34, Y: 5.67, Angle: 3 * math.Pi / 2}, // facing north
		Draw:    70 * time.Microsecond,
		Present: 1100 * time.Microsecond,
		Show:    2 * time.Millisecond,
		Keys:    "keys async (focused)",
	}
}

func hudLine(t *testing.T, cols int, h HUD) string {
	t.Helper()
	s := term.NewFake(cols, 24)
	DrawHUD(s, h)
	return strings.TrimRight(s.Row(23), " ")
}

func TestHUDCoreAt80Columns(t *testing.T) {
	line := hudLine(t, 80, sampleHUD())
	for _, want := range []string{"HP", "AMMO", "KILLS 0", "TIER 0", "0:00", "270° N", "FPS 30.0", "| half-block"} {
		if !strings.Contains(line, want) {
			t.Errorf("HUD %q missing %q", line, want)
		}
	}
	// The timings don't fit after the core part at 80 and are left out whole.
	if strings.Contains(line, "draw") || strings.Contains(line, "keys") {
		t.Errorf("HUD %q includes extras that don't fit at 80", line)
	}
}

func TestHUDTimingsAt140Columns(t *testing.T) {
	if line := hudLine(t, 140, sampleHUD()); !strings.Contains(line, "show 2.00 ms") {
		t.Errorf("HUD %q missing timings at 140 columns", line)
	}
}

func TestHUDKillsTierTime(t *testing.T) {
	h := sampleHUD()
	h.Kills, h.Tier, h.Time = 23, 2, 187.9
	if line := hudLine(t, 80, h); !strings.Contains(line, "KILLS 23 | TIER 2 | 3:07 |") {
		t.Errorf("HUD %q, want KILLS 23 | TIER 2 | 3:07", line)
	}
}

func TestClock(t *testing.T) {
	for in, want := range map[float64]string{0: "0:00", 59.9: "0:59", 60: "1:00", 607: "10:07"} {
		if got := Clock(in); got != want {
			t.Errorf("Clock(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestHUDAmmo(t *testing.T) {
	h := sampleHUD()
	h.Player.Weapon = game.Weapon{Mag: 18, Reserve: 50}
	if line := hudLine(t, 80, h); !strings.Contains(line, "AMMO 18 / 50 |") {
		t.Errorf("HUD %q, want AMMO 18 / 50", line)
	}
}

func TestHUDShowsMode(t *testing.T) {
	h := sampleHUD()
	h.Mode = game.ModeASCII
	if line := hudLine(t, 80, h); !strings.Contains(line, "| ascii") {
		t.Errorf("HUD %q doesn't show the ascii mode", line)
	}
}

// The half-block caution (82 characters) wraps into two sentence lines at
// 80 columns, on the rows just above the HUD.
func TestHUDNoticeWrapsAbove(t *testing.T) {
	h := sampleHUD()
	h.Notice = "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."
	s := term.NewFake(80, 24)
	DrawHUD(s, h)
	if got := strings.TrimSpace(s.Row(21)); got != "Half-block needs a truecolor terminal and more bandwidth." {
		t.Errorf("row 21 = %q, want the first sentence", got)
	}
	if got := strings.TrimSpace(s.Row(22)); got != "Use ASCII over slow SSH." {
		t.Errorf("row 22 = %q, want the second sentence", got)
	}
	if got := strings.TrimSpace(s.Row(20)); got != "" {
		t.Errorf("row 20 = %q, want untouched", got)
	}
}

func TestHUDNoticeOneLineWhenWide(t *testing.T) {
	h := sampleHUD()
	h.Notice = "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."
	s := term.NewFake(120, 24)
	DrawHUD(s, h)
	if got := strings.TrimSpace(s.Row(22)); got != h.Notice {
		t.Errorf("row 22 = %q, want the whole notice on one line", got)
	}
	if got := strings.TrimSpace(s.Row(21)); got != "" {
		t.Errorf("row 21 = %q, want untouched", got)
	}
}

// Extras never appear cut off: at 70 columns only the core part shows.
func TestHUDNoPartialExtras(t *testing.T) {
	line := hudLine(t, 70, sampleHUD())
	if strings.Contains(line, "draw") || strings.Contains(line, "|  ") {
		t.Errorf("HUD %q shows a partial extra at 70 columns", line)
	}
	if !strings.HasSuffix(line, "| FPS 30.0") {
		t.Errorf("HUD %q, want it to end with the last extra that fits whole (FPS)", line)
	}
}

func TestHUDWideShowsEverything(t *testing.T) {
	line := hudLine(t, 160, sampleHUD())
	if !strings.Contains(line, "keys async (focused)") || !strings.Contains(line, "draw 0.07") {
		t.Errorf("HUD %q missing extras at 160 columns", line)
	}
}

func TestHUDMinimapHidden(t *testing.T) {
	h := sampleHUD()
	h.MinimapHidden = true
	if line := hudLine(t, 120, h); !strings.Contains(line, "minimap hidden") {
		t.Errorf("HUD %q doesn't say the minimap is hidden", line)
	}
}

// The HUD only writes the last row, so it can't overlap the minimap.
func TestHUDOnlyLastRow(t *testing.T) {
	s := term.NewFake(80, 24)
	DrawHUD(s, sampleHUD())
	for row := 0; row < 23; row++ {
		if txt := strings.TrimSpace(s.Row(row)); txt != "" {
			t.Fatalf("row %d has %q, want empty", row, txt)
		}
	}
}

func TestHUDAngleWraps(t *testing.T) {
	h := sampleHUD()
	h.Player.Angle = 2*math.Pi - 0.001 // 359.94°, rounds to 360 -> 0
	if line := hudLine(t, 80, h); !strings.Contains(line, "  0° E") {
		t.Errorf("HUD %q, want 0° E", line)
	}
}

func TestHUDHealthFirst(t *testing.T) {
	h := sampleHUD()
	h.Player.HP = 73
	if line := hudLine(t, 80, h); !strings.HasPrefix(line, "HP  73 | AMMO") {
		t.Errorf("HUD %q, want it to start with HP  73 | AMMO", line)
	}
}

// DrawMessage centers its lines in a box with swapped colors.
func TestDrawMessage(t *testing.T) {
	s := term.NewFake(80, 24)
	DrawMessage(s, []string{"YOU DIED", "Enter: try again"})
	found := 0
	for row := 0; row < 24; row++ {
		txt := s.Row(row)
		for _, want := range []string{"YOU DIED", "Enter: try again"} {
			if i := strings.Index(txt, want); i >= 0 {
				found++
				center := i + len(want)/2
				if center < 36 || center > 44 {
					t.Errorf("%q centered at column %d, want about 40", want, center)
				}
				if _, style := s.Get(i, row); style != noticeStyle {
					t.Errorf("%q not in the box style", want)
				}
			}
		}
	}
	if found != 2 {
		t.Errorf("found %d of 2 message lines", found)
	}
}
