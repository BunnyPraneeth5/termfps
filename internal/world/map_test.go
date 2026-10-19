package world

import (
	"strings"
	"testing"
)

const smallMap = "#####\n" +
	"#P..#\n" +
	"#.#.#\n" +
	"#####\n"

func TestParseValid(t *testing.T) {
	m, err := Parse([]byte(smallMap))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.W != 5 || m.H != 4 {
		t.Errorf("size = %dx%d, want 5x4", m.W, m.H)
	}
	if len(m.Cells) != 20 {
		t.Errorf("len(Cells) = %d, want 20", len(m.Cells))
	}
	if m.SpawnX != 1.5 || m.SpawnY != 1.5 {
		t.Errorf("spawn = (%v, %v), want (1.5, 1.5)", m.SpawnX, m.SpawnY)
	}
	if got := m.Cells[2*m.W+2]; got != Wall {
		t.Errorf("cell (2,2) = %q, want '#'", got)
	}
}

// These inputs are the same map in different file shapes; all must parse.
func TestParseFileShapes(t *testing.T) {
	cases := map[string]string{
		"CRLF line endings":   strings.ReplaceAll(smallMap, "\n", "\r\n"),
		"UTF-8 BOM":           "\xEF\xBB\xBF" + smallMap,
		"BOM and CRLF":        "\xEF\xBB\xBF" + strings.ReplaceAll(smallMap, "\n", "\r\n"),
		"no final newline":    strings.TrimSuffix(smallMap, "\n"),
		"extra newlines":      smallMap + "\n\n\n",
		"extra CRLF newlines": strings.ReplaceAll(smallMap, "\n", "\r\n") + "\r\n\r\n",
		"trailing spaces":     strings.ReplaceAll(smallMap, "\n", "  \t\n"),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			m, err := Parse([]byte(input))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if m.W != 5 || m.H != 4 {
				t.Errorf("size = %dx%d, want 5x4", m.W, m.H)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr string // a piece of the expected error message
	}{
		{"empty file", "", "empty"},
		{"only newlines", "\n\r\n\n", "empty"},
		{"too small", "###\n#P#\n", "at least 3x3"},
		{"ragged row", "#####\n#P..#\n#..#\n#####\n", "row 3 has 4 columns"},
		{"unknown char", "#####\n#P.x#\n#...#\n#####\n", "row 2, column 4: unknown character 'x'"},
		{"unicode char", "#####\n#P.é#\n#...#\n#####\n", "unknown character 'é'"},
		{"tab inside row", "#####\n#P\t.#\n#...#\n#####\n", "unknown character '\\t'"},
		{"target T in P0", "#####\n#P.T#\n#...#\n#####\n", "'T' targets are not supported"},
		{"no spawn", "#####\n#...#\n#...#\n#####\n", "0 spawn points"},
		{"two spawns", "#####\n#P.P#\n#...#\n#####\n", "2 spawn points"},
		{"open top border", "##.##\n#P..#\n#...#\n#####\n", "row 1, column 3: border"},
		{"open side border", "#####\n.P..#\n#...#\n#####\n", "row 2, column 1: border"},
		{"spawn on border", "##P##\n#...#\n#...#\n#####\n", "border"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse([]byte(c.input))
			if err == nil {
				t.Fatalf("Parse succeeded, want error containing %q", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, c.wantErr)
			}
		})
	}
}
