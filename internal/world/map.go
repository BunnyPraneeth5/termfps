// Package world holds the map: parsing, cell lookup, collision and raycasting.
// It must never import tcell.
package world

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Cell is one map square, stored as the character from the map file.
type Cell byte

const (
	Wall   Cell = '#' // brick wall (P1-10 texture)
	Stone  Cell = '%' // stone wall
	Metal  Cell = '&' // metal wall
	Empty  Cell = '.'
	Spawn  Cell = 'P'
	Target Cell = 'T' // red block, solid; shooting removes it (P1-03)
	// ZombieStart marks where a zombie starts (P1-06). After parsing, the
	// cell is floor and the position is in Map.EnemyStarts.
	ZombieStart Cell = 'Z'
	// ShooterStart marks where a shooter starts (P1-07), like ZombieStart.
	ShooterStart Cell = 'S'

	// Props (M2) stay in the map as their character and are drawn as
	// sprites; rays (walls) see past them. Some block movement and shots.
	Barrel      Cell = 'b' // blocks movement and shots
	Crate       Cell = 'c' // blocks movement and shots
	Pillar      Cell = 'i' // blocks movement and shots
	Lamp        Cell = 'l' // a standing lamp: blocks movement, not shots
	FlickerLamp Cell = 'L' // a lamp that flickers (M4)
	Debris      Cell = ',' // rubble: blocks nothing
	Body        Cell = 'x' // a body on the floor: blocks nothing

	// Door is a sliding door (M3) between two walls. Its state is in
	// Map.Doors; rays, shots and movement go through it as far as it's open.
	Door Cell = 'D'
)

// IsProp reports whether the cell is a prop (M2).
func (c Cell) IsProp() bool {
	switch c {
	case Barrel, Crate, Pillar, Lamp, FlickerLamp, Debris, Body:
		return true
	}
	return false
}

// BlocksMove reports whether a prop in this cell stops the player and
// enemies. Walls aren't props: see Map.Blocked for everything that blocks.
func (c Cell) BlocksMove() bool {
	return c == Barrel || c == Crate || c == Pillar || c == Lamp || c == FlickerLamp
}

// BlocksShot reports whether a prop in this cell stops bullets and line of
// sight (cover).
func (c Cell) BlocksShot() bool {
	return c == Barrel || c == Crate || c == Pillar
}

// IsWall reports whether the cell is a permanent wall (any texture).
func (c Cell) IsWall() bool {
	return c == Wall || c == Stone || c == Metal
}

// Solid reports whether the cell blocks movement and rays.
func (c Cell) Solid() bool {
	return c.IsWall() || c == Target
}

// Map is a parsed, validated map.
type Map struct {
	Name           string // from the "; name:" header; "" if there is none
	W, H           int
	Cells          []Cell  // row-major, index = y*W + x
	SpawnX, SpawnY float64 // center of the P cell (x+0.5, y+0.5)
	EnemyStarts    []EnemyStart
	Props          []Prop        // every prop cell, in map order (M2)
	Doors          []SlidingDoor // every door, in map order (M3)
}

// SlidingDoor is a door in cell (X, Y). It's a thin panel through the
// middle of the cell, between walls on two opposite sides, that slides
// sideways into the wall as it opens.
type SlidingDoor struct {
	X, Y int
	// Across is true when the walls are left and right of the door, so the
	// panel runs east-west and you walk through it north-south; false when
	// the walls are above and below.
	Across bool
	// Open is how far it has slid open, 0 (closed) to 1 (open).
	Open float64
	// Hold is how many more seconds it stays open; the game sets it while
	// someone is near (game.updateDoors).
	Hold float64
}

// DoorPassable is how far open a door must be to walk through it.
const DoorPassable = 0.8

// DoorAt returns the door in cell (x, y), or nil if there is none. Maps
// have few doors, so a scan is quick.
func (m *Map) DoorAt(x, y int) *SlidingDoor {
	for i := range m.Doors {
		if m.Doors[i].X == x && m.Doors[i].Y == y {
			return &m.Doors[i] // a pointer into the slice, so callers can change it
		}
	}
	return nil
}

// Prop is a prop placed in the map file; its cell keeps the character.
type Prop struct {
	Kind Cell    // the map character, e.g. Crate
	X, Y float64 // center of its cell
}

// EnemyStart is an enemy placed in the map file.
type EnemyStart struct {
	Kind Cell    // the map character, e.g. ZombieStart
	X, Y float64 // center of its cell
}

// Clone returns a deep copy, so a new run can start from the original map
// after shots changed the old one.
func (m *Map) Clone() *Map {
	c := *m                                   // copies W, H, spawn; the slices below still share memory...
	c.Cells = append([]Cell(nil), m.Cells...) // ...so copy them
	c.EnemyStarts = append([]EnemyStart(nil), m.EnemyStarts...)
	c.Props = append([]Prop(nil), m.Props...)
	c.Doors = append([]SlidingDoor(nil), m.Doors...) // doors open and close per run
	return &c
}

// minSize is the smallest map side: a border on both sides plus one open cell.
const minSize = 3

// Parse reads a map file. See ARCHITECTURE.md "Map format" for the rules.
// Errors use 1-based row and column numbers, like a text editor.
func Parse(data []byte) (*Map, error) {
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF")) // UTF-8 BOM (Notepad)
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	// Header lines start with ";" and come before the grid. off counts them,
	// so error messages still give the row as the file's line number.
	name, off, err := parseHeaders(lines)
	if err != nil {
		return nil, err
	}
	lines = lines[off:]
	if len(lines) == 0 {
		return nil, errors.New("map is empty")
	}

	// Pass 1: every character is known and every row has the same width.
	w, h := len(lines[0]), len(lines)
	for y, line := range lines {
		for x := 0; x < len(line); x++ {
			if err := checkChar(line, x, y+off); err != nil {
				return nil, err
			}
		}
		if len(line) != w {
			return nil, fmt.Errorf("row %d has %d columns, expected %d (all rows must be the same length)", y+off+1, len(line), w)
		}
	}
	if w < minSize || h < minSize {
		return nil, fmt.Errorf("map is %dx%d, must be at least %dx%d", w, h, minSize, minSize)
	}

	// Pass 2: build cells, check the border, find the spawn.
	m := &Map{Name: name, W: w, H: h, Cells: make([]Cell, 0, w*h)}
	spawns := 0
	for y, line := range lines {
		for x := 0; x < w; x++ {
			c := Cell(line[x])
			onBorder := x == 0 || y == 0 || x == w-1 || y == h-1
			if onBorder && !c.IsWall() {
				return nil, fmt.Errorf("row %d, column %d: border must be a wall ('#', '%%' or '&'), found %q", y+off+1, x+1, line[x])
			}
			if c == Spawn {
				spawns++
				m.SpawnX, m.SpawnY = float64(x)+0.5, float64(y)+0.5
			}
			if c.IsProp() {
				m.Props = append(m.Props, Prop{Kind: c, X: float64(x) + 0.5, Y: float64(y) + 0.5})
			}
			if c == ZombieStart || c == ShooterStart {
				m.EnemyStarts = append(m.EnemyStarts, EnemyStart{Kind: c, X: float64(x) + 0.5, Y: float64(y) + 0.5})
				c = Empty
			}
			m.Cells = append(m.Cells, c)
		}
	}
	if spawns != 1 {
		return nil, fmt.Errorf("map has %d spawn points 'P', expected exactly 1", spawns)
	}
	// Doors need walls on two opposite sides to slide into.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if m.At(x, y) != Door {
				continue
			}
			across := m.At(x-1, y).IsWall() && m.At(x+1, y).IsWall()
			if !across && !(m.At(x, y-1).IsWall() && m.At(x, y+1).IsWall()) {
				return nil, fmt.Errorf("row %d, column %d: a door 'D' needs walls on both sides (left and right, or above and below)", y+off+1, x+1)
			}
			m.Doors = append(m.Doors, SlidingDoor{X: x, Y: y, Across: across})
		}
	}
	return m, nil
}

// At returns the cell at (x, y). Outside the map it returns Wall, so rays
// and collision treat the outside as solid.
func (m *Map) At(x, y int) Cell {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return Wall
	}
	return m.Cells[y*m.W+x]
}

// Set changes the cell at (x, y), e.g. when a shot destroys a target.
// It does nothing outside the map.
func (m *Map) Set(x, y int, c Cell) {
	if x < 0 || y < 0 || x >= m.W || y >= m.H {
		return
	}
	m.Cells[y*m.W+x] = c
}

// IsSolid reports whether (x, y) blocks rays: a wall or a target block.
// Outside the map counts as solid.
func (m *Map) IsSolid(x, y int) bool {
	return m.At(x, y).Solid()
}

// Blocked reports whether (x, y) stops the player and enemies: anything
// solid, a prop that blocks movement (M2), or a door that isn't open far
// enough (M3).
func (m *Map) Blocked(x, y int) bool {
	if m.At(x, y) == Door {
		d := m.DoorAt(x, y)
		return d == nil || d.Open < DoorPassable
	}
	return m.pathBlocked(x, y)
}

// pathBlocked is Blocked without doors: enemies plan their way through
// doors, which open when they come near.
func (m *Map) pathBlocked(x, y int) bool {
	c := m.At(x, y)
	return c.Solid() || c.BlocksMove()
}

// checkChar returns an error if the byte at line[x] is not a known map char.
func checkChar(line string, x, y int) error {
	switch Cell(line[x]) {
	case Wall, Stone, Metal, Empty, Spawn, Target, ZombieStart, ShooterStart, Door:
		return nil
	}
	if Cell(line[x]).IsProp() {
		return nil
	}
	// Decode the full character so the message shows it properly,
	// even if it is a multi-byte UTF-8 character like 'é'.
	r, _ := utf8.DecodeRuneInString(line[x:])
	return fmt.Errorf("row %d, column %d: unknown character %q (use '#', '%%', '&', '.', 'P', 'T', 'Z', 'S', 'D', or a prop: 'b', 'c', 'i', 'l', 'L', ',', 'x')", y+1, x+1, r)
}

// parseHeaders reads the header lines at the top of a map file. The only
// header is "; name: The Arena". It returns the name and how many lines
// were headers. An unknown header is an error, so a typo doesn't go unseen.
func parseHeaders(lines []string) (name string, n int, err error) {
	for n < len(lines) && strings.HasPrefix(lines[n], ";") {
		// Cut splits at the first ":"; found is false if there is none.
		key, value, found := strings.Cut(strings.TrimPrefix(lines[n], ";"), ":")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch {
		case !found || key != "name":
			return "", 0, fmt.Errorf("row %d: unknown header %q (the only header is \"; name: ...\")", n+1, lines[n])
		case value == "":
			return "", 0, fmt.Errorf("row %d: map name is empty", n+1)
		case name != "":
			return "", 0, fmt.Errorf("row %d: second name header (the map already has the name %q)", n+1, name)
		}
		name = value
		n++
	}
	return name, n, nil
}
