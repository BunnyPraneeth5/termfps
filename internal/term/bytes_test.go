// Package term_test is an external test package: it uses term like any
// other package would, so it can also use render (which imports term)
// without an import cycle.
package term_test

import (
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gdamore/tcell/v3/tty"

	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/render"
	"github.com/BunnyPraneeth5/termfps/internal/term"
	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// countingTty is a fake terminal connection: it counts the bytes tcell
// writes and never sends any input.
type countingTty struct {
	w, h    int
	written atomic.Int64 // tcell may write from more than one goroutine
	mu      sync.Mutex
	stop    chan struct{} // closed by Stop, so a waiting Read returns
}

func (c *countingTty) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stop = make(chan struct{})
	return nil
}

func (c *countingTty) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stop != nil {
		close(c.stop)
		c.stop = nil
	}
	return nil
}

func (c *countingTty) Drain() error             { return nil }
func (c *countingTty) NotifyResize(chan<- bool) {}
func (c *countingTty) Close() error             { return nil }
func (c *countingTty) WindowSize() (tty.WindowSize, error) {
	return tty.WindowSize{Width: c.w, Height: c.h}, nil
}

func (c *countingTty) Write(p []byte) (int, error) {
	c.written.Add(int64(len(p)))
	return len(p), nil
}

// Read waits until Stop: the fake user never types.
func (c *countingTty) Read(p []byte) (int, error) {
	c.mu.Lock()
	stop := c.stop
	c.mu.Unlock()
	if stop != nil {
		<-stop
	}
	return 0, io.EOF
}

// warehouse loads the Warehouse map for the byte benchmarks.
func warehouse(b *testing.B) *world.Map {
	data, err := os.ReadFile("../../maps/02-warehouse.txt")
	if err != nil {
		b.Fatal(err)
	}
	m, err := world.Parse(data)
	if err != nil {
		b.Fatal(err)
	}
	return m
}

// measureBytes reports how many bytes tcell sends per frame on a w x h
// terminal while the player turns 6 degrees a frame (180 degrees per
// second at 30 FPS), after the first frame (which draws every cell).
func measureBytes(b *testing.B, m *world.Map, w, h int, present func(term.Screen, *render.Frame), threshold uint8, scale int) {
	tt := &countingTty{w: w, h: h}
	scr, err := term.OpenTty(tt)
	if err != nil {
		b.Fatal(err)
	}
	defer scr.Fini()
	st := game.NewState(m.Clone())
	f := render.Frame{PixelAspect: render.DefaultPixelAspect, Textures: true,
		MixedBackground: render.DefaultMixedBackground, ColorThreshold: threshold, Scale: scale}
	cols, rows := scr.Size()
	f.Resize(render.ScaledFrameSize(cols, rows, scale))
	frame := func() {
		st.Player.Angle = math.Mod(st.Player.Angle+6*math.Pi/180, 2*math.Pi)
		render.DrawFrame(st, &f)
		present(scr, &f)
		scr.Show()
	}
	frame()
	start := tt.written.Load()
	frames := 0
	for b.Loop() {
		frame()
		frames++
	}
	b.ReportMetric(float64(tt.written.Load()-start)/float64(frames), "bytes/frame")
}

// BenchmarkBytesPerFrame measures bytes per frame at 120x40 on the
// Warehouse map for each truecolor render mode and color threshold (S3).
// Run with:
//
//	go test -run - -bench BytesPerFrame ./internal/term
//
// The bytes/frame column is the number to compare; ns/op is mostly tcell
// building escape codes.
func BenchmarkBytesPerFrame(b *testing.B) {
	m := warehouse(b)
	presenters := []struct {
		name    string
		present func(term.Screen, *render.Frame)
	}{
		{"half-block", render.PresentHalfBlock},
		{"mixed", render.PresentMixed},
		{"ascii", render.PresentASCII}, // reference: no RGB colors
	}
	for _, p := range presenters {
		for _, threshold := range []uint8{0, 4, 8, 16, 32} {
			if p.name == "ascii" && threshold > 0 {
				continue // the threshold only applies to RGB colors
			}
			b.Run(fmt.Sprintf("%s/threshold-%d", p.name, threshold), func(b *testing.B) {
				measureBytes(b, m, 120, 40, p.present, threshold, 1)
			})
		}
	}
}

// BenchmarkBytesPerFrameBig measures a big (zoomed-out) 320x90 terminal in
// half-block at full resolution and at the scale "auto" picks there (V6),
// with the threshold off and at 8.
func BenchmarkBytesPerFrameBig(b *testing.B) {
	m := warehouse(b)
	for _, scale := range []int{1, render.AutoScale(320)} {
		for _, threshold := range []uint8{0, 8} {
			b.Run(fmt.Sprintf("scale-%d/threshold-%d", scale, threshold), func(b *testing.B) {
				measureBytes(b, m, 320, 90, render.PresentHalfBlock, threshold, scale)
			})
		}
	}
}
