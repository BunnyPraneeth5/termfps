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
> text. An ASCII mode for slow connections (like SSH) is planned.

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
| `M` | Show / hide the minimap |
| `Esc` / `Ctrl+C` | Quit |

On Windows you can move and turn at the same time.

## Options

| Flag | Default | What it does |
|---|---|---|
| `-fps N` | `30` | Target frames per second (1-240) |
| `-keys async\|events` | `async` on Windows, `events` elsewhere | `async` reads real key state (Windows only, ignored while the window isn't focused). `events` uses terminal key events. In that mode, moving and turning at the same time may not work. |
| `-aspect X` | `1.0` | Pixel height / width (0.5-2.0). Change it if walls look squashed or stretched with your font. |

Example: `.\termfps.exe -fps 60 -aspect 0.9`

## The screen

- **3D view:** walls fade with distance, and north/south faces are drawn
  darker so corners stand out. Ceiling near black, floor brightening toward you.
- **Minimap** (top right): walls, floor, and an arrow showing where you are
  and which way you face.
- **HUD** (bottom line): FPS, position, facing angle and compass direction.
  If the window is wide enough, it also shows frame timings and the input mode.

## Troubleshooting

- **"An Application Control policy has blocked this file" (Windows).** That's
  Windows Smart App Control blocking a freshly built, unsigned program, not a
  bug in the game. Restarting Windows has cleared it; turning Smart App
  Control off fixes it for good. Note that Windows can't turn Smart App
  Control back on without a reinstall.
- **Grays look banded.** The terminal is reducing colors. Use Windows
  Terminal, or another terminal with 24-bit color.
- **The terminal looks broken after a crash.** It shouldn't: the game
  restores the terminal on exit and on crashes. If you ever see it, please
  open an issue with the printed error.

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
| `internal/render` | Pixel frame, shading, half-block presenter, minimap, HUD |
| `maps/arena.txt` | The map, built into the binary |

Map files use `#` for walls, `.` for floor, and exactly one `P` for the start.
The border must be all walls.

Design notes live in [PRD.md](PRD.md), [ARCHITECTURE.md](ARCHITECTURE.md) and
[DECISIONS.md](DECISIONS.md). Progress is in [TASKS.md](TASKS.md) and
[PROGRESS.md](PROGRESS.md).

Built with [tcell](https://github.com/gdamore/tcell).
