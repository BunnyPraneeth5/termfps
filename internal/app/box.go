package app

import (
	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// The error box (U3, D54): a red-bordered box over whatever screen is
// showing, for problems the player must know about (a save that failed, a
// save file or map that couldn't be loaded). It shows the real error text
// and what it means, and stays until OK. No silent fallbacks.

// errorBox is one problem to show.
type errorBox struct {
	title   string // in capitals, e.g. "COULDN'T SAVE YOUR PROGRESS"
	detail  string // the real error text
	meaning string // what it means for the player; "" = the detail says it
}

// boxW is the error box's width.
const boxW = 64

// raise queues an error box. If the game is running it pauses first, so
// nothing happens to the player while they read.
func (a *App) raise(b errorBox) {
	a.boxes = append(a.boxes, b)
	if _, playing := a.current.(*playingScreen); playing {
		a.switchTo(a.pauseMenu())
	}
}

// boxKey handles a key while an error box is up: Enter or Esc closes it.
func (a *App) boxKey(ev term.Key) {
	if ev.Action == term.Release {
		return
	}
	if k, ok := input.MenuKeyFor(ev); ok && (k == ui.KeyEnter || k == ui.KeyBack) {
		a.closeBox()
	}
}

// closeBox closes the error box on top.
func (a *App) closeBox() {
	a.boxes = a.boxes[1:]
	a.menuSound(audio.MenuClick)
}

// drawBox draws the first queued error box, centered.
func (a *App) drawBox() {
	if len(a.boxes) == 0 {
		return
	}
	b, s := a.boxes[0], a.scr
	lines := wrap(b.detail, boxW-6)
	var meaning []string
	if b.meaning != "" {
		meaning = wrap(b.meaning, boxW-6)
	}
	rows := 2 + len(lines) + 2 // title, blank, detail, blank, OK
	if len(meaning) > 0 {
		rows += len(meaning) + 1
	}
	w, h := s.Size()
	x, y := (w-boxW)/2, max((h-rows-2)/2, 0)
	card := ui.On(ui.ColText, ui.ColCard)
	ui.Panel(s, x, y, boxW, rows+2, ui.BoxDouble, ui.On(ui.ColRed, ui.ColCard), card)
	ui.Put(s, x+3, y+1, "! "+b.title, ui.Bold(ui.On(ui.ColRed, ui.ColCard)))
	row := y + 3
	for _, l := range lines {
		ui.Put(s, x+3, row, l, ui.Bold(ui.On(ui.ColBright, ui.ColCard)))
		row++
	}
	if len(meaning) > 0 {
		row++
		for _, l := range meaning {
			ui.Put(s, x+3, row, l, ui.On(ui.ColDim, ui.ColCard))
			row++
		}
	}
	ui.Center(s, x, boxW, row+1, "  OK  ", ui.StSelected)
}

// show draws the error box (if any) over the screen and sends the frame
// to the terminal. Every screen ends its tick with it.
func (a *App) show() {
	a.drawBox()
	a.scr.Show()
}
