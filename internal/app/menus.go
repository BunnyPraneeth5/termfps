package app

import (
	"fmt"
	"slices"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// menuScreen shows one ui.Menu and hands what the player did to act.
// Every menu (title, map select, settings, pause, game over) is one of
// these with its own items and act function.
type menuScreen struct {
	menu *ui.Menu
	// act runs when a key or click did something (r is never ui.None);
	// menu.Selected is the item it applies to.
	act func(a *App, r ui.Result)
	// overGame draws the frozen game, dimmed, behind the menu; otherwise
	// the background is blank.
	overGame bool
	// text is the menu's own lines under the title; tick adds the save and
	// sound problems (if any) after them every frame, so new ones show at once.
	text []string
}

func (m *menuScreen) key(a *App, ev term.Key) {
	if ev.Action == term.Release {
		return // menus act on presses (and repeats, so a held arrow scrolls)
	}
	k, ok := input.MenuKeyFor(ev)
	if !ok {
		return
	}
	if r := m.menu.Key(k); r != ui.None {
		m.act(a, r)
	}
}

// mouse: pressing the left button on an item picks it.
func (m *menuScreen) mouse(a *App, b mouseButton, down bool, x, y int) {
	if b != mouseLeft || !down {
		return
	}
	i, ok := m.menu.ItemAt(x, y)
	if !ok {
		return
	}
	if r := m.menu.Click(i); r != ui.None {
		m.act(a, r)
	}
}

func (m *menuScreen) tick(a *App, now time.Time, dt float64) {
	if m.overGame {
		// No game.Update here: the game is frozen. Redrawing (not reusing
		// the last frame) keeps it visible after a resize.
		a.drawScene()
		render.Dim(&a.frame)
		a.present()
	} else {
		a.scr.Clear()
	}
	// m.text[:n:n] limits the capacity to n, so append makes a new slice
	// instead of writing past the end of m.text.
	n := len(m.text)
	m.menu.Text = append(m.text[:n:n], a.messages()...)
	ui.Draw(a.scr, m.menu)
	a.scr.Show()
}

// titleMenu is the start menu.
func (a *App) titleMenu() *menuScreen {
	const (
		play = iota // iota numbers the items 0, 1, 2, ...
		selectMap
		settings
		quit
	)
	s := &menuScreen{text: []string{"Survive as long as you can."}, menu: ui.New("TERMFPS", nil, []ui.Item{
		{Label: "Play"},
		{Label: "Select map"},
		{Label: "Settings"},
		{Label: "Quit"},
	})}
	s.act = func(a *App, r ui.Result) {
		if r != ui.Chosen {
			return // Esc on the title menu does nothing; Quit or Ctrl+C quits
		}
		switch s.menu.Selected {
		case play:
			a.newRun()
		case selectMap:
			a.switchTo(a.mapMenu())
		case settings:
			a.switchTo(a.settingsMenu(s))
		case quit:
			a.quit = true
		}
	}
	return s
}

// mapMenu lists the maps; picking an open one starts a run on it. Locked
// maps are shown dimmed with what opens them.
func (a *App) mapMenu() *menuScreen {
	items := make([]ui.Item, 0, len(a.maps)+1)
	for i, m := range a.maps {
		it := ui.Item{Label: fmt.Sprintf("%-14s %dx%d", m.Name, m.W, m.H)}
		if b, ok := a.progress.Best[m.Name]; ok { // ok is false if the map has no best yet
			it.Label += fmt.Sprintf("   best %s, %d kills", render.Clock(b.Time), b.Kills)
		}
		if i >= a.builtinMaps() {
			it.Label += "   (your map)" // M9
		}
		if a.mapLocked(i) {
			it.Disabled = true
			it.Label += fmt.Sprintf("   locked: survive %s on %s", render.Clock(unlockAfter(a.cfg.UnlockRule)), a.maps[i-1].Name)
		}
		items = append(items, it)
	}
	items = append(items, ui.Item{Label: "Back"})
	s := &menuScreen{menu: ui.New("SELECT MAP", nil, items)}
	s.menu.Selected = a.mapIndex
	s.act = func(a *App, r ui.Result) {
		i := s.menu.Selected
		switch {
		case r == ui.Back || i == len(a.maps):
			a.switchTo(a.titleMenu())
		case r == ui.Chosen:
			a.mapIndex = i
			a.newRun()
		}
	}
	return s
}

// pauseMenu freezes the game behind a dimmed screen.
func (a *App) pauseMenu() *menuScreen {
	const (
		resume = iota
		settings
		restart
		quitToMenu
	)
	s := &menuScreen{overGame: true, menu: ui.New("PAUSED", nil, []ui.Item{
		{Label: "Resume"},
		{Label: "Settings"},
		{Label: "Restart"},
		{Label: "Quit to menu"},
	})}
	s.act = func(a *App, r ui.Result) {
		if r == ui.Back {
			a.resume()
			return
		}
		switch s.menu.Selected {
		case resume:
			a.resume()
		case settings:
			a.switchTo(a.settingsMenu(s))
		case restart:
			a.newRun()
		case quitToMenu:
			a.endRun()
			a.switchTo(a.titleMenu())
		}
	}
	return s
}

// resume goes back to the paused game. Keys held before the pause don't
// count any more.
func (a *App) resume() {
	a.keys = input.Tracker{}
	a.switchTo(&playingScreen{})
}

// gameOverMenu shows the run's result over the frozen last frame. R isn't
// a menu key: it's reload, and a held R would pick an item by accident.
func (a *App) gameOverMenu() *menuScreen {
	const (
		retry = iota
		mainMenu
	)
	st := a.state
	name := a.maps[a.mapIndex].Name
	best := a.progress.Best[name] // endRun has already counted this run
	s := &menuScreen{overGame: true, text: []string{
		fmt.Sprintf("Kills: %d    Time: %s    Tier: %d", st.Kills, render.Clock(st.Time), st.Tier),
		fmt.Sprintf("Best on %s: %d kills, %s, tier %d", name, best.Kills, render.Clock(best.Time), best.Tier),
	}, menu: ui.New("YOU DIED", nil, []ui.Item{
		{Label: "Try again"},
		{Label: "Main menu"},
	})}
	s.act = func(a *App, r ui.Result) {
		switch {
		case r == ui.Back || s.menu.Selected == mainMenu:
			a.switchTo(a.titleMenu())
		case s.menu.Selected == retry:
			a.newRun()
		}
	}
	return s
}

// Settings items, in menu order.
const (
	setMode = iota
	setMixedBackground
	setThreshold
	setScale
	setFPS
	setMinimap
	setTextures
	setSound
	setVolume
	setKeys
	setAspect
	setUnlocks
	setMinimapFog
	setBack
)

// settingsMenu changes the settings; each change applies right away.
// back is the menu to return to (title or pause), and also decides the
// background.
func (a *App) settingsMenu(back *menuScreen) *menuScreen {
	var modes []game.RenderMode
	for m := game.ModeHalfBlock; ; m = m.Next() { // every mode, in V order
		modes = append(modes, m)
		if m.Next() == game.ModeHalfBlock {
			break
		}
	}
	fpsChoices := []int{30, 60}
	if !slices.Contains(fpsChoices, a.cfg.FPS) {
		fpsChoices = append(fpsChoices, a.cfg.FPS) // a value from -fps stays choosable
		slices.Sort(fpsChoices)
	}
	keyChoices := []string{"events"}
	if input.AsyncSupported() == nil {
		keyChoices = []string{"async", "events"}
	}
	backgrounds := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	if !slices.Contains(backgrounds, a.cfg.MixedBackground) {
		backgrounds = append(backgrounds, a.cfg.MixedBackground) // e.g. 25 typed into save.json
		slices.Sort(backgrounds)
	}
	thresholds := []int{0, 4, 8, 16, 32}
	if !slices.Contains(thresholds, a.cfg.ColorThreshold) {
		thresholds = append(thresholds, a.cfg.ColorThreshold)
		slices.Sort(thresholds)
	}
	var aspects []float64
	for tenths := 5; tenths <= 20; tenths++ {
		aspects = append(aspects, float64(tenths)/10)
	}
	if !slices.Contains(aspects, a.cfg.PixelAspect) {
		aspects = append(aspects, a.cfg.PixelAspect) // e.g. -aspect 1.25
		slices.Sort(aspects)
	}

	items := make([]ui.Item, setBack+1)
	items[setMode] = ui.Item{Label: "Render mode", Values: mapStrings(modes, game.RenderMode.String),
		Value: slices.Index(modes, a.state.Mode)}
	items[setMixedBackground] = ui.Item{Label: "Mixed background",
		Values: mapStrings(backgrounds, func(n int) string { return fmt.Sprintf("%d%%", n) }),
		Value:  slices.Index(backgrounds, a.cfg.MixedBackground)}
	items[setThreshold] = ui.Item{Label: "Color threshold", Values: mapStrings(thresholds, thresholdLabel),
		Value: slices.Index(thresholds, a.cfg.ColorThreshold)}
	items[setScale] = ui.Item{Label: "Render scale", Values: scaleChoices,
		Value: max(slices.Index(scaleChoices, a.cfg.RenderScale), 0)} // "" (unset) shows auto
	items[setFPS] = ui.Item{Label: "FPS", Values: mapStrings(fpsChoices, func(n int) string { return fmt.Sprint(n) }),
		Value: slices.Index(fpsChoices, a.cfg.FPS)}
	items[setMinimap] = onOff("Minimap at start", a.cfg.MinimapAtStart)
	items[setTextures] = onOff("Wall textures", a.cfg.Textures)
	items[setSound] = onOff("Sound", a.cfg.Sound)
	items[setVolume] = ui.Item{Label: "Volume", Values: []string{"0%", "10%", "20%", "30%", "40%", "50%", "60%", "70%", "80%", "90%", "100%"},
		Value: min(max(a.cfg.Volume/10, 0), 10)}
	items[setKeys] = ui.Item{Label: "Key mode", Values: keyChoices, Value: slices.Index(keyChoices, a.cfg.KeyMode)}
	items[setAspect] = ui.Item{Label: "Pixel aspect", Values: mapStrings(aspects, func(v float64) string { return fmt.Sprint(v) }),
		Value: slices.Index(aspects, a.cfg.PixelAspect)}
	items[setUnlocks] = ui.Item{Label: "Map unlocks", Values: unlockRuleLabels,
		Value: max(slices.Index(unlockRules, a.cfg.UnlockRule), 0)} // "" (unset) shows 5:00
	items[setMinimapFog] = onOff("Minimap fog of war", a.cfg.MinimapFog)
	items[setBack] = ui.Item{Label: "Back"}

	s := &menuScreen{overGame: back.overGame, menu: ui.New("SETTINGS", nil, items)}
	s.act = func(a *App, r ui.Result) {
		it := s.menu.Items[s.menu.Selected]
		switch {
		case r == ui.Back || s.menu.Selected == setBack:
			a.switchTo(back)
			return
		case r != ui.Changed:
			return
		case s.menu.Selected == setMode:
			a.state.Mode = modes[it.Value]
		case s.menu.Selected == setMixedBackground:
			a.cfg.MixedBackground = backgrounds[it.Value]
			a.frame.MixedBackground = float64(a.cfg.MixedBackground) / 100
		case s.menu.Selected == setThreshold:
			a.cfg.ColorThreshold = thresholds[it.Value]
			a.frame.ColorThreshold = uint8(a.cfg.ColorThreshold)
		case s.menu.Selected == setScale:
			a.cfg.RenderScale = scaleChoices[it.Value]
			a.resizeFrame()
		case s.menu.Selected == setFPS:
			a.cfg.FPS = fpsChoices[it.Value] // Run resets its ticker
		case s.menu.Selected == setMinimap:
			a.cfg.MinimapAtStart = it.Value == 1
			a.state.ShowMinimap = a.cfg.MinimapAtStart // the current run too
		case s.menu.Selected == setTextures:
			a.cfg.Textures = it.Value == 1
			a.frame.Textures = a.cfg.Textures
		case s.menu.Selected == setSound:
			a.cfg.Sound = it.Value == 1
			a.applySound()
		case s.menu.Selected == setVolume:
			a.cfg.Volume = it.Value * 10
			a.applySound()
		case s.menu.Selected == setKeys:
			a.cfg.KeyMode = keyChoices[it.Value]
			a.keys.Reset() // keeps what it learned about the terminal (releases)
			if a.cfg.KeyMode == "async" {
				a.scr.EnableFocus() // async keys only count while focused
			}
		case s.menu.Selected == setMinimapFog:
			a.cfg.MinimapFog = it.Value == 1
		case s.menu.Selected == setUnlocks:
			a.cfg.UnlockRule = unlockRules[it.Value]
		case s.menu.Selected == setAspect:
			a.cfg.PixelAspect = aspects[it.Value]
			a.frame.PixelAspect = a.cfg.PixelAspect
		}
		a.persist() // saved on every change (D25)
	}
	return s
}

// scaleChoices are the render scale settings (V6), in menu order.
var scaleChoices = []string{"auto", "full", "half"}

// thresholdLabel is the menu text for a color threshold.
func thresholdLabel(n int) string {
	if n == 0 {
		return "off"
	}
	return fmt.Sprint(n)
}

// onOff is an off/on setting item.
func onOff(label string, on bool) ui.Item {
	v := 0
	if on {
		v = 1
	}
	return ui.Item{Label: label, Values: []string{"off", "on"}, Value: v}
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
