package world

// DistanceField returns, for every cell, how many steps (up, down, left,
// right) it takes to walk from (fromX, fromY) to it, or -1 if it can't be
// reached or is solid. Enemies walk downhill on it to reach the player
// around walls. It reuses buf when it is big enough.
//
// It's a breadth-first search: cells are visited in rings of equal
// distance, using a slice as a first-in-first-out queue.
func DistanceField(m *Map, fromX, fromY int, buf []int) []int {
	n := m.W * m.H
	if cap(buf) >= n {
		buf = buf[:n]
	} else {
		buf = make([]int, n)
	}
	for i := range buf {
		buf[i] = -1
	}
	if m.Blocked(fromX, fromY) {
		return buf
	}
	start := fromY*m.W + fromX
	buf[start] = 0
	queue := []int{start}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%m.W, i/m.W
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if m.Blocked(nx, ny) {
				continue // walls, blocking props (M2); also covers outside the map
			}
			j := ny*m.W + nx
			if buf[j] == -1 {
				buf[j] = buf[i] + 1
				queue = append(queue, j)
			}
		}
	}
	return buf
}

// LineOfSight reports whether nothing solid, and no prop that blocks shots
// (M2: cover), stands between (ax, ay) and (bx, by). It casts a ray toward
// b; with the direction b-a (not unit length), distances are measured in
// "trips from a to b", so a hit at 1 or more is at or past b.
func LineOfSight(m *Map, ax, ay, bx, by float64) bool {
	dx, dy := bx-ax, by-ay
	if dx == 0 && dy == 0 {
		return true
	}
	return CastRay(m, ax, ay, dx, dy).PerpDist >= 1 && ShotDistance(m, ax, ay, dx, dy, 1) >= 1
}
