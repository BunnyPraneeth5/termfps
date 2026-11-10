package app

import (
	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// Pieces the redesigned screens share (U2, D54).

// Version is the game's version and codename (D54). The codename is always
// written masked like this: the masking is the joke.
const Version = "v0.75 “As*****”"

// The fixed-layout screens (title, Play, Settings, the end screens) are
// laid out for the smallest window and centered in bigger ones.
const (
	stageW = render.MinCols
	stageH = render.MinRows
)

// stage is the top-left cell of the stageW x stageH block on screen s.
func stage(s term.Screen) (ox, oy int) {
	w, h := s.Size()
	return max((w-stageW)/2, 0), max((h-stageH)/2, 0)
}

// backdrop is what a screen draws behind itself.
type backdrop int

const (
	dropNone  backdrop = iota // the plain background
	dropGame                  // the frozen game, dimmed (pause, the end screens)
	dropTitle                 // the slow map tour of the title screen, if it's on
)

// drawBackdrop fills the screen with backdrop b; dt moves the title tour.
func (a *App) drawBackdrop(b backdrop, dt float64) {
	switch {
	case b == dropGame:
		// No game.Update here: the game is frozen. Redrawing (not reusing
		// the last frame) keeps it visible after a resize.
		a.drawScene()
		render.Dim(&a.frame)
		a.present() // the HUD rows stay as the game last drew them
	case b == dropTitle && a.cfg.TitleBackground:
		a.drawTour(dt)
	default:
		a.clearScreen()
	}
}

// hit is a block of screen cells that does something when clicked; id
// says what, to the screen that added it.
type hit struct {
	r  rect
	id int
}

// hitList is a screen's clickable blocks, rebuilt every time it draws.
type hitList []hit

// add records the w x h block at (x, y) as clickable.
func (h *hitList) add(x, y, w, ht, id int) {
	*h = append(*h, hit{r: rect{x: x, y: y, w: w, h: ht}, id: id})
}

// at returns the id of the block at cell (x, y).
func (h hitList) at(x, y int) (id int, ok bool) {
	for _, b := range h {
		if x >= b.r.x && x < b.r.x+b.r.w && y >= b.r.y && y < b.r.y+b.r.h {
			return b.id, true
		}
	}
	return 0, false
}

// menuSounds are the effects the menus play (U2): a tick when the
// selection moves, a click on choose, a low sound on back, a buzz on a
// locked item.
var menuSounds = []audio.Effect{audio.MenuTick, audio.MenuClick, audio.MenuBack, audio.MenuBuzz}

// menuSound plays a menu effect. Like every effect it follows the Sound
// and Volume settings (the player holds them). lastCue remembers it, so
// tests can tell what a key did without an audio device.
func (a *App) menuSound(e audio.Effect) {
	a.lastCue, a.cues = e, a.cues+1
	if a.sound != nil {
		a.sound.Play(e)
	}
}

// resultSound plays the menu sound for what a list menu's key or click did.
func (a *App) resultSound(r ui.Result, moved bool) {
	switch {
	case r == ui.Back:
		a.menuSound(audio.MenuBack)
	case r == ui.Chosen || r == ui.Changed:
		a.menuSound(audio.MenuClick)
	case moved:
		a.menuSound(audio.MenuTick)
	}
}

// scaleColor returns RGB color c with its brightness times k (0-1).
func scaleColor(c term.Color, k float64) term.Color {
	r, g, b, ok := c.RGB()
	if !ok {
		return c
	}
	k = min(max(k, 0), 1)
	return term.RGB(uint8(float64(r)*k), uint8(float64(g)*k), uint8(float64(b)*k))
}
