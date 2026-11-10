package app

import (
	"math"
	"math/rand/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// The title background (U2, D54): a camera wandering slowly through one of
// the maps the player has unlocked, behind the title menu (and behind the
// Settings and the Guide opened from it). Camera only: no enemies, no
// pickups, no sounds; doors stand open.
const (
	tourSpeed    = 0.35 * game.MoveSpeed // cells per second: slow motion
	tourTurn     = 0.9                   // radians per second while walking
	tourLookTurn = 0.45                  // radians per second while looking around at a stop
	tourLookMin  = math.Pi / 2           // a look-around turns 90 to 180 degrees
	tourLookMax  = math.Pi
	tourMinTrip  = 4        // cells: a trip's target is at least this many steps away, when there is one
	tourRedraw   = 1.0 / 15 // seconds between redraws of the 3D picture: keeps the CPU cost down
	tourTries    = 30       // random cells tried for a trip's target
)

// tour is the wandering camera. The zero value isn't usable: make one with
// newTour.
type tour struct {
	m   *world.Map
	st  *game.State // only carries the camera (Player) and the time for the renderer
	rng *rand.Rand

	// route is the cells still to walk, as cell centers; the camera heads
	// for route[0]. Empty while it looks around.
	route [][2]float64
	// look is how many radians are left of the look-around, lookDir its
	// direction (+1 or -1).
	look, lookDir float64
	field         []int // distance field buffer, reused

	frame   render.Frame
	sprites []render.Sprite
	since   float64 // seconds since the 3D picture was last drawn
	drawn   bool
}

// newTour starts a tour of m (a copy of it: the doors are opened) at its
// start, with all its choices from rng.
func newTour(m *world.Map, rng *rand.Rand) *tour {
	c := m.Clone()
	for i := range c.Doors {
		c.Doors[i].Open = 1
	}
	st := game.NewState(c)
	st.Spawning = false
	st.Enemies = nil
	t := &tour{m: c, st: st, rng: rng}
	t.st.Player.Angle = rng.Float64() * 2 * math.Pi
	t.startLook()
	return t
}

// startLook begins a slow look-around on the spot.
func (t *tour) startLook() {
	t.look = tourLookMin + t.rng.Float64()*(tourLookMax-tourLookMin)
	t.lookDir = 1
	if t.rng.IntN(2) == 0 {
		t.lookDir = -1
	}
}

// walkable reports whether the camera may stand in cell (x, y): no wall,
// no blocking prop. Doors count as open.
func (t *tour) walkable(x, y int) bool {
	c := t.m.At(x, y)
	return !c.Solid() && !c.BlocksMove()
}

// plan picks a random cell the camera can reach and sets route to the
// way there: downhill on the distance field from that cell, as the
// enemies walk. With nowhere to go, route stays empty.
func (t *tour) plan() {
	p := &t.st.Player
	px, py := int(p.X), int(p.Y)
	// Steps from the camera's cell to every cell, to find a far target.
	t.field = world.DistanceField(t.m, px, py, t.field)
	tx, ty, best := -1, -1, 0
	for range tourTries {
		x, y := t.rng.IntN(t.m.W), t.rng.IntN(t.m.H)
		d := t.field[y*t.m.W+x]
		if d > best {
			tx, ty, best = x, y, d
			if d >= tourMinTrip {
				break
			}
		}
	}
	if tx < 0 {
		return
	}
	// Steps from the target to every cell; walk from the camera to ever
	// smaller numbers.
	t.field = world.DistanceField(t.m, tx, ty, t.field)
	t.route = t.route[:0]
	x, y := px, py
	for t.field[y*t.m.W+x] > 0 {
		moved := false
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if !t.walkable(nx, ny) {
				continue
			}
			if f := t.field[ny*t.m.W+nx]; f >= 0 && f < t.field[y*t.m.W+x] {
				x, y, moved = nx, ny, true
				break
			}
		}
		if !moved {
			break // can't happen on a consistent field; stop rather than loop
		}
		t.route = append(t.route, [2]float64{float64(x) + 0.5, float64(y) + 0.5})
	}
}

// update moves the camera for dt seconds: it turns smoothly toward the
// next cell of its route and walks there (slower while it still has to
// turn), and at the end of a trip it stops, looks around, and plans the
// next one.
func (t *tour) update(dt float64) {
	p := &t.st.Player
	t.st.Time += dt // lamps flicker
	if t.look > 0 {
		step := min(tourLookTurn*dt, t.look)
		p.Angle = normAngle(p.Angle + t.lookDir*step)
		t.look -= step
		if t.look <= 0 {
			t.plan()
			if len(t.route) == 0 {
				t.startLook() // a map with nowhere to walk: keep looking around
			}
		}
		return
	}
	if len(t.route) == 0 {
		t.startLook()
		return
	}
	// The camera starts each trip at its cell's center or on the line to
	// it, so the straight line to the next cell's center stays in open
	// cells (the route only steps up, down, left or right).
	tx, ty := t.route[0][0], t.route[0][1]
	dx, dy := tx-p.X, ty-p.Y
	dist := math.Hypot(dx, dy)
	want := math.Atan2(dy, dx)
	diff := math.Remainder(want-p.Angle, 2*math.Pi) // -pi to pi: the short way round
	turn := min(max(diff, -tourTurn*dt), tourTurn*dt)
	p.Angle = normAngle(p.Angle + turn)
	// Walk only as fast as the camera faces the way it goes.
	step := tourSpeed * dt * max(math.Cos(diff-turn), 0)
	if step >= dist {
		p.X, p.Y = tx, ty
		t.route = t.route[1:]
		return
	}
	p.X += dx / dist * step
	p.Y += dy / dist * step
}

// normAngle wraps an angle into [0, 2*pi).
func normAngle(a float64) float64 {
	a = math.Mod(a, 2*math.Pi)
	if a < 0 {
		a += 2 * math.Pi
	}
	return a
}

// draw advances the tour by dt and puts its picture on the screen, dimmed.
// The 3D picture is only drawn again every tourRedraw seconds; between
// those the last one is shown, which costs the terminal nothing.
func (a *App) drawTour(dt float64) {
	t := a.tour
	if t == nil {
		// A random map among the open ones (locked maps stay a surprise).
		var open []int
		for i := range a.maps {
			if !a.mapLocked(i) {
				open = append(open, i)
			}
		}
		if len(open) == 0 {
			a.clearScreen()
			return
		}
		t = newTour(a.maps[open[a.rng.IntN(len(open))]], a.rng)
		a.tour = t
	}
	t.since += dt
	cols, rows := a.scr.Size()
	// Half the usual detail, and over every row (there's no HUD here):
	// ScaledFrameSize leaves the HUD's rows out, so ask for that many more.
	scale := 2 * renderScale(a.cfg.RenderScale, cols)
	w, h := render.ScaledFrameSize(cols, rows+render.HUDRows, scale)
	t.frame.ViewRows = rows
	resized := t.frame.W != w || t.frame.H != h
	if resized {
		t.frame.Scale = scale
		t.frame.Resize(w, h)
	}
	if !t.drawn || resized || t.since >= tourRedraw {
		t.update(t.since)
		t.since, t.drawn = 0, true
		f := &t.frame
		f.PixelAspect, f.Textures = a.frame.PixelAspect, a.frame.Textures
		render.DrawFrame(t.st, f)
		t.sprites = t.sprites[:0]
		for _, pr := range t.m.Props {
			t.sprites = append(t.sprites, render.PropSprite(pr.Kind, pr.X, pr.Y))
		}
		render.DrawSprites(f, t.st.Player, t.sprites)
		render.Dim(f)
	}
	a.presentFrame(&t.frame)
}
