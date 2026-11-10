package render

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// HUDRows is how many rows at the bottom of the screen the HUD takes (U4:
// two lines with bars). The 3D view gets the rows above.
const HUDRows = 2

// hudBarCells is how many cells the health, armor and stamina bars have.
const hudBarCells = 10

// HUD is everything shown on the HUD's two rows.
type HUD struct {
	FPS                 float64
	Player              game.Player
	Draw, Present, Show time.Duration // previous frame's timings
	Keys                string        // input mode label, e.g. "keys async (focused)"
	MinimapHidden       bool          // minimap is on but didn't fit on screen
	Mode                game.RenderMode
	Kills, Tier         int
	// Wave is the waves-mode wave (M15), 0 in survival; Break the seconds
	// left before it starts (0 while it runs).
	Wave  int
	Break float64
	Time  float64 // seconds survived this run
}

// HUDParts says in which column each part of the HUD starts, so the guide
// can point at them.
type HUDParts struct {
	Health, Armor, Stamina, Kills int // on the first line
	Gun, Mag, Guns, Stage, Time   int // on the second
}

// compass names the 8 directions in the same order as the minimap arrows.
var compass = [8]string{"E", "SE", "S", "SW", "W", "NW", "N", "NE"}

// DrawHUD draws the HUD on the last two rows of the screen.
func DrawHUD(s term.Screen, h HUD) HUDParts {
	w, rows := s.Size()
	return DrawHUDAt(s, h, 0, rows-HUDRows, w)
}

// DrawHUDAt draws the HUD on rows y and y+1, in the w columns from x (the
// guide draws an example that way).
//
// Line 1: health, armor and stamina as bars (health and armor with their
// numbers), and the kills at the right. Line 2: the gun in hand with the
// rounds in it and the spare ones, the magazine as a strip (used rounds
// dim), the gun numbers (the one in hand bright), and at the right the
// tier (or the wave, and the break's countdown) and the time. The extras
// (FPS, render mode, facing, timings, position, key mode) go in the gap on
// line 2, in that order, only while they fit whole.
func DrawHUDAt(s term.Screen, h HUD, x0, y, w int) HUDParts {
	ground := ui.ColPanel
	text := ui.On(ui.ColText, ground)
	dim := ui.On(ui.ColDim, ground)
	bright := ui.Bold(ui.On(ui.ColBright, ground))
	off := ui.On(ui.ColBorder, ground)
	sy := ui.Sym()
	ui.Fill(s, x0, y, w, HUDRows, text)
	var parts HUDParts
	p := h.Player

	// Line 1.
	x := x0 + 1
	meter := func(label string, c term.Color, frac float64, number int) int {
		start := x
		x = ui.Put(s, x, y, label+" ", ui.Bold(ui.On(c, ground)))
		x = ui.Bar(s, x, y, hudBarCells, frac, ui.On(c, ground), off)
		if number >= 0 {
			st := bright
			if c == ui.ColRed && number <= lowHP {
				st = ui.Bold(ui.On(ui.ColRed, ground)) // nearly dead: the number turns red
			}
			x = ui.Put(s, x, y, fmt.Sprintf(" %-3d", number), st)
		}
		x += 3
		return start
	}
	parts.Health = meter(sy.Heart, ui.ColRed, float64(p.HP)/game.PlayerMaxHP, p.HP)
	parts.Armor = meter(sy.Armor, ui.ColBlue, float64(p.Armor)/game.MaxArmor, p.Armor)
	parts.Stamina = meter(sy.Stamina, ui.ColYellow, p.Stamina, -1)
	kills := fmt.Sprint(h.Kills)
	parts.Kills = x0 + w - 1 - len("KILLS ") - len(kills)
	ui.Put(s, ui.Put(s, parts.Kills, y, "KILLS ", dim), y, kills, bright)

	// Line 2, the left part.
	y++
	x = x0 + 1
	wp := *p.Weapon()
	st := wp.Kind.Stats()
	parts.Gun = x
	x = ui.Put(s, x, y, st.Name, bright)
	x = ui.Put(s, x, y, fmt.Sprintf(" %d/%d", wp.Mag, wp.Reserve), text) + 2
	parts.Mag = x
	if wp.Reloading() {
		x = ui.Put(s, x, y, "RELOADING", ui.Bold(ui.On(ui.ColYellow, ground)))
	} else {
		left := min(max(wp.Mag, 0), st.Mag)
		x = ui.Put(s, x, y, strings.Repeat(sy.MagOn, left), ui.On(ui.ColAmber, ground))
		x = ui.Put(s, x, y, strings.Repeat(sy.MagOff, st.Mag-left), off)
	}
	x += 2
	parts.Guns = x
	for k := range game.NumGuns {
		label, gst := fmt.Sprint(int(k)+1), off // a gun not found yet stays dark
		switch {
		case k == p.Gun:
			gst = ui.Bold(ui.On(ui.ColAmber, ground))
			if sy.Simple {
				label = "[" + label + "]" // the simple set can't rely on colors alone
			}
		case p.Guns[k].Owned:
			gst = dim
		}
		x = ui.Put(s, x, y, label, gst) + 1
	}
	leftEnd := x

	// Line 2, the right part: the longest form that leaves the left part
	// alone.
	stage, clock := stageText(h), Clock(h.Time)
	forms := [][2]string{{stage + "  TIME ", clock}, {stage + "  ", clock}, {"", clock}}
	var right [2]string
	for _, f := range forms {
		right = f
		if x0+w-1-ui.RuneLen(f[0]+f[1]) >= leftEnd+1 {
			break
		}
	}
	rx := x0 + w - 1 - ui.RuneLen(right[0]+right[1])
	parts.Stage = rx
	parts.Time = ui.Put(s, rx, y, right[0], text)
	ui.Put(s, parts.Time, y, right[1], bright)

	// The extras, in the gap.
	line := ""
	for _, e := range hudExtras(h) {
		next := e
		if line != "" {
			next = line + sy.Sep + e
		}
		if leftEnd+1+ui.RuneLen(next)+2 > rx {
			break
		}
		line = next
	}
	ui.Put(s, leftEnd+1, y, line, dim)
	return parts
}

// lowHP: at this much health or less the HUD's number turns red.
const lowHP = 25

// hudExtras are the optional parts of the HUD, in the order they're added
// while they fit.
func hudExtras(h HUD) []string {
	p := h.Player
	deg := int(math.Round(p.Angle*180/math.Pi)) % 360
	var extras []string
	if h.MinimapHidden {
		extras = append(extras, "minimap hidden: screen too small")
	}
	return append(extras,
		fmt.Sprintf("FPS %4.1f", h.FPS),
		h.Mode.String(),
		fmt.Sprintf("%3d° %s", deg, compass[dirIndex(p.Angle)]),
		fmt.Sprintf("draw %.2f present %.2f show %.2f ms", ms(h.Draw), ms(h.Present), ms(h.Show)),
		fmt.Sprintf("x %5.2f y %5.2f", p.X, p.Y),
		h.Keys,
	)
}

// Clock formats seconds as m:ss (e.g. 3:07).
func Clock(seconds float64) string {
	s := int(seconds)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
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

// stageText is the HUD's difficulty part: the tier in survival, the wave
// in a waves run (M15), with the break's countdown before it starts.
func stageText(h HUD) string {
	switch {
	case h.Wave == 0:
		return fmt.Sprintf("TIER %d", h.Tier)
	case h.Break > 0:
		return fmt.Sprintf("WAVE %d/%d  BREAK %s", h.Wave, game.NumWaves, Clock(math.Ceil(h.Break)))
	}
	return fmt.Sprintf("WAVE %d/%d", h.Wave, game.NumWaves)
}
