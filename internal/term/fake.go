package term

import (
	"strings"
	"sync"
)

// Fake is an in-memory Screen for tests. Like tcell's simulation screen it
// reports no truecolor unless SetTruecolor says otherwise. Show and Sync
// do nothing: Get and Row read the buffer directly.
type Fake struct {
	w, h      int
	cells     []cell
	truecolor bool

	// events holds what Post sends until PollEvent takes it. A channel is
	// safe to use from two goroutines (a test and an input goroutine).
	events chan Event
	// closeOnce makes Fini close the channel only once; sync.Once runs its
	// function at most once.
	closeOnce sync.Once
}

// cell is what one Fake cell holds.
type cell struct {
	str   string
	style Style
}

// NewFake returns a blank w x h fake screen.
func NewFake(w, h int) *Fake {
	f := &Fake{events: make(chan Event, 64)}
	f.SetSize(w, h)
	return f
}

// SetSize resizes the screen and blanks it. It doesn't post a Resize event;
// tests send one themselves when they want the app to see it.
func (f *Fake) SetSize(w, h int) {
	f.w, f.h = w, h
	f.cells = make([]cell, w*h)
	f.Clear()
}

// SetTruecolor sets what Truecolor reports.
func (f *Fake) SetTruecolor(on bool) { f.truecolor = on }

// Get returns the text and style of cell (x, y). Outside the screen it
// returns "" and the zero Style.
func (f *Fake) Get(x, y int) (s string, st Style) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return "", Style{}
	}
	c := f.cells[y*f.w+x]
	return c.str, c.style
}

// Row returns the text of row y.
func (f *Fake) Row(y int) string {
	var b strings.Builder
	for x := 0; x < f.w; x++ {
		s, _ := f.Get(x, y)
		b.WriteString(s)
	}
	return b.String()
}

// Text returns every row, joined with newlines.
func (f *Fake) Text() string {
	rows := make([]string, f.h)
	for y := range rows {
		rows[y] = f.Row(y)
	}
	return strings.Join(rows, "\n")
}

// Post queues an event for PollEvent. Don't call it after Fini.
func (f *Fake) Post(ev Event) { f.events <- ev }

// Fini closes the screen: PollEvent returns nil once the queue is empty.
func (f *Fake) Fini() { f.closeOnce.Do(func() { close(f.events) }) }

func (f *Fake) Size() (w, h int) { return f.w, f.h }

func (f *Fake) Put(x, y int, s string, st Style) {
	if x < 0 || y < 0 || x >= f.w || y >= f.h {
		return
	}
	f.cells[y*f.w+x] = cell{str: s, style: st}
}

func (f *Fake) Clear() {
	for i := range f.cells {
		f.cells[i] = cell{str: " "}
	}
}

func (f *Fake) Show()           {}
func (f *Fake) Sync()           {}
func (f *Fake) Truecolor() bool { return f.truecolor }
func (f *Fake) EnableMouse()    {}
func (f *Fake) EnableFocus()    {}

// PollEvent waits for a posted event; nil after Fini.
func (f *Fake) PollEvent() Event {
	// Receiving from a closed channel gives ok = false right away.
	ev, ok := <-f.events
	if !ok {
		return nil
	}
	return ev
}
