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
	// Bob is the walking bob phase in radians, [0, 2*pi). It advances with
	// distance walked; the gun graphic sways with it.
	Bob float64
	HP  int // 0 = dead

	hurt float64 // seconds left of the red "you got hit" flash
}

// Hurting reports whether the player was just hit (red screen edges).
func (p Player) Hurting() bool {
	return p.hurt > 0
}

// Dead reports whether the player has no HP left.
func (p Player) Dead() bool {
	return p.HP <= 0
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

	Enemies []Enemy
	Kills   int     // enemies killed this run
	Time    float64 // seconds survived this run

	// dist is the distance field from the player's cell (distCell), used by
	// enemies to walk around walls. -1 = not computed yet.
	dist     []int
	distCell int
}

// NewState puts the player on the map's spawn, facing east, minimap on,
// with full health and the map's starting enemies.
func NewState(m *world.Map) *State {
	s := &State{
		Map:         m,
		Player:      Player{X: m.SpawnX, Y: m.SpawnY, Angle: 0, Weapon: NewWeapon(), HP: PlayerMaxHP},
		ShowMinimap: true,
		distCell:    -1,
	}
	for _, e := range m.EnemyStarts {
		if e.Kind == world.ZombieStart {
			s.Enemies = append(s.Enemies, newZombie(e.X, e.Y))
		}
	}
	return s
}
