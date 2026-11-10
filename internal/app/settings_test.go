package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/save"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

var tab = key(term.KeyTab, 0)

// openSettings opens the Settings from the title screen.
func openSettings(t *testing.T, a *App, now *time.Time) *settingsScreen {
	t.Helper()
	press(a, now, down, down, enter)
	ss, ok := a.current.(*settingsScreen)
	if !ok {
		t.Fatalf("the title's third item opened %T, want the Settings", a.current)
	}
	return ss
}

// set selects the setting called label and presses Right n times.
func set(t *testing.T, a *App, now *time.Time, label string, n int) {
	t.Helper()
	if !a.current.(*settingsScreen).find(label) {
		t.Fatalf("no setting called %q", label)
	}
	for range n {
		press(a, now, right)
	}
}

// Settings change the app right away, and Esc returns to where they were
// opened from.
func TestSettingsApplyRightAway(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	openSettings(t, a, &now)
	press(a, &now)
	if !screenHas(s, "SETTINGS") || !screenHas(s, "changes apply at once and are saved") {
		t.Fatal("settings screen not shown")
	}
	// Render mode starts at ascii (256-color screen); Right goes to mixed.
	press(a, &now, right)
	if a.state.Mode != game.ModeMixed {
		t.Errorf("mode = %v after Right, want mixed", a.state.Mode)
	}
	set(t, a, &now, "Mixed background", 1) // 20% -> 30%
	if a.cfg.MixedBackground != 30 || a.frame.MixedBackground != 0.3 {
		t.Errorf("mixed background = %d%% (frame %v), want 30%%", a.cfg.MixedBackground, a.frame.MixedBackground)
	}
	set(t, a, &now, "Color threshold", 1) // off -> 4
	if a.cfg.ColorThreshold != 4 || a.frame.ColorThreshold != 4 {
		t.Errorf("color threshold = %d (frame %d), want 4", a.cfg.ColorThreshold, a.frame.ColorThreshold)
	}
	set(t, a, &now, "Render scale", 1) // auto -> full
	if a.cfg.RenderScale != "full" || a.frame.Scale != 1 {
		t.Errorf("render scale = %q (frame scale %d), want full, 1", a.cfg.RenderScale, a.frame.Scale)
	}
	press(a, &now, right) // full -> half: the view is drawn at half size
	if a.cfg.RenderScale != "half" || a.frame.Scale != 2 || a.frame.W != 40 {
		t.Errorf("render scale = %q (frame scale %d, width %d), want half, 2, 40", a.cfg.RenderScale, a.frame.Scale, a.frame.W)
	}
	set(t, a, &now, "FPS", 1) // 30 -> 60
	if a.cfg.FPS != 60 {
		t.Errorf("FPS = %d, want 60", a.cfg.FPS)
	}
	set(t, a, &now, "Minimap at start", 0)
	press(a, &now, enter) // on -> off: Enter moves to the next value too
	if a.cfg.MinimapAtStart {
		t.Error("minimap at start still on")
	}
	set(t, a, &now, "Wall textures", 1)
	if a.cfg.Textures || a.frame.Textures {
		t.Error("textures still on")
	}
	set(t, a, &now, "Volume", 1) // 80% -> 90%
	if a.cfg.Volume != 90 {
		t.Errorf("volume = %d, want 90", a.cfg.Volume)
	}
	set(t, a, &now, "Pixel aspect", 1) // 1 -> 1.1
	if a.frame.PixelAspect != 1.1 {
		t.Errorf("pixel aspect = %v, want 1.1", a.frame.PixelAspect)
	}
	set(t, a, &now, "Title background", 1)
	set(t, a, &now, "Animations", 1)
	set(t, a, &now, "Minimap fog of war", 1)
	set(t, a, &now, "Map unlocks", 2)
	set(t, a, &now, "Sound", 1)
	if !a.cfg.TitleBackground || !a.cfg.Animations || !a.cfg.MinimapFog || a.cfg.UnlockRule != "open" || a.cfg.Sound {
		t.Errorf("after switching them: title background %v, animations %v, fog %v, unlocks %q, sound %v",
			a.cfg.TitleBackground, a.cfg.Animations, a.cfg.MinimapFog, a.cfg.UnlockRule, a.cfg.Sound)
	}
	press(a, &now, esc)
	if !onTitle(a) {
		t.Error("Esc in settings didn't go back to the title screen")
	}
	press(a, &now, up, up, enter, enter) // Play (from Settings: up twice), Start
	if a.state.ShowMinimap {
		t.Error("new run shows the minimap; setting says off")
	}
}

// Every setting is on one of the four tabs, and every tab fits 80x24 with
// a help line per setting.
func TestSettingsTabs(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	ss := openSettings(t, a, &now)
	want := map[int][]string{
		tabVideo:    {"Render mode", "Render scale", "FPS", "Wall textures", "Mixed background", "Symbols", "Title background", "Animations"},
		tabAudio:    {"Sound", "Volume"},
		tabControls: {"Key mode", "Pixel aspect", "Minimap at start"},
		tabGameplay: {"Map unlocks", "Minimap fog of war", "Color threshold"},
	}
	total := 0
	for tb := 0; tb < numTabs; tb++ {
		ss.tab = tb
		press(a, &now)
		rows := ss.rows()
		total += len(rows)
		if len(rows) != len(want[tb]) {
			t.Errorf("tab %s has %d settings, want %d", tabNames[tb], len(rows), len(want[tb]))
		}
		for _, label := range want[tb] {
			if !screenHas(s, label) {
				t.Errorf("tab %s doesn't show %q", tabNames[tb], label)
			}
		}
		for _, i := range rows {
			if it := ss.items[i]; it.help == "" || !screenHas(s, it.help) {
				t.Errorf("%s: help line %q missing or not on screen at 80x24", it.label, it.help)
			}
		}
		if !screenHas(s, " esc  back") || !screenHas(s, " tab  next tab") {
			t.Errorf("tab %s: the key help is missing", tabNames[tb])
		}
	}
	if total != len(ss.items) {
		t.Errorf("%d settings on the tabs, %d in all", total, len(ss.items))
	}
	if !screenHas(s, "╭") || !strings.Contains(s.Row(4), "GAMEPLAY") {
		t.Error("the panel border or the tab row is missing")
	}
}

// Tab switches tabs anywhere; Left/Right switch them on the tab row, and
// change the value on a setting. Values draw as arrows, switches and bars.
func TestSettingsKeysAndValues(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	ss := openSettings(t, a, &now)
	press(a, &now)
	for _, want := range []string{"▶ Render mode", "◀ ascii ▶", "[■ on ]", "▮▮▯▯▯▯▯▯▯▯ 20%"} {
		if !screenHas(s, want) {
			t.Errorf("video tab doesn't show %q", want)
		}
	}
	press(a, &now, tab)
	if ss.tab != tabAudio || ss.sel != 0 {
		t.Fatalf("Tab: tab %d row %d, want the audio tab, first row", ss.tab, ss.sel)
	}
	press(a, &now, up) // onto the tab row
	press(a, &now, right)
	if ss.sel != -1 || ss.tab != tabControls {
		t.Errorf("Up then Right: row %d tab %d, want the tab row and the controls tab", ss.sel, ss.tab)
	}
	press(a, &now, left, left, left) // wraps round to gameplay
	if ss.tab != tabGameplay {
		t.Errorf("Left three times from controls: tab %d, want gameplay (wrapped)", ss.tab)
	}
	press(a, &now, down, left) // Map unlocks: Left wraps to "all open"
	if a.cfg.UnlockRule != "open" || a.lastCue != audio.MenuClick {
		t.Errorf("Left on Map unlocks: rule %q, want open, with a click", a.cfg.UnlockRule)
	}
	// A click on a tab opens it; a click on a setting moves it on.
	press(a, &now)
	click := func(x, y int) {
		now = now.Add(time.Second / 30)
		a.step(now, []term.Event{term.Mouse{X: x, Y: y, Buttons: term.ButtonLeft}, term.Mouse{X: x, Y: y}})
	}
	click(strings.Index(s.Row(4), "AUDIO")-len("▶")+1, 4) // Index counts bytes: the marker is one cell
	if ss.tab != tabAudio {
		t.Fatalf("clicking AUDIO: tab %d", ss.tab)
	}
	click(10, 6) // Sound
	if a.cfg.Sound {
		t.Error("clicking Sound didn't switch it off")
	}
}

// The Symbols setting (U1) switches the whole look at once and is saved.
func TestSymbolsSettingSwitches(t *testing.T) {
	a, s := newMenuApp(t)
	defer ui.UseSimple(false)
	now := t0
	openSettings(t, a, &now)
	set(t, a, &now, "Symbols", 1)
	press(a, &now)
	if !a.cfg.SimpleSymbols || !ui.Sym().Simple || a.settings().Symbols != "simple" || !screenHas(s, "< simple >") {
		t.Errorf("after switching: config %v, set simple %v, saved as %q; want simple everywhere, drawn with < >",
			a.cfg.SimpleSymbols, ui.Sym().Simple, a.settings().Symbols)
	}
	for _, r := range s.Text() {
		if r > 126 && r != '█' && r != '▀' && r != '▄' { // the logo's blocks are in every console font
			t.Fatalf("the simple set drew %q on the settings screen", r)
		}
	}
}

// The pause card shows the run so far and what comes next, and its items
// work by key and by click.
func TestPauseCard(t *testing.T) {
	a, s := newTestApp(t)
	now := t0
	a.state.Kills, a.state.Time = 3, 127
	press(a, &now, esc)
	press(a, &now)
	for _, want := range []string{"II PAUSED", " · survival", "TIME 2:07", "KILLS 3", "TIER 1", "HP 100", "or 17 kills",
		"▶ RESUME", "RESTART", "GUIDE", "SETTINGS", "QUIT TO MENU", "this run still counts", " esc  resume"} {
		if !screenHas(s, want) {
			t.Errorf("pause card doesn't show %q", want)
		}
	}
	row := -1
	for y := 0; y < 24; y++ {
		if strings.Contains(s.Row(y), "GUIDE") {
			row = y
		}
	}
	a.step(now.Add(time.Second/30), []term.Event{term.Mouse{X: 40, Y: row, Buttons: term.ButtonLeft}, term.Mouse{X: 40, Y: row}})
	if m, ok := a.current.(*menuScreen); !ok || m.menu.Title != "GUIDE" {
		t.Errorf("clicking GUIDE on the pause card opened %T", a.current)
	}
}

// In waves mode the pause card shows the wave instead of the next tier.
func TestPauseCardWaves(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	press(a, &now, enter, right, enter) // Play, Waves, Start
	press(a, &now, esc)
	press(a, &now)
	if !screenHas(s, "wave 1 of 10 starts in 0:10") || !screenHas(s, "WAVE 1/10") {
		t.Error("pause card in the first break doesn't count down to wave 1")
	}
	if k, sec := a.state.NextTier(); k != game.TierKills || sec <= 0 {
		t.Errorf("NextTier at the start = %d kills, %.0f s", k, sec)
	}
}

// A failed save raises an error box with the real error over the current
// screen; it takes the keys until OK, and the title keeps a warning line.
// A problem from loading shows the same way.
func TestErrorBox(t *testing.T) {
	a, s := newMenuApp(t)
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// The save folder would be a file, so every write fails.
	store := save.Store{Path: filepath.Join(blocker, "save.json")}
	a.useSaved(Saved{Store: &store, Message: "The save file was damaged (test). Kept it as save.json.bad, using defaults.",
		MapProblem: "Skipped maps: bad.txt: row 1."})
	now := t0
	press(a, &now)
	if len(a.boxes) != 2 || !screenHas(s, "! PROBLEM WITH THE SAVE FILE") || !screenHas(s, "save.json.bad") || !screenHas(s, "  OK  ") {
		t.Fatalf("%d error boxes; want the save file problem shown first", len(a.boxes))
	}
	press(a, &now, down) // the box takes the keys
	if a.current.(*titleScreen).sel != 0 {
		t.Error("a key went to the title screen while the error box was up")
	}
	press(a, &now, enter)
	press(a, &now)
	if !screenHas(s, "! COULDN'T LOAD SOME OF YOUR MAPS") || !screenHas(s, "bad.txt") || !screenHas(s, "The other maps work as usual.") {
		t.Fatal("the map problem isn't shown after the first box")
	}
	a.step(now.Add(time.Second/30), []term.Event{term.Mouse{X: 1, Y: 1, Buttons: term.ButtonLeft}, term.Mouse{X: 1, Y: 1}})
	if len(a.boxes) != 0 {
		t.Fatal("a click didn't close the error box")
	}
	now = now.Add(time.Second)
	a.persist()
	a.persist() // a second failure: no second box
	press(a, &now)
	if len(a.boxes) != 1 || !screenHas(s, "! COULDN'T SAVE YOUR PROGRESS") || !screenHas(s, "it is not saved") {
		t.Fatalf("%d boxes after two failed saves; want one, saying it isn't saved", len(a.boxes))
	}
	if _, st := s.Get((80-boxW)/2, 12); st.Fg != ui.ColRed {
		t.Errorf("the error box's border is %+v, want red", st)
	}
	press(a, &now, esc)
	press(a, &now)
	if len(a.boxes) != 0 || !screenHas(s, "Not saved: ") {
		t.Error("after OK the title doesn't keep the warning line")
	}
}

// A save that fails in the middle of a game pauses it behind the box.
func TestErrorBoxPausesGame(t *testing.T) {
	a, _ := newTestApp(t)
	a.raise(errorBox{title: "TEST", detail: errors.New("boom").Error()})
	if _, ok := a.current.(*pauseScreen); !ok {
		t.Errorf("an error box during the game left %T showing, want the pause card", a.current)
	}
}
