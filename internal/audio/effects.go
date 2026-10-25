// Package audio plays the game's sound effects through the speakers with
// the oto library (D27). The effects are generated in code at startup, so
// there are no sound files. effects.go makes the samples (no oto, easy to
// test); player.go plays them.
package audio

import (
	"math"
	"math/rand/v2"
)

// SampleRate is samples per second for every effect (mono).
const SampleRate = 44100

// Effect is one sound effect.
type Effect int

const (
	Gunshot Effect = iota
	Reload
	EmptyClick
	ZombieGroan
	EnemyShot
	Hurt
	Pickup
	TierUp
	NumEffects // not an effect: how many there are
)

// peak is the loudest sample any effect may have, below 1.0 so several
// effects playing together are less likely to clip.
const peak = 0.8

// Generate returns the effect's samples, mono, each between -1 and 1.
// The same effect always gives the same samples (the noise has a fixed seed).
func Generate(e Effect) []float64 {
	rng := rand.New(rand.NewPCG(uint64(e)+1, 7)) // fixed seed per effect
	noise := func() float64 { return rng.Float64()*2 - 1 }
	var s []float64
	switch e {
	case Gunshot: // a noise crack over a low thump
		s = synth(0.25, func(t float64) float64 {
			return noise()*decay(t, 25) + 0.7*math.Sin(2*math.Pi*70*t)*decay(t, 12)
		})
	case EnemyShot: // shorter, higher and quieter than the player's gun
		s = synth(0.18, func(t float64) float64 {
			return 0.6*noise()*decay(t, 35) + 0.3*square(sweep(t, 0.18, 900, 300))*decay(t, 20)
		})
	case EmptyClick:
		s = synth(0.03, func(t float64) float64 {
			return 0.5 * square(2000*t) * decay(t, 150)
		})
	case Reload: // click (magazine out), click (in), clack (slide)
		s = synth(1.2, func(t float64) float64 {
			v := 0.0
			for _, c := range [...]struct{ at, freq float64 }{{0, 3000}, {0.6, 2500}, {1.05, 1200}} {
				if dt := t - c.at; dt >= 0 {
					v += (0.5*noise() + 0.5*square(c.freq*dt)) * decay(dt, 80)
				}
			}
			return v
		})
	case ZombieGroan: // a low, wobbling sawtooth that sinks
		s = synth(0.9, func(t float64) float64 {
			f := 85 - 25*t/0.9 + 3*math.Sin(2*math.Pi*4*t)
			return 0.5 * saw(f*t) * attack(t, 0.15) * decay(t, 2.5)
		})
	case Hurt: // a falling buzz
		s = synth(0.25, func(t float64) float64 {
			return 0.5 * square(sweep(t, 0.25, 380, 140)) * decay(t, 6)
		})
	case Pickup: // two rising beeps
		s = synth(0.2, func(t float64) float64 {
			if t < 0.08 {
				return 0.4 * square(660*t) * decay(t, 10)
			}
			return 0.4 * square(990*(t-0.08)) * decay(t-0.08, 10)
		})
	case TierUp: // a rising arpeggio
		notes := [...]float64{440, 554, 659, 880}
		const step = 0.12
		s = synth(step*float64(len(notes)), func(t float64) float64 {
			i := min(int(t/step), len(notes)-1)
			dt := t - float64(i)*step
			return 0.35 * square(notes[i]*dt) * decay(dt, 8)
		})
	default:
		panic("audio: unknown effect")
	}
	return normalize(s)
}

// synth samples f (time in seconds -> value) for dur seconds.
func synth(dur float64, f func(t float64) float64) []float64 {
	s := make([]float64, int(dur*SampleRate))
	for i := range s {
		s[i] = f(float64(i) / SampleRate)
	}
	return s
}

// sweep returns the phase (cycles so far) of a tone whose pitch slides
// linearly from f0 to f1 Hz over dur seconds, at time t.
func sweep(t, dur, f0, f1 float64) float64 {
	return f0*t + (f1-f0)*t*t/(2*dur)
}

// square and saw are wave shapes for a phase in cycles (1 = one period).
func square(phase float64) float64 {
	if phase-math.Floor(phase) < 0.5 {
		return 1
	}
	return -1
}

func saw(phase float64) float64 {
	return 2*(phase-math.Floor(phase)) - 1
}

// decay fades from 1 toward 0; k sets how fast (larger = shorter).
func decay(t, k float64) float64 {
	return math.Exp(-k * t)
}

// attack rises from 0 to 1 over the first d seconds, so a sound doesn't
// start with a click.
func attack(t, d float64) float64 {
	return min(t/d, 1)
}

// normalize scales s so its loudest sample is at most peak.
func normalize(s []float64) []float64 {
	loudest := 0.0
	for _, v := range s {
		loudest = max(loudest, math.Abs(v))
	}
	if loudest > peak {
		for i := range s {
			s[i] *= peak / loudest
		}
	}
	return s
}

// PCM converts samples to 16-bit little-endian bytes, the format the
// player gives oto.
func PCM(s []float64) []byte {
	out := make([]byte, 2*len(s))
	for i, v := range s {
		// Clamp before converting; then the low byte goes first (little-endian).
		n := int16(math.Round(max(-1, min(1, v)) * math.MaxInt16))
		out[2*i] = byte(n)
		out[2*i+1] = byte(n >> 8)
	}
	return out
}
