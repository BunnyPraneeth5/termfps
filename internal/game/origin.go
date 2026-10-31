package game

// Origin is where an event in State.Events happened in the world (M17), so
// its sound can come from there: quieter far away, panned left or right.
// Events without a place (a pickup, the tier going up) have none.
type Origin struct {
	Event Event
	X, Y  float64
	Kind  EnemyKind // the enemy's kind, for events about one enemy
}

// NoticeRange: an enemy makes its "found you" sound (M17) the first time it
// sees the player this close or closer.
const NoticeRange = 8.0

// emitAt adds event ev, which happened at (x, y), to this frame's events.
// kind is the enemy's kind for events about one enemy (else ignored).
func emitAt(s *State, ev Event, x, y float64, kind EnemyKind) {
	s.Events = append(s.Events, ev)
	s.Origins = append(s.Origins, Origin{Event: ev, X: x, Y: y, Kind: kind})
}

// OriginOf returns where event ev happened this frame (the first one, if
// it happened more than once); ok is false if it didn't, or has no place.
func (s *State) OriginOf(ev Event) (o Origin, ok bool) {
	for _, o := range s.Origins {
		if o.Event == ev {
			return o, true
		}
	}
	return Origin{}, false
}
