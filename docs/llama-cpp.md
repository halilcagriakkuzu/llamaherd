# Managing llama.cpp

LlamaHerd runs your own `llama-server` build; it does not bundle llama.cpp.

## Locating it

`serverBin` / `LLAMAHERD_SERVER_BIN` wins; otherwise `PATH` and the usual
`~/llama.cpp/build*/bin/llama-server` locations are searched. `llamaherd doctor` verifies
the binary, its version (warning below the minimum build the presets need,
`LLAMAHERD_MIN_LLAMA_BUILD`), the GPU, GUI libraries, models folder, disk space and API port.

## Updating from the GUI

The **llama.cpp** tab shows the installed version/commit and can update and rebuild the
checkout (Linux, macOS, WSL). It runs `scripts/update-llamacpp.sh`:

1. `git pull --ff-only` (never creates a merge commit),
2. `cmake -S . -B <build> -DGGML_CUDA=ON -DCMAKE_BUILD_TYPE=Release -DCMAKE_CUDA_ARCHITECTURES=<arch>`,
3. `cmake --build <build> --config Release -j<cores>`.

Output streams into the log; a running server keeps its old binary until restarted. Paths
come from `llamaDir` / `llamaBuildDir`. The script's own overrides: `LLAMA_DIR`,
`BUILD_DIR`, `CUDA_ARCH` (auto-detected from `nvidia-smi`, else `native`), `JOBS`,
`GIT_PULL_ARGS`.

The tab can also list and **force-stop** every `llama-server` process, including ones
LlamaHerd did not start (SIGTERM, then SIGKILL after 4 s).
