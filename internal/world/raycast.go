package world

import "math"

// Hit is where a ray stopped.
type Hit struct {
	PerpDist float64 // distance along the view direction, not along the ray (prevents fisheye)
	Side     int     // 0 = hit an X-side face (east/west), 1 = Y-side face (north/south)
	MapX     int
	MapY     int
	Cell     Cell
	// WallX is where along the wall face the ray hit, from 0 to 1 (the
	// fraction of the cell). Textures use it to pick a column (P1-10).
	WallX float64
}

// CastRay walks the grid from (posX, posY) along (rayDirX, rayDirY) with the
// DDA algorithm and returns the first wall it hits.
//
// The ray direction doesn't need to be unit length. When it is
// dir + plane*cameraX (with a unit dir), PerpDist is the distance
// perpendicular to the camera plane, which is what wall height needs.
//
// A zero direction (0, 0) can't hit anything and returns PerpDist = +Inf.
func CastRay(m *Map, posX, posY, rayDirX, rayDirY float64) Hit {
	mapX, mapY := int(math.Floor(posX)), int(math.Floor(posY))
	hit := Hit{PerpDist: math.Inf(1), MapX: mapX, MapY: mapY, Cell: m.At(mapX, mapY)}
	walkCells(m, posX, posY, rayDirX, rayDirY, func(x, y, side int, enter float64) bool {
		hit.MapX, hit.MapY, hit.Cell = x, y, m.At(x, y)
		if hit.Cell == Door {
			if d := m.DoorAt(x, y); d != nil {
				var ok bool
				hit, ok = hitDoor(hit, d, posX, posY, rayDirX, rayDirY)
				return !ok // a hit stops the ray; through the gap it goes on
			}
		}
		if !m.IsSolid(x, y) {
			return true // keep walking
		}
		// The hit point's other coordinate: y on an X-side face, x on a
		// Y-side face. Its fraction is the position along the face.
		along := posY + enter*rayDirY
		if side == 1 {
			along = posX + enter*rayDirX
		}
		hit.PerpDist, hit.Side, hit.WallX = enter, side, along-math.Floor(along)
		return false
	})
	return hit
}

// hitDoor checks a ray in door d's cell against the door's panel, which
// runs through the middle of the cell. The panel slides toward its high
// end as it opens, so the part from Open to 1 along it is still there.
// WallX is measured from the panel's edge, so its texture moves with it.
func hitDoor(hit Hit, d *SlidingDoor, posX, posY, rayDirX, rayDirY float64) (Hit, bool) {
	var t, along float64
	side := 1
	if d.Across { // panel on the line y = Y + 0.5
		if rayDirY == 0 {
			return hit, false
		}
		t = (float64(d.Y) + 0.5 - posY) / rayDirY
		along = posX + t*rayDirX - float64(d.X)
	} else { // panel on the line x = X + 0.5
		if rayDirX == 0 {
			return hit, false
		}
		t = (float64(d.X) + 0.5 - posX) / rayDirX
		along = posY + t*rayDirY - float64(d.Y)
		side = 0
	}
	if t <= 0 || along < 0 || along >= 1 || along < d.Open {
		return hit, false // behind us, outside the cell, or through the gap
	}
	hit.PerpDist, hit.Side, hit.WallX = t, side, along-d.Open
	return hit, true
}

// walkCells visits the cells a ray from (posX, posY) along (rayDirX,
// rayDirY) enters, in order, with the DDA algorithm: visit gets the cell,
// the side of the face the ray came through (0 = an X-side face, crossing a
// vertical grid line; 1 = a Y-side face) and the ray length where it
// entered (in units of the direction's length). It stops when visit
// returns false, or after W+H+2 cells (outside the map is solid, so a ray
// that keeps going is a bug the cap guards against).
func walkCells(m *Map, posX, posY, rayDirX, rayDirY float64, visit func(x, y, side int, enter float64) bool) {
	if rayDirX == 0 && rayDirY == 0 {
		return
	}
	mapX, mapY := int(math.Floor(posX)), int(math.Floor(posY))

	// Ray length needed to cross one full cell in x (or y). Dividing a float
	// by zero gives +Inf in Go (no panic), which means "never crosses".
	deltaX := math.Abs(1 / rayDirX)
	deltaY := math.Abs(1 / rayDirY)

	// Step direction, and ray length to the first x (or y) grid line.
	// When rayDirX is 0 the else branch runs, so the factor is never 0 and
	// 0 * Inf (which would be NaN) can't happen.
	var stepX, stepY int
	var sideDistX, sideDistY float64
	if rayDirX < 0 {
		stepX = -1
		sideDistX = (posX - float64(mapX)) * deltaX
	} else {
		stepX = 1
		sideDistX = (float64(mapX) + 1 - posX) * deltaX
	}
	if rayDirY < 0 {
		stepY = -1
		sideDistY = (posY - float64(mapY)) * deltaY
	} else {
		stepY = 1
		sideDistY = (float64(mapY) + 1 - posY) * deltaY
	}

	for range m.W + m.H + 2 {
		var side int
		var enter float64
		if sideDistX < sideDistY {
			enter = sideDistX
			sideDistX += deltaX
			mapX += stepX
		} else {
			enter = sideDistY
			sideDistY += deltaY
			mapY += stepY
			side = 1
		}
		if !visit(mapX, mapY, side, enter) {
			return
		}
	}
}

// ShotDistance returns how far along (dirX, dirY) from (posX, posY) the
// first prop that blocks shots is (M2), as a ray length like CastRay's, or
// +Inf if there is none before a wall or maxDist. A prop counts as its
// whole cell.
func ShotDistance(m *Map, posX, posY, dirX, dirY, maxDist float64) float64 {
	dist := math.Inf(1)
	walkCells(m, posX, posY, dirX, dirY, func(x, y, _ int, enter float64) bool {
		if enter > maxDist || m.IsSolid(x, y) {
			return false
		}
		if m.At(x, y).BlocksShot() {
			dist = enter
			return false
		}
		return true
	})
	return dist
}
