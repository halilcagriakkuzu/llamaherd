# Managing llama.cpp

LlamaHerd runs your own `llama-server` build; it does not bundle llama.cpp.

## Locating it

`serverBin` / `LLAMAHERD_SERVER_BIN` wins; otherwise `PATH` and the usual
`~/llama.cpp/build*/bin/llama-server` locations are searched. `llamaherd doctor` verifies
the binary, its version (warning below the minimum build the presets need,
`LLAMAHERD_MIN_LLAMA_BUILD`), the GPU, GUI libraries, models folder, disk space and API port.

No build yet? `scripts/install.sh` offers to clone and build one when it finds no
`llama-server` (`--install-llamacpp` to skip the question, `--llama-server PATH` to point at
an existing build). The GUI's first-run dialog and the llama.cpp tab can also set the path.

## Updating from the GUI

The **llama.cpp** tab shows the installed version/commit and can update and rebuild the
checkout (Linux, macOS, WSL). It runs `scripts/update-llamacpp.sh` — from a source checkout,
or the copy `install.sh` puts next to the binary (`llamaherd-update-llamacpp.sh`):

1. `git fetch --unshallow` when the checkout is shallow (a shallow clone reports build 1,
   because llama.cpp counts commits),
2. `git pull --ff-only` (never creates a merge commit),
3. `cmake -S . -B <build> -DGGML_CUDA=ON -DCMAKE_BUILD_TYPE=Release -DCMAKE_CUDA_ARCHITECTURES=<arch>`,
4. `cmake --build <build> --config Release -j<cores>`.

Output streams into the log; a running server keeps its old binary until restarted. Paths
come from `llamaDir` / `llamaBuildDir`. The script's own overrides: `LLAMA_DIR`,
`BUILD_DIR`, `CUDA_ARCH` (auto-detected from `nvidia-smi`, else `native`), `JOBS`,
`GIT_PULL_ARGS`.

## Metrics endpoint (`--metrics`)

The Advanced → Runtime **Metrics** toggle passes `--metrics`, which makes llama-server expose
Prometheus counters at `http://<host>:<port>/metrics` (slots, deferred requests, prompt-eval
seconds, KV-cache usage). Benchmark runs scrape it before and after the fixed prompt and store
the deltas/gauges in the result; the benchmark table shows prompt-eval time, busy/active slots,
queue depth and context usage. Older builds without `/metrics` simply leave those fields empty.

The tab can also list and **force-stop** every `llama-server` process, including ones
LlamaHerd did not start (SIGTERM, then SIGKILL after 4 s).
