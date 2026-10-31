package game

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Smarter enemies (M13, D52). Enemies start as before (tier 1: everyone
// walks the shortest way to the player) and learn by tier. Balancing (M19)
// may tune the numbers.
const (
	HearTier  = 2 // from this tier, gunshots alert enemies
	CoverTier = 3 // from this tier, shooters hide after shooting
	FlankTier = 4 // from this tier, zombies come from several sides

	HearingSteps = 15  // a gunshot alerts enemies this many steps away (walking distance) or closer
	AlertTime    = 3.0 // seconds an alerted enemy rushes
	RushMul      = 1.4 // speed while rushing

	ShooterHideTime = 1.5 // seconds a shooter stays in cover after a shot
	coverSearch     = 3   // cells: how far a shooter looks for cover

	flankDist    = 3.0  // cells beside or behind the player a flanking zombie aims for
	flankRefresh = 0.75 // seconds between working out a flanking zombie's way again
	flankDone    = 1.0  // cells: this close to its spot, it comes straight from there on
	flankClear   = 2.0  // cells: its way keeps this far from the player

	CampTime = 60.0 // seconds without a kill before every enemy hunts the player
	HuntMul  = 1.2  // speed while hunting
)

// Hunting reports whether the enemies are hunting the player (anti-camping,
// M13): no kill for CampTime seconds. It ends with the next kill.
func (s *State) Hunting() bool {
	return s.sinceKill >= CampTime
}

// updateBrains counts the time since the last kill and lets enemies hear
// the player's gunshot this frame. Update calls it before updateEnemies.
func updateBrains(s *State, dt float64) {
	wasHunting := s.Hunting()
	s.sinceKill += dt
	if !wasHunting && s.Hunting() {
		s.Events = append(s.Events, EventHuntStarted)
	}
	if s.Tier < HearTier || !s.HasEvent(EventShotFired) || len(s.Enemies) == 0 {
		return
	}
	ensureDistanceField(s)
	for i := range s.Enemies {
		e := &s.Enemies[i]
		// The distance field counts steps from the player, so it is how far
		// the sound walks: around walls, through open doors.
		if d := s.dist[int(e.Y)*s.Map.W+int(e.X)]; d >= 0 && d <= HearingSteps {
			e.alert = AlertTime
		}
	}
}

// speedFor is how fast e moves this frame: its base speed, its tier
// scaling, faster while rushing (it heard a shot) or hunting.
func speedFor(s *State, e *Enemy, base float64) float64 {
	v := base * e.speedMul
	if e.alert > 0 {
		v *= RushMul
	}
	if s.Hunting() {
		v *= HuntMul
	}
	return v
}

// moveMelee walks a melee enemy toward the player: straight down the
// shared distance field, or, for a flanking zombie from tier FlankTier,
// to a spot beside or behind the player first.
func moveMelee(s *State, e *Enemy, step float64, dt float64) {
	if e.Kind != KindZombie || e.role == 0 || s.Tier < FlankTier || s.Hunting() || e.alert > 0 {
		chase(s, e, step)
		return
	}
	p := s.Player
	e.routeAge -= dt
	if e.routeAge <= 0 {
		e.routeAge = flankRefresh
		e.flankX, e.flankY, e.flanking = flankSpot(s, e.role)
		if e.flanking {
			// A way that keeps clear of the player, so it doesn't walk
			// through them on the way to their side or back.
			near := func(x, y int) bool {
				return math.Hypot(float64(x)+0.5-p.X, float64(y)+0.5-p.Y) < flankClear
			}
			e.route = world.DistanceFieldExcept(s.Map, e.flankX, e.flankY, e.route, near)
		}
	}
	spotX, spotY := float64(e.flankX)+0.5, float64(e.flankY)+0.5
	if e.flanking && math.Hypot(spotX-e.X, spotY-e.Y) < flankDone {
		e.role = 0 // it got round: from here on it comes straight
	}
	if !e.flanking || e.role == 0 {
		chase(s, e, step)
		return
	}
	if !followField(s, e, e.route, step) {
		chase(s, e, step)
	}
}

// flankSpot is where a flanking zombie with role (1 left, 2 right, 3
// behind, as the player faces) aims: flankDist cells that way, moved to
// the nearest cell enemies can stand on. ok is false if there is none.
func flankSpot(s *State, role int) (x, y int, ok bool) {
	p := s.Player
	turn := [4]float64{0, -math.Pi / 2, math.Pi / 2, math.Pi}[role]
	a := p.Angle + turn
	tx, ty := p.X+flankDist*math.Cos(a), p.Y+flankDist*math.Sin(a)
	return nearestStand(s, tx, ty, 2)
}

// nearestStand returns the open cell enemies can reach nearest to (tx, ty)
// within radius cells, or ok = false.
func nearestStand(s *State, tx, ty float64, radius int) (x, y int, ok bool) {
	ensureDistanceField(s)
	best := math.Inf(1)
	cx, cy := int(math.Floor(tx)), int(math.Floor(ty))
	for yy := cy - radius; yy <= cy+radius; yy++ {
		for xx := cx - radius; xx <= cx+radius; xx++ {
			if xx < 0 || yy < 0 || xx >= s.Map.W || yy >= s.Map.H || s.Map.Blocked(xx, yy) || s.dist[yy*s.Map.W+xx] < 0 {
				continue
			}
			if d := math.Hypot(float64(xx)+0.5-tx, float64(yy)+0.5-ty); d < best {
				best, x, y, ok = d, xx, yy, true
			}
		}
	}
	return x, y, ok
}

// takeCover sends shooter e, which just fired, to the cell fewest steps
// away (at most coverSearch) where the player can't see it (behind a crate,
// a pillar or a corner), to stay there ShooterHideTime after it arrives.
// If there is none it stays and fights.
func takeCover(s *State, e *Enemy) {
	ensureDistanceField(s)
	p := s.Player
	ex, ey := int(e.X), int(e.Y)
	steps := world.DistanceField(s.Map, ex, ey, e.route) // how far each cell is from the shooter
	best := coverSearch + 1
	found := false
	for y := ey - coverSearch; y <= ey+coverSearch; y++ {
		for x := ex - coverSearch; x <= ex+coverSearch; x++ {
			if x < 0 || y < 0 || x >= s.Map.W || y >= s.Map.H || s.Map.Blocked(x, y) || s.dist[y*s.Map.W+x] < 0 {
				continue
			}
			cx, cy := float64(x)+0.5, float64(y)+0.5
			if d := steps[y*s.Map.W+x]; d < 0 || d >= best || world.LineOfSight(s.Map, p.X, p.Y, cx, cy) {
				continue
			}
			best, e.flankX, e.flankY, found = steps[y*s.Map.W+x], x, y, true
		}
	}
	if !found {
		return
	}
	e.route = world.DistanceField(s.Map, e.flankX, e.flankY, steps)      // reuses the buffer: now the way to the cover
	e.hide = ShooterHideTime + float64(best+1)/(ShooterSpeed*e.speedMul) // the walk there, then the wait
}

// followField moves e up to step cells downhill on field (a distance field
// toward some cell), like chase does on the player's. It reports false
// when e's cell isn't on the field (the way is shut).
func followField(s *State, e *Enemy, field []int, step float64) bool {
	m := s.Map
	cx, cy := int(e.X), int(e.Y)
	i := cy*m.W + cx
	if i < 0 || i >= len(field) || field[i] < 0 {
		return false
	}
	best := field[i]
	tx, ty := float64(cx)+0.5, float64(cy)+0.5 // the middle of its own cell, if that's the goal
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := cx+d[0], cy+d[1]
		if nx < 0 || ny < 0 || nx >= m.W || ny >= m.H {
			continue
		}
		if v := field[ny*m.W+nx]; v >= 0 && v < best {
			best = v
			tx, ty = float64(nx)+0.5, float64(ny)+0.5
		}
	}
	mx, my := tx-e.X, ty-e.Y
	l := math.Hypot(mx, my)
	if l < 1e-6 {
		return true
	}
	step = min(step, l)
	e.X, e.Y = world.TryMove(m, e.X, e.Y, mx/l*step, my/l*step, EnemyRadius)
	return true
}
