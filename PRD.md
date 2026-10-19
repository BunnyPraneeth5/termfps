# PRD: termfps (working title)

## Problem
Retro FPS games need a GPU and a game client. I want a first-person shooter that runs entirely inside a terminal, mostly monochrome, as a single small binary. Later it becomes multiplayer TDM over LAN and SSH.

## Who it's for
- Me first. Learning Go, realtime rendering, and later game networking.
- Developers who live in the terminal and like retro games.
- Demo audience: friends, my company team, GitHub/X/Reddit.

## This version (P0 prototype)
Single player. Walk around one small map in first person. That's it.

Time budget: 4-5 days (2 weekends).

## v1 features (P0)
1. Opens full-screen in the terminal, quits cleanly with Esc or Ctrl+C, terminal restored after exit (even after a crash).
2. Loads one 16x16 map from a text file embedded in the binary.
3. Raycasted first-person view of the walls.
4. Walls look proportional (no fisheye, corrected for terminal cell shape).
5. Movement: W/S forward/back, A/D strafe, Left/Right arrows (or Q/E) to turn.
6. Collision with walls, with sliding along them.
7. Half-block pixel rendering (`▀`, two grayscale pixels per cell): closer walls brighter, side faces darker, distance fog, simple floor and ceiling.
8. Minimap overlay, toggle with M.
9. HUD line: FPS, position, facing direction.
10. Handles terminal resize without breaking.
11. A 30-second demo recording and a short README (build, run, controls, caution).

Known P0 limit: with tcell's key events you may not be able to move and turn at the same time. Spike T8 tries real key state; see DECISIONS D12.

Stretch (only if 1-11 are done, in this order):
- Render mode toggle with V: half-block, shade blocks (`█▓▒░`), ASCII. Default to ASCII if the terminal doesn't support truecolor.
- Target blocks (red) on the map. Space shoots straight ahead. A hit target disappears.

## Out of scope for P0
- Multiplayer (LAN, SSH, online)
- Enemies or bots that move
- Sprites, weapons graphics, sound
- Wall textures (bricks etc.)
- Menus, settings, multiple maps, saving
- Agent arena / LLM players
- Mouse input
- Linux/Mac packaging (should work, not tested yet)

## Success criteria
- `go build` produces one binary that runs on Windows in Windows Terminal (PowerShell).
- Holds 30 FPS at 120x40 in half-block mode on my laptop (i5 10th gen, 8GB).
- I can walk the full map without passing through walls or crashing.
- Walls look like walls, not stretched or curved.
- A 30-second screen recording looks good enough to show people.

## Style
- Mostly monochrome. Walls, floor, and ceiling are grayscale.
- Default look is half-block pixels. Shade blocks and ASCII are alternatives (stretch).
- Color only for rare important things (targets now, players/enemies later).
- User caution: "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."
