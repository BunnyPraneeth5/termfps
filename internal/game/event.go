package game

// Event is something that happened during one Update. The app reads
// State.Events after Update to react (sounds, HUD effects), so the game
// package never needs to know who listens.
type Event int

const (
	// EventRenderModeChanged: V switched the render mode this frame.
	EventRenderModeChanged Event = iota + 1 // start at 1 so the zero value means "no event"
	EventShotFired                          // a round left the gun
	EventEmptyClick                         // tried to fire with no rounds at all
	EventReloadStarted                      // a reload began (by R or automatically)
	EventReloaded                           // a reload finished; rounds moved into the gun
	EventTargetDestroyed                    // a shot removed a red target block
	EventEnemyHit                           // a shot hit an enemy
	EventEnemyKilled                        // an enemy's HP reached 0
	EventPlayerHurt                         // the player lost HP
	EventPlayerDied                         // the player's HP reached 0
	EventEnemyShot                          // a shooter fired at the player (hit or miss)
	EventEnemySpawned                       // the survival spawner added an enemy
	EventTierUp                             // the difficulty tier went up
	EventPickupSpawned                      // an ammo box appeared
	EventPickupTaken                        // the player took an ammo box
	EventMedkitTaken                        // the player took a medkit (M10)
	EventArmorTaken                         // the player took armor (M10)
	EventArmorHit                           // armor took part of a hit (M10)
	EventGunSwitched                        // another gun is in hand (M11)
	EventShotgunFired                       // the shot was a shotgun's (also EventShotFired, M11)
	EventRifleFired                         // the shot was a rifle's (also EventShotFired, M11)
	EventGunTaken                           // the player found a new gun (M11)
	EventExplosion                          // something blew up (M12)
	EventExploderArmed                      // an exploder lit its fuse (M12)
	EventTankHit                            // a tank hit the player (M12)
	EventRunnerSpawned                      // the spawner added a runner (also EventEnemySpawned, M12)
	EventTankSpawned                        // the spawner added a tank (also EventEnemySpawned, M12)
	EventExploderSpawned                    // the spawner added an exploder (also EventEnemySpawned, M12)
	EventAcidBurn                           // acid burned the player (M14)
	EventFootstep                           // the player took a step (M16)
	EventOutOfBreath                        // sprinting used up the stamina (M16)
)

// HasEvent reports whether e happened in the last Update.
func (s *State) HasEvent(e Event) bool {
	for _, got := range s.Events {
		if got == e {
			return true
		}
	}
	return false
}
