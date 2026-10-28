package game

import (
	"math"
	"testing"

	"github.com/BunnyPraneeth5/termfps/internal/world"
)

// doorHall has a door at (2, 3) between a room at the top and a long
// corridor below.
const doorHall = "#####\n" +
	"#P..#\n" +
	"#...#\n" +
	"##D##\n" +
	"#...#\n" +
	"#...#\n" +
	"#...#\n" +
	"#...#\n" +
	"#...#\n" +
	"#####\n"

// Walking up to a door opens it, and the player walks through.
func TestPlayerOpensDoorAndWalksThrough(t *testing.T) {
	s := stateFrom(t, doorHall)
	s.Player.X, s.Player.Y, s.Player.Angle = 2.5, 1.5, math.Pi/2 // facing south
	for range 2 * 30 {
		Update(s, Input{Forward: true}, frameDT)
	}
	if s.Player.Y < 4.5 {
		t.Errorf("player at y %v after 2 s, want through the door (past y 4.5)", s.Player.Y)
	}
}

// A door stays shut while nobody is near, closes again DoorHold seconds
// after the player leaves, and never closes on someone inside it.
func TestDoorClosesAfterHold(t *testing.T) {
	s := stateFrom(t, doorHall)
	d := s.Map.DoorAt(2, 3)
	s.Player.X, s.Player.Y = 2.5, 8.5 // far below
	Update(s, Input{}, frameDT)
	if d.Open != 0 {
		t.Fatalf("door opened with nobody near: %v", d.Open)
	}
	s.Player.Y = 4.5 // one cell from its center
	for range 30 {
		Update(s, Input{}, frameDT)
	}
	if d.Open != 1 {
		t.Fatalf("door %v open after 1 s nearby, want 1", d.Open)
	}
	s.Player.Y = 8.5
	for range int((DoorHold - 0.5) * 30) {
		Update(s, Input{}, frameDT)
	}
	if d.Open != 1 {
		t.Errorf("door started closing before DoorHold: %v", d.Open)
	}
	for range 2 * 30 {
		Update(s, Input{}, frameDT)
	}
	if d.Open != 0 {
		t.Errorf("door %v open 1.5 s after the hold, want closed", d.Open)
	}

	d.Open = 1
	s.Player.X, s.Player.Y = 2.5, 3.5 // standing in the doorway
	for range 10 * 30 {
		Update(s, Input{}, frameDT)
	}
	if d.Open != 1 {
		t.Errorf("door closed on the player: %v", d.Open)
	}
}

// Enemies open doors too.
func TestEnemyOpensDoor(t *testing.T) {
	s := stateFrom(t, doorHall)
	s.Player.X, s.Player.Y = 2.5, 8.5
	s.Enemies = append(s.Enemies, newZombie(2.5, 2.5))
	for range 15 {
		Update(s, Input{}, frameDT)
	}
	if d := s.Map.DoorAt(2, 3); d.Open <= 0 {
		t.Error("door didn't open for a zombie next to it")
	}
}

// Enemies and boxes never appear in a door.
func TestFreeCellSkipsDoors(t *testing.T) {
	m, err := world.Parse([]byte(doorHall))
	if err != nil {
		t.Fatal(err)
	}
	if freeCell(m, 2, 3) || !freeCell(m, 2, 4) || freeCell(m, 0, 0) {
		t.Error("freeCell: want false for the door and walls, true for floor")
	}
}
