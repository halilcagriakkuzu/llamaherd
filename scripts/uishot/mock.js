// Fake Wails backend so frontend/dist can be rendered in a plain browser
// (used by scripts/uishot/shoot.sh for UI screenshots).
(function () {
  // Headless screenshots run on virtual time: skip CSS transitions so they settle.
  document.head.insertAdjacentHTML("beforeend", "<style>*{transition:none!important}</style>");
  const params = new URLSearchParams(location.search);
  const status = params.get("status") || "running";
  if (params.get("theme")) { try { localStorage.setItem("theme", params.get("theme")); } catch (e) {} }
  if (params.get("logs") === "open") { try { localStorage.setItem("logsCollapsed", "0"); } catch (e) {} }
  const models = [
    { display: "Qwen3.8-27B-UD-IQ4_XS", path: "/m/Qwen3.8-27B-UD-IQ4_XS.gguf", alias: "qwen3-8-27b-ud-iq4_xs", category: ["Qwen", "3.8"], mmproj: "/m/mmproj-F16.gguf", mtp: true },
    { display: "Qwen3.8-27B-UD-Q3_K_XL", path: "/m/q3.gguf", alias: "qwen3-8-27b-ud-q3_k_xl", category: ["Qwen", "3.8"], mtp: false },
    { display: "Qwen3.8-27B-Uncensored-Q3_K_P", path: "/m/u.gguf", alias: "qwen3-8-uncensored", category: ["Qwen", "3.8"], mmproj: "/m/mm.gguf", mtp: true, mtpDraft: "/m/d.gguf" },
    { display: "Ornith-1.5-9B-Q8_0", path: "/m/o.gguf", alias: "ornith-1-5-9b-q8_0", category: ["Ornith"] },
    { display: "LFM2.5-24B-A2B-IQ3_M", path: "/m/l.gguf", alias: "lfm2", category: [] },
  ];
  const running = status === "running";
  const snap = {
    status, crashExit: status === "crashed" ? 3 : 0,
    server: {
      status, pid: running ? "749941" : "—", binary: "/home/me/llama.cpp/build/bin/llama-server",
      listen: running ? "127.0.0.1:8090" : "", wslIp: "172.24.16.1",
      model: running ? "Qwen3.8-27B-UD-IQ4_XS.gguf" : "—", context: running ? "25556" : "—",
      gpuLayers: running ? "all" : "—", vram: running ? "14.9 GB" : "—", cpuRam: running ? "1.2 GB" : "—",
      uptime: running ? "01:23:45" : "00:00:00",
    },
    models, selectedModel: 0, alias: models[0].alias,
    presetNames: ["default", "high-ctx", "vision+mtp"], selectedPreset: 0,
    config: {
      basic: { model: models[0].path, alias: models[0].alias, host: "127.0.0.1", port: 8090, context: 25556, gpuLayers: "all", parallelSlots: 1, flashAttention: "on", kCache: "q8_0", vCache: "q8_0", jinja: true, reasoningFmt: "deepseek" },
      advanced: { specType: "draft-mtp", draftNMax: 3, draftNgl: "", draftTypeK: "", draftTypeV: "", draftModel: "", mmproj: params.get("vision") ? models[0].mmproj : "", noMMProjOffload: false, fit: "off", perf: true, sleepIdle: 300, cacheRAM: 0, mlock: false, noMmap: false, cpuThreads: 0, ubatch: 0, batch: 0, overrideTensor: "" },
    },
    configPath: "configs/qwen3-8-27b-ud-iq4_xs.json", dirty: params.get("dirty") === "1",
    commandLine: "llama-server --model /m/Qwen3.8-27B-UD-IQ4_XS.gguf --alias qwen3-8-27b-ud-iq4_xs --host 127.0.0.1 --port 8090 -ngl all --fit off -c 25556 -np 1 -fa on -ctk q8_0 -ctv q8_0 --jinja --reasoning-format deepseek --spec-type draft-mtp --spec-draft-n-max 3 --perf --sleep-idle-seconds 300",
    bootLines: ["settings: llama-server = /home/me/llama.cpp/build/bin/llama-server", "settings: models dir = /home/me/LlamaHerd/models", "models: scanned 5 *.gguf under /home/me/LlamaHerd/models", "backend: /home/me/llama.cpp/build/bin/llama-server"],
    warning: params.get("warn") ? "start blocked: estimated VRAM 17.4 GiB, card's 90% budget 14.4 GiB (total 16.0 GiB). Reduce the context and/or the q4 KV cache, or lower the GPU layer count (-ngl)." : "",
    vramEstimate: "estimated VRAM 13.6 GiB / 16.0 GiB (85%) [model 12.9, KV 0.55+0.55, SSM 0.03, compute 0.6; all layers on GPU, 16/65 layers full-attention]",
    llamaUpdating: false, firstRun: params.get("firstrun") === "1",
    serverBin: "/home/me/llama.cpp/build/bin/llama-server", modelsDir: "/home/me/LlamaHerd/models", userConfigPath: "/home/me/.config/llamaherd/config.json",
    doctor: { ok: true, summary: "all checks passed", checks: [
      { id: "server-bin", title: "llama-server binary", status: "ok", detail: "/home/me/llama.cpp/build/bin/llama-server" },
      { id: "server-version", title: "llama-server version", status: "ok", detail: "0.7.0-dev (build 11708, commit 83f938941)" },
      { id: "gpu", title: "GPU / CUDA", status: "ok", detail: "1 GPU(s), 16.0 GiB total: NVIDIA GeForce RTX 4070 Ti SUPER" },
      { id: "wsl", title: "WSL model access", status: "warn", detail: "models are read through /mnt/<drive> (slow first load)", fix: "keep models inside WSL" },
      { id: "models", title: "Models directory", status: "ok", detail: "5 model(s), 1 mmproj sidecar(s)" },
    ] },
    download: { active: false, repo: "", file: "", dest: "", received: 0, total: 0, percent: 0, speedBps: 0, status: "", error: "", message: "" },
  };
  const llama = { updating: false, lastUpdate: "last update: 2026-10-16 ok", info: { version: "0.7.0-dev", commit: "83f938941", describe: "b11708", commitDate: "2026-10-17", subject: "server: fix slot reuse", branch: "master", dirty: false, behind: 0, fetched: "2026-10-18 09:12", binary: "/home/me/llama.cpp/build-latest/bin/llama-server", builtAt: "2026-10-17 21:40", buildDir: "build-latest", buildType: "Release", cudaOn: true, cudaArch: "89", gpu: "NVIDIA GeForce RTX 4070 Ti SUPER", repoDir: "/home/me/llama.cpp" }, procs: running ? [{ pid: 749941, uptime: "01:23:45", listen: "127.0.0.1:8090", model: "Qwen3.8-27B-UD-IQ4_XS", managed: true, cmd: "llama-server --model /m/Qwen3.8-27B-UD-IQ4_XS.gguf --port 8090" }] : [] };
  const catalog = { vramMB: 16376, ramMB: 31800, entries: [
    { entry: { name: "Qwen3.5-9B", repo: "unsloth/Qwen3.5-9B-GGUF", bytes: 6012954214, context: 50000, starter: true, vision: true }, fit: "gpu", why: "fits fully on GPU" },
    { entry: { name: "Qwen3.8-27B (UD-IQ4_XS)", repo: "unsloth/Qwen3.8-27B-GGUF", bytes: 16320875724, context: 50000, mtp: true }, fit: "cpu", why: "partial GPU offload" },
    { entry: { name: "Llama-3.3-70B", repo: "bartowski/Llama-3.3-70B-GGUF", bytes: 42520000000, context: 32000 }, fit: "too-big", why: "needs ~48 GB" },
  ] };
  const noop = () => Promise.resolve();
  window.go = { main: { App: new Proxy({
    GetState: () => Promise.resolve(snap),
    GetLlamaPanel: () => Promise.resolve(llama), CheckLlamaUpdates: () => Promise.resolve(llama),
    GetCatalog: () => Promise.resolve(catalog), GetHFTokenStatus: () => Promise.resolve({ configured: false }),
  }, { get: (t, k) => (k in t ? t[k] : noop) }) } };
  window.runtime = { EventsOn: () => {} };
  setTimeout(() => {
    const log = document.getElementById("logView");
    if (!log) return;
    ["settings: llama-server = /home/me/llama.cpp/build/bin/llama-server", "models: scanned 5 *.gguf", "pid 749941 spawned: llama-server --model /m/Qwen3.8-27B-UD-IQ4_XS.gguf ...", "[stderr] load_tensors: offloaded 65/65 layers to GPU", "[stderr] llama_context: n_ctx = 25556", "ready in 14.2s (127.0.0.1:8090)"].forEach((l) => { const d = document.createElement("div"); d.textContent = l; log.appendChild(d); });
  }, 50);
  const tab = params.get("tab");
  const modal = params.get("modal");
  window.addEventListener("load", () => setTimeout(() => {
    if (tab) document.getElementById({ basic: "tabBtnBasic", advanced: "tabBtnAdvanced", llama: "tabBtnLlama" }[tab]).click();
    if (modal === "download") document.getElementById("dlOpen").click();
  }, 150));
})();
