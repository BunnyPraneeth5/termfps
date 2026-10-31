package audio

import (
	"math"
	"testing"
)

// Every effect has a sensible length, makes sound, never clips, and ends
// quietly (no click when it stops).
func TestEffects(t *testing.T) {
	for e := Effect(0); e < NumEffects; e++ {
		s := Generate(e)
		dur := float64(len(s)) / SampleRate
		if dur < 0.02 || dur > 1.5 {
			t.Errorf("effect %d lasts %.2f s, want 0.02-1.5 s", e, dur)
		}
		loudest := 0.0
		for _, v := range s {
			loudest = max(loudest, math.Abs(v))
		}
		if loudest > peak+1e-9 {
			t.Errorf("effect %d peaks at %.3f, over %.1f (clipping risk)", e, loudest, peak)
		}
		if loudest < 0.1 {
			t.Errorf("effect %d peaks at %.3f, nearly silent", e, loudest)
		}
		tail := s[len(s)-len(s)/20:] // the last 5%
		for _, v := range tail {
			if math.Abs(v) > 0.5*loudest {
				t.Errorf("effect %d still loud at the end (%.3f)", e, v)
				break
			}
		}
	}
}

func TestGenerateRepeats(t *testing.T) {
	a, b := Generate(Gunshot), Generate(Gunshot)
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("Gunshot differs between two calls; the noise seed should be fixed")
		}
	}
}

func TestPCM(t *testing.T) {
	got := PCM([]float64{0, 1, -1, 2, 0.5})
	want := []int16{0, math.MaxInt16, -math.MaxInt16, math.MaxInt16, 16384}
	if len(got) != 2*len(want) {
		t.Fatalf("PCM length %d, want %d", len(got), 2*len(want))
	}
	for i, w := range want {
		if v := int16(uint16(got[2*i]) | uint16(got[2*i+1])<<8); v != w {
			t.Errorf("sample %d = %d, want %d", i, v, w)
		}
	}
}

// No effect plays silent (M17: the effects added in M10-M16 had gain 0).
func TestGainNeverSilent(t *testing.T) {
	for e := Effect(0); e < NumEffects; e++ {
		if g := Gain(e); g <= 0 || g > 1 {
			t.Errorf("effect %d has gain %.2f, want 0 < gain <= 1", e, g)
		}
	}
}

// Middle: both sides full; edges: only that side.
func TestPanGains(t *testing.T) {
	for _, c := range []struct{ pan, l, r float64 }{{0, 1, 1}, {-1, 1, 0}, {1, 0, 1}} {
		l, r := PanGains(c.pan)
		if math.Abs(l-c.l) > 1e-9 || math.Abs(r-c.r) > 1e-9 {
			t.Errorf("PanGains(%v) = %.3f, %.3f; want %v, %v", c.pan, l, r, c.l, c.r)
		}
	}
	if l, r := PanGains(0.5); r != 1 || l <= 0 || l >= 1 {
		t.Errorf("PanGains(0.5) = %.3f, %.3f; want right full, left between", l, r)
	}
	if PanStep(-1) != 0 || PanStep(0) != panSteps/2 || PanStep(1) != panSteps-1 || PanStep(5) != panSteps-1 {
		t.Error("PanStep doesn't map -1, 0, 1 to the first, middle and last position")
	}
}

// Stereo bytes: left then right per sample; full right leaves left silent.
func TestStereoPCM(t *testing.T) {
	b := StereoPCM([]float64{0.5, -0.5}, 1)
	if len(b) != 8 {
		t.Fatalf("%d bytes for 2 samples, want 8", len(b))
	}
	left := int16(uint16(b[0]) | uint16(b[1])<<8)
	right := int16(uint16(b[2]) | uint16(b[3])<<8)
	if left != 0 || right != int16(math.Round(0.5*math.MaxInt16)) {
		t.Errorf("first sample panned right: left %d, right %d; want 0 and half scale", left, right)
	}
}
