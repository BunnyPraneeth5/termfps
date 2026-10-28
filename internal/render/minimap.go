package render

import (
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Minimap colors: light walls (all textures alike) on a dark floor, white
// player arrow.
var (
	minimapWall   = term.Style{Bg: grayColor(140)}
	minimapTarget = term.Style{Bg: rgbColor(170, TintRed)}
	minimapFloor  = term.Style{Bg: grayColor(25)}
	minimapProp   = term.Style{Bg: grayColor(75)} // crates, barrels, pillars, lamps
	minimapPlayer = term.Style{Bg: grayColor(25), Fg: grayColor(255), Bold: true}
	minimapAmmo   = term.Style{Bg: grayColor(25), Fg: rgbColor(230, TintYellow)}
	// The border (M1) is a light line on black, so the minimap stands apart
	// from the 3D view behind it.
	minimapBorder = term.Style{Bg: grayColor(0), Fg: grayColor(170)}
)

// arrows point in 8 directions, starting east and turning clockwise on
// screen (y points down): E, SE, S, SW, W, NW, N, NE.
var arrows = [8]string{"→", "↘", "↓", "↙", "←", "↖", "↑", "↗"}

// minimapCellWidth: each map cell is 2 columns wide, so cells look about
// square (terminal cells are about twice as tall as wide).
const minimapCellWidth = 2

// The minimap shows at most this many map cells across and down, so on big
// maps it stays a corner of the view and scrolls with the player (D26).
// Below minimapMinCells either way it isn't worth drawing.
const (
	minimapMaxW     = 16 // was 24 before M1
	minimapMaxH     = 16
	minimapMinCells = 5
)

// DrawMinimap draws the map in the top-right corner, on top of the 3D view,
// inside a one-cell border (M1). When the map is bigger than the space it
// gets, it shows a window around the player. It returns false (and draws
// nothing) when the screen is too small for even a small window above the
// HUD row.
func DrawMinimap(s term.Screen, st *game.State) bool {
	m := st.Map
	w, h := s.Size()
	viewW := min(m.W, minimapMaxW, (w-2)/minimapCellWidth) // the border's two columns
	viewH := min(m.H, minimapMaxH, h-3)                    // the border's two rows, and the HUD row
	if viewW < min(m.W, minimapMinCells) || viewH < min(m.H, minimapMinCells) {
		return false
	}
	px, py := int(st.Player.X), int(st.Player.Y)
	ox, oy := minimapWindow(m.W, m.H, viewW, viewH, px, py)
	x0 := w - viewW*minimapCellWidth - 1 // the right border takes the last column
	drawMinimapBorder(s, x0-1, 0, viewW*minimapCellWidth+2, viewH+2)
	const y0 = 1 // map rows start under the top border

	for vy := 0; vy < viewH; vy++ {
		for vx := 0; vx < viewW; vx++ {
			x, y := ox+vx, oy+vy // map cell shown at view cell (vx, vy)
			sx := x0 + vx*minimapCellWidth
			switch {
			case x == px && y == py:
				s.Put(sx, y0+vy, arrowFor(st.Player.Angle), minimapPlayer)
				s.Put(sx+1, y0+vy, " ", minimapFloor)
			case m.At(x, y) == world.Target:
				s.Put(sx, y0+vy, " ", minimapTarget)
				s.Put(sx+1, y0+vy, " ", minimapTarget)
			case m.At(x, y).BlocksMove(): // a prop you can't walk through (M2)
				s.Put(sx, y0+vy, " ", minimapProp)
				s.Put(sx+1, y0+vy, " ", minimapProp)
			case m.At(x, y).Solid():
				s.Put(sx, y0+vy, " ", minimapWall)
				s.Put(sx+1, y0+vy, " ", minimapWall)
			default:
				s.Put(sx, y0+vy, " ", minimapFloor)
				s.Put(sx+1, y0+vy, " ", minimapFloor)
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
		s.Put(x0+vx*minimapCellWidth, y0+vy, "●", minimapAmmo)
	}
	return true
}

// drawMinimapBorder draws a box w cells wide and h tall with its top-left
// corner at (x, y).
func drawMinimapBorder(s term.Screen, x, y, w, h int) {
	for i := 1; i < w-1; i++ {
		s.Put(x+i, y, "─", minimapBorder)
		s.Put(x+i, y+h-1, "─", minimapBorder)
	}
	for j := 1; j < h-1; j++ {
		s.Put(x, y+j, "│", minimapBorder)
		s.Put(x+w-1, y+j, "│", minimapBorder)
	}
	s.Put(x, y, "┌", minimapBorder)
	s.Put(x+w-1, y, "┐", minimapBorder)
	s.Put(x, y+h-1, "└", minimapBorder)
	s.Put(x+w-1, y+h-1, "┘", minimapBorder)
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
func arrowFor(angle float64) string {
	return arrows[dirIndex(angle)]
}
