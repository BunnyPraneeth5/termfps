package app

import (
	"strings"
	"testing"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// newMenuApp is like newTestApp but stays on the title menu.
func newMenuApp(t *testing.T) (*App, *term.Fake) {
	t.Helper()
	s := term.NewFake(80, 24)
	m, err := world.Parse([]byte(testMap))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cfg := Config{FPS: 30, KeyMode: "events", PixelAspect: 1, Textures: true, MinimapAtStart: true, Sound: true, Volume: 80, MixedBackground: 20}
	return New(s, []*world.Map{m}, cfg, t0), s
}

// press runs one frame with the given keys and returns whether it quit.
func press(a *App, now *time.Time, evs ...term.Key) bool {
	*now = now.Add(time.Second / 30)
	batch := make([]term.Event, len(evs))
	for i, ev := range evs {
		batch[i] = ev
	}
	return a.step(*now, batch)
}

// onTitle reports whether the title screen is showing.
func onTitle(a *App) bool {
	_, ok := a.current.(*titleScreen)
	return ok
}

func screenHas(s *term.Fake, text string) bool {
	_, h := s.Size()
	for row := 0; row < h; row++ {
		if strings.Contains(s.Row(row), text) {
			return true
		}
	}
	return false
}

// quitToMenu pauses the game and picks Quit to menu, the pause menu's
// last item (Up from Resume wraps to it).
func quitToMenu(a *App, now *time.Time) {
	press(a, now, esc, up, enter)
}

var (
	left  = key(term.KeyLeft, 0)
	up    = key(term.KeyUp, 0)
	down  = key(term.KeyDown, 0)
	right = key(term.KeyRight, 0)
	enter = key(term.KeyEnter, 0)
	esc   = key(term.KeyEscape, 0)
)

// The app starts on the title menu; Play starts the game.
func TestTitlePlay(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	press(a, &now)
	if !onTitle(a) || !screenHas(s, "PLAY") || !screenHas(s, "SETTINGS") || !screenHas(s, Version) {
		t.Fatal("title screen not shown at start, with its menu and the version")
	}
	press(a, &now, esc)
	if !onTitle(a) || a.quit {
		t.Error("Esc on the title screen left it; want nothing to happen")
	}
	press(a, &now, enter, enter) // Play, then Start (U2)
	if _, ok := a.current.(*playingScreen); !ok {
		t.Errorf("Enter on Play: screen is %T, want the game", a.current)
	}
}

func TestTitleQuitAndCtrlC(t *testing.T) {
	a, _ := newMenuApp(t)
	now := t0
	if !press(a, &now, up, enter) { // Up from Play wraps to Quit
		t.Error("Quit didn't quit")
	}
	a, _ = newMenuApp(t)
	if !press(a, &now, key(term.KeyCtrlC, 0)) {
		t.Error("Ctrl+C didn't quit from the menu")
	}
	a, _ = newTestApp(t)
	if !press(a, &now, key(term.KeyCtrlC, 0)) {
		t.Error("Ctrl+C didn't quit from the game")
	}
}

// Esc pauses: the game stops updating and is drawn dimmed behind the menu.
// Esc again (or Resume) goes back.
func TestPauseFreezesGame(t *testing.T) {
	a, s := newTestApp(t)
	now := t0
	press(a, &now)
	press(a, &now, esc)
	if !screenHas(s, "PAUSED") {
		t.Fatal("no pause menu after Esc")
	}
	time0, x0 := a.state.Time, a.state.Player.X
	for range 10 {
		press(a, &now, key(term.KeyRune, 'w'))
	}
	if a.state.Time != time0 || a.state.Player.X != x0 {
		t.Error("game updated while paused")
	}
	press(a, &now, esc)
	if _, ok := a.current.(*playingScreen); !ok {
		t.Errorf("Esc in the pause menu: screen is %T, want the game", a.current)
	}
	press(a, &now)
	if a.state.Time == time0 {
		t.Error("game didn't run again after resuming")
	}
}

func TestPauseRestartAndQuitToMenu(t *testing.T) {
	a, _ := newTestApp(t)
	now := t0
	for range 30 {
		press(a, &now)
	}
	press(a, &now, esc, down, enter) // Restart
	// The restart frame itself runs one tick, so time is just above 0.
	if a.state.Time > 0.1 {
		t.Errorf("Restart: time = %v, want a fresh run", a.state.Time)
	}
	press(a, &now, esc, up, enter) // Quit to menu (wraps from Resume)
	press(a, &now)
	if !onTitle(a) {
		t.Error("Quit to menu didn't show the title menu")
	}
}

// Settings opened from the pause menu go back to it.
func TestSettingsFromPause(t *testing.T) {
	a, s := newTestApp(t)
	now := t0
	press(a, &now, esc, down, down, down, enter) // Pause -> Settings
	if _, ok := a.current.(*settingsScreen); !ok {
		t.Fatalf("the pause card's fourth item opened %T, want the Settings", a.current)
	}
	press(a, &now, esc)
	press(a, &now)
	if !screenHas(s, "PAUSED") {
		t.Error("Back from settings didn't return to the pause menu")
	}
}

// Clicking a menu item picks it; the release doesn't click again.
func TestMenuClick(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	press(a, &now) // draw once, so the menu knows where its items are
	_, h := s.Size()
	row := -1
	for y := 0; y < h; y++ {
		if strings.Contains(s.Row(y), "SETTINGS") {
			row = y
		}
	}
	if row < 0 {
		t.Fatal("Settings item not on screen")
	}
	now = now.Add(time.Second / 30)
	a.step(now, []term.Event{
		term.Mouse{X: 40, Y: row, Buttons: term.ButtonLeft},
		term.Mouse{X: 40, Y: row},
	})
	if onTitle(a) || !screenHas(s, "Render mode") {
		t.Error("clicking Settings didn't open the settings")
	}
}

// Dying shows the game over menu; Main menu goes to the title.
func TestGameOverMainMenu(t *testing.T) {
	a, _ := newTestApp(t)
	now := t0
	a.state.Player.HP = 0
	press(a, &now)
	press(a, &now)
	if es, ok := a.current.(*endScreen); !ok || es.won {
		t.Fatal("no game over screen")
	}
	press(a, &now, right, right, enter) // Try again, Change map, Main menu
	press(a, &now)
	if !onTitle(a) {
		t.Error("Main menu didn't show the title menu")
	}
}

func TestUnlockRule(t *testing.T) {
	cases := []struct {
		name                string
		open, played, total int
		survived            float64
		want                int
	}{
		{"5:00 on the newest open map", 1, 0, 4, 300, 2},
		{"not long enough", 1, 0, 4, 299.9, 1},
		{"an older map doesn't count", 2, 0, 4, 600, 2},
		{"last map: nothing left", 4, 3, 4, 900, 4},
	}
	for _, c := range cases {
		if got := unlocks(c.open, c.played, c.survived, c.total, unlockAfter("5:00")); got != c.want {
			t.Errorf("%s: unlocks = %d, want %d", c.name, got, c.want)
		}
	}
}

// Locked maps are dimmed and can't be picked; surviving 5 minutes opens the
// next one, with a notice, and then it can be played.
func TestMapUnlockAndSelect(t *testing.T) {
	s := term.NewFake(80, 24)
	var maps []*world.Map
	for _, name := range []string{"First", "Second"} {
		m, err := world.Parse([]byte("; name: " + name + "\n" + testMap))
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		maps = append(maps, m)
	}
	a := New(s, maps, Config{FPS: 30, KeyMode: "events", PixelAspect: 1}, t0)
	now := t0
	press(a, &now, enter) // Play: the Play screen (U2)
	press(a, &now)
	if !screenHas(s, "▒ locked") {
		t.Fatal("second map not shown as locked")
	}
	press(a, &now, down) // onto the locked map: it says what opens it
	press(a, &now)
	if !screenHas(s, "locked: survive 5:00 on First") {
		t.Error("the locked map doesn't say what opens it")
	}
	press(a, &now, enter) // Start on a locked map: a buzz, nothing else
	if _, ok := a.current.(*playScreen); !ok || a.lastCue != audio.MenuBuzz {
		t.Fatalf("Start on a locked map: screen %T, sound %d; want the Play screen and a buzz", a.current, a.lastCue)
	}
	press(a, &now, up, enter) // First, Start
	a.state.Spawning = false
	a.state.Time = unlockAfter("5:00") - 0.01
	press(a, &now)
	if a.progress.UnlockedMaps != 2 || !screenHas(s, "NEW MAP UNLOCKED · Second") {
		t.Fatalf("unlocked = %d after 5:00, want 2 with a notice", a.progress.UnlockedMaps)
	}
	quitToMenu(a, &now)
	press(a, &now, enter)       // Play
	press(a, &now, down, enter) // Second, Start
	if a.mapIndex != 1 || a.state.Map.Name != "Second" {
		t.Errorf("playing map %d %q, want 1 \"Second\"", a.mapIndex, a.state.Map.Name)
	}
}

// Menus act on presses only: a release doesn't move the selection.
func TestMenuIgnoresReleases(t *testing.T) {
	a, _ := newMenuApp(t)
	now := t0
	sel := a.current.(*titleScreen).sel
	press(a, &now, release(term.KeyDown, 0))
	if got := a.current.(*titleScreen).sel; got != sel {
		t.Errorf("selection %d -> %d after a Down release, want unchanged", sel, got)
	}
}

// Under "2:00" two minutes open the next map; under "open" every map can
// be picked without unlocking anything (M7).
func TestUnlockRuleSetting(t *testing.T) {
	if unlockAfter("2:00") != 120 || unlockAfter("5:00") != 300 || unlockAfter("") != 300 {
		t.Errorf("unlockAfter: 2:00 = %v, 5:00 = %v, unset = %v", unlockAfter("2:00"), unlockAfter("5:00"), unlockAfter(""))
	}
	a, _ := newMenuApp(t)
	a.maps = append(a.maps, a.maps[0].Clone(), a.maps[0].Clone())
	if !a.mapLocked(1) || !a.mapLocked(2) || a.mapLocked(0) {
		t.Fatal("with the default rule, maps after the first should be locked")
	}
	a.cfg.UnlockRule = "open"
	if a.mapLocked(1) || a.mapLocked(2) {
		t.Error("with all maps open, a map is still locked")
	}
	if a.progress.UnlockedMaps != 1 {
		t.Errorf("the open rule changed the saved progress to %d", a.progress.UnlockedMaps)
	}
}

// The player's own maps (M9) are never locked and don't count for unlocks;
// StartMap picks the one -map named.
func TestCustomMapsOpen(t *testing.T) {
	m, err := world.Parse([]byte(testMap))
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{FPS: 30, KeyMode: "events", PixelAspect: 1, BuiltinMaps: 2, StartMap: 2}
	a := New(term.NewFake(80, 24), []*world.Map{m, m.Clone(), m.Clone()}, cfg, t0)
	if !a.mapLocked(1) || a.mapLocked(2) {
		t.Errorf("locked: built-in 2 %v, custom %v; want true, false", a.mapLocked(1), a.mapLocked(2))
	}
	if a.mapIndex != 2 {
		t.Errorf("map index %d, want 2 (StartMap)", a.mapIndex)
	}
	if a.builtinMaps() != 2 {
		t.Errorf("builtinMaps = %d, want 2", a.builtinMaps())
	}
}
