package app

import (
	"math"

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
	game.EventShotFired:       audio.Gunshot,
	game.EventEmptyClick:      audio.EmptyClick,
	game.EventReloadStarted:   audio.Reload,
	game.EventEnemyShot:       audio.EnemyShot,
	game.EventPlayerHurt:      audio.Hurt,
	game.EventPickupTaken:     audio.Pickup,
	game.EventMedkitTaken:     audio.Medkit,
	game.EventArmorTaken:      audio.ArmorUp,
	game.EventArmorHit:        audio.ArmorHit,
	game.EventTierUp:          audio.TierUp,
	game.EventEnemySpawned:    audio.ZombieGroan, // "something arrived" cue
	game.EventShotgunFired:    audio.ShotgunBlast,
	game.EventRifleFired:      audio.RifleShot,
	game.EventGunSwitched:     audio.GunSwitch,
	game.EventGunTaken:        audio.GunPickup,
	game.EventExplosion:       audio.Explosion,
	game.EventExploderArmed:   audio.FuseBeep,
	game.EventRunnerSpawned:   audio.RunnerScreech,
	game.EventTankSpawned:     audio.TankRoar,
	game.EventExploderSpawned: audio.ExploderHiss,
	game.EventAcidBurn:        audio.AcidSizzle,
	game.EventFootstep:        audio.Footstep,
	game.EventOutOfBreath:     audio.OutOfBreath,
	game.EventEnemyHit:        audio.HitTick, // M16, with the hit marker
	game.EventHuntStarted:     audio.HuntHorn,
	game.EventWaveStarted:     audio.WaveStart,
	game.EventWaveCleared:     audio.WaveClear,
	game.EventWavesWon:        audio.Victory,
	game.EventShotgunReload:   audio.ShotgunReload,
	game.EventRifleReload:     audio.RifleReload,
}

// soundReplaces: when an event here comes with the event it names in the
// same frame, only its own sound plays (a shotgun shot is also a shot, but
// shouldn't also sound like the pistol).
var soundReplaces = map[game.Event]game.Event{
	game.EventShotgunFired: game.EventShotFired,
	game.EventRifleFired:   game.EventShotFired,
	// The M12 kinds arrive with their own sound instead of the groan.
	game.EventRunnerSpawned:   game.EventEnemySpawned,
	game.EventTankSpawned:     game.EventEnemySpawned,
	game.EventExploderSpawned: game.EventEnemySpawned,
	// The shotgun and rifle reload with their own sound (M17).
	game.EventShotgunReload: game.EventReloadStarted,
	game.EventRifleReload:   game.EventReloadStarted,
	// A wave starting may raise the tier too: the alarm says it (M15).
	game.EventWaveStarted: game.EventTierUp,
	// Acid sizzles instead of the hurt buzz (M14).
	game.EventAcidBurn: game.EventPlayerHurt,
}

// kindSounds: events about one enemy that sound different for each kind
// (M17). A kind not listed is silent for that event (an exploder's death
// is its explosion).
var kindSounds = map[game.Event]map[game.EnemyKind]audio.Effect{
	game.EventEnemyNoticed: {
		game.KindZombie: audio.ZombieGroan, game.KindShooter: audio.ShooterCall, game.KindRunner: audio.RunnerScreech,
		game.KindTank: audio.TankRoar, game.KindExploder: audio.ExploderHiss,
	},
	game.EventEnemyAttack: {game.KindZombie: audio.ZombieBite, game.KindRunner: audio.RunnerSlash, game.KindTank: audio.TankSmash},
	game.EventEnemyKilled: {
		game.KindZombie: audio.ZombieDie, game.KindShooter: audio.ShooterDie, game.KindRunner: audio.RunnerDie, game.KindTank: audio.TankDie,
	},
}

// centered: these events have a place, but their sound is feedback for
// the player and plays in the middle at full loudness (M17).
var centered = map[game.Event]bool{game.EventEnemyHit: true}

// cue is one sound to play: an effect, and where it comes from (nil = in
// the middle, at full loudness).
type cue struct {
	effect audio.Effect
	at     *game.Origin
}

// soundsFor returns the sounds for one frame's events and their origins,
// each effect at most once (three spawns in one frame groan once).
func soundsFor(events []game.Event, origins []game.Origin) []cue {
	var seen [audio.NumEffects]bool
	replaced := map[game.Event]bool{}
	for _, ev := range events {
		if r, ok := soundReplaces[ev]; ok {
			replaced[r] = true
		}
	}
	var out []cue
	add := func(e audio.Effect, at *game.Origin) {
		if !seen[e] {
			seen[e] = true
			out = append(out, cue{effect: e, at: at})
		}
	}
	for _, ev := range events {
		e, ok := eventSounds[ev]
		if !ok || replaced[ev] {
			continue
		}
		var at *game.Origin
		for i := range origins {
			if origins[i].Event == ev && !centered[ev] {
				at = &origins[i] // a pointer into the slice: valid until the next Update
				break
			}
		}
		add(e, at)
	}
	for i := range origins { // one sound per enemy kind and event
		if e, ok := kindSounds[origins[i].Event][origins[i].Kind]; ok {
			add(e, &origins[i])
		}
	}
	return out
}

// Sound in space (M17): loudness halves at hearHalf cells, never below
// hearMin, and sounds from behind are a little quieter (stereo can't tell
// front from back otherwise).
const (
	hearHalf    = 5.0
	hearMin     = 0.15
	behindQuiet = 0.8
)

// spatial returns how loud (0-1) and how far left (-1) or right (1) a
// sound at o is for player p.
func spatial(p game.Player, o game.Origin) (loudness, pan float64) {
	dx, dy := o.X-p.X, o.Y-p.Y
	d := math.Hypot(dx, dy)
	loudness = max(1/(1+d/hearHalf), hearMin)
	if d < 0.5 {
		return loudness, 0 // right here: in the middle
	}
	rel := math.Atan2(dy, dx) - p.Angle // 0 = straight ahead; growing = to the right (y points down)
	if math.Cos(rel) < 0 {
		loudness *= behindQuiet
	}
	return loudness, math.Sin(rel)
}

// playSounds plays the sounds for the last game.Update, each from where it
// happened.
func (a *App) playSounds() {
	if a.sound == nil {
		return
	}
	for _, c := range soundsFor(a.state.Events, a.state.Origins) {
		if c.at == nil {
			a.sound.Play(c.effect)
			continue
		}
		loudness, pan := spatial(a.state.Player, *c.at)
		a.sound.PlayAt(c.effect, loudness, pan)
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
		return "Sound is off: " + a.soundErr.Error() + ". The game runs without it."
	case a.sound != nil && a.sound.Err() != nil:
		return "Sound stopped: " + a.sound.Err().Error() + ". The game runs without it."
	}
	return ""
}
