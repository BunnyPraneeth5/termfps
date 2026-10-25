package save

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func testStore(t *testing.T) Store {
	// t.TempDir is a new folder for this test, deleted when it ends.
	return Store{Path: filepath.Join(t.TempDir(), "termfps", "save.json")}
}

func sample() Data {
	return Data{
		Settings: &Settings{RenderMode: "shade", FPS: 60, MinimapAtStart: true, Textures: false,
			Sound: true, Volume: 70, KeyMode: "events", PixelAspect: 1.1},
		Progress: Progress{
			UnlockedMaps: 2,
			Best:         map[string]Best{"The Arena": {Kills: 31, Time: 312.5, Tier: 3}},
			TotalKills:   57,
		},
	}
}

func TestMissingFileIsFirstRun(t *testing.T) {
	d, problem := Load(testStore(t))
	if problem != "" || d.Settings != nil || d.Progress.UnlockedMaps != 0 {
		t.Errorf("missing file: %+v, problem %q; want empty data, no problem", d, problem)
	}
}

func TestRoundTrip(t *testing.T) {
	s := testStore(t)
	want := sample()
	if err := Write(s, want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, problem := Load(s)
	if problem != "" {
		t.Fatalf("Load problem: %s", problem)
	}
	want.Version = Version
	if !reflect.DeepEqual(got, want) { // compares the whole structs, maps and pointers too
		t.Errorf("round trip:\n got %+v %+v\nwant %+v %+v", got, *got.Settings, want, *want.Settings)
	}
}

// Writing again replaces the file and leaves no temp file behind.
func TestWriteIsAtomic(t *testing.T) {
	s := testStore(t)
	d := sample()
	if err := Write(s, d); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	d.Progress.TotalKills = 99
	if err := Write(s, d); err != nil {
		t.Fatalf("second Write: %v", err)
	}
	if _, err := os.Stat(s.Path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temp file left behind (stat err %v)", err)
	}
	got, _ := Load(s)
	if got.Progress.TotalKills != 99 {
		t.Errorf("total kills = %d after the second write, want 99", got.Progress.TotalKills)
	}
}

// A damaged file is moved to save.json.bad, defaults are used, and the
// problem names what was wrong.
func TestDamagedFile(t *testing.T) {
	cases := []struct{ name, content, wantProblem string }{
		{"not JSON", "{not json", "damaged"},
		{"wrong version", `{"version": 7}`, "version 7"},
		{"bad render mode", `{"version": 1, "settings": {"render_mode": "sixel", "fps": 30, "key_mode": "events", "pixel_aspect": 1}}`, "render mode"},
		{"fps out of range", `{"version": 1, "settings": {"render_mode": "ascii", "fps": 0, "key_mode": "events", "pixel_aspect": 1}}`, "fps 0"},
		{"negative kills", `{"version": 1, "progress": {"total_kills": -1}}`, "negative"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := testStore(t)
			os.MkdirAll(filepath.Dir(s.Path), 0o755)
			if err := os.WriteFile(s.Path, []byte(c.content), 0o644); err != nil {
				t.Fatal(err)
			}
			d, problem := Load(s)
			if d.Settings != nil || d.Progress.TotalKills != 0 {
				t.Errorf("damaged file gave data %+v, want defaults", d)
			}
			if !strings.Contains(problem, c.wantProblem) || !strings.Contains(problem, "save.json.bad") {
				t.Errorf("problem = %q, want it to mention %q and save.json.bad", problem, c.wantProblem)
			}
			if kept, err := os.ReadFile(s.Path + ".bad"); err != nil || string(kept) != c.content {
				t.Errorf("save.json.bad = %q, %v; want the damaged content", kept, err)
			}
			if _, err := os.Stat(s.Path); !os.IsNotExist(err) {
				t.Error("damaged save.json still in place")
			}
		})
	}
}
