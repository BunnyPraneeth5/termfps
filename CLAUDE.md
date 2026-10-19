# termfps: agent essentials
(Copy as AGENTS.md if using Antigravity or Codex.)

## What this is
Terminal first-person game. Raycaster, mostly monochrome, half-block pixels by default (ASCII and shade blocks later). Single player prototype (P0). Read PRD.md for scope, ARCHITECTURE.md for design, DECISIONS.md D11 for the render approach and D12 for input.

## Stack
Go (latest stable) + tcell v2, pinned version. No other dependencies. Windows Terminal + PowerShell is the main test target.

## Modules
- internal/world: map parse, collision, raycast (`CastRay`). No tcell.
- internal/game: player, state, update. No tcell.
- internal/input: tcell events -> game.Input (held-key tracking).
- internal/render: Frame buffer, DrawFrame, presenters, minimap, HUD. DrawFrame has no tcell; only presenters, minimap, and HUD touch the screen.

## Rules
- Stay in P0 scope. If a task needs something out of scope, stop and ask.
- No new dependencies without asking.
- Only touch files the current task needs.
- world and game packages must never import tcell.
- Movement and turning use dt (per second), never per frame or per key event.
- Terminal must always be restored on exit and on panic, including panics in the input goroutine.
- The input goroutine only sends events. It never reads or writes game State.
- No silent fallbacks or fake data. If something fails, return or show the error.
- Small changes. One task at a time from TASKS.md.
- I'm new to Go: add a short comment when using a Go concept a beginner might not know (goroutines, channels, select, embed, interfaces, sync.Once, build-tagged files).

## Verify before saying done
Works in both Windows PowerShell 5.1 and PowerShell 7:
```
go build ./...; if ($?) { go vet ./... }; if ($?) { gofmt -l . }; if ($?) { go test ./... }
```
gofmt -l must print nothing. Then run the game.
Agents report build, vet, gofmt and test results only. Visual acceptance checks (anything you have to look at on screen) are signed off by the user. Don't mark a task done until the user confirms them.

## End of session
Append an entry to PROGRESS.md. Log any new decision in DECISIONS.md.

## Gotchas
- Ctrl+C arrives as a tcell key event (`tcell.KeyCtrlC`). Handle it, or the game won't quit.
- Restore order on panic: call `restore` (tcell `Fini`) first, then print the panic and stack. Printing first writes into the alternate screen and the message is lost.
- Never `os.Exit` or `log.Fatal` after screen `Init()` except right after `restore`. They skip deferred functions, so the terminal stays broken.
- `recover()` only works for panics on the same goroutine. The input goroutine needs its own, and it hands the panic to the main loop over a channel instead of restoring the terminal itself. Only the main goroutine touches the screen; otherwise the main loop can draw after restore and leave HUD text in the user's terminal (seen in T2).
- `PollEvent()` returns nil after `Fini`. The input goroutine must exit on nil.
- Map files may have `\r\n` line endings, a UTF-8 BOM, and trailing newlines on Windows.
- Go files must have LF line endings, or `gofmt -l` lists every file. `.gitattributes` handles this (T1).
- Use perpendicular distance, not euclidean, or walls curve (fisheye).
- The Frame is W x 2*(rows-1) (the last row is the HUD), so Frame pixels are roughly square. Use `pixelAspect` (~1.0) in wall height. Don't apply the 2x terminal cell aspect again on top.
- Clamp drawStart/drawEnd to the Frame. Up close, wall height can be huge.
- Half-block: `▀` foreground = top pixel, background = bottom pixel. Getting these swapped flips every cell upside down.
- Clamp float brightness to 0-255 before converting to `uint8`. Go gives a garbage value (no panic) for out-of-range conversions.
- y points down: angle 0 = east, 90° = south, Right arrow increases the angle. See ARCHITECTURE "Coordinates".
- Compare rune keys lower-cased, so Shift and Caps Lock don't break W/A/S/D/Q/E/M.
- Truecolor on Windows (checked in tcell v2.13.10 source): tcell uses its VT screen with built-in `xterm-truecolor`, so truecolor is always on. `COLORTERM` and `WT_SESSION` are not used. `TCELL_TRUECOLOR=disable` turns it off. If grays look banded, the terminal itself is downgrading.
- Reuse the Frame between frames. Don't allocate a new one every tick.

## Current focus
Day 3-4 (T10-T17) code committed; waiting for the user's on-screen checks, the T16 other-terminal check, and T17's Show() time. Next: T18 (demo recording + README).
