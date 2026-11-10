package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// picturePages returns every picture page of the guide with its section.
func picturePages(a *App) (pages []guideEntry, secs []guideSection, idx []int) {
	for _, sec := range a.guideSections() {
		for i, e := range sec.entries {
			if e.kind == pagePicture {
				pages, secs, idx = append(pages, e), append(secs, sec), append(idx, i)
			}
		}
	}
	return pages, secs, idx
}

// The six text pages are pictures now (U6). Each shows the lines with the
// game's numbers, stays inside the guide's content area at 80x24 (nothing
// on the panel's border or beyond it), and works in both symbol sets.
func TestGuidePictures(t *testing.T) {
	defer ui.UseSimple(false)
	for _, simple := range []bool{false, true} {
		a, s := newMenuApp(t)
		a.cfg.SimpleSymbols = simple
		ui.UseSimple(simple)
		pages, secs, idx := picturePages(a)
		var names []string
		for _, e := range pages {
			names = append(names, e.name)
		}
		if got := strings.Join(names, ", "); got != "Tiers, Enemies learn, Waves mode, Health and stamina, Settings, Tips" {
			t.Fatalf("picture pages: %s", got)
		}
		panel := guidePanel(80, 24)
		area := guideArea(panel)
		for k, e := range pages {
			g := &guidePage{back: a.current, section: secs[k], index: idx[k], shown: -1}
			g.tick(a, t0, 0)
			for _, want := range e.must() {
				if !screenHas(s, want) {
					t.Errorf("simple %v, %s: the page doesn't show %q", simple, e.name, want)
				}
			}
			// The panel's right border is whole, and the rows under the
			// content area hold only the footer.
			border := ui.Sym().Boxes[ui.BoxRound][5]
			for y := 1; y < 23; y++ {
				if str, _ := s.Get(panel.x+panel.w-1, y); str != border {
					t.Errorf("simple %v, %s: row %d of the right border is %q: the picture is too wide", simple, e.name, y, str)
				}
				if str, _ := s.Get(panel.x+panel.w-2, y); str != " " {
					t.Errorf("simple %v, %s: row %d has %q in the panel's padding", simple, e.name, y, str)
				}
			}
			if txt := strings.Trim(s.Row(area.y+area.h), " "+border); txt != "" {
				t.Errorf("simple %v, %s: the row under the content area has %q: the picture is too tall", simple, e.name, txt)
			}
			if simple {
				for y := area.y; y < area.y+area.h; y++ {
					for x := area.x; x < area.x+area.w; x++ {
						str, _ := s.Get(x, y)
						// The settings page shows the fancy set on purpose, and its
						// 3D views use block and shade characters.
						if r := []rune(str)[0]; r > 126 && e.name != "Settings" && r != '█' {
							t.Fatalf("simple set, %s: %q at (%d, %d)", e.name, str, x, y)
						}
					}
				}
			}
		}
	}
}

// The pictures' numbers come from the game code.
func TestGuidePictureNumbers(t *testing.T) {
	a, s := newMenuApp(t)
	pages, secs, idx := picturePages(a)
	show := func(name string) {
		for k, e := range pages {
			if e.name == name {
				(&guidePage{back: a.current, section: secs[k], index: idx[k], shown: -1}).tick(a, t0, 0)
			}
		}
	}
	show("Tiers")
	speed, every, alive := tierLimits()
	if speed != 1.5 || every != 2 || alive != 14 {
		t.Errorf("tier limits %v, %v, %v; the balancing changed (fine): update this test", speed, every, alive)
	}
	for tier := 1; tier <= tiersShown; tier++ {
		st := game.TierFor(tier)
		for _, want := range []string{"x" + num(st.HPMul), "every " + num(st.SpawnEvery) + " s", strings.Repeat("●", st.MaxAlive)} {
			if !strings.Contains(s.Row(guideArea(guidePanel(80, 24)).y+3+tier), want) {
				t.Errorf("tier %d row doesn't show %q", tier, want)
			}
		}
	}
	for _, want := range []string{"T1 zombie, shooter", "T2 runner", "T3 tank", "T4 exploder", fmt.Sprintf("7/%d kills", game.TierKills), "1:30/3:00"} {
		if !screenHas(s, want) {
			t.Errorf("tiers page doesn't show %q", want)
		}
	}
	show("Waves mode")
	for n := 1; n <= game.NumWaves; n++ {
		if !screenHas(s, fmt.Sprintf("W%d", n)) || !screenHas(s, fmt.Sprint(game.WaveSize(n))) {
			t.Errorf("waves chart has no bar for wave %d with its %d enemies", n, game.WaveSize(n))
		}
	}
	if !screenHas(s, "tier 5") || !screenHas(s, "WAVE 10") {
		t.Error("waves page: the tier legend or the timeline is missing")
	}
	show("Health and stamina")
	for _, want := range []string{"HEALTH", "ARMOR", "STAMINA", "↑ 75: above it, medkits only", "↑ 30%"} {
		if !screenHas(s, want) {
			t.Errorf("health page doesn't show %q", want)
		}
	}
	show("Enemies learn")
	for _, want := range []string{"TIER 1", "straight at you", "shooters hide", "zombies surround", "DON'T CAMP", "●", "→"} {
		if !screenHas(s, want) {
			t.Errorf("enemies learn page doesn't show %q", want)
		}
	}
	show("Settings")
	for _, want := range []string{"Render mode: the same wall four ways", "Symbols", "HP [###--] >", "Volume", "▮▮▮▮▮▮▮▮▯▯ 80%"} {
		if !screenHas(s, want) {
			t.Errorf("settings page doesn't show %q", want)
		}
	}
	show("Tips")
	for _, tp := range guideTips {
		if !screenHas(s, strings.Fields(tp.text)[0]+" "+strings.Fields(tp.text)[1]) {
			t.Errorf("tips page doesn't show the tip starting %q", tp.text[:20])
		}
	}
}

// The small pictures are the sizes their boxes expect.
func TestGuideArtSizes(t *testing.T) {
	for i, art := range learnArt {
		for j, row := range art {
			if len(row) != 15 {
				t.Errorf("learn picture %d row %d is %d wide, want 15", i, j, len(row))
			}
		}
	}
	for i, c := range learnCaptions() {
		if len(c[0]) > 16 || len(c[1]) > 16 {
			t.Errorf("caption %d (%q, %q) is wider than its box", i, c[0], c[1])
		}
	}
	rows := 0
	for _, tp := range guideTips {
		if len(tp.art) > 5 {
			t.Errorf("tip picture %q is wider than 5", tp.art)
		}
		rows += max(len(wrap(tp.text, pictureW-9)), 2)
	}
	if len(guideTips) != 7 || rows > pictureH {
		t.Errorf("%d tips taking %d rows; want 7 in at most %d", len(guideTips), rows, pictureH)
	}
}

// The settings page shows the same wall in the four render modes: four
// views side by side that differ in how they're drawn.
func TestSettingsPictureModes(t *testing.T) {
	a, s := newMenuApp(t)
	s.SetTruecolor(true)
	pages, secs, idx := picturePages(a)
	for k, e := range pages {
		if e.name == "Settings" {
			(&guidePage{back: a.current, section: secs[k], index: idx[k], shown: -1}).tick(a, t0, 0)
		}
	}
	area := guideArea(guidePanel(80, 24))
	cell := func(view int) (string, term.Style) { return s.Get(area.x+view*(modeViewW+3)+3, area.y+2) }
	hb, _ := cell(0)
	shade, _ := cell(1)
	_, mixed := cell(3)
	if hb != "▀" || shade == "▀" || mixed.Bg == (term.Style{}).Bg {
		t.Errorf("mode views: half-block cell %q, shade cell %q, mixed background %v; want a half block, a shade character, a colored ground", hb, shade, mixed.Bg)
	}
	if a.state.Mode != game.ModeASCII {
		t.Errorf("drawing the four modes changed the game's mode to %v", a.state.Mode)
	}
}
