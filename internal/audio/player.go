package audio

import (
	"bytes"
	"fmt"

	"github.com/ebitengine/oto/v3"
)

// maxVoices is how many effects may play at once; more are skipped, so a
// burst of events can't pile up dozens of sounds.
const maxVoices = 16

// gain makes some effects quieter than others (1 = as generated).
var gain = [NumEffects]float64{
	ZombieGroan: 0.6, // a spawn far away shouldn't drown out the gun
	EmptyClick:  0.8,
	Gunshot:     1, Reload: 1, EnemyShot: 1, Hurt: 1, Pickup: 1, TierUp: 1,
}

// Player plays effects. Use New to make one; there can be only one per
// program (oto allows a single context).
type Player struct {
	ctx *oto.Context
	pcm [NumEffects][]byte
	// playing holds the effects still sounding. oto stops a player that
	// is no longer referenced, so they're kept here until they finish.
	playing []*oto.Player
	on      bool
	volume  float64 // 0-1
}

// New opens the audio device and generates every effect. It returns an
// error if the device can't be opened (no sound card, no audio system).
func New() (*Player, error) {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:      SampleRate,
		ChannelCount:    1,
		Format:          oto.FormatSignedInt16LE,
		ApplicationName: "termfps",
	})
	if err != nil {
		return nil, fmt.Errorf("open audio: %w", err)
	}
	<-ready // a receive from the channel waits until oto has the device ready
	p := &Player{ctx: ctx, on: true, volume: 1}
	for e := Effect(0); e < NumEffects; e++ {
		p.pcm[e] = PCM(Generate(e))
	}
	return p, nil
}

// Set turns sound on or off and sets the volume (0-1).
func (p *Player) Set(on bool, volume float64) {
	p.on, p.volume = on, volume
}

// Play starts effect e. Effects overlap: each gets its own oto player.
func (p *Player) Play(e Effect) {
	p.prune()
	if !p.on || p.volume == 0 || len(p.playing) >= maxVoices {
		return
	}
	pl := p.ctx.NewPlayer(bytes.NewReader(p.pcm[e])) // each player reads its own copy position
	pl.SetVolume(p.volume * gain[e])
	pl.Play()
	p.playing = append(p.playing, pl)
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
