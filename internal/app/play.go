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
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// The Play screen (U2, D54): the mode and the map picked on one screen,
// with a top-down preview of the map and the bests on it.

// playModes are the modes, in the order of the toggle.
var playModes = []game.PlayMode{game.Survival, game.Waves}

// Play screen layout, on the 80x24 stage.
const (
	playListRows = 10 // maps shown at once; the list scrolls beyond that
	playListW    = 37
	playPicRows  = 9 // rows of the preview picture
)

// Clickable ids: 0..n-1 are the maps; then these.
const (
	playHitStart = -1 - iota
	playHitSurvival
	playHitWaves
)

// playScreen is the Play screen.
type playScreen struct {
	mode game.PlayMode
	sel  int // the map
	top  int // first map shown in the list
	hits hitList
	pic  render.Frame // the preview picture
}

// playMenu opens the Play screen on the mode and map chosen last.
func (a *App) playMenu() *playScreen {
	return &playScreen{mode: a.playMode, sel: a.mapIndex}
}

func (p *playScreen) key(a *App, ev term.Key) {
	if ev.Action == term.Release {
		return
	}
	k, ok := input.MenuKeyFor(ev)
	if !ok {
		return
	}
	n := len(a.maps)
	switch k {
	case ui.KeyUp:
		p.sel = (p.sel + n - 1) % n
		a.menuSound(audio.MenuTick)
	case ui.KeyDown:
		p.sel = (p.sel + 1) % n
		a.menuSound(audio.MenuTick)
	case ui.KeyLeft, ui.KeyRight: // two modes: either key switches
		p.mode = playModes[(int(p.mode)+1)%len(playModes)]
		a.menuSound(audio.MenuTick)
	case ui.KeyEnter:
		p.start(a)
	case ui.KeyBack:
		a.menuSound(audio.MenuBack)
		a.switchTo(a.titleMenu())
	}
}

// mouse: a click picks a map or a mode, or starts.
func (p *playScreen) mouse(a *App, b mouseButton, down bool, x, y int) {
	if b != mouseLeft || !down {
		return
	}
	id, ok := p.hits.at(x, y)
	switch {
	case !ok:
	case id == playHitStart:
		p.start(a)
	case id == playHitSurvival, id == playHitWaves:
		p.mode = game.Survival
		if id == playHitWaves {
			p.mode = game.Waves
		}
		a.menuSound(audio.MenuTick)
	default:
		p.sel = id
		a.menuSound(audio.MenuTick)
	}
}

// start begins a run on the selected map in the selected mode; a locked
// map only buzzes.
func (p *playScreen) start(a *App) {
	if a.mapLocked(p.sel) {
		a.menuSound(audio.MenuBuzz)
		return
	}
	a.menuSound(audio.MenuClick)
	a.mapIndex, a.playMode = p.sel, p.mode
	a.newRun()
}

// modeNote is the one-line explanation of a mode.
func modeNote(m game.PlayMode) string {
	if m == game.Waves {
		return fmt.Sprintf("%d waves with breaks, clear them all to win", game.NumWaves)
	}
	return fmt.Sprintf("endless%stier up every %d kills or %s", ui.Sym().Sep, game.TierKills, render.Clock(game.TierTime))
}

// percent is share (0-1) as a whole percent.
func percent(share float64) int {
	return int(math.Round(100 * share))
}

// mapCounts are the doors and red barrels of m, for the preview line.
func mapCounts(m *world.Map) (doors, barrels int) {
	for _, c := range m.Cells {
		if c == world.ExplosiveBarrel {
			barrels++
		}
	}
	return len(m.Doors), barrels
}

// plural is "1 door", "3 doors".
func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// lockReason says what opens locked map i.
func (a *App) lockReason(i int) string {
	return fmt.Sprintf("locked: survive %s on %s", render.Clock(unlockAfter(a.cfg.UnlockRule)), a.maps[i-1].Name)
}

// bestLine is the BEST line for map i: the survival best and the waves
// best, or that there is none yet.
func (a *App) bestLine(i int) string {
	name, sep := a.maps[i].Name, ui.Sym().Sep
	var parts []string
	if b, ok := a.progress.Best[name]; ok { // ok is false if the map has no best yet
		parts = append(parts, fmt.Sprintf("survival %s%s%d kills%stier %d", render.Clock(b.Time), sep, b.Kills, sep, b.Tier))
	}
	if w, ok := a.progress.Waves[name]; ok {
		parts = append(parts, "waves "+wavesBestText(w))
	}
	if len(parts) == 0 {
		return "no runs here yet"
	}
	return strings.Join(parts, "    ")
}

func (p *playScreen) tick(a *App, now time.Time, dt float64) {
	a.drawBackdrop(dropNone, dt)
	s, sy := a.scr, ui.Sym()
	ox, oy := stage(s)
	p.hits = p.hits[:0]

	ui.SmallLogo(s, ox+2, oy+1, ui.ColBg)
	ui.PutRight(s, ox+stageW-2, oy+1, "PLAY", ui.Bold(ui.On(ui.ColAmber, ui.ColBg)))

	// The mode toggle.
	x := ui.Put(s, ox+2, oy+4, "MODE  ", ui.On(ui.ColDim, ui.ColBg))
	for i, m := range playModes {
		st := ui.On(ui.ColDim, ui.ColPanel)
		if m == p.mode {
			st = ui.StSelected
		}
		label := " " + strings.ToUpper(m.String()) + " "
		p.hits.add(x, oy+4, ui.RuneLen(label), 1, playHitSurvival-i)
		x = ui.Put(s, x, oy+4, label, st) + 1
	}
	ui.Put(s, x+2, oy+4, modeNote(p.mode), ui.StBg)

	// The map list.
	const listY = 6
	n := len(a.maps)
	p.top = min(max(p.top, p.sel-playListRows+1), p.sel) // keep the selection in view
	p.top = max(min(p.top, n-playListRows), 0)
	lx := ox + 2
	ui.Panel(s, lx, oy+listY, playListW, playListRows+2, ui.BoxRound, ui.StBorder, ui.StPanel)
	ui.BoxTitle(s, lx, oy+listY, "MAP", ui.StDim)
	ui.BoxTitle(s, lx, oy+listY+playListRows+1, "+ your maps (maps folder)", ui.StDim)
	for row := 0; row < playListRows && p.top+row < n; row++ {
		i := p.top + row
		m := a.maps[i]
		y := oy + listY + 1 + row
		st, marker := ui.StPanel, "  "
		locked := a.mapLocked(i)
		if locked {
			st = ui.StDim
		}
		if i == p.sel {
			st, marker = ui.StSelected, sy.Select+" "
			ui.Fill(s, lx+1, y, playListW-2, 1, st)
		}
		right := sy.Locked + " locked"
		if !locked {
			right = fmt.Sprintf("%dx%d", m.W, m.H)
			if seen, played := a.explored(i); played {
				right += fmt.Sprintf(" %3d%%", percent(exploredShare(m, seen)))
			} else {
				right += "   - "
			}
		}
		name := []rune(m.Name)
		if room := playListW - 6 - ui.RuneLen(right); len(name) > room {
			name = name[:room]
		}
		ui.Put(s, lx+1, y, marker+string(name), st)
		ui.PutRight(s, lx+playListW-2, y, right, st)
		p.hits.add(lx+1, y, playListW-2, 1, i)
	}

	// The preview: a top-down picture of the explored part, and a line
	// about the map.
	m := a.maps[p.sel]
	px, pw := lx+playListW+1, stageW-playListW-5
	ui.Panel(s, px, oy+listY, pw, playListRows+2, ui.BoxRound, ui.StBorder, ui.StPanel)
	ui.BoxTitle(s, px, oy+listY, strings.ToUpper(m.Name), ui.StTitle)
	seen, played := a.explored(p.sel)
	if !played || a.mapLocked(p.sel) {
		seen = make([]bool, m.W*m.H) // nothing seen: only the border shows
	}
	p.pic.PixelAspect = 1
	p.pic.Resize(pw-2, 2*playPicRows)
	p.pic.OffsetX, p.pic.OffsetY = px+1, oy+listY+1
	render.DrawMapPicture(&p.pic, m, seen)
	a.presentFrame(&p.pic)
	info := a.lockReason0(p.sel)
	if info == "" {
		doors, barrels := mapCounts(m)
		info = fmt.Sprintf("explored %d%%%s%s%s%s", percent(exploredShare(m, seen)), sy.Sep, plural(doors, "door"), sy.Sep, plural(barrels, "red barrel"))
	}
	ui.Center(s, px+1, pw-2, oy+listY+playPicRows+1, info, ui.StDim)

	best := "locked"
	if !a.mapLocked(p.sel) {
		best = a.bestLine(p.sel)
	}
	x = ui.Put(s, ox+2, oy+19, "BEST  ", ui.On(ui.ColDim, ui.ColBg))
	ui.Put(s, x, oy+19, best, ui.StBg)

	// The start button: dim while the map is locked.
	label, st := "  "+sy.Select+" START  ", ui.StSelected
	if a.mapLocked(p.sel) {
		st = ui.On(ui.ColDim, ui.ColPanel)
	}
	bx := ox + (stageW-ui.RuneLen(label))/2
	ui.Put(s, bx, oy+20, label, st)
	p.hits.add(bx, oy+20, ui.RuneLen(label), 1, playHitStart)

	ui.Hints(s, ox, stageW, oy+stageH-2, []ui.Hint{
		{Keys: sy.UpDown, Text: "map"}, {Keys: sy.LeftRight, Text: "mode"}, {Keys: sy.Enter, Text: "start"}, {Keys: "esc", Text: "back"},
	}, ui.ColBg)
	a.show()
}

// lockReason0 is lockReason for a locked map and "" for an open one.
func (a *App) lockReason0(i int) string {
	if !a.mapLocked(i) {
		return ""
	}
	return a.lockReason(i)
}
