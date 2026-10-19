package render

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

// newSimScreen returns tcell's in-memory screen, so presenters can be tested
// without a real terminal.
func newSimScreen(t testing.TB, cols, rows int) tcell.SimulationScreen {
	t.Helper()
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatalf("simulation screen Init: %v", err)
	}
	s.SetSize(cols, rows)
	t.Cleanup(s.Fini) // runs when the test ends
	return s
}

// A top pixel of 255 over a bottom pixel of 0 must give a white foreground
// and black background. Swapping them would flip every cell upside down.
func TestPresentHalfBlockTopIsForeground(t *testing.T) {
	s := newSimScreen(t, 4, 3) // Frame 4 x 4 (2 view rows + HUD row)
	f := newFrame(4, 3)
	for x := 0; x < f.W; x++ {
		f.Pix[0*f.W+x].Gray = 255 // view row 0, top
		f.Pix[1*f.W+x].Gray = 0   // view row 0, bottom
		f.Pix[2*f.W+x].Gray = 10  // view row 1, top
		f.Pix[3*f.W+x].Gray = 200 // view row 1, bottom
	}
	PresentHalfBlock(s, f)

	cases := []struct {
		row    int
		fg, bg int32
	}{
		{0, 255, 0},
		{1, 10, 200},
	}
	for _, c := range cases {
		str, style, _ := s.Get(2, c.row)
		if str != string(halfBlock) {
			t.Errorf("row %d: cell = %q, want %q", c.row, str, string(halfBlock))
		}
		fg, bg, _ := style.Decompose()
		if r, _, _ := fg.RGB(); r != c.fg {
			t.Errorf("row %d: foreground gray = %d, want %d", c.row, r, c.fg)
		}
		if r, _, _ := bg.RGB(); r != c.bg {
			t.Errorf("row %d: background gray = %d, want %d", c.row, r, c.bg)
		}
	}
}

// The HUD row (the last one) is not touched by the presenter.
func TestPresentHalfBlockLeavesHUDRow(t *testing.T) {
	s := newSimScreen(t, 4, 3)
	f := newFrame(4, 3)
	PresentHalfBlock(s, f)
	if str, _, _ := s.Get(0, 2); str == string(halfBlock) {
		t.Error("presenter drew on the HUD row")
	}
}

// Benchmarks for the T6/T17 budget at 120x40. Run with:
// go test -bench . ./internal/render
func BenchmarkDrawFrame120x40(b *testing.B) {
	s := newState(b, 2.0, 3.5)
	f := newFrame(120, 40)
	for b.Loop() {
		DrawFrame(s, f)
	}
}

func BenchmarkPresentHalfBlock120x40(b *testing.B) {
	scr := newSimScreen(b, 120, 40)
	f := newFrame(120, 40)
	DrawFrame(newState(b, 2.0, 3.5), f)
	for b.Loop() {
		PresentHalfBlock(scr, f)
	}
}
