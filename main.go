// Command termfps is a terminal first-person game (P0 prototype).
package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

func main() {
	// os.Exit skips deferred functions, so all the real work (and every defer)
	// lives in run. By the time run returns, the terminal is already restored.
	os.Exit(run())
}

// run opens the screen, runs the game, and returns the process exit code.
// exitCode is a named result so the deferred panic handler can change it.
func run() (exitCode int) {
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
	// restore is called (normal exit, panic, later the input goroutine).
	var once sync.Once
	restore := func() { once.Do(screen.Fini) }

	// Deferred functions run when run returns, and also while a panic unwinds.
	// recover() stops the panic so we can restore the terminal before printing.
	defer func() {
		r := recover()
		restore() // first, or the message below lands in the alternate screen
		if r != nil {
			fmt.Fprintf(os.Stderr, "termfps: panic: %v\n\n%s", r, debug.Stack())
			exitCode = 1
		}
	}()

	panicTest := os.Getenv("TERMFPS_PANIC") == "1"
	if panicTest {
		// time.AfterFunc runs the function on another goroutine after the delay.
		// A panic there would skip our recover, so instead it posts an event
		// and the main loop below panics on the main goroutine.
		time.AfterFunc(time.Second, func() {
			for screen.PostEvent(tcell.NewEventInterrupt(nil)) != nil {
				time.Sleep(10 * time.Millisecond) // event queue full, try again
			}
		})
	}

	drawTitle(screen)
	for {
		// A type switch picks a case by the concrete type behind the
		// tcell.Event interface; ev has that type inside each case.
		switch ev := screen.PollEvent().(type) {
		case nil:
			return 0 // screen was finalized
		case *tcell.EventResize:
			screen.Sync()
			drawTitle(screen)
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				return 0
			}
		case *tcell.EventInterrupt:
			if panicTest {
				panic("forced test panic (TERMFPS_PANIC=1)")
			}
		}
	}
}

// drawTitle shows "termfps" centered, with a quit hint below it.
func drawTitle(s tcell.Screen) {
	s.Clear()
	w, h := s.Size()
	const title = "termfps"
	const hint = "Esc or Ctrl+C to quit"
	s.PutStr((w-len(title))/2, h/2, title)
	s.PutStr((w-len(hint))/2, h/2+1, hint)
	s.Show()
}
