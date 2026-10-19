# PRD: termfps (working title)

## Problem
Retro FPS games need a GPU and a game client. I want a first-person shooter that runs entirely inside a terminal, mostly monochrome, as a single small binary. Later it becomes a full single-player game, then you vs AI agents (Claude Code, Codex, opencode...), then multiplayer TDM over LAN and SSH. See "Roadmap" below.

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

Input: on Windows the game reads real key state, so moving and turning at the same time works (DECISIONS D19). Elsewhere it falls back to terminal key events, where that may not work (D12).

Stretch (only if 1-11 are done, in this order):
- Render mode toggle with V: half-block, shade blocks (`█▓▒░`), ASCII. Default to ASCII if the terminal doesn't support truecolor.
- Target blocks (red) on the map. Space shoots straight ahead. A hit target disappears.

## Out of scope for P0
These are planned for later phases; see "Roadmap".
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

## Roadmap (after P0)
Agreed 2026-10-19 (DECISIONS D20). Nothing here is built yet. Each phase gets its own plan (scope, design, tasks) before any code, the same way P0 did.

P0 status: core (T1-T18) and stretch T19 are done. Stretch T20 (red targets, Space to shoot) is still open and doubles as the first step of P1, because enemies need shooting.

### P1: a real single-player game
In scope:
- **Enemies / bots that move:** patrol, chase when they see you (line of sight with the existing raycaster), health and damage.
- **Sprites:** enemies drawn as billboard images in the 3D view, hidden correctly behind walls. Enemies get color (Style: color only for important things).
- **Weapon graphics:** a gun at the bottom center, with a muzzle flash when firing.
- **Wall textures:** simple patterns (for example bricks). They'll be coarse at terminal resolution and mostly lost in ASCII mode.
- **Menus:** title screen, pause menu, settings.
- **Multiple maps:** a map select screen.
- **Saving:** settings and progress. What exactly to save is decided when planning P1.

Undecided, decide when planning P1:
- **Sound:** leaning toward necessary. Terminals can't play sound, so it needs an audio library (a new dependency, needs approval). Works only when playing locally; over SSH there's only the terminal bell.
- **Mouse input:** terminals report clicks and which cell the mouse is over, but can't lock the mouse, so smooth mouse-look isn't possible. Click to shoot is.

Suggested order: T20 -> enemies + sprites -> weapon graphics -> menus + multiple maps -> wall textures -> saving -> mouse and sound (if chosen).

### P2: server / client split
The game logic runs as a server and the screen is a client. Both AI opponents (P3) and multiplayer (P4) need this. `world` and `game` already have no terminal code (D8), which is what makes the split possible.

### P3: AI-agent opponents (you vs an AI agent)
The player plays against an AI agent they already use in the terminal: Claude Code, Codex, opencode, or others.
- **Likely connection:** the game exposes an MCP server with game-only tools (look, move, turn, shoot). Most CLI agents support MCP, so one server works for all of them.
- **Game modes:** several, decided when building P3. Ideas so far: hide and seek, face-to-face duel.
- **Open questions for P3:**
  - Speed: agents take seconds per decision while the game runs at 30 FPS. Options: turn-based play, high-level commands ("hold the north room") carried out by a built-in bot, or the agent writes a bot script before the match. Different modes may use different options.
  - Cost: every agent decision uses the user's tokens, so matches need limits.
  - Fairness: the agent only sees what its bot can see (line of sight), not the whole map.
  - Safety: the agent gets game tools only, never anything that reads files or runs commands.
  - An MCP library for Go would be a new dependency (needs approval).

### P4: multiplayer
Team deathmatch over LAN first, then over SSH (ASCII mode for low bandwidth, Linux server).
