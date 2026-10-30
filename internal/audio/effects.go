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
	Medkit        // M10
	ArmorUp       // M10: picking up armor
	ArmorHit      // M10: armor took part of a hit
	ShotgunBlast  // M11
	RifleShot     // M11
	GunSwitch     // M11
	GunPickup     // M11: finding a new gun
	Explosion     // M12
	FuseBeep      // M12: an exploder lit its fuse
	TankSmash     // M12: a tank hit you
	RunnerScreech // M12: a runner arrived
	TankRoar      // M12: a tank arrived
	ExploderHiss  // M12: an exploder arrived
	AcidSizzle    // M14: acid burned you
	NumEffects    // not an effect: how many there are
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
	case Medkit: // a soft rising chord: healing
		s = synth(0.35, func(t float64) float64 {
			v := 0.0
			for i, f := range [...]float64{523, 659, 784} {
				if dt := t - 0.06*float64(i); dt >= 0 {
					v += 0.25 * math.Sin(2*math.Pi*f*dt) * attack(dt, 0.01) * decay(dt, 7)
				}
			}
			return v
		})
	case ArmorUp: // a metal clink, then a low buckle
		s = synth(0.3, func(t float64) float64 {
			v := 0.35 * (math.Sin(2*math.Pi*1800*t) + math.Sin(2*math.Pi*2650*t)) * decay(t, 18)
			if dt := t - 0.12; dt >= 0 {
				v += 0.5 * square(sweep(dt, 0.18, 160, 220)) * decay(dt, 14)
			}
			return v
		})
	case ArmorHit: // a dull metal clank
		s = synth(0.2, func(t float64) float64 {
			return (0.4*noise()*decay(t, 40) + 0.5*math.Sin(2*math.Pi*420*t)*decay(t, 16) + 0.3*math.Sin(2*math.Pi*1130*t)*decay(t, 25))
		})
	case ShotgunBlast: // a wide, long noise burst over a deep boom
		s = synth(0.45, func(t float64) float64 {
			return 0.9*noise()*decay(t, 12) + 0.8*math.Sin(2*math.Pi*sweep(t, 0.45, 90, 40))*decay(t, 7)
		})
	case RifleShot: // a sharp crack with a ringing tail
		s = synth(0.6, func(t float64) float64 {
			return noise()*decay(t, 45) + 0.4*noise()*decay(t, 6)*0.5 + 0.5*math.Sin(2*math.Pi*120*t)*decay(t, 9)
		})
	case GunSwitch: // two quick metal clicks
		s = synth(0.15, func(t float64) float64 {
			v := 0.5 * square(2200*t) * decay(t, 120)
			if dt := t - 0.07; dt >= 0 {
				v += 0.5 * square(1500*dt) * decay(dt, 90)
			}
			return v
		})
	case GunPickup: // a rack (noise slide) and a rising beep
		s = synth(0.4, func(t float64) float64 {
			v := 0.5 * noise() * attack(t, 0.02) * decay(t, 18)
			if dt := t - 0.18; dt >= 0 {
				v += 0.35 * square(sweep(dt, 0.22, 500, 1100)) * decay(dt, 9)
			}
			return v
		})
	case Explosion: // a long, low roar of noise with a deep boom
		low := 0.0
		s = synth(1.3, func(t float64) float64 {
			low += (noise() - low) * 0.05 // a simple low-pass: rumbling noise
			return 2.5*low*attack(t, 0.005)*decay(t, 2.5) + 0.8*math.Sin(2*math.Pi*sweep(t, 1.3, 60, 25))*decay(t, 3) + 0.5*noise()*decay(t, 20)
		})
	case FuseBeep: // three fast high beeps
		s = synth(0.45, func(t float64) float64 {
			dt := math.Mod(t, 0.15)
			return 0.4 * square(1800*dt) * decay(dt, 30)
		})
	case TankSmash: // a heavy thud with a crunch
		s = synth(0.4, func(t float64) float64 {
			return 0.9*math.Sin(2*math.Pi*sweep(t, 0.4, 110, 45))*decay(t, 9) + 0.5*noise()*decay(t, 30)
		})
	case RunnerScreech: // a rising, rough shriek
		s = synth(0.5, func(t float64) float64 {
			f := sweep(t, 0.5, 600, 1300) + 4*math.Sin(2*math.Pi*30*t)
			return 0.4 * saw(f) * attack(t, 0.05) * decay(t, 5)
		})
	case TankRoar: // a very low, long growl
		s = synth(1.2, func(t float64) float64 {
			f := 55 - 15*t + 4*math.Sin(2*math.Pi*6*t)
			return 0.6 * (saw(f*t) + 0.5*square(f*t/2)) * attack(t, 0.2) * decay(t, 2)
		})
	case ExploderHiss: // a hiss that swells, like a lit fuse
		s = synth(0.8, func(t float64) float64 {
			return 0.5 * noise() * attack(t, 0.4) * decay(t, 4) * (0.7 + 0.3*math.Sin(2*math.Pi*9*t))
		})
	case AcidSizzle: // a short, bright, crackling hiss
		high := 0.0
		s = synth(0.3, func(t float64) float64 {
			n := noise()
			high, n = n, n-high // a simple high-pass: only the hiss
			crackle := 0.0
			if rng.Float64() < 0.004 {
				crackle = 0.8
			}
			return (0.35*n + crackle) * attack(t, 0.01) * decay(t, 9)
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
