package app

import (
	"fmt"
	"slices"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
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
}

func (m *menuScreen) key(a *App, ev *tcell.EventKey) {
	k, ok := input.MenuKeyFor(ev)
	if !ok {
		return
	}
	if r := m.menu.Key(k); r != ui.None {
		m.act(a, r)
	}
}

func (m *menuScreen) click(a *App, x, y int) {
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
	s := &menuScreen{menu: ui.New("TERMFPS", []string{"Survive as long as you can."}, []ui.Item{
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

// mapMenu lists the maps; picking one starts a run on it.
func (a *App) mapMenu() *menuScreen {
	items := make([]ui.Item, 0, len(a.maps)+1)
	for i, m := range a.maps {
		items = append(items, ui.Item{Label: fmt.Sprintf("Map %d (%dx%d)", i+1, m.W, m.H)})
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
	s := &menuScreen{overGame: true, menu: ui.New("YOU DIED", []string{
		fmt.Sprintf("Kills: %d    Time: %s    Tier: %d", st.Kills, render.Clock(st.Time), st.Tier),
	}, []ui.Item{
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
	setFPS
	setMinimap
	setTextures
	setSound
	setVolume
	setKeys
	setAspect
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
	items[setBack] = ui.Item{Label: "Back"}

	s := &menuScreen{overGame: back.overGame, menu: ui.New("SETTINGS",
		[]string{"Sound isn't in the game yet; its settings are kept for when it is."}, items)}
	s.act = func(a *App, r ui.Result) {
		it := s.menu.Items[s.menu.Selected]
		switch {
		case r == ui.Back || s.menu.Selected == setBack:
			a.switchTo(back)
		case r != ui.Changed:
			return
		case s.menu.Selected == setMode:
			a.state.Mode = modes[it.Value]
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
		case s.menu.Selected == setVolume:
			a.cfg.Volume = it.Value * 10
		case s.menu.Selected == setKeys:
			a.cfg.KeyMode = keyChoices[it.Value]
			a.keys = input.Tracker{}
			if a.cfg.KeyMode == "async" {
				a.scr.EnableFocus() // async keys only count while focused
			}
		case s.menu.Selected == setAspect:
			a.cfg.PixelAspect = aspects[it.Value]
			a.frame.PixelAspect = a.cfg.PixelAspect
		}
	}
	return s
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
