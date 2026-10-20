# termfps

A first-person game that runs entirely inside your terminal. A raycaster
draws the walls with half-block pixels (`▀`, two grayscale pixels per
character cell), so you get a retro 3D view with no GPU and no game client,
from one small binary.

This is the **P0 prototype**: single player, one 16x16 map, walk around in
first person. Multiplayer is planned later.

[![termfps running in Windows Terminal](docs/screenshot.png)](docs/termfps-demo.mp4)

▶ [Watch the 27-second demo](docs/termfps-demo.mp4) (MP4, 4.7 MB)

## Requirements

- **Go 1.27.1 or newer** to build.
- **A truecolor terminal.** Windows Terminal is the main target. VS Code's
  terminal also works.
- **At least 80x24 characters.** Smaller windows show a "Terminal too small"
  message until you resize. About 120x40 looks best.

> Half-block mode needs a truecolor terminal and more bandwidth than plain
> text. Press `V` for shade-block or ASCII mode, which use plain characters
> and the terminal's own colors. Without truecolor, the game starts in ASCII.

## Build and run

Windows (PowerShell), from the project folder:

```powershell
go build -o termfps.exe .
.\termfps.exe
```

Linux and macOS should work too (`go build -o termfps . && ./termfps`), but
they haven't been tested yet.

## Controls

| Key | Action |
|---|---|
| `W` / `↑` | Move forward |
| `S` / `↓` | Move back |
| `A` / `D` | Strafe left / right |
| `Q` / `←` | Turn left |
| `E` / `→` | Turn right |
| `Space` | Shoot (hold for 3 shots per second) |
| `R` | Reload (the gun also reloads by itself when empty) |
| `M` | Show / hide the minimap |
| `V` | Switch render mode: half-block → shade → ASCII |
| `Esc` / `Ctrl+C` | Quit |

On Windows you can move and turn at the same time.

## Options

| Flag | Default | What it does |
|---|---|---|
| `-fps N` | `30` | Target frames per second (1-240) |
| `-keys async\|events` | `async` on Windows, `events` elsewhere | `async` reads real key state (Windows only, ignored while the window isn't focused). `events` uses terminal key events. In that mode, moving and turning at the same time may not work. |
| `-aspect X` | `1.0` | Pixel height / width (0.5-2.0). Change it if walls look squashed or stretched with your font. |

Example: `.\termfps.exe -fps 60 -aspect 0.9`

## Playing

- **Zombies (green)** hunt you through the map. Each takes 4 hits. Up close
  they hit you for 10 HP per second.
- **Shooters (red)** keep their distance and shoot when they can see you
  (8 damage, less accurate from far away). Each takes 2 hits. Break line of
  sight to stay safe.
- More enemies keep coming, out of your sight. Every 20 kills or 4 minutes
  the **tier** goes up: enemies get tougher, faster and more frequent, and
  more of them are shooters. How long can you last?
- **Ammo boxes** (yellow, also a yellow dot on the minimap) appear every
  20-40 seconds. Walk over one for +25 spare rounds (up to 125).
- The screen edges flash red when you're hurt; enemies flash white when hit.
- At 0 HP the game ends: press Enter to try again or Esc to quit.

## The screen

- **3D view:** walls fade with distance, and north/south faces are drawn
  darker so corners stand out. Ceiling near black, floor brightening toward you.
  Three looks, switched with `V`: half-block pixels (24-bit gray, the default),
  shade blocks (`█▓▒░`), and ASCII (`@%#*+=-:.`).
- **Minimap** (top right): walls, floor, and an arrow showing where you are
  and which way you face.
- **HUD** (bottom line): health, ammo, kills, tier, time survived, facing
  angle and compass direction.
  If the window is wide enough, it also shows FPS, render mode, position,
  frame timings and the input mode.

## Troubleshooting

- **"An Application Control policy has blocked this file" (Windows).** That's
  Windows Smart App Control blocking a freshly built, unsigned program, not a
  bug in the game. Restarting Windows has cleared it; turning Smart App
  Control off fixes it for good. Note that Windows can't turn Smart App
  Control back on without a reinstall.
- **Grays look banded.** The terminal is reducing colors. Use Windows
  Terminal, or another terminal with 24-bit color, or press `V` for ASCII.
- **The terminal looks broken after a crash.** It shouldn't: the game
  restores the terminal on exit and on crashes. If you ever see it, please
  open an issue with the printed error.

## Roadmap

The prototype (P0) is done. Planned next, in order (details in [PRD.md](PRD.md)):

1. **Single-player survival game** (planned in detail, building next): shooting with ammo and reloads, zombies and shooter enemies that get tougher over time, a gun on screen, wall textures, start/pause/settings menus, four maps of different shapes, saved settings and best scores, mouse clicks, and sound.
2. **Server / client split** inside the game, needed for the next two.
3. **You vs an AI agent:** play against Claude Code, Codex, opencode or another CLI agent, in modes like hide and seek or a face-to-face duel.
4. **Multiplayer:** team deathmatch over LAN, then over SSH.

## Development

```powershell
go build ./...; if ($?) { go vet ./... }; if ($?) { gofmt -l . }; if ($?) { go test ./... }
go test -bench . ./internal/render   # DrawFrame and presenter timings
```

Code layout:

| Path | What's there |
|---|---|
| `main.go` | Startup, game loop, terminal restore and crash handling |
| `internal/world` | Map parsing, raycasting (DDA), collision |
| `internal/game` | Player, movement and turning (no terminal code) |
| `internal/input` | Keys to game input, real key state on Windows |
| `internal/render` | Pixel frame, shading, presenters (half-block, shade, ASCII), minimap, HUD |
| `maps/arena.txt` | The map, built into the binary |

Map files use `#` for walls, `.` for floor, and exactly one `P` for the start.
The border must be all walls.

Design notes live in [PRD.md](PRD.md), [ARCHITECTURE.md](ARCHITECTURE.md) and
[DECISIONS.md](DECISIONS.md). Progress is in [TASKS.md](TASKS.md) and
[PROGRESS.md](PROGRESS.md).

Built with [tcell](https://github.com/gdamore/tcell).
