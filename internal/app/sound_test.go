package app

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/audio"
	"github.com/BunnyPraneeth5/termfps/internal/game"
	"github.com/BunnyPraneeth5/termfps/internal/term"
)

// effectsOf lists the effects of a frame's events without origins.
func effectsOf(events ...game.Event) []audio.Effect {
	var out []audio.Effect
	for _, c := range soundsFor(events, nil) {
		out = append(out, c.effect)
	}
	return out
}

func TestSoundsFor(t *testing.T) {
	got := effectsOf(
		game.EventShotFired, game.EventEnemyHit, // a hit ticks (M16)
		game.EventEnemySpawned, game.EventEnemySpawned, // two spawns: one groan
		game.EventPlayerHurt, game.EventRenderModeChanged,
	)
	want := []audio.Effect{audio.Gunshot, audio.HitTick, audio.ZombieGroan, audio.Hurt}
	if !slices.Equal(got, want) {
		t.Errorf("soundsFor = %v, want %v", got, want)
	}
	if got := soundsFor(nil, nil); len(got) != 0 {
		t.Errorf("no events gave sounds %v", got)
	}
}

// Every effect is played by some event, so none is generated for nothing.
func TestEveryEffectHasAnEvent(t *testing.T) {
	used := map[audio.Effect]bool{}
	for _, e := range eventSounds {
		used[e] = true
	}
	for _, byKind := range kindSounds { // M17
		for _, e := range byKind {
			used[e] = true
		}
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
	got := effectsOf(game.EventShotFired, game.EventShotgunFired)
	if !slices.Equal(got, []audio.Effect{audio.ShotgunBlast}) {
		t.Errorf("shotgun shot sounds = %v, want only the blast", got)
	}
	got = effectsOf(game.EventShotFired, game.EventRifleFired)
	if !slices.Equal(got, []audio.Effect{audio.RifleShot}) {
		t.Errorf("rifle shot sounds = %v, want only the rifle", got)
	}
}

// The M12 enemies arrive with their own sound instead of the groan.
func TestNewEnemySpawnSounds(t *testing.T) {
	got := effectsOf(game.EventEnemySpawned, game.EventTankSpawned)
	if !slices.Equal(got, []audio.Effect{audio.TankRoar}) {
		t.Errorf("tank spawn sounds = %v, want only the roar", got)
	}
}

// Enemy events sound by kind (M17): two kills of different kinds in one
// frame give both death sounds, each from where it happened.
func TestKindSounds(t *testing.T) {
	origins := []game.Origin{
		{Event: game.EventEnemyKilled, X: 3, Y: 1, Kind: game.KindShooter},
		{Event: game.EventEnemyKilled, X: 9, Y: 1, Kind: game.KindTank},
		{Event: game.EventEnemyAttack, X: 2, Y: 1, Kind: game.KindRunner},
	}
	got := soundsFor([]game.Event{game.EventEnemyKilled, game.EventEnemyKilled, game.EventEnemyAttack}, origins)
	var effects []audio.Effect
	for _, c := range got {
		effects = append(effects, c.effect)
		if c.at == nil {
			t.Errorf("effect %d plays in the middle, want it from its enemy", c.effect)
		}
	}
	want := []audio.Effect{audio.ShooterDie, audio.TankDie, audio.RunnerSlash}
	if !slices.Equal(effects, want) {
		t.Errorf("sounds = %v, want %v", effects, want)
	}
}

// The shotgun and rifle reload with their own sound (M17).
func TestReloadSounds(t *testing.T) {
	if got := effectsOf(game.EventReloadStarted, game.EventShotgunReload); !slices.Equal(got, []audio.Effect{audio.ShotgunReload}) {
		t.Errorf("shotgun reload sounds = %v, want only its own", got)
	}
	if got := effectsOf(game.EventReloadStarted); !slices.Equal(got, []audio.Effect{audio.Reload}) {
		t.Errorf("pistol reload sounds = %v, want the old reload", got)
	}
}

// The hit tick is feedback: it plays in the middle even though the hit
// has a place.
func TestHitTickCentered(t *testing.T) {
	got := soundsFor([]game.Event{game.EventEnemyHit}, []game.Origin{{Event: game.EventEnemyHit, X: 20, Y: 1}})
	if len(got) != 1 || got[0].effect != audio.HitTick || got[0].at != nil {
		t.Errorf("hit sounds = %+v, want the tick in the middle", got)
	}
}

// Sounds pan toward their source and get quieter with distance and from
// behind (M17).
func TestSpatial(t *testing.T) {
	p := game.Player{X: 5, Y: 5, Angle: 0} // facing east: south (y+) is to the right
	_, right := spatial(p, game.Origin{X: 5, Y: 8})
	_, left := spatial(p, game.Origin{X: 5, Y: 2})
	_, ahead := spatial(p, game.Origin{X: 9, Y: 5})
	if right < 0.99 || left > -0.99 || math.Abs(ahead) > 1e-9 {
		t.Errorf("pan right %.2f, left %.2f, ahead %.2f; want 1, -1, 0", right, left, ahead)
	}
	near, _ := spatial(p, game.Origin{X: 7, Y: 5})
	far, _ := spatial(p, game.Origin{X: 25, Y: 5})
	behind, _ := spatial(p, game.Origin{X: 3, Y: 5})
	if !(near > far) || far < hearMin || !(behind < near) {
		t.Errorf("loudness near %.2f, far %.2f, behind %.2f; want near > far >= %.2f, behind < near", near, far, behind, hearMin)
	}
	if l, _ := spatial(p, game.Origin{X: 5 + hearHalf, Y: 5}); math.Abs(l-0.5) > 1e-9 {
		t.Errorf("loudness at %.0f cells = %.2f, want 0.5", hearHalf, l)
	}
}
