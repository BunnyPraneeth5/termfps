package app

import (
	"os"
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
	press(a, &now, enter) // Play
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
	press(a, &now, enter)
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
	press(a, &now, enter)
	a.state.Kills = 3
	press(a, &now, esc, down, down, enter) // Restart: counts 3
	a.state.Kills = 4
	press(a, &now, esc, up, enter) // Quit to menu: counts 4
	press(a, &now, enter)          // Play again
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
}

// Unlocking a map saves it, and loaded progress opens the maps again.
func TestUnlockSavesAndLoads(t *testing.T) {
	a, _ := newMenuApp(t)
	a.maps = append(a.maps, a.maps[0])
	store := withStore(t, a)
	now := t0
	press(a, &now, enter)
	a.state.Spawning = false
	a.state.Time = unlockAfter
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

// A load problem and a failed save both show in the title menu.
func TestSaveProblemsShowInMenu(t *testing.T) {
	a, s := newMenuApp(t)
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// The save folder would be a file, so every write fails.
	store := save.Store{Path: filepath.Join(blocker, "save.json")}
	a.useSaved(Saved{Store: &store, Message: "The save file was damaged (test). Kept it as save.json.bad, using defaults."})
	now := t0
	a.persist()
	press(a, &now)
	if !screenHas(s, "save.json.bad") || !screenHas(s, "Couldn't save") {
		t.Error("title menu doesn't show the load problem and the save error")
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

func TestWrap(t *testing.T) {
	got := wrap("one two three four", 9)
	if strings.Join(got, "|") != "one two|three|four" {
		t.Errorf("wrap = %q", got)
	}
}
