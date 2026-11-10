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

Spec: PRD.md "Roadmap" -> P1. Design: ARCHITECTURE.md "P1 design". Decisions: D21-D31.
Same rules as P0: one task at a time, verify command before marking done, one commit per task, visual checks signed off by the user.
Estimated: about 5 weekends.

## Foundations

### [x] P1-01: App structure and screens
- Move the game loop out of `main.go` into a new `internal/app` package. `main.go` keeps only startup (flags, map parse, screen init, restore, panic handling).
- `app` holds the current screen (Playing for now; Title, Paused, Settings, MapSelect, GameOver come later) and switches between them.
- `game.State` gets a per-frame event list (`[]game.Event`: shot fired, enemy hit, pickup taken...) that app reads after Update, so later sound and HUD effects don't need game to know about them.
- Acceptance:
  - The game plays exactly as before (all P0 checks still pass by eye).
  - All existing tests pass; new tests for screen switching.

### [x] P1-02: Color in the renderer + red target blocks (was T20)
- `Pixel.Color` (gray, red, green, yellow). Half-block uses RGB tints; shade and ASCII use the terminal's basic red/green/yellow, which works without truecolor.
- Map character `T`: a solid red block, shootable later (P1-03).
- Acceptance: a `T` block shows red in all three render modes. Unit tests for parsing `T` and for the color in each presenter.

## Shooting and weapon

### [x] P1-03: Shooting, crosshair, ammo and reload
- Space fires a hitscan shot along the center ray, about 3 per second while held. Crosshair `+` at the screen center.
- Magazine 25, start with 25 + 50 spare. R reloads (~1.5 s, no shooting meanwhile). Auto-reload when the magazine is empty and there are spares. Firing with 0 ammo does nothing (later: empty click sound).
- A shot that hits a `T` block removes it. HUD: `AMMO 18 / 50`, `RELOADING...`.
- Space and R work in both key modes (async: GetAsyncKeyState; events: one-shot presses).
- Acceptance: unit tests for fire rate, magazine and reserve counts, reload timing, auto-reload, a target hit, and a wall blocking the shot. On screen: crosshair centered, ammo counts right.

### [x] P1-04: Weapon graphics and muzzle flash
- Pixel-art gun at the bottom center, drawn into the Frame after DrawFrame. Yellow muzzle flash ~0.1 s and a small recoil on each shot. Optional gentle bobbing while walking.
- Acceptance: the gun shows in all three render modes; the flash appears only right after a shot. Tests for the overlay position and flash timing.

## Enemies

### [x] P1-05: Sprites (billboards hidden by walls)
- DrawFrame records each column's wall distance (a depth buffer in the Frame). A sprite renderer draws small pixel-art images that always face the camera, scaled by distance, skipping columns where a wall is closer.
- Sprite images stored as text (like maps), embedded.
- Acceptance: unit tests: a sprite in front of a wall is drawn, one behind a wall isn't, and size shrinks with distance. On screen: a test sprite looks right and hides behind corners.

### [x] P1-06: Zombies, player health, death
- Green melee enemy: chases along a path around walls (a grid distance map from the player, recomputed when the player changes cell), hits when close. Takes several shots.
- Player HP 100, shown in the HUD. HP 0 -> Game over screen (kills, time) with Retry or Quit.
- Acceptance: unit tests for pathfinding around a wall, melee range and damage, taking damage from shots, death. On screen: zombies come around corners and kill you if you don't shoot.

### [x] P1-07: Shooter enemies
- Red ranged enemy: needs line of sight (raycast), keeps some distance, shoots with a cooldown; accuracy drops with distance. Fragile.
- Acceptance: unit tests: no shot without line of sight, cooldown respected, hit chance falls with distance. On screen: shooters hold distance and hurt you when they see you.

### [x] P1-08: Survival spawner, tiers, score
- Spawns zombies and shooters at random free cells out of the player's sight and at least a few cells away.
- Tier up every 20 kills or 4 minutes, whichever comes first: more enemy HP, damage, speed, spawn rate, max alive. HUD shows tier, kills, time.
- Random source is injectable so tests are repeatable.
- Acceptance: unit tests: never spawns in sight or too close, tier rules (kills and time), scaling applied. On screen: pressure keeps rising.

### [x] P1-09: Ammo pickups
- Spawn at random free cells every ~20-40 s, at most 3 alive. Walking over one gives +25 spare (up to a cap). Shown as a sprite and a minimap dot.
- Acceptance: unit tests: spawn timing and limit, pickup radius, cap. On screen: pickups appear and refill ammo.

## World, menus, maps, saving

### [x] P1-10: Wall textures
- `CastRay` also returns where on the wall the ray hit. Wall characters `#` brick, `%` stone, `&` metal map to 16x16 grayscale text textures (embedded). Fog and side shading still apply. Textures on/off flag (setting in P1-11).
- Acceptance: unit tests for the hit position and texture sampling. On screen: textures look right and line up at corners; turning them off gives the old flat look.

### [x] P1-11: Menus: start, pause, settings (keyboard + mouse clicks)
- New `internal/ui` menu model (items, selection, values), tested without a terminal, plus drawing.
- Start: Play, Select map, Settings, Quit. Esc during play -> Pause: Resume, Settings, Restart, Quit to menu (game frozen and dimmed behind). Settings: render mode, FPS 30/60, minimap at start, textures, sound on/off + volume (applies from P1-17), key mode, pixel aspect.
- Keyboard: Up/Down or W/S, Enter, Left/Right, Esc back. Mouse: click an item to select it. Ctrl+C quits from anywhere.
- Acceptance: unit tests for menu navigation and value changes. On screen: all menus work by keyboard and by clicking; settings take effect right away.

### [x] P1-12: Four maps, map select, unlocks, scrolling minimap
- Maps 16x16, 32x32, 64x32, 16x64 in `maps/`, all embedded (directory embed). Optional `; name:` header line.
- Map select menu; the first map is open, each next one unlocks after surviving 5 minutes on the previous (kept in memory until P1-13).
- Minimap shows a window around the player when the map doesn't fit.
- Acceptance: tests: every embedded map parses, every open cell is reachable, header parsing, unlock rule, minimap window. On screen: all four maps play; the minimap scrolls on big ones.

### [x] P1-13: Saving settings and progress
- `%AppData%\termfps\save.json` (`os.UserConfigDir`): settings, unlocked maps, best kills / time / tier per map, total kills. Saved automatically when a setting changes or a run ends.
- Atomic write (temp file + rename). A damaged file -> message in the menu, the file kept as `save.json.bad`, defaults used.
- Acceptance: tests (temp dir): save and load round trip, damaged file handling, atomic write. On screen: settings and bests survive a restart.

## Feedback (S0ra, D31)

Ideas and reference from S0ra (YouTube + Discord, 2026-10-23): events-mode input on Linux and a "Mixed" ASCII render mode. Key release events (needs a tcell change) are left for when Linux/mac support starts.

### [x] P1-14: Smoother events-mode keys
- Tracker learns the OS repeat delay from the first press -> first repeat gap and sets FirstWindow to it plus a margin, so holds don't stop just before repeats begin on Linux (600-660ms delays).
- Turning eases in over the first ~0.4s of a hold, so a single tap turns ~30-50° instead of ~99°. Movement too, if it feels better.
- README: in events mode only the last pressed key repeats, so moving + turning together is limited; async (Windows) doesn't have this.
- Acceptance: unit tests for the learned window and the ease-in (fake times). On screen (user, `-keys events`): a tap turns a small amount, holding is smooth.

### [x] P1-15: "Mixed" render mode
- New RenderMode after ASCII in the V cycle: per cell, ASCII ramp character from the averaged gray, fg = pixel color, bg = pixel color x 0.2. Needs truecolor (same fallback as half-block).
- Optional: round colors to steps so neighbouring cells share a style and tcell sends fewer color codes; measure bytes per frame with and without, keep only if it helps.
- Credit S0ra / libasciic in README.
- Acceptance: unit test for the presenter (character and both colors for known pixels). On screen (user): mode looks right, FPS holds at 30/60 in Windows Terminal and conhost.

## Extras

### [x] P1-16: Mouse in game
- Left-click shoots, right-click reloads. No mouse-look.
- Acceptance: unit tests for the mouse-button mapping. On screen: clicking shoots and reloads.

### [x] P1-17: Sound (oto)
- Add `github.com/ebitengine/oto/v3` (approved, D27) behind a small `internal/audio` package. Effects generated in code: gunshot, reload, empty click, zombie groan, enemy shot, hurt, pickup, tier up. Triggered by game events (P1-01). Settings: sound on/off and volume.
- If audio can't start, the game still runs and the menu shows "sound unavailable: <reason>".
- Acceptance: tests for the generated waveforms (length, no clipping) and the event -> sound mapping. On screen and ears: sounds play and overlap; volume and mute work.

### [~] P1-18: Docs update (README, maps, gameplay)
- Changed 2026-10-25 (D38): docs only. Bring README, ARCHITECTURE and PRD up to date with what P1 built: menus, pause, settings, saving, the four maps and unlocks, mouse, sound, render modes, survival rules, events-mode keys.
- Balancing (playtest tiers 3+, ammo cap, health pickups or not) and a new demo recording move to "Maps and gameplay update" below, to discuss there. Early balancing is in D30.
- Acceptance: the user reads the README and it matches the game.

# S0ra follow-ups

A separate track from maps and gameplay (D39): finishing what S0ra suggested (P1-14 and P1-15 stay [~] until then). Tasks S1 to Sn; the outline below is from the 2026-10-25 research (libasciic source, crossterm events, tcell releases). Details get confirmed with the user before each one starts.

### [x] S1: Our own terminal layer
- New `internal/term`: our screen interface (size, put a cell, show, clear, sync), our event types (key with pressed / released / repeat, mouse, resize, focus), a tcell v2 adapter, and a fake screen for tests. app, render and ui use only `term`.
- Why first: tcell v3 removes the simulation screen our tests use, and this layer means switching to tcell v3, or to our own input reader, only touches the adapter.
- Acceptance: nothing changes on screen; all tests pass on the fake screen; build, vet, gofmt, tests on Windows and Linux CI.

### [x] S2: Real key releases (tcell v3)
- Spike first (timeboxed): tcell v3.4+ key events (`Pressed()`, `Repeat()`, kitty protocol event types, Windows key-up). Then a v3 adapter for `term`. In events mode, held keys come from press/release, so moving and turning together works and taps are exact. Terminals without releases keep the P1-14 guessing; async stays the Windows default.
- Needs the user's OK: tcell v2 -> v3 is a dependency change.
- Acceptance: unit tests for press/release tracking; on screen (user): events mode in Windows Terminal (and, through S0ra, Ghostty or kitty on Linux) moves and turns at the same time.

### [x] S3: The rest of the Mixed style
- Color threshold (libasciic's compression): reuse the previous cell's style when no color channel differs by more than a threshold, so tcell sends fewer color codes. Works for mixed and half-block; a setting, off by default. Measure bytes per frame with and without.
- Adjustable mixed background brightness (0-100%, default 20%) in Settings.
- Acceptance: unit tests for the threshold and the background setting; bytes-per-frame numbers recorded; on screen (user): mixed looks right at a few thresholds.

# Visual quality

A separate track (D44): the user asked for better visuals. Five tasks in two runs, each run checked on screen by the user: V1-V3, then V4-V5. Lighting touches (darker corners) were left out; they belong with M4.

### [x] V1: Textured floor and ceiling
- Floor and ceiling casting (Lodev's method, matched to the wall math): a tile texture on the floor and a panel texture on the ceiling, with the same distance fog as walls. The floor stays darker than close walls and the ceiling stays dim. With wall textures off (Settings), floor and ceiling stay the flat gradient.
- Update (D45): after trying it, the user chose a flat near-black ceiling again; only the floor is textured.
- Acceptance: unit tests (floor pixels follow world position, fog with distance, textures-off unchanged); render benchmark within budget; on screen (user): the floor reads as a surface that moves correctly when walking and turning.

### [x] V2: Smooth wall edges
- The top and bottom pixel of each wall column blend wall and floor/ceiling by how much of the pixel the wall covers, so wall edges don't step.
- Acceptance: unit tests (coverage blending, full pixels unchanged); on screen (user): wall tops and bottoms look smoother in half-block.

### [x] V3: Less shimmer on far walls and the floor
- Mipmaps: each texture also gets half, quarter, ... size versions (box-averaged); far walls and far floor sample the version that matches how many texture pixels one screen pixel covers, so patterns don't flicker when turning.
- Acceptance: unit tests (mip levels, level choice by distance); on screen (user): far walls and the floor near the horizon flicker less when turning.

### [x] V4: Sharper wall textures
- Brick, stone and metal redrawn at 32x32 (more detail up close; V3's mipmaps keep them calm far away).
- Acceptance: textures parse, tests pass; on screen (user).

### [x] V5: Finer ASCII and mixed shading
- A longer brightness ramp for the ASCII and mixed modes, so shading has more steps.
- Acceptance: unit tests (ramp order, full range used); on screen (user).

### [x] V6: Render scale for big windows
- Settings "Render scale": auto (default), full, half. Auto draws the 3D view at most 160 columns wide (2x2 screen cells per view cell from 161 columns, 3x3 from 321, ...) so neighboring cells share colors and the terminal gets far fewer color codes; full is today's behavior; half always uses 2x2. HUD, minimap and menus stay at full resolution. Saved in save.json (older files: auto).
- Why: the user found big windows (zoomed out) lag in Windows Terminal: bytes per frame grow with the cell count (D45).
- Acceptance: unit tests (scaled presenting, frame size, auto choice, setting saved); bytes per frame at a big size with and without; on screen (user): a zoomed-out window stays smooth.
- Result (D46): only 4-12% fewer bytes at 320x90; the user chose to keep it.

### [x] V7: Stronger texture and floor contrast
- Texture detail stands out more (the darkest texture pixel keeps 30% of the shade, was 45%) and the floor is a little brighter (base 140, was 120), so brick lines, stone gaps and floor tiles read clearly.
- Acceptance: tests updated; on screen (user).

### [x] V8: Faster drawing
- From the user's Take 4 recording: zoomed out at 60 FPS the HUD showed draw 5-9 ms. DrawFrame now works out per-row floor values once per frame, walks each column in row ranges (only the two wall edge rows compute coverage), looks wall texture shades up in a per-column table, and toGray uses plain comparisons. Same picture.
- Acceptance: all render tests pass unchanged; benchmarks faster; on screen (user): smoother when zoomed out.

# Maps and gameplay update

All the ideas from the 2026-10-25/26 discussion, added to the plan first (D40). The numbers are IDs, not the order: the order, the batches of five and the details get decided before any of them starts.

## Maps and world

### [~] M1: Minimap size and border
- The scrolling window shows at most 16x16 map cells (now 24x16); a 16x16 map like the Arena keeps its full size. A border goes around the minimap so it stands apart from the 3D view.
- Follow-up (D48, from the Take 4 review): the window is also at most a third of the view rows across and down, so on a normal-size terminal it stays clear of the horizon.

### [~] M2: Props
- Barrels, crates, pillars, lamps, debris, bodies: map characters drawn as sprites. Some block movement and bullets (crates, pillars), some don't (debris).

### [~] M3: Doors
- Wolfenstein-style sliding doors: open when you walk up (or press a key), close after a while; enemies open them too.

### [~] M4: Lighting per map
- Header lines for fog and darkness; lamps that light nearby walls and floor; some flickering lights.

### [~] M5: Redesign all four maps
- Rebuilt from scratch with M2-M4, so every map follows the same design rules: real layouts (rooms, a loading dock, buildings you can enter), cover, landmarks.

### [~] M6: New maps
- Two to four more, for example a sewer, an office floor, a hospital, a subway station.

### [~] M7: Map unlock rule
- Decide: keep "survive 5:00 to unlock the next map", shorten it, open all maps, or a setting for it.
- Built as a setting (my default, for the user's review): Settings "Map unlocks": survive 5:00 (default, as before), survive 2:00, or all open. "All open" doesn't change saved progress.

### [~] M8: Minimap fog of war
- The minimap shows only places you've seen.

### [~] M9: Custom maps
- A `-map file.txt` flag and a maps folder next to the save file, for your own maps.

## Gameplay

Decided with the user on 2026-10-30 (D49; the guide and release added 2026-10-31, D51). Order and batches: **batch 1** M10, M11, M12, M14, M16; **batch 2** M13, M15, M17, M18a, M18b, M18c (D53); **batch 3** M19, M20. The user reviews after each batch. Every new thing gets its own sound effect.

**Tiers (changed for M12 and M13):** a tier lasts 20 kills or **3 minutes** (was 4), whichever comes first, so a player who only hides still moves up. New enemy types and smarter behavior arrive by tier:

| Tier | Reached at (whichever first) | New enemy (M12) | Enemies learn (M13) |
|---|---|---|---|
| 1 | start | zombies, shooters (as now) | nothing new (as now) |
| 2 | 20 kills or 3:00 | runner | hear gunshots and come to the sound |
| 3 | 40 kills or 6:00 | tank | shooters take cover behind props |
| 4 | 60 kills or 9:00 | exploder | zombies come from several sides |
| 5+ | every 20 kills or 3:00 | all types | faster spawns, tougher enemies (as now) |

### [x] M10: Medkits and armor (batch 1)
- Regeneration only up to 75 HP (same delay and speed as now); above that only medkits heal.
- Medkit pickups heal (amount tuned in M19); armor pickups give 50 armor, up to 100. Armor takes half of each hit until it runs out. The HUD shows armor.
- Both spawn like ammo boxes; their own sound when picked up.

### [x] M11: More weapons (batch 1)
- Keep the current gun; add a shotgun (several pellets in a spread, strong up close, slow) and a rifle (accurate, high damage, slow).
- Each gun has its own ammo type, found as pickups. You start with the current gun and find the others on the map (gun pickups).
- Switch with 1, 2, 3 and the mouse wheel. Changes the PRD's "one gun" rule.
- Own sprites for the gun in hand and pickups; own shot sounds.

### [x] M12: More enemy types (batch 1)
- Runner (fast, weak) from tier 2, tank (slow, tough, hits hard) from tier 3, exploder (runs at you, blows up near you or when shot) from tier 4; see the tier table.
- Own sprites and sounds.

### [x] M13: Smarter enemies (batch 2)
- Enemies start dumb (as now) and learn by tier (table above): tier 2 they hear gunshots and come to the sound; tier 3 shooters take cover behind props; tier 4 zombies come from several sides.
- Anti-camping: after 60 seconds without a kill, every enemy hunts the player directly (heads straight for them), until the next kill.

### [x] M14: Hazards (batch 1)
- Explosive barrels: a new red barrel map character; the gray barrel stays plain cover. Shooting one blows it up and hurts enemies and the player nearby; chain reactions with other red barrels.
- Acid floor: a new green map character that hurts while you stand on it; placed on a few maps.
- Explosion sound.

### [x] M15: Waves (batch 2)
- A second mode next to endless survival, chosen when you press Play: 10 waves with a 15-second break between them (reload, pick things up); you win after wave 10. Bests kept per mode.

### [x] M16: Game feel (batch 1)
- Sprint while F is held, with stamina (drains while sprinting, refills when not); the HUD shows it.
- A marker showing which way a hit came from; a hit marker on the crosshair when you hit; screen shake on big hits (explosions, tank hits); footstep sounds while walking.

### [x] M17: 3D sound (batch 2)
- Louder when close, quieter far away, and panned left/right toward the source, so you hear where enemies are. Needs stereo playback (oto supports it).
- Missing sounds found after the batch 1 review (2026-10-31): each enemy kind makes its own sound near the player (e.g. a runner screeches when it starts chasing you), not only when it arrives; zombie and runner hits get their own sound (only the tank's hit has one now); a death sound per enemy kind; the shotgun and rifle get their own reload sounds (now all three use the pistol's).

### [x] M18a: In-game guide (batch 2)
- A **Guide** item in the title menu and the pause menu (so it can be read mid-run). A list of pages, moved through with the arrows or W/S, Enter, Esc; pages scroll so they fit from 80x24 up.
- Pages: controls and HUD; enemies (picture, HP, speed, damage, the tier it arrives in, how to beat it); weapons and ammo (picture, damage, magazine, fire rate, when it appears, when to use it); pickups; hazards (red barrels, acid, explosions); tiers (what changes, 20 kills or 3 minutes); maps (name, size, hazards, a tip); settings; tips for surviving longer.
- Numbers come straight from the game code (TierFor, the gun table, the enemy constants), not typed in, so balancing (M19) can't make the guide wrong.
- Pictures are the game's own sprites. Locked maps show only their name and "locked" until unlocked (a surprise, the user's choice).
- Acceptance: unit tests (every enemy kind, gun, pickup kind and map has a page; numbers match the code; locked maps hidden; fits 80x24; both menus open it); on screen (user).

### [x] M18b: Guide 3D views (batch 2)
- Every guide entry gets a live 3D view drawn by the game's own renderer: enemies, gun pickups, ammo, medkit, armor, red barrel and acid each stand in a small lit display room, and the camera slowly circles them (360°). Weapons also show the gun in hand, firing now and then. Maps show a top-down picture and a view where the camera stands in the map and slowly turns all the way around.
- The objects are flat pictures that always face the camera (like Doom), so circling shows the room turning around them, not their sides. Drawing every object from 8 angles for a true turntable is a separate, bigger idea (not planned).
- Acceptance: unit tests (each entry has a view; the camera goes all the way around; locked maps get no view); on screen (user).

### [x] M18c: Guide polish (batch 2, after the user's test of M18a/M18b)
- Decided with the user on 2026-10-31 (D53).
- **Panel:** guide pages sit in a centered panel that looks like the menus (light box, dark text), with the 3D view as a dark window inside it. The panel grows with the window up to 100 columns wide, then stops and stays centered; the 3D view grows with it.
- **Explored maps:** the cells you've seen on each map are remembered across runs (save.json). A map page shows a big top-down picture (most of the panel width, the text under it): explored parts normal, unexplored parts dark, the map's outer border always drawn, and "Explored: n%". The 360° turn shows only after you've played that map once. Locked maps: the name and the full border, dark inside.
- **Controls:** a keyboard picture with a small mouse (left, wheel, right). Keys the game uses are colored by group (movement, combat, view, other), the rest dim, with a legend underneath. Up/Down step through the used keys one by one; the selected key lights up and a line says what it does (Left/Right still turn the guide pages).
- **HUD:** a drawn example of the bottom line with each part pointed out and named.
- **Not met yet:** enemies you haven't met and guns you haven't found show as a dark silhouette and "???" until you meet or find them (remembered across runs).
- Acceptance: unit tests (panel size and centering up to 100 columns; explored cells saved, merged and drawn, border always drawn; turn only after played; locked maps; every used key has a description and a group; HUD parts labeled; silhouettes until met or found); on screen (user).

### [~] M19: Balancing and a new demo (batch 3)
- Was M18. Carried over from P1-18: playtest tiers 3+, the ammo cap, the new pickups, weapons, enemies, hazards and waves; record a new demo.
- 2026-11-11, the user's playtest: everything feels right except ammo, which runs short; the user is fine with low ammo for now (it keeps the tension), so no numbers changed. Revisit with the public testers' feedback. Still to do: the new demo, from the user's video edit.

### [~] M20: Public test release (batch 3; done before M19's demo, the user's choice)
- Downloadable builds (Windows and Linux; macOS if it can be tested) on a GitHub Releases page, built by CI.
- The README says up front that Windows SmartScreen / Smart App Control may warn about an unsigned exe, and what to do.
- A version number, shown in the game (title menu) and by `-version`.
- A GitHub issue template for feedback: version, system, terminal, what happened.
- Linux (sound needs ALSA or PulseAudio) and macOS tested, or the release says "Windows first".
- Built 2026-11-11 (D55): `-version`; `.github/workflows/release.yml` (a version tag runs the checks, checks the tag against `-version`, builds windows/amd64, linux/amd64 and linux/arm64 without cgo, and makes a DRAFT pre-release with the files, SHA256SUMS and `.github/release-notes/<tag>.md`); the feedback issue form (`.github/ISSUE_TEMPLATE/feedback.yml`, with an ammo question); README download section and SmartScreen / Smart App Control notes. Windows first: Linux passes CI but is little played; no macOS build (can't be tested). Left for the user: check the draft release and publish it; try a download.

# UI redesign (direction A, "Arcade terminal")

Decided with the user on 2026-11-10 (D54) from mockups on a design canvas (private, only the user can open it: https://claude.ai/artifact/TLTTTMFedGYf166nu3AjAy; where the canvas and this file differ, this file wins). Built in one run, U1 to U6 back to back, one commit per task, one review of everything at the end. It comes before M19 and M20. Nothing is pushed until the user asks.

For every U task: it fits 80x24 (tests check it); mouse clicks still work; `go build`, `go vet`, `gofmt -l`, `go test` pass; the user signs off what's on screen. Project rules stay: world and game never import tcell or audio; game emits events; movement uses dt; beginner comments for Go concepts.

## Shared rules for U1-U6

**Palette** (truecolor, `term.RGB`):

| Use | Color |
|---|---|
| Background | `#0b0d0e` |
| Panels, HUD background | `#111416`, cards `#15181a` |
| Borders, empty bar cells | `#3a3f42`; lines inside panels `#2a2f33` |
| Normal text | `#c9ccce`; bright/bold text `#f2f2f2` |
| Dim text (help lines, labels) | `#8a9196` (readable on the dark ground; `#5b6266` on the early boards was too dark) |
| Accent (selection bar, titles, highlights) | amber `#ffb000`; logo gradient `#ffc533` → `#ffb000` → `#ff9a1f` → `#ff8a3d` → `#e8702e` |
| Health | red `#ff5c5c` |
| Armor | blue `#6aa9ff` |
| Stamina, tagline, warnings | yellow `#ffe066` |
| Good news (NEW BEST, wave cleared, win) | green `#5ee27a` |
| Key-cap chips | `#2a2f33` background, `#e8e8e8` text |

**Symbols:** fancy by default (♥ ◆ » ▰▱ ❙ ╭╮ ╔╗ ▶ ✓ ✦, key caps ↑↓ ⏎ esc), plus a "Symbols: fancy / simple" setting. Simple: `HP [#######---]`, `AR [#####-----]`, `ST [######----]`, magazine `||||||....`, current gun `[1] 2 3`, boxes `+--+`, keys `Up/Down Enter Esc`. Why: Windows Terminal draws the fancy ones; old console windows and some fonts show `?` or empty boxes instead.

**Crowded text (must fix):** the block letters touched each other on the first boards ("TERMFPS": R ran into M; "YOU DIED", "YOU WON" too). At least 2 empty columns between letters, at least 6 between words. Check in the real terminal (Windows Terminal), not only on the canvas. The small two-row logo at the top of the Play and Settings screens says "TERM FPS" as two words too. The user knows of no other crowded text so far; keep an eye out.

**Version:** v0.75, codename "As*****". The masking is the joke: write it exactly like that everywhere (title screen, release notes, commits, docs), never spelled out. Shown bottom right on the title screen as `v0.75 “As*****”`, and used for the M20 public test release. Everyone who plays the public test sees the codename.

**Animations (only if they don't slow the game):** the screen is already redrawn 30 to 60 times a second; a small animation only changes a few characters in frames that are drawn anyway. The 3D view (thousands of cells) is what costs time, and animations don't touch it. Keep them short (a few frames), never inside the 3D drawing, measure with the FPS display and a benchmark, add "Animations: on / off", and drop any that measurably slow things. Candidates: logo shimmer and tagline pulse (title only); selection bar moving smoothly; toasts sliding in from the top; "YOU DIED" / "YOU WON" appearing row by row; game-over numbers counting up; NEW BEST blinking a few times.

**Screen size:** as it is for now (fits 80x24, centered in bigger windows). Revisit later.

### [x] U1: Look kit
- Shared colors and pieces in internal/ui: boxes, ▶ selection, key-cap footer, bars, switches, tabs, badges, toast, block font, symbol sets.
- Every menu and the guide switch to the new look. The Symbols setting (fancy / simple).
- Guide panel: dark (`#111416`) with a border, header "GUIDE › SECTION NAME" on the left and page dots "● ○ ○ ○ 1/4" on the right, the page title as an amber label, footer key caps "←→ page ↑↓ scroll esc back". The keyboard picture, map pictures and 3D views stay, recolored.

### [x] U2: Title and Play screen
- **Logo "TERM FPS"**: 5-row block letters, 5 columns each, 2-column gaps, 6-column word gap: 51 columns wide, centered. Rows from amber (top) to orange (bottom). Slow shimmer: every ~6 s a light band passes over the letters, about one letter per 0.2 s.
- **Random tagline each start** (like Minecraft's splash text), centered under the logo in yellow, gently pulsing. Picked with the injected `*rand.Rand`. The list:
  - a first-person shooter game in your terminal
  - the shooter that lives in your terminal
  - survive the horde, one character at a time
  - how long can you last?
  - keep moving. keep shooting.
  - made of text, full of teeth
  - mind the red barrels
  - now with exploders
  - don't camp
  - no mouse-look required
  - rendered one character at a time
  - the walls are ASCII, the danger is real
- **Menu** in a double-line box: PLAY (survival or waves), SELECT MAP (6 maps · 2 locked), GUIDE (how to play), SETTINGS, QUIT. ▶ on the selected item; a description line under the box ("› Pick a mode and a map, then go."). SELECT MAP may go away because the Play screen already picks the map; decide when building (keep the item only if it adds something, e.g. bests and explored % per map).
- **Best run** line ("best on The Arena 12:31 · 142 kills · tier 5"), key-cap footer, version bottom right.
- **Moving map background** (slow motion only):
  - Map: random among the maps the player has unlocked (locked maps stay a surprise). Injected rand.
  - Camera only, no game simulation: no enemies, no pickups, no sounds. Doors treated as open.
  - Walk at about 30 to 40% of walking speed along a route through open cells (world path / distance field, never through walls). Smooth turns (limited degrees per second). At some waypoints it stops and slowly looks around (90 to 180 degrees), then walks on. Speeds use dt.
  - Drawn with the normal renderer, dimmed so the menu stays readable; the logo and menu on top.
  - Keeps moving slowly behind Settings and the Guide opened from the title.
  - Cost: lower frame rate and/or half render scale while on the title. Memory stays flat (the Frame is reused, nothing new per frame); the cost is CPU, which the lower frame rate keeps down.
  - Setting "Title background: on / off" for slow terminals.
  - Tests: the route never enters a wall; movement follows dt; the same seed gives the same tour.
- **Menu sounds**: a soft tick when the selection moves, a click on choose, a soft low sound on back (Esc), a short buzz on a locked item. Synthesized in internal/audio like the other effects; they follow Sound on/off and Volume. Played by the app's menus (game never calls audio).
- **Play screen** (the mode and map menus merged):
  - Small "TERM FPS" logo top left, screen name "PLAY" top right.
  - MODE: `SURVIVAL` / `WAVES` toggle (←→), with a one-line explanation ("endless · tier up every 20 kills or 3:00", or "10 waves with breaks, clear them all to win").
  - MAP list box: name, size, explored % per map; locked maps dim with "▒ locked"; "+ your maps (maps folder)".
  - Preview box titled with the map's name: top-down picture (explored parts lit, the rest dark with the border visible, like the guide's map pages), plus "explored 64% · 3 doors · 4 red barrels".
  - BEST line for the map: survival (time, kills, tier) and waves (best wave or "won in 7:12").
  - `▶ START` button. Footer: ↑↓ map, ←→ mode, ⏎ start, esc back.

### [x] U3: Settings, pause, messages
- **Settings, four tabs.** Tab key or ←→ on the tab row switches tabs. Each setting: label, value, one dim help line under it. "changes apply at once and are saved". Values: `◀ value ▶` for choices, `[■ on ]` switches, `▮▮▮▮▯▯ 40%` bars.
  - VIDEO: Render mode (half-block · shade · ASCII · mixed, also V in game), Render scale (auto keeps big windows smooth), FPS (30 or 60), Wall textures, Mixed background (bar), Symbols (fancy / simple, new), Title background (on/off, new), Animations (on/off, new).
  - AUDIO: Sound (stereo, from where things happen), Volume (bar, ←→ by 10%). Menu sounds follow these too.
  - CONTROLS: Key mode (Windows: move and turn at once), Pixel aspect (if walls look too tall or too flat), Minimap at start (M shows or hides it).
  - GAMEPLAY: Map unlocks (survive 5:00 · survive 2:00 · all open), Minimap fog of war, Color threshold (fewer color codes for slow terminals).
  - Check when building: every current setting has a tab (render mode, mixed background, color threshold, render scale, FPS, minimap at start, wall textures, sound, volume, key mode, pixel aspect, map unlocks, minimap fog).
- **Pause**: a card over the dimmed 3D view: "II PAUSED", map name and mode; TIME, KILLS, TIER, HP; "next tier in 0:53 or 17 kills" (waves mode: wave progress instead); RESUME, RESTART, GUIDE, SETTINGS, QUIT TO MENU ("this run still counts"). Footer: ↑↓ pick, ⏎ choose, esc resume.
- **Error box** over any screen: red border, "! COULDN'T SAVE YOUR PROGRESS", the real error text ("open save.json: Access is denied."), what it means ("This run was played, but it is not saved."), an OK button. The same box for map load errors. No silent fallbacks.
- **Sound warning** (yellow, one line): "Sound is off: no audio device found. The game runs without it."
- **Window too small** (below 80 x 24): "WINDOW TOO SMALL", "now 62 × 20, needs 80 × 24", a small picture of the two sizes, "Make the window bigger or the font smaller. The game waits here and carries on by itself."

### [x] U4: In game
- **HUD, two lines** (background `#111416`; costs 1 row of the 3D view):
  - Line 1: `♥ ▰▰▰▰▰▰▰▱▱▱ 76   ◆ ▰▰▰▰▰▱▱▱▱▱ 50   » ▰▰▰▰▰▰▱▱▱▱` ... `KILLS 23`
  - Line 2: `PISTOL 18/50  ❙❙❙❙(magazine strip, used rounds dim)   1 2 3 (current gun bright)` ... `TIER 2  TIME 3:07` (waves mode: `WAVE 3/10`, and during a break `BREAK 0:12`).
  - FPS / mode / facing / timings / position stay available as the optional extras line (as today).
- **Notices** (tier up, hunt, new map, waves): a toast at the top center, amber on `#1d2124`, e.g. "TIER 2 · runners join · enemies hear you".
- **Minimap**: box top right with the map name in the frame ("┌─ WAREHOUSE ─┐").
- **Wave cleared**: a centered banner with an amber border: "✓ WAVE 3 CLEARED", "next wave in 12", a countdown bar, "wave 4: 10 enemies · tier 2", "reload and pick things up". When the break ends: a toast "WAVE 4 · 10 enemies · GO".
- Crosshair, hit markers, red hurt edge, gun drawing: unchanged (game art, not UI).
- The guide's HUD page explains the new HUD.

### [x] U5: Run stats, game over, win
- **Run stats to track** (new, in the game package through events; the injected rand untouched): shots fired and hits (accuracy), what killed you (enemy kind, explosion, acid), kills per enemy kind, damage taken, HP left, explored % this run.
- **Game over**: big red "YOU DIED" (spaced block letters), "killed by a tank · The Warehouse · survival"; stat boxes TIME, KILLS, TIER, ACCURACY with NEW BEST badges (green) or "best 5"; "this run: 4 tanks · 2 exploders · 31 runners · explored 71% of the map"; buttons TRY AGAIN, CHANGE MAP, MAIN MENU (←→ pick).
- **Win** (waves): green "YOU WON", a few amber ✦; "all 10 waves cleared · map · waves mode"; boxes TIME (NEW BEST = fastest win), KILLS ("best 142"), ACCURACY, HP LEFT; the per-kind line; PLAY AGAIN, CHANGE MAP, MAIN MENU.
- NEW BEST compares with the player's best on that map (bests are already kept per map in save.json).

### [x] U6: Guide pages as pictures
Text-only guide pages become pictures (numbers always from the code, as now):
- **Tiers**: "20 kills or 3:00, whichever first"; progress bars "to next tier 7/20 kills or 1:30/3:00"; a table per tier 1 to 6: enemy HP bar (red), damage bar (orange), speed bar (yellow), new enemy every N s, max alive as dots; "NEW KINDS T1 zombie shooter, T2 runner, T3 tank, T4 exploder"; "keeps rising: speed up to x1.5, at most every 2 s, at most 14 alive".
- **Enemies learn**: four boxes with top-down pictures: TIER 1 straight at you; TIER 2 they hear you (15 steps, rush 1.4x); TIER 3 shooters hide; TIER 4 they surround you; then a "DON'T CAMP" bar (1:00 without a kill → horn → hunt, 1.2x faster until your next kill).
- **Waves mode**: a bar chart of enemies per wave (4, 6, ... 22) colored by tier (T1 to T5), labels W1 to W10; a timeline "WAVE 1 ─ 15 s break ─ WAVE 2 ─ ... ─ WAVE 10 ─ ★ WIN"; first break 10 s; bests per map.
- **Health and stamina**: a 100 HP bar with the 75 line (heals by itself up to 75, medkits go higher); "hit → wait 4 s → +5 HP every second, up to 75"; armor halves damage; stamina: F to sprint at 1.6x; full lasts 4 s; refills in 5 s; empty → no sprint until back to 30%.
- **Settings**: the same wall in half-block, shade, ASCII and mixed side by side; a small preview per other setting.
- **Tips**: a small picture beside each of the 7 tips.
