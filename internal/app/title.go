package app

import (
	"fmt"
	"math"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// The title screen (U2, D54): the TERM FPS logo with a slow shimmer, a
// random tagline, the menu with a description line, the best run, the
// version, and (if it's on) the slow map tour behind it all.

// logoText is the logo, drawn in the block font.
const logoText = "TERM FPS"

// taglines are the lines one of which shows under the logo, picked at
// random each start (like Minecraft's splash text).
var taglines = []string{
	"a first-person shooter game in your terminal",
	"the shooter that lives in your terminal",
	"survive the horde, one character at a time",
	"how long can you last?",
	"keep moving. keep shooting.",
	"made of text, full of teeth",
	"mind the red barrels",
	"now with exploders",
	"don't camp",
	"no mouse-look required",
	"rendered one character at a time",
	"the walls are ASCII, the danger is real",
}

// Logo shimmer and tagline pulse numbers.
const (
	shimmerEvery  = 6.0 // seconds between light bands over the logo
	shimmerLetter = 0.2 // seconds the band takes per letter
	shimmerWidth  = 4   // columns the band is wide
	pulsePeriod   = 2.4 // seconds for the tagline to dim and come back
	pulseDepth    = 0.25
)

// Title menu items, in order.
const (
	titlePlay = iota
	titleGuide
	titleSettings
	titleQuit
	titleItems
)

// titleItem is one line of the title menu.
type titleItem struct {
	label, note string // the note shows dim, at the right
	desc        string // the description line under the menu
}

// titleScreen is the title screen.
type titleScreen struct {
	sel  int
	t    float64 // seconds it has been showing, for the shimmer and the pulse
	hits hitList
}

// titleMenu returns a fresh title screen.
func (a *App) titleMenu() *titleScreen {
	return &titleScreen{}
}

// items builds the menu's lines (the map count changes as maps unlock).
func (ts *titleScreen) items(a *App) [titleItems]titleItem {
	sep := ui.Sym().Sep
	locked := 0
	for i := range a.maps {
		if a.mapLocked(i) {
			locked++
		}
	}
	maps := fmt.Sprintf("%d maps", len(a.maps))
	if len(a.maps) == 1 {
		maps = "1 map"
	}
	if locked > 0 {
		maps += fmt.Sprintf("%s%d locked", sep, locked)
	}
	return [titleItems]titleItem{
		titlePlay:     {"PLAY", "survival or waves", "Pick a mode and a map, then go. " + maps + "."},
		titleGuide:    {"GUIDE", "how to play", "Enemies, weapons, maps and tips, with the game's own numbers."},
		titleSettings: {"SETTINGS", "video, audio, controls", "Video, audio, controls and gameplay. Changes apply at once."},
		titleQuit:     {"QUIT", "", "Leave the game. Your progress is already saved."},
	}
}

func (ts *titleScreen) key(a *App, ev term.Key) {
	if ev.Action == term.Release {
		return
	}
	k, ok := input.MenuKeyFor(ev)
	if !ok {
		return
	}
	switch k {
	case ui.KeyUp:
		ts.sel = (ts.sel + titleItems - 1) % titleItems
		a.menuSound(audio.MenuTick)
	case ui.KeyDown:
		ts.sel = (ts.sel + 1) % titleItems
		a.menuSound(audio.MenuTick)
	case ui.KeyEnter:
		ts.choose(a)
	}
	// Esc on the title screen does nothing; Quit or Ctrl+C quits.
}

// mouse: pressing the left button on an item picks it.
func (ts *titleScreen) mouse(a *App, b mouseButton, down bool, x, y int) {
	if b != mouseLeft || !down {
		return
	}
	if id, ok := ts.hits.at(x, y); ok {
		ts.sel = id
		ts.choose(a)
	}
}

// choose acts on the selected item.
func (ts *titleScreen) choose(a *App) {
	a.menuSound(audio.MenuClick)
	switch ts.sel {
	case titlePlay:
		a.switchTo(a.playMenu())
	case titleGuide:
		a.switchTo(a.guideMenu(ts, dropTitle))
	case titleSettings:
		a.switchTo(a.settingsMenu(ts, dropTitle))
	case titleQuit:
		a.quit = true
	}
}

// bestRun is the title's best-run line: the longest survival run over all
// maps, or "" before the first run.
func (a *App) bestRun() string {
	name, found := "", false
	for _, m := range a.maps { // in map order, so equal bests don't flip between frames
		if b, ok := a.progress.Best[m.Name]; ok && (!found || b.Time > a.progress.Best[name].Time) {
			name, found = m.Name, true
		}
	}
	if !found {
		return ""
	}
	b, sep := a.progress.Best[name], ui.Sym().Sep
	return fmt.Sprintf("best on %s %s%s%d kills%stier %d", name, render.Clock(b.Time), sep, b.Kills, sep, b.Tier)
}

func (ts *titleScreen) tick(a *App, now time.Time, dt float64) {
	ts.t += dt
	a.drawBackdrop(dropTitle, dt)
	s, sy := a.scr, ui.Sym()
	ox, oy := stage(s)
	w, h := s.Size()

	// The logo: rows from amber to orange; every shimmerEvery seconds a
	// light band passes over it, a letter per shimmerLetter seconds.
	band := -1000.0
	if a.cfg.Animations {
		pitch := float64(ui.FontCols + ui.LetterGap)
		band = math.Mod(ts.t, shimmerEvery)/shimmerLetter*pitch - shimmerWidth
	}
	lx := ox + (stageW-ui.BigWidth(logoText))/2
	ui.BigText(s, lx, oy+1, logoText, ui.FontRows, func(row, col int) term.Style {
		c := ui.LogoRows[row]
		if d := float64(col+row) - band; d >= 0 && d < shimmerWidth { // +row slants the band
			c = ui.ColBright
		}
		return term.Style{Fg: c, Bg: ui.ColBg}
	})

	// The tagline, gently pulsing.
	k := 1.0
	if a.cfg.Animations {
		k = 1 - pulseDepth/2 + pulseDepth/2*math.Cos(2*math.Pi*ts.t/pulsePeriod)
	}
	ui.Center(s, ox, stageW, oy+7, " "+a.tagline+" ", ui.On(scaleColor(ui.ColYellow, k), ui.ColBg))

	// The menu, in a double-line box.
	const boxW = 46
	bx, by := ox+(stageW-boxW)/2, oy+9
	items := ts.items(a)
	ui.Panel(s, bx, by, boxW, titleItems+2, ui.BoxDouble, ui.StBorder, ui.StPanel)
	ts.hits = ts.hits[:0]
	for i, it := range items {
		y := by + 1 + i
		st, note := ui.StPanel, ui.StDim
		marker := "  "
		if i == ts.sel {
			st, note = ui.StSelected, ui.StSelected
			marker = sy.Select + " "
			ui.Fill(s, bx+1, y, boxW-2, 1, st)
		}
		ui.Put(s, bx+2, y, marker+it.label, st)
		ui.PutRight(s, bx+boxW-2, y, it.note, note)
		ts.hits.add(bx+1, y, boxW-2, 1, i)
	}
	ui.Center(s, ox, stageW, by+titleItems+2, " "+sy.Arrow+" "+items[ts.sel].desc+" ", ui.StBg)

	y := by + titleItems + 4
	if best := a.bestRun(); best != "" {
		ui.Center(s, ox, stageW, y, " "+best+" ", ui.On(ui.ColDim, ui.ColBg))
	}
	a.drawNotes(ox, y+1) // sound and save problems

	ui.Hints(s, ox, stageW, oy+stageH-2, []ui.Hint{{Keys: sy.UpDown, Text: "select"}, {Keys: sy.Enter, Text: "choose"}}, ui.ColBg)
	ui.PutRight(s, w-1, h-1, Version, ui.On(ui.ColDim, ui.ColBg))
	s.Show()
}

// drawNotes writes the save and sound problems (if any) as yellow lines,
// centered on the stage from row y, at most three.
func (a *App) drawNotes(ox, y int) {
	for i, l := range a.messages() {
		if i == 3 {
			break
		}
		ui.Center(a.scr, ox, stageW, y+i, " "+l+" ", ui.On(ui.ColYellow, ui.ColBg))
	}
}
