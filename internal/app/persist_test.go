package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/save"
	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// withStore makes the app save into a temp folder.
func withStore(t *testing.T, a *App) save.Store {
	t.Helper()
	store := save.Store{Path: filepath.Join(t.TempDir(), "save.json")}
	a.useSaved(Saved{Store: &store})
	return store
}

// A run that ends by dying updates the bests and total kills, saves them,
// and the game over menu shows the best.
func TestDeathRecordsAndSaves(t *testing.T) {
	a, s := newMenuApp(t)
	store := withStore(t, a)
	now := t0
	press(a, &now, enter, enter) // Play, Survival
	a.state.Kills, a.state.Time = 7, 95
	a.state.Player.HP = 0
	press(a, &now)
	press(a, &now)
	if !screenHas(s, "Best on : 7 kills, 1:35, tier 1") { // the test map has no name
		t.Error("game over menu doesn't show the best")
	}
	d, problem := save.Load(store)
	if problem != "" {
		t.Fatalf("Load: %s", problem)
	}
	if b := d.Progress.Best[""]; b.Kills != 7 || b.Time < 95 || d.Progress.TotalKills != 7 {
		t.Errorf("saved best %+v, total %d; want 7 kills, 1:35, total 7", b, d.Progress.TotalKills)
	}

	// Try again, then a worse run: the bests stay, total kills add up.
	press(a, &now, enter) // Try again (same mode, no mode menu)
	a.state.Kills, a.state.Time = 2, 10
	a.state.Player.HP = 0
	press(a, &now)
	d, _ = save.Load(store)
	if b := d.Progress.Best[""]; b.Kills != 7 || b.Time < 95 || d.Progress.TotalKills != 9 {
		t.Errorf("after a worse run: best %+v, total %d; want the old best, total 9", b, d.Progress.TotalKills)
	}
}

// Restart and Quit to menu count the run once; Ctrl+C in the game counts it.
func TestAbandonedRunsCountOnce(t *testing.T) {
	a, _ := newMenuApp(t)
	store := withStore(t, a)
	now := t0
	press(a, &now, enter, enter) // Play, Survival
	a.state.Kills = 3
	press(a, &now, esc, down, enter) // Restart: counts 3
	a.state.Kills = 4
	press(a, &now, esc, up, enter) // Quit to menu: counts 4
	press(a, &now, enter, enter)   // Play again, Survival
	a.state.Kills = 5
	press(a, &now, key(term.KeyCtrlC, 0)) // counts 5
	d, _ := save.Load(store)
	if d.Progress.TotalKills != 12 {
		t.Errorf("total kills = %d, want 3+4+5 = 12", d.Progress.TotalKills)
	}
}

// Changing a setting saves it right away, render mode included.
func TestSettingChangeSaves(t *testing.T) {
	a, _ := newMenuApp(t)
	store := withStore(t, a)
	now := t0
	press(a, &now, down, down, enter, right) // Settings, render mode ascii -> mixed
	d, problem := save.Load(store)
	if problem != "" || d.Settings == nil {
		t.Fatalf("nothing saved after a setting change (problem %q)", problem)
	}
	if d.Settings.RenderMode != "mixed" || d.Settings.FPS != 30 || d.Settings.Volume != 80 {
		t.Errorf("saved settings %+v, want mixed, 30 FPS, volume 80", *d.Settings)
	}
	ss := a.current.(*settingsScreen)
	if !ss.find("Color threshold") {
		t.Fatal("no Color threshold setting")
	}
	press(a, &now, right) // off -> 4
	d, _ = save.Load(store)
	if bg, thr := d.Settings.MixedBackground, d.Settings.ColorThreshold; bg == nil || *bg != 20 || thr == nil || *thr != 4 {
		t.Errorf("saved mixed background %v and threshold %v, want 20 and 4", bg, thr)
	}
}

// Unlocking a map saves it, and loaded progress opens the maps again.
func TestUnlockSavesAndLoads(t *testing.T) {
	a, _ := newMenuApp(t)
	a.maps = append(a.maps, a.maps[0])
	store := withStore(t, a)
	now := t0
	press(a, &now, enter, enter) // Play, Survival
	a.state.Spawning = false
	a.state.Time = unlockAfter("5:00")
	press(a, &now)
	d, _ := save.Load(store)
	if d.Progress.UnlockedMaps != 2 {
		t.Fatalf("saved unlocked maps = %d, want 2", d.Progress.UnlockedMaps)
	}

	b, _ := newMenuApp(t)
	b.maps = a.maps
	b.useSaved(Saved{Progress: save.Progress{UnlockedMaps: 9}})
	if b.progress.UnlockedMaps != 2 || b.progress.Best == nil {
		t.Errorf("loaded progress: %d maps open (want 2, clamped), best map nil: %v", b.progress.UnlockedMaps, b.progress.Best == nil)
	}
}

func TestApplySettings(t *testing.T) {
	base := Config{FPS: 30, KeyMode: "events", PixelAspect: 1, Textures: true, Volume: 80}
	c, note := ApplySettings(base, save.Settings{RenderMode: "shade", FPS: 60, KeyMode: "events",
		PixelAspect: 1.2, Volume: 40, Sound: true, MinimapAtStart: true})
	if c.Mode != game.ModeShade || c.FPS != 60 || c.PixelAspect != 1.2 || c.Volume != 40 || c.Textures || !c.Sound || note != "" {
		t.Errorf("ApplySettings = %+v, note %q", c, note)
	}
}

// The S3 settings come from the save file when it has them; a file from
// before S3 (nil) keeps the defaults.
func TestApplySettingsS3(t *testing.T) {
	base := Config{FPS: 30, KeyMode: "events", PixelAspect: 1, MixedBackground: 20}
	old := save.Settings{RenderMode: "mixed", FPS: 30, KeyMode: "events", PixelAspect: 1}
	if c, _ := ApplySettings(base, old); c.MixedBackground != 20 || c.ColorThreshold != 0 {
		t.Errorf("pre-S3 save: background %d threshold %d, want 20 and 0", c.MixedBackground, c.ColorThreshold)
	}
	bg, thr := 0, 8
	saved := old
	saved.MixedBackground, saved.ColorThreshold = &bg, &thr // & takes the address: a pointer to the variable
	saved.RenderScale = "half"
	if c, _ := ApplySettings(base, saved); c.MixedBackground != 0 || c.ColorThreshold != 8 || c.RenderScale != "half" {
		t.Errorf("saved 0%%, 8, half: got %d, %d, %q", c.MixedBackground, c.ColorThreshold, c.RenderScale)
	}
	base.RenderScale = "auto"
	if c, _ := ApplySettings(base, old); c.RenderScale != "auto" {
		t.Errorf("pre-V6 save: render scale %q, want auto", c.RenderScale)
	}
}

// The Symbols setting (U1) is saved and loaded; a file from before U1 means
// fancy.
func TestSymbolsSetting(t *testing.T) {
	base := Config{FPS: 30, KeyMode: "events", PixelAspect: 1}
	old := save.Settings{RenderMode: "mixed", FPS: 30, KeyMode: "events", PixelAspect: 1}
	if c, _ := ApplySettings(base, old); c.SimpleSymbols {
		t.Error("a save file without symbols gives the simple set, want fancy")
	}
	old.Symbols = "simple"
	if c, _ := ApplySettings(base, old); !c.SimpleSymbols {
		t.Error("symbols: simple in the save file isn't applied")
	}
}

func TestWrap(t *testing.T) {
	got := wrap("one two three four", 9)
	if strings.Join(got, "|") != "one two|three|four" {
		t.Errorf("wrap = %q", got)
	}
}

// Waves (M15): Play -> Waves starts a waves run; winning shows the win
// menu and saves a waves best apart from the survival best.
func TestWavesRunWinSaves(t *testing.T) {
	a, s := newMenuApp(t)
	store := withStore(t, a)
	now := t0
	press(a, &now, enter, right, enter) // Play, Waves, Start
	if a.state.Play != game.Waves || !a.state.InBreak() {
		t.Fatalf("after choosing Waves: mode %v, break %v; want a waves run in its first break", a.state.Play, a.state.InBreak())
	}
	press(a, &now)
	if !screenHas(s, "WAVE 1/10 IN") {
		t.Error("HUD doesn't count down to wave 1")
	}
	a.state.Wave, a.state.Kills, a.state.Time, a.state.Won = game.NumWaves, 40, 400, true
	press(a, &now)
	press(a, &now) // the menu draws on the next frame
	if _, ok := a.current.(*menuScreen); !ok || !screenHas(s, "YOU WIN") || !screenHas(s, "won in 6:40") {
		t.Fatalf("after winning: screen %T; want the win menu with the time", a.current)
	}
	d, problem := save.Load(store)
	if problem != "" {
		t.Fatalf("Load: %s", problem)
	}
	if w := d.Progress.Waves[""]; w.Wins != 1 || w.Wave != game.NumWaves || w.WinTime != 400 || w.Kills != 40 {
		t.Errorf("saved waves best %+v, want 1 win at wave %d in 400 s with 40 kills", w, game.NumWaves)
	}
	if _, ok := d.Progress.Best[""]; ok {
		t.Error("a waves run changed the survival best")
	}
	press(a, &now, enter) // Play again: waves again, no mode menu
	if _, ok := a.current.(*playingScreen); !ok || a.state.Play != game.Waves {
		t.Errorf("Play again: screen %T, mode %v; want a new waves run", a.current, a.state.Play)
	}
}
