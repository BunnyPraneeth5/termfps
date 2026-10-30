package app

import (
	"errors"
	"slices"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
)

func TestSoundsFor(t *testing.T) {
	got := soundsFor([]game.Event{
		game.EventShotFired, game.EventEnemyHit, // a hit ticks (M16)
		game.EventEnemySpawned, game.EventEnemySpawned, // two spawns: one groan
		game.EventPlayerHurt, game.EventRenderModeChanged,
	})
	want := []audio.Effect{audio.Gunshot, audio.HitTick, audio.ZombieGroan, audio.Hurt}
	if !slices.Equal(got, want) {
		t.Errorf("soundsFor = %v, want %v", got, want)
	}
	if got := soundsFor(nil); len(got) != 0 {
		t.Errorf("no events gave sounds %v", got)
	}
}

// Every effect is played by some event, so none is generated for nothing.
func TestEveryEffectHasAnEvent(t *testing.T) {
	used := map[audio.Effect]bool{}
	for _, e := range eventSounds {
		used[e] = true
	}
	for e := audio.Effect(0); e < audio.NumEffects; e++ {
		if !used[e] {
			t.Errorf("effect %d is never played", e)
		}
	}
}

// Without a player the game still runs, and the menu says why.
func TestSoundUnavailableShowsInMenu(t *testing.T) {
	a, s := newMenuApp(t)
	a.soundErr = errors.New("no audio device")
	a.applySound() // no player: must not crash
	now := t0
	press(a, &now)
	if !screenHas(s, "Sound unavailable: no audio device") {
		t.Error("title menu doesn't say why sound is off")
	}
	a.newRun()
	a.state.Player.Weapon().Mag = 25
	press(a, &now, key(term.KeyRune, ' ')) // fire with no player: no crash
	if a.state.Player.Weapon().Mag != 24 {
		t.Error("Space didn't fire without a sound player")
	}
}

// A shotgun or rifle shot plays its own sound instead of the pistol's (M11).
func TestGunShotSounds(t *testing.T) {
	got := soundsFor([]game.Event{game.EventShotFired, game.EventShotgunFired})
	if !slices.Equal(got, []audio.Effect{audio.ShotgunBlast}) {
		t.Errorf("shotgun shot sounds = %v, want only the blast", got)
	}
	got = soundsFor([]game.Event{game.EventShotFired, game.EventRifleFired})
	if !slices.Equal(got, []audio.Effect{audio.RifleShot}) {
		t.Errorf("rifle shot sounds = %v, want only the rifle", got)
	}
}

// The M12 enemies arrive with their own sound instead of the groan.
func TestNewEnemySpawnSounds(t *testing.T) {
	got := soundsFor([]game.Event{game.EventEnemySpawned, game.EventTankSpawned})
	if !slices.Equal(got, []audio.Effect{audio.TankRoar}) {
		t.Errorf("tank spawn sounds = %v, want only the roar", got)
	}
}
