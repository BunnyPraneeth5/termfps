package main

import (
	"io/fs"
	"testing"
	"testing/fstest"

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
func TestEmbeddedMapsAllReachable(t *testing.T) {
	maps, err := loadMaps(mapFiles)
	if err != nil {
		t.Fatalf("loadMaps: %v", err)
	}
	for _, m := range maps {
		dist := world.DistanceField(m, int(m.SpawnX), int(m.SpawnY), nil)
		for i, c := range m.Cells {
			if !c.Solid() && dist[i] < 0 {
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
