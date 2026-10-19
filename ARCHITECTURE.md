# ARCHITECTURE: termfps (P0)

## Stack
- Language: Go (latest stable)
- Terminal library: tcell v2 (`github.com/gdamore/tcell/v2`), pinned at v2.13.10 in go.mod. Only direct dependency (go.mod also lists tcell's own dependencies as `// indirect`). Don't use a newer major version (its API is different).
- Build: `go build` -> single binary
- Target: Windows Terminal + PowerShell first. Linux/Mac should work since tcell is cross-platform.

## Folder layout
```
termfps/
  go.mod
  main.go              entry point, embeds maps, wires everything, restores terminal, recovers panics
  maps/
    arena.txt          16x16 map
  internal/
    world/             map parsing, cell lookup, collision, raycast (no tcell)
    game/              player, game state, update logic (no tcell)
    input/             tcell key events -> game.Input (held-key tracking)
    render/            frame buffer, DrawFrame, presenters, minimap, HUD
```
Note: `//go:embed` only sees files in the same package dir or below. We keep the embed in `main.go` and pass the bytes to `world.Parse`.

## Modules

### world
- `Parse(data []byte) (*Map, error)`
- `(*Map) At(x, y int) Cell`
- `(*Map) IsWall(x, y int) bool` (out of bounds counts as wall)
- `TryMove(m *Map, x, y, dx, dy, radius float64) (newX, newY float64)`: moves on X and Y separately so the player slides along walls. Each axis checks the player's full box (`x±radius`, `y±radius`), not just the center, so corners can't be clipped.
- `CastRay(m *Map, posX, posY, rayDirX, rayDirY float64) Hit`: pure function, DDA algorithm, testable without a terminal. Lives in world (not render) because it only needs the map, and game needs it later for shooting. If it were in render, game -> render -> game would be an import cycle, which Go refuses to compile.

### game
- `Player`, `State`, `Input`
- `Update(state *State, in Input, dt float64)`
- Movement speed and turn speed are per second, multiplied by `dt`. Never per frame, never per key event.
- `dt` is clamped to at most 0.1s, so a stall (e.g. window drag) can't move the player through a wall in one step.

### input
- Reads tcell events (from a goroutine, sent over a channel).
- Tracks which keys are held (see Input approach) and produces one `game.Input` per frame. The game package never sees tcell types.
- The held-key tracker takes `now time.Time` as a parameter, so it can be unit-tested without a terminal or real waiting.
- `input.Pump` is the event goroutine's loop; `main.go` starts it and handles its panics. `KeyFor` maps keys: W/Up forward, S/Down back, A/D strafe, Q/Left and E/Right turn.

### render
Two steps: draw the world into a pixel buffer, then present the buffer as terminal cells.
- `DrawFrame(state *game.State, f *Frame)`: casts one ray per column with `world.CastRay` and fills the Frame (walls, floor, ceiling). No tcell. Testable.
- Presenters write a Frame into the tcell screen:
  - `PresentHalfBlock(s tcell.Screen, f *Frame)` (P0 core)
  - `PresentShade`, `PresentASCII` (stretch, T19)
- `DrawMinimap`, `DrawHUD` write text cells on top, after the presenter.

## Data models
```go
// world
type Cell byte // '#' wall, '.' empty, 'P' spawn
type Map struct {
    W, H           int
    Cells          []Cell // row-major, index = y*W + x
    SpawnX, SpawnY float64 // center of the P cell (x+0.5, y+0.5)
}
type Hit struct {
    PerpDist float64 // perpendicular distance (prevents fisheye)
    Side     int     // 0 = hit an X-side wall, 1 = Y-side wall
    MapX     int
    MapY     int
    Cell     Cell
}

// game
type Player struct {
    X, Y  float64 // position in map units
    Angle float64 // radians, kept in [0, 2*pi). See Coordinates.
}
// Dir() and Plane() are methods computed from Angle every frame:
//   Dir   = (cos(Angle), sin(Angle))
//   Plane = (-sin(Angle), cos(Angle)) * planeLen   // points to the camera's right
type State struct {
    Map         *world.Map
    Player      Player
    ShowMinimap bool
    Quit        bool
}
type Input struct {
    Forward, Back, StrafeLeft, StrafeRight, TurnLeft, TurnRight bool // held this frame
    ToggleMinimap bool // pressed this frame (one-shot)
    Quit          bool // pressed this frame (one-shot)
}

// render
type Pixel struct {
    Gray uint8 // brightness 0-255. T20 (stretch) adds a Color field.
}
type Frame struct {
    W, H int     // W = screen columns, H = 2 * (screen rows - 1). The last row is the HUD.
    Pix  []Pixel // row-major, index = y*W + x
}
```
The Frame is reused every frame. Resize it only when the terminal size changes, and reuse the slice's capacity when it is big enough. No per-frame allocation.

Stretch-only fields and types (`RenderMode`, `Pixel.Color`, a shoot action) are added by their stretch tasks, not in P0.

## Coordinates
- Map x grows to the right (east), map y grows down (south), same as the text file.
- `Angle = 0` faces east. Because y points down, a growing angle turns clockwise on screen: 90° faces south.
- Right arrow / E turns right = angle increases. Left arrow / Q = angle decreases.
- Strafe right moves along `+Plane`, strafe left along `-Plane`.
- Spawn facing: east (`Angle = 0`). The map format has no facing.
- The HUD shows `Angle` in degrees (0-359).

## Map format (`maps/*.txt`)
- Rectangular grid, one row per line.
- `#` wall, `.` empty, `P` player spawn (exactly one).
- Border must be fully walls.
- Before checking: strip a UTF-8 BOM (Notepad adds `EF BB BF`), handle `\n` and `\r\n` line endings, strip trailing spaces/tabs on each line, drop empty lines at the end of the file.
- Parse must reject (with a clear error naming the row and column): empty file, smaller than 3x3, ragged rows, unknown chars (including `T` in P0, which T20 adds), zero or multiple `P`, open border.

## Flow (game loop)
1. `main` creates the tcell screen. If `Init()` fails, print the error and exit 1.
2. Restoring the terminal is one `restore` function wrapped in `sync.Once` (so it runs only once even if called from two places). `main` defers a function that calls `recover()`, then `restore`, then prints the panic and stack trace, then exits 1. Order matters: printing before `restore` writes into the alternate screen and the message is lost.
   - Never call `os.Exit` or `log.Fatal` after `Init()` except after `restore`. They skip deferred functions.
3. Input goroutine: `screen.PollEvent()` in a loop -> buffered channel.
   - `recover()` in `main` only catches panics on the main goroutine. The input goroutine has its own deferred recover, which sends the panic and its stack to the main loop over a channel (buffer 1). The main loop restores, prints, and exits 1. Only the main goroutine ever touches the screen, so a crash can't race with drawing.
   - `PollEvent()` returns nil after the screen is finalized. The goroutine exits on nil.
   - The goroutine only sends events. It never reads or writes `State`.
4. Main loop on a 30 FPS ticker (`-fps N` flag overrides 30, for testing):
   - drain the input channel without blocking (`select` with a `default` case)
   - on a resize event: `screen.Sync()`, then read `screen.Size()` again and resize the Frame
   - if the screen is below the minimum size: draw the "resize your terminal" message (clipped to the screen width), `Show()`, skip the rest of this frame
   - build `game.Input` from the held-key tracker, then `game.Update(state, in, dt)` where `dt` is real elapsed seconds, clamped to 0.1
   - `render.DrawFrame(state, frame)`
   - `PresentHalfBlock`
   - minimap and HUD on top
   - `screen.Show()` (tcell only sends cells that changed)
5. Quit on Esc or Ctrl+C. tcell delivers Ctrl+C as a key event (`tcell.KeyCtrlC`), so it must be handled explicitly.

## Rendering math
All math happens in Frame pixel space. Frame is `W` pixels wide and `H` pixels tall.
For each column `x`:
- `cameraX = 2*x/W - 1` (float math; DrawFrame is never called with W or H of 0, see Flow)
- `rayDir = Dir + Plane * cameraX`
- `world.CastRay` with DDA, get `PerpDist` (clamp to a minimum like 0.0001)
  - Keep DDA math in float64. A zero ray component gives `+Inf` delta distance, which is fine in float math (integer division by zero would panic).
  - Cap the DDA loop at `m.W + m.H` steps as a safety net.
- A terminal cell is about 2x taller than wide. Each cell holds 2 Frame pixels stacked, so Frame pixels are close to square:
  `lineHeight = W / (2 * planeLen * pixelAspect * PerpDist)`
  with `planeLen = 0.66`, `pixelAspect = 1.0` (constant, tune by eye per font in T11)
- Wall drawn centered vertically. `drawStart` and `drawEnd` are clamped to `[0, H-1]` before the draw loop. Up close, lineHeight can be huge.
- Above = ceiling, below = floor.
- Because aspect is handled once in pixel space, all presenters get the same proportions.

## Shading (writes Pixel.Gray)
- Walls: brightness falls with distance (fog), for example `gray = base / (1 + k*PerpDist)`. Y-side walls multiplied by about 0.7 so corners read clearly.
- Floor: dark at the horizon, brighter toward the bottom of the screen.
- Ceiling: near black.
- Compute in float64, clamp to 0-255, then convert to `uint8`. Go does not panic on an out-of-range float -> uint8 conversion; it gives a garbage value.

## Presenters
- Half-block: for each cell (x, row), top = Pix[(2*row)*W + x], bottom = Pix[(2*row+1)*W + x]. Draw `▀` with foreground = top gray, background = bottom gray, using `tcell.NewRGBColor`.
- Shade (stretch): average the two pixels, map to `█▓▒░ ` (bright to dark).
- ASCII (stretch): average the two pixels, map to `@ % # * + = - : . ` (bright to dark).
- Truecolor: half-block assumes 24-bit color. Checked in the tcell v2.13.10 source: on Windows, `tcell.NewScreen()` uses the VT screen with a built-in `xterm-truecolor` definition, so truecolor is always on (`COLORTERM` and `WT_SESSION` are not consulted). `TCELL_TRUECOLOR=disable` turns it off. The legacy console driver is only a fallback if the VT screen can't start. T6 still confirms on screen.
- Without truecolor, tcell downgrades gray to the nearest palette color. At 16 colors that is about 4 grays, so fog becomes hard bands. P0 accepts this outside Windows Terminal. Stretch T19 starts in ASCII when truecolor is not available.

## Input approach
- tcell only gives key press and repeat events. It drops key-up events (the Win32 console API has them; tcell doesn't pass them through).
- Windows key repeat waits about 500ms after the first press, then repeats about 30 times per second. It only repeats the most recently pressed key.
- P0 baseline: adaptive "held window". A key counts as held until a timeout after its last event:
  - after the first press, the timeout is about 550ms (covers the initial repeat delay)
  - once repeats are arriving, the timeout drops to about 100ms (stops soon after release)
  - Update moves `speed * dt` per frame for every held key. Movement is smooth and FPS independent.
- Known limit: holding W and then pressing Left stops W's repeats, so W times out. You can't move and turn at the same time with the baseline.
- Spike T8 (2-hour timebox): read real key state with Win32 `GetAsyncKeyState` through Go's standard `syscall` package (no new dependency), in a Windows-only file (`keystate_windows.go`; Go builds `_windows.go` files only on Windows). Catch: it reads keys even when another window has focus. Keep the spike only if it works and the focus problem is solvable, and log the result as a decision.
- Second option for T8 (found in tcell v2.13.10 source): tcell turns on Windows Terminal's win32-input-mode, which does send key-up events, and tcell throws them away (`handleWinKey` in tcell's `input.go`). Getting them would mean patching or forking tcell, or parsing that input ourselves. That's a dependency change, so ask before going this way.
- Toggle and quit keys are one-shot: they fire on the press event, not while held.
- Rune keys (W/A/S/D/Q/E/M) are compared lower-cased, so Shift or Caps Lock doesn't break them.

## Boundaries
- `world` and `game` must not import tcell. This keeps the game logic reusable for the future multiplayer server.
- `render` only reads state. It never changes it.
- `DrawFrame` must not import or call tcell. Only presenters, minimap, and HUD touch the screen.
- The input goroutine only sends events over its channel. All state changes happen on the main goroutine.
- No global mutable state.
- No new dependencies beyond tcell without asking.

## Performance budget
- 30 FPS target, frame time under 33ms at 120x40.
- Measure three numbers separately: `DrawFrame`, presenter, and `screen.Show()`. The presenter only fills tcell's in-memory buffer. `Show()` is where bytes go to the terminal and is likely the real cost (about 4,800 cells with two RGB colors each, while moving).
- First measurement in T6, full check in T17. The HUD can show ms per frame (T2).
- Minimum terminal size 80x24. Smaller shows a "resize your terminal" message and skips rendering.

## Open questions
- Does the `GetAsyncKeyState` spike (T8) give move-and-turn without focus problems?
- Does Windows Terminal render `▀` without gaps between rows? Recent versions draw block characters themselves (`builtinGlyphs` setting). Check in T6.
- Final `pixelAspect` value for the Windows Terminal font (T11).
