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
