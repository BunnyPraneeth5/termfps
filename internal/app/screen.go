package app

import (
	"fmt"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// screen is one mode of the app: the game itself, or a menu (menus.go).
// It's an interface: any type with these three methods counts as a screen,
// so the app can switch between very different screens the same way.
type screen interface {
	// key handles one key event: a press, a repeat or a release. It may set
	// a.quit or switch screens.
	key(a *App, ev term.Key)
	// mouse handles a mouse button going down (down = true) or up, with the
	// pointer on screen cell (x, y).
	mouse(a *App, b mouseButton, down bool, x, y int)
	// tick advances one frame (dt seconds) and draws it.
	tick(a *App, now time.Time, dt float64)
}

// mouseButton is a mouse button the game uses (D28).
type mouseButton int

const (
	mouseLeft  mouseButton = iota // shoot; pick menu items
	mouseRight                    // reload
)

// playingScreen is the game itself: 3D view, minimap and HUD.
type playingScreen struct {
	// One-shot inputs pressed since the last frame. With terminal key events,
	// holding Space sends repeats, so fire keeps going (limited by the
	// gun's fire rate). Async key mode reads Space and R directly.
	toggleMinimap bool
	cycleMode     bool
	fire          bool
	reload        bool

	// shooting is true while the left button, pressed during the game, is
	// held: it fires like holding Space. A button still held from a menu
	// click doesn't count, since this screen didn't see it go down.
	shooting bool
}

func (p *playingScreen) key(a *App, ev term.Key) {
	if ev.Action == term.Release {
		// Only movement keys are held; Esc, M, V, Space and R act on
		// presses.
		if k, ok := input.KeyFor(ev); ok {
			a.keys.Release(k)
		}
		return
	}
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
			a.keys.Press(k, ev.When)
		}
	}
}

// mouse: the left button shoots (held = keeps firing), the right button
// reloads (P1-16). There's no mouse-look (D28), so the position is unused.
func (p *playingScreen) mouse(a *App, b mouseButton, down bool, x, y int) {
	switch {
	case b == mouseLeft && down:
		p.fire = true // a click shorter than a frame still fires once
		p.shooting = true
	case b == mouseLeft:
		p.shooting = false
	case b == mouseRight && down:
		p.reload = true
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
	in.Fire = in.Fire || p.fire || p.shooting
	in.Reload = in.Reload || p.reload
	p.toggleMinimap, p.cycleMode, p.fire, p.reload = false, false, false, false

	st := a.state
	game.Update(st, in, dt)
	a.playSounds()
	if st.HasEvent(game.EventRenderModeChanged) {
		a.persist() // V changed a setting
	}
	if st.HasEvent(game.EventRenderModeChanged) && st.Mode == game.ModeHalfBlock && !a.cautionShown {
		a.cautionShown = true
		a.cautionUntil = now.Add(cautionFor)
	}
	if st.HasEvent(game.EventTierUp) {
		a.tierUntil = now.Add(tierNoticeFor)
	}
	n := unlocks(a.progress.UnlockedMaps, a.mapIndex, st.Time, len(a.maps), unlockAfter(a.cfg.UnlockRule))
	if a.cfg.UnlockRule != "open" && n > a.progress.UnlockedMaps { // with all maps open there is nothing to announce
		a.progress.UnlockedMaps = n
		a.persist()
		a.unlockUntil = now.Add(tierNoticeFor)
	}
	notice := ""
	switch {
	case now.Before(a.unlockUntil):
		notice = fmt.Sprintf("New map unlocked: %s (Select map in the main menu).", a.maps[a.progress.UnlockedMaps-1].Name)
	case now.Before(a.tierUntil):
		notice = fmt.Sprintf("TIER %d: enemies are tougher and come faster.", st.Tier)
	case now.Before(a.cautionUntil) && st.Mode == game.ModeHalfBlock:
		notice = caution // only while half-block is actually on screen
	}

	t0 := time.Now()
	a.drawScene()
	t1 := time.Now()
	a.present()
	minimapHidden := st.ShowMinimap && !render.DrawMinimap(a.scr, st, a.cfg.MinimapFog)
	render.DrawHUD(a.scr, render.HUD{
		FPS:           a.fps.value(),
		Player:        st.Player,
		Draw:          a.times.avg.draw,
		Present:       a.times.avg.present,
		Show:          a.times.avg.show,
		Keys:          keysLabel(a.cfg.KeyMode, a.focused, a.keys.Releases()),
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
		a.endRun() // before the menu, so it shows the updated best
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
	for _, pr := range st.Map.Props { // barrels, crates, lamps, ... (M2)
		a.sprites = append(a.sprites, render.PropSprite(pr.Kind, pr.X, pr.Y))
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
