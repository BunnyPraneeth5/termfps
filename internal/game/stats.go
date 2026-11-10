package game

// Run stats (U5, D54): what the end screens tell about a run. Update keeps
// them as it goes; nothing here uses the random source.

// NumEnemyKinds is how many enemy kinds there are.
const NumEnemyKinds = int(KindExploder) + 1

// CauseKind says what hurt the player.
type CauseKind int

const (
	CauseUnknown   CauseKind = iota // the zero value: not hurt by anything the game tracks
	CauseEnemy                      // an enemy's hit or shot
	CauseExplosion                  // a blast: a red barrel or an exploder
	CauseAcid                       // the acid floor
)

// Cause is what hurt the player: its kind, and for CauseEnemy which enemy.
type Cause struct {
	Kind  CauseKind
	Enemy EnemyKind
}

// RunStats are the numbers of one run.
type RunStats struct {
	Shots int // shots fired (a shotgun blast is one shot)
	Hits  int // shots that hit an enemy (a shotgun blast counts once)
	// KillsBy counts the kills per enemy kind; they add up to State.Kills.
	KillsBy     [NumEnemyKinds]int
	DamageTaken int   // HP lost (what armor took doesn't count)
	Killer      Cause // what took the last HP; CauseUnknown while alive
}

// Accuracy is the share (0-1) of shots that hit an enemy, and false when
// no shot was fired.
func (r RunStats) Accuracy() (share float64, ok bool) {
	if r.Shots == 0 {
		return 0, false
	}
	return float64(r.Hits) / float64(r.Shots), true
}

// hurtPlayerBy is hurtPlayer for damage with a known cause, for the stats.
func hurtPlayerBy(s *State, damage int, c Cause) {
	s.cause = c
	hurtPlayer(s, damage)
}
