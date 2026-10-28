package game

import "testing"

// Looking east, the player sees the room ahead up to the walls, and the
// cells right around them, but not past the inner wall or behind a closed
// door (M8).
func TestMarkSeen(t *testing.T) {
	s := stateFrom(t, "##########\n"+
		"#P.....#.#\n"+
		"#......#.#\n"+
		"####D###.#\n"+
		"#........#\n"+
		"##########\n")
	Update(s, Input{}, frameDT) // facing east from (1, 1)
	m := s.Map
	at := func(x, y int) bool { return s.Seen[y*m.W+x] }
	if !at(1, 1) || !at(2, 2) || !at(6, 1) {
		t.Error("the player's cell, a neighbor, or a cell ahead isn't seen")
	}
	if !at(7, 1) {
		t.Error("the wall that stopped the view isn't seen")
	}
	if at(8, 1) {
		t.Error("the corridor behind the inner wall is seen")
	}
	if at(4, 4) {
		t.Error("the room behind the closed door is seen")
	}
}
