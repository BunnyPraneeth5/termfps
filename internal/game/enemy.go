package game

import (
	"math"
	"slices"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Combat numbers. Balancing (P1-16) may tune them; tiers (P1-08) scale the
// enemy ones.
const (
	PlayerMaxHP  = 100
	BulletDamage = 25 // per hit, enemies and players alike

	// EnemyRadius is every enemy's collision half-size with walls.
	EnemyRadius = 0.25

	ZombieHP            = 100 // 4 shots
	ZombieSpeed         = 2.0 // cells per second (the player does 3)
	ZombieMeleeRange    = 0.8 // center-to-center distance it can hit from
	ZombieMeleeDamage   = 10
	ZombieMeleeInterval = 1.0 // seconds between hits
	ZombieFirstHitDelay = 0.4 // seconds after reaching the player before the first hit

	ShooterHP             = 50  // 2 shots
	ShooterSpeed          = 1.6 // cells per second
	ShooterRange          = 6.0 // fires when it sees the player this close or closer
	ShooterMinRange       = 2.5 // backs away when the player is closer than this
	ShooterDamage         = 8
	ShooterInterval       = 1.5 // seconds between shots
	ShooterFirstShotDelay = 0.8 // seconds from spotting the player to the first shot
	// Hit chance falls with distance: ShooterAccNear at 0 cells, minus
	// ShooterAccPerCell per cell, never below ShooterAccMin.
	ShooterAccNear    = 0.9
	ShooterAccPerCell = 0.11
	ShooterAccMin     = 0.25

	// EnemyHitRadius: a shot hits an enemy if it passes this close to its
	// center (about the width of its sprite).
	EnemyHitRadius = 0.3
	// directChaseRange: closer than this with a clear view, enemies walk
	// straight at the player instead of following the grid.
	directChaseRange = 2.5

	hurtFlashTime  = 0.25 // seconds the screen edges glow red after a hit
	enemyFlashTime = 0.12 // seconds an enemy shows white after being shot
)

// EnemyKind tells enemy types apart.
type EnemyKind int

const (
	KindZombie  EnemyKind = iota // green, melee
	KindShooter                  // red, ranged
)

// Enemy is one enemy in the world.
type Enemy struct {
	Kind EnemyKind
	X, Y float64
	HP   int

	attackCD float64 // seconds until it may attack again
	flash    float64 // seconds left of its white "hit" flash
	sawLast  bool    // shooter: it saw the player last frame

	// Tier scaling (P1-08), set when it spawns; 1 for map enemies.
	damageMul, speedMul float64
}

// damage scales a base damage by the enemy's tier.
func (e Enemy) damage(base int) int {
	return int(math.Round(float64(base) * e.damageMul))
}

// Flashing reports whether the enemy was just shot (drawn white).
func (e Enemy) Flashing() bool {
	return e.flash > 0
}

// newZombie returns a zombie standing at (x, y).
func newZombie(x, y float64) Enemy {
	return Enemy{Kind: KindZombie, X: x, Y: y, HP: ZombieHP, attackCD: ZombieFirstHitDelay, damageMul: 1, speedMul: 1}
}

// newShooter returns a shooter standing at (x, y).
func newShooter(x, y float64) Enemy {
	return Enemy{Kind: KindShooter, X: x, Y: y, HP: ShooterHP, damageMul: 1, speedMul: 1}
}

// updateEnemies moves every enemy and lets it attack. It refreshes the
// distance field when the player is in a new cell.
func updateEnemies(s *State, dt float64) {
	if len(s.Enemies) == 0 {
		return
	}
	ensureDistanceField(s)
	for i := range s.Enemies {
		e := &s.Enemies[i] // pointer: change the enemy in the slice, not a copy
		e.flash = max(e.flash-dt, 0)
		e.attackCD = max(e.attackCD-dt, 0)
		switch e.Kind {
		case KindZombie:
			updateZombie(s, e, dt)
		case KindShooter:
			updateShooter(s, e, dt)
		}
		if s.Player.Dead() {
			return
		}
	}
}

// updateZombie walks toward the player and hits when in reach.
func updateZombie(s *State, e *Enemy, dt float64) {
	p := s.Player
	if math.Hypot(p.X-e.X, p.Y-e.Y) > ZombieMeleeRange {
		e.attackCD = max(e.attackCD, ZombieFirstHitDelay) // fresh delay on (re)arrival
		chase(s, e, ZombieSpeed*e.speedMul*dt)
		return
	}
	if e.attackCD == 0 {
		hurtPlayer(s, e.damage(ZombieMeleeDamage))
		e.attackCD = ZombieMeleeInterval
	}
}

// updateShooter fires when it sees the player within range, backs off when
// the player is too close, and otherwise walks toward the player (around
// walls when it can't see them).
func updateShooter(s *State, e *Enemy, dt float64) {
	p := s.Player
	dist := math.Hypot(p.X-e.X, p.Y-e.Y)
	sees := dist <= ShooterRange && world.LineOfSight(s.Map, e.X, e.Y, p.X, p.Y)
	if sees && !e.sawLast {
		e.attackCD = max(e.attackCD, ShooterFirstShotDelay) // a moment to aim
	}
	e.sawLast = sees
	if !sees {
		chase(s, e, ShooterSpeed*e.speedMul*dt)
		return
	}
	if dist < ShooterMinRange {
		retreat(s, e, ShooterSpeed*e.speedMul*dt)
	}
	if e.attackCD == 0 {
		e.attackCD = ShooterInterval
		s.Events = append(s.Events, EventEnemyShot)
		if s.Rand.Float64() < ShooterHitChance(dist) {
			hurtPlayer(s, e.damage(ShooterDamage))
		}
	}
}

// ShooterHitChance is the chance (0-1) a shooter's shot hits from dist cells.
func ShooterHitChance(dist float64) float64 {
	return max(ShooterAccNear-ShooterAccPerCell*dist, ShooterAccMin)
}

// retreat moves e up to step cells straight away from the player.
func retreat(s *State, e *Enemy, step float64) {
	mx, my := e.X-s.Player.X, e.Y-s.Player.Y
	l := math.Hypot(mx, my)
	if l < 1e-6 {
		return
	}
	e.X, e.Y = world.TryMove(s.Map, e.X, e.Y, mx/l*step, my/l*step, EnemyRadius)
}

// chase moves e up to step cells toward the player: straight at them when
// close and in view, otherwise to the neighboring cell that is fewer steps
// away on the distance field (so it walks around walls).
func chase(s *State, e *Enemy, step float64) {
	p := s.Player
	tx, ty := p.X, p.Y
	if math.Hypot(p.X-e.X, p.Y-e.Y) > directChaseRange || !world.LineOfSight(s.Map, e.X, e.Y, p.X, p.Y) {
		cx, cy := int(e.X), int(e.Y)
		best := s.dist[cy*s.Map.W+cx]
		if best >= 0 { // reachable: follow the field (else walk straight)
			tx, ty = e.X, e.Y
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := cx+d[0], cy+d[1]
				if nx < 0 || ny < 0 || nx >= s.Map.W || ny >= s.Map.H {
					continue
				}
				if v := s.dist[ny*s.Map.W+nx]; v >= 0 && v < best {
					best = v
					tx, ty = float64(nx)+0.5, float64(ny)+0.5
				}
			}
		}
	}
	mx, my := tx-e.X, ty-e.Y
	l := math.Hypot(mx, my)
	if l < 1e-6 {
		return
	}
	step = min(step, l)
	e.X, e.Y = world.TryMove(s.Map, e.X, e.Y, mx/l*step, my/l*step, EnemyRadius)
}

// hurtPlayer takes HP from the player (never below 0) and flashes the
// screen edges.
func hurtPlayer(s *State, damage int) {
	p := &s.Player
	if p.HP == 0 || damage <= 0 {
		return
	}
	p.HP = max(p.HP-damage, 0)
	p.hurt = hurtFlashTime
	s.Events = append(s.Events, EventPlayerHurt)
	if p.HP == 0 {
		s.Events = append(s.Events, EventPlayerDied)
	}
}

// shotHitsEnemy returns the index of the nearest enemy the shot from the
// player along (dirX, dirY) passes through before reaching maxDist, or -1.
// dirX, dirY must be a unit vector.
func shotHitsEnemy(s *State, dirX, dirY, maxDist float64) int {
	hit, best := -1, maxDist
	for i, e := range s.Enemies {
		rx, ry := e.X-s.Player.X, e.Y-s.Player.Y
		along := rx*dirX + ry*dirY // distance along the shot
		if along <= 0 || along >= best {
			continue
		}
		if off := math.Abs(rx*dirY - ry*dirX); off <= EnemyHitRadius { // sideways distance
			hit, best = i, along
		}
	}
	return hit
}

// damageEnemy hurts enemy i; at 0 HP it is removed and counts as a kill.
func damageEnemy(s *State, i, damage int) {
	e := &s.Enemies[i]
	e.HP -= damage
	e.flash = enemyFlashTime
	s.Events = append(s.Events, EventEnemyHit)
	if e.HP <= 0 {
		s.Enemies = slices.Delete(s.Enemies, i, i+1) // remove it, keeping the order
		s.Kills++
		s.Events = append(s.Events, EventEnemyKilled)
	}
}
