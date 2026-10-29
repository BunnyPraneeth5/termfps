package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/BunnyPraneeth5/termfps/internal/app"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// The real maps are embedded into the binary, so a broken map must fail
// here, not when someone runs the game. Sizes and order are from the PRD.
func TestEmbeddedMapsParse(t *testing.T) {
	maps, err := loadMaps(mapFiles)
	if err != nil {
		t.Fatalf("loadMaps: %v", err)
	}
	want := []struct {
		name string
		w, h int
	}{
		{"The Arena", 16, 16},
		{"The Warehouse", 32, 32},
		{"The Yard", 64, 32},
		{"The Tunnel", 16, 64},
	}
	if len(maps) != len(want) {
		t.Fatalf("%d maps, want %d", len(maps), len(want))
	}
	for i, w := range want {
		m := maps[i]
		if m.Name != w.name || m.W != w.w || m.H != w.h {
			t.Errorf("map %d: %q %dx%d, want %q %dx%d", i+1, m.Name, m.W, m.H, w.name, w.w, w.h)
		}
	}
}

// Every open cell must be reachable from the spawn, so the whole map can be
// walked (PRD success criterion) and the spawner never picks a sealed cell.
// Doors count as open (enemies and the player open them); cells a prop
// fills don't need to be reached.
func TestEmbeddedMapsAllReachable(t *testing.T) {
	maps, err := loadMaps(mapFiles)
	if err != nil {
		t.Fatalf("loadMaps: %v", err)
	}
	for _, m := range maps {
		dist := world.DistanceField(m, int(m.SpawnX), int(m.SpawnY), nil)
		for i, c := range m.Cells {
			if !c.Solid() && !c.BlocksMove() && dist[i] < 0 {
				t.Errorf("%s: open cell (%d, %d) can't be reached from the spawn", m.Name, i%m.W, i/m.W)
			}
		}
	}
}

func TestLoadMapsErrors(t *testing.T) {
	good := &fstest.MapFile{Data: []byte("; name: A\n###\n#P#\n###\n")}
	cases := []struct {
		name string
		fsys fstest.MapFS
	}{
		{"no maps", fstest.MapFS{"maps": {Mode: fs.ModeDir}}},
		{"broken map", fstest.MapFS{"maps/1.txt": good, "maps/2.txt": {Data: []byte("; name: B\n#P#\n")}}},
		{"no name header", fstest.MapFS{"maps/1.txt": {Data: []byte("###\n#P#\n###\n")}}},
	}
	for _, c := range cases {
		if _, err := loadMaps(c.fsys); err == nil {
			t.Errorf("%s: loadMaps succeeded, want an error", c.name)
		}
	}
}

// Flags given on the command line win over saved settings; the others
// keep the saved value.
func TestWithFlags(t *testing.T) {
	saved := app.Config{FPS: 60, KeyMode: "events", PixelAspect: 1.5, Textures: false}
	flags := app.Config{FPS: 120, KeyMode: "async", PixelAspect: 1.0, Textures: true}
	got := withFlags(saved, flags, map[string]bool{"fps": true, "textures": true})
	if got.FPS != 120 || !got.Textures || got.KeyMode != "events" || got.PixelAspect != 1.5 {
		t.Errorf("withFlags = %+v, want fps and textures from flags, the rest saved", got)
	}
}

// The player's own maps (M9): every .txt file in the folder, in name order;
// a map without a name is named after its file, a taken name gets the file
// name added, and a broken file is skipped and named in the problem.
func TestLoadCustomMaps(t *testing.T) {
	dir := t.TempDir() // removed when the test ends
	files := map[string]string{
		"a.txt":    "; name: The Arena\n###\n#P#\n###\n", // name taken by a built-in map
		"b.txt":    "###\n#P#\n###\n",                    // no name header
		"c.txt":    "; name: C\n#P#\n",                   // too small: broken
		"notes.md": "not a map",
		"d.TXT":    "; name: D\n###\n#P#\n###\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	taken := map[string]bool{"The Arena": true}
	maps, problem := loadCustomMaps(dir, taken)
	var names []string
	for _, m := range maps {
		names = append(names, m.Name)
	}
	if got := strings.Join(names, ", "); got != "The Arena (a.txt), b, D" {
		t.Errorf("custom maps = %q, want \"The Arena (a.txt), b, D\"", got)
	}
	if !strings.Contains(problem, "c.txt") {
		t.Errorf("problem = %q, want it to name the broken c.txt", problem)
	}
	if maps, problem := loadCustomMaps(filepath.Join(dir, "missing"), taken); maps != nil || problem != "" {
		t.Errorf("missing folder: %v, %q; want no maps and no problem", maps, problem)
	}
	if _, err := loadMapFile(filepath.Join(dir, "nope.txt"), taken); err == nil {
		t.Error("loadMapFile of a missing file: no error")
	}
}

// The map design rules (M5, D47): every built-in map has a lighting
// header, lamps, cover, a door, starting enemies and red targets.
func TestEmbeddedMapsFollowDesignRules(t *testing.T) {
	maps, err := loadMaps(mapFiles)
	if err != nil {
		t.Fatalf("loadMaps: %v", err)
	}
	for _, m := range maps {
		cover, targets := 0, 0
		for _, c := range m.Cells {
			switch {
			case c.BlocksShot():
				cover++
			case c == world.Target:
				targets++
			}
		}
		switch {
		case m.Fog == 0 && m.Darkness == 0:
			t.Errorf("%s: no fog or darkness header", m.Name)
		case len(m.Lamps) == 0:
			t.Errorf("%s: no lamps", m.Name)
		case cover < 3:
			t.Errorf("%s: %d cover props, want 3 or more", m.Name, cover)
		case len(m.Doors) == 0:
			t.Errorf("%s: no doors", m.Name)
		case len(m.EnemyStarts) == 0 || targets == 0:
			t.Errorf("%s: %d starting enemies, %d targets; want some of each", m.Name, len(m.EnemyStarts), targets)
		}
	}
}
