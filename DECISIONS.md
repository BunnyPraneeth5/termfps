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
- Revisit when: starting the server split (P2) or multiplayer (P4), per the D20 roadmap.

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
- Status: superseded by D20 (2026-10-19). AI-agent opponents are now phase P3, before multiplayer, as a player-vs-agent game rather than a benchmark.
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
- Revisit when: T16 shows terminals outside Windows Terminal can't handle it, T17 misses the performance budget, or SSH play starts (P4 in the D20 roadmap).

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
- Trade-off: Windows-only code path. If a terminal doesn't send focus events, keys from other windows would move the player; the HUD's focus label makes that visible. Won't work over SSH (P4 in the D20 roadmap), where the events tracker is the only option.
- Revisit when: SSH play starts, or a terminal shows focus problems.

## D20: Roadmap after P0 (2026-10-19)
- What: four phases after P0, each planned (scope, design, tasks) before any code. Details in PRD "Roadmap".
  - P1, single-player game: enemies/bots that move, sprites, weapon graphics, wall textures, menus, multiple maps, saving. Sound and mouse input are undecided (sound leaning toward necessary); decide when planning P1. P0's T20 (targets + shooting) is the bridge into P1.
  - P2, server / client split: game logic as a server, the screen as a client.
  - P3, AI-agent opponents: the player vs a CLI AI agent (Claude Code, Codex, opencode, ...), likely through an MCP server with game-only tools. Several game modes (ideas: hide and seek, face-to-face duel), decided when building P3.
  - P4, multiplayer: team deathmatch over LAN, then SSH.
- Why: the user's direction after P0. A real single-player game first makes the AI and multiplayer phases worth playing. AI opponents come before multiplayer because they're the project's unique hook and don't need networking, only the P2 split. This supersedes D9.
- Trade-off: P1 grows the scope a lot (several weekends, not one). Sound would need a new dependency and only works locally. AI opponents bring open problems (decision speed vs 30 FPS, token cost, fairness, safety) that P3 must solve.
- Revisit when: planning each phase.

## D21: P1 is an endless survival game (2026-10-20)
- What: each map is an arena; enemies keep spawning out of sight; difficulty tiers rise every 20 kills or 4 minutes, whichever comes first (more enemy HP, damage, speed, spawn rate). Score = kills, time survived, tier. No "clear the level" mode.
- Why: the user wants a complete single-player FPS with rising pressure rather than hand-built levels; random spawns and pickups fit it, and one mode keeps P1 smaller.
- Trade-off: no story or level progression; the difficulty curve depends on balancing (P1-18).
- Revisit when: balancing shows it gets boring, or a level mode is wanted.

## D22: Limited ammo with magazines and random pickups (2026-10-20)
- What: magazine 25; start with 25 + 50 spare; R reloads (~1.5 s); auto-reload when the magazine is empty; ammo pickups (+25) at random free cells every ~20-40 s, at most 3 on the map. Hitscan shots, ~3 per second.
- Why: the user's choice: resource pressure and reloading make fights tense.
- Trade-off: more state, HUD and balancing work; running dry must stay rare enough to be fun.
- Revisit when: balancing (P1-18).

## D23: Two enemy types, colored (2026-10-20)
- What: zombies (green, melee, rush, tanky) and shooters (red, ranged hitscan, keep distance, need line of sight, fragile). Muzzle flash is yellow. Everything else stays gray.
- Why: the user's choice; two clearly different threats; color only for important things (PRD Style).
- Trade-off: two AI behaviors to build and balance. Colors in shade/ASCII mode use basic terminal colors.
- Revisit when: adding more enemy types.

## D24: Esc pauses; Ctrl+C quits (2026-10-20)
- What: Esc during play opens the pause menu (Resume, Settings, Restart, Quit to menu). Quitting happens from menus. Ctrl+C still quits instantly from anywhere.
- Why: with menus, an instant-quit Esc loses a run by accident.
- Trade-off: one more key press to quit.
- Revisit when: never for P1.

## D25: Save settings and progress, no mid-game saves (2026-10-20)
- What: `%AppData%\termfps\save.json` via `os.UserConfigDir` and `encoding/json` (standard library): settings, unlocked maps, best kills/time/tier per map, total kills. Atomic write; a damaged file is reported, kept as `save.json.bad`, and replaced with defaults.
- Why: survival runs are short, so only bests and settings matter. Settings are saved so they don't need resetting each launch.
- Trade-off: a run can't be resumed after quitting.
- Revisit when: runs get long enough that people want to resume.

## D26: Four maps of different shapes, unlocked by survival (2026-10-20)
- What: 16x16, 32x32, 64x32, 16x64 arenas embedded from `maps/` (directory embed); optional `; name:` header; the first map is open, the next unlocks after surviving 5 minutes on the previous. The minimap scrolls on maps bigger than the screen allows.
- Why: different shapes (open, big, long corridor) change how fights play; unlocking gives a goal across runs.
- Trade-off: the minimap needs a scrolling window; bigger maps mean longer rays (still cheap).
- Revisit when: adding more maps or a map editor.

## D27: Sound through the oto library, effects generated in code (2026-10-20)
- What: `github.com/ebitengine/oto/v3` (new dependency, approved by the user) behind `internal/audio`. Effects are synthesized in code (no sound files). Sound on/off and volume settings. If audio can't start, the game runs silently and the menu says why.
- Why: terminals can't play sound, but the game process can play through the speakers. A shooter needs overlapping sounds, which Windows' built-in PlaySound can't do. oto is pure Go on Windows and macOS. Generated effects avoid asset and copyright questions and fit the retro style.
- Trade-off: a second dependency. On Linux oto needs cgo and ALSA development libraries. Sound plays only where the game runs, so SSH players (P4) won't hear it.
- Revisit when: SSH play (P4), or if oto causes build problems.

## D28: Mouse clicks only (2026-10-20)
- What: menus: click an item to select it. In game: left-click shoots, right-click reloads. No mouse-look.
- Why: terminals report clicks and the cell under the pointer but can't lock the mouse, so mouse-look would feel clumsy.
- Trade-off: turning stays on the keyboard.
- Revisit when: a terminal offers relative mouse movement.

## D29: T20 moves into P1 (2026-10-20)
- What: P0's last stretch task (red targets + shooting) becomes P1-02 (color in the renderer, `T` blocks) and part of P1-03 (shooting). P0 closes with T1-T19 done.
- Why: shooting is now a full feature (ammo, reload, enemies), so it belongs in the P1 plan, not as a small P0 extra.
- Trade-off: none.
- Revisit when: never.

## D30: Early balancing of the start (2026-10-21)
- What: after playing P1-06 to P1-10, the user found the start too hard. Changes, in the user's words where given:
  - Zombie: 75 HP (3 hits), 1.5 cells/s, a random 7-10 damage per hit, once per second (user's numbers). Tiers still scale these.
  - Shooter: unchanged (user: "everything is good").
  - Arena: only 1 zombie placed at the start (was 2 zombies + 2 shooters).
  - Tier 1: first spawn after 10 s (was 3), then every 8 s (was 6), at most 3 alive (was 5). The curve still reaches spawns every 2 s and 14 alive.
  - Health regeneration: +5 HP per second after 4 s without being hit, up to 100.
- Why: about 23 damage per second could reach the player within ~15 s of starting with no way to recover. Now the start is gentle and the tiers do the ramping.
- Trade-off: a careful player can heal between fights indefinitely; tiers must keep the pressure up. Not yet tuned by playtesting.
- Revisit when: balancing (P1-18), after playtesting tiers 3+.

## D31: S0ra feedback tasks join P1 (2026-10-25)
- What: two tasks from S0ra's feedback (YouTube + Discord, 2026-10-23) go in before Extras: P1-14 smoother events-mode keys, P1-15 a "Mixed" render mode. Extras renumbered to P1-16 (mouse), P1-17 (sound), P1-18 (balancing).
- Why: on Linux (events mode) a single tap turned ~99° and moved ~1.65 cells, because a key counts as held for FirstWindow (550ms) after its first event; Linux repeat delays of 600-660ms also cause a stop just before repeats begin. The Mixed mode (ASCII character, fg = color, bg = color x 0.2) costs about the same as half-block, which already sends both colors per cell.
- Trade-off: true key releases would fix input properly, but tcell v2.13.10 asks for kitty keyboard level 1 only and drops win32-input-mode key-ups. That needs a tcell change, so it waits for Linux/mac support.
- Credit: S0ra (libasciic, MIT) for the Mixed style idea and the input report.
- Revisit when: Linux/mac support starts (key releases).

## D32: GitHub Actions CI on Windows and Linux (2026-10-25)
- What: `.github/workflows/ci.yml` runs build, vet, gofmt and test on every push to main and every pull request, on windows-latest and ubuntu-latest. No macOS for now (user's choice). No releases or packaging.
- Why: the game is only built on Windows here, but people run it on Linux (S0ra). Files like `keystate_other.go` never compile on Windows, so a break there would go unnoticed. CI is free for public repos and adds no Go dependency.
- Trade-off: it only checks what the verify command checks; visual checks still need the user. From P1-17 (oto), the Linux job will need the ALSA development libraries installed.
- Revisit when: P1-17 (add ALSA to the Linux job), or when Linux/mac support starts (add macOS).

## D33: Menu details (2026-10-25)
- What: the game opens on the title menu. Esc on the title menu does nothing (Quit or Ctrl+C quits); Esc on game over goes to the title menu. The game over screen is a menu (Try again, Main menu). Sound and volume settings are in the menu now and stored, with a note that sound isn't in yet (P1-17). FPS changes from the menu reset the frame ticker. Mouse clicks are turned on for the whole run (button events only, no motion).
- Why: Esc already means "back" everywhere else, so it shouldn't quit by accident on the first screen. Showing the sound settings now keeps the settings list stable for P1-13 saving.
- Trade-off: while the game runs, the terminal's own mouse text selection doesn't work (most terminals allow Shift+drag to select anyway).
- Revisit when: P1-16 (mouse in game), or if players miss text selection.

## D34: Save file details (2026-10-25)
- What: saved settings replace the flag defaults, but a flag given on the command line (`-fps`, `-keys`, `-aspect`, `-textures`) wins for that launch, and the next save stores it. A run counts (bests, total kills) when the player dies, restarts, quits to the menu, or presses Ctrl+C mid-run; each run counts once. Bests are per field (best kills, best time and best tier can come from different runs). Values the game can't use (unknown render mode, FPS out of range, wrong version) make the file "damaged": kept as `save.json.bad`, defaults used, the menu says why. A file that exists but can't be read is left alone (it may only be locked), with a message. If the saved key mode is async on a system without it, events is used and the menu says so.
- Why: flags are for testing a launch without editing the save; a run quit early is still a run the player played.
- Trade-off: a mistyped flag value gets saved if a setting is then changed in the menu.
- Revisit when: settings gain values that need migrating (bump `save.Version`).

## D35: Events-mode tap handling (2026-10-25)
- What: the Tracker measures the OS repeat delay (first press to first repeat, confirmed by a quick third event) and uses it plus 100ms as the first window; 550ms until measured. While a key has no repeats yet it might be a tap, so turning runs at 25% speed and moving at 40%; once repeats arrive the slowdown fades out over 150ms. A tap now turns about 25-34 degrees and moves about 0.7 cells (was ~99 degrees and 1.65 cells). Async mode is unchanged.
- Why: S0ra's report (D31). Without key releases, a tap and the start of a hold look the same for the length of the repeat delay; slowing that stretch makes taps small without cutting holds short.
- Trade-off: every hold starts slowly for the repeat delay (about 0.5-0.66 s). The first hold of a session may still stop briefly on systems whose repeat delay is over 550ms.
- Revisit when: real key releases are available (kitty keyboard protocol through tcell).

## D36: Mixed render mode (2026-10-25)
- What: a fourth render mode, after ASCII in the V cycle: the ASCII ramp character (from the cell's average gray, as in ASCII mode) in the cell's color, on a background of that color at 20% brightness. Tints carry over. Needs truecolor; without it the game still starts in ASCII. Idea: S0ra's libasciic "Mixed" style (D31), credited in the README.
- Why: it fills the dark gaps of plain ASCII and gives a textured look, at about half-block's cost (both send two colors per cell; filling tcell's buffer measured 0.58 ms vs 0.88 ms at 120x40).
- Trade-off: half the vertical resolution of half-block. The optional color rounding (fewer color codes for neighbouring cells) is not built: it would only matter on slow terminals, and needs measuring there first. The half-block caution isn't shown for mixed.
- Revisit when: someone reports mixed or half-block stuttering on a slow terminal (try the rounding then).

## D37: Sound details (2026-10-25)
- What: `internal/audio` on oto v3.5.1. Eight effects generated at startup (mono, 44.1 kHz, 16-bit): gunshot (shot fired), reload (reload started: click, click, clack over ~1.1 s), empty click, zombie groan (an enemy spawned; there is no zombie-only event, and it works as an "enemy arrived" cue), enemy shot, hurt, pickup, tier up. Each effect plays at most once per frame; at most 16 at once. Audio is opened before the screen (audio libraries may print warnings); if it fails, the game runs silently and every menu shows "Sound unavailable: <reason>". The Sound and Volume settings now work.
- Why: D27. Mapping sounds to events keeps `game` free of audio.
- Dependencies: oto v3.5.1 brings `github.com/ebitengine/purego`, `github.com/jfreymuth/pulse` (pure-Go PulseAudio) and a newer `golang.org/x/sys`. On Linux, oto now loads ALSA/PulseAudio at runtime through purego: no cgo and no ALSA development libraries at build time. This replaces the Linux note in D27 and the CI note in D32 (the Linux CI job needs no extra step). Linux players need ALSA or PulseAudio installed to hear sound.
- Trade-off: the groan plays for shooter spawns too.
- Revisit when: a zombie-only cue is wanted (add a game event), or effects need tuning by ear.

## D38: P1-18 becomes docs only; "Maps and gameplay update" next (2026-10-25)
- What: the user marked P1-01 to P1-13, P1-16 and P1-17 done. P1-14 and P1-15 stay [~] because S0ra's suggestions are only partly done (no true key releases; Mixed without color-threshold skipping or an adjustable background). P1-18 is now a docs update only (README, ARCHITECTURE, PRD). Balancing and a new demo move to a new "Maps and gameplay update" section, numbered M1 to Mn, which gets planned in a later discussion.
- Why: the user's choice; the next work is about making maps and gameplay feel better and more realistic, which needs discussing first.
- Trade-off: balancing past tier 2 stays untested until then.
- Revisit when: the M1-Mn tasks are written.

## D39: S0ra follow-ups are their own track (2026-10-25)
- What: the S0ra follow-ups move out of "Maps and gameplay update" into a separate TASKS section, S1 to Sn. Outline: S1 our own terminal layer (`internal/term`, tcell v2 adapter, fake screen for tests), S2 real key releases through tcell v3.4+ (spike first; dependency change needs the user's OK), S3 the rest of the libasciic Mixed style (color threshold, adjustable background). P1-14 and P1-15 stay [~] until they're done.
- Why: the user's choice: finishing S0ra's suggestions is different work from making maps and gameplay better. Research: tcell v3.4.0 (May 2026) added key release reporting, so no fork is needed; v3 also removes the public simulation screen our tests use, which is why S1 comes first. crossterm confirms releases on Unix need the kitty protocol's event-type flags. libasciic's Mixed: max(r,g,b) brightness, background factor 0-1 (default 0.2), and a per-row color threshold that skips repeating similar colors.
- Trade-off: S1 is a refactor with no visible change; it pays off when S2 or our own input reader lands.
- Revisit when: S2's spike shows whether tcell v3 is enough.

## D40: Maps and gameplay ideas go into the plan as M1-M18 (2026-10-26)
- What: every idea from the discussion is in TASKS.md as M1-M18. The user's input so far: the minimap window shrinks to at most 16x16 cells (a 16x16 map keeps its size) and gets a border (M1); all four maps get redesigned from scratch for consistency (M5); new maps are welcome (M6); medkits and armor (M10), more weapons (M11) and more enemy types (M12) are wanted. The unlock rule is open (M7).
- Why: the user wants everything in the plan first, then to decide order and details.
- Trade-off: the numbers are IDs, not the order, so the order has to be written down separately when it's decided.
- Revisit when: the first batch of five is chosen.

## D41: Our own terminal layer, internal/term (2026-10-26)
- What: S1. `internal/term` has our `Screen` interface (Size, Put, Clear, Show, Sync, Truecolor, EnableMouse, EnableFocus, PollEvent), our `Style` (Fg, Bg, Bold, Dim, Reverse) and `Color` (default, basic red/green/yellow, 24-bit RGB) as plain comparable values, a `PutStr` helper (one character per cell, clipped at both edges), and our events: `Key` (Code, Rune, Mod, Action Press/Repeat/Release, When), `Mouse` (X, Y, Buttons left/right/middle), `Resize`, `Focus`. `term.Tcell` (tcell.go) is the tcell v2 adapter and the only code that imports tcell; `term.Fake` is the in-memory screen every app, render, ui and input test now uses. `internal/input` moved too (its key helpers took tcell events), and main opens the screen with `term.Open`. Fini is on the two concrete types, not in the interface (only main's restore calls it).
- Why: tcell v3 drops the simulation screen our tests used (D39); with this layer S2 (tcell v3, real key releases) or our own input reader only changes the adapter. Keys carry Press/Repeat/Release now so S2 doesn't have to change the event types.
- Details: tcell v2 can't tell a repeat from a press and never reports a release, so the v2 adapter marks every key Press. The app skips Release keys for now (S2 feeds them to the tracker). tcell events the game never used (paste, interrupts, errors) are skipped in the adapter, as the app ignored them before. tcell drops Shift from upper-case letters; our Mod keeps whatever tcell reports, and Meta counts as Alt. Benchmarks at 120x40: PresentHalfBlock on the Fake ~0.09 ms, filling tcell's buffer through the adapter ~0.85 ms with 0 allocations, about the same total as before (0.75-1 ms).
- Trade-off: one more layer and a style conversion per cell (no measurable cost). PutStr assumes no wide or combining characters, which the game's text doesn't have.
- Revisit when: S2 (the v3 adapter), or text with wide characters is needed.
