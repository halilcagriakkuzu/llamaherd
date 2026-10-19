# CLI and control API

The GUI, the CLI and the HTTP API are three front ends of one controller, so they always
agree. The API listens on `127.0.0.1:8517` only (`LLAMAHERD_API_PORT` to change). Set
`LLAMAHERD_API_TOKEN` (or `apiToken`) to require `Authorization: Bearer <token>` or
`?token=<token>`; the bundled CLI sends it for you.

## CLI

If nothing is listening, the first command starts a headless daemon (`llamaherd daemon`).
`llamaherd help` prints the authoritative list.

```bash
llamaherd list [substr]                    # models and their presets
llamaherd status                           # full snapshot (JSON)
llamaherd open <alias> [--preset name]     # start llama-server
llamaherd stop
llamaherd save | load                      # persist / restore the current preset

llamaherd config add|update <alias> <name> [--set basic.port=8090 --set advanced.fit=on]
llamaherd config remove <alias> <name>
llamaherd config rename <alias> <old-name> <new-name>

llamaherd model rename <alias> [--name N] [--alias A]
llamaherd model move <alias> [<category/path>]      # no path = Uncategorized
llamaherd category create|delete <a/b/c>
llamaherd category rename <a/b/c> <new-name>

llamaherd download list [--all]            # picks for this machine (--all: the whole catalog)
llamaherd download files <owner/name>      # every .gguf of a repo with sizes (models, vision, MTP)
llamaherd download <owner/name> <file> [--vision <f>|--no-vision] [--mtp <f>|--no-mtp]
llamaherd download status | cancel | token [<t>|--clear]

llamaherd doctor [--json]                  # environment check with fixes
llamaherd version
```

## HTTP API

| Method & path | Body / query | Purpose |
| --- | --- | --- |
| `GET /status` | — | full UI snapshot |
| `GET /models`, `/presets?alias=`, `/download`, `/doctor` | — | read state |
| `GET /catalog[?all=1]` | — | model picks for this machine (`all=1`: every catalog entry) |
| `GET /download/files?ref=` | `owner/name` or URL | a repo's `.gguf` files grouped as models / vision / MTP |
| `POST /open` | `{alias, preset}` | start the server |
| `POST /stop` | — | stop the managed server |
| `POST /save`, `/load` | — | persist / restore the current preset |
| `POST /config/add`, `/config/update`, `/config/remove` | `{alias, name, config?}` | edit presets |
| `POST /config/rename` | `{alias, name, newName}` | rename a preset in place |
| `POST /model/rename` | `{model, name?, alias?}` | display name / alias (moves `configs/<alias>.json`) |
| `POST /model/category` | `{model, category: [..]}` | place a model; `[]` = Uncategorized |
| `POST /category/create`, `/rename`, `/delete` | `{path: [..], name?}` | category folders (delete keeps models) |
| `POST /download` | `{ref, file, vision, mtp}` | start a download (`vision`/`mtp`: `""` = repo default, `"none"` = skip, or a file) |
| `POST /download/cancel`, `/token` | see `api/api.go` | cancel / Hugging Face token |

`model` is a file path or an alias. Mutating calls return `{"ok":true}` or
`{"ok":false,"error":"..."}`.

```bash
curl -s 127.0.0.1:8517/status | jq '.status, .server.tokPerSec'
curl -s -X POST 127.0.0.1:8517/open -H 'content-type: application/json' \
  -d '{"alias":"qwen3-8-27b-ud-iq4_xs"}'
```
