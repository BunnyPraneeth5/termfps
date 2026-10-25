package app

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

const testMap = "##########\n" +
	"#P.......#\n" +
	"#........#\n" +
	"#...##...#\n" +
	"#........#\n" +
	"##########\n"

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// newTestApp runs the app on tcell's in-memory screen (80x24), with
// terminal key events (async keys would read the real keyboard).
func newTestApp(t *testing.T) (*App, tcell.SimulationScreen) {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("simulation screen Init: %v", err)
	}
	s.SetSize(80, 24)
	t.Cleanup(s.Fini) // runs when the test ends
	m, err := world.Parse([]byte(testMap))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	a := New(s, []*world.Map{m}, Config{FPS: 30, KeyMode: "events", PixelAspect: 1, MinimapAtStart: true}, t0)
	a.newRun() // start in the game, past the title menu
	return a, s
}

func key(k tcell.Key, r rune) *tcell.EventKey {
	return tcell.NewEventKey(k, r, tcell.ModNone)
}

func rowText(s tcell.Screen, row int) string {
	w, _ := s.Size()
	var b strings.Builder
	for x := 0; x < w; x++ {
		str, _, _ := s.Get(x, row)
		b.WriteString(str)
	}
	return b.String()
}

// The simulation screen reports 256 colors, so the app starts in ASCII
// (D17), draws the view, and shows the HUD.
func TestStepDrawsGame(t *testing.T) {
	a, s := newTestApp(t)
	if a.state.Mode != game.ModeASCII {
		t.Errorf("mode = %v on a 256-color screen, want ascii", a.state.Mode)
	}
	if quit := a.step(t0.Add(time.Second/30), nil); quit {
		t.Fatal("step quit without input")
	}
	if hud := rowText(s, 23); !strings.Contains(hud, "FPS") || !strings.Contains(hud, "| ascii") {
		t.Errorf("HUD row = %q, want FPS and the mode", hud)
	}
	if str, _, _ := s.Get(0, 10); str == " " || str == "" {
		t.Errorf("view cell (0, 10) = %q, want a wall or floor character", str)
	}
}

// Keys reach the game through the playing screen: W moves forward.
func TestKeysMovePlayer(t *testing.T) {
	a, _ := newTestApp(t)
	x0 := a.state.Player.X
	now := t0
	for i := 0; i < 10; i++ {
		now = now.Add(time.Second / 30)
		ev := tcell.NewEventKey(tcell.KeyRune, 'w', tcell.ModNone)
		a.step(now, []tcell.Event{ev})
	}
	if a.state.Player.X <= x0 {
		t.Errorf("x = %v after holding W, want more than %v", a.state.Player.X, x0)
	}
}

// V cycles the mode; the caution shows the first time it comes back to
// half-block, and never again.
func TestCautionOnce(t *testing.T) {
	a, _ := newTestApp(t)
	a.state.Mode = game.ModeASCII
	now := t0
	press := func() {
		now = now.Add(time.Second / 30)
		a.step(now, []tcell.Event{key(tcell.KeyRune, 'v')})
	}
	press() // ascii -> mixed
	if a.cautionShown {
		t.Fatal("caution shown before half-block")
	}
	press() // mixed -> half-block
	if !a.cautionShown || !a.cautionUntil.After(now) {
		t.Fatal("no caution when switching to half-block")
	}
	until := a.cautionUntil
	press()
	press()
	press()
	press() // around again to half-block (4 modes)
	if a.cautionUntil != until {
		t.Error("caution shown a second time")
	}
}

func TestTooSmallSkipsGame(t *testing.T) {
	a, s := newTestApp(t)
	s.SetSize(40, 10)
	x0 := a.state.Player.X
	a.step(t0.Add(time.Second/30), []tcell.Event{
		tcell.NewEventResize(40, 10),
		key(tcell.KeyRune, 'w'),
	})
	if a.state.Player.X != x0 {
		t.Error("game updated while the terminal was too small")
	}
	found := false
	for row := 0; row < 10; row++ {
		if strings.Contains(rowText(s, row), "too small") {
			found = true
		}
	}
	if !found {
		t.Error("no 'too small' message")
	}
}

func TestResizeResizesFrame(t *testing.T) {
	a, s := newTestApp(t)
	s.SetSize(100, 30)
	a.step(t0.Add(time.Second/30), []tcell.Event{tcell.NewEventResize(100, 30)})
	if a.frame.W != 100 || a.frame.H != 58 {
		t.Errorf("frame = %dx%d after resize to 100x30, want 100x58", a.frame.W, a.frame.H)
	}
}

// recordScreen is a fake screen that remembers what it was asked to do.
type recordScreen struct {
	keys  int
	ticks int
}

func (r *recordScreen) key(a *App, ev *tcell.EventKey)         { r.keys++ }
func (r *recordScreen) click(a *App, x, y int)                 {}
func (r *recordScreen) tick(a *App, now time.Time, dt float64) { r.ticks++ }

// After switchTo, keys and frames go to the new screen.
func TestSwitchTo(t *testing.T) {
	a, _ := newTestApp(t)
	rec := &recordScreen{}
	a.switchTo(rec)
	a.step(t0.Add(time.Second/30), []tcell.Event{key(tcell.KeyRune, 'w'), key(tcell.KeyEscape, 0)})
	if rec.keys != 2 || rec.ticks != 1 {
		t.Errorf("new screen got %d keys and %d ticks, want 2 and 1", rec.keys, rec.ticks)
	}
	if a.quit {
		t.Error("Esc quit the app; it should have gone to the new screen")
	}
}

// Space fires through the playing screen; R starts a reload; the HUD shows
// the ammo.
func TestSpaceFiresAndRReloads(t *testing.T) {
	a, s := newTestApp(t)
	a.step(t0.Add(time.Second/30), []tcell.Event{key(tcell.KeyRune, ' ')})
	if got := a.state.Player.Weapon.Mag; got != 24 {
		t.Errorf("magazine = %d after Space, want 24", got)
	}
	if hud := rowText(s, 23); !strings.Contains(hud, "AMMO 24 / 50") {
		t.Errorf("HUD %q, want AMMO 24 / 50", hud)
	}
	a.step(t0.Add(2*time.Second/30), []tcell.Event{key(tcell.KeyRune, 'r')})
	if !a.state.Player.Weapon.Reloading() {
		t.Error("R didn't start a reload")
	}
}

// Zombies from the map are drawn (green). Standing next to one, the player
// dies, the Game over screen shows, and Enter starts a fresh run.
func TestZombieKillsPlayerAndRetry(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("simulation screen Init: %v", err)
	}
	s.SetSize(80, 24)
	t.Cleanup(s.Fini)
	m, err := world.Parse([]byte("##########\n#PZ......#\n##########\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	a := New(s, []*world.Map{m}, Config{FPS: 30, KeyMode: "events", PixelAspect: 1}, t0)
	a.newRun()
	a.state.Mode = game.ModeHalfBlock

	now := t0
	a.step(now.Add(time.Second/30), nil)
	green := false
	for _, p := range a.frame.Pix {
		green = green || p.Tint == render.TintGreen
	}
	if !green {
		t.Error("zombie not drawn")
	}

	for i := 0; i < 15*30 && !a.state.Player.Dead(); i++ {
		now = now.Add(time.Second / 30)
		a.step(now, nil)
	}
	if !a.state.Player.Dead() {
		t.Fatal("player still alive after 15 s next to a zombie")
	}
	now = now.Add(time.Second / 30)
	a.step(now, nil) // the Game over screen draws
	found := false
	for row := 0; row < 24; row++ {
		found = found || strings.Contains(rowText(s, row), "YOU DIED")
	}
	if !found {
		t.Fatal("no YOU DIED message")
	}

	now = now.Add(time.Second / 30)
	a.step(now, []tcell.Event{key(tcell.KeyEnter, 0)})
	if a.state.Player.HP != game.PlayerMaxHP || len(a.state.Enemies) != 1 || a.state.Mode != game.ModeHalfBlock {
		t.Errorf("after retry: HP %d, enemies %d, mode %v; want full HP, the map's zombie, mode kept",
			a.state.Player.HP, len(a.state.Enemies), a.state.Mode)
	}
}

// Leaving half-block within the caution's 5 seconds hides it right away
// (bug seen in the user's video: the caution stayed on in shade mode).
func TestCautionHiddenOutsideHalfBlock(t *testing.T) {
	a, s := newTestApp(t)
	a.state.Mode = game.ModeMixed
	now := t0.Add(time.Second / 30)
	a.step(now, []tcell.Event{key(tcell.KeyRune, 'v')}) // mixed -> half-block: caution on
	if !strings.Contains(rowText(s, 22)+rowText(s, 21), "Half-block needs") {
		t.Fatal("caution not shown in half-block")
	}
	now = now.Add(time.Second / 30)
	a.step(now, []tcell.Event{key(tcell.KeyRune, 'v')}) // half-block -> shade, within 5 s
	if strings.Contains(rowText(s, 22)+rowText(s, 21), "Half-block needs") {
		t.Error("caution still shown after switching to shade")
	}
}

// A tier-up shows "TIER 2" above the HUD.
func TestTierNotice(t *testing.T) {
	a, s := newTestApp(t)
	a.state.Kills = game.TierKills
	a.step(t0.Add(time.Second/30), nil)
	if a.state.Tier != 2 {
		t.Fatalf("tier = %d, want 2", a.state.Tier)
	}
	if txt := rowText(s, 22) + rowText(s, 21); !strings.Contains(txt, "TIER 2") {
		t.Errorf("no TIER 2 notice above the HUD, got %q", txt)
	}
}
