// Command termfps is a terminal first-person game (P0 prototype).
package main

import (
	_ "embed" // the blank import (_) enables the //go:embed directive below
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/BunnyPraneeth5/termfps/internal/game"
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

// run opens the screen, runs the game, and returns the process exit code.
// exitCode is a named result so the deferred panic handler can change it.
func run() (exitCode int) {
	targetFPS := flag.Int("fps", 30, "target frames per second (1-240)")
	flag.Parse()
	if *targetFPS < 1 || *targetFPS > 240 {
		fmt.Fprintf(os.Stderr, "termfps: -fps must be between 1 and 240, got %d\n", *targetFPS)
		return 2
	}

	// TERMFPS_PANIC forces a panic to test terminal restore:
	// "1" panics in the main loop after 1 second,
	// "input" panics in the input goroutine on the first key press.
	panicMode := os.Getenv("TERMFPS_PANIC")

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
		pumpEvents(screen, events, panicMode == "input")
	}()

	state := game.NewState(arena)

	// The Frame is reused every frame; it only changes size on resize.
	// FrameSize(screen.Size()) passes Size's two results as the two arguments.
	var frame render.Frame
	frame.Resize(render.FrameSize(screen.Size()))

	interval := time.Second / time.Duration(*targetFPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	start := time.Now()
	last := start
	var fps fpsCounter
	var times frameTimes // measured on the previous frame, shown in the HUD

	for {
		<-ticker.C // wait for the next tick (receives from the ticker's channel)
		frameStart := time.Now()

		quit := false
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
				// A type switch picks a case by the concrete type behind the
				// tcell.Event interface; ev has that type inside each case.
				switch ev := ev.(type) {
				case *tcell.EventResize:
					screen.Sync()
					frame.Resize(render.FrameSize(screen.Size()))
				case *tcell.EventKey:
					if isQuitKey(ev) {
						quit = true
					}
				}
			default:
				break drain
			}
		}
		if quit {
			return 0
		}

		if panicMode == "1" && frameStart.Sub(start) > time.Second {
			panic("forced test panic (TERMFPS_PANIC=1)")
		}

		dt := clampDT(frameStart.Sub(last).Seconds())
		last = frameStart
		fps.tick(frameStart)
		_ = dt // used by game.Update from T7

		t0 := time.Now()
		render.DrawFrame(state, &frame)
		t1 := time.Now()
		render.PresentHalfBlock(screen, &frame)
		drawHUD(screen, fps.value(), times)
		t2 := time.Now()
		screen.Show()
		times = frameTimes{draw: t1.Sub(t0), present: t2.Sub(t1), show: time.Since(t2)}
	}
}

// frameTimes are the three parts of a frame measured separately (see
// ARCHITECTURE "Performance budget").
type frameTimes struct {
	draw    time.Duration // DrawFrame: rays into the pixel buffer
	present time.Duration // presenter + HUD: pixels into tcell's cell buffer
	show    time.Duration // Show: tcell sends changed cells to the terminal
}

// pumpEvents runs on its own goroutine. It reads screen events and sends them
// to out. It only sends; it never touches game state. It returns when the
// screen is finalized (PollEvent returns nil).
func pumpEvents(s tcell.Screen, out chan<- tcell.Event, panicOnKey bool) {
	for {
		ev := s.PollEvent()
		if ev == nil {
			return
		}
		if _, isKey := ev.(*tcell.EventKey); isKey && panicOnKey {
			panic("forced test panic in input goroutine (TERMFPS_PANIC=input)")
		}
		out <- ev
	}
}

// isQuitKey reports whether the key is Esc or Ctrl+C.
func isQuitKey(ev *tcell.EventKey) bool {
	return ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC
}

// drawHUD clears the last row and writes the stats on it. T15 replaces this
// with render.DrawHUD.
func drawHUD(s tcell.Screen, fps float64, t frameTimes) {
	w, h := s.Size()
	for x := 0; x < w; x++ {
		s.SetContent(x, h-1, ' ', nil, tcell.StyleDefault)
	}
	hud := fmt.Sprintf("FPS %4.1f | draw %.2f present %.2f show %.2f ms | colors %d",
		fps, ms(t.draw), ms(t.present), ms(t.show), s.Colors())
	s.PutStr(0, h-1, hud)
}

// ms converts a duration to milliseconds with fractions.
func ms(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}
