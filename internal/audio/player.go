package audio

import (
	"bytes"
	"fmt"
	"math"

	"github.com/ebitengine/oto/v3"
)

// maxVoices is how many effects may play at once; more are skipped, so a
// burst of events can't pile up dozens of sounds.
const maxVoices = 16

// quieter lists the effects played below full volume (as generated). Every
// other effect plays at 1. (Before M17 this was a table of every effect's
// gain, and the effects added in M10-M16 were missing from it, so they
// played at 0: silent.)
var quieter = map[Effect]float64{
	ZombieGroan: 0.6, // a spawn far away shouldn't drown out the gun
	EmptyClick:  0.8,
	Footstep:    0.5, // it plays all the time
	HitTick:     0.7,
}

// Gain is how loud effect e plays, 0-1, before the volume setting and
// distance.
func Gain(e Effect) float64 {
	if g, ok := quieter[e]; ok {
		return g
	}
	return 1
}

// Pan positions (M17): a sound is panned to the nearest of panSteps
// positions from full left to full right, each made once when first used.
const panSteps = 9

// Player plays effects. Use New to make one; there can be only one per
// program (oto allows a single context).
type Player struct {
	ctx  *oto.Context
	mono [NumEffects][]float64
	// pcm holds each effect's stereo bytes per pan position, made the first
	// time that pair plays (nil until then).
	pcm [NumEffects][panSteps][]byte
	// playing holds the effects still sounding. oto stops a player that
	// is no longer referenced, so they're kept here until they finish.
	playing []*oto.Player
	on      bool
	volume  float64 // 0-1
}

// New opens the audio device (stereo since M17) and generates every
// effect. It returns an error if the device can't be opened (no sound
// card, no audio system).
func New() (*Player, error) {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:      SampleRate,
		ChannelCount:    2,
		Format:          oto.FormatSignedInt16LE,
		ApplicationName: "termfps",
	})
	if err != nil {
		return nil, fmt.Errorf("open audio: %w", err)
	}
	<-ready // a receive from the channel waits until oto has the device ready
	p := &Player{ctx: ctx, on: true, volume: 1}
	for e := Effect(0); e < NumEffects; e++ {
		p.mono[e] = Generate(e)
	}
	return p, nil
}

// Set turns sound on or off and sets the volume (0-1).
func (p *Player) Set(on bool, volume float64) {
	p.on, p.volume = on, volume
}

// Play starts effect e in the middle, at full loudness.
func (p *Player) Play(e Effect) {
	p.PlayAt(e, 1, 0)
}

// PlayAt starts effect e with loudness 0-1 (from distance) and pan -1
// (left) to 1 (right) (M17). Effects overlap: each gets its own oto player.
func (p *Player) PlayAt(e Effect, loudness, pan float64) {
	p.prune()
	if !p.on || p.volume == 0 || loudness <= 0 || len(p.playing) >= maxVoices {
		return
	}
	step := PanStep(pan)
	if p.pcm[e][step] == nil {
		p.pcm[e][step] = StereoPCM(p.mono[e], panOf(step))
	}
	pl := p.ctx.NewPlayer(bytes.NewReader(p.pcm[e][step])) // each player reads its own copy position
	pl.SetVolume(min(p.volume*Gain(e)*loudness, 1))
	pl.Play()
	p.playing = append(p.playing, pl)
}

// PanStep is the pan position nearest to pan (-1 to 1).
func PanStep(pan float64) int {
	pan = min(max(pan, -1), 1)
	return int(math.Round((pan + 1) / 2 * (panSteps - 1)))
}

// panOf is the pan (-1 to 1) of pan position step.
func panOf(step int) float64 {
	return float64(step)/(panSteps-1)*2 - 1
}

// PanGains are the left and right levels for pan -1 (left) to 1 (right):
// both 1 in the middle, the far side fading to 0 at the edge. (A constant
// power pan, scaled so the middle stays as loud as before M17.)
func PanGains(pan float64) (left, right float64) {
	a := (min(max(pan, -1), 1) + 1) * math.Pi / 4 // 0 (left) to pi/2 (right)
	return min(math.Sqrt2*math.Cos(a), 1), min(math.Sqrt2*math.Sin(a), 1)
}

// StereoPCM converts mono samples to 16-bit little-endian stereo bytes
// (left, right, left, right, ...) panned to pan.
func StereoPCM(s []float64, pan float64) []byte {
	l, r := PanGains(pan)
	out := make([]byte, 4*len(s))
	for i, v := range s {
		put16(out[4*i:], v*l)
		put16(out[4*i+2:], v*r)
	}
	return out
}

// Err is an error the audio device ran into after starting, or nil.
func (p *Player) Err() error {
	return p.ctx.Err()
}

// prune closes and drops the players that finished.
func (p *Player) prune() {
	kept := p.playing[:0] // reuses the slice's memory
	for _, pl := range p.playing {
		if pl.IsPlaying() {
			kept = append(kept, pl)
		} else {
			pl.Close()
		}
	}
	p.playing = kept
}
