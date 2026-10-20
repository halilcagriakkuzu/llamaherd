# UI screenshot and smoke-test harness

`frontend/dist` is dependency-free, so it can be rendered in a plain Chromium
with a mocked Wails backend (`mock.js`).

## Screenshots

```bash
scripts/uishot/shoot.sh <outdir> [light|dark]
```

Needs a Chromium. Playwright's cache is auto-detected; otherwise set `CHROME`:

```bash
CHROME=/usr/bin/google-chrome scripts/uishot/shoot.sh /tmp/ui light
```

## Smoke test

```bash
scripts/uishot/smoke.sh
```

Boots the real frontend twice and asserts visible state (no pixel comparison):

1. **running** — model list, benchmark table (sorted), current row, import
   dialog opens and the mmproj picker appears.
2. **failed** — the "not responding" banner, its log tail, Retry/Stop buttons.

The page-side assertions live at the bottom of `mock.js` (`?selftest=main` and
`?selftest=failed`). Chromium's `--dump-dom` output is grepped for
`SELFTEST PASS`; a removed element id or changed label fails the run.

CI installs Chromium with Playwright (`npx playwright install --with-deps
chromium`) and runs the same script. The smoke test is Linux-only.
