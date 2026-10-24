// Command termfps is a terminal first-person game.
//
// main only does startup: flags, the embedded map, the terminal, and crash
// handling. The game loop and screens live in internal/app.
package main

import (
	_ "embed" // the blank import (_) enables the //go:embed directive below
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/app"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// arenaFile is the map file, copied into the binary at build time by
// //go:embed. The directive must sit directly above a package-level var.
//
//go:embed maps/arena.txt
var arenaFile []byte

func main() {
	// os.Exit skips deferred functions, so all the real work (and every defer)
	// lives in run. By the time run returns, the terminal is already restored.
	os.Exit(run())
}

// run checks the flags, opens the screen, runs the app, and returns the
// process exit code. exitCode is a named result so the deferred panic
// handler can change it.
func run() (exitCode int) {
	cfg, ok := parseFlags()
	if !ok {
		return 2
	}

	// Parse the map before opening the screen, so errors print normally.
	arena, err := world.Parse(arenaFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "termfps: maps/arena.txt:", err)
		return 1
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "termfps: create screen:", err)
		return 1
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "termfps: init screen:", err)
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

	return app.Run(screen, []*world.Map{arena}, cfg, restore)
}

// parseFlags reads and checks the command-line flags. On a bad value it
// prints why and returns ok = false.
func parseFlags() (cfg app.Config, ok bool) {
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

	if *aspect < 0.5 || *aspect > 2.0 {
		fmt.Fprintf(os.Stderr, "termfps: -aspect must be between 0.5 and 2.0, got %v\n", *aspect)
		return cfg, false
	}
	if *targetFPS < 1 || *targetFPS > 240 {
		fmt.Fprintf(os.Stderr, "termfps: -fps must be between 1 and 240, got %d\n", *targetFPS)
		return cfg, false
	}
	switch *keyMode {
	case "events":
	case "async":
		if err := input.AsyncSupported(); err != nil {
			fmt.Fprintln(os.Stderr, "termfps: -keys async:", err)
			return cfg, false
		}
	default:
		fmt.Fprintf(os.Stderr, "termfps: -keys must be events or async, got %q\n", *keyMode)
		return cfg, false
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
		MinimapAtStart: true,
		Sound:          true,
		Volume:         80,
	}, true
}
