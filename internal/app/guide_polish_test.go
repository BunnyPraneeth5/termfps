package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/save"
	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// The panel is centered and grows with the window up to guideMaxW; the 3D
// view grows with it (M18c).
func TestGuidePanelSize(t *testing.T) {
	small, big := guidePanel(80, 24), guidePanel(200, 60)
	if small.w != 78 || small.x != 1 || small.h != 22 {
		t.Errorf("panel at 80x24 = %+v, want 78 wide at x 1, 22 tall", small)
	}
	if big.w != guideMaxW || big.x != (200-guideMaxW)/2 || big.h != guideMaxH {
		t.Errorf("panel at 200x60 = %+v, want %d wide, centered, %d tall", big, guideMaxW, guideMaxH)
	}
	c1, _ := viewSize(rect{w: small.w - 4, h: small.h - 4})
	c2, r2 := viewSize(rect{w: big.w - 4, h: big.h - 4})
	if c1 < viewMinCols || c2 <= c1 || c2 > viewMaxCols || r2 < 10 {
		t.Errorf("view box %d cols at 80x24, %dx%d in a big window; want it to grow, %d-%d cols", c1, c2, r2, viewMinCols, viewMaxCols)
	}
}

// What a run sees of a map is added to what earlier runs saw, and saved
// (M18c).
func TestExploredMerges(t *testing.T) {
	a, _ := newMenuApp(t)
	store := withStore(t, a)
	now := t0
	run := func(cells ...int) {
		press(a, &now, enter, enter) // Play, Survival
		for i := range a.state.Seen {
			a.state.Seen[i] = slices.Contains(cells, i)
		}
		press(a, &now, esc, up, enter) // Pause -> Quit to menu: the run ends
	}
	run(11, 12)
	run(12, 13)
	seen, played := a.explored(0)
	if !played || !seen[11] || !seen[12] || !seen[13] || seen[14] {
		t.Fatalf("after two runs: played %v, cells 11-14 %v; want 11, 12, 13 seen", played, seen[11:15])
	}
	d, problem := save.Load(store)
	if problem != "" {
		t.Fatalf("Load: %s", problem)
	}
	if e := d.Progress.Explored[a.maps[0].Name]; e.W != a.maps[0].W || e.Cells == "" {
		t.Errorf("saved explored %+v, want this map's cells", e)
	}
}

// Meeting an enemy and finding a gun are remembered (M18c).
func TestRecordFinds(t *testing.T) {
	a, _ := newMenuApp(t)
	a.newRun()
	if a.met(game.KindRunner) || a.found(game.Shotgun) || !a.found(game.Pistol) {
		t.Fatal("a new player has met a runner or found the shotgun, or lacks the pistol")
	}
	a.state.Events = append(a.state.Events, game.EventEnemyNoticed, game.EventGunTaken)
	a.state.Origins = append(a.state.Origins, game.Origin{Event: game.EventEnemyNoticed, Kind: game.KindRunner})
	a.state.Player.Gun = game.Shotgun
	a.recordFinds()
	a.recordFinds() // twice: still listed once
	if !a.met(game.KindRunner) || !a.found(game.Shotgun) || len(a.progress.Met) != 1 || len(a.progress.Found) != 1 {
		t.Errorf("met %v, found %v; want the runner and the shotgun, once each", a.progress.Met, a.progress.Found)
	}
}

// Unmet enemies and unfound guns are dark "???" pages until met or found.
func TestSilhouettes(t *testing.T) {
	a, _ := newMenuApp(t)
	enemies := section(t, a, "Enemies").entries
	if enemies[0].name != "???" || enemies[0].view.kind == viewNone || strings.Contains(textOf(enemies[0]), "HP") {
		t.Errorf("unmet zombie page %q: %q; want ??? with a view and no numbers", enemies[0].name, textOf(enemies[0]))
	}
	guns := section(t, a, "Weapons").entries
	if guns[0].name != "Pistol" || guns[1].name != "???" {
		t.Errorf("weapon pages %q, %q; want Pistol and ??? for the shotgun", guns[0].name, guns[1].name)
	}
	a.progress.Met = []string{"zombie"}
	a.progress.Found = []string{"shotgun"}
	if n := section(t, a, "Enemies").entries[0].name; n != "Zombie" {
		t.Errorf("met zombie page is %q, want Zombie", n)
	}
	if n := section(t, a, "Weapons").entries[1].name; n != "Shotgun" {
		t.Errorf("found shotgun page is %q, want Shotgun", n)
	}
}

// Every key on the controls page has a group and a description, and is on
// the keyboard or the mouse; Up/Down step through them (M18c).
func TestControlsKeys(t *testing.T) {
	onKeyboard := map[string]bool{"LMB": true, "RMB": true, "Wheel": true}
	for _, row := range keyboardRows {
		for _, k := range row {
			onKeyboard[k.cap] = true
		}
	}
	for _, k := range guideKeys {
		if !onKeyboard[k.cap] || k.desc == "" || int(k.group) >= len(groupNames) {
			t.Errorf("key %q: on the picture %v, description %q, group %d", k.cap, onKeyboard[k.cap], k.desc, k.group)
		}
	}
	a, s := newMenuApp(t)
	sec := section(t, a, "Controls and HUD")
	g := &guidePage{back: a.current, section: sec, shown: -1}
	g.key(a, term.Key{Code: term.KeyUp})
	g.tick(a, t0, 0)
	if g.keySel != len(guideKeys)-1 || !screenHas(s, guideKeys[len(guideKeys)-1].desc) || !screenHas(s, "pick a key") {
		t.Errorf("Up from the first key: selected %d, want the last one, with its description shown", g.keySel)
	}
}

// The HUD page points out the parts of an example HUD line.
func TestHUDPageLabels(t *testing.T) {
	a, s := newMenuApp(t)
	g := &guidePage{back: a.current, section: section(t, a, "Controls and HUD"), index: 1, shown: -1}
	g.tick(a, t0, 0)
	for _, want := range []string{"HP  76 ARMOR 50", "└ health, armor", "└ kills", "└ gun: in it / spare", "└ stamina"} {
		if !screenHas(s, want) {
			t.Errorf("HUD page doesn't show %q", want)
		}
	}
}

// A map page draws the picture with only the explored part, and says how
// much; a locked one is a dark outline with no numbers.
func TestMapPage(t *testing.T) {
	a, s := newMenuApp(t)
	a.maps = append(a.maps, a.maps[0])
	m := a.maps[0]
	seen := make([]bool, m.W*m.H)
	for i := range seen {
		seen[i] = true
	}
	a.progress.Explored[m.Name] = save.Explored{W: m.W, H: m.H, Cells: save.PackCells(seen)}
	maps := section(t, a, "Maps").entries
	if maps[0].kind != pageMap || !strings.Contains(textOf(maps[0]), "Explored: 100%") {
		t.Errorf("explored map page: kind %d, %q; want a map page saying 100%%", maps[0].kind, textOf(maps[0]))
	}
	g := &guidePage{back: a.current, section: section(t, a, "Maps"), index: 1, shown: -1}
	g.tick(a, t0, 0)
	if !screenHas(s, "Locked.") || g.preview != nil {
		t.Error("the locked map page doesn't say it's locked, or turns round inside it")
	}
}
