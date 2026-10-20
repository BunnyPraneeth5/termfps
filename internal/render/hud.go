package render

import (
	"fmt"
	"math"
	"strings"
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
	Mode                game.RenderMode
	// Notice is a temporary message shown just above the HUD row (empty =
	// none). Too wide for the screen, it wraps at sentence ends.
	Notice string
}

// compass names the 8 directions in the same order as the minimap arrows.
var compass = [8]string{"E", "SE", "S", "SW", "W", "NW", "N", "NE"}

// DrawHUD writes the HUD on the last row. The first part (FPS, ammo,
// position, facing, render mode) always shows; the extras are added in order only
// while they fit whole, so nothing is cut off mid-word at 80 columns. A
// Notice goes on the row(s) just above, over the bottom of the 3D view,
// never over the minimap (which sits at the top).
func DrawHUD(s tcell.Screen, h HUD) {
	w, rows := s.Size()
	y := rows - 1
	for x := 0; x < w; x++ {
		s.SetContent(x, y, ' ', nil, tcell.StyleDefault)
	}

	p := h.Player
	deg := int(math.Round(p.Angle*180/math.Pi)) % 360
	line := fmt.Sprintf("FPS %4.1f | %s | x %5.2f y %5.2f | %3d° %-2s | %s",
		h.FPS, ammoText(p.Weapon), p.X, p.Y, deg, compass[dirIndex(p.Angle)], h.Mode)

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

	if h.Notice != "" {
		lines := wrapSentences(h.Notice, w-2) // one space of padding each side
		for i, l := range lines {
			row := y - len(lines) + i
			if row < 0 {
				continue
			}
			s.PutStrStyled(0, row, " "+l+" ", noticeStyle)
		}
	}
}

// noticeStyle is reversed (swapped colors) so it stands out in every
// render mode, with or without truecolor.
var noticeStyle = tcell.StyleDefault.Reverse(true)

// wrapSentences returns text as one line if it fits in width, otherwise
// one line per sentence (split after ". "). A sentence still too long is
// left whole; PutStr clips it at the screen edge.
func wrapSentences(text string, width int) []string {
	if len([]rune(text)) <= width {
		return []string{text}
	}
	parts := strings.SplitAfter(text, ". ")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// ammoText is the HUD's ammo part: rounds in the gun / spare rounds.
func ammoText(w game.Weapon) string {
	s := fmt.Sprintf("AMMO %2d / %d", w.Mag, w.Reserve)
	if w.Reloading() {
		s += " RELOADING"
	}
	return s
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
