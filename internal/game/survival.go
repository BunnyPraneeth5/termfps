package game

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Survival rules (D21). Balancing (P1-16) may tune them.
const (
	TierKills         = 20    // kills in a tier before the next one
	TierTime          = 180.0 // seconds in a tier before the next one (3 minutes; 4 before M12, D49)
	FirstSpawnDelay   = 10.0  // seconds before the first spawn of a run
	SpawnMinDistance  = 6.0   // cells: never spawn closer to the player
	spawnTries        = 40    // random cells to try per spawn before giving up
	spawnMinSeparated = 1.0   // cells: never spawn on top of another enemy
)

// TierStats is how strong and how many enemies are at a tier.
type TierStats struct {
	HPMul        float64 // enemy HP multiplier
	DamageMul    float64 // enemy damage multiplier
	SpeedMul     float64 // enemy speed multiplier
	SpawnEvery   float64 // seconds between spawns
	MaxAlive     int     // spawns pause while this many enemies are alive
	ShooterShare float64 // chance (0-1) a zombie-or-shooter spawn is a shooter
	// Chances (0-1) a spawn is one of the M12 kinds instead; they start at
	// tiers 2, 3 and 4 (D49).
	RunnerShare, TankShare, ExploderShare float64
}

// TierFor returns the stats for a tier (1 = start). Every number for the
// difficulty curve is here, so balancing is one place.
func TierFor(tier int) TierStats {
	t := float64(max(tier, 1) - 1)
	return TierStats{
		HPMul:         1 + 0.25*t,
		DamageMul:     1 + 0.15*t,
		SpeedMul:      math.Min(1+0.05*t, 1.5),
		SpawnEvery:    math.Max(8-0.6*t, 2), // tier 1: every 8 s
		MaxAlive:      min(2+tier, 14),      // tier 1: 3 alive
		ShooterShare:  math.Min(0.2+0.05*t, 0.5),
		RunnerShare:   from(tier, 2, 0.2),
		TankShare:     from(tier, 3, 0.1),
		ExploderShare: from(tier, 4, 0.15),
	}
}

// from is share at tier start and above, 0 below it.
func from(tier, start int, share float64) float64 {
	if tier < start {
		return 0
	}
	return share
}

// spawnKind picks the kind of a new enemy at stats' tier.
func spawnKind(s *State, stats TierStats) EnemyKind {
	if stats.RunnerShare > 0 { // only from tier 2, so tier 1 uses the random numbers as before M12
		r := s.Rand.Float64()
		switch {
		case r < stats.RunnerShare:
			return KindRunner
		case r < stats.RunnerShare+stats.TankShare:
			return KindTank
		case r < stats.RunnerShare+stats.TankShare+stats.ExploderShare:
			return KindExploder
		}
	}
	if s.Rand.Float64() < stats.ShooterShare {
		return KindShooter
	}
	return KindZombie
}

// updateSurvival raises the tier when enough kills or time pass, and spawns
// enemies on a timer.
func updateSurvival(s *State, dt float64) {
	if s.Kills-s.tierKills >= TierKills || s.Time-s.tierStart >= TierTime {
		s.Tier++
		s.tierKills, s.tierStart = s.Kills, s.Time
		s.Events = append(s.Events, EventTierUp)
	}
	if !s.Spawning {
		return
	}
	s.spawnTimer -= dt
	if s.spawnTimer > 0 {
		return
	}
	stats := TierFor(s.Tier)
	s.spawnTimer = stats.SpawnEvery
	if len(s.Enemies) >= stats.MaxAlive {
		return
	}
	x, y, ok := spawnSpot(s)
	if !ok {
		return // no hidden, far-enough cell found this time; try again next timer
	}
	e := newEnemy(spawnKind(s, stats), x, y)
	e.HP = int(math.Round(float64(e.HP) * stats.HPMul))
	e.damageMul, e.speedMul = stats.DamageMul, stats.SpeedMul
	s.Enemies = append(s.Enemies, e)
	s.Events = append(s.Events, EventEnemySpawned)
	switch e.Kind {
	case KindRunner:
		s.Events = append(s.Events, EventRunnerSpawned)
	case KindTank:
		s.Events = append(s.Events, EventTankSpawned)
	case KindExploder:
		s.Events = append(s.Events, EventExploderSpawned)
	}
}

// spawnSpot picks a random free cell the player can reach but can't see, at
// least SpawnMinDistance away and not on another enemy. ok is false if
// none of spawnTries random cells qualify.
func spawnSpot(s *State) (x, y float64, ok bool) {
	ensureDistanceField(s)
	p := s.Player
	m := s.Map
	for range spawnTries {
		cx, cy := s.Rand.IntN(m.W), s.Rand.IntN(m.H)
		if !freeCell(m, cx, cy) || s.dist[cy*m.W+cx] < 0 {
			continue
		}
		x, y = float64(cx)+0.5, float64(cy)+0.5
		if math.Hypot(x-p.X, y-p.Y) < SpawnMinDistance || world.LineOfSight(m, p.X, p.Y, x, y) {
			continue
		}
		if nearEnemy(s, x, y) {
			continue
		}
		return x, y, true
	}
	return 0, 0, false
}

// nearEnemy reports whether an enemy stands within spawnMinSeparated of (x, y).
func nearEnemy(s *State, x, y float64) bool {
	for _, e := range s.Enemies {
		if math.Hypot(e.X-x, e.Y-y) < spawnMinSeparated {
			return true
		}
	}
	return false
}

// ensureDistanceField recomputes the distance field when the player is in
// a new cell (or after the map changed).
func ensureDistanceField(s *State) {
	p := s.Player
	cell := int(p.Y)*s.Map.W + int(p.X)
	if cell != s.distCell || s.dist == nil {
		s.dist = world.DistanceField(s.Map, int(p.X), int(p.Y), s.dist)
		s.distCell = cell
	}
}
