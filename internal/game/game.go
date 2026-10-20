// Package game holds the player and game state. It must never import tcell.
package game

import (
	"math"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// PlaneLen is the camera plane length. 0.66 gives about a 66 degree FOV.
const PlaneLen = 0.66

// Player is the camera in the world. See ARCHITECTURE.md "Coordinates":
// y points down, Angle 0 faces east, a growing Angle turns right (clockwise).
type Player struct {
	X, Y   float64 // position in map units
	Angle  float64 // radians, kept in [0, 2*pi)
	Weapon Weapon
}

// Dir returns the unit facing direction.
func (p Player) Dir() (x, y float64) {
	return math.Cos(p.Angle), math.Sin(p.Angle)
}

// Plane returns the camera plane: perpendicular to Dir, pointing to the
// camera's right, with length PlaneLen.
func (p Player) Plane() (x, y float64) {
	return -math.Sin(p.Angle) * PlaneLen, math.Cos(p.Angle) * PlaneLen
}

// State is everything the game needs to update and draw one frame.
type State struct {
	Map         *world.Map
	Player      Player
	ShowMinimap bool
	Mode        RenderMode
	// Events lists what happened in the last Update. Update empties it first,
	// reusing the slice so there is no allocation per frame.
	Events []Event
}

// NewState puts the player on the map's spawn, facing east, minimap on.
func NewState(m *world.Map) *State {
	return &State{
		Map:         m,
		Player:      Player{X: m.SpawnX, Y: m.SpawnY, Angle: 0, Weapon: NewWeapon()},
		ShowMinimap: true,
	}
}
