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

llamaherd model rename <alias> [--name N] [--alias A]
llamaherd model move <alias> [<category/path>]      # no path = Uncategorized
llamaherd category create|delete <a/b/c>
llamaherd category rename <a/b/c> <new-name>

llamaherd download list                    # catalog scored for this machine
llamaherd download <owner/name> [<file>] [--vision <file>]
llamaherd download status | cancel | token [<t>|--clear]

llamaherd doctor [--json]                  # environment check with fixes
llamaherd version
```

## HTTP API

| Method & path | Body / query | Purpose |
| --- | --- | --- |
| `GET /status` | — | full UI snapshot |
| `GET /models`, `/presets?alias=`, `/catalog`, `/download`, `/doctor` | — | read state |
| `POST /open` | `{alias, preset}` | start the server |
| `POST /stop` | — | stop the managed server |
| `POST /save`, `/load` | — | persist / restore the current preset |
| `POST /config/add`, `/config/update`, `/config/remove` | `{alias, name, config?}` | edit presets |
| `POST /model/rename` | `{model, name?, alias?}` | display name / alias (moves `configs/<alias>.json`) |
| `POST /model/category` | `{model, category: [..]}` | place a model; `[]` = Uncategorized |
| `POST /category/create`, `/rename`, `/delete` | `{path: [..], name?}` | category folders (delete keeps models) |
| `POST /download`, `/download/cancel`, `/token` | see `api/api.go` | Hugging Face downloads / token |

`model` is a file path or an alias. Mutating calls return `{"ok":true}` or
`{"ok":false,"error":"..."}`.

```bash
curl -s 127.0.0.1:8517/status | jq '.status, .server.tokPerSec'
curl -s -X POST 127.0.0.1:8517/open -H 'content-type: application/json' \
  -d '{"alias":"qwen3-8-27b-ud-iq4_xs"}'
```
