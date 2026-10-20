package app

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
)

// screen is one mode of the app: playing now; title, pause and settings
// menus, map select and game over come later (P1-11, P1-12). It's an
// interface: any type with these two methods counts as a screen, so the
// app can switch between very different screens the same way.
type screen interface {
	// key handles one key press. It may set a.quit or switch screens.
	key(a *App, ev *tcell.EventKey)
	// tick advances one frame (dt seconds) and draws it.
	tick(a *App, now time.Time, dt float64)
}

// playingScreen is the game itself: 3D view, minimap and HUD.
type playingScreen struct {
	// One-shot inputs pressed since the last frame. With terminal key events,
	// holding Space sends repeats, so fire keeps going (limited by the
	// gun's fire rate). Async key mode reads Space and R directly.
	toggleMinimap bool
	cycleMode     bool
	fire          bool
	reload        bool
}

func (p *playingScreen) key(a *App, ev *tcell.EventKey) {
	switch {
	case input.IsQuit(ev): // Esc (Ctrl+C is handled for every screen in step)
		a.quit = true
	case input.IsToggleMinimap(ev):
		p.toggleMinimap = !p.toggleMinimap // two presses in one frame cancel
	case input.IsCycleRenderMode(ev):
		p.cycleMode = true
	case input.IsFire(ev):
		p.fire = true
	case input.IsReload(ev):
		p.reload = true
	default:
		if k, ok := input.KeyFor(ev); ok {
			a.keys.Press(k, ev.When())
		}
	}
}

func (p *playingScreen) tick(a *App, now time.Time, dt float64) {
	var in game.Input
	switch {
	case a.cfg.KeyMode == "events":
		in = a.keys.Input(now)
	case a.focused:
		in = input.AsyncInput()
	}
	in.ToggleMinimap, in.CycleRenderMode = p.toggleMinimap, p.cycleMode
	in.Fire = in.Fire || p.fire
	in.Reload = in.Reload || p.reload
	p.toggleMinimap, p.cycleMode, p.fire, p.reload = false, false, false, false

	st := a.state
	game.Update(st, in, dt)
	if st.HasEvent(game.EventRenderModeChanged) && st.Mode == game.ModeHalfBlock && !a.cautionShown {
		a.cautionShown = true
		a.cautionUntil = now.Add(cautionFor)
	}
	notice := ""
	if now.Before(a.cautionUntil) {
		notice = caution
	}

	t0 := time.Now()
	render.DrawFrame(st, &a.frame)
	render.DrawWeapon(&a.frame, st.Player)
	render.DrawCrosshair(&a.frame)
	t1 := time.Now()
	switch st.Mode {
	case game.ModeHalfBlock:
		render.PresentHalfBlock(a.scr, &a.frame)
	case game.ModeShade:
		render.PresentShade(a.scr, &a.frame)
	case game.ModeASCII:
		render.PresentASCII(a.scr, &a.frame)
	default:
		panic(fmt.Sprintf("unknown render mode %d", st.Mode))
	}
	minimapHidden := st.ShowMinimap && !render.DrawMinimap(a.scr, st)
	render.DrawHUD(a.scr, render.HUD{
		FPS:           a.fps.value(),
		Player:        st.Player,
		Draw:          a.times.avg.draw,
		Present:       a.times.avg.present,
		Show:          a.times.avg.show,
		Keys:          keysLabel(a.cfg.KeyMode, a.focused),
		MinimapHidden: minimapHidden,
		Mode:          st.Mode,
		Notice:        notice,
	})
	t2 := time.Now()
	a.scr.Show()
	a.times.add(frameTimes{draw: t1.Sub(t0), present: t2.Sub(t1), show: time.Since(t2)})
}
