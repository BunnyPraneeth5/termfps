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

// DrawMinimap draws the map in the top-right corner, on top of the 3D view.
// It returns false (and draws nothing) when the screen is too small to fit
// it above the HUD row.
func DrawMinimap(s tcell.Screen, st *game.State) bool {
	m := st.Map
	w, h := s.Size()
	mapCols := m.W * minimapCellWidth
	if mapCols+1 > w || m.H > h-1 {
		return false
	}
	x0 := w - mapCols - 1 // one column of margin on the right

	px, py := int(st.Player.X), int(st.Player.Y)
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			sx := x0 + x*minimapCellWidth
			switch {
			case x == px && y == py:
				s.SetContent(sx, y, arrowFor(st.Player.Angle), nil, minimapPlayer)
				s.SetContent(sx+1, y, ' ', nil, minimapFloor)
			case m.At(x, y) == world.Target:
				s.SetContent(sx, y, ' ', nil, minimapTarget)
				s.SetContent(sx+1, y, ' ', nil, minimapTarget)
			case m.At(x, y).Solid():
				s.SetContent(sx, y, ' ', nil, minimapWall)
				s.SetContent(sx+1, y, ' ', nil, minimapWall)
			default:
				s.SetContent(sx, y, ' ', nil, minimapFloor)
				s.SetContent(sx+1, y, ' ', nil, minimapFloor)
			}
		}
	}
	for _, pk := range st.Pickups { // ammo boxes: a yellow dot
		x, y := int(pk.X), int(pk.Y)
		if x == px && y == py {
			continue // the player's arrow wins
		}
		s.SetContent(x0+x*minimapCellWidth, y, '●', nil, minimapAmmo)
	}
	return true
}

// arrowFor picks the arrow closest to the facing angle.
func arrowFor(angle float64) rune {
	return arrows[dirIndex(angle)]
}
