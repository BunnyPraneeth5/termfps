package app

import (
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/save"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// tourMap has rooms, a door, a crate and a red barrel to walk around.
const tourMap = "############\n" +
	"#P....#....#\n" +
	"#..c..D....#\n" +
	"#.....#..B.#\n" +
	"#.....#....#\n" +
	"############\n"

func newTestTour(t *testing.T, seed uint64) *tour {
	t.Helper()
	m, err := world.Parse([]byte(tourMap))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return newTour(m, rand.New(rand.NewPCG(seed, 9)))
}

// The title tour never enters a wall or a blocking prop, gets around (also
// through the door), and never walks faster than its slow speed.
func TestTourStaysInOpenCells(t *testing.T) {
	tr := newTestTour(t, 1)
	const dt = 1.0 / 15
	cells := map[int]bool{}
	for i := 0; i < 15*600; i++ { // ten minutes
		x0, y0 := tr.st.Player.X, tr.st.Player.Y
		tr.update(dt)
		p := tr.st.Player
		if !tr.walkable(int(p.X), int(p.Y)) {
			t.Fatalf("after %d steps the camera is in cell (%d, %d): %q", i, int(p.X), int(p.Y), tr.m.At(int(p.X), int(p.Y)))
		}
		if d := math.Hypot(p.X-x0, p.Y-y0); d > tourSpeed*dt+1e-9 {
			t.Fatalf("moved %.4f cells in one step, more than %.4f", d, tourSpeed*dt)
		}
		cells[int(p.Y)*tr.m.W+int(p.X)] = true
	}
	if len(cells) < 10 {
		t.Errorf("the camera visited %d cells in ten minutes, want it to get around", len(cells))
	}
	east := false
	for c := range cells {
		east = east || c%tr.m.W > 6
	}
	if !east {
		t.Error("the camera never went through the door to the east room")
	}
}

// The tour follows dt: the same time in small or big steps ends in about
// the same place; and the same seed gives the same tour.
func TestTourFollowsDTAndSeed(t *testing.T) {
	a, b, c := newTestTour(t, 5), newTestTour(t, 5), newTestTour(t, 5)
	for i := 0; i < 600; i++ { // 10 s at 60 a second
		a.update(1.0 / 60)
		c.update(1.0 / 60)
	}
	for i := 0; i < 150; i++ { // 10 s at 15 a second
		b.update(1.0 / 15)
	}
	pa, pb, pc := a.st.Player, b.st.Player, c.st.Player
	if pa.X != pc.X || pa.Y != pc.Y || pa.Angle != pc.Angle {
		t.Error("two tours with the same seed and steps differ")
	}
	if d := math.Hypot(pa.X-pb.X, pa.Y-pb.Y); d > 0.5 {
		t.Errorf("after 10 s the camera is %.2f cells apart at 60 and 15 steps a second, want about the same place", d)
	}
	other := newTestTour(t, 6)
	if other.st.Player.Angle == a.st.Player.Angle {
		t.Error("a different seed starts facing the same way")
	}
}

// The tour only moves the camera: no enemies, doors open.
func TestTourCameraOnly(t *testing.T) {
	tr := newTestTour(t, 1)
	for i := 0; i < 300; i++ {
		tr.update(0.1)
	}
	if len(tr.st.Enemies) != 0 || len(tr.st.Pickups) != 0 || tr.st.Spawning {
		t.Errorf("the tour has %d enemies and %d pickups (spawning %v), want none", len(tr.st.Enemies), len(tr.st.Pickups), tr.st.Spawning)
	}
	if tr.m.Blocked(6, 2) {
		t.Error("the door is closed on the tour's map")
	}
}

// With the title background on, the tour shows behind the title, the
// Settings and the Guide opened from it, on an open map; off, it doesn't.
func TestTitleBackground(t *testing.T) {
	a, s := newMenuApp(t)
	a.maps = append(a.maps, a.maps[0].Clone()) // locked
	now := t0
	press(a, &now)
	if a.tour != nil {
		t.Fatal("the tour runs with the title background off")
	}
	a.cfg.TitleBackground = true
	press(a, &now)
	if a.tour == nil {
		t.Fatal("no tour behind the title with the setting on")
	}
	if str, _ := s.Get(0, 0); str == " " {
		t.Error("the corner of the title screen is blank, want the 3D picture")
	}
	if !screenHas(s, "PLAY") || !screenHas(s, Version) {
		t.Error("the menu or the version is hidden behind the tour")
	}
	x := a.tour.st.Player
	for range 90 {
		press(a, &now)
	}
	press(a, &now, down, down, enter) // Settings
	for range 90 {
		press(a, &now)
	}
	if y := a.tour.st.Player; x.X == y.X && x.Y == y.Y && x.Angle == y.Angle {
		t.Error("the tour's camera didn't move in six seconds")
	}
	if str, _ := s.Get(0, 0); str == " " {
		t.Error("behind Settings opened from the title the picture is gone")
	}
}

// The title fits 80x24: the logo in spaced block letters, a tagline from
// the list, the menu, a description, the best run and the version.
func TestTitleLook(t *testing.T) {
	a, s := newMenuApp(t)
	a.progress.Best[""] = save.Best{Kills: 142, Time: 751, Tier: 5}
	now := t0
	press(a, &now)
	if !slices.Contains(taglines, a.tagline) || !screenHas(s, a.tagline) {
		t.Errorf("tagline %q isn't from the list or isn't shown", a.tagline)
	}
	blocks := 0
	for y := 1; y <= ui.FontRows; y++ {
		blocks += strings.Count(s.Row(y), "█")
	}
	if blocks < 80 {
		t.Errorf("%d logo blocks on rows 1-5, want the TERM FPS logo", blocks)
	}
	for _, want := range []string{"▶ PLAY", "GUIDE", "QUIT", "╔", "› Pick a mode and a map, then go. 1 map.", "12:31 · 142 kills · tier 5", " ⏎  choose"} {
		if !screenHas(s, want) {
			t.Errorf("title doesn't show %q", want)
		}
	}
	if row := s.Row(23); !strings.HasSuffix(strings.TrimRight(row, " "), Version) || !strings.Contains(Version, "As*****") {
		t.Errorf("bottom row %q, want the version %q at the right", row, Version)
	}
	press(a, &now, down)
	press(a, &now)
	if !screenHas(s, "▶ GUIDE") || a.lastCue != audio.MenuTick {
		t.Error("Down didn't move to GUIDE with a tick")
	}
}

// The logo shimmers and the tagline pulses only with animations on.
func TestTitleAnimations(t *testing.T) {
	bright := func(a *App, s *term.Fake) (logo int, tagline term.Color) {
		for y := 1; y <= ui.FontRows; y++ {
			for x := 0; x < 80; x++ {
				if str, st := s.Get(x, y); str == "█" && st.Fg == ui.ColBright {
					logo++
				}
			}
		}
		_, st := s.Get(40, 7)
		return logo, st.Fg
	}
	a, s := newMenuApp(t)
	now := t0
	for range 30 {
		press(a, &now)
	}
	if n, c := bright(a, s); n != 0 || c != ui.ColYellow {
		t.Errorf("animations off: %d bright logo cells, tagline %v; want a still logo and a plain yellow tagline", n, c)
	}
	a.cfg.Animations = true
	seen, dimmed := 0, false
	for range 30 * 7 { // more than one shimmer period
		press(a, &now)
		n, c := bright(a, s)
		seen += n
		dimmed = dimmed || c != ui.ColYellow
	}
	if seen == 0 || !dimmed {
		t.Errorf("animations on: %d bright logo cells in 7 s, tagline dimmed %v; want a shimmer and a pulse", seen, dimmed)
	}
}

// The Play screen picks the mode and the map on one screen and starts.
func TestPlayScreen(t *testing.T) {
	a, s := newMenuApp(t)
	second, err := world.Parse([]byte("; name: Second\n" + testMap))
	if err != nil {
		t.Fatal(err)
	}
	a.maps = append(a.maps, second)
	a.cfg.UnlockRule = "open"
	a.progress.Waves["Second"] = save.WavesBest{Wave: 10, Wins: 1, WinTime: 432}
	now := t0
	press(a, &now, enter)
	press(a, &now)
	for _, want := range []string{"PLAY", "MODE", " SURVIVAL ", " WAVES ", "endless · tier up every 20 kills or 3:00", "MAP", "Second", "10x6",
		"+ your maps (maps folder)", "explored 0% · 0 doors · 0 red barrels", "no runs here yet", "▶ START", " esc  back"} {
		if !screenHas(s, want) {
			t.Errorf("Play screen doesn't show %q", want)
		}
	}
	press(a, &now, right, down)
	press(a, &now)
	for _, want := range []string{"10 waves with breaks, clear them all to win", " SECOND ", "waves won in 7:12"} {
		if !screenHas(s, want) {
			t.Errorf("Play screen on Waves and the second map doesn't show %q", want)
		}
	}
	// A click on the first map's row picks it; a click on START starts.
	row, col := -1, -1
	for y := 0; y < 24; y++ {
		if i := strings.Index(s.Row(y), "START"); i >= 0 {
			row, col = y, len([]rune(s.Row(y)[:i]))
		}
	}
	click := func(x, y int) {
		now = now.Add(time.Second / 30)
		a.step(now, []term.Event{term.Mouse{X: x, Y: y, Buttons: term.ButtonLeft}, term.Mouse{X: x, Y: y}})
	}
	click(10, 7) // the first map's row
	if p := a.current.(*playScreen); p.sel != 0 {
		t.Fatalf("clicking the first map: selected %d", p.sel)
	}
	click(col, row)
	if _, ok := a.current.(*playingScreen); !ok || a.state.Play != game.Waves || a.mapIndex != 0 || a.lastCue != audio.MenuClick {
		t.Errorf("clicking START: screen %T, mode %v, map %d; want a waves run on the first map, with a click", a.current, a.state.Play, a.mapIndex)
	}
	// The mode and the map are remembered; Esc goes back to the title.
	quitToMenu(a, &now)
	press(a, &now, enter)
	if p := a.current.(*playScreen); p.mode != game.Waves {
		t.Error("the Play screen forgot the mode")
	}
	press(a, &now, left, esc)
	if !onTitle(a) || a.lastCue != audio.MenuBack {
		t.Error("Esc on the Play screen didn't go back to the title with the back sound")
	}
}

// A long map list scrolls, keeping the selection in view at 80x24.
func TestPlayScreenScrolls(t *testing.T) {
	a, s := newMenuApp(t)
	a.cfg.UnlockRule = "open"
	for i := 0; i < 14; i++ {
		m := a.maps[0].Clone()
		m.Name = "Map " + string(rune('A'+i))
		a.maps = append(a.maps, m)
	}
	now := t0
	press(a, &now, enter, up) // wraps to the last map
	press(a, &now)
	if !screenHas(s, "▶ Map N") || screenHas(s, "Map A ") {
		t.Error("the list didn't scroll to the last map")
	}
}

// The menus make their sounds: tick, click, back.
func TestMenuSounds(t *testing.T) {
	a, _ := newMenuApp(t)
	now := t0
	press(a, &now, down, down)
	if a.lastCue != audio.MenuTick || a.cues != 2 {
		t.Errorf("two Downs: last sound %d, %d sounds; want two ticks", a.lastCue, a.cues)
	}
	press(a, &now, enter) // Settings
	if a.lastCue != audio.MenuClick {
		t.Errorf("Enter: sound %d, want the click", a.lastCue)
	}
	press(a, &now, down)
	if a.lastCue != audio.MenuTick {
		t.Errorf("Down in a list menu: sound %d, want the tick", a.lastCue)
	}
	press(a, &now, esc)
	if a.lastCue != audio.MenuBack {
		t.Errorf("Esc: sound %d, want the back sound", a.lastCue)
	}
}
