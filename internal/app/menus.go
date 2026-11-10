package app

import (
	"fmt"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/save"
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
	// drop is what's drawn behind the menu: the frozen game, the title's
	// map tour, or the plain background.
	drop backdrop
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
	before := m.menu.Selected
	r := m.menu.Key(k)
	a.resultSound(r, m.menu.Selected != before)
	if r != ui.None {
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
	r := m.menu.Click(i)
	if r == ui.None { // a locked item
		a.menuSound(audio.MenuBuzz)
		return
	}
	a.resultSound(r, false)
	m.act(a, r)
}

func (m *menuScreen) tick(a *App, now time.Time, dt float64) {
	a.drawBackdrop(m.drop, dt)
	// m.text[:n:n] limits the capacity to n, so append makes a new slice
	// instead of writing past the end of m.text.
	n := len(m.text)
	m.menu.Text = append(m.text[:n:n], a.messages()...)
	ui.Draw(a.scr, m.menu)
	a.show()
}

// gameOverMenu shows the run's result over the frozen last frame. R isn't
// a menu key: it's reload, and a held R would pick an item by accident.
func (a *App) gameOverMenu() *menuScreen {
	const (
		retry = iota
		mainMenu
	)
	s := &menuScreen{drop: dropGame, text: a.resultText(), menu: ui.New("YOU DIED", nil, []ui.Item{
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

// resultText is a finished run's result and the best on its map, for the
// game over and win menus. endRun has already counted the run.
func (a *App) resultText() []string {
	st := a.state
	name := a.maps[a.mapIndex].Name
	if st.Play == game.Waves { // M15
		return []string{
			fmt.Sprintf("Wave %d of %d    Kills: %d    Time: %s", st.Wave, game.NumWaves, st.Kills, render.Clock(st.Time)),
			fmt.Sprintf("Waves best on %s: %s", name, wavesBestText(a.progress.Waves[name])),
		}
	}
	best := a.progress.Best[name]
	return []string{
		fmt.Sprintf("Kills: %d    Time: %s    Tier: %d", st.Kills, render.Clock(st.Time), st.Tier),
		fmt.Sprintf("Best on %s: %d kills, %s, tier %d", name, best.Kills, render.Clock(best.Time), best.Tier),
	}
}

// wavesBestText describes a waves best: the fastest win, or the furthest
// wave reached (M15).
func wavesBestText(w save.WavesBest) string {
	if w.Wins > 0 {
		return fmt.Sprintf("won in %s", render.Clock(w.WinTime))
	}
	return fmt.Sprintf("wave %d of %d", w.Wave, game.NumWaves)
}

// winMenu shows a won waves run (M15) over the frozen last frame.
func (a *App) winMenu() *menuScreen {
	const (
		again = iota
		mainMenu
	)
	s := &menuScreen{drop: dropGame, text: a.resultText(), menu: ui.New("YOU WIN: ALL WAVES CLEARED", nil, []ui.Item{
		{Label: "Play again"},
		{Label: "Main menu"},
	})}
	s.act = func(a *App, r ui.Result) {
		switch {
		case r == ui.Back || s.menu.Selected == mainMenu:
			a.switchTo(a.titleMenu())
		case s.menu.Selected == again:
			a.newRun()
		}
	}
	return s
}

// clearScreen fills the whole screen with the background color (U1).
func (a *App) clearScreen() {
	w, h := a.scr.Size()
	ui.Fill(a.scr, 0, 0, w, h, ui.StBg)
}
