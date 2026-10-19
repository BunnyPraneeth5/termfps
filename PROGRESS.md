# PROGRESS: termfps

Append one entry per session. Newest at the bottom.

Template:
```
## YYYY-MM-DD
- Done:
- Not done / blocked:
- Issues found:
- Next:
```

## 2026-10-14
- Done: idea research, stack chosen (Go + tcell), planning files written (PRD, ARCHITECTURE, CLAUDE, TASKS, DECISIONS, PROGRESS).
- Not done / blocked: no code yet.
- Issues found: none yet.
- Next: install Go, set up Windows Terminal, start T1.

## 2026-10-15
- Done: built browser mocks of the P0 screen in three render modes (ASCII, shade blocks, half-block pixels). Had Gemini and ChatGPT build the same mock from one prompt and compared. Gemini's was the most correct (camera-space sprite projection, collision radius). ChatGPT's half-block mode had real bugs. Picked half-block as the default look (D11). Updated PRD, ARCHITECTURE, TASKS, CLAUDE, DECISIONS for the Frame buffer + presenter design.
- Not done / blocked: no code yet. Project instructions in claude.ai still say "mostly monochrome ASCII", update manually.
- Issues found: none in code. Open questions added to ARCHITECTURE (truecolor in Windows Terminal, `▀` row gaps).
- Next: T1 on Saturday.

## 2026-10-18
- Done: reviewed the plan (55 problems) and applied fixes to all docs. Input is now held-key tracking with dt (D12), with spike T8 for real key state via GetAsyncKeyState. Player stores Angle with a written y-down convention (D13). CastRay moved to world to avoid an import cycle (D14). Demo + README are core (D15). Scope trims (D16) and truecolor-based ASCII fallback (D17). TASKS renumbered T1-T20 (the old T4 is split into T4-T6; the old T13 experiment is replaced by T8). Added panic/restore order, map edge cases (BOM, trailing newline), dt clamp, render clamps, and a PowerShell-5.1-safe verify line.
- Not done / blocked: no code yet. Project instructions in claude.ai still need the manual update from 2026-10-15.
- Issues found: tcell truecolor detection on Windows is unverified (believed to be `WT_SESSION`, not `COLORTERM`). Check in T6.
- Next: T1.

## 2026-10-19
- Done: T1 code. `git init` (branch main), `.gitignore`, `.gitattributes` (LF for Go files; global autocrlf is on). Module `github.com/BunnyPraneeth5/termfps`, Go 1.27.1, tcell v2.13.10 pinned (D18). `main.go`: screen opens with "termfps" centered, Esc and Ctrl+C quit, restore via `sync.Once`, panic handler restores first and then prints the stack, `TERMFPS_PANIC=1` forces a panic after 1s, `Init()` errors print and exit 1. Build, vet, gofmt, test all clean.
- Not done / blocked: T1 visual checks need the user to run it in Windows Terminal. Nothing committed yet. Empty `internal/` folders not created (git doesn't track empty folders); each task creates its own.
- Issues found (from reading tcell v2.13.10 source):
  - On Windows, tcell uses its VT screen with built-in `xterm-truecolor`, so truecolor is always on. `COLORTERM`/`WT_SESSION` are not used; `TCELL_TRUECOLOR=disable` turns it off. Docs corrected.
  - Windows Terminal sends key-up events (win32-input-mode) but tcell drops them. Noted as a second option for T8.
  - Ctrl+C arrives as `tcell.KeyCtrlC` in this mode.
- Next: user checks T1 in Windows Terminal, then commit, then T2.
- Update: user confirmed all T1 checks in Windows Terminal (Esc, Ctrl+C, clean prompt, forced panic restores and prints the stack). T1 done and committed.
- Done (Day 1 rest):
  - T2 code: input goroutine (`pumpEvents` in main.go, own recover) + buffered channel drained with select/default, ticker loop with `-fps N` (1-240), dt clamped to 0.1, HUD line with FPS (1-second window), dt and frame work time. `TERMFPS_PANIC=input` tests the goroutine's recover. Tests for clampDT and the FPS counter.
  - T3: `world.Parse` with BOM, CRLF, trailing whitespace and trailing newline handling; rejects empty, <3x3, ragged, unknown chars, `T`, 0 or 2+ spawns, open border. `maps/arena.txt` 16x16, embedded in main.go and parsed before the screen opens. Tests also check every open arena cell is reachable from the spawn.
  - T4: `world.CastRay` (DDA), `At`/`IsWall` (outside = wall). Tests: four straight directions off-center, axis-aligned ray on a grid line (no NaN), no-fisheye across the camera plane, diagonal into a corner and into a pillar, zero direction.
  - T5: `game.Player` with Angle, `Dir()`, `Plane()`; `game.State`, `NewState`. `render.Frame` with `FrameSize` and allocation-free `Resize`, `render.DrawFrame` with flat walls and clamped draw range. Tests: facing a wall 2 away fills rows 16-61 of a 120x78 frame, touching a wall fills everything without panic, zero-height frame.
  - Build, vet, gofmt, test all clean (45 tests). world, game and render don't depend on tcell.
- Not done / blocked: T2's on-screen checks (FPS 28-32, `-fps 15`, idle CPU, `-fps 0`, `TERMFPS_PANIC=input`) need the user; this sandbox blocks running the .exe. T3-T5 not committed yet.
- Issues found: two of my own test mistakes, caught by the first run (fisheye test geometry hit side walls; Resize test skipped FrameSize). Code was correct in both.
- Next: user checks T2, commit Day 1, then T6.
- Update: user confirmed FPS 30.0 (default) and 15.0 (`-fps 15`), frame work about 0.4-0.5ms; `-fps 0` rejected; `TERMFPS_PANIC=input` restores and prints the stack.
- Issue found in that check: a HUD line was printed into the normal terminal before the panic message. The input goroutine restored the terminal while the main loop was still drawing (race). Fixed: the input goroutine now sends its panic to the main loop over a channel, and only the main loop restores and prints. Needs one re-run of the `TERMFPS_PANIC=input` check. Idle CPU not reported yet.
- Update: user re-ran the input panic check (clean, no HUD text) and confirmed idle CPU under 5%. T2 done. Day 1 (T1-T5) complete and committed.
- Next: T6 (half-block presenter, first walls on screen).
- Repo pushed to GitHub by the user (`gh repo create`), remote `origin`.

## 2026-10-19 (Day 2: T6-T9)
- Done (code, tests, one commit per task):
  - T6 `79859ca`: `render.PresentHalfBlock` (top pixel = fg, bottom = bg of `▀`), main loop draws the 3D view, HUD shows draw/present/show times and `screen.Colors()`. Tests on tcell's simulation screen. Benchmarks at 120x40: DrawFrame ~0.03ms with 0 allocations, PresentHalfBlock ~1.1ms (4680 small allocations inside tcell's SetContent; fine for now, revisit in T17 if Show is slow).
  - T7 `f42fa6d`: `input.Tracker` (550ms first window, 100ms repeat window), `input.KeyFor`/`IsQuit`/`Pump` (goroutine moved out of main), `game.Input` + `game.Update` with dt, angle wrap, diagonal normalization. HUD shows x, y, angle. Up/Down arrows also move forward/back (small addition beyond the PRD's W/S).
  - T8 `07d486e`: `-keys async` reads real key state via GetAsyncKeyState (standard `syscall`, Windows-only file). Ignored when tcell reports lost focus; HUD shows mode and focus. Default stays `-keys events`.
  - T9 `4c15114`: `world.TryMove` (per-axis box collision, touching is not a hit), used by `game.Update` with radius 0.2. Tests include 20000 random moves never ending inside a wall.
  - 73 tests pass. world and game have no tcell dependency.
- Not done / blocked: T6-T9 are marked [~]. Their on-screen and feel checks need the user. The T8 keep/drop decision needs the user's hands-on result before it's logged in DECISIONS.md.
- Issues found: one test of mine (frame-rate distance) walked through where a wall is; it started failing once T9 added collision, as it should. Shortened to stay in open space.
- Next: user runs the Day 2 checks; then log the T8 decision; then Day 3 (T10 resize).
- Update: user reported all Day 2 checks passed (walls on screen, movement and turn directions, `-keys async` move+turn together with focus handling, collision). No numbers reported for `colors` or `show` time; T17 measures performance. T6-T9 done. T8 kept as D19: `-keys async` is now the default on Windows, `-keys events` the fallback.
- Next: push, then Day 3 (T10 resize).
- Pushed Day 2 (`d588b6e`).

## 2026-10-19 (Day 3-4: T10-T17)
- Done (code, tests, one commit per task):
  - T10 `e523ca9`: `render.TooSmall`/`DrawTooSmall` (80x24 minimum); main skips update and 3D while too small.
  - T11 `20b0e0a`: `Frame.PixelAspect` + `-aspect 0.5-2.0` flag. Default 1.0, measured from the user's screenshot (cells ~14x28 px, so half-block pixels ~14x14). `wallSpan` pulled out and tested.
  - T12 `f8170d4`: wall fog `230/(1+0.25*dist)`, Y-side x0.7, clamped `toGray`.
  - T13 `06fe179`: ceiling 8, floor 12 at horizon to 80 at the bottom.
  - T14 `1b961ad`: minimap top-right (2 columns per cell, 8-direction arrow), M toggles (one-shot `Input.ToggleMinimap`), on at start.
  - T15 `685dd89`: `render.DrawHUD`: FPS, x/y, degrees + compass letter always; timings and key mode only if they fit whole; "minimap hidden" notice if it doesn't fit. `colors` dropped from the HUD (T6 is done).
  - T17 code part `aeed153`: presenter uses `Put` with a constant string: 0 allocations per frame (was 4680).
  - 93 tests pass. world and game have no tcell dependency.
- Not done / blocked: T10-T15 and T17 marked [~] for the user's on-screen checks; T16 (other terminal) and T17's `Show()` reading need the user's terminal.
- Next: user runs the Day 3-4 checks; then T18 (demo + README).
- Windows Smart App Control (on) blocked the rebuilt exe ("Application Control policy"; Code Integrity events 3077/3033). Confirmed via registry and event log. A restart cleared it. The user kept Smart App Control on.
- User screenshots (Windows Terminal, ~133x33 cells): shading, fog, side faces, floor/ceiling, minimap (arrow cell and direction match x/y/angle at 300° NE and 198° W), HUD all look right. show 0.51-1.13ms, present ~0.6ms, 30 FPS. T12, T13, T14, T15, T17 done.
- Issue found: HUD showed `draw 0.00`. Measured Go's clock on this Windows machine: smallest step 0.53ms, so 0.07ms reads as 0. Fixed: HUD timings are now averaged over each 1-second FPS window (`timeAverager`, tested).
- Still open: T10 (resize by hand), T11 (aspect by eye), T16 (VS Code terminal).
- Update: user confirmed T10 (resize message and recovery) and T11 (proportions look right at the default 1.0). Both done. T16 still open.
- Update: user ran the game in VS Code's terminal and reported it working fine. T16 done. No details given on gray banding or the focus label there. Legacy conhost not tried (optional per D16). Day 3-4 (T10-T17) complete.
- Next: T18 (README + 30-second demo recording).

## 2026-10-19 (T18: README + demo)
- Done: README (build, run, controls, options, screen, troubleshooting incl. Smart App Control, code layout). User recorded a 27-second demo with Snipping Tool (1918x944, 23.9 MB, over GitHub's 10 MB attachment limit for free accounts). Re-encoded with Windows' built-in Media Foundation encoder (WinRT MediaTranscoder from PowerShell, no new tools): 1280x630, 1.5 Mbps, no audio, 4.7 MB. Added as `docs/termfps-demo.mp4` with a still frame as `docs/screenshot.png`, linked at the top of the README. `.gitattributes` marks mp4/png as binary.
- P0 core complete: T1-T18 all done.
- Not done: stretch tasks T19 (render modes) and T20 (targets + shooting).
- Next: decide on stretch tasks, or wrap up P0.
