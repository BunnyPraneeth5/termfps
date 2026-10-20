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

P0 status: core (T1-T18) and stretch T19 are done. Stretch T20 (red targets) moved into P1 as task P1-02 (D29).

### P1: a real single-player game (planned 2026-10-20)
Tasks: TASKS.md "P1". Decisions: DECISIONS.md D21-D29.

**The game: endless survival.** Each map is an arena. Enemies keep coming and get stronger. You play until you die, then try to beat your best.

**Flow:** Start menu (Play, Select map, Settings, Quit) -> pick a map -> survive -> Game over screen (kills, time survived, tier reached, your best) -> Retry or back to the menu.

**Survival rules**
- Enemies spawn at random free cells that you can't see and that are at least a few cells away.
- **Difficulty tiers:** every **20 kills or 4 minutes, whichever comes first**, the next tier starts. Enemies get more HP and damage, move a bit faster, and spawn more often, with more alive at once. The HUD shows the tier. Exact numbers are tuned in balancing (P1-16).
- **Score:** kills, time survived, tier reached.
- **Player:** 100 HP. Health pickups or slow regeneration are open for balancing (P1-16), not planned yet.

**Weapon and ammo**
- One gun. The magazine holds **25**. You start with **25 loaded + 50 spare**.
- **Space** (or **left-click**) fires: an instant hitscan shot along the crosshair at the screen center, about 3 shots per second while held.
- **R** (or **right-click**) reloads, taking about 1.5 s with no shooting meanwhile. **Auto-reload** when the magazine hits 0 and there are spares.
- **Ammo pickups** appear at random free cells at random times (about every 20-40 s, at most 3 on the map). Walking over one gives +25 spare. The spare ammo cap is tuned in balancing.
- HUD: `AMMO 18 / 50`, `RELOADING...`.

**Enemies** (both are sprites hidden correctly behind walls, and both find a path around walls)

| | Zombie | Shooter |
|---|---|---|
| Color | Green | Red |
| Attack | Melee, up close | Shoots from a distance (hitscan, accuracy drops with distance) |
| Behavior | Rushes straight at you, often in groups | Keeps some distance, needs line of sight |
| Toughness | Tanky, easy to hit | Fragile |

**Weapon graphics:** a pixel-art gun at the bottom center, a **yellow** muzzle flash (~0.1 s) and a small recoil when firing, maybe gentle bobbing while walking. Drawn into the Frame, so it shows in all render modes.

**Wall textures:** each wall character has its own pattern: `#` brick, `%` stone, `&` metal. Textures are 16x16 grayscale images stored as text files, embedded like maps. Fog and side-darkening still apply. Settings: Textures on/off. They're coarse at terminal resolution and mostly lost in ASCII mode.

**Menus**
- **Start:** Play, Select map, Settings, Quit.
- **Pause** (Esc during play; the game freezes, dimmed behind the menu): Resume, Settings, Restart, Quit to menu.
- **Settings:** render mode, FPS (30/60), minimap at start, textures on/off, sound on/off + volume, key mode, pixel aspect.
- **Keyboard:** Up/Down or W/S move, Enter selects, Left/Right change a setting, Esc goes back.
- **Mouse:** click a menu item to select it. Ctrl+C quits instantly from anywhere.

**Maps:** four arenas of different shapes: **16x16, 32x32, 64x32, 16x64**, all embedded from `maps/`. An optional header line names a map (`; name: The Warehouse`). Map characters: `#` `%` `&` walls, `.` floor, `P` start, `T` red target block (shootable). The first map is open; each next map **unlocks after surviving 5 minutes** on the previous one. On maps too big for the screen, the minimap shows a window around the player that scrolls.

**Saving:** **settings and progress** (unlocked maps; best kills, time and tier per map; total kills), written automatically to `%AppData%\termfps\save.json`. The write is atomic: a temp file, then a rename. A damaged save file is reported in the menu, kept as `save.json.bad`, and replaced with defaults. **No mid-game saves.**

**Mouse in the game:** left-click shoots, right-click reloads. No mouse-look: terminals can't lock the mouse.

**Sound:** played through the computer's speakers by the game itself (the terminal can't play sound), using the **oto** audio library. That's a new dependency, approved by the user. Effects are **generated in code** (gunshot, reload, empty click, zombie groan, enemy shot, hurt, pickup, tier up), so there are no sound files and no copyright questions. Settings: sound on/off and volume. Local play only. On Linux, oto needs the ALSA development libraries.

**Not in P1:** mouse-look, mid-game saves, a "clear the level" mode, health pickups (unless balancing needs them), multiplayer, AI agents.

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
