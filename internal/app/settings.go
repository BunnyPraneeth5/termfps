package app

import (
	"fmt"
	"slices"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// The Settings screen (U3, D54): four tabs; every setting has its value
// and one dim help line; changes apply at once and are saved.

// The tabs, in order.
const (
	tabVideo = iota
	tabAudio
	tabControls
	tabGameplay
	numTabs
)

var tabNames = []string{"VIDEO", "AUDIO", "CONTROLS", "GAMEPLAY"}

// setting is one line of the Settings screen.
type setting struct {
	tab         int
	label, help string
	values      []string // the choices, as shown
	value       int      // index into values
	toggle      bool     // an off/on setting: drawn as a switch
	// levels, when set, draws the setting as a bar: how full (0-1) each
	// value is.
	levels []float64
	// set applies value v to the app. It's a function value: each setting
	// carries its own code.
	set func(a *App, v int)
}

// scaleChoices are the render scale settings (V6), in menu order.
var scaleChoices = []string{"auto", "full", "half"}

// symbolChoices are the Symbols setting's values (U1): fancy draws ♥ ▰ ▶
// and rounded boxes; simple only characters every console font has.
var symbolChoices = []string{"fancy", "simple"}

// thresholdLabel is the menu text for a color threshold.
func thresholdLabel(n int) string {
	if n == 0 {
		return "off"
	}
	return fmt.Sprint(n)
}

// boolIndex is 1 for true and 0 for false: the index of an off/on value.
func boolIndex(on bool) int {
	if on {
		return 1
	}
	return 0
}

// mapStrings turns each value into its menu text. [T any] makes it a
// generic function: it works for a slice of any type T.
func mapStrings[T any](vs []T, str func(T) string) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = str(v)
	}
	return out
}

// with returns choices with current added (in order) if it's missing, so
// a value from a flag or typed into save.json stays choosable.
func with[T int | float64](choices []T, current T) []T {
	if !slices.Contains(choices, current) {
		choices = append(choices, current)
		slices.Sort(choices)
	}
	return choices
}

// percentLevels are the bar fills for percent values.
func percentLevels(percents []int) []float64 {
	out := make([]float64, len(percents))
	for i, p := range percents {
		out[i] = float64(p) / 100
	}
	return out
}

// settingsList builds every setting from the current configuration.
func (a *App) settingsList() []setting {
	sep := ui.Sym().Sep
	var modes []game.RenderMode
	for m := game.ModeHalfBlock; ; m = m.Next() { // every mode, in V order
		modes = append(modes, m)
		if m.Next() == game.ModeHalfBlock {
			break
		}
	}
	fpsChoices := with([]int{30, 60}, a.cfg.FPS)
	keyChoices := []string{"events"}
	if input.AsyncSupported() == nil {
		keyChoices = []string{"async", "events"}
	}
	tens := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	backgrounds := with(slices.Clone(tens), a.cfg.MixedBackground) // e.g. 25 typed into save.json
	volumes := with(slices.Clone(tens), a.cfg.Volume)
	thresholds := with([]int{0, 4, 8, 16, 32}, a.cfg.ColorThreshold)
	var aspects []float64
	for tenths := 5; tenths <= 20; tenths++ {
		aspects = append(aspects, float64(tenths)/10)
	}
	aspects = with(aspects, a.cfg.PixelAspect) // e.g. -aspect 1.25
	pct := func(n int) string { return fmt.Sprintf("%d%%", n) }
	onOff := func(tab int, label, help string, on bool, set func(a *App, on bool)) setting {
		return setting{tab: tab, label: label, help: help, values: []string{"off", "on"}, value: boolIndex(on), toggle: true,
			set: func(a *App, v int) { set(a, v == 1) }}
	}

	return []setting{
		{tab: tabVideo, label: "Render mode", help: "half-block" + sep + "shade" + sep + "ASCII" + sep + "mixed; V switches it in the game",
			values: mapStrings(modes, game.RenderMode.String), value: slices.Index(modes, a.state.Mode),
			set: func(a *App, v int) { a.state.Mode = modes[v] }},
		{tab: tabVideo, label: "Render scale", help: "auto keeps big windows smooth; half draws faster, full is sharpest",
			values: scaleChoices, value: max(slices.Index(scaleChoices, a.cfg.RenderScale), 0), // "" (unset) shows auto
			set: func(a *App, v int) {
				a.cfg.RenderScale = scaleChoices[v]
				a.resizeFrame()
			}},
		{tab: tabVideo, label: "FPS", help: "frames a second: 30 or 60",
			values: mapStrings(fpsChoices, func(n int) string { return fmt.Sprint(n) }), value: slices.Index(fpsChoices, a.cfg.FPS),
			set: func(a *App, v int) { a.cfg.FPS = fpsChoices[v] }}, // Run resets its ticker
		onOff(tabVideo, "Wall textures", "brick, stone and metal on the walls; off gives flat walls", a.cfg.Textures, func(a *App, on bool) {
			a.cfg.Textures = on
			a.frame.Textures = on
		}),
		{tab: tabVideo, label: "Mixed background", help: "how bright the fill behind mixed-mode characters is",
			values: mapStrings(backgrounds, pct), value: slices.Index(backgrounds, a.cfg.MixedBackground), levels: percentLevels(backgrounds),
			set: func(a *App, v int) {
				a.cfg.MixedBackground = backgrounds[v]
				a.frame.MixedBackground = float64(a.cfg.MixedBackground) / 100
			}},
		{tab: tabVideo, label: "Symbols", help: "pick simple if you see ? or empty boxes in the menus and the HUD",
			values: symbolChoices, value: boolIndex(a.cfg.SimpleSymbols),
			set: func(a *App, v int) {
				a.cfg.SimpleSymbols = v == 1
				ui.UseSimple(a.cfg.SimpleSymbols)
			}},
		onOff(tabVideo, "Title background", "the slow map tour behind the title; off for slow terminals", a.cfg.TitleBackground,
			func(a *App, on bool) { a.cfg.TitleBackground = on }),
		onOff(tabVideo, "Animations", "logo shimmer, sliding notices, counting numbers", a.cfg.Animations,
			func(a *App, on bool) { a.cfg.Animations = on }),

		onOff(tabAudio, "Sound", "stereo, from where things happen; the menu sounds follow it too", a.cfg.Sound, func(a *App, on bool) {
			a.cfg.Sound = on
			a.applySound()
		}),
		{tab: tabAudio, label: "Volume", help: "left and right change it by 10%",
			values: mapStrings(volumes, pct), value: slices.Index(volumes, a.cfg.Volume), levels: percentLevels(volumes),
			set: func(a *App, v int) {
				a.cfg.Volume = volumes[v]
				a.applySound()
			}},

		{tab: tabControls, label: "Key mode", help: "async (Windows): move and turn at once; events: plain terminal keys",
			values: keyChoices, value: slices.Index(keyChoices, a.cfg.KeyMode),
			set: func(a *App, v int) {
				a.cfg.KeyMode = keyChoices[v]
				a.keys.Reset() // keeps what it learned about the terminal (releases)
				if a.cfg.KeyMode == "async" {
					a.scr.EnableFocus() // async keys only count while focused
				}
			}},
		{tab: tabControls, label: "Pixel aspect", help: "change it if walls look too tall or too flat",
			values: mapStrings(aspects, func(v float64) string { return fmt.Sprint(v) }), value: slices.Index(aspects, a.cfg.PixelAspect),
			set: func(a *App, v int) {
				a.cfg.PixelAspect = aspects[v]
				a.frame.PixelAspect = a.cfg.PixelAspect
			}},
		onOff(tabControls, "Minimap at start", "each run starts with the minimap shown; M shows or hides it", a.cfg.MinimapAtStart, func(a *App, on bool) {
			a.cfg.MinimapAtStart = on
			a.state.ShowMinimap = on // the current run too
		}),

		{tab: tabGameplay, label: "Map unlocks", help: "what opens the next map: survive 5:00" + sep + "survive 2:00" + sep + "all open",
			values: unlockRuleLabels, value: max(slices.Index(unlockRules, a.cfg.UnlockRule), 0), // "" (unset) shows 5:00
			set: func(a *App, v int) { a.cfg.UnlockRule = unlockRules[v] }},
		onOff(tabGameplay, "Minimap fog of war", "the minimap shows only what you've seen", a.cfg.MinimapFog,
			func(a *App, on bool) { a.cfg.MinimapFog = on }),
		{tab: tabGameplay, label: "Color threshold", help: "fewer color codes for slow terminals; off sends every color",
			values: mapStrings(thresholds, thresholdLabel), value: slices.Index(thresholds, a.cfg.ColorThreshold),
			set: func(a *App, v int) {
				a.cfg.ColorThreshold = thresholds[v]
				a.frame.ColorThreshold = uint8(a.cfg.ColorThreshold)
			}},
	}
}

// Clickable ids on the Settings screen: 0.. are rows of the open tab; then
// these.
const (
	setHitBack = -1
	setHitTab  = -100 // setHitTab - t is tab t
)

// settingsScreen is the Settings screen.
type settingsScreen struct {
	back  screen   // where Esc goes: the title or the pause card
	drop  backdrop // what shows behind it
	tab   int
	sel   int // the row of the open tab; -1 = the tab row itself
	items []setting
	hits  hitList
}

// settingsMenu opens the Settings. back is the screen to return to (title
// or pause) and drop what shows behind.
func (a *App) settingsMenu(back screen, drop backdrop) *settingsScreen {
	return &settingsScreen{back: back, drop: drop, items: a.settingsList()}
}

// rows returns the indexes (into items) of the open tab's settings.
func (ss *settingsScreen) rows() []int {
	var out []int
	for i, it := range ss.items {
		if it.tab == ss.tab {
			out = append(out, i)
		}
	}
	return out
}

// find selects the setting called label, opening its tab (for tests and
// for jumping to a setting).
func (ss *settingsScreen) find(label string) bool {
	for _, it := range ss.items {
		if it.label == label {
			ss.tab = it.tab
			for row, i := range ss.rows() {
				if ss.items[i].label == label {
					ss.sel = row
				}
			}
			return true
		}
	}
	return false
}

// setTab opens tab t (wrapping round), keeping the cursor where it is on
// the tab row and on the first setting otherwise.
func (ss *settingsScreen) setTab(a *App, t int) {
	ss.tab = (t%numTabs + numTabs) % numTabs
	if ss.sel > 0 {
		ss.sel = 0
	}
	a.menuSound(audio.MenuTick)
}

// change moves the selected setting's value by d (wrapping round),
// applies it and saves.
func (ss *settingsScreen) change(a *App, d int) {
	rows := ss.rows()
	if ss.sel < 0 || ss.sel >= len(rows) {
		return
	}
	it := &ss.items[rows[ss.sel]]
	n := len(it.values)
	if n < 2 {
		return
	}
	v := ((it.value+d)%n + n) % n // the extra +n keeps it >= 0
	it.set(a, v)
	a.menuSound(audio.MenuClick)
	a.persist()                 // saved on every change (D25)
	ss.items = a.settingsList() // rebuilt, so every line shows what's now true
}

func (ss *settingsScreen) key(a *App, ev term.Key) {
	if ev.Action == term.Release {
		return // menus act on presses (and repeats, so a held arrow scrolls)
	}
	k, ok := input.MenuKeyFor(ev)
	if !ok {
		return
	}
	n := len(ss.rows())
	switch k {
	case ui.KeyBack:
		a.menuSound(audio.MenuBack)
		a.switchTo(ss.back)
	case ui.KeyTab:
		ss.setTab(a, ss.tab+1)
	case ui.KeyUp:
		ss.sel = (ss.sel+1+n)%(n+1) - 1 // -1 (the tab row) to n-1, wrapping
		a.menuSound(audio.MenuTick)
	case ui.KeyDown:
		ss.sel = (ss.sel+2)%(n+1) - 1
		a.menuSound(audio.MenuTick)
	case ui.KeyLeft, ui.KeyRight:
		d := 1
		if k == ui.KeyLeft {
			d = -1
		}
		if ss.sel < 0 {
			ss.setTab(a, ss.tab+d)
		} else {
			ss.change(a, d)
		}
	case ui.KeyEnter:
		ss.change(a, 1)
	}
}

// mouse: a click opens a tab, or picks a setting and moves it to its
// next value; a click on the footer goes back.
func (ss *settingsScreen) mouse(a *App, b mouseButton, down bool, x, y int) {
	if b != mouseLeft || !down {
		return
	}
	id, ok := ss.hits.at(x, y)
	switch {
	case !ok:
	case id == setHitBack:
		a.menuSound(audio.MenuBack)
		a.switchTo(ss.back)
	case id <= setHitTab:
		ss.sel = -1
		ss.setTab(a, setHitTab-id)
	default:
		ss.sel = id
		ss.change(a, 1)
	}
}

// settingValue draws a setting's value at (x, y) and returns the column
// after it: a switch, a bar with its number, or "◀ value ▶".
func settingValue(s term.Screen, x, y int, it setting, st term.Style) int {
	sy := ui.Sym()
	switch {
	case it.toggle:
		return ui.Put(s, x, y, ui.Switch(it.value == 1), st)
	case it.levels != nil:
		off := st
		if st != ui.StSelected {
			st, off = ui.On(ui.ColAmber, st.Bg), ui.On(ui.ColBorder, st.Bg)
		}
		x = ui.LevelBar(s, x, y, 10, it.levels[it.value], st, off)
		return ui.Put(s, x, y, " "+it.values[it.value], off)
	}
	return ui.Put(s, x, y, sy.Prev+" "+it.values[it.value]+" "+sy.Next, st)
}

func (ss *settingsScreen) tick(a *App, now time.Time, dt float64) {
	a.drawBackdrop(ss.drop, dt)
	s, sy := a.scr, ui.Sym()
	ox, oy := stage(s)
	ss.hits = ss.hits[:0]
	ui.Panel(s, ox, oy, stageW, stageH, ui.BoxRound, ui.StBorder, ui.StPanel)
	ui.SmallLogo(s, ox+3, oy+1, ui.ColPanel)
	ui.PutRight(s, ox+stageW-3, oy+1, "SETTINGS", ui.StTitle)
	ui.PutRight(s, ox+stageW-3, oy+2, "changes apply at once and are saved", ui.StDim)

	// The tab row; with the cursor on it, the open tab is marked.
	x := ox + 3
	if ss.sel < 0 {
		ui.Put(s, x, oy+4, sy.Select, ui.StTitle)
	}
	for t, span := range ui.Tabs(s, x+2, oy+4, tabNames, ss.tab, ui.ColPanel) {
		ss.hits.add(span[0], oy+4, span[1]-span[0], 1, setHitTab-t)
	}

	// The open tab's settings: two rows each, the value and a help line.
	const valueX = 26
	for row, i := range ss.rows() {
		it := ss.items[i]
		y := oy + 6 + 2*row
		st, marker := ui.StPanel, "  "
		if row == ss.sel {
			st, marker = ui.StSelected, sy.Select+" "
			ui.Fill(s, ox+2, y, stageW-4, 1, st)
		}
		ui.Put(s, ox+3, y, marker+it.label, st)
		settingValue(s, ox+valueX, y, it, st)
		ui.Put(s, ox+5, y+1, it.help, ui.StDim)
		ss.hits.add(ox+2, y, stageW-4, 2, row)
	}
	if ss.tab == tabAudio { // the sound warning belongs with the sound settings
		if p := a.soundProblem(); p != "" {
			for i, l := range wrap(p, stageW-10) {
				ui.Put(s, ox+5, oy+11+i, l, ui.On(ui.ColYellow, ui.ColPanel))
			}
		}
	}

	hints := []ui.Hint{{Keys: "tab", Text: "next tab"}, {Keys: sy.UpDown, Text: "select"}, {Keys: sy.LeftRight, Text: "change"}, {Keys: "esc", Text: "back"}}
	ui.Hints(s, ox+1, stageW-2, oy+stageH-2, hints, ui.ColPanel)
	ss.hits.add(ox+1, oy+stageH-2, stageW-2, 1, setHitBack)
	a.show()
}
