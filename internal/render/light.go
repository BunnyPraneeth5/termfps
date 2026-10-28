package render

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// lightGrid is the light level of every map cell for one frame (M4): 1 is
// the usual brightness, lower in a dark map, higher near lamps.
type lightGrid struct {
	on      bool // false: no darkness and no lamps, every cell is 1
	w, h    int
	ambient float64   // the light outside lamps, and outside the map
	cells   []float64 // row-major, reused between frames
}

// maxLight caps how bright a lamp can make a cell.
const maxLight = 1.8

// prepare fills the grid for map m at time t (seconds, for flicker).
func (g *lightGrid) prepare(m *world.Map, t float64) {
	g.on = m.Darkness > 0 || len(m.Lamps) > 0
	g.ambient = 1 - m.Darkness
	if !g.on {
		return
	}
	g.w, g.h = m.W, m.H
	if n := m.W * m.H; cap(g.cells) >= n {
		g.cells = g.cells[:n]
	} else {
		g.cells = make([]float64, n)
	}
	for i := range g.cells {
		g.cells[i] = g.ambient
	}
	for i, l := range m.Lamps {
		power := 1.0
		if l.Flicker {
			power = flicker(t, i)
		}
		for _, c := range l.Reach {
			g.cells[c.Index] += c.Weight * power
		}
	}
	for i := range g.cells {
		g.cells[i] = math.Min(g.cells[i], maxLight)
	}
}

// at returns the light of cell (x, y).
func (g *lightGrid) at(x, y int) float64 {
	if !g.on {
		return 1
	}
	if x < 0 || y < 0 || x >= g.w || y >= g.h {
		return g.ambient
	}
	return g.cells[y*g.w+x]
}

// smooth returns the light at world point (x, y), blended between the four
// nearest cell centers, so lamp light on the floor has no hard cell edges.
func (g *lightGrid) smooth(x, y float64) float64 {
	if !g.on {
		return 1
	}
	gx, gy := x-0.5, y-0.5 // cell centers are at +0.5
	x0, y0 := int(math.Floor(gx)), int(math.Floor(gy))
	fx, fy := gx-float64(x0), gy-float64(y0)
	top := g.at(x0, y0)*(1-fx) + g.at(x0+1, y0)*fx
	bottom := g.at(x0, y0+1)*(1-fx) + g.at(x0+1, y0+1)*fx
	return top*(1-fy) + bottom*fy
}

// flicker is a flickering lamp's power at time t: a restless 0.6-1.0, and
// now and then (about 6% of eighth-seconds) a short drop to 0.25. It's a
// fixed function of the time and the lamp's number, so it needs no random
// source and looks the same every run.
func flicker(t float64, lamp int) float64 {
	seed := float64(lamp) * 1.7
	v := 0.8 + 0.12*math.Sin(9*t+seed) + 0.08*math.Sin(23*t+2*seed)
	step := math.Floor(t * 8)
	h := math.Sin(step*12.9898+seed*78.233) * 43758.5453 // a well-known one-line hash
	if h-math.Floor(h) < 0.06 {
		v = 0.25
	}
	return v
}
