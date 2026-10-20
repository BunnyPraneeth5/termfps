package game

import "testing"

func TestRenderModeCycle(t *testing.T) {
	s := newOpenState(t)
	if s.Mode != ModeHalfBlock {
		t.Fatalf("start mode = %v, want half-block", s.Mode)
	}
	want := []RenderMode{ModeShade, ModeASCII, ModeHalfBlock, ModeShade}
	for i, w := range want {
		Update(s, Input{CycleRenderMode: true}, 1.0/30)
		if s.Mode != w {
			t.Errorf("after %d presses: mode = %v, want %v", i+1, s.Mode, w)
		}
	}
	Update(s, Input{}, 1.0/30) // no press: mode stays
	if s.Mode != ModeShade {
		t.Errorf("mode changed without a press: %v", s.Mode)
	}
}

// Cycling the mode emits an event for that frame only.
func TestRenderModeChangedEvent(t *testing.T) {
	s := newOpenState(t)
	Update(s, Input{CycleRenderMode: true}, 1.0/30)
	if !s.HasEvent(EventRenderModeChanged) {
		t.Error("no EventRenderModeChanged after V")
	}
	Update(s, Input{Forward: true}, 1.0/30)
	if len(s.Events) != 0 {
		t.Errorf("events = %v on a frame without V, want none", s.Events)
	}
}

func TestRenderModeString(t *testing.T) {
	cases := map[RenderMode]string{
		ModeHalfBlock: "half-block", ModeShade: "shade", ModeASCII: "ascii", RenderMode(9): "unknown",
	}
	for m, want := range cases {
		if got := m.String(); got != want {
			t.Errorf("RenderMode(%d).String() = %q, want %q", int(m), got, want)
		}
	}
}
