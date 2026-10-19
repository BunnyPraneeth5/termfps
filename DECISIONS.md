# DECISIONS: termfps

Format: what, why, trade-off, revisit when.

## D1: Go over Rust (2026-10-14)
- Why: I'm new to both. Go is faster to learn, compiles fast on 8GB RAM, and has the best SSH game ecosystem (Charm wish, gliderlabs/ssh). Existing SSH terminal games mostly use Go.
- Trade-off: lose Rust's raw speed and crossterm's key-release support on Windows.
- Revisit when: Go can't hold 30 FPS, or input feel needs true key state.

## D2: tcell over raw ANSI or Bubble Tea (2026-10-14)
- Why: tcell gives direct cell control, key events, resize events, and only redraws changed cells. Bubble Tea is built for UI apps, not a 30 FPS full-screen renderer.
- Trade-off: no key-release events. (Correction 2026-10-18: the Win32 console API does report key-up; tcell drops it. Real key state is possible on Windows without leaving tcell, see D12.)
- Revisit when: input feel is not good enough after T8.

## D3: Raycasting over full 3D (2026-10-14)
- Why: simple, fast, well documented (Lodev tutorial), looks good in ASCII. Fits a small TDM map.
- Trade-off: flat floors, no height levels, no looking up or down.
- Revisit when: never for this game. It's the style.

## D4: Monochrome ASCII, rare color (2026-10-14)
- Status: superseded by D11 (2026-10-15). The "mostly monochrome, rare color" part still holds. ASCII is no longer the default.
- Why: works on every terminal, lighter bandwidth for future SSH play, gives the game its own look. Color reserved for important things.
- Trade-off: depth has to come from character shading.
- Revisit when: playtesters can't read the scene.

## D5: Native Windows for the prototype (2026-10-14)
- Why: simpler setup, Windows Terminal is my daily terminal, and the main test target.
- Trade-off: future server will run on Linux, so Linux testing comes later.
- Revisit when: starting multiplayer (P2/P3).

## D6: Per-event input first (2026-10-14)
- Status: superseded by D12 (2026-10-18). Per-event steps break the "movement uses dt" rule and stutter.
- Why: tcell has no key release. Per-event steps work with OS key repeat and need no extra code.
- Trade-off: short delay before key repeat kicks in.
- Revisit when: (superseded by D12; the old T13 experiment no longer exists).

## D7: Maps as text files embedded in the binary (2026-10-14)
- Why: easy to edit by hand, single binary to share.
- Trade-off: rebuild to change a map.
- Revisit when: adding multiple maps or a map editor.

## D8: Game logic separate from rendering (2026-10-14)
- Why: world and game don't import tcell, so the future multiplayer server can reuse them as-is.
- Trade-off: slightly more structure than a quick hack needs.
- Revisit when: never. This is the foundation for P2.

## D9: Agent arena deferred (2026-10-14)
- Why: with LLM players it becomes a benchmark more than a game. Prototype first, decide later.
- Trade-off: none right now.
- Revisit when: multiplayer works.

## D10: Working title "termfps" (2026-10-14)
- Why: name and theme not decided yet.
- Revisit when: before any public post.

## D11: Half-block pixels by default, selectable render modes (2026-10-15)
- What: the raycaster writes into a pixel buffer (`Frame`, width x 2*rows, grayscale). Presenters turn the Frame into terminal cells. Default presenter is half-block: each cell is `▀` with the top pixel as foreground and the bottom pixel as background, in 24-bit gray. Shade blocks (`█▓▒░`) and ASCII are selectable later with V.
- Still mostly monochrome: walls, floor, and ceiling are grayscale. Color stays reserved for important things (targets now, players later).
- Scope: half-block presenter is P0 core. Shade and ASCII presenters plus the V toggle are the first stretch task (T19).
- Why: half-block doubles vertical resolution and gives smooth fog and shading. Browser mocks of all three modes (mine, plus Claude, Gemini, and ChatGPT versions) made half-block clearly the best looking. Keeping a Frame buffer means ASCII stays cheap to add, which keeps the SSH path open.
- Trade-off: two colors per cell means more bytes per frame than ASCII. Heavier over SSH. Some terminals and fonts render `▀` with gaps or poor color. Render code has one more layer.
- User caution (shown in docs and when switching modes): "Half-block needs a truecolor terminal and more bandwidth. Use ASCII over slow SSH."
- Revisit when: T16 shows terminals outside Windows Terminal can't handle it, T17 misses the performance budget, or SSH play starts (P3).

## D12: Held-key input with dt, real key state as a spike (2026-10-18)
- What: the input package tracks held keys with an adaptive timeout (about 550ms after the first press, about 100ms once repeats arrive) and gives game one `game.Input` per frame. Update moves `speed * dt` for each held key. T8 spikes Win32 `GetAsyncKeyState` through the standard `syscall` package in a Windows-only file.
- Why: per-event steps (D6) tie speed to the OS repeat rate, not dt, and stutter against the frame rate. A plain short window doesn't cover Windows' ~500ms repeat delay.
- Trade-off: a quick tap moves for up to about half a second. Windows only repeats the last key pressed, so moving and turning at the same time doesn't work with the baseline. GetAsyncKeyState fixes that but reads keys even when another window has focus, and is Windows-only.
- Revisit when: T8 results are in (log them here as a new decision), or when SSH play starts (no key state over SSH).
- Update 2026-10-19: T8 results are in, see D19. This tracker is now the fallback (`-keys events`).

## D13: Player stores an angle (2026-10-18)
- What: `Player` has `Angle` (radians). `Dir()` and `Plane()` are computed from it every frame. y points down, angle 0 = east, 90° = south, Right arrow increases the angle. Spawn faces east.
- Why: rotating Dir/Plane vectors over and over drifts (plane length and the 360° view slowly change). The HUD needs an angle anyway. One written convention stops reversed turn, strafe, and minimap arrow bugs.
- Trade-off: a sin and cos per frame. Nothing measurable.
- Revisit when: never for P0.

## D14: CastRay lives in world, not render (2026-10-18)
- What: `world.CastRay` and `world.Hit`. render's DrawFrame calls it.
- Why: CastRay only needs the map. Shooting (T20) needs it inside `game.Update`. In render it would make an import cycle (game -> render -> game), which Go refuses to compile. It also keeps raycasting available to a future server.
- Trade-off: world is slightly bigger.
- Revisit when: never.

## D15: Demo recording and README are core, not stretch (2026-10-18)
- What: T18 (demo + README) runs before any stretch task.
- Why: the 30-second recording is a PRD success criterion. It was gated behind stretch work, so P0 could "finish" without it.
- Trade-off: stretch tasks get less of day 5.
- Revisit when: never for P0.

## D16: P0 scope trims (2026-10-18)
- What:
  - Legacy conhost testing is a 10-minute optional look in T16, no requirements. The required check is Windows Terminal plus one other terminal.
  - The per-event vs held-window experiment (old T13) is replaced by D12 and spike T8.
  - Stretch-only fields (`RenderMode`, `Pixel.Color`, a shoot action) are added by their stretch tasks, not in P0 structs.
  - The big raycaster task is split into T4 (CastRay), T5 (DrawFrame), T6 (presenter). The old "correct proportions" task becomes T11 "tune pixelAspect", since perpendicular distance and the wall-height formula are in T4/T5.
  - Kept the input package: with held-key tracking it has real, testable logic.
- Why: 4-5 days, new to Go. PRD success criteria only name Windows Terminal.
- Trade-off: no data on how legacy conhost performs.
- Revisit when: someone reports it's unplayable in conhost.

## D17: Stretch ASCII fallback triggers on "no truecolor", not "< 256 colors" (2026-10-18)
- What: T19 starts in ASCII when the terminal has no truecolor, using whatever the pinned tcell version reports (found in T6).
- Why: the user caution says half-block needs truecolor. The old 256-color threshold contradicted it.
- Trade-off: 256-color terminals start in ASCII even though half-block would look okay-ish there. V still switches.
- Revisit when: T16 shows a common terminal that looks fine with 256 colors.

## D18: Module path and tcell version (2026-10-19)
- What: module `github.com/BunnyPraneeth5/termfps`. tcell pinned at v2.13.10 (latest v2 on this date).
- Why: the module path goes into every import, so it's set to the planned GitHub path now. Pinning keeps tcell's behavior (truecolor default, key handling) the same as what was checked in its source.
- Trade-off: no automatic tcell fixes. Upgrades are a deliberate step.
- Revisit when: a tcell bug blocks a task, or before publishing.

## D19: Keep real key state (T8 spike), default on Windows (2026-10-19)
- What: `-keys async` (Win32 `GetAsyncKeyState` via standard `syscall`) is kept and is the default on Windows. `-keys events` (the D12 held-window tracker) stays as the fallback and the default on other systems.
- Why: the user's T8 check passed: moving and turning at the same time works, movement stops on release, and keys typed in another window are ignored (tcell focus events work in Windows Terminal; the HUD showed "NOT focused"). That removes the main input limit of D12.
- Trade-off: Windows-only code path. If a terminal doesn't send focus events, keys from other windows would move the player; the HUD's focus label makes that visible. Won't work over SSH (P3), where the events tracker is the only option.
- Revisit when: SSH play starts, or a terminal shows focus problems.
