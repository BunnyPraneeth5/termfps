package app

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// The end screens (U5, D54): game over and, in waves mode, the win. Big
// block letters, what killed you, the run's numbers in boxes with NEW BEST
// badges (against the best on that map), what you killed, and buttons.

// End screen buttons, in order.
const (
	endAgain = iota
	endChangeMap
	endMainMenu
	endButtons
)

// End screen animations (only with the Animations setting on).
const (
	endRowTime   = 0.1 // seconds per row of the big letters
	endCountTime = 0.8 // seconds the numbers take to count up
	endBlinkTime = 1.6 // seconds NEW BEST blinks before it stays on
	endBlinkRate = 0.2 // seconds per blink half
)

// statBox is one number of the run, for a box on the end screen.
type statBox struct {
	label string
	// value is the number counted up to; show formats it (a clock, a
	// percent, a count).
	value float64
	show  func(v float64) string
	// newBest marks the value as the best on this map so far; else best is
	// the line under it ("best 5"), or "" for none.
	newBest bool
	best    string
}

// endScreen is the game over or win screen. Its boxes are worked out once,
// when the run ends, so later changes to the bests don't change them.
type endScreen struct {
	won      bool
	sel      int
	t        float64 // seconds it has been showing, for the animations
	headline string  // under the big letters
	boxes    []statBox
	lines    []string // "this run: ..." and the line under it
	hits     hitList
}

// count formats a whole number.
func count(v float64) string { return fmt.Sprint(int(math.Round(v))) }

// killedBy says what killed the player, or "" when the game doesn't know.
func killedBy(c game.Cause) string {
	switch c.Kind {
	case game.CauseEnemy:
		if c.Enemy == game.KindExploder {
			return "killed by an exploder"
		}
		return "killed by a " + c.Enemy.String()
	case game.CauseExplosion:
		return "killed by an explosion"
	case game.CauseAcid:
		return "killed by acid"
	}
	return ""
}

// killsLine lists the run's kills per enemy kind, most first: "4 tanks ·
// 2 exploders · 31 runners".
func killsLine(by [game.NumEnemyKinds]int) string {
	var parts []string
	for range by { // pick the biggest one left, as often as there are kinds
		best := -1
		for k, n := range by {
			if n > 0 && (best < 0 || n > by[best]) {
				best = k
			}
		}
		if best < 0 {
			break
		}
		parts = append(parts, plural(by[best], game.EnemyKind(best).String()))
		by[best] = 0 // by is a copy (arrays are passed by value), so this is safe
	}
	if len(parts) == 0 {
		return "no kills"
	}
	return strings.Join(parts, ui.Sym().Sep)
}

// endMenu builds the end screen for the run that just ended. endRun has
// already counted it; a.prevBest and a.prevWaves are the bests before it.
func (a *App) endMenu(won bool) *endScreen {
	st, sep := a.state, ui.Sym().Sep
	m := a.maps[a.mapIndex]
	es := &endScreen{won: won}
	name := m.Name
	if name == "" {
		name = "this map"
	}

	var head []string
	switch {
	case won:
		head = append(head, fmt.Sprintf("all %d waves cleared", game.NumWaves))
	case killedBy(st.Stats.Killer) != "":
		head = append(head, killedBy(st.Stats.Killer))
	}
	mode := st.Play.String()
	if won {
		mode += " mode"
	}
	es.headline = strings.Join(append(head, name, mode), sep)

	acc := statBox{label: "ACCURACY", show: func(v float64) string { return fmt.Sprintf("%d%%", int(math.Round(v))) }}
	if share, ok := st.Stats.Accuracy(); ok {
		acc.value = 100 * share
		acc.best = fmt.Sprintf("%d of %d shots", st.Stats.Hits, st.Stats.Shots)
	} else {
		acc.show = func(float64) string { return "-" }
		acc.best = "no shots fired"
	}
	// better says whether v beats the old best, and the line to show when
	// it doesn't.
	box := func(label string, v, old float64, show func(float64) string) statBox {
		b := statBox{label: label, value: v, show: show, newBest: v > old}
		if !b.newBest && old > 0 {
			b.best = "best " + show(old)
		}
		return b
	}
	timeBox := statBox{label: "TIME", value: st.Time, show: render.Clock}
	switch {
	case won:
		// The best time of a win is the fastest one.
		pw := a.prevWaves
		timeBox.newBest = pw.Wins == 0 || st.Time < pw.WinTime
		if !timeBox.newBest {
			timeBox.best = "best " + render.Clock(pw.WinTime)
		}
		es.boxes = []statBox{timeBox, box("KILLS", float64(st.Kills), float64(pw.Kills), count), acc,
			{label: "HP LEFT", value: float64(st.Player.HP), show: count, best: fmt.Sprintf("of %d", game.PlayerMaxHP)}}
	case st.Play == game.Waves:
		pw := a.prevWaves
		wave := box("WAVE", float64(st.Wave), float64(pw.Wave), func(v float64) string { return fmt.Sprintf("%d/%d", int(math.Round(v)), game.NumWaves) })
		es.boxes = []statBox{wave, box("KILLS", float64(st.Kills), float64(pw.Kills), count), timeBox, acc}
	default:
		pb := a.prevBest
		es.boxes = []statBox{box("TIME", st.Time, pb.Time, render.Clock), box("KILLS", float64(st.Kills), float64(pb.Kills), count),
			box("TIER", float64(st.Tier), float64(pb.Tier), count), acc}
	}

	es.lines = []string{
		"this run: " + killsLine(st.Stats.KillsBy),
		fmt.Sprintf("explored %d%% of the map%sdamage taken %d", percent(exploredShare(m, st.Seen)), sep, st.Stats.DamageTaken),
	}
	return es
}

func (es *endScreen) key(a *App, ev term.Key) {
	if ev.Action == term.Release {
		return
	}
	// R isn't a key here: it's reload, and a held R would pick a button by
	// accident. Up and Down aren't either: W and S may still be held.
	k, ok := input.MenuKeyFor(ev)
	if !ok {
		return
	}
	switch k {
	case ui.KeyLeft:
		es.sel = (es.sel + endButtons - 1) % endButtons
		a.menuSound(audio.MenuTick)
	case ui.KeyRight:
		es.sel = (es.sel + 1) % endButtons
		a.menuSound(audio.MenuTick)
	case ui.KeyEnter:
		es.choose(a)
	case ui.KeyBack:
		a.menuSound(audio.MenuBack)
		a.switchTo(a.titleMenu())
	}
}

func (es *endScreen) mouse(a *App, b mouseButton, down bool, x, y int) {
	if b != mouseLeft || !down {
		return
	}
	if id, ok := es.hits.at(x, y); ok {
		es.sel = id
		es.choose(a)
	}
}

// choose acts on the selected button.
func (es *endScreen) choose(a *App) {
	a.menuSound(audio.MenuClick)
	switch es.sel {
	case endAgain:
		a.newRun()
	case endChangeMap:
		a.switchTo(a.playMenu())
	case endMainMenu:
		a.switchTo(a.titleMenu())
	}
}

// Stat box size, and where the boxes start on the stage.
const (
	statW   = 17
	statGap = 2
	statY   = 9
)

func (es *endScreen) tick(a *App, now time.Time, dt float64) {
	es.t += dt
	a.drawBackdrop(dropGame, dt)
	s, sy := a.scr, ui.Sym()
	ox, oy := stage(s)
	anim := a.cfg.Animations

	// The big letters, row by row.
	word, color, again := "YOU DIED", ui.ColRed, "TRY AGAIN"
	if es.won {
		word, color, again = "YOU WON", ui.ColGreen, "PLAY AGAIN"
	}
	rows := ui.FontRows
	if anim {
		rows = min(int(es.t/endRowTime)+1, ui.FontRows)
	}
	lx := ox + (stageW-ui.BigWidth(word))/2
	ui.BigText(s, lx, oy+1, word, rows, func(row, col int) term.Style { return term.Style{Fg: color, Bg: ui.ColBg} })
	if es.won { // a few amber stars round the word
		for _, p := range [][2]int{{-5, 1}, {-3, 4}, {ui.BigWidth(word) + 2, 2}, {ui.BigWidth(word) + 4, 5}, {-8, 3}, {ui.BigWidth(word) + 7, 4}} {
			ui.Put(s, lx+p[0], oy+p[1], sy.Star, ui.Bold(ui.On(ui.ColAmber, ui.ColBg)))
		}
	}
	ui.Center(s, ox, stageW, oy+7, " "+es.headline+" ", ui.StBg)

	// The stat boxes: the numbers count up, NEW BEST blinks a few times.
	grow := 1.0
	if anim {
		grow = min(es.t/endCountTime, 1)
	}
	blinkOn := !anim || es.t >= endBlinkTime || int(es.t/endBlinkRate)%2 == 0
	total := len(es.boxes)*statW + (len(es.boxes)-1)*statGap
	x := ox + (stageW-total)/2
	card := ui.On(ui.ColText, ui.ColCard)
	for _, b := range es.boxes {
		ui.Panel(s, x, oy+statY, statW, 5, ui.BoxRound, ui.On(ui.ColBorder, ui.ColCard), card)
		ui.Center(s, x, statW, oy+statY+1, b.label, ui.On(ui.ColDim, ui.ColCard))
		ui.Center(s, x, statW, oy+statY+2, b.show(b.value*grow), ui.Bold(ui.On(ui.ColBright, ui.ColCard)))
		switch {
		case b.newBest && blinkOn:
			ui.Center(s, x, statW, oy+statY+3, " NEW BEST ", term.Style{Fg: ui.ColBg, Bg: ui.ColGreen, Bold: true})
		case !b.newBest:
			ui.Center(s, x, statW, oy+statY+3, b.best, ui.On(ui.ColDim, ui.ColCard))
		}
		x += statW + statGap
	}

	for i, l := range es.lines {
		ui.Center(s, ox, stageW, oy+15+i, " "+clip(l, stageW-4)+" ", ui.StBg)
	}

	// The buttons.
	labels := [endButtons]string{again, "CHANGE MAP", "MAIN MENU"}
	width := 0
	for _, l := range labels {
		width += ui.RuneLen(l) + 6
	}
	width += 2 * (endButtons - 1)
	x = ox + (stageW-width)/2
	es.hits = es.hits[:0]
	for i, l := range labels {
		text, st := "   "+l+"   ", ui.On(ui.ColText, ui.ColPanel)
		if i == es.sel {
			text, st = " "+sy.Select+" "+l+"   ", ui.StSelected
		}
		es.hits.add(x, oy+18, ui.RuneLen(text), 1, i)
		x = ui.Put(s, x, oy+18, text, st) + 2
	}
	a.drawNotes(ox, oy+19) // a save or sound problem
	ui.Hints(s, ox, stageW, oy+stageH-2, []ui.Hint{{Keys: sy.LeftRight, Text: "pick"}, {Keys: sy.Enter, Text: "choose"}, {Keys: "esc", Text: "main menu"}}, ui.ColBg)
	a.show()
}
