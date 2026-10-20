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
