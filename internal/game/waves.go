package game

// Waves mode (M15, D49): a second way to play next to endless survival.
// Balancing (M19) may tune the numbers.
const (
	NumWaves       = 10   // clear this many to win
	WaveBreak      = 15.0 // seconds between waves: reload, pick things up
	FirstWaveBreak = FirstSpawnDelay
	waveFirstSize  = 4 // enemies in wave 1
	waveGrowth     = 2 // more enemies in each next wave
)

// PlayMode is how a run is played.
type PlayMode int

const (
	Survival PlayMode = iota // endless, tiers by kills and time (the zero value, as before M15)
	Waves                    // NumWaves waves with breaks between; win after the last
)

// String names the mode for menus and the save file.
func (m PlayMode) String() string {
	if m == Waves {
		return "waves"
	}
	return "survival"
}

// WaveSize is how many enemies wave n (from 1) sends.
func WaveSize(n int) int {
	return waveFirstSize + waveGrowth*(n-1)
}

// waveTier is the difficulty tier during wave n: one tier every two
// waves, so all enemy kinds and what they learn (M12, M13) show up.
func waveTier(n int) int {
	return 1 + (max(n, 1)-1)/2
}

// StartWaves turns s into a waves run: the first wave comes after
// FirstWaveBreak. Call it right after NewState.
func (s *State) StartWaves() {
	s.Play = Waves
	s.Wave = 0
	s.BreakLeft = FirstWaveBreak
}

// InBreak reports whether a waves run is between waves (or before the
// first): no enemies come.
func (s *State) InBreak() bool {
	return s.Play == Waves && s.BreakLeft > 0
}

// WaveLeft is how many of the current wave's enemies are still to kill:
// those alive and those still to come.
func (s *State) WaveLeft() int {
	return s.waveLeft + len(s.Enemies)
}

// updateWaves runs the waves: counts the break down, starts the next wave,
// and ends a wave once all its enemies are out and dead. It replaces the
// survival tier rules and spawner timing in a waves run.
func updateWaves(s *State, dt float64) {
	if s.Won {
		return
	}
	if s.BreakLeft > 0 {
		s.BreakLeft = max(s.BreakLeft-dt, 0)
		if s.BreakLeft > 0 {
			return
		}
		s.Wave++
		s.waveLeft = WaveSize(s.Wave)
		s.spawnTimer = 0 // the first enemy comes right away
		s.sinceKill = 0  // a break isn't camping (M13)
		if t := waveTier(s.Wave); t != s.Tier {
			s.Tier = t
			s.Events = append(s.Events, EventTierUp)
		}
		s.Events = append(s.Events, EventWaveStarted)
	}
	if s.waveLeft == 0 && len(s.Enemies) == 0 {
		if s.Wave >= NumWaves {
			s.Won = true
			s.Events = append(s.Events, EventWavesWon)
			return
		}
		s.BreakLeft = WaveBreak
		s.Events = append(s.Events, EventWaveCleared)
		return
	}
	if s.waveLeft > 0 && spawnOne(s, dt) {
		s.waveLeft--
	}
}
