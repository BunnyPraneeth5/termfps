// Command termfps is a terminal first-person game.
//
// main only does startup: flags, the embedded maps, the terminal, and crash
// handling. The game loop and screens live in internal/app.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/BunnyPraneeth5/termfps/internal/app"
	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/save"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// mapFiles holds every maps/*.txt file, copied into the binary at build
// time by //go:embed. An embed.FS is a small read-only file system; the
// directive must sit directly above a package-level var.
//
//go:embed maps/*.txt
var mapFiles embed.FS

func main() {
	// os.Exit skips deferred functions, so all the real work (and every defer)
	// lives in run. By the time run returns, the terminal is already restored.
	os.Exit(run())
}

// run checks the flags, opens the screen, runs the app, and returns the
// process exit code. exitCode is a named result so the deferred panic
// handler can change it.
func run() (exitCode int) {
	flagCfg, explicit, ok := parseFlags()
	if !ok {
		return 2
	}
	// Saved settings replace the flag defaults; flags given on the command
	// line win over both. Save problems are shown in the menu, not here.
	saved, settings := loadSave()
	cfg := flagCfg
	if settings != nil {
		var note string
		cfg, note = app.ApplySettings(flagCfg, *settings)
		cfg = withFlags(cfg, flagCfg, explicit)
		saved.Message = strings.TrimSpace(saved.Message + " " + note)
	}

	// Parse the maps before opening the screen, so errors print normally.
	maps, err := loadMaps(mapFiles)
	if err != nil {
		fmt.Fprintln(os.Stderr, "termfps:", err)
		return 1
	}

	// Open audio before the screen: audio libraries may print warnings, which
	// would land in the game screen. If it fails, the game runs without
	// sound and the menu says why (D27).
	player, err := audio.New()
	snd := app.Sound{Player: player, Err: err}

	screen, err := term.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "termfps:", err) // "create screen: ..." or "init screen: ..."
		return 1
	}

	// sync.Once runs the function at most once, no matter how many times
	// restore is called.
	var once sync.Once
	restore := func() { once.Do(screen.Fini) }

	// Deferred functions run when run returns, and also while a panic unwinds.
	// recover() stops the panic so we can restore the terminal before printing.
	// Restore comes first: printing first would land in the alternate screen.
	defer func() {
		if r := recover(); r != nil {
			restore()
			fmt.Fprintf(os.Stderr, "termfps: panic: %v\n\n%s", r, debug.Stack())
			exitCode = 1
			return
		}
		restore()
	}()

	return app.Run(screen, maps, cfg, saved, snd, restore)
}

// loadSave opens the save file in the user's config folder (P1-13). It
// returns the progress and any problem for the menu, plus the saved
// settings (nil on a first run or when the file couldn't be used).
func loadSave() (app.Saved, *save.Settings) {
	path, err := save.DefaultPath()
	if err != nil {
		return app.Saved{Message: "Progress and settings won't be saved: " + err.Error()}, nil
	}
	store := &save.Store{Path: path}
	data, problem := save.Load(*store)
	return app.Saved{Store: store, Progress: data.Progress, Message: problem}, data.Settings
}

// withFlags copies the settings whose flags were given on the command line
// (explicit, by flag name) from flagCfg into cfg.
func withFlags(cfg, flagCfg app.Config, explicit map[string]bool) app.Config {
	if explicit["fps"] {
		cfg.FPS = flagCfg.FPS
	}
	if explicit["keys"] {
		cfg.KeyMode = flagCfg.KeyMode
	}
	if explicit["aspect"] {
		cfg.PixelAspect = flagCfg.PixelAspect
	}
	if explicit["textures"] {
		cfg.Textures = flagCfg.Textures
	}
	return cfg
}

// parseFlags reads and checks the command-line flags. On a bad value it
// prints why and returns ok = false. explicit holds the names of the flags
// given on the command line (not left at their defaults).
func parseFlags() (cfg app.Config, explicit map[string]bool, ok bool) {
	targetFPS := flag.Int("fps", 30, "target frames per second (1-240)")
	// Real key state (async) is the default on Windows, where it exists (D19).
	defaultKeys := "events"
	if runtime.GOOS == "windows" {
		defaultKeys = "async"
	}
	keyMode := flag.String("keys", defaultKeys, `movement keys: "async" (real key state, Windows only) or "events" (terminal key events)`)
	aspect := flag.Float64("aspect", render.DefaultPixelAspect, "pixel height / width, for tuning wall proportions (0.5-2.0)")
	textures := flag.Bool("textures", true, "wall textures (brick, stone, metal); -textures=false for flat walls")
	flag.Parse()
	explicit = map[string]bool{}
	// flag.Visit calls the function for each flag that was set on the
	// command line.
	flag.Visit(func(f *flag.Flag) { explicit[f.Name] = true })

	if *aspect < 0.5 || *aspect > 2.0 {
		fmt.Fprintf(os.Stderr, "termfps: -aspect must be between 0.5 and 2.0, got %v\n", *aspect)
		return cfg, nil, false
	}
	if *targetFPS < 1 || *targetFPS > 240 {
		fmt.Fprintf(os.Stderr, "termfps: -fps must be between 1 and 240, got %d\n", *targetFPS)
		return cfg, nil, false
	}
	switch *keyMode {
	case "events":
	case "async":
		if err := input.AsyncSupported(); err != nil {
			fmt.Fprintln(os.Stderr, "termfps: -keys async:", err)
			return cfg, nil, false
		}
	default:
		fmt.Fprintf(os.Stderr, "termfps: -keys must be events or async, got %q\n", *keyMode)
		return cfg, nil, false
	}

	return app.Config{
		FPS:         *targetFPS,
		KeyMode:     *keyMode,
		PixelAspect: *aspect,
		// TERMFPS_PANIC forces a panic to test terminal restore:
		// "1" panics in the main loop after 1 second,
		// "input" panics in the input goroutine on the first key press.
		PanicMode: os.Getenv("TERMFPS_PANIC"),
		Textures:  *textures,
		// Defaults for settings that have no flag (the Settings menu changes them).
		MinimapAtStart:  true,
		Sound:           true,
		Volume:          80,
		MixedBackground: int(render.DefaultMixedBackground * 100),
		RenderScale:     "auto",
		UnlockRule:      "5:00",
		MinimapFog:      true,
	}, explicit, true
}

// loadMaps parses every maps/*.txt file in fsys, in file name order (the
// number prefix sets the order in the map menu). A map without a "; name:"
// header is an error, since the menu needs a name.
func loadMaps(fsys fs.FS) ([]*world.Map, error) {
	entries, err := fs.ReadDir(fsys, "maps") // sorted by file name
	if err != nil {
		return nil, fmt.Errorf("read maps: %w", err)
	}
	var maps []*world.Map
	for _, e := range entries {
		path := "maps/" + e.Name()
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		m, err := world.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if m.Name == "" {
			return nil, fmt.Errorf("%s: no \"; name:\" header on the first line", path)
		}
		maps = append(maps, m)
	}
	if len(maps) == 0 {
		return nil, fmt.Errorf("no maps in maps/")
	}
	return maps, nil
}
