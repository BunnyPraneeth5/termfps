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
// The guide's section lists are these (the other screens have their own
// layouts since the UI redesign, D54).
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

// wavesBestText describes a waves best: the fastest win, or the furthest
// wave reached (M15).
func wavesBestText(w save.WavesBest) string {
	if w.Wins > 0 {
		return fmt.Sprintf("won in %s", render.Clock(w.WinTime))
	}
	return fmt.Sprintf("wave %d of %d", w.Wave, game.NumWaves)
}

// clearScreen fills the whole screen with the background color (U1).
func (a *App) clearScreen() {
	w, h := a.scr.Size()
	ui.Fill(a.scr, 0, 0, w, h, ui.StBg)
}
