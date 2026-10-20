# Security and privacy

LlamaHerd is a local desktop app. It does not run a public service, and it
does not collect telemetry. This page explains what the control API can do,
what leaves the machine, and how to run fully offline.

## The control API

The GUI, the CLI and the HTTP API are three front ends of one controller. The
API listens only on `127.0.0.1` (default port `8517`, `LLAMAHERD_API_PORT` to
change). It is never bound to a LAN or public interface.

Requests are checked before they reach a handler:

- **Loopback `Host` only.** A request whose `Host` header is not `localhost` or
  a loopback IP is rejected (DNS-rebinding defence).
- **Loopback `Origin` only.** A browser `Origin` that is not loopback is
  rejected (CSRF defence).
- **JSON for mutations.** `POST`/`PUT`/`PATCH`/`DELETE` must use
  `Content-Type: application/json`, which forces a CORS preflight for
  cross-origin pages.
- **Bearer token.** On first setup LlamaHerd generates a random token into the
  user config (`apiToken`, file mode `0600`). The bundled CLI reads it
  automatically. Set `LLAMAHERD_API_TOKEN` or `apiToken` to override it, or
  `"apiToken": "off"` to disable authentication explicitly. The `?token=`
  query form still works for compatibility but headers are preferred because
  query strings leak into shell history and logs.

### What the API can do

With a valid token, a caller can do everything the window can:

- start, stop and restart `llama-server` (including killing a managed process);
- create, edit, reset and delete presets (`configs/<alias>.json`);
- import local `.gguf` files (copy or move into the models directory), rename
  models, move them between categories;
- delete model files from disk (`POST /model/delete`);
- start Hugging Face downloads and cancel them;
- run benchmarks against the loaded server;
- read status, logs, doctor output and benchmark results.

Only the model files the user adds are touchable: import/delete accept a path,
and deletion is limited to models already in the library. There is no arbitrary
file-write endpoint.

If you do not need programmatic access, set `"apiToken": "off"` only on a
single-user machine you trust — otherwise every local process (and any browser
page that can reach loopback) could drive the API. The default token is
stronger; keep it.

The API is also documented in [cli-and-api.md](cli-and-api.md).

## What leaves the machine

LlamaHerd contacts these hosts, and only for the listed purpose:

| Host | When | Data |
| --- | --- | --- |
| `huggingface.co` | loading a repo file list (plus its licence/gated status from `/api/models/<repo>`), downloading a model, saving an HF token | the repo/file you asked for; your HF token only if you configured one; no usage data |
| `raw.githubusercontent.com` | catalog refresh at startup / at most every 10 minutes | a plain GET for `hf/catalog.json`; no query, no identifiers |
| `github.com` (via the user-initiated **Update & rebuild** action) | only when you press it | `git pull`, dependency fetch and a local build in your llama.cpp checkout |
| your `llama-server` | status, metrics, benchmark, chat | prompts you send to your own local server |

There is **no telemetry, analytics, crash reporting or account system**. Model
downloads go directly from Hugging Face to your models directory; LlamaHerd is
not a proxy.

### Running fully offline

- Set `LLAMAHERD_CATALOG_URL=off` (or `"off"`) to disable catalog refresh. The
  catalog embedded in the binary stays available.
- Do not start Hugging Face downloads; use **Import** for local files. Models on
  disk are never checked against Hugging Face, so Start works offline.
- Do not press **Update & rebuild** (it fetches from GitHub).
- The GUI, CLI, API, VRAM guard, presets, benchmarks and local logs all work
  without a network.

The HF token is stored in a separate `0600` file (`docs/configuration.md`
lists the path). It is never written to presets, logs or the README.

## Data stored on disk

| Path | Contents |
| --- | --- |
| user config (`config.json`, `0600`) | paths, API port/token, HF token setting |
| `configs/<alias>.json` | presets (portable, model paths relative to the models dir) |
| `library/<id>.json` | display name, alias, category, sidecars, source, favourites |
| `benchmarks/<id>.json` | measured tok/s, VRAM and config fingerprints |
| `logs/llamaherd.log*` (unless disabled) | app/server output for diagnosis; `0600`, rotated |
| HF token file (`0600`) | the token only |

No personal data is embedded in presets or the repository. The app does not
read files outside the paths you configure, the model you import, and its own
data directories.

## Reporting a vulnerability

Open a private security advisory on the GitHub repository (or contact the
maintainer directly) instead of filing a public issue with exploit details.
Include the version (`llamaherd version`), OS, and a minimal reproduction.
