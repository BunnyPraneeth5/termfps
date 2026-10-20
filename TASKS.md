# TASKS: termfps

One task at a time. Run the verify command from CLAUDE.md before marking done.
Visual acceptance checks (anything you have to look at on screen) are signed off by me, not by an agent.
Status: [ ] todo, [~] in progress, [x] done, [-] moved

# P0: prototype (done)

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

### [x] T10: Resize handling
- Handle resize events. Resize the Frame. Under 80x24, show "resize your terminal" (clipped to the width) and skip rendering.
- Acceptance:
  - Resizing the window mid-game, including down to a few columns and back, doesn't crash or leave garbage.

### [x] T11: Tune pixelAspect
- Adjust `pixelAspect` by eye for the Windows Terminal font. Write the final value in PROGRESS.md.
- Acceptance:
  - Straight walls look straight, not curved.
  - A 1x1 wall block looks roughly square at medium distance (compare with the browser mock).

### [x] T12: Wall shading
- In DrawFrame: brightness falls with distance (fog). Y-side walls about 0.7x brightness. Clamp before converting to uint8.
- Acceptance: corners and depth are clearly readable in half-block mode (compare with the browser mock).

### [x] T13: Floor and ceiling
- Floor dark at the horizon, brighter toward the bottom. Ceiling near black.
- Acceptance: gives a sense of ground without washing out the walls (compare with the browser mock).

### [x] T14: Minimap
- Top-right overlay as text cells drawn after the presenter. Player shown as an arrow for facing direction. Toggle with M (one-shot).
- Acceptance: position and facing match the 3D view, including after turning 90° right (arrow points south).

### [x] T15: HUD
- Bottom line (the row outside the Frame): FPS, X/Y, facing angle in degrees (0 = east, 90 = south).
- Acceptance: never overlaps the minimap, updates every frame, fits at 80 columns.

### [x] T16: Terminal check
- Windows Terminal (PowerShell), plus one terminal that is not Windows Terminal (e.g. VS Code's terminal).
- Check: FPS, `▀` gaps between rows, gray quality (truecolor or downgraded).
- Optional, 10 minutes max: a quick look in the legacy console (run `conhost.exe` directly). No requirements.
- Acceptance: results written in PROGRESS.md.

### [x] T17: Performance check
- Measure DrawFrame, presenter, and `Show()` at 120x40 in half-block mode while moving.
- Acceptance: within budget in ARCHITECTURE.md, or the bottleneck is written down.

## Day 5: ship

### [x] T18: Demo recording + README
- README: what it is, how to build and run, controls, the caution "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."
- 30-second screen recording walking the map in half-block mode.
- Acceptance: good enough to show friends and the team.

## Stretch (only if T1-T18 are done, in this order)

### [x] T19: Render modes (shade + ASCII) + V toggle
- Add `RenderMode` to State. `PresentShade` and `PresentASCII` from the same Frame (average the two pixels per cell).
- V cycles half-block -> shade -> ASCII. HUD shows the current mode.
- If the terminal doesn't have truecolor (per the T6 findings), start in ASCII.
- When switching to half-block, show the caution once in the HUD: "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."
- Acceptance:
  - All three modes show the same scene with the same proportions.
  - Switching modes mid-game doesn't flicker or leave garbage.
  - DrawFrame code is unchanged by this task.

### [-] T20: Targets + shooting (moved to P1-02 and P1-03, D29)
- Parse accepts `T` (solid, like a wall, for collision and rays). Add `Pixel.Color`; `T` cells set `Color = 1` and render red in every mode.
- Space (one-shot) casts the center ray with `world.CastRay`. If it hits a `T`, that cell becomes empty.
- Acceptance: shooting a visible target removes it. Shooting a wall does nothing. Unit test for the shot on a small test map.

## After P0

T20 moved into P1 as P1-02 (DECISIONS D29). P2-P4 are in PRD.md "Roadmap"; their tasks get written when each phase is planned.

# P1: single-player survival game

Spec: PRD.md "Roadmap" -> P1. Design: ARCHITECTURE.md "P1 design". Decisions: D21-D29.
Same rules as P0: one task at a time, verify command before marking done, one commit per task, visual checks signed off by the user.
Estimated: about 5 weekends.

## Foundations

### [~] P1-01: App structure and screens
- Move the game loop out of `main.go` into a new `internal/app` package. `main.go` keeps only startup (flags, map parse, screen init, restore, panic handling).
- `app` holds the current screen (Playing for now; Title, Paused, Settings, MapSelect, GameOver come later) and switches between them.
- `game.State` gets a per-frame event list (`[]game.Event`: shot fired, enemy hit, pickup taken...) that app reads after Update, so later sound and HUD effects don't need game to know about them.
- Acceptance:
  - The game plays exactly as before (all P0 checks still pass by eye).
  - All existing tests pass; new tests for screen switching.

### [ ] P1-02: Color in the renderer + red target blocks (was T20)
- `Pixel.Color` (gray, red, green, yellow). Half-block uses RGB tints; shade and ASCII use the terminal's basic red/green/yellow, which works without truecolor.
- Map character `T`: a solid red block, shootable later (P1-03).
- Acceptance: a `T` block shows red in all three render modes. Unit tests for parsing `T` and for the color in each presenter.

## Shooting and weapon

### [ ] P1-03: Shooting, crosshair, ammo and reload
- Space fires a hitscan shot along the center ray, about 3 per second while held. Crosshair `+` at the screen center.
- Magazine 25, start with 25 + 50 spare. R reloads (~1.5 s, no shooting meanwhile). Auto-reload when the magazine is empty and there are spares. Firing with 0 ammo does nothing (later: empty click sound).
- A shot that hits a `T` block removes it. HUD: `AMMO 18 / 50`, `RELOADING...`.
- Space and R work in both key modes (async: GetAsyncKeyState; events: one-shot presses).
- Acceptance: unit tests for fire rate, magazine and reserve counts, reload timing, auto-reload, a target hit, and a wall blocking the shot. On screen: crosshair centered, ammo counts right.

### [ ] P1-04: Weapon graphics and muzzle flash
- Pixel-art gun at the bottom center, drawn into the Frame after DrawFrame. Yellow muzzle flash ~0.1 s and a small recoil on each shot. Optional gentle bobbing while walking.
- Acceptance: the gun shows in all three render modes; the flash appears only right after a shot. Tests for the overlay position and flash timing.

## Enemies

### [ ] P1-05: Sprites (billboards hidden by walls)
- DrawFrame records each column's wall distance (a depth buffer in the Frame). A sprite renderer draws small pixel-art images that always face the camera, scaled by distance, skipping columns where a wall is closer.
- Sprite images stored as text (like maps), embedded.
- Acceptance: unit tests: a sprite in front of a wall is drawn, one behind a wall isn't, and size shrinks with distance. On screen: a test sprite looks right and hides behind corners.

### [ ] P1-06: Zombies, player health, death
- Green melee enemy: chases along a path around walls (a grid distance map from the player, recomputed when the player changes cell), hits when close. Takes several shots.
- Player HP 100, shown in the HUD. HP 0 -> Game over screen (kills, time) with Retry or Quit.
- Acceptance: unit tests for pathfinding around a wall, melee range and damage, taking damage from shots, death. On screen: zombies come around corners and kill you if you don't shoot.

### [ ] P1-07: Shooter enemies
- Red ranged enemy: needs line of sight (raycast), keeps some distance, shoots with a cooldown; accuracy drops with distance. Fragile.
- Acceptance: unit tests: no shot without line of sight, cooldown respected, hit chance falls with distance. On screen: shooters hold distance and hurt you when they see you.

### [ ] P1-08: Survival spawner, tiers, score
- Spawns zombies and shooters at random free cells out of the player's sight and at least a few cells away.
- Tier up every 20 kills or 4 minutes, whichever comes first: more enemy HP, damage, speed, spawn rate, max alive. HUD shows tier, kills, time.
- Random source is injectable so tests are repeatable.
- Acceptance: unit tests: never spawns in sight or too close, tier rules (kills and time), scaling applied. On screen: pressure keeps rising.

### [ ] P1-09: Ammo pickups
- Spawn at random free cells every ~20-40 s, at most 3 alive. Walking over one gives +25 spare (up to a cap). Shown as a sprite and a minimap dot.
- Acceptance: unit tests: spawn timing and limit, pickup radius, cap. On screen: pickups appear and refill ammo.

## World, menus, maps, saving

### [ ] P1-10: Wall textures
- `CastRay` also returns where on the wall the ray hit. Wall characters `#` brick, `%` stone, `&` metal map to 16x16 grayscale text textures (embedded). Fog and side shading still apply. Textures on/off flag (setting in P1-11).
- Acceptance: unit tests for the hit position and texture sampling. On screen: textures look right and line up at corners; turning them off gives the old flat look.

### [ ] P1-11: Menus: start, pause, settings (keyboard + mouse clicks)
- New `internal/ui` menu model (items, selection, values), tested without a terminal, plus drawing.
- Start: Play, Select map, Settings, Quit. Esc during play -> Pause: Resume, Settings, Restart, Quit to menu (game frozen and dimmed behind). Settings: render mode, FPS 30/60, minimap at start, textures, sound on/off + volume (applies from P1-15), key mode, pixel aspect.
- Keyboard: Up/Down or W/S, Enter, Left/Right, Esc back. Mouse: click an item to select it. Ctrl+C quits from anywhere.
- Acceptance: unit tests for menu navigation and value changes. On screen: all menus work by keyboard and by clicking; settings take effect right away.

### [ ] P1-12: Four maps, map select, unlocks, scrolling minimap
- Maps 16x16, 32x32, 64x32, 16x64 in `maps/`, all embedded (directory embed). Optional `; name:` header line.
- Map select menu; the first map is open, each next one unlocks after surviving 5 minutes on the previous (kept in memory until P1-13).
- Minimap shows a window around the player when the map doesn't fit.
- Acceptance: tests: every embedded map parses, every open cell is reachable, header parsing, unlock rule, minimap window. On screen: all four maps play; the minimap scrolls on big ones.

### [ ] P1-13: Saving settings and progress
- `%AppData%\termfps\save.json` (`os.UserConfigDir`): settings, unlocked maps, best kills / time / tier per map, total kills. Saved automatically when a setting changes or a run ends.
- Atomic write (temp file + rename). A damaged file -> message in the menu, the file kept as `save.json.bad`, defaults used.
- Acceptance: tests (temp dir): save and load round trip, damaged file handling, atomic write. On screen: settings and bests survive a restart.

## Extras

### [ ] P1-14: Mouse in game
- Left-click shoots, right-click reloads. No mouse-look.
- Acceptance: unit tests for the mouse-button mapping. On screen: clicking shoots and reloads.

### [ ] P1-15: Sound (oto)
- Add `github.com/ebitengine/oto/v3` (approved, D27) behind a small `internal/audio` package. Effects generated in code: gunshot, reload, empty click, zombie groan, enemy shot, hurt, pickup, tier up. Triggered by game events (P1-01). Settings: sound on/off and volume.
- If audio can't start, the game still runs and the menu shows "sound unavailable: <reason>".
- Acceptance: tests for the generated waveforms (length, no clipping) and the event -> sound mapping. On screen and ears: sounds play and overlap; volume and mute work.

### [ ] P1-16: Balancing, README, demo
- Tune tiers, damage, HP, spawn rates, ammo cap; decide health pickups or regen. Update the README (controls, menus, survival rules) and record a new demo.
- Acceptance: the user plays each map for a few minutes and is happy with the difficulty curve.
