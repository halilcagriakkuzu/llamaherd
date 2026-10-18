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
      gpuLayers: running ? "all" : "—", tokPerSec: running ? "47.5 tok/s" : "—", tokPerSecTip: "Average of 3 request(s), 190 tokens\nmin 30.0 · max 60.0 · last 60.0 tok/s", vram: running ? "14.9 GB" : "—", cpuRam: running ? "1.2 GB" : "—",
      uptime: running ? "01:23:45" : "00:00:00",
    },
    models, categories: [["Coding"], ["Qwen", "Experiments"]], selectedModel: 0, alias: models[0].alias,
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
    doctor: { ok: true, summary: "1 warning", checks: [
      { id: "server-bin", title: "llama-server binary", status: "ok", detail: "/home/me/llama.cpp/build/bin/llama-server" },
      { id: "server-version", title: "llama-server version", status: "ok", detail: "0.7.0-dev (build 11708, commit 83f938941)" },
      { id: "gpu", title: "GPU / CUDA", status: "ok", detail: "1 GPU(s), 16.0 GiB total: NVIDIA GeForce RTX 4070 Ti SUPER" },
      { id: "wsl", title: "WSL model access", status: "warn", detail: "models are read through /mnt/<drive> (slow first load)", fix: "keep models inside WSL" },
      { id: "models", title: "Models directory", status: "ok", detail: "5 model(s), 1 mmproj sidecar(s)" },
    ] },
    download: params.get("dl") === "1" ? { active: true, repo: "unsloth/Qwen3.5-9B-GGUF", file: "Qwen3.5-9B-Q8_0.gguf", dest: "", received: 4100000000, total: 9520000000, percent: 43, speedBps: 38000000, status: "downloading", error: "", message: "" } : { active: false, repo: "", file: "", dest: "", received: 0, total: 0, percent: 0, speedBps: 0, status: "", error: "", message: "" },
  };
  const llama = { updating: false, lastUpdate: "last update: 2026-10-16 ok", info: { version: "0.7.0-dev", commit: "83f938941", describe: "b11708", commitDate: "2026-10-17", subject: "server: fix slot reuse", branch: "master", dirty: false, behind: 0, fetched: "2026-10-18 09:12", binary: "/home/me/llama.cpp/build-latest/bin/llama-server", builtAt: "2026-10-17 21:40", buildDir: "build-latest", buildType: "Release", cudaOn: true, cudaArch: "89", gpu: "NVIDIA GeForce RTX 4070 Ti SUPER", repoDir: "/home/me/llama.cpp" }, procs: running ? [{ pid: 749941, uptime: "01:23:45", listen: "127.0.0.1:8090", model: "Qwen3.8-27B-UD-IQ4_XS", managed: true, cmd: "llama-server --model /m/Qwen3.8-27B-UD-IQ4_XS.gguf --port 8090" }] : [] };
  const E = (id, name, family, quant, bytes, paramsB, extra) => Object.assign({ id, name, family, repo: "unsloth/" + family, file: name.replace(/\W+/g, "-") + ".gguf", quant, bytes, paramsB, vision: false, mtp: false }, extra || {});
  const picks = [
    { role: "quality", label: "Best quality for this machine", rec: { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("a", "Qwen3.8 27B (UD-IQ3_XXS, vision)", "Qwen3.8-27B-GGUF", "UD-IQ3_XXS", 11862468192, 27, { vision: true }) } },
    { role: "vision", label: "Vision (understands images)", rec: { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("b", "Qwen3.5 9B (Q8_0, vision)", "Qwen3.5-9B-GGUF", "Q8_0", 10445668128, 9, { vision: true }) } },
    { role: "context", label: "Longest context", rec: { fit: "gpu", why: "fits 16.0 GiB VRAM; up to ~212k context", entry: E("c", "Ornith 1.5 9B (Q8_0, vision, MTP)", "Ornith-1.5-9B-GGUF", "Q8_0", 10707765056, 9, { vision: true, mtp: true }) } },
    { role: "light", label: "Lightweight and fast", rec: { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("d", "Llama 3.2 3B Instruct (Q4_K_M)", "Llama-3.2-3B-GGUF", "Q4_K_M", 2019377696, 3) } },
  ];
  const catalog = { vramMB: 16376, ramMB: 31800, picks, fitting: 11, total: 14, info: { source: "remote", updated: "2026-10-18", count: 33 } };
  const full = { vramMB: 16376, ramMB: 31800, info: catalog.info, entries: [
    { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("e1", "Qwen3.8 27B (UD-IQ4_XS, vision)", "Qwen3.8-27B-GGUF", "UD-IQ4_XS", 15180453472, 27, { vision: true, family: "qwen3.8-27b", moe: false }) },
    { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("e2", "Qwen3.8 27B (UD-IQ3_XXS, vision)", "Qwen3.8-27B-GGUF", "UD-IQ3_XXS", 11862468192, 27, { vision: true, family: "qwen3.8-27b" }) },
    { fit: "none", why: "needs ~27.0 GiB VRAM or ~25.5 GiB RAM", entry: E("e3", "Qwen3.8 27B (UD-Q6_K, vision)", "Qwen3.8-27B-GGUF", "UD-Q6_K", 22911284832, 27, { vision: true, family: "qwen3.8-27b" }) },
    { fit: "hybrid", why: "~64% of the weights fit in 16.0 GiB VRAM, the rest runs from RAM (slower)", entry: E("e4", "Qwen3-Coder-Next (UD-IQ1_M)", "Qwen3-Coder-Next-GGUF", "UD-IQ1_M", 21701693440, 80, { family: "qwen3-coder-next", moe: true }) },
    { fit: "none", why: "needs ~42.0 GiB VRAM or ~39.0 GiB RAM", entry: E("e5", "Qwen3-Coder-Next (UD-IQ4_XS)", "Qwen3-Coder-Next-GGUF", "UD-IQ4_XS", 38429272064, 80, { family: "qwen3-coder-next", moe: true }) },
  ] };
  const repoFiles = { repo: "unsloth/Qwen3.5-9B-GGUF",
    models: [
      { path: "Qwen3.5-9B-UD-IQ2_M.gguf", name: "Qwen3.5-9B-UD-IQ2_M.gguf", quant: "UD-IQ2_M", size: 3651000000 },
      { path: "Qwen3.5-9B-UD-Q3_K_XL.gguf", name: "Qwen3.5-9B-UD-Q3_K_XL.gguf", quant: "UD-Q3_K_XL", size: 5057000000 },
      { path: "Qwen3.5-9B-Q4_K_M.gguf", name: "Qwen3.5-9B-Q4_K_M.gguf", quant: "Q4_K_M", size: 5680000000 },
      { path: "Qwen3.5-9B-Q8_0.gguf", name: "Qwen3.5-9B-Q8_0.gguf", quant: "Q8_0", size: 9520000000 },
      { path: "BF16/Qwen3.5-9B-BF16-00001-of-00002.gguf", name: "Qwen3.5-9B-BF16.gguf", quant: "BF16", size: 17920000000, parts: 2 },
    ],
    mmprojs: [{ path: "mmproj-F16.gguf", name: "mmproj-F16.gguf", quant: "F16", size: 921000000 }, { path: "mmproj-F32.gguf", name: "mmproj-F32.gguf", quant: "F32", size: 1830000000 }],
    mtpDrafts: [{ path: "MTP/mtp-Qwen3.5-9B-Q4_0.gguf", name: "mtp-Qwen3.5-9B-Q4_0.gguf", quant: "Q4_0", size: 430000000 }] };
  const noop = () => Promise.resolve();
  window.go = { main: { App: new Proxy({
    GetState: () => Promise.resolve(snap),
    GetLlamaPanel: () => Promise.resolve(llama), CheckLlamaUpdates: () => Promise.resolve(llama),
    GetCatalog: () => Promise.resolve(catalog), RefreshCatalog: () => Promise.resolve(catalog), GetFullCatalog: () => Promise.resolve(full), ListRepoFiles: () => Promise.resolve(repoFiles), GetHFTokenStatus: () => Promise.resolve({ configured: false }),
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
    if (modal === "catalog") { document.getElementById("dlOpen").click(); setTimeout(() => document.getElementById("dlBrowse").click(), 50); }
    if (modal === "repo") { document.getElementById("dlOpen").click(); document.getElementById("dlRef").value = "https://huggingface.co/unsloth/Qwen3.5-9B-GGUF"; document.getElementById("dlFind").click(); }
    const click = (sel, text) => { const n = Array.from(document.querySelectorAll(sel)).find((x) => !text || x.textContent.trim().startsWith(text)); if (n) n.click(); return n; };
    if (params.get("view") === "all") click("#crumbs button", "All");
    const menu = params.get("menu");
    if (menu === "model") document.querySelector("#modelList li:not(.folder)").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 560 }));
    if (menu === "folder") document.querySelector("#modelList li.folder").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 520 }));
    const dialog = params.get("dialog");
    if (dialog === "rename") { document.querySelector("#modelList li:not(.folder)").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 560 })); click("#ctxMenu button", "Rename"); }
    if (dialog === "move") { document.querySelector("#modelList li:not(.folder)").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 560 })); click("#ctxMenu button", "Move"); }
  }, 150));
})();
