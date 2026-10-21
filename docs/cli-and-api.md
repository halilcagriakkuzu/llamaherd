# CLI and control API

The GUI, the CLI and the HTTP API are three front ends of one controller, so they always
agree. The API listens on `127.0.0.1:8517` only (`LLAMAHERD_API_PORT` to change). Requests
must address a loopback `Host`, browser requests must come from a loopback `Origin`, and
mutating requests must use `Content-Type: application/json` (CSRF / DNS-rebinding guards).
An access token is generated into the user config on first setup; set
`LLAMAHERD_API_TOKEN` (or `apiToken`) to override it, or `"apiToken": "off"` to disable it.
The bundled CLI sends the token for you.

## CLI

If nothing is listening, the first command starts a headless daemon (`llamaherd daemon`).
`llamaherd help` prints the authoritative list.

```bash
llamaherd list [substr]                    # models and their presets
llamaherd status                           # full snapshot (JSON)
llamaherd open <alias> [--preset name]     # start llama-server
llamaherd stop
llamaherd restart                          # stop + start with current settings
llamaherd rescan                           # re-walk the models directory
llamaherd logs [-f] [--tail n]             # app/server log (follow with -f)
llamaherd list --json                      # machine-readable models + presets
llamaherd status --summary                 # one-screen human summary
llamaherd completion bash|zsh|fish         # shell completion
llamaherd bench [list|clear]               # benchmark the running model/config, or list results
llamaherd save | load                      # persist / restore the current preset

llamaherd config add|update <alias> <name> [--set basic.port=8090 --set advanced.fit=on]
llamaherd config remove <alias> <name>
llamaherd config rename <alias> <old-name> <new-name>
llamaherd config reset [<name>]             # auto-fitted model default
llamaherd config export                     # portable preset JSON
llamaherd config autofit                    # fit context + GPU layers to this machine

llamaherd model rename <alias> [--name N] [--alias A]
llamaherd model move <alias> [<category/path>]      # no path = Uncategorized
llamaherd model favorite <alias> [--off]
llamaherd model delete <alias>                      # removes the .gguf from disk
llamaherd model import <file.gguf|dir>... [--mmproj <file>] [--move] [--allow-duplicate] [--check]
                                                    # files already in the library are skipped
llamaherd category create|delete <a/b/c>
llamaherd category rename <a/b/c> <new-name>

llamaherd download list [--all]            # picks for this machine (--all: the whole catalog)
llamaherd download files <owner/name>      # every .gguf of a repo with sizes (models, vision, MTP)
llamaherd download <owner/name> <file> [--vision <f>|--no-vision] [--mtp <f>|--no-mtp]
llamaherd download status | cancel | token [<t>|--clear]
llamaherd download queue [rm <id>] | pause | resume | retry

llamaherd doctor [--json]                  # environment check with fixes
llamaherd version
```

## HTTP API

| Method & path | Body / query | Purpose |
| --- | --- | --- |
| `GET /status` | — | full UI snapshot |
| `GET /models`, `/presets?alias=`, `/download`, `/doctor` | — | read state |
| `GET /logs?tail=n` (or `?after=seq`) | — | recent app/server log lines; `after` streams new ones |
| `GET /catalog[?all=1]` | — | model picks for this machine (`all=1`: every catalog entry) |
| `GET /download/files?ref=` | `owner/name` or URL | a repo's `.gguf` files grouped as models / vision / MTP, plus `license` and `gated` (`no` / `auto` / `manual`; gated repos need the licence accepted on Hugging Face and a token). Multi-part rows carry `parts` and the ordered `shards` paths. |
| `POST /open` | `{alias, preset}` | start the server |
| `POST /stop`, `/restart`, `/rescan` | — | stop / restart / rescan models |
| `POST /save`, `/load` | — | persist / restore the current preset |
| `POST /config/add`, `/config/update`, `/config/remove` | `{alias, name, config?}` | edit presets |
| `POST /config/rename` | `{alias, name, newName}` | rename a preset in place |
| `POST /config/reset` | `{name}` | reset to the auto-fitted model default |
| `POST /config/autofit` | — | fit context and GPU layers to this machine |
| `GET /config/export` | — | portable preset JSON |
| `POST /model/rename` | `{model, name?, alias?}` | display name / alias (moves `configs/<alias>.json`) |
| `POST /model/delete` | `{model}` | delete the `.gguf` (presets are kept) |
| `POST /model/import` | `{paths: [..], mmproj?, move?, allowDuplicate?}` (or legacy `{src}`) | copy/move local `.gguf` files or folders (plus one shared vision projector) into the models dir and create fitted presets; returns `report.items[]` with `status` `imported` / `duplicate` / `error` per file |
| `POST /model/import/check` | `{paths: [..]}` | duplicate preview: `checks[]` with `duplicate`, `existingAlias`, `existingPath`, `error` |
| `POST /benchmark` | — | measure the running model/config; returns and stores the result |
| `GET /benchmarks`, `POST /benchmarks/clear` | — | stored results (fastest first) / delete them |
| `POST /model/favorite` | `{model, favorite}` | pin / unpin a model |
| `POST /model/category` | `{model, category: [..]}` | place a model; `[]` = Uncategorized |
| `POST /category/create`, `/rename`, `/delete` | `{path: [..], name?}` | category folders (delete keeps models) |
| `POST /download` | `{ref, file, vision, mtp}` | start a download (`vision`/`mtp`: `""` = repo default, `"none"` = skip, or a file) |
| `POST /download/cancel`, `/token` | see `api/api.go` | cancel / Hugging Face token |
| `POST /download/pause`, `/download/resume`, `/download/retry` | — | pause the active transfer (keeps `.part`), resume it, retry the last failure once |
| `POST /download/queue` | `{id}` | remove one waiting job; the active transfer is untouched |

`model` is a file path or an alias. Mutating calls return `{"ok":true}` or
`{"ok":false,"error":"..."}`.

```bash
# The token is generated into the user config on first setup; set it in the
# environment for these examples (or use "apiToken": "off" to opt out).
export LLAMAHERD_API_TOKEN=$(jq -r .apiToken ~/.config/llamaherd/config.json)
curl -s -H "Authorization: Bearer $LLAMAHERD_API_TOKEN" \
  127.0.0.1:8517/status | jq '.status, .server.tokPerSec'
curl -s -X POST -H "Authorization: Bearer $LLAMAHERD_API_TOKEN" \
  -H 'content-type: application/json' 127.0.0.1:8517/open \
  -d '{"alias":"qwen3-8-27b-ud-iq4_xs"}'
```
