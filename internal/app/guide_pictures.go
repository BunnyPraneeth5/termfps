package app

import (
	"fmt"
	"math"
	"strings"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// Guide pages as pictures (U6, D54): the pages that were only text (tiers,
// what enemies learn, waves mode, health and stamina, settings, tips) are
// drawn with bars, charts and small top-down pictures. Every number still
// comes from the game code. Each fits the guide's content area at 80x24
// (74 x 18 cells) and stays that size in bigger windows.

// The picture pages' size.
const (
	pictureW = 74
	pictureH = 18
)

// pen draws on a picture page: positions are relative to the content
// area's top-left corner, on the panel's ground.
type pen struct {
	s    term.Screen
	x, y int
}

// put writes text at (x, y) in color c and returns the column after it.
func (p pen) put(x, y int, text string, c term.Color) int {
	return ui.Put(p.s, p.x+x, p.y+y, text, ui.On(c, ui.ColPanel)) - p.x
}

// bold is put in bold.
func (p pen) bold(x, y int, text string, c term.Color) int {
	return ui.Put(p.s, p.x+x, p.y+y, text, ui.Bold(ui.On(c, ui.ColPanel))) - p.x
}

// bar draws a bar of cells cells, frac of it in color c.
func (p pen) bar(x, y, cells int, frac float64, c term.Color) int {
	return ui.Bar(p.s, p.x+x, p.y+y, cells, frac, ui.On(c, ui.ColPanel), ui.On(ui.ColBorder, ui.ColPanel)) - p.x
}

// tierLimits are where the tier numbers stop rising: the top speed, the
// shortest time between spawns, the most enemies alive. Read from TierFor.
func tierLimits() (speed, every float64, alive int) {
	st := game.TierFor(200) // far past every limit
	return st.SpeedMul, st.SpawnEvery, st.MaxAlive
}

// tiersShown is how many tiers the tiers page lists.
const tiersShown = 6

// tiersMust are the lines the tiers page must show (tests check them, so
// the picture can't drift from the code).
func tiersMust() []string {
	speed, every, alive := tierLimits()
	return []string{
		fmt.Sprintf("A new tier every %d kills or %s, whichever comes first.", game.TierKills, render.Clock(game.TierTime)),
		fmt.Sprintf("every %s s", num(game.TierFor(2).SpawnEvery)),
		fmt.Sprintf("keeps rising: speed up to x%s, at most every %s s, at most %d alive", num(speed), num(every), alive),
	}
}

// drawTiers draws the tiers page: the rule, an example of the progress to
// the next tier, a table of tiers 1 to 6 as bars, when each enemy kind
// joins, and the limits.
func drawTiers(a *App, p pen) {
	sy, must := ui.Sym(), tiersMust()
	p.put(0, 0, must[0], ui.ColText)
	x := p.put(0, 1, "to next tier  ", ui.ColDim)
	exampleKills := game.TierKills * 7 / 20
	x = p.bar(x, 1, 10, float64(exampleKills)/game.TierKills, ui.ColAmber)
	x = p.put(x, 1, fmt.Sprintf(" %d/%d kills   or   ", exampleKills, game.TierKills), ui.ColText)
	x = p.bar(x, 1, 10, 0.5, ui.ColAmber)
	p.put(x, 1, fmt.Sprintf(" %s/%s", render.Clock(game.TierTime/2), render.Clock(game.TierTime)), ui.ColText)

	// Columns: tier, three bars with their multiplier, the spawn time, the
	// most alive as dots.
	barCells := 8
	if sy.Simple {
		barCells = 6 // the brackets take two columns
	}
	pitch := ui.BarWidth(barCells) + 7
	cols := [...]int{0, 6, 6 + pitch, 6 + 2*pitch, 6 + 3*pitch, 6 + 3*pitch + 13}
	for i, h := range []string{"TIER", "ENEMY HP", "DAMAGE", "SPEED", "NEW ENEMY", "MAX ALIVE"} {
		p.put(cols[i], 3, h, ui.ColDim)
	}
	top := game.TierFor(tiersShown)
	for t := 1; t <= tiersShown; t++ {
		st, y := game.TierFor(t), 3+t
		p.bold(1, y, fmt.Sprint(t), ui.ColBright)
		for i, v := range []struct {
			mul, max float64
			c        term.Color
		}{{st.HPMul, top.HPMul, ui.ColRed}, {st.DamageMul, top.DamageMul, ui.ColOrange}, {st.SpeedMul, top.SpeedMul, ui.ColYellow}} {
			x := p.bar(cols[1+i], y, barCells, v.mul/v.max, v.c)
			p.put(x, y, " x"+num(v.mul), ui.ColText)
		}
		p.put(cols[4], y, "every "+num(st.SpawnEvery)+" s", ui.ColText)
		p.put(cols[5], y, strings.Repeat(sy.DotOn, st.MaxAlive), ui.ColGreen)
	}

	// When each kind joins, from FirstTier.
	x = p.put(0, 11, "NEW KINDS  ", ui.ColDim)
	for t := 1; t <= tiersShown; t++ {
		var kinds []string
		for k := game.KindZombie; k <= game.KindExploder; k++ {
			if game.FirstTier(k) == t {
				kinds = append(kinds, k.String())
			}
		}
		if len(kinds) == 0 {
			continue
		}
		x = p.bold(x, 11, fmt.Sprintf("T%d ", t), ui.ColAmber)
		x = p.put(x, 11, strings.Join(kinds, ", "), ui.ColText) + 3
	}
	p.put(0, 13, must[2], ui.ColText)
	p.put(0, 15, "In waves mode the tier goes up every two waves instead.", ui.ColDim)
}

// learnArt are the four top-down pictures of the "enemies learn" page, 15
// columns by 5 rows each: @ you, z a zombie, s a shooter, # a wall,
// > < ^ v the way they walk, ) the sound of your shot, . a shot.
var learnArt = [4][5]string{
	{
		"               ",
		"     < < z     ",
		" @ < < < < < z ",
		"     < < z     ",
		"               ",
	},
	{
		"   ) )     ### ",
		" @ ) ) )   # z ",
		"   ) )     # v ",
		"           # v ",
		"   < < < < < < ",
	},
	{
		"               ",
		" @ . . .  ##   ",
		"          ## s ",
		"          ##   ",
		"               ",
	},
	{
		"       z       ",
		"       v       ",
		" z > > @ < < z ",
		"       ^       ",
		"       z       ",
	},
}

// artCell is how one character of a small picture is drawn: the text and
// its color. The fancy symbol set draws arrows and dots; the simple one
// keeps the plain characters.
func artCell(c rune) (string, term.Color) {
	fancy := !ui.Sym().Simple
	pick := func(f, s string) string {
		if fancy {
			return f
		}
		return s
	}
	switch c {
	case '@':
		return "@", ui.ColBright
	case 'z':
		return pick("●", "z"), ui.ColGreen
	case 's':
		return pick("●", "s"), ui.ColRed
	case 'B':
		return pick("●", "B"), ui.ColRed
	case '#':
		return "█", ui.ColBorder
	case '>':
		return pick("→", ">"), ui.ColDim
	case '<':
		return pick("←", "<"), ui.ColDim
	case '^':
		return pick("↑", "^"), ui.ColDim
	case 'v':
		return pick("↓", "v"), ui.ColDim
	case ')':
		return ")", ui.ColYellow
	case '.':
		return pick("·", "."), ui.ColAmber
	}
	return string(c), ui.ColText
}

// drawArt draws a small picture's rows from (x, y).
func drawArt(p pen, x, y int, rows []string) {
	for j, row := range rows {
		for i, c := range row {
			if c != ' ' {
				text, col := artCell(c)
				p.put(x+i, y+j, text, col)
			}
		}
	}
}

// learnCaptions are the two lines under each "enemies learn" box.
func learnCaptions() [4][2]string {
	return [4][2]string{
		{"straight at you", "the shortest way"},
		{"they hear you", fmt.Sprintf("%d steps, %sx", game.HearingSteps, num(game.RushMul))},
		{"shooters hide", "after they shoot"},
		{"zombies surround", "you"},
	}
}

// learnMust are the lines the "enemies learn" page must show.
func learnMust() []string {
	sy := ui.Sym()
	return []string{
		fmt.Sprintf("TIER %d", game.HearTier), fmt.Sprintf("TIER %d", game.CoverTier), fmt.Sprintf("TIER %d", game.FlankTier),
		learnCaptions()[1][1],
		fmt.Sprintf("%s without a kill %s horn %s the hunt: %sx faster until your next kill", render.Clock(game.CampTime), sy.Then, sy.Then, num(game.HuntMul)),
	}
}

// drawLearn draws the "enemies learn" page: four boxes with top-down
// pictures (tier 1 to 4), then the DON'T CAMP rule with its timer.
func drawLearn(a *App, p pen) {
	const boxW, gap = 17, 2
	tiers := [4]int{1, game.HearTier, game.CoverTier, game.FlankTier}
	captions := learnCaptions()
	for i := range learnArt {
		x := i * (boxW + gap)
		ui.Box(p.s, p.x+x, p.y, boxW, 7, ui.BoxRound, ui.StBorder)
		ui.BoxTitle(p.s, p.x+x, p.y, fmt.Sprintf("TIER %d", tiers[i]), ui.StTitle)
		drawArt(p, x+1, 1, learnArt[i][:])
		p.put(x+1, 7, captions[i][0], ui.ColText)
		p.put(x+1, 8, captions[i][1], ui.ColDim)
	}
	ui.Badge(p.s, p.x, p.y+10, "DON'T CAMP", ui.ColBg, ui.ColRed)
	p.put(0, 11, learnMust()[4], ui.ColText)
	x := p.put(0, 12, "0:00 ", ui.ColDim)
	x = p.bar(x, 12, 50, 0.8, ui.ColRed)
	p.put(x, 12, " "+render.Clock(game.CampTime), ui.ColDim)
	p.put(0, 14, "A kill starts the timer over. Breaks between waves don't count.", ui.ColDim)
}

// waveTierColors color the waves chart's bars by tier, cool to hot.
var waveTierColors = []term.Color{ui.ColGreen, ui.ColYellow, ui.ColAmber, ui.ColOrange, ui.ColRed}

// waveColor is the chart color of tier t (the last color for any higher).
func waveColor(t int) term.Color {
	return waveTierColors[min(max(t, 1), len(waveTierColors))-1]
}

// wavesMust are the lines the waves page must show.
func wavesMust() []string {
	sy := ui.Sym()
	return []string{
		fmt.Sprintf("%d waves with breaks: clear the last one to win.", game.NumWaves),
		fmt.Sprintf("WAVE 1 %s %s s break %s WAVE 2 %s ... %s WAVE %d %s %s WIN", line(1), num(game.WaveBreak), line(1), line(1), line(1), game.NumWaves, line(1), sy.Star),
		fmt.Sprintf("first break %s s", num(game.FirstWaveBreak)),
	}
}

// line is n horizontal line characters of the symbol set in use.
func line(n int) string {
	return strings.Repeat(ui.Sym().Boxes[ui.BoxSingle][4], n)
}

// drawWaves draws the waves page: a bar chart of the enemies per wave,
// colored by tier, then the run as a timeline.
func drawWaves(a *App, p pen) {
	sy, must := ui.Sym(), wavesMust()
	p.put(0, 0, must[0], ui.ColText)
	const (
		barW, pitch = 4, 6
		chartRows   = 7
		baseY       = 9 // the row of the bars' feet
	)
	left := (pictureW - game.NumWaves*pitch) / 2
	biggest := game.WaveSize(game.NumWaves)
	for n := 1; n <= game.NumWaves; n++ {
		size := game.WaveSize(n)
		h := max(int(math.Ceil(float64(size)*chartRows/float64(biggest))), 1)
		x := left + (n-1)*pitch
		for j := 0; j < h; j++ {
			ui.Put(p.s, p.x+x, p.y+baseY-j, strings.Repeat(" ", barW), term.Style{Bg: waveColor(game.WaveTier(n))})
		}
		p.bold(x+(barW-len(fmt.Sprint(size)))/2, baseY-h, fmt.Sprint(size), ui.ColBright)
		p.put(x, baseY+1, fmt.Sprintf("W%d", n), ui.ColDim)
	}
	p.put(0, 2, "enemies", ui.ColDim)
	x := left
	for t := 1; t <= game.WaveTier(game.NumWaves); t++ {
		ui.Put(p.s, p.x+x, p.y+baseY+2, "  ", term.Style{Bg: waveColor(t)})
		x = p.put(x+3, baseY+2, fmt.Sprintf("tier %d", t), ui.ColDim) + 3
	}
	p.put(0, 13, must[1], ui.ColAmber)
	x = p.put(0, 14, must[2], ui.ColText)
	p.put(x, 14, sy.Sep+"use the breaks to reload and pick things up", ui.ColText)
	p.put(0, 16, "Your best wave and your fastest win are kept for every map.", ui.ColDim)
}

// healthMust are the lines the health and stamina page must show.
func healthMust() []string {
	sy := ui.Sym()
	return []string{
		fmt.Sprintf("hit %s wait %s s %s +%s HP every second, up to %d", sy.Then, num(game.RegenDelay), sy.Then, num(game.RegenPerSec), game.RegenMaxHP),
		fmt.Sprintf("medkit +%d HP, up to %d", game.MedkitHeal, game.PlayerMaxHP),
		fmt.Sprintf("takes half of every hit until it runs out%svest +%d, up to %d", sy.Sep, game.ArmorPickup, game.MaxArmor),
		fmt.Sprintf("F: sprint at %sx%sfull lasts %s s%srefills in %s s", num(game.SprintMul), sy.Sep, num(1/game.StaminaDrain), sy.Sep, num(1/game.StaminaRegen)),
		fmt.Sprintf("empty %s no sprint until it's back to %d%%", sy.Then, int(math.Round(game.StaminaResume*100))),
	}
}

// drawHealth draws the health and stamina page: the health bar with the
// line regeneration stops at, armor, and the stamina bar with the line
// sprinting comes back at.
func drawHealth(a *App, p pen) {
	sy, must := ui.Sym(), healthMust()
	const cells = 40
	up := sy.Up
	if sy.Simple {
		up = "^"
	}
	// mark puts a tick under a bar whose first cell is in column barX, at
	// share at of it, with what it marks.
	mark := func(y, barX int, at float64, label string) {
		p.put(barX+int(math.Round(at*cells)), y, up+" "+label, ui.ColDim)
	}

	p.bold(0, 0, "HEALTH", ui.ColRed)
	// Up to RegenMaxHP it heals by itself (bright); above, only medkits.
	regen := float64(game.RegenMaxHP) / game.PlayerMaxHP
	n := int(math.Round(regen * cells))
	x := p.put(0, 1, sy.Heart+" "+sy.BarOpen, ui.ColRed)
	barX := x
	x = p.put(x, 1, strings.Repeat(sy.BarOn, n), ui.ColRed)
	x = p.put(x, 1, strings.Repeat(sy.BarOn, cells-n), scaleColor(ui.ColRed, 0.45))
	x = p.put(x, 1, sy.BarClose, ui.ColRed)
	p.bold(x+1, 1, fmt.Sprint(game.PlayerMaxHP), ui.ColBright)
	p.put(barX, 2, "heals by itself", ui.ColDim)
	mark(2, barX, regen, fmt.Sprintf("%d: above it, medkits only", game.RegenMaxHP))
	p.put(2, 3, must[0], ui.ColText)
	p.put(2, 4, must[1], ui.ColText)

	p.bold(0, 6, "ARMOR", ui.ColBlue)
	x = p.put(0, 7, sy.Armor+" ", ui.ColBlue)
	p.bar(x, 7, 10, 0.5, ui.ColBlue)
	p.put(2, 8, must[2], ui.ColText)

	p.bold(0, 10, "STAMINA", ui.ColYellow)
	x = p.put(0, 11, sy.Stamina+" ", ui.ColYellow)
	p.bar(x, 11, cells, 1, ui.ColYellow)
	mark(12, x+ui.RuneLen(sy.BarOpen), game.StaminaResume, fmt.Sprintf("%d%%: sprinting comes back here", int(math.Round(game.StaminaResume*100))))
	p.put(2, 13, must[3], ui.ColText)
	p.put(2, 14, must[4], ui.ColText)
}

// Settings page numbers: the four render-mode views.
const (
	modeViewW = 16
	modeViewH = 5
)

// settingsMust are the lines the settings page must show.
func settingsMust() []string {
	return []string{"half-block", "shade", "ascii", "mixed", "Every setting takes effect at once and is saved."}
}

// drawSettingsPicture draws the settings page: the same wall in the four
// render modes side by side, and a small preview for the other settings.
func (g *guidePage) drawSettingsPicture(a *App, p pen) {
	sy := ui.Sym()
	p.put(0, 0, "Render mode: the same wall four ways (V switches it in the game)", ui.ColText)
	if g.preview == nil || g.shown != g.index {
		g.preview, g.shown = a.newPreview(guideView{}), g.index
		g.preview.Player.X, g.preview.Player.Y, g.preview.Player.Angle = 2.2, 3.1, 0.55 // looking into a corner of the room
	}
	g.view.Resize(modeViewW, 2*modeViewH)
	g.view.PixelAspect, g.view.Textures = a.frame.PixelAspect, true
	render.DrawFrame(g.preview, &g.view)
	for i, mode := range []game.RenderMode{game.ModeHalfBlock, game.ModeShade, game.ModeASCII, game.ModeMixed} {
		x := i * (modeViewW + 3)
		g.view.OffsetX, g.view.OffsetY = p.x+x, p.y+1
		a.presentAs(&g.view, mode)
		p.put(x, 1+modeViewH, mode.String(), ui.ColAmber)
	}

	// One line per other setting: its name, and what it looks like.
	rows := []struct {
		name string
		draw func(x, y int)
	}{
		{"Symbols", func(x, y int) {
			x = p.put(x, y, "♥ ▰▰▰▱▱ ▶", ui.ColRed)
			x = p.put(x, y, " or ", ui.ColDim)
			x = p.put(x, y, "HP [###--] >", ui.ColRed)
			p.put(x, y, "  (the simple set)", ui.ColDim)
		}},
		{"Volume", func(x, y int) {
			x = ui.LevelBar(p.s, p.x+x, p.y+y, 10, 0.8, ui.On(ui.ColAmber, ui.ColPanel), ui.On(ui.ColBorder, ui.ColPanel)) - p.x
			p.put(x, y, " 80%   the menu sounds follow it too", ui.ColDim)
		}},
		{"Wall textures", func(x, y int) {
			x = p.put(x, y, ui.Switch(true), ui.ColText)
			p.put(x, y, "  brick, stone, metal; off gives flat walls", ui.ColDim)
		}},
		{"Minimap fog", func(x, y int) {
			x = p.put(x, y, "███", ui.ColText)
			x = p.put(x, y, "███", scaleColor(ui.ColBorder, 0.5))
			p.put(x, y, "  the minimap shows only what you've seen", ui.ColDim)
		}},
		{"Render scale", func(x, y int) {
			p.put(x, y, "auto"+sy.Sep+"full"+sy.Sep+"half   auto keeps big windows smooth", ui.ColDim)
		}},
		{"Key mode", func(x, y int) { p.put(x, y, "async (Windows): move and turn at once"+sy.Sep+"events", ui.ColDim) }},
		{"Map unlocks", func(x, y int) { p.put(x, y, strings.Join(unlockRuleLabels, sy.Sep), ui.ColDim) }},
		{"Title, animations", func(x, y int) { p.put(x, y, "the map tour and the animations can be switched off", ui.ColDim) }},
	}
	for i, r := range rows {
		p.put(0, 8+i, r.name, ui.ColText)
		r.draw(19, 8+i)
	}
	p.put(0, 17, settingsMust()[4], ui.ColText)
}

// presentAs writes f to the screen in render mode mode, whatever the
// game's own mode is (the guide's settings page shows all four).
func (a *App) presentAs(f *render.Frame, mode game.RenderMode) {
	old := a.state.Mode
	a.state.Mode = mode
	a.presentFrame(f)
	a.state.Mode = old
}

// tip is one tip with its small picture: rows of picture characters (see
// artCell), three columns wide at most five.
type tip struct {
	art  string
	text string
}

// guideTips are the tips, each with a small picture beside it.
var guideTips = []tip{
	{"@ > >", "Keep moving: standing still lets enemies surround you, and camping starts the hunt."},
	{"< @ z", "Walk backwards while you shoot zombies and tanks: they can't catch you."},
	{"@ ...", "Reload in quiet moments; switching guns cancels a reload."},
	{"@ . s", "Save the rifle for tanks and shooters far away, the shotgun for close calls."},
	{"@ # z", "Grab armor when you can: it halves the damage you take."},
	{"@ # s", "Use corners and cover against shooters; listen for where sounds come from."},
	{"z B z", "Lure groups to red barrels, and shoot exploders before they reach you."},
}

// drawTips draws the tips page: every tip with its picture, two rows each.
func drawTips(a *App, p pen) {
	const textX = 9
	y := 0
	for _, t := range guideTips {
		ui.Fill(p.s, p.x, p.y+y, 7, 1, ui.StBg) // a dark tile behind the picture
		for i, c := range t.art {
			if c != ' ' {
				text, col := artCell(c)
				ui.Put(p.s, p.x+1+i, p.y+y, text, ui.On(col, ui.ColBg))
			}
		}
		lines := wrap(t.text, pictureW-textX)
		for i, l := range lines {
			p.put(textX, y+i, l, ui.ColText)
		}
		y += max(len(lines), 2)
	}
}
