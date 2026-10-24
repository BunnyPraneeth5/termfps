package app

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// newMenuApp is like newTestApp but stays on the title menu.
func newMenuApp(t *testing.T) (*App, tcell.SimulationScreen) {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("simulation screen Init: %v", err)
	}
	s.SetSize(80, 24)
	t.Cleanup(s.Fini)
	m, err := world.Parse([]byte(testMap))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cfg := Config{FPS: 30, KeyMode: "events", PixelAspect: 1, Textures: true, MinimapAtStart: true, Sound: true, Volume: 80}
	return New(s, []*world.Map{m}, cfg, t0), s
}

// press runs one frame with the given keys and returns whether it quit.
func press(a *App, now *time.Time, evs ...*tcell.EventKey) bool {
	*now = now.Add(time.Second / 30)
	batch := make([]tcell.Event, len(evs))
	for i, ev := range evs {
		batch[i] = ev
	}
	return a.step(*now, batch)
}

func screenHas(s tcell.Screen, text string) bool {
	_, h := s.Size()
	for row := 0; row < h; row++ {
		if strings.Contains(rowText(s, row), text) {
			return true
		}
	}
	return false
}

var (
	up    = key(tcell.KeyUp, 0)
	down  = key(tcell.KeyDown, 0)
	right = key(tcell.KeyRight, 0)
	enter = key(tcell.KeyEnter, 0)
	esc   = key(tcell.KeyEscape, 0)
)

// The app starts on the title menu; Play starts the game.
func TestTitlePlay(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	press(a, &now)
	if !screenHas(s, "TERMFPS") || !screenHas(s, "Select map") {
		t.Fatal("title menu not shown at start")
	}
	press(a, &now, esc)
	if _, ok := a.current.(*menuScreen); !ok || a.quit {
		t.Error("Esc on the title menu left it; want nothing to happen")
	}
	press(a, &now, enter)
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
	if !press(a, &now, key(tcell.KeyCtrlC, 0)) {
		t.Error("Ctrl+C didn't quit from the menu")
	}
	a, _ = newTestApp(t)
	if !press(a, &now, key(tcell.KeyCtrlC, 0)) {
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
		press(a, &now, key(tcell.KeyRune, 'w'))
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
	a, s := newTestApp(t)
	now := t0
	for range 30 {
		press(a, &now)
	}
	press(a, &now, esc, down, down, enter) // Restart
	// The restart frame itself runs one tick, so time is just above 0.
	if a.state.Time > 0.1 {
		t.Errorf("Restart: time = %v, want a fresh run", a.state.Time)
	}
	press(a, &now, esc, up, enter) // Quit to menu (wraps from Resume)
	press(a, &now)
	if !screenHas(s, "TERMFPS") {
		t.Error("Quit to menu didn't show the title menu")
	}
}

// Settings change the app right away and Back returns to where they were
// opened from.
func TestSettingsApplyRightAway(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	press(a, &now, down, down, enter) // Settings
	if !screenHas(s, "SETTINGS") {
		t.Fatal("settings menu not shown")
	}
	// Render mode starts at ascii (256-color screen); Right wraps to half-block.
	press(a, &now, right)
	if a.state.Mode != game.ModeHalfBlock {
		t.Errorf("mode = %v after Right, want half-block", a.state.Mode)
	}
	press(a, &now, down, right) // FPS 30 -> 60
	if a.cfg.FPS != 60 {
		t.Errorf("FPS = %d, want 60", a.cfg.FPS)
	}
	press(a, &now, down, enter) // minimap at start: on -> off
	if a.cfg.MinimapAtStart {
		t.Error("minimap at start still on")
	}
	press(a, &now, down, enter) // textures: on -> off
	if a.cfg.Textures || a.frame.Textures {
		t.Error("textures still on")
	}
	press(a, &now, down, down, right) // volume 80% -> 90%
	if a.cfg.Volume != 90 {
		t.Errorf("volume = %d, want 90", a.cfg.Volume)
	}
	press(a, &now, down, down, right) // pixel aspect 1 -> 1.1
	if a.frame.PixelAspect != 1.1 {
		t.Errorf("pixel aspect = %v, want 1.1", a.frame.PixelAspect)
	}
	press(a, &now, esc)
	press(a, &now)
	if !screenHas(s, "TERMFPS") {
		t.Error("Esc in settings didn't go back to the title menu")
	}
	press(a, &now, up, up, enter) // Play (from Settings: up twice)
	if a.state.ShowMinimap {
		t.Error("new run shows the minimap; setting says off")
	}
}

// Settings opened from the pause menu go back to it.
func TestSettingsFromPause(t *testing.T) {
	a, s := newTestApp(t)
	now := t0
	press(a, &now, esc, down, enter) // Pause -> Settings
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
		if strings.Contains(rowText(s, y), "Settings") {
			row = y
		}
	}
	if row < 0 {
		t.Fatal("Settings item not on screen")
	}
	now = now.Add(time.Second / 30)
	a.step(now, []tcell.Event{
		tcell.NewEventMouse(40, row, tcell.Button1, tcell.ModNone),
		tcell.NewEventMouse(40, row, tcell.ButtonNone, tcell.ModNone),
	})
	if !screenHas(s, "SETTINGS") {
		t.Error("clicking Settings didn't open the settings menu")
	}
}

// Dying shows the game over menu; Main menu goes to the title.
func TestGameOverMainMenu(t *testing.T) {
	a, s := newTestApp(t)
	now := t0
	a.state.Player.HP = 0
	press(a, &now)
	press(a, &now)
	if !screenHas(s, "YOU DIED") {
		t.Fatal("no game over menu")
	}
	press(a, &now, down, enter)
	press(a, &now)
	if !screenHas(s, "TERMFPS") {
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
		if got := unlocks(c.open, c.played, c.survived, c.total); got != c.want {
			t.Errorf("%s: unlocks = %d, want %d", c.name, got, c.want)
		}
	}
}

// Locked maps are dimmed and can't be picked; surviving 5 minutes opens the
// next one, with a notice, and then it can be played.
func TestMapUnlockAndSelect(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("simulation screen Init: %v", err)
	}
	s.SetSize(80, 24)
	t.Cleanup(s.Fini)
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
	press(a, &now, down, enter) // Select map
	press(a, &now)
	if !screenHas(s, "locked: survive 5:00 on First") {
		t.Fatal("second map not shown as locked")
	}
	press(a, &now, down) // skips the locked map, lands on Back
	if a.current.(*menuScreen).menu.Selected != 2 {
		t.Errorf("selection = %d after Down, want 2 (Back; the locked map is skipped)", a.current.(*menuScreen).menu.Selected)
	}
	press(a, &now, up, enter) // First
	a.state.Spawning = false
	a.state.Time = unlockAfter - 0.01
	press(a, &now)
	if a.unlocked != 2 || !screenHas(s, "New map unlocked: Second") {
		t.Fatalf("unlocked = %d after 5:00, want 2 with a notice", a.unlocked)
	}
	press(a, &now, esc, up, enter) // Pause -> Quit to menu
	press(a, &now, down, enter)    // Select map
	press(a, &now, down, enter)    // Second
	if a.mapIndex != 1 || a.state.Map.Name != "Second" {
		t.Errorf("playing map %d %q, want 1 \"Second\"", a.mapIndex, a.state.Map.Name)
	}
}
