# TASKS: termfps P0

One task at a time. Run the verify command from CLAUDE.md before marking done.
Visual acceptance checks (anything you have to look at on screen) are signed off by me, not by an agent.
Status: [ ] todo, [~] in progress, [x] done

## Day 1: setup, loop, map, raycaster

### [x] T1: Project setup + clean screen
- `git init`, `.gitignore` (the built `.exe`), `.gitattributes` with `*.go text eol=lf` (so gofmt doesn't flag every file).
- `go mod init` with the final module path (renaming it later touches every import). Add tcell v2 at a pinned version; write the version in PROGRESS.md.
- Folder layout from ARCHITECTURE.md.
- Open full-screen, show "termfps" centered, quit on Esc or Ctrl+C.
- `restore` with `sync.Once` + panic recovery per ARCHITECTURE "Flow".
- Debug switch: env var `TERMFPS_PANIC=1` makes the game panic about 1 second after start.
- Acceptance:
  - `go build` works on Windows.
  - Esc and Ctrl+C both quit.
  - After quit, PowerShell prompt is normal (cursor visible, no garbage).
  - With `TERMFPS_PANIC=1`, the terminal is restored and the panic message + stack trace are readable in PowerShell.
  - The `Init()` error is checked: on failure it prints the error and exits 1. (Hard to force on Windows; check by reading the code.)

### [x] T2: Game loop at 30 FPS
- Input goroutine + buffered channel, drained with `select`/`default`. Goroutine exits when `PollEvent()` returns nil, and has its own recover.
- 30 FPS ticker, `-fps N` flag, dt in seconds clamped to 0.1.
- HUD shows FPS (frames counted over the last 1 second) and ms per frame.
- Acceptance:
  - FPS reads 28-32 in Windows Terminal. `-fps 15` reads 14-16.
  - Idle CPU under 5% in Task Manager.
  - `-fps 0` prints an error and exits without opening the screen.
  - `TERMFPS_PANIC=input` + any key: terminal restored, panic printed (tests the input goroutine's recover).

### [x] T3: Map loader + tests
- `world.Parse` per ARCHITECTURE map format. Create `maps/arena.txt` (16x16).
- Acceptance:
  - Tests cover: valid map, ragged rows, unknown char, `T` rejected, no P, two P, open border, `\r\n` endings, UTF-8 BOM, trailing newline(s), trailing spaces, empty file, smaller than 3x3.
  - A test parses the real embedded `maps/arena.txt`.
  - All tests pass.

### [x] T4: Raycaster (world.CastRay) + tests
- `world.CastRay` with DDA, float64 math, step cap.
- Acceptance:
  - Unit tests: ray straight at a wall (east, west, north, south) returns the expected PerpDist and Side.
  - Unit test: an axis-aligned ray (one direction component exactly 0) works.
  - Unit test: a diagonal ray toward a corner returns a sane hit.

### [x] T5: Frame + DrawFrame (flat walls) + tests
- `render.Frame` sized `W x 2*(rows-1)`, reused between frames.
- `render.DrawFrame`: walls one flat gray, floor and ceiling black. Uses the wall-height formula with `pixelAspect` from ARCHITECTURE. Clamps drawStart/drawEnd.
- `Player.Angle` with `Dir()` and `Plane()` methods per ARCHITECTURE "Coordinates".
- Acceptance:
  - Unit test: DrawFrame facing a wall at distance 2 fills the expected middle rows with wall gray.
  - Unit test: DrawFrame with the player almost touching a wall (PerpDist near the minimum) doesn't panic and fills the column.
  - Unit test: `Plane()` points to the camera's right for angles 0 and 90°.

## Day 2: see it, move in it

### [x] T6: Half-block presenter + terminal color check
- `render.PresentHalfBlock`: `▀` with fg = top pixel, bg = bottom pixel, 24-bit gray.
- Time DrawFrame, presenter, and `Show()` separately; show them in the HUD.
- Acceptance:
  - On screen, facing a wall shows a solid vertical band with no gaps between rows.
  - In PROGRESS.md: does truecolor look smooth in Windows Terminal (tcell turns it on by default, see CLAUDE.md), what `screen.Colors()` reports, the `builtinGlyphs` setting, and the three timings at 120x40.

### [x] T7: Input tracker + movement + turning
- `input` package: adaptive held window per ARCHITECTURE "Input approach", produces `game.Input` each frame. Tracker takes `now` as a parameter.
- W/S forward/back, A/D strafe, Left/Right arrows (and Q/E) turn. Speeds per second with dt. Rune keys lower-cased.
- Acceptance:
  - Unit test: 360 turn steps of 1° return `Angle` to its start value (within 1e-9).
  - Unit test: tracker reports a key held 300ms after its first press, and not held 150ms after its last repeat.
  - Moving 3 seconds forward at `-fps 15` and at `-fps 30` covers about the same distance (check the HUD position).
  - Right arrow turns right, D strafes right.

### [x] T8: Spike: real key state on Windows (2-hour timebox)
- Try `GetAsyncKeyState` via standard `syscall` in `internal/input/keystate_windows.go`. No new dependencies.
- Note the other option in ARCHITECTURE "Input approach" (Windows Terminal key-up events that tcell drops). Don't build it without asking.
- Acceptance:
  - Can hold W and Left together and move while turning: yes or no.
  - Does it react to keys pressed in another window? Is there a fix?
  - Keep or drop. Log the result as a decision in DECISIONS.md. If dropped, the T7 baseline stays.

### [x] T9: Collision with sliding
- `world.TryMove` with a small radius (about 0.2), checking the full box on each axis.
- Acceptance:
  - Unit tests: blocked by a wall head-on, slides when moving diagonally along a wall, can't enter a wall diagonally at an outside corner.
  - In game: can't walk through any wall, including corners.

## Day 3-4: look and feel

### [~] T10: Resize handling
- Handle resize events. Resize the Frame. Under 80x24, show "resize your terminal" (clipped to the width) and skip rendering.
- Acceptance:
  - Resizing the window mid-game, including down to a few columns and back, doesn't crash or leave garbage.

### [~] T11: Tune pixelAspect
- Adjust `pixelAspect` by eye for the Windows Terminal font. Write the final value in PROGRESS.md.
- Acceptance:
  - Straight walls look straight, not curved.
  - A 1x1 wall block looks roughly square at medium distance (compare with the browser mock).

### [~] T12: Wall shading
- In DrawFrame: brightness falls with distance (fog). Y-side walls about 0.7x brightness. Clamp before converting to uint8.
- Acceptance: corners and depth are clearly readable in half-block mode (compare with the browser mock).

### [~] T13: Floor and ceiling
- Floor dark at the horizon, brighter toward the bottom. Ceiling near black.
- Acceptance: gives a sense of ground without washing out the walls (compare with the browser mock).

### [~] T14: Minimap
- Top-right overlay as text cells drawn after the presenter. Player shown as an arrow for facing direction. Toggle with M (one-shot).
- Acceptance: position and facing match the 3D view, including after turning 90° right (arrow points south).

### [~] T15: HUD
- Bottom line (the row outside the Frame): FPS, X/Y, facing angle in degrees (0 = east, 90 = south).
- Acceptance: never overlaps the minimap, updates every frame, fits at 80 columns.

### [ ] T16: Terminal check
- Windows Terminal (PowerShell), plus one terminal that is not Windows Terminal (e.g. VS Code's terminal).
- Check: FPS, `▀` gaps between rows, gray quality (truecolor or downgraded).
- Optional, 10 minutes max: a quick look in the legacy console (run `conhost.exe` directly). No requirements.
- Acceptance: results written in PROGRESS.md.

### [~] T17: Performance check
- Measure DrawFrame, presenter, and `Show()` at 120x40 in half-block mode while moving.
- Acceptance: within budget in ARCHITECTURE.md, or the bottleneck is written down.

## Day 5: ship

### [ ] T18: Demo recording + README
- README: what it is, how to build and run, controls, the caution "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."
- 30-second screen recording walking the map in half-block mode.
- Acceptance: good enough to show friends and the team.

## Stretch (only if T1-T18 are done, in this order)

### [ ] T19: Render modes (shade + ASCII) + V toggle
- Add `RenderMode` to State. `PresentShade` and `PresentASCII` from the same Frame (average the two pixels per cell).
- V cycles half-block -> shade -> ASCII. HUD shows the current mode.
- If the terminal doesn't have truecolor (per the T6 findings), start in ASCII.
- When switching to half-block, show the caution once in the HUD: "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."
- Acceptance:
  - All three modes show the same scene with the same proportions.
  - Switching modes mid-game doesn't flicker or leave garbage.
  - DrawFrame code is unchanged by this task.

### [ ] T20: Targets + shooting
- Parse accepts `T` (solid, like a wall, for collision and rays). Add `Pixel.Color`; `T` cells set `Color = 1` and render red in every mode.
- Space (one-shot) casts the center ray with `world.CastRay`. If it hits a `T`, that cell becomes empty.
- Acceptance: shooting a visible target removes it. Shooting a wall does nothing. Unit test for the shot on a small test map.
