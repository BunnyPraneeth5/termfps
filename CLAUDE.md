# termfps: agent essentials
(Copy as AGENTS.md if using Antigravity or Codex.)

## What this is
Terminal first-person game. Raycaster, mostly monochrome, half-block pixels by default, shade, ASCII and mixed modes with V. The P0 prototype and P1 (single-player survival game) are built; the "S0ra follow-ups" (S1-Sn) are outlined and the "Maps and gameplay update" ideas are in the plan as M1-M18, order not decided yet. Read PRD.md (Roadmap -> P1) for scope, ARCHITECTURE.md (incl. "P1 design") for design, DECISIONS.md D21-D40 for P1 and later choices, D11 for rendering and D19 for input.

## Stack
Go (latest stable) + tcell v3 (v3.6.0 since S2, D42), pinned version. `github.com/ebitengine/oto/v3` for sound (approved, D27; added in P1-17 with its own dependencies purego and jfreymuth/pulse, D37). No other dependencies. Windows Terminal + PowerShell is the main test target.

## Modules
- internal/world: map parse, collision, raycast (`CastRay`). No tcell.
- internal/game: player, state, update. No tcell.
- internal/input: term key events -> game.Input (held-key tracking, exact once releases arrive).
- internal/render: Frame buffer, DrawFrame, presenters, minimap, HUD. DrawFrame has no tcell; only presenters, minimap, and HUD touch the screen.
- internal/term (S1, D41): our Screen interface, Style/Color, events (Key with Press/Repeat/Release, Mouse, Resize, Focus). tcell.go is the tcell v3 adapter (advanced keys on, so releases arrive where the terminal has them) and the only file that imports tcell; Fake is the test screen. app, render, ui and input use only term.
- Built in P1 (ARCHITECTURE "P1 design"): internal/app (loop + screens), internal/ui (menus), internal/save (save.json), internal/audio (oto), internal/assets (pixel art, textures). world and game never import tcell or audio.

## Rules
- Stay in scope (PRD Roadmap, TASKS.md: P1, the "S0ra follow-ups" S1-Sn, then "Maps and gameplay update" M1-M18 once ordered). If a task needs something out of scope, stop and ask.
- No new dependencies without asking. Approved: tcell v3 (was v2), oto v3.
- Only touch files the current task needs.
- world and game packages must never import tcell.
- Movement and turning use dt (per second), never per frame or per key event.
- Terminal must always be restored on exit and on panic, including panics in the input goroutine.
- The input goroutine only sends events. It never reads or writes game State.
- No silent fallbacks or fake data. If something fails, return or show the error.
- Small changes. One task at a time from TASKS.md. One commit per task.
- Randomness (spawns, pickups, enemy accuracy) goes through an injected `*rand.Rand` so tests are repeatable.
- game emits per-frame events; sound and HUD effects react to them. game never calls audio or render.
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
- Ctrl+C arrives as a key event (`term.KeyCtrlC`). With tcell v3 advanced keys it comes as "c" + Ctrl; the adapter maps it. Handle it, or the game won't quit. A Ctrl+C release must not quit.
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
- Truecolor on Windows (checked in tcell v2.13.10 and v3.6.0 source): truecolor is always on there. `COLORTERM` and `WT_SESSION` are not used on Windows. `TCELL_TRUECOLOR=disable` turns it off. If grays look banded, the terminal itself is downgrading.
- Reuse the Frame between frames. Don't allocate a new one every tick.
- On Windows, Go's `time.Now()` moves in steps of about 0.5ms (measured). A single sub-millisecond timing often reads 0; average over many frames (the HUD does, per FPS window) or use `go test -bench`.
- Windows Smart App Control (on, on the user's machine) sometimes blocks a freshly built `termfps.exe` ("An Application Control policy has blocked this file"). Not a code problem; a restart has cleared it. Rebuild only when code changed. Agents can't run the exe here either. It can block `go test` binaries too (seen in S2 after the tcell v3 / x/sys update); don't try to get around it (e.g. by changing build flags): report it and let the user run the tests or use CI.

## Current focus
P1-01 to P1-13, P1-16, P1-17 done. P1-14 and P1-15 stay [~] until the "S0ra follow-ups" track (S1-S3) is done. S1 and S2 (tcell v3, real key releases) done; S3 (rest of the Mixed style) is next on that track. P1-18 (docs update) waiting for the user to read the README. "Maps and gameplay update" ideas are in TASKS.md as M1-M18 (D40); next: decide their order and the first batch of five. Nothing starts before that.
