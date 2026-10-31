package game

import "testing"

func wavesState(t *testing.T) *State {
	t.Helper()
	s := survivalState(t)
	s.StartWaves()
	return s
}

// The first wave comes after FirstWaveBreak; no enemies before it.
func TestFirstWave(t *testing.T) {
	s := wavesState(t)
	if !s.InBreak() || s.Wave != 0 {
		t.Fatalf("start: break %v, wave %d; want a break before wave 1", s.InBreak(), s.Wave)
	}
	starts := run(s, Input{}, int(FirstWaveBreak*30)-2, EventWaveStarted)
	if starts != 0 || len(s.Enemies) != 0 {
		t.Fatalf("during the first break: %d wave starts, %d enemies; want none", starts, len(s.Enemies))
	}
	if run(s, Input{}, 4, EventWaveStarted) != 1 || s.Wave != 1 || len(s.Enemies) != 1 {
		t.Errorf("after the break: wave %d, %d enemies; want wave 1 and its first enemy", s.Wave, len(s.Enemies))
	}
}

// A wave sends exactly WaveSize enemies, then ends when they're all dead;
// a break follows.
func TestWaveSizeAndClear(t *testing.T) {
	s := wavesState(t)
	s.BreakLeft = frameDT / 2
	spawned := 0
	for i := 0; i < 120*30 && !s.HasEvent(EventWaveCleared); i++ {
		Update(s, Input{}, frameDT)
		s.Player.HP = PlayerMaxHP
		spawned += count(s, EventEnemySpawned)
		s.Enemies = s.Enemies[:0] // as if killed at once
	}
	if !s.HasEvent(EventWaveCleared) || spawned != WaveSize(1) {
		t.Fatalf("wave 1: cleared %v after %d enemies, want cleared after %d", s.HasEvent(EventWaveCleared), spawned, WaveSize(1))
	}
	if !s.InBreak() || s.BreakLeft != WaveBreak {
		t.Errorf("after clearing: break %v, %.1f s left; want %.0f s", s.InBreak(), s.BreakLeft, WaveBreak)
	}
}

// Waves grow, tiers follow them, and clearing the last one wins.
func TestWavesToTheEnd(t *testing.T) {
	if WaveSize(2) <= WaveSize(1) || waveTier(1) != 1 || waveTier(3) != 2 || waveTier(7) != 4 || waveTier(NumWaves) < FlankTier {
		t.Fatal("waves don't grow, or tiers don't follow them")
	}
	s := wavesState(t)
	won := 0
	for i := 0; i < 30*60*30 && !s.Won; i++ {
		Update(s, Input{}, frameDT)
		s.Player.HP = PlayerMaxHP
		s.Enemies = s.Enemies[:0]
		won += count(s, EventWavesWon)
	}
	if !s.Won || won != 1 || s.Wave != NumWaves || s.Tier != waveTier(NumWaves) {
		t.Fatalf("won %v (%d events) at wave %d, tier %d; want a win at wave %d, tier %d", s.Won, won, s.Wave, s.Tier, NumWaves, waveTier(NumWaves))
	}
	if run(s, Input{}, 60, EventWaveStarted) != 0 {
		t.Error("a wave started after the win")
	}
}

// Kills and time don't raise the tier in a waves run.
func TestWavesIgnoreTierRules(t *testing.T) {
	s := wavesState(t)
	s.Kills = 3 * TierKills
	run(s, Input{}, 30, 0)
	if s.Tier != 1 {
		t.Errorf("tier %d after many kills in the first break, want 1", s.Tier)
	}
}
