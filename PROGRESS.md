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
