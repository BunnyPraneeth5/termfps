package app

import (
	"fmt"
	"math"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
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
	mouseLeft      mouseButton = iota // shoot; pick menu items
	mouseRight                        // reload
	mouseWheelUp                      // previous gun (M11); one call per notch, always "down"
	mouseWheelDown                    // next gun (M11)
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
	selectGun     int // gun number key pressed (M11), 0 = none
	cycleGun      int // wheel notches since the last frame: + next gun, - previous

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
	case isGunKey(ev):
		p.selectGun, _ = input.GunKey(ev)
	default:
		if k, ok := input.KeyFor(ev); ok {
			a.keys.Press(k, ev.When)
		}
	}
}

// mouse: the left button shoots (held = keeps firing), the right button
// reloads (P1-16). The wheel switches guns (M11). There's no mouse-look (D28), so the position is unused.
func (p *playingScreen) mouse(a *App, b mouseButton, down bool, x, y int) {
	switch {
	case b == mouseLeft && down:
		p.fire = true // a click shorter than a frame still fires once
		p.shooting = true
	case b == mouseLeft:
		p.shooting = false
	case b == mouseRight && down:
		p.reload = true
	case b == mouseWheelUp:
		p.cycleGun--
	case b == mouseWheelDown:
		p.cycleGun++
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
	in.SelectGun, in.CycleGun = p.selectGun, max(min(p.cycleGun, 1), -1) // one gun per frame
	p.toggleMinimap, p.cycleMode, p.fire, p.reload = false, false, false, false
	p.selectGun, p.cycleGun = 0, 0

	st := a.state
	game.Update(st, in, dt)
	a.recordFinds() // enemies met, guns found (M18c)
	a.playSounds()
	if st.HasEvent(game.EventRenderModeChanged) {
		a.persist() // V changed a setting
	}
	// Notices (U4: a toast at the top). When several come in one frame the
	// later one here wins: a wave starting says more than its tier.
	sep := ui.Sym().Sep
	if st.HasEvent(game.EventRenderModeChanged) && st.Mode == game.ModeHalfBlock && !a.cautionShown {
		a.cautionShown = true
		a.toast = toast{text: caution, from: now, until: now.Add(cautionFor), halfBlockOnly: true}
	}
	if st.HasEvent(game.EventTierUp) {
		a.notify(now, tierNotice(st.Tier))
	}
	if st.HasEvent(game.EventWaveStarted) { // waves (M15); a cleared wave gets the banner instead
		a.notify(now, fmt.Sprintf("WAVE %d%s%d enemies%sGO", st.Wave, sep, game.WaveSize(st.Wave), sep))
	}
	if st.HasEvent(game.EventHuntStarted) {
		a.notify(now, "THE HUNT IS ON"+sep+"kill one to stop it")
	}
	n := unlocks(a.progress.UnlockedMaps, a.mapIndex, st.Time, a.builtinMaps(), unlockAfter(a.cfg.UnlockRule))
	if a.cfg.UnlockRule != "open" && n > a.progress.UnlockedMaps { // with all maps open there is nothing to announce
		a.progress.UnlockedMaps = n
		a.persist()
		a.notify(now, "NEW MAP UNLOCKED"+sep+a.maps[a.progress.UnlockedMaps-1].Name)
	}

	t0 := time.Now()
	a.drawScene()
	t1 := time.Now()
	a.present()
	minimapHidden := st.ShowMinimap && !render.DrawMinimap(a.scr, st, a.cfg.MinimapFog)
	a.drawWaveBanner()
	a.drawToast(now, st.ShowMinimap && !minimapHidden)
	render.DrawHUD(a.scr, render.HUD{
		FPS:           a.fps.value(),
		Player:        st.Player,
		Draw:          a.times.avg.draw,
		Present:       a.times.avg.present,
		Show:          a.times.avg.show,
		Keys:          keysLabel(a.cfg.KeyMode, a.focused, a.keys.Releases()),
		MinimapHidden: minimapHidden,
		Mode:          st.Mode,
		Kills:         st.Kills,
		Tier:          st.Tier,
		Wave:          wave(st),
		Break:         st.BreakLeft,
		Time:          st.Time,
	})
	t2 := time.Now()
	a.show()
	a.times.add(frameTimes{draw: t1.Sub(t0), present: t2.Sub(t1), show: time.Since(t2)})

	if st.Player.Dead() {
		a.endRun() // before the menu, so it shows the updated best
		a.switchTo(a.endMenu(false))
	}
	if st.Won { // M15
		a.endRun()
		a.switchTo(a.endMenu(true))
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
		a.sprites = append(a.sprites, render.EnemySprite(e))
	}
	for _, b := range st.Booms { // explosions (M12)
		a.sprites = append(a.sprites, render.BoomSprite(b))
	}
	for _, pr := range st.Map.Props { // barrels, crates, lamps, ... (M2)
		a.sprites = append(a.sprites, render.PropSprite(pr.Kind, pr.X, pr.Y))
	}
	for _, pk := range st.Pickups { // ammo, medkits, armor (M10)
		a.sprites = append(a.sprites, render.PickupSprite(pk))
	}
	render.DrawSprites(&a.frame, st.Player, a.sprites)
	render.ShakeView(&a.frame, st.Player) // M16: the world shakes, the gun doesn't
	render.DrawWeapon(&a.frame, st.Player)
	render.DrawCrosshair(&a.frame)
	render.DrawHitMarker(&a.frame, st.Player)    // M16
	render.DrawHitDirection(&a.frame, st.Player) // M16
	render.DrawHurt(&a.frame, st.Player)
}

// isGunKey reports whether the key is a gun number key (1-3, M11).
func isGunKey(ev term.Key) bool {
	_, ok := input.GunKey(ev)
	return ok
}

// tierNotice is the notice for reaching tier: what's new (M12 enemies,
// M13 what enemies learn), or just that it got harder.
func tierNotice(tier int) string {
	sep := ui.Sym().Sep
	what := "tougher and faster"
	switch tier {
	case game.HearTier:
		what = "runners join" + sep + "enemies hear you"
	case game.CoverTier:
		what = "tanks join" + sep + "shooters hide"
	case game.FlankTier:
		what = "exploders join" + sep + "zombies surround you"
	}
	return fmt.Sprintf("TIER %d%s%s", tier, sep, what)
}

// toast is the notice at the top of the game (U4).
type toast struct {
	text        string
	from, until time.Time
	// halfBlockOnly hides it as soon as the render mode isn't half-block
	// (the half-block caution).
	halfBlockOnly bool
}

// toastSlide is how long a toast takes to slide in from the top row, with
// animations on.
const toastSlide = 100 * time.Millisecond

// notify shows text as the game's notice for noticeFor.
func (a *App) notify(now time.Time, text string) {
	a.toast = toast{text: text, from: now, until: now.Add(noticeFor)}
}

// drawToast draws the notice, if one is showing: centered on the second
// row (sliding in from the first), in the part of the row the minimap
// leaves free when it would reach into the minimap.
func (a *App) drawToast(now time.Time, minimap bool) {
	t := a.toast
	if t.text == "" || !now.Before(t.until) || (t.halfBlockOnly && a.state.Mode != game.ModeHalfBlock) {
		return
	}
	y := 1
	if a.cfg.Animations && now.Sub(t.from) < toastSlide {
		y = 0
	}
	w, h := a.scr.Size()
	width := ui.RuneLen(t.text) + 4
	room := w
	if minimap {
		if mw := render.MinimapWidth(a.state.Map, w, h); (w+width)/2 > w-mw {
			room = w - mw // centered, it would cover the minimap: use what's left of it
		}
	}
	ui.Center(a.scr, 0, room, y, "  "+t.text+"  ", term.Style{Fg: ui.ColAmber, Bg: ui.ColToast, Bold: true})
}

// Wave banner size.
const (
	bannerW = 38
	bannerH = 7
	bannerY = 3
)

// drawWaveBanner draws the banner of a waves run's break (U4): the wave
// just cleared, the countdown to the next one as a number and a bar, and
// what's coming. Before the first wave it says to get ready.
func (a *App) drawWaveBanner() {
	st := a.state
	if !st.InBreak() || st.Won {
		return
	}
	s, sy := a.scr, ui.Sym()
	w, _ := s.Size()
	x := (w - bannerW) / 2
	card := ui.On(ui.ColText, ui.ColCard)
	ui.Panel(s, x, bannerY, bannerW, bannerH, ui.BoxRound, ui.On(ui.ColAmber, ui.ColCard), card)
	title, total := fmt.Sprintf("%s WAVE %d CLEARED", sy.Check, st.Wave), game.WaveBreak
	if st.Wave == 0 {
		title, total = "WAVES"+sy.Sep+"GET READY", game.FirstWaveBreak
	}
	next := st.Wave + 1
	ui.Center(s, x, bannerW, bannerY+1, title, ui.Bold(ui.On(ui.ColGreen, ui.ColCard)))
	ui.Center(s, x, bannerW, bannerY+2, fmt.Sprintf("next wave in %.0f", math.Ceil(st.BreakLeft)), ui.Bold(ui.On(ui.ColBright, ui.ColCard)))
	const cells = 24
	ui.Bar(s, x+(bannerW-ui.BarWidth(cells))/2, bannerY+3, cells, st.BreakLeft/total, ui.On(ui.ColAmber, ui.ColCard), ui.On(ui.ColBorder, ui.ColCard))
	ui.Center(s, x, bannerW, bannerY+4, fmt.Sprintf("wave %d: %d enemies%stier %d", next, game.WaveSize(next), sy.Sep, game.WaveTier(next)), card)
	ui.Center(s, x, bannerW, bannerY+5, "reload and pick things up", ui.On(ui.ColDim, ui.ColCard))
}

// wave is the HUD's wave number: 0 in survival (it shows the tier
// instead), else the current wave, or the next one during a break (M15).
func wave(st *game.State) int {
	switch {
	case st.Play != game.Waves:
		return 0
	case st.InBreak():
		return st.Wave + 1
	}
	return st.Wave
}
