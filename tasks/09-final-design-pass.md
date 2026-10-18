# 09 — Final design pass

Status: TODO
Depends on: 01–08

## Goal

Make the dummy look/feel right and verify the whole interaction matrix, then
hand it to the user for design review.

## Checklist

- Build clean: `CGO_ENABLED=1 go build -o llamaherd .`
- Run: `DISPLAY=:0 ./llamaherd`
- Zero real I/O: `grep -rn "net/http\|os/exec\|os.Create\|os.WriteFile\|os.ReadFile" *.go ui/ dummy/` returns nothing.
- Interaction matrix (each must work from a cold launch):
  - select each of the 4 model list items
  - switch presets for each alias (incl. `vision+mtp` vs `plain` diff)
  - start/stop cycle (button states + status panel + log)
  - starting state: 2s amber `STARTING`, both buttons disabled mid-transition
  - save/load config (log + path label only)
  - dirty `*` marker on path label appears on edit, clears on preset switch/save
  - mmproj item fills the advanced field
  - one tick timer: uptime increments without any panel creating its own QTimer
- Visual polish:
  - no elided/truncated labels at 1100x720; model paths in tooltips only
  - consistent 8px spacing; group box titles uniform
  - status dot color readable (green `#2e7d32` / red `#c62828` or similar)
  - action buttons roughly equal height; path label right-aligned
- Update `README.md` task table statuses to DONE.
- Ask the user to review the running window; collect design feedback.

## Acceptance

- All checklist items pass.
- User has seen the window and the design is either approved or feedback is
  recorded as new task files (10-…) in this directory.
