// Package app runs the game: the main loop, the current screen, and the
// wiring between input, game state and rendering. main.go only does startup
// (flags, map, terminal) and crash handling, then calls Run.
package app

import (
	"fmt"
	"math/rand/v2"
	"os"
	"runtime/debug"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/input"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// Config holds the startup options (from command-line flags).
type Config struct {
	FPS         int     // target frames per second
	KeyMode     string  // "async" (real key state, Windows) or "events"
	PixelAspect float64 // pixel height / width
	PanicMode   string  // TERMFPS_PANIC test switch: "", "1" or "input"
	Textures    bool    // wall textures on (P1-10); a menu setting in P1-11
}

// App is everything the loop needs between frames.
type App struct {
	scr     tcell.Screen
	cfg     Config
	level   *world.Map // the map as loaded; each run plays on a fresh copy
	state   *game.State
	frame   render.Frame
	keys    input.Tracker // which movement keys count as held (D12)
	focused bool          // terminal has focus (async keys only count then)
	current screen        // what's showing: playing now, menus later (P1-11)

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
}

// cautionFor is how long the half-block caution stays on screen.
const cautionFor = 5 * time.Second

// tierNoticeFor is how long the "TIER n" notice shows after a tier-up.
const tierNoticeFor = 3 * time.Second

const caution = "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."

// New sets up the app on an initialized screen, starting on the playing
// screen at time now.
func New(scr tcell.Screen, m *world.Map, cfg Config, now time.Time) *App {
	a := &App{
		scr:     scr,
		cfg:     cfg,
		level:   m,
		state:   game.NewState(m.Clone()),
		frame:   render.Frame{PixelAspect: cfg.PixelAspect, Textures: cfg.Textures},
		focused: true,
		start:   now,
		last:    now,
	}
	a.state.Rand = newRand() // real games differ each time; tests reset it if needed
	// Half-block needs 24-bit color (D17). tcell reports 1<<24 colors when it
	// has truecolor, and fewer with TCELL_TRUECOLOR=disable or NO_COLOR.
	if scr.Colors() < 1<<24 {
		a.state.Mode = game.ModeASCII
	}
	// Async key state sees keys pressed in any window, so it only counts
	// while the terminal has focus. Focus events need terminal support; if
	// none arrive, focused stays true (the HUD shows it).
	if cfg.KeyMode == "async" {
		scr.EnableFocus()
	}
	// FrameSize(scr.Size()) passes Size's two results as the two arguments.
	a.frame.Resize(render.FrameSize(scr.Size()))
	a.switchTo(&playingScreen{})
	return a
}

// newRun starts a fresh run on a clean copy of the map, keeping the
// player's render mode and minimap choice.
func (a *App) newRun() {
	old := a.state
	a.state = game.NewState(a.level.Clone())
	a.state.Mode = old.Mode
	a.state.ShowMinimap = old.ShowMinimap
	a.state.Rand = newRand()
	a.keys = input.Tracker{}
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
func Run(scr tcell.Screen, m *world.Map, cfg Config, restore func()) (exitCode int) {
	a := New(scr, m, cfg, time.Now())

	// A channel passes values between goroutines. The buffer (64) lets the
	// input goroutine queue events while the main loop is busy drawing.
	events := make(chan tcell.Event, 64)

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

	ticker := time.NewTicker(time.Second / time.Duration(cfg.FPS))
	defer ticker.Stop()

	var batch []tcell.Event // reused every frame
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
	}
}

// step runs one frame at time now with the events that arrived since the
// last frame. It reports whether the app should quit. Tests call it directly
// with a simulation screen, without the ticker or the input goroutine.
func (a *App) step(now time.Time, events []tcell.Event) (quit bool) {
	for _, ev := range events {
		// A type switch picks a case by the concrete type behind the
		// tcell.Event interface; ev has that type inside each case.
		switch ev := ev.(type) {
		case *tcell.EventResize:
			a.scr.Sync()
			a.frame.Resize(render.FrameSize(a.scr.Size()))
		case *tcell.EventFocus:
			a.focused = ev.Focused
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyCtrlC {
				return true // Ctrl+C quits from any screen (D24)
			}
			a.current.key(a, ev)
		}
	}
	if a.quit {
		return true
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

// keysLabel names the input mode for the HUD; async also shows focus.
func keysLabel(mode string, focused bool) string {
	if mode != "async" {
		return "keys events"
	}
	if focused {
		return "keys async (focused)"
	}
	return "keys async (NOT focused)"
}
