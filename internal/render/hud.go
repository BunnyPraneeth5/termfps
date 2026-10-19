package render

import (
	"fmt"
	"math"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
)

// HUD is everything shown on the bottom row.
type HUD struct {
	FPS                 float64
	Player              game.Player
	Draw, Present, Show time.Duration // previous frame's timings
	Keys                string        // input mode label, e.g. "keys async (focused)"
	MinimapHidden       bool          // minimap is on but didn't fit on screen
}

// compass names the 8 directions in the same order as the minimap arrows.
var compass = [8]string{"E", "SE", "S", "SW", "W", "NW", "N", "NE"}

// DrawHUD writes the HUD on the last row. The first part (FPS, position,
// facing) always shows; the extras are added in order only while they fit
// whole, so nothing is cut off mid-word at 80 columns. It only touches the
// last row, so it never overlaps the minimap.
func DrawHUD(s tcell.Screen, h HUD) {
	w, rows := s.Size()
	y := rows - 1
	for x := 0; x < w; x++ {
		s.SetContent(x, y, ' ', nil, tcell.StyleDefault)
	}

	p := h.Player
	deg := int(math.Round(p.Angle*180/math.Pi)) % 360
	line := fmt.Sprintf("FPS %4.1f | x %5.2f y %5.2f | %3d° %-2s",
		h.FPS, p.X, p.Y, deg, compass[dirIndex(p.Angle)])

	var extras []string
	if h.MinimapHidden {
		extras = append(extras, " | minimap hidden: screen too small")
	}
	extras = append(extras,
		fmt.Sprintf(" | draw %.2f present %.2f show %.2f ms", ms(h.Draw), ms(h.Present), ms(h.Show)),
		" | "+h.Keys,
	)
	for _, e := range extras {
		if len([]rune(line+e)) > w { // count runes, not bytes: '°' is 2 bytes
			break
		}
		line += e
	}
	s.PutStr(0, y, line)
}

// dirIndex returns the nearest of 8 directions (0 = east, clockwise).
func dirIndex(angle float64) int {
	i := int(math.Round(angle/(math.Pi/4))) % 8
	if i < 0 {
		i += 8
	}
	return i
}

// ms converts a duration to milliseconds with fractions.
func ms(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}
