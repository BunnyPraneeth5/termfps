package game

import (
	"math"
	"testing"
)

// The first box comes 20-40 s into a run; at most 3 lie around at once.
func TestPickupTimingAndLimit(t *testing.T) {
	s := survivalState(t)
	s.Enemies = nil
	var first float64
	for range 5 * 60 * 30 {
		Update(s, Input{}, frameDT)
		s.Player.HP = PlayerMaxHP
		s.Enemies = s.Enemies[:0] // only boxes matter here
		if count(s, EventPickupSpawned) > 0 && first == 0 {
			first = s.Time
		}
		if len(s.Pickups) > MaxPickups {
			t.Fatalf("%d boxes on the map, max %d", len(s.Pickups), MaxPickups)
		}
	}
	if first < PickupEveryMin || first > PickupEveryMax+frameDT {
		t.Errorf("first box at %.1f s, want between %.0f and %.0f s", first, PickupEveryMin, PickupEveryMax)
	}
	if len(s.Pickups) != MaxPickups {
		t.Errorf("after 5 minutes untouched: %d boxes, want %d", len(s.Pickups), MaxPickups)
	}
	for _, pk := range s.Pickups {
		if d := math.Hypot(pk.X-s.Player.X, pk.Y-s.Player.Y); d < pickupMinDistance {
			t.Errorf("box %.1f cells from the player, want at least %.0f", d, pickupMinDistance)
		}
		if s.Map.IsSolid(int(pk.X), int(pk.Y)) {
			t.Errorf("box inside a wall at (%.1f, %.1f)", pk.X, pk.Y)
		}
	}
}

func TestTakePickup(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	s.Pickups = []Pickup{{X: s.Player.X + 0.4, Y: s.Player.Y, Amount: PickupAmmo}}
	Update(s, Input{}, frameDT)
	if len(s.Pickups) != 0 || s.Player.Weapon.Reserve != StartReserve+PickupAmmo || count(s, EventPickupTaken) != 1 {
		t.Errorf("after stepping on a box: %d boxes, reserve %d; want 0 and %d", len(s.Pickups), s.Player.Weapon.Reserve, StartReserve+PickupAmmo)
	}
}

func TestPickupCappedAndLeftWhenFull(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	s.Player.Weapon.Reserve = MaxReserve - 10
	s.Pickups = []Pickup{{X: s.Player.X, Y: s.Player.Y, Amount: PickupAmmo}}
	Update(s, Input{}, frameDT)
	if s.Player.Weapon.Reserve != MaxReserve || len(s.Pickups) != 0 {
		t.Fatalf("reserve %d, boxes %d; want capped at %d and the box taken", s.Player.Weapon.Reserve, len(s.Pickups), MaxReserve)
	}
	s.Pickups = []Pickup{{X: s.Player.X, Y: s.Player.Y, Amount: PickupAmmo}}
	Update(s, Input{}, frameDT)
	if len(s.Pickups) != 1 {
		t.Error("box taken with a full reserve; it should stay for later")
	}
}

func TestPickupOutOfReach(t *testing.T) {
	s := stateFrom(t, "######\n#P...#\n######\n")
	s.Pickups = []Pickup{{X: s.Player.X + 1.5, Y: s.Player.Y, Amount: PickupAmmo}}
	Update(s, Input{}, frameDT)
	if len(s.Pickups) != 1 {
		t.Error("box 1.5 cells away was taken")
	}
}

// A medkit heals up to full; at full HP it stays on the floor (M10).
func TestTakeMedkit(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	s.Player.HP = 80
	s.Pickups = []Pickup{{X: s.Player.X, Y: s.Player.Y, Kind: Medkit, Amount: MedkitHeal}}
	Update(s, Input{}, frameDT)
	if s.Player.HP != PlayerMaxHP || len(s.Pickups) != 0 || count(s, EventMedkitTaken) != 1 {
		t.Fatalf("HP %d, %d pickups after a medkit at 80 HP; want %d and 0", s.Player.HP, len(s.Pickups), PlayerMaxHP)
	}
	s.Pickups = []Pickup{{X: s.Player.X, Y: s.Player.Y, Kind: Medkit, Amount: MedkitHeal}}
	Update(s, Input{}, frameDT)
	if len(s.Pickups) != 1 {
		t.Error("medkit taken at full HP; it should stay for later")
	}
}

// Armor adds up to MaxArmor; at full armor it stays on the floor (M10).
func TestTakeArmor(t *testing.T) {
	s := stateFrom(t, "#####\n#P..#\n#####\n")
	s.Player.Armor = 70
	s.Pickups = []Pickup{{X: s.Player.X, Y: s.Player.Y, Kind: ArmorVest, Amount: ArmorPickup}}
	Update(s, Input{}, frameDT)
	if s.Player.Armor != MaxArmor || len(s.Pickups) != 0 || count(s, EventArmorTaken) != 1 {
		t.Fatalf("armor %d, %d pickups; want %d and 0", s.Player.Armor, len(s.Pickups), MaxArmor)
	}
	s.Pickups = []Pickup{{X: s.Player.X, Y: s.Player.Y, Kind: ArmorVest, Amount: ArmorPickup}}
	Update(s, Input{}, frameDT)
	if len(s.Pickups) != 1 {
		t.Error("armor taken at full armor; it should stay for later")
	}
}

// Over many pickups, all three kinds show up, ammo most often.
func TestPickupMix(t *testing.T) {
	s := survivalState(t)
	var n [3]int
	for range 1000 {
		n[newPickup(s, 1.5, 1.5).Kind]++
	}
	if n[AmmoBox] < n[Medkit] || n[Medkit] == 0 || n[ArmorVest] == 0 {
		t.Errorf("pickup mix ammo/medkit/armor = %v, want all three, ammo most", n)
	}
}
