// Package world holds the map: parsing, cell lookup, collision and raycasting.
// It must never import tcell.
package world

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
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
	// ExplosiveBarrel is a red barrel (M14): cover like Barrel, but a shot
	// blows it up.
	ExplosiveBarrel Cell = 'B'

	// Acid is a green floor (M14) that hurts while you stand on it. It's
	// not a prop: nothing is drawn standing on it.
	Acid Cell = '~'

	// Door is a sliding door (M3) between two walls. Its state is in
	// Map.Doors; rays, shots and movement go through it as far as it's open.
	Door Cell = 'D'
)

// IsProp reports whether the cell is a prop (M2).
func (c Cell) IsProp() bool {
	switch c {
	case Barrel, Crate, Pillar, Lamp, FlickerLamp, Debris, Body, ExplosiveBarrel:
		return true
	}
	return false
}

// BlocksMove reports whether a prop in this cell stops the player and
// enemies. Walls aren't props: see Map.Blocked for everything that blocks.
func (c Cell) BlocksMove() bool {
	return c == Barrel || c == Crate || c == Pillar || c == Lamp || c == FlickerLamp || c == ExplosiveBarrel
}

// BlocksShot reports whether a prop in this cell stops bullets and line of
// sight (cover).
func (c Cell) BlocksShot() bool {
	return c == Barrel || c == Crate || c == Pillar || c == ExplosiveBarrel
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

	// Lighting (M4). Fog is the "; fog:" header, 0 if the map has none (the
	// renderer then uses its default). Darkness is "; darkness:", 0 = the
	// usual light. Lamps are the lamp props with the cells they light.
	Fog, Darkness float64
	Lamps         []LightSource
}

// LightSource is a lamp prop (M4) and the floor cells its light reaches.
type LightSource struct {
	X, Y    float64 // center of its cell
	Flicker bool    // an 'L': its light flickers
	Reach   []Lit   // cells it lights, worked out once when the map loads
}

// Lit is how much light a lamp adds to one cell: Weight at full power.
type Lit struct {
	Index  int // y*W + x
	Weight float64
}

// Lamp light (M4): a lamp adds up to LampPower to cells within LampRadius,
// fading to nothing at the edge, if nothing stands between (walls, closed
// doors, and props that block shots cast shadows).
const (
	LampRadius = 4.5
	LampPower  = 0.9
)

// lampReach lists the open cells a lamp at (lx, ly) lights.
func lampReach(m *Map, lx, ly float64) []Lit {
	var out []Lit
	r := int(math.Ceil(LampRadius))
	for y := int(ly) - r; y <= int(ly)+r; y++ {
		for x := int(lx) - r; x <= int(lx)+r; x++ {
			if m.IsSolid(x, y) || m.At(x, y).BlocksShot() {
				continue
			}
			cx, cy := float64(x)+0.5, float64(y)+0.5
			d := math.Hypot(cx-lx, cy-ly)
			if d >= LampRadius || !LineOfSight(m, lx, ly, cx, cy) {
				continue
			}
			f := 1 - d/LampRadius
			out = append(out, Lit{Index: y*m.W + x, Weight: LampPower * f * f})
		}
	}
	return out
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

// ReplaceProp turns the prop in cell (x, y) into c (Empty, or another prop
// such as Debris), keeping Props in step, and works out the lamps' light
// again, since the old prop may have cast a shadow (M14: a red barrel
// blowing up).
func (m *Map) ReplaceProp(x, y int, c Cell) {
	m.Set(x, y, c)
	cx, cy := float64(x)+0.5, float64(y)+0.5
	for i, p := range m.Props {
		if p.X == cx && p.Y == cy {
			m.Props = append(m.Props[:i], m.Props[i+1:]...)
			break
		}
	}
	if c.IsProp() {
		m.Props = append(m.Props, Prop{Kind: c, X: cx, Y: cy})
	}
	for i := range m.Lamps {
		m.Lamps[i].Reach = lampReach(m, m.Lamps[i].X, m.Lamps[i].Y)
	}
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
	c.Lamps = append([]LightSource(nil), m.Lamps...) // ReplaceProp (M14) changes their Reach per run
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
	hdr, off, err := parseHeaders(lines)
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
	m := &Map{Name: hdr.name, W: w, H: h, Cells: make([]Cell, 0, w*h), Fog: hdr.fog, Darkness: hdr.darkness}
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
	// Lamps last: their light needs the doors (closed ones cast shadows).
	for _, p := range m.Props {
		if p.Kind == Lamp || p.Kind == FlickerLamp {
			m.Lamps = append(m.Lamps, LightSource{X: p.X, Y: p.Y, Flicker: p.Kind == FlickerLamp, Reach: lampReach(m, p.X, p.Y)})
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
	case Wall, Stone, Metal, Empty, Spawn, Target, ZombieStart, ShooterStart, Door, Acid:
		return nil
	}
	if Cell(line[x]).IsProp() {
		return nil
	}
	// Decode the full character so the message shows it properly,
	// even if it is a multi-byte UTF-8 character like 'é'.
	r, _ := utf8.DecodeRuneInString(line[x:])
	return fmt.Errorf("row %d, column %d: unknown character %q (use '#', '%%', '&', '.', 'P', 'T', 'Z', 'S', 'D', '~', or a prop: 'b', 'B', 'c', 'i', 'l', 'L', ',', 'x')", y+1, x+1, r)
}

// headers are the values of a map file's header lines.
type headers struct {
	name           string
	fog, darkness  float64
	hasFog, hasDrk bool
}

// parseHeaders reads the header lines at the top of a map file:
// "; name: The Arena", and for lighting (M4) "; fog: 0.4" (how fast things
// fade with distance, 0-2; without it the game's default) and
// "; darkness: 0.6" (0 = normal light, up to 0.95 = only lamps light the
// map). It returns them and how many lines were headers. An unknown or
// repeated header is an error, so a typo doesn't go unseen.
func parseHeaders(lines []string) (h headers, n int, err error) {
	for n < len(lines) && strings.HasPrefix(lines[n], ";") {
		// Cut splits at the first ":"; found is false if there is none.
		key, value, found := strings.Cut(strings.TrimPrefix(lines[n], ";"), ":")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !found {
			return h, 0, fmt.Errorf("row %d: unknown header %q (headers are \"; name: ...\", \"; fog: ...\", \"; darkness: ...\")", n+1, lines[n])
		}
		switch key {
		case "name":
			switch {
			case value == "":
				return h, 0, fmt.Errorf("row %d: map name is empty", n+1)
			case h.name != "":
				return h, 0, fmt.Errorf("row %d: second name header (the map already has the name %q)", n+1, h.name)
			}
			h.name = value
		case "fog", "darkness":
			v, perr := strconv.ParseFloat(value, 64)
			lo, hi := 0.0, 2.0
			if key == "darkness" {
				hi = 0.95
			}
			if perr != nil || v < lo || v > hi {
				return h, 0, fmt.Errorf("row %d: %s must be a number from %v to %v, got %q", n+1, key, lo, hi, value)
			}
			if (key == "fog" && h.hasFog) || (key == "darkness" && h.hasDrk) {
				return h, 0, fmt.Errorf("row %d: second %s header", n+1, key)
			}
			if key == "fog" {
				h.fog, h.hasFog = v, true
			} else {
				h.darkness, h.hasDrk = v, true
			}
		default:
			return h, 0, fmt.Errorf("row %d: unknown header %q (headers are \"; name: ...\", \"; fog: ...\", \"; darkness: ...\")", n+1, lines[n])
		}
		n++
	}
	return h, n, nil
}

// HasAcid reports whether the map has acid floor (M14), so the renderer
// can skip looking for it on maps without.
func (m *Map) HasAcid() bool {
	for _, c := range m.Cells {
		if c == Acid {
			return true
		}
	}
	return false
}
