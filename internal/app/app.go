// Package app runs the game: the main loop, the current screen, and the
// wiring between input, game state and rendering. main.go only does startup
// (flags, maps, terminal) and crash handling, then calls Run.
package app

import (
	"fmt"
	"math/rand/v2"
	"os"
	"runtime/debug"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/save"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Config holds the settings. They start from the command-line flags; the
// Settings menu changes them while the game runs (P1-11).
type Config struct {
	FPS            int             // target frames per second
	KeyMode        string          // "async" (real key state, Windows) or "events"
	PixelAspect    float64         // pixel height / width
	PanicMode      string          // TERMFPS_PANIC test switch: "", "1" or "input"
	Textures       bool            // wall textures on (P1-10)
	Mode           game.RenderMode // render mode at startup (the game state holds it after that)
	MinimapAtStart bool            // each run starts with the minimap shown
	Sound          bool            // sound effects on
	Volume         int             // sound volume in percent, 0-100
}

// App is everything the loop needs between frames.
type App struct {
	scr      term.Screen
	cfg      Config
	maps     []*world.Map  // as loaded; each run plays on a fresh copy
	mapIndex int           // the map Play starts
	progress save.Progress // unlocked maps, bests, total kills (D25, D26)
	store    *save.Store   // where progress and settings are saved; nil = not saved
	state    *game.State
	frame    render.Frame
	keys     input.Tracker // which movement keys count as held (D12)
	focused  bool          // terminal has focus (async keys only count then)
	current  screen        // what's showing: a menu, the game, game over

	// runActive is true from the start of a run until its result is
	// recorded, so a run never counts twice.
	runActive bool
	// loadMsg is a problem from loading the save file and saveErr the last
	// failed save; the menus show them.
	loadMsg, saveErr string

	// sound plays the effects; nil if it couldn't start (soundErr says why).
	sound    *audio.Player
	soundErr error

	// leftDown and rightDown are the mouse buttons at the last mouse event,
	// so a screen hears each press and release once.
	leftDown, rightDown bool

	// sprites is the per-frame list handed to the renderer, reused between
	// frames so it doesn't allocate.
	sprites []render.Sprite

	start, last time.Time
	fps         fpsCounter
	times       timeAverager // frame timings averaged per FPS window, for the HUD
	quit        bool

	// The half-block caution shows once, the first time V switches back
	// to half-block, until cautionUntil.
	cautionShown bool
	cautionUntil time.Time

	// After a tier-up, "TIER n" shows above the HUD until tierUntil.
	tierUntil time.Time

	// After a map unlocks, "New map unlocked" shows until unlockUntil.
	unlockUntil time.Time
}

// cautionFor is how long the half-block caution stays on screen.
const cautionFor = 5 * time.Second

// tierNoticeFor is how long the "TIER n" notice shows after a tier-up.
const tierNoticeFor = 3 * time.Second

// unlockAfter is how many seconds to survive on a map to open the next (D26).
const unlockAfter = 5 * 60.0

const caution = "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."

// New sets up the app on an initialized screen at time now, showing the
// title menu. maps must not be empty.
func New(scr term.Screen, maps []*world.Map, cfg Config, now time.Time) *App {
	a := &App{
		scr:      scr,
		cfg:      cfg,
		maps:     maps,
		progress: save.Progress{UnlockedMaps: 1, Best: map[string]save.Best{}},
		frame:    render.Frame{PixelAspect: cfg.PixelAspect, Textures: cfg.Textures},
		focused:  true,
		start:    now,
		last:     now,
	}
	// A state exists from the start, so the render mode has a home before
	// the first run.
	a.state = game.NewState(maps[0].Clone())
	a.state.Mode = cfg.Mode
	// Half-block needs 24-bit color (D17).
	if !scr.Truecolor() {
		a.state.Mode = game.ModeASCII
	}
	// Async key state sees keys pressed in any window, so it only counts
	// while the terminal has focus. Focus events need terminal support; if
	// none arrive, focused stays true (the HUD shows it).
	if cfg.KeyMode == "async" {
		scr.EnableFocus()
	}
	scr.EnableMouse() // clicks only, no motion (D28)
	// FrameSize(scr.Size()) passes Size's two results as the two arguments.
	a.frame.Resize(render.FrameSize(scr.Size()))
	a.switchTo(a.titleMenu())
	return a
}

// newRun starts a fresh run on a clean copy of the selected map, keeping
// the render mode. The minimap starts as the settings say.
func (a *App) newRun() {
	a.endRun() // a restarted run still counts
	old := a.state
	a.state = game.NewState(a.maps[a.mapIndex].Clone())
	a.state.Mode = old.Mode
	a.state.ShowMinimap = a.cfg.MinimapAtStart
	a.state.Rand = newRand() // real games differ each time
	a.keys.Reset()           // keeps what it learned about the terminal
	a.runActive = true
	a.switchTo(&playingScreen{})
}

// newRand returns a random source seeded from the clock.
func newRand() *rand.Rand {
	t := uint64(time.Now().UnixNano())
	return rand.New(rand.NewPCG(t, t>>32|1))
}

// present writes the Frame to the screen with the current render mode.
func (a *App) present() {
	switch a.state.Mode {
	case game.ModeHalfBlock:
		render.PresentHalfBlock(a.scr, &a.frame)
	case game.ModeShade:
		render.PresentShade(a.scr, &a.frame)
	case game.ModeASCII:
		render.PresentASCII(a.scr, &a.frame)
	case game.ModeMixed:
		render.PresentMixed(a.scr, &a.frame)
	default:
		panic(fmt.Sprintf("unknown render mode %d", a.state.Mode))
	}
}

// switchTo makes s the current screen.
func (a *App) switchTo(s screen) {
	a.current = s
}

// Run starts the input goroutine and runs the frame loop until the player
// quits. restore puts the terminal back; Run calls it itself only when the
// input goroutine panics (main's deferred handler covers everything else).
// saved is the save file as loaded at startup; snd is the audio player.
func Run(scr term.Screen, maps []*world.Map, cfg Config, saved Saved, snd Sound, restore func()) (exitCode int) {
	a := New(scr, maps, cfg, time.Now())
	a.useSaved(saved)
	a.sound, a.soundErr = snd.Player, snd.Err
	a.applySound()

	// A channel passes values between goroutines. The buffer (64) lets the
	// input goroutine queue events while the main loop is busy drawing.
	events := make(chan term.Event, 64)

	// inputPanic carries a panic report from the input goroutine to the main
	// loop. Only the main loop touches the screen, so a crash in the input
	// goroutine can't race with drawing. Buffer 1: the send never waits.
	inputPanic := make(chan string, 1)

	// "go" starts the function on a new goroutine, which runs at the same
	// time as the main loop. recover() only catches panics on its own
	// goroutine, so this one needs its own deferred handler.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// debug.Stack here captures this goroutine's stack.
				inputPanic <- fmt.Sprintf("%v (input goroutine)\n\n%s", r, debug.Stack())
			}
		}()
		input.Pump(scr, events, cfg.PanicMode == "input")
	}()

	fps := a.cfg.FPS
	ticker := time.NewTicker(time.Second / time.Duration(fps))
	defer ticker.Stop()

	var batch []term.Event // reused every frame
	for {
		<-ticker.C // wait for the next tick (receives from the ticker's channel)
		now := time.Now()

		batch = batch[:0]
	drain:
		// Take every waiting event without blocking. With "default", select
		// doesn't wait: if the channel is empty it runs default right away.
		// "break drain" leaves the labeled for loop, not just the select.
		for {
			select {
			case report := <-inputPanic:
				restore() // first, then print, then stop: no more drawing
				fmt.Fprintf(os.Stderr, "termfps: panic: %s", report)
				return 1
			case ev := <-events:
				batch = append(batch, ev)
			default:
				break drain
			}
		}
		if a.step(now, batch) {
			return 0
		}
		if a.cfg.FPS != fps { // changed in the Settings menu
			fps = a.cfg.FPS
			ticker.Reset(time.Second / time.Duration(fps))
		}
	}
}

// step runs one frame at time now with the events that arrived since the
// last frame. It reports whether the app should quit. Tests call it directly
// with a fake screen, without the ticker or the input goroutine.
func (a *App) step(now time.Time, events []term.Event) (quit bool) {
	for _, ev := range events {
		// A type switch picks a case by the concrete type behind the
		// term.Event interface; ev has that type inside each case.
		switch ev := ev.(type) {
		case term.Resize:
			a.scr.Sync()
			a.frame.Resize(render.FrameSize(a.scr.Size()))
		case term.Focus:
			a.focused = ev.Focused
			if !ev.Focused {
				// Keys let go in another window never send a release here.
				a.keys.ReleaseAll()
			}
		case term.Key:
			// Releases go to the screen too (the playing screen ends held
			// keys with them), but never quit.
			if ev.Code == term.KeyCtrlC && ev.Action != term.Release {
				a.endRun()  // a run quit with Ctrl+C still counts
				return true // Ctrl+C quits from any screen (D24)
			}
			a.current.key(a, ev)
		case term.Mouse:
			// The screen hears about each button going down or up, not
			// about every event (holding a button can repeat events).
			left, right := input.MouseButtons(ev)
			x, y := ev.X, ev.Y
			if left != a.leftDown {
				a.current.mouse(a, mouseLeft, left, x, y)
			}
			if right != a.rightDown {
				a.current.mouse(a, mouseRight, right, x, y)
			}
			a.leftDown, a.rightDown = left, right
		}
		if a.quit {
			return true
		}
	}

	if a.cfg.PanicMode == "1" && now.Sub(a.start) > time.Second {
		panic("forced test panic (TERMFPS_PANIC=1)")
	}

	dt := clampDT(now.Sub(a.last).Seconds())
	a.last = now
	if a.fps.tick(now) {
		a.times.roll()
	}

	// Below the minimum size: show a message, skip the screen this frame.
	if render.TooSmall(a.scr.Size()) {
		render.DrawTooSmall(a.scr)
		a.scr.Show()
		return false
	}

	a.current.tick(a, now, dt)
	return a.quit
}

// keysLabel names the input mode for the HUD; async also shows focus,
// events whether the terminal sends key releases (exact held keys, D42).
func keysLabel(mode string, focused, releases bool) string {
	if mode != "async" {
		if releases {
			return "keys events (releases)"
		}
		return "keys events"
	}
	if focused {
		return "keys async (focused)"
	}
	return "keys async (NOT focused)"
}

// unlocks returns how many maps are open after surviving survived seconds
// on map played (0-based), when open of total maps were open. Surviving
// long enough on the newest open map opens the next one.
func unlocks(open, played int, survived float64, total int) int {
	if played == open-1 && survived >= unlockAfter && open < total {
		return open + 1
	}
	return open
}
