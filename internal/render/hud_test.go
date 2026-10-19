package render

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/game"
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
	s := newSimScreen(t, cols, 24)
	DrawHUD(s, h)
	return strings.TrimRight(rowText(s, cols, 23), " ")
}

func TestHUDCoreAt80Columns(t *testing.T) {
	line := hudLine(t, 80, sampleHUD())
	for _, want := range []string{"FPS 30.0", "x 12.34 y  5.67", "270° N"} {
		if !strings.Contains(line, want) {
			t.Errorf("HUD %q missing %q", line, want)
		}
	}
	// Timings fit at 80; the keys label doesn't and must be left out whole.
	if !strings.Contains(line, "show 2.00 ms") {
		t.Errorf("HUD %q missing timings at 80 columns", line)
	}
	if strings.Contains(line, "keys") {
		t.Errorf("HUD %q includes keys label that doesn't fit at 80", line)
	}
}

// Extras never appear cut off: at 50 columns only the core part shows.
func TestHUDNoPartialExtras(t *testing.T) {
	line := hudLine(t, 50, sampleHUD())
	if strings.Contains(line, "draw") || strings.Contains(line, "|  ") {
		t.Errorf("HUD %q shows a partial extra at 50 columns", line)
	}
	if !strings.HasSuffix(line, "270° N") {
		t.Errorf("HUD %q, want it to end with the core part", line)
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
	s := newSimScreen(t, 80, 24)
	DrawHUD(s, sampleHUD())
	for row := 0; row < 23; row++ {
		if txt := strings.TrimSpace(rowText(s, 80, row)); txt != "" {
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
