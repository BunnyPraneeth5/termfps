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

// The pause card (U3, D54): over the dimmed, frozen game, with the run so
// far and what comes next.

// Pause items, in order.
const (
	pauseResume = iota
	pauseRestart
	pauseGuide
	pauseSettings
	pauseQuit
	pauseItems
)

var pauseLabels = [pauseItems]string{"RESUME", "RESTART", "GUIDE", "SETTINGS", "QUIT TO MENU"}

// Card size.
const (
	pauseW = 50
	pauseH = 15
)

// pauseScreen is the pause card.
type pauseScreen struct {
	sel  int
	hits hitList
}

// pauseMenu freezes the game behind the pause card.
func (a *App) pauseMenu() *pauseScreen {
	return &pauseScreen{}
}

// resume goes back to the paused game. Keys held before the pause don't
// count any more.
func (a *App) resume() {
	a.keys = input.Tracker{}
	a.switchTo(&playingScreen{})
}

func (p *pauseScreen) key(a *App, ev term.Key) {
	if ev.Action == term.Release {
		return
	}
	k, ok := input.MenuKeyFor(ev)
	if !ok {
		return
	}
	switch k {
	case ui.KeyUp:
		p.sel = (p.sel + pauseItems - 1) % pauseItems
		a.menuSound(audio.MenuTick)
	case ui.KeyDown:
		p.sel = (p.sel + 1) % pauseItems
		a.menuSound(audio.MenuTick)
	case ui.KeyEnter:
		p.choose(a)
	case ui.KeyBack:
		a.menuSound(audio.MenuBack)
		a.resume()
	}
}

func (p *pauseScreen) mouse(a *App, b mouseButton, down bool, x, y int) {
	if b != mouseLeft || !down {
		return
	}
	if id, ok := p.hits.at(x, y); ok {
		p.sel = id
		p.choose(a)
	}
}

// choose acts on the selected item.
func (p *pauseScreen) choose(a *App) {
	a.menuSound(audio.MenuClick)
	switch p.sel {
	case pauseResume:
		a.resume()
	case pauseRestart:
		a.newRun()
	case pauseGuide:
		a.switchTo(a.guideMenu(p, dropGame)) // M18a
	case pauseSettings:
		a.switchTo(a.settingsMenu(p, dropGame))
	case pauseQuit:
		a.endRun()
		a.switchTo(a.titleMenu())
	}
}

// nextText says what comes next in the run: the next tier in survival,
// the wave's progress in waves mode.
func nextText(st *game.State) string {
	if st.Play == game.Waves {
		if st.InBreak() {
			return fmt.Sprintf("wave %d of %d starts in %s", st.Wave+1, game.NumWaves, render.Clock(math.Ceil(st.BreakLeft)))
		}
		left := "1 enemy"
		if n := st.WaveLeft(); n != 1 {
			left = fmt.Sprintf("%d enemies", n)
		}
		return fmt.Sprintf("wave %d of %d%s%s left", st.Wave, game.NumWaves, ui.Sym().Sep, left)
	}
	kills, seconds := st.NextTier()
	return fmt.Sprintf("next tier in %s or %s", render.Clock(math.Ceil(seconds)), plural(kills, "kill"))
}

func (p *pauseScreen) tick(a *App, now time.Time, dt float64) {
	a.drawBackdrop(dropGame, dt)
	s, sy, st := a.scr, ui.Sym(), a.state
	w, h := s.Size()
	x, y := (w-pauseW)/2, (h-pauseH)/2
	card := ui.On(ui.ColText, ui.ColCard)
	dim := ui.On(ui.ColDim, ui.ColCard)
	ui.Panel(s, x, y, pauseW, pauseH, ui.BoxRound, ui.On(ui.ColBorder, ui.ColCard), card)
	ui.Put(s, x+3, y+1, "II PAUSED", ui.Bold(ui.On(ui.ColAmber, ui.ColCard)))
	ui.PutRight(s, x+pauseW-3, y+1, a.maps[a.mapIndex].Name+sy.Sep+st.Play.String(), dim)

	// The run so far.
	stage := fmt.Sprintf("%d", st.Tier)
	stageName := "TIER"
	if st.Play == game.Waves {
		stageName, stage = "WAVE", fmt.Sprintf("%d/%d", wave(st), game.NumWaves)
	}
	cx := x + 3
	for _, part := range [][2]string{{"TIME", render.Clock(st.Time)}, {"KILLS", fmt.Sprint(st.Kills)}, {stageName, stage}, {"HP", fmt.Sprint(st.Player.HP)}} {
		cx = ui.Put(s, cx, y+3, part[0]+" ", dim)
		cx = ui.Put(s, cx, y+3, part[1], ui.Bold(ui.On(ui.ColBright, ui.ColCard))) + 3
	}
	ui.Put(s, x+3, y+4, nextText(st), card)
	ui.Put(s, x+1, y+5, strings.Repeat(sy.Boxes[ui.BoxSingle][4], pauseW-2), ui.On(ui.ColLine, ui.ColCard))

	p.hits = p.hits[:0]
	for i, label := range pauseLabels {
		iy := y + 6 + i
		ist, marker := card, "  "
		if i == p.sel {
			ist, marker = ui.StSelected, sy.Select+" "
			ui.Fill(s, x+1, iy, pauseW-2, 1, ist)
		}
		ui.Put(s, x+3, iy, marker+label, ist)
		if i == pauseQuit {
			note := dim
			if i == p.sel {
				note = ist
			}
			ui.PutRight(s, x+pauseW-3, iy, "this run still counts", note)
		}
		p.hits.add(x+1, iy, pauseW-2, 1, i)
	}
	// A save or sound problem, on the row under the items.
	if notes := a.messages(); len(notes) > 0 {
		ui.Center(s, x+1, pauseW-2, y+pauseH-3, clip(notes[0], pauseW-4), ui.On(ui.ColYellow, ui.ColCard))
	}
	ui.Hints(s, x+1, pauseW-2, y+pauseH-2, []ui.Hint{{Keys: sy.UpDown, Text: "pick"}, {Keys: sy.Enter, Text: "choose"}, {Keys: "esc", Text: "resume"}}, ui.ColCard)
	a.show()
}

// clip cuts text to at most n characters.
func clip(text string, n int) string {
	r := []rune(text)
	if len(r) > n {
		return string(r[:max(n, 0)])
	}
	return text
}
