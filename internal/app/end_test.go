package app

import (
	"strings"
	"testing"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/save"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// die ends the current run by death with the given numbers and shows the
// game over screen.
func die(a *App, now *time.Time, kills int, seconds float64) {
	a.state.Kills, a.state.Time = kills, seconds
	a.state.Player.HP = 0
	press(a, now)
	press(a, now)
}

// bigBlocks counts the block-letter cells on rows 1-5 in color c.
func bigBlocks(s *term.Fake, c term.Color) int {
	n := 0
	for y := 1; y <= ui.FontRows; y++ {
		for x := 0; x < 80; x++ {
			if str, st := s.Get(x, y); str == "█" && st.Fg == c {
				n++
			}
		}
	}
	return n
}

// The game over screen: YOU DIED in red block letters, what killed you,
// the run's numbers in boxes with NEW BEST against the best on that map,
// what you killed, and three buttons. It fits 80x24.
func TestGameOverScreen(t *testing.T) {
	a, s := newMenuApp(t)
	a.progress.Best[""] = save.Best{Kills: 9, Time: 60, Tier: 3}
	now := t0
	press(a, &now, enter, enter) // Play, Start
	st := a.state
	st.Stats = game.RunStats{Shots: 40, Hits: 30, DamageTaken: 180, Killer: game.Cause{Kind: game.CauseEnemy, Enemy: game.KindTank}}
	st.Stats.KillsBy[game.KindRunner], st.Stats.KillsBy[game.KindTank], st.Stats.KillsBy[game.KindZombie] = 3, 1, 1
	st.Tier = 2
	die(a, &now, 5, 95)
	if bigBlocks(s, ui.ColRed) < 80 {
		t.Errorf("%d red block cells, want YOU DIED in big red letters", bigBlocks(s, ui.ColRed))
	}
	for _, want := range []string{"killed by a tank · this map · survival", "TIME", "1:35", " NEW BEST ", "KILLS", "best 9", "TIER", "best 3",
		"ACCURACY", "75%", "30 of 40 shots", "this run: 3 runners · 1 zombie · 1 tank", "damage taken 180", "explored ",
		"▶ TRY AGAIN", "CHANGE MAP", "MAIN MENU", " ←→  pick", " esc  main menu"} {
		if !screenHas(s, want) {
			t.Errorf("game over screen doesn't show %q", want)
		}
	}
	if strings.Count(s.Text(), "NEW BEST") != 1 {
		t.Errorf("%d NEW BEST badges, want only the time's", strings.Count(s.Text(), "NEW BEST"))
	}
	press(a, &now, right, enter) // Change map
	if _, ok := a.current.(*playScreen); !ok {
		t.Errorf("CHANGE MAP opened %T, want the Play screen", a.current)
	}
}

// Buttons work by click; Esc goes to the main menu; Up and Down do
// nothing (W and S may still be held from the game).
func TestEndScreenButtons(t *testing.T) {
	a, s := newTestApp(t)
	now := t0
	die(a, &now, 0, 3)
	press(a, &now, down, up, key(term.KeyRune, 's'))
	if es := a.current.(*endScreen); es.sel != endAgain {
		t.Errorf("Up/Down moved the selection to %d", es.sel)
	}
	if !screenHas(s, "no shots fired") || !screenHas(s, "this run: no kills") {
		t.Error("a run without shots or kills doesn't say so")
	}
	row, col := -1, -1
	for y := 0; y < 24; y++ {
		if i := strings.Index(s.Row(y), "MAIN MENU"); i >= 0 {
			row, col = y, len([]rune(s.Row(y)[:i]))
		}
	}
	a.step(now.Add(time.Second/30), []term.Event{term.Mouse{X: col, Y: row, Buttons: term.ButtonLeft}, term.Mouse{X: col, Y: row}})
	if !onTitle(a) || a.lastCue != audio.MenuClick {
		t.Errorf("clicking MAIN MENU: screen %T", a.current)
	}
	a.newRun()
	die(a, &now, 0, 3)
	press(a, &now, esc)
	if !onTitle(a) {
		t.Error("Esc on the game over screen didn't go to the title")
	}
}

// The win screen: YOU WON in green, stars, the time as NEW BEST only when
// it's the fastest win, HP left.
func TestWinScreen(t *testing.T) {
	a, s := newMenuApp(t)
	a.progress.Waves[""] = save.WavesBest{Wave: 10, Wins: 1, WinTime: 300, Kills: 142}
	now := t0
	press(a, &now, enter, right, enter) // Play, Waves, Start
	a.state.Wave, a.state.Kills, a.state.Time, a.state.Won = game.NumWaves, 130, 432, true
	a.state.Player.HP = 64
	press(a, &now)
	press(a, &now)
	if bigBlocks(s, ui.ColGreen) < 60 {
		t.Errorf("%d green block cells, want YOU WON in big green letters", bigBlocks(s, ui.ColGreen))
	}
	for _, want := range []string{"all 10 waves cleared · this map · waves mode", "7:12", "best 5:00", "best 142", "HP LEFT", "64", "of 100", "▶ PLAY AGAIN", "✦"} {
		if !screenHas(s, want) {
			t.Errorf("win screen doesn't show %q", want)
		}
	}
	if screenHas(s, "NEW BEST") {
		t.Error("a slower win with fewer kills shows NEW BEST")
	}
	press(a, &now, enter) // Play again
	a.state.Wave, a.state.Kills, a.state.Time, a.state.Won = game.NumWaves, 150, 250, true
	press(a, &now)
	press(a, &now)
	if strings.Count(s.Text(), "NEW BEST") != 2 {
		t.Errorf("a faster win with more kills shows %d NEW BEST badges, want 2", strings.Count(s.Text(), "NEW BEST"))
	}
}

// Dying in waves mode shows the wave reached.
func TestGameOverInWaves(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	press(a, &now, enter, right, enter)
	a.state.Wave = 4
	die(a, &now, 12, 200)
	for _, want := range []string{"WAVE", "4/10", "this map · waves"} {
		if !screenHas(s, want) {
			t.Errorf("game over in waves mode doesn't show %q", want)
		}
	}
}

// With animations on, the letters come row by row, the numbers count up
// and NEW BEST blinks, then everything stands still.
func TestEndScreenAnimations(t *testing.T) {
	a, s := newMenuApp(t)
	a.cfg.Animations = true
	now := t0
	press(a, &now, enter, enter)
	a.state.Kills, a.state.Time = 20, 95
	a.state.Player.HP = 0
	press(a, &now) // the run ends
	press(a, &now) // the first frame of the screen
	first := bigBlocks(s, ui.ColRed)
	if first == 0 || strings.Contains(s.Row(5), "█") || strings.Contains(s.Row(11), "1:35") {
		t.Errorf("first frame: %d block cells, last letter row drawn %v, final time shown %v; want it to build up",
			first, strings.Contains(s.Row(5), "█"), strings.Contains(s.Row(11), "1:35"))
	}
	blinked := false
	for range 45 { // 1.5 s
		press(a, &now)
		blinked = blinked || !screenHas(s, "NEW BEST")
	}
	if !blinked {
		t.Error("NEW BEST never blinked off")
	}
	for range 30 {
		press(a, &now)
	}
	if bigBlocks(s, ui.ColRed) <= first || !strings.Contains(s.Row(11), "1:35") || !screenHas(s, "NEW BEST") {
		t.Error("after the animations the letters, the time or NEW BEST aren't all there")
	}
}

func TestKilledByAndKillsLine(t *testing.T) {
	for c, want := range map[game.Cause]string{
		{Kind: game.CauseEnemy, Enemy: game.KindExploder}: "killed by an exploder",
		{Kind: game.CauseEnemy, Enemy: game.KindZombie}:   "killed by a zombie",
		{Kind: game.CauseExplosion}:                       "killed by an explosion",
		{Kind: game.CauseAcid}:                            "killed by acid",
		{}:                                                "",
	} {
		if got := killedBy(c); got != want {
			t.Errorf("killedBy(%+v) = %q, want %q", c, got, want)
		}
	}
	var by [game.NumEnemyKinds]int
	by[game.KindTank], by[game.KindExploder], by[game.KindRunner] = 4, 2, 31
	if got := killsLine(by); got != "31 runners · 4 tanks · 2 exploders" {
		t.Errorf("killsLine = %q", got)
	}
}
