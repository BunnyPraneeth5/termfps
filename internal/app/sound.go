package app

import (
	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
)

// Sound is the audio player main opened, or why there is none (P1-17).
type Sound struct {
	Player *audio.Player // nil when Err is set
	Err    error
}

// eventSounds says which effect each game event plays (D37). Events not
// listed are silent.
var eventSounds = map[game.Event]audio.Effect{
	game.EventShotFired:     audio.Gunshot,
	game.EventEmptyClick:    audio.EmptyClick,
	game.EventReloadStarted: audio.Reload,
	game.EventEnemyShot:     audio.EnemyShot,
	game.EventPlayerHurt:    audio.Hurt,
	game.EventPickupTaken:   audio.Pickup,
	game.EventMedkitTaken:   audio.Medkit,
	game.EventArmorTaken:    audio.ArmorUp,
	game.EventArmorHit:      audio.ArmorHit,
	game.EventTierUp:        audio.TierUp,
	game.EventEnemySpawned:  audio.ZombieGroan, // "something arrived" cue
	game.EventShotgunFired:  audio.ShotgunBlast,
	game.EventRifleFired:    audio.RifleShot,
	game.EventGunSwitched:   audio.GunSwitch,
	game.EventGunTaken:      audio.GunPickup,
}

// soundReplaces: when an event here comes with the event it names in the
// same frame, only its own sound plays (a shotgun shot is also a shot, but
// shouldn't also sound like the pistol).
var soundReplaces = map[game.Event]game.Event{
	game.EventShotgunFired: game.EventShotFired,
	game.EventRifleFired:   game.EventShotFired,
}

// soundsFor returns the effects for one frame's events, each at most once
// (three spawns in one frame groan once).
func soundsFor(events []game.Event) []audio.Effect {
	var seen [audio.NumEffects]bool
	replaced := map[game.Event]bool{}
	for _, ev := range events {
		if r, ok := soundReplaces[ev]; ok {
			replaced[r] = true
		}
	}
	var out []audio.Effect
	for _, ev := range events {
		e, ok := eventSounds[ev]
		if ok && !seen[e] && !replaced[ev] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}

// playSounds plays the effects for the last game.Update.
func (a *App) playSounds() {
	if a.sound == nil {
		return
	}
	for _, e := range soundsFor(a.state.Events) {
		a.sound.Play(e)
	}
}

// applySound passes the sound settings to the player.
func (a *App) applySound() {
	if a.sound != nil {
		a.sound.Set(a.cfg.Sound, float64(a.cfg.Volume)/100)
	}
}

// soundProblem is the reason sound doesn't work, for the menus, or "".
func (a *App) soundProblem() string {
	switch {
	case a.soundErr != nil:
		return "Sound unavailable: " + a.soundErr.Error()
	case a.sound != nil && a.sound.Err() != nil:
		return "Sound stopped: " + a.sound.Err().Error()
	}
	return ""
}
