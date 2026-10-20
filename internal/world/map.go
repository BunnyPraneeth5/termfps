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
)

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
	W, H           int
	Cells          []Cell  // row-major, index = y*W + x
	SpawnX, SpawnY float64 // center of the P cell (x+0.5, y+0.5)
	EnemyStarts    []EnemyStart
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
	if len(lines) == 0 {
		return nil, errors.New("map is empty")
	}

	// Pass 1: every character is known and every row has the same width.
	w, h := len(lines[0]), len(lines)
	for y, line := range lines {
		for x := 0; x < len(line); x++ {
			if err := checkChar(line, x, y); err != nil {
				return nil, err
			}
		}
		if len(line) != w {
			return nil, fmt.Errorf("row %d has %d columns, expected %d (all rows must be the same length)", y+1, len(line), w)
		}
	}
	if w < minSize || h < minSize {
		return nil, fmt.Errorf("map is %dx%d, must be at least %dx%d", w, h, minSize, minSize)
	}

	// Pass 2: build cells, check the border, find the spawn.
	m := &Map{W: w, H: h, Cells: make([]Cell, 0, w*h)}
	spawns := 0
	for y, line := range lines {
		for x := 0; x < w; x++ {
			c := Cell(line[x])
			onBorder := x == 0 || y == 0 || x == w-1 || y == h-1
			if onBorder && !c.IsWall() {
				return nil, fmt.Errorf("row %d, column %d: border must be a wall ('#', '%%' or '&'), found %q", y+1, x+1, line[x])
			}
			if c == Spawn {
				spawns++
				m.SpawnX, m.SpawnY = float64(x)+0.5, float64(y)+0.5
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

// IsSolid reports whether (x, y) blocks movement and rays. Outside the map
// counts as solid.
func (m *Map) IsSolid(x, y int) bool {
	return m.At(x, y).Solid()
}

// checkChar returns an error if the byte at line[x] is not a known map char.
func checkChar(line string, x, y int) error {
	switch Cell(line[x]) {
	case Wall, Stone, Metal, Empty, Spawn, Target, ZombieStart, ShooterStart:
		return nil
	}
	// Decode the full character so the message shows it properly,
	// even if it is a multi-byte UTF-8 character like 'é'.
	r, _ := utf8.DecodeRuneInString(line[x:])
	return fmt.Errorf("row %d, column %d: unknown character %q (use '#', '%%', '&', '.', 'P', 'T', 'Z' or 'S')", y+1, x+1, r)
}
