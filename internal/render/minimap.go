package render

import (
	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Minimap colors: light walls (all textures alike) on a dark floor, white
// player arrow.
var (
	minimapWall   = tcell.StyleDefault.Background(grayColor(140))
	minimapTarget = tcell.StyleDefault.Background(rgbColor(170, TintRed))
	minimapFloor  = tcell.StyleDefault.Background(grayColor(25))
	minimapPlayer = tcell.StyleDefault.Background(grayColor(25)).Foreground(grayColor(255)).Bold(true)
	minimapAmmo   = tcell.StyleDefault.Background(grayColor(25)).Foreground(rgbColor(230, TintYellow))
)

// arrows point in 8 directions, starting east and turning clockwise on
// screen (y points down): E, SE, S, SW, W, NW, N, NE.
var arrows = [8]rune{'→', '↘', '↓', '↙', '←', '↖', '↑', '↗'}

// minimapCellWidth: each map cell is 2 columns wide, so cells look about
// square (terminal cells are about twice as tall as wide).
const minimapCellWidth = 2

// The minimap shows at most this many map cells across and down, so on big
// maps it stays a corner of the view and scrolls with the player (D26).
// Below minimapMinCells either way it isn't worth drawing.
const (
	minimapMaxW     = 24
	minimapMaxH     = 16
	minimapMinCells = 5
)

// DrawMinimap draws the map in the top-right corner, on top of the 3D view.
// When the map is bigger than the space it gets, it shows a window around
// the player. It returns false (and draws nothing) when the screen is too
// small for even a small window above the HUD row.
func DrawMinimap(s tcell.Screen, st *game.State) bool {
	m := st.Map
	w, h := s.Size()
	viewW := min(m.W, minimapMaxW, (w-1)/minimapCellWidth) // one column of margin
	viewH := min(m.H, minimapMaxH, h-1)                    // the HUD row stays free
	if viewW < min(m.W, minimapMinCells) || viewH < min(m.H, minimapMinCells) {
		return false
	}
	px, py := int(st.Player.X), int(st.Player.Y)
	ox, oy := minimapWindow(m.W, m.H, viewW, viewH, px, py)
	x0 := w - viewW*minimapCellWidth - 1

	for vy := 0; vy < viewH; vy++ {
		for vx := 0; vx < viewW; vx++ {
			x, y := ox+vx, oy+vy // map cell shown at view cell (vx, vy)
			sx := x0 + vx*minimapCellWidth
			switch {
			case x == px && y == py:
				s.SetContent(sx, vy, arrowFor(st.Player.Angle), nil, minimapPlayer)
				s.SetContent(sx+1, vy, ' ', nil, minimapFloor)
			case m.At(x, y) == world.Target:
				s.SetContent(sx, vy, ' ', nil, minimapTarget)
				s.SetContent(sx+1, vy, ' ', nil, minimapTarget)
			case m.At(x, y).Solid():
				s.SetContent(sx, vy, ' ', nil, minimapWall)
				s.SetContent(sx+1, vy, ' ', nil, minimapWall)
			default:
				s.SetContent(sx, vy, ' ', nil, minimapFloor)
				s.SetContent(sx+1, vy, ' ', nil, minimapFloor)
			}
		}
	}
	for _, pk := range st.Pickups { // ammo boxes: a yellow dot
		vx, vy := int(pk.X)-ox, int(pk.Y)-oy
		if vx < 0 || vy < 0 || vx >= viewW || vy >= viewH {
			continue // outside the window
		}
		if int(pk.X) == px && int(pk.Y) == py {
			continue // the player's arrow wins
		}
		s.SetContent(x0+vx*minimapCellWidth, vy, '●', nil, minimapAmmo)
	}
	return true
}

// minimapWindow returns the top-left map cell of a viewW x viewH window on
// a mapW x mapH map, centered on the player's cell (px, py) but kept
// inside the map, so the window stops scrolling at the edges.
func minimapWindow(mapW, mapH, viewW, viewH, px, py int) (ox, oy int) {
	ox = min(max(px-viewW/2, 0), mapW-viewW)
	oy = min(max(py-viewH/2, 0), mapH-viewH)
	return ox, oy
}

// arrowFor picks the arrow closest to the facing angle.
func arrowFor(angle float64) rune {
	return arrows[dirIndex(angle)]
}
