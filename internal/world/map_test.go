package world

import (
	"math"
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
		{"unknown char", "#####\n#P.?#\n#...#\n#####\n", "row 2, column 4: unknown character '?'"},
		{"unicode char", "#####\n#P.é#\n#...#\n#####\n", "unknown character 'é'"},
		{"tab inside row", "#####\n#P\t.#\n#...#\n#####\n", "unknown character '\\t'"},
		{"target T on the border", "##T##\n#P..#\n#...#\n#####\n", "row 1, column 3: border"},
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

// Stone and metal are walls too: solid, and allowed on the border.
func TestStoneAndMetalWalls(t *testing.T) {
	m, err := Parse([]byte("%%&##\n%P.&#\n#%..#\n&&&%#\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, c := range []Cell{Wall, Stone, Metal} {
		if !c.IsWall() || !c.Solid() {
			t.Errorf("%q: IsWall %v, Solid %v; want both true", c, c.IsWall(), c.Solid())
		}
	}
	if !m.IsSolid(3, 1) || !m.IsSolid(1, 2) {
		t.Error("interior stone/metal cells aren't solid")
	}
	if Target.IsWall() {
		t.Error("a target block counts as a permanent wall")
	}
}

// An optional "; name:" header names the map; error rows still match the
// file's lines.
func TestParseNameHeader(t *testing.T) {
	m, err := Parse([]byte("\xEF\xBB\xBF;  name:  The Arena \r\n#####\r\n#P..#\r\n#####\r\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if m.Name != "The Arena" || m.W != 5 || m.H != 3 {
		t.Errorf("name %q, %dx%d; want \"The Arena\", 5x3", m.Name, m.W, m.H)
	}
	if m.SpawnY != 1.5 {
		t.Errorf("spawn y = %v, want 1.5 (the header is not a map row)", m.SpawnY)
	}
	if m, err := Parse([]byte("###\n#P#\n###\n")); err != nil || m.Name != "" {
		t.Errorf("no header: name %q, err %v; want empty name, no error", m.Name, err)
	}
}

func TestParseHeaderErrors(t *testing.T) {
	cases := []struct{ name, input, wantErr string }{
		{"unknown key", "; size: big\n###\n#P#\n###\n", "row 1: unknown header"},
		{"no colon", "; hello\n###\n#P#\n###\n", "unknown header"},
		{"empty name", "; name:\n###\n#P#\n###\n", "name is empty"},
		{"two names", "; name: A\n; name: B\n###\n#P#\n###\n", "row 2: second name"},
		{"only a header", "; name: A\n", "empty"},
		{"row numbers count the header", "; name: A\n#####\n#P.?#\n#####\n", "row 3, column 4"},
		{"fog too high", "; fog: 3\n###\n#P#\n###\n", "row 1: fog must be a number from 0 to 2"},
		{"darkness not a number", "; darkness: dim\n###\n#P#\n###\n", "darkness must be a number"},
		{"two fogs", "; fog: 0.3\n; fog: 0.4\n###\n#P#\n###\n", "row 2: second fog"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse([]byte(c.input))
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, c.wantErr)
			}
		})
	}
}

// Props (M2) stay in the map as their character and are listed in order.
func TestParseProps(t *testing.T) {
	m, err := Parse([]byte("#########\n#Pbcil,x#\n#..L....#\n#########\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Cell{Barrel, Crate, Pillar, Lamp, Debris, Body, FlickerLamp}
	if len(m.Props) != len(want) {
		t.Fatalf("props = %+v, want %d", m.Props, len(want))
	}
	for i, k := range want {
		if m.Props[i].Kind != k {
			t.Errorf("prop %d = %q, want %q", i, m.Props[i].Kind, k)
		}
	}
	if p := m.Props[0]; p.X != 2.5 || p.Y != 1.5 || m.At(2, 1) != Barrel {
		t.Errorf("barrel at (%v, %v), cell %q; want (2.5, 1.5), 'b'", p.X, p.Y, m.At(2, 1))
	}
	// Movement: blocking props stop it, debris and bodies don't; rays see past all.
	for _, c := range []struct {
		x       int
		blocked bool
	}{{2, true}, {3, true}, {4, true}, {5, true}, {6, false}, {7, false}} {
		if got := m.Blocked(c.x, 1); got != c.blocked {
			t.Errorf("Blocked(%d, 1) = %v, want %v", c.x, got, c.blocked)
		}
		if m.IsSolid(c.x, 1) {
			t.Errorf("IsSolid(%d, 1): props must not stop rays", c.x)
		}
	}
	if !m.Blocked(0, 0) {
		t.Error("a wall isn't Blocked")
	}
}

// Blocking props are left out of the distance field (enemies walk around them).
func TestDistanceFieldAvoidsProps(t *testing.T) {
	m, err := Parse([]byte("#######\n#P.c..#\n#.....#\n#######\n"))
	if err != nil {
		t.Fatal(err)
	}
	d := DistanceField(m, 1, 1, nil)
	if d[1*m.W+3] != -1 {
		t.Errorf("crate cell distance = %d, want -1", d[1*m.W+3])
	}
	if d[1*m.W+4] != 5 { // around the crate through row 2
		t.Errorf("cell past the crate = %d, want 5 (the way around)", d[1*m.W+4])
	}
}

// A door needs walls on two opposite sides; which pair sets its direction (M3).
func TestParseDoors(t *testing.T) {
	m, err := Parse([]byte("#####\n#P..#\n##D##\n#...#\n#.#.#\n#.D.#\n#.#.#\n#####\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Doors) != 2 || !m.Doors[0].Across || m.Doors[1].Across {
		t.Fatalf("doors = %+v, want one across (walls left and right), one not", m.Doors)
	}
	if d := m.DoorAt(2, 5); d == nil || d != &m.Doors[1] {
		t.Error("DoorAt(2, 5) didn't find the second door")
	}
	if _, err := Parse([]byte("#####\n#P..#\n#.D.#\n#####\n")); err == nil || !strings.Contains(err.Error(), "row 3, column 3: a door") {
		t.Errorf("door without walls: error %v, want one naming row 3, column 3", err)
	}
	// Closed doors block movement, open ones don't; paths go through either.
	if !m.Blocked(2, 2) {
		t.Error("closed door isn't Blocked")
	}
	m.Doors[0].Open = 0.9
	if m.Blocked(2, 2) {
		t.Error("door 90% open is still Blocked")
	}
	m.Doors[0].Open = 0
	if d := DistanceField(m, 1, 1, nil); d[3*m.W+2] < 0 {
		t.Error("distance field doesn't go through the closed door")
	}
}

// Lighting headers (M4) set the map's fog and darkness.
func TestParseLightHeaders(t *testing.T) {
	m, err := Parse([]byte("; name: Dark\n; fog: 0.4\n; darkness: 0.6\n###\n#P#\n###\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Dark" || m.Fog != 0.4 || m.Darkness != 0.6 {
		t.Errorf("name %q fog %v darkness %v, want Dark, 0.4, 0.6", m.Name, m.Fog, m.Darkness)
	}
}

// A lamp lights open cells within LampRadius it can see, brighter close
// up; walls and cover props cast shadows (M4).
func TestLampReach(t *testing.T) {
	m, err := Parse([]byte("##########\n" +
		"#P.......#\n" +
		"#...l.c..#\n" +
		"#........#\n" +
		"######...#\n" +
		"#........#\n" +
		"##########\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Lamps) != 1 || m.Lamps[0].Flicker {
		t.Fatalf("lamps = %+v, want one steady lamp", m.Lamps)
	}
	lit := map[int]float64{}
	for _, c := range m.Lamps[0].Reach {
		lit[c.Index] = c.Weight
	}
	at := func(x, y int) float64 { return lit[y*m.W+x] }
	if near, far := at(5, 2), at(3, 3); near <= far || near <= 0 {
		t.Errorf("light next to the lamp %v, two cells away %v: want brighter close up", near, far)
	}
	if want := LampPower * (1 - 1/LampRadius) * (1 - 1/LampRadius); math.Abs(at(5, 2)-want) > 1e-9 {
		t.Errorf("light one cell away = %v, want %v", at(5, 2), want)
	}
	if at(7, 2) != 0 {
		t.Error("cell behind the crate is lit: cover should cast a shadow")
	}
	if at(2, 5) != 0 {
		t.Error("cell behind the wall is lit")
	}
	if at(6, 2) != 0 {
		t.Error("the crate's own cell is in the light list")
	}
}

// Red barrels (M14) are cover like barrels; acid is open floor.
func TestParseHazards(t *testing.T) {
	m, err := Parse([]byte("#######\n#PB~..#\n#######\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Props) != 1 || m.Props[0].Kind != ExplosiveBarrel || !m.Blocked(2, 1) || !m.At(2, 1).BlocksShot() {
		t.Errorf("red barrel: props %+v, blocked %v; want a prop that blocks moves and shots", m.Props, m.Blocked(2, 1))
	}
	if m.At(3, 1) != Acid || m.Blocked(3, 1) || m.IsSolid(3, 1) || m.At(3, 1).IsProp() {
		t.Errorf("acid cell %q: want open floor, not a prop", m.At(3, 1))
	}
}

// ReplaceProp swaps a prop for another (or floor), keeps Props in step, and
// lets lamp light through where the old prop cast a shadow.
func TestReplaceProp(t *testing.T) {
	m, err := Parse([]byte("#######\n#PlB..#\n#######\n"))
	if err != nil {
		t.Fatal(err)
	}
	lit := func(x int) bool {
		for _, l := range m.Lamps[0].Reach {
			if l.Index == 1*m.W+x {
				return true
			}
		}
		return false
	}
	if lit(4) {
		t.Fatal("cell behind the red barrel is lit before it blows up")
	}
	m.ReplaceProp(3, 1, Debris)
	if m.At(3, 1) != Debris || len(m.Props) != 2 || m.Props[1].Kind != Debris || m.Props[1].X != 3.5 {
		t.Errorf("after ReplaceProp: cell %q, props %+v; want debris in the barrel's place", m.At(3, 1), m.Props)
	}
	if !lit(4) {
		t.Error("cell behind the barrel still dark after it's gone")
	}
}
