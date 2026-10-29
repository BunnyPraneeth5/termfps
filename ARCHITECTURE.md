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
    01-arena.txt       16x16 (plus 02-warehouse 32x32, 03-yard 64x32, 04-tunnel 16x64; P1-12)
  internal/
    world/             map parsing, cell lookup, collision, raycast (no tcell)
    game/              player, game state, update logic (no tcell)
    input/             tcell key events -> game.Input (held-key tracking)
    render/            frame buffer, DrawFrame, presenters, minimap, HUD
    app/ ui/ save/ audio/ assets/   P1 packages, see "P1 design" -> Packages
```
Note: `//go:embed` only sees files in the same package dir or below. We keep the embed in `main.go` (`maps/*.txt` into an `embed.FS`) and pass each file to `world.Parse`.

## Modules

### world
- `Parse(data []byte) (*Map, error)`
- `(*Map) At(x, y int) Cell`
- `(*Map) IsSolid(x, y int) bool` (walls and `T` blocks; out of bounds counts as solid)
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
    Gray uint8 // brightness 0-255
    Tint Tint  // TintNone (gray), TintRed, TintGreen, TintYellow (P1-02)
}
type Frame struct {
    W, H int     // W = screen columns, H = 2 * (screen rows - 1). The last row is the HUD.
    Pix  []Pixel // row-major, index = y*W + x
}
```
The Frame is reused every frame. Resize it only when the terminal size changes, and reuse the slice's capacity when it is big enough. No per-frame allocation.

Fields beyond P0 (`RenderMode` in T19, `Pixel.Tint` in P1-02, shooting in P1-03) were added by the task that needed them.

## Coordinates
- Map x grows to the right (east), map y grows down (south), same as the text file.
- `Angle = 0` faces east. Because y points down, a growing angle turns clockwise on screen: 90° faces south.
- Right arrow / E turns right = angle increases. Left arrow / Q = angle decreases.
- Strafe right moves along `+Plane`, strafe left along `-Plane`.
- Spawn facing: east (`Angle = 0`). The map format has no facing.
- The HUD shows `Angle` in degrees (0-359).

## Map format (`maps/*.txt`)
- Optional header lines first: `; name: The Arena` (P1-12). Every embedded map needs one, for the map menu. Any other line starting with `;` is an error. Error rows count header lines, so they match the file.
- Rectangular grid, one row per line.
- `#` wall, `.` empty, `P` player spawn (exactly one).
- Border must be fully walls.
- Before checking: strip a UTF-8 BOM (Notepad adds `EF BB BF`), handle `\n` and `\r\n` line endings, strip trailing spaces/tabs on each line, drop empty lines at the end of the file.
- Parse must reject (with a clear error naming the row and column): empty file, smaller than 3x3, ragged rows, unknown chars, zero or multiple `P`, open border (the border must be `#`, so a `T` block can't be on it). `T` is a solid red target block (P1-02). `Z` and `S` mark zombie and shooter starts (P1-06, P1-07); after parsing the cell is floor and the position is in `Map.EnemyStarts`. Walls come in three textures (P1-10): `#` brick, `%` stone, `&` metal; any of them may form the border.
- Props (M2) are drawn as sprites and listed in `Map.Props`; their cell keeps the character. `b` barrel, `c` crate, `i` pillar: block movement and shots (cover). `l` lamp, `L` flickering lamp: block movement, not shots. `,` debris, `x` body: block nothing. Rays (walls) see past every prop. `Map.Blocked` (walls, targets, blocking props) is what collision and the enemies' distance field use; `world.ShotDistance` finds the first prop that stops a shot, and `LineOfSight` includes it.
- Doors (M3): `D`, between walls on two opposite sides (else a parse error), listed in `Map.Doors` (`SlidingDoor`: cell, `Across` = walls left and right, `Open` 0-1, `Hold`). The panel runs through the middle of the cell and slides toward its high end: `CastRay` hits it on the mid-line where `along >= Open` (WallX = along - Open, so the texture moves with it), and goes through the gap otherwise. `Map.Blocked` blocks a door until it's `DoorPassable` (0.8) open; the enemies' distance field always goes through doors. `game.updateDoors` opens a door while the player or an enemy is within 1.5 cells, keeps it open 2.5 s after, slides at 2 per second, and never closes on someone inside. Spawns and boxes skip door cells.
- Lighting (M4): headers `; fog: 0.4` (0-2, the fog factor; default 0.25) and `; darkness: 0.6` (0-0.95; the base light is 1 - darkness). Each lamp (`l`, `L`) lists at parse time the open cells within 4.5 it can see (walls, closed doors and cover props cast shadows), each with weight `0.9 * (1 - d/4.5)^2`. Every frame `render.lightGrid` sets each cell to the base light plus the lamps' weights (times the flicker for `L`: a fixed function of time, 0.6-1 with short drops to 0.25), capped at 1.8. Walls take the light of the cell in front of them, the cast floor blends between cell centers, sprites take their cell's light, the flat floor (textures off) only the base light, and the ceiling stays black. Without darkness or lamps the grid is off and nothing changes.
- Fog of war (M8): `game.State.Seen` (one bool per cell, per run). Each Update, `markSeen` marks the player's 3x3 cells and casts 48 rays across the view with `world.VisitCells`, marking every cell up to and including the first wall or closed door (props don't hide what's behind them). `render.DrawMinimap(s, st, fog)` blanks unseen cells and the boxes on them; Settings "Minimap fog of war" (on by default).
- Custom maps (M9): main loads every `.txt` in the `maps` folder next to save.json (name order) and the `-map` file after the built-in maps. `Config.BuiltinMaps` says how many are built in: only those unlock in order; the player's own are always open and marked "(your map)" in Select map. A map without a name header is named after its file; a taken name gets the file name added. A broken file in the folder is skipped and named in the menus; a broken `-map` file is a startup error (exit 2). `Config.StartMap` selects the `-map` map.

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
- With textures on (V1, D44): the floor is cast. Pixel row y at `offset` rows below the middle sees the floor at perpendicular distance `W / (4 * PlaneLen * aspect * offset)`, the distance whose wall would end on that row, so floor and walls meet without a seam. The world point is player + ray * distance; its position inside its cell picks the texel of a 16x16 tile texture. Brightness is `140 / (1 + k*distance)` like walls (V7; was 120). Texture pixels scale the shade from 30% (darkest, V7; was 45%) to 100%. The ceiling stays flat near-black either way (D45). With textures off, the flat floor gradient above.
- Wall edges (V2): each column's wall covers rows `top` to `top + height` (floats). A pixel row the wall only partly covers mixes the wall pixel with the floor or ceiling pixel by the covered fraction (tint from whichever covers more), so wall tops and bottoms don't step.
- Mipmaps (V3): every texture also has half-size, quarter-size, ... versions down to 1x1 (2x2 averages; 32x32 wall textures since V4 have 6 levels, 16x16 floor and ceiling 5). Each sample picks the level where one screen pixel covers about one texture pixel: walls from texture height / wall height and, across columns, how far apart neighbor columns hit the same face; floor and ceiling from the spacing of neighbor points (2*PlaneLen*d/W across, d*d/K down). Far surfaces become their average instead of flickering.
- Compute in float64, clamp to 0-255, then convert to `uint8`. Go does not panic on an out-of-range float -> uint8 conversion; it gives a garbage value.

## Presenters
- Half-block: for each cell (x, row), top = Pix[(2*row)*W + x], bottom = Pix[(2*row+1)*W + x]. Draw `▀` with foreground = top gray, background = bottom gray, using `tcell.NewRGBColor`.
- Shade (stretch, T19): average the two pixels, map to `█▓▒░ ` (bright to dark), default terminal colors.
- ASCII (stretch, T19): average the two pixels, map to the 16-step `asciiRamp` (V5; `@` brightest down to blank), default terminal colors.
- Mixed (P1-15, D36): the ASCII character for the averaged gray, foreground = the cell color, background = that color at `Frame.MixedBackground` (default 20%, Settings 0-100%, S3). Needs truecolor.
- Color threshold (S3, D43): half-block and mixed reuse the style of the cell to the left when every RGB channel of both colors is within `Frame.ColorThreshold` of it (compared with the style actually used, so drift stays within the threshold; each row starts fresh; tints never merge into gray). tcell then sends no color code for that cell. Off by default; Settings: off, 4, 8, 16, 32.
- Render scale (V6, D46): `Frame.Scale` cells of the screen per Frame cell, across and down. The Frame is sized with `ScaledFrameSize`, and the presenters write each Frame cell to its Scale x Scale block (clipped to the view rows). Settings: auto (`AutoScale`: 1 up to 160 columns, then 2, 3, ...), full (1), half (2). HUD, minimap and menus are drawn on the screen directly, so they stay sharp. Measured saving is small (4-12% at 320x90), since the cost in big windows is the cell count.
- The text ramps scale brightness 0-230 (`wallBase`, the brightest gray DrawFrame makes) onto the ramp, so dim far walls keep a visible character instead of becoming blank.
- `game.State.Mode` (`game.RenderMode`) picks the presenter; V cycles it via the one-shot `Input.CycleRenderMode`. The HUD shows the mode. The first time V switches back to half-block, the caution shows for 5 seconds on the row(s) above the HUD (wrapped by sentence below 84 columns).
- Truecolor: half-block assumes 24-bit color. Checked in the tcell v2.13.10 source: on Windows, `tcell.NewScreen()` uses the VT screen with a built-in `xterm-truecolor` definition, so truecolor is always on (`COLORTERM` and `WT_SESSION` are not consulted). `TCELL_TRUECOLOR=disable` turns it off. The legacy console driver is only a fallback if the VT screen can't start. T6 still confirms on screen.
- Without truecolor, tcell downgrades gray to the nearest palette color. At 16 colors that is about 4 grays, so fog becomes hard bands. P0 accepts this outside Windows Terminal. Stretch T19 starts in ASCII when truecolor is not available.

## Input approach
- (P0, tcell v2) tcell only gives key press and repeat events. It drops key-up events (the Win32 console API has them; tcell doesn't pass them through).
- Windows key repeat waits about 500ms after the first press, then repeats about 30 times per second. It only repeats the most recently pressed key.
- P0 baseline: adaptive "held window". A key counts as held until a timeout after its last event:
  - after the first press, the timeout is about 550ms (covers the initial repeat delay)
  - once repeats are arriving, the timeout drops to about 100ms (stops soon after release)
  - Update moves `speed * dt` per frame for every held key. Movement is smooth and FPS independent.
- Known limit: holding W and then pressing Left stops W's repeats, so W times out. You can't move and turn at the same time with the baseline.
- S2 (D42): tcell v3 with advanced keys reports key releases through win32 input mode (Windows Terminal) and the kitty keyboard protocol. After the first real release, the Tracker holds a key exactly from press to release (no timeout, no tap slowdown); losing focus lets go of every key. Until a release arrives (terminals without them) it guesses as below. The HUD shows `keys events (releases)` in exact mode.
- P1-14 (D35): the Tracker measures the OS repeat delay and uses it + 100ms as the first window; while a key has no repeats yet, turning runs at 25% and moving at 40% (`Input.TurnSlow`, `MoveSlow`), so a tap stays small.
- Result (D19): the spike below worked and is the default on Windows (`-keys async`). The held window above is the fallback (`-keys events`) and the default elsewhere.
- Spike T8 (2-hour timebox): read real key state with Win32 `GetAsyncKeyState` through Go's standard `syscall` package (no new dependency), in a Windows-only file (`keystate_windows.go`; Go builds `_windows.go` files only on Windows). Catch: it reads keys even when another window has focus. Keep the spike only if it works and the focus problem is solvable, and log the result as a decision.
- Second option for T8 (found in tcell v2.13.10 source): tcell turns on Windows Terminal's win32-input-mode, which does send key-up events, and tcell throws them away (`handleWinKey` in tcell's `input.go`). Getting them would mean patching or forking tcell, or parsing that input ourselves. That's a dependency change, so ask before going this way.
- Toggle and quit keys are one-shot: they fire on the press event, not while held.
- Rune keys (W/A/S/D/Q/E/M) are compared lower-cased, so Shift or Caps Lock doesn't break them.

## Boundaries
- `world` and `game` must not import tcell. This keeps the game logic reusable for the future server (P2), which AI opponents (P3) and multiplayer (P4) build on.
- `render` only reads state. It never changes it.
- `DrawFrame` must not import or call tcell. Only presenters, minimap, and HUD touch the screen.
- The input goroutine only sends events over its channel. All state changes happen on the main goroutine.
- No global mutable state.
- No new dependencies beyond tcell without asking.

## Performance budget
- 30 FPS target, frame time under 33ms at 120x40.
- Measure three numbers separately: `DrawFrame`, presenter, and `screen.Show()`. The presenter only fills tcell's in-memory buffer. `Show()` is where bytes go to the terminal and is likely the real cost (about 4,800 cells with two RGB colors each, while moving).
- First measurement in T6, full check in T17. The HUD shows the three timings when the line has room.
- Benchmarks at 120x40 (`go test -bench . ./internal/render`, this laptop): DrawFrame with shading ~0.07ms, 0 allocations; PresentHalfBlock ~0.75-1ms, 0 allocations since T17 switched from `SetContent` (one small allocation per cell) to `Put` with a constant string. `Show()` can only be measured in a real terminal (HUD).
- Measured in Windows Terminal (T17, ~133x33 cells while moving): show 0.5-1.1ms, present ~0.6ms, 30 FPS steady. About 2ms of the 33ms frame budget.
- HUD timings are averages over each 1-second FPS window, because Go's clock on Windows moves in ~0.5ms steps (measured: smallest step 0.53ms).
- Minimum terminal size 80x24. Smaller shows a "resize your terminal" message and skips rendering.

## Open questions
- Does the `GetAsyncKeyState` spike (T8) give move-and-turn without focus problems?
- Does Windows Terminal render `▀` without gaps between rows? Recent versions draw block characters themselves (`builtinGlyphs` setting). Check in T6.
- Final `pixelAspect` value for the Windows Terminal font (T11).

## P1 design (planned 2026-10-20, built 2026-10-25)
Spec: PRD "Roadmap" -> P1. Tasks: TASKS.md "P1". The P0 rules above still hold: `world` and `game` never import tcell (and never import audio), movement uses dt, only the main goroutine touches the screen.

### Packages
```
main.go              flags, map loading, screen init, restore + panic handling only
internal/app         game loop, current screen (Title, MapSelect, Settings, Playing, Paused, GameOver), wiring
internal/ui          menu model (items, selection, values; no tcell) + menu drawing (tcell)
internal/world       + map header (`; name:`), wall types (# % &), T blocks, WallX in Hit, grid pathfinding
internal/game        + weapon/ammo, enemies, pickups, spawner + tiers, score, per-frame events
internal/render      + Pixel.Tint, depth buffer, sprites, textures, weapon overlay, crosshair
internal/input       + Space, R, mouse buttons, menu keys
internal/save        save.json load/save (encoding/json, atomic write)
internal/term        our screen interface, styles and events; tcell v3 adapter (the only tcell import, S2) and a fake screen for tests (S1, D41)
internal/audio       oto player + generated effects (P1-17)
internal/assets      pixel-art images (gun, flash, sprites) and textures as text files, embedded; format in render.ParseImage
```

### Screens
`app` owns one current screen. Each frame it drains events, gives keys/clicks to the current screen, and draws it.
- Playing: game.Update, then render (3D, sprites, weapon, minimap, HUD).
- Paused: no Update. The scene is redrawn from the frozen state each frame (so a resize works), dimmed, with the pause menu on top.
- Menus (Title, MapSelect, Settings): `ui` menus over a plain background.
- GameOver: a menu over the dimmed frozen game: run summary + best, Try again / Main menu.
All menus are one `menuScreen` type with their own items and act function (`internal/app/menus.go`). Mouse buttons reach screens as press/release per button.
Async key state is read only on the Playing screen, so a key held while leaving a menu can't leak into the game.

### Game state additions
- `Player`: + HP, weapon (magazine, reserve, cooldown, reload timer).
- `Enemy`: kind (zombie, shooter), position, HP, state (chase, attack, dead), attack cooldown.
- `Pickup`: position, ammo amount.
- `Run`: kills, time, tier, spawner timers. Tier scaling factors live in one table so balancing is one place.
- `Events []Event` per frame (ShotFired, EnemyHit, EnemyKilled, PlayerHurt, Reloaded, PickupTaken, TierUp, EmptyClick). App uses them for sound and HUD effects; game doesn't know who listens.
- Randomness through an injected `*rand.Rand`, so spawner and accuracy tests are repeatable.

### Shooting
Hitscan along the center ray: `world.CastRay` gives the wall distance; the nearest enemy (or `T` block) in front of it, within its radius of the ray, is hit.

### Enemy movement
A grid distance map (BFS) from the player's cell, recomputed only when the player enters a new cell (cheap even on 64x32). Each enemy steps toward a neighbor cell with a smaller distance, moving with `world.TryMove` and dt like the player. Zombies follow it all the way; shooters stop at a preferred range when they have line of sight (raycast from enemy to player).

### Rendering additions
- `Pixel.Tint` (done in P1-02): none (gray), red, green, yellow. Half-block tints the gray with the color in RGB. Shade/ASCII use the terminal's basic red/green/yellow foreground (works without truecolor).
- Depth buffer: DrawFrame stores each column's PerpDist in the Frame (reused slice, no per-frame allocation).
- Sprites (Lodev billboard method): transform the sprite into camera space, project, scale by distance, draw only columns where the sprite is closer than the depth buffer. Sort sprites far to near. Images are small text pixel-art files with a transparent character.
- Textures: Hit gains `WallX` (0-1 position along the wall face). The texture column comes from WallX, rows are stretched over the wall span; the texture gray is multiplied by the existing fog and side shading. Off by setting.
- Weapon overlay and crosshair are drawn into the Frame after sprites, so all render modes show them.

### Maps
`//go:embed maps/*.txt` into an `embed.FS`; every file is parsed at startup (a broken map is an error before the screen opens, like now). Optional first lines starting with `;` are headers (`; name: The Warehouse`). The minimap draws a window (at most 16x16 cells since M1) centered on the player when the map is bigger than the space it has, inside a one-cell border.

### Saving
`os.UserConfigDir()` + `termfps/save.json`. Write to `save.json.tmp`, then rename over `save.json`. On load: missing file = defaults (first run); unparsable or invalid file = rename to `save.json.bad`, use defaults, show a message in the menu; a file that exists but can't be read is left alone, with a message. Flags given on the command line win over saved settings (D34).

### Sound
`internal/audio` wraps oto: one context at startup, effects pre-generated as PCM buffers (sine/square waves + noise with envelopes), each event starts a new player so sounds overlap (max 16). Volume from settings. If the context fails, audio is disabled and the error is shown in the menu, never silently ignored. Audio opens before the screen. Event -> effect mapping in `internal/app/sound.go` (D37).
