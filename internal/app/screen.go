package app

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
)

// screen is one mode of the app: the game itself, or a menu (menus.go).
// It's an interface: any type with these three methods counts as a screen,
// so the app can switch between very different screens the same way.
type screen interface {
	// key handles one key press. It may set a.quit or switch screens.
	key(a *App, ev *tcell.EventKey)
	// click handles a left click on screen cell (x, y).
	click(a *App, x, y int)
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
	case input.IsQuit(ev): // Esc pauses (Ctrl+C is handled for every screen in step)
		a.switchTo(a.pauseMenu())
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

// click does nothing in the game yet; P1-16 adds shooting and reloading.
func (p *playingScreen) click(a *App, x, y int) {}

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
	if st.HasEvent(game.EventTierUp) {
		a.tierUntil = now.Add(tierNoticeFor)
	}
	notice := ""
	switch {
	case now.Before(a.tierUntil):
		notice = fmt.Sprintf("TIER %d: enemies are tougher and come faster.", st.Tier)
	case now.Before(a.cautionUntil) && st.Mode == game.ModeHalfBlock:
		notice = caution // only while half-block is actually on screen
	}

	t0 := time.Now()
	a.drawScene()
	t1 := time.Now()
	a.present()
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
		Kills:         st.Kills,
		Tier:          st.Tier,
		Time:          st.Time,
	})
	t2 := time.Now()
	a.scr.Show()
	a.times.add(frameTimes{draw: t1.Sub(t0), present: t2.Sub(t1), show: time.Since(t2)})

	if st.Player.Dead() {
		a.switchTo(a.gameOverMenu())
	}
}

// drawScene draws the 3D view of the current state into the Frame: walls,
// sprites, gun, crosshair, hurt border. The pause and game over menus use
// it too, so the frozen game stays visible behind them after a resize.
func (a *App) drawScene() {
	st := a.state
	render.DrawFrame(st, &a.frame)
	a.sprites = a.sprites[:0]
	for _, e := range st.Enemies {
		a.sprites = append(a.sprites, render.Sprite{
			X: e.X, Y: e.Y, Image: enemyImage(e.Kind), Height: 0.8, Flash: e.Flashing(),
		})
	}
	for _, pk := range st.Pickups {
		a.sprites = append(a.sprites, render.Sprite{X: pk.X, Y: pk.Y, Image: render.AmmoImage, Height: 0.3})
	}
	render.DrawSprites(&a.frame, st.Player, a.sprites)
	render.DrawWeapon(&a.frame, st.Player)
	render.DrawCrosshair(&a.frame)
	render.DrawHurt(&a.frame, st.Player)
}

// enemyImage is the sprite picture for an enemy kind.
func enemyImage(k game.EnemyKind) *render.Image {
	switch k {
	case game.KindZombie:
		return render.ZombieImage
	case game.KindShooter:
		return render.ShooterImage
	}
	panic(fmt.Sprintf("no image for enemy kind %d", k))
}
