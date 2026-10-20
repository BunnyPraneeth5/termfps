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
	if rayDirX == 0 && rayDirY == 0 {
		return Hit{PerpDist: math.Inf(1), MapX: mapX, MapY: mapY, Cell: m.At(mapX, mapY)}
	}

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

	// Outside the map counts as wall, so any ray stops within W+H steps.
	// The cap is a safety net against an endless loop.
	maxSteps := m.W + m.H + 2
	for range maxSteps {
		side := 0
		if sideDistX < sideDistY {
			sideDistX += deltaX
			mapX += stepX
		} else {
			sideDistY += deltaY
			mapY += stepY
			side = 1
		}
		if m.IsSolid(mapX, mapY) {
			// sideDist went one step past the face we hit, so step back.
			perp := sideDistX - deltaX
			// The hit point's other coordinate: y on an X-side face, x on a
			// Y-side face. Its fraction is the position along the face.
			along := posY + perp*rayDirY
			if side == 1 {
				perp = sideDistY - deltaY
				along = posX + perp*rayDirX
			}
			return Hit{PerpDist: perp, Side: side, MapX: mapX, MapY: mapY, Cell: m.At(mapX, mapY),
				WallX: along - math.Floor(along)}
		}
	}
	return Hit{PerpDist: math.Inf(1), MapX: mapX, MapY: mapY, Cell: m.At(mapX, mapY)}
}
