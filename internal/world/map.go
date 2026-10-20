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
	Wall   Cell = '#'
	Empty  Cell = '.'
	Spawn  Cell = 'P'
	Target Cell = 'T' // red block, solid; shooting removes it (P1-03)
)

// Solid reports whether the cell blocks movement and rays.
func (c Cell) Solid() bool {
	return c == Wall || c == Target
}

// Map is a parsed, validated map.
type Map struct {
	W, H           int
	Cells          []Cell  // row-major, index = y*W + x
	SpawnX, SpawnY float64 // center of the P cell (x+0.5, y+0.5)
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
			if onBorder && c != Wall {
				return nil, fmt.Errorf("row %d, column %d: border must be '#', found %q", y+1, x+1, line[x])
			}
			if c == Spawn {
				spawns++
				m.SpawnX, m.SpawnY = float64(x)+0.5, float64(y)+0.5
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

// IsSolid reports whether (x, y) blocks movement and rays. Outside the map
// counts as solid.
func (m *Map) IsSolid(x, y int) bool {
	return m.At(x, y).Solid()
}

// checkChar returns an error if the byte at line[x] is not a known map char.
func checkChar(line string, x, y int) error {
	switch Cell(line[x]) {
	case Wall, Empty, Spawn, Target:
		return nil
	}
	// Decode the full character so the message shows it properly,
	// even if it is a multi-byte UTF-8 character like 'é'.
	r, _ := utf8.DecodeRuneInString(line[x:])
	return fmt.Errorf("row %d, column %d: unknown character %q (use '#', '.', 'P' or 'T')", y+1, x+1, r)
}
