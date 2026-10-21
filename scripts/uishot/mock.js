// Fake Wails backend so frontend/dist can be rendered in a plain browser
// (used by scripts/uishot/shoot.sh for UI screenshots).
(function () {
  // Headless screenshots run on virtual time: skip CSS transitions so they settle.
  document.head.insertAdjacentHTML("beforeend", "<style>*{transition:none!important}</style>");
  const params = new URLSearchParams(location.search);
  const status = params.get("status") || "running";
  if (params.get("theme")) { try { localStorage.setItem("theme", params.get("theme")); } catch (e) {} }
  if (params.get("notifications") === "1") {
    try {
      sessionStorage.setItem("llamaherd.notifications", JSON.stringify([
        { at: Date.now() - 60000, message: "server ready (127.0.0.1:8090)", kind: "ok" },
        { at: Date.now() - 30000, message: "download finished: Qwen3.5-9B-Q8_0.gguf", kind: "ok" },
        { at: Date.now(), message: "benchmark: Qwen3.8-27B-UD-IQ4_XS [default] 47.5 tok/s", kind: "" },
      ]));
    } catch (e) {}
  }
  if (params.get("logs") === "open") { try { localStorage.setItem("logsCollapsed", "0"); } catch (e) {} }
  const models = [
    { id: "qwen3-8-27b-ud-iq4_xs", display: "Qwen3.8-27B-UD-IQ4_XS", path: "/m/Qwen3.8-27B-UD-IQ4_XS.gguf", alias: "qwen3-8-27b-ud-iq4_xs", category: ["Qwen", "3.8"], mmproj: "/m/mmproj-F16.gguf", mtp: true, favorite: true, lastUsed: 1760000000, arch: "qwen35", layers: 64, heads: 24, kvHeads: 4, fullAttnLayers: 32, trainedContext: 262144, fileSize: 15180453472, quant: "UD-IQ4_XS", source: "hf:unsloth/Qwen3.8-27B-GGUF/Qwen3.8-27B-UD-IQ4_XS.gguf", provenance: { repo: "unsloth/Qwen3.8-27B-GGUF", url: "https://huggingface.co/unsloth/Qwen3.8-27B-GGUF", license: "apache-2.0" } },
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
      basic: { model: models[0].path, alias: models[0].alias, host: "127.0.0.1", port: 8090, context: 25556, gpuLayers: "all", parallelSlots: 1, flashAttention: "on", kCache: "q8_0", vCache: "q8_0", jinja: true, reasoningFmt: "deepseek", reasoningBudget: 0 },
      advanced: { specType: "draft-mtp", draftNMax: 3, draftNgl: "", draftTypeK: "", draftTypeV: "", draftModel: "", mmproj: params.get("vision") ? models[0].mmproj : "", noMMProjOffload: false, fit: "off", perf: true, sleepIdle: 300, cacheRAM: 0, mlock: false, noMmap: false, cpuThreads: 0, ubatch: 0, batch: 0, overrideTensor: "" },
    },
    configPath: "configs/qwen3-8-27b-ud-iq4_xs.json", dirty: params.get("dirty") === "1",
    commandLine: "llama-server --model /m/Qwen3.8-27B-UD-IQ4_XS.gguf --alias qwen3-8-27b-ud-iq4_xs --host 127.0.0.1 --port 8090 -ngl all --fit off -c 25556 -np 1 -fa on -ctk q8_0 -ctv q8_0 --jinja --reasoning-format deepseek --spec-type draft-mtp --spec-draft-n-max 3 --perf --sleep-idle-seconds 300",
    bootLines: ["settings: llama-server = /home/me/llama.cpp/build/bin/llama-server", "settings: models dir = /home/me/LlamaHerd/models", "models: scanned 5 *.gguf under /home/me/LlamaHerd/models", "backend: /home/me/llama.cpp/build/bin/llama-server"],
    warning: params.get("warn") ? "start blocked: estimated VRAM 17.4 GiB, card's 90% budget 14.4 GiB (total 16.0 GiB). Reduce the context and/or the q4 KV cache, or lower the GPU layer count (-ngl)." : "",
    vramEstimate: "estimated VRAM 13.6 GiB / 16.0 GiB (85%) [model 12.9, KV 0.55+0.55, SSM 0.03, compute 0.6; all layers on GPU, 16/65 layers full-attention]",
    vramPercent: params.get("warn") || params.get("issues") ? 109 : 85,
    vramFits: !(params.get("warn") || params.get("issues")),
    validation: params.get("issues") ? [
      { field: "context", severity: "warn", message: "Context 25556 exceeds the model's trained 8192 tokens." },
      { field: "gpuLayers", severity: "error", message: "Estimated VRAM use is 109% of the card; start will be blocked." },
    ] : [],
    metrics: running ? Array.from({ length: 60 }, (_, i) => ({
      at: 1760000000 + i * 5,
      vramMB: 14500 + Math.round(Math.sin(i / 4) * 550),
      ramMB: 1150 + Math.round(Math.cos(i / 6) * 220),
      tokPerSec: 47.5 + Math.sin(i / 5) * 7 * (1 - i / 60),
      gpuUtil: 88, gpuTemp: 61,
    })) : [],
    benchmarks: [
      { modelId: "qwen3-8-27b-ud-iq4_xs", model: "Qwen3.8-27B-UD-IQ4_XS", alias: "qwen3-8-27b-ud-iq4_xs", preset: "default", fingerprint: "3f2a91c47b0d55e1", at: 1760000100, context: 25556, gpuLayers: "all", promptN: 214, promptTokPerSec: 612.3, genN: 128, genTokPerSec: 47.5, peakVramMB: 15258, durationMs: 3100 },
      { modelId: "qwen3-8-27b-ud-iq4_xs", model: "Qwen3.8-27B-UD-IQ4_XS", alias: "qwen3-8-27b-ud-iq4_xs", preset: "high-ctx", fingerprint: "8d10ee5c22a0c901", at: 1759990000, context: 51200, gpuLayers: "48", promptN: 214, promptTokPerSec: 431.8, genN: 128, genTokPerSec: 39.2, peakVramMB: 16010, durationMs: 3800 },
      { modelId: "lfm2", model: "LFM2.5-24B-A2B-IQ3_M", alias: "lfm2", preset: "default", fingerprint: "aabbccddeeff0011", at: 1759980000, context: 32768, gpuLayers: "all", promptN: 214, promptTokPerSec: 390.1, genN: 128, genTokPerSec: 61.4, peakVramMB: 13980, durationMs: 2600 },
    ],
    benchmarking: false,
    configFingerprint: "3f2a91c47b0d55e1",
    runFingerprint: "3f2a91c47b0d55e1",
    recentLog: status === "crashed" || status === "failed"
      ? ["load_tensors: loading model tensors, this can take a while...", "llama_model_load: error loading model: unknown tensor", "failed to load model from file"]
      : [],
    llamaUpdating: false, firstRun: params.get("firstrun") === "1",
    serverBin: "/home/me/llama.cpp/build/bin/llama-server", modelsDir: "/home/me/LlamaHerd/models", userConfigPath: "/home/me/.config/llamaherd/config.json",
    doctor: { ok: true, summary: "1 warning", checks: [
      { id: "server-bin", title: "llama-server binary", status: "ok", detail: "/home/me/llama.cpp/build/bin/llama-server" },
      { id: "server-version", title: "llama-server version", status: "ok", detail: "0.7.0-dev (build 11708, commit 83f938941)" },
      { id: "gpu", title: "GPU / CUDA", status: "ok", detail: "1 GPU(s), 16.0 GiB total: NVIDIA GeForce RTX 4070 Ti SUPER" },
      { id: "wsl", title: "WSL model access", status: "warn", detail: "models are read through /mnt/<drive> (slow first load)", fix: "keep models inside WSL" },
      { id: "models", title: "Models directory", status: "ok", detail: "5 model(s), 1 mmproj sidecar(s)" },
    ] },
    download: params.get("dl") === "1" ? { active: true, currentId: "dl-1", paused: false, repo: "unsloth/Qwen3.5-9B-GGUF", file: "Qwen3.5-9B-Q8_0.gguf", dest: "", received: 4100000000, total: 9520000000, percent: 43, speedBps: 38000000, status: "downloading", error: "", message: "", queue: [
      { id: "dl-2", repo: "unsloth/Qwen3.8-27B-GGUF", file: "Qwen3.8-27B-UD-IQ3_XXS.gguf", status: "queued", message: "" },
      { id: "dl-3", repo: "unsloth/Ornith-1.5-9B-GGUF", file: "Ornith-1.5-9B-Q8_0.gguf", status: "queued", message: "" },
    ] } : { active: false, currentId: "", paused: false, repo: "", file: "", dest: "", received: 0, total: 0, percent: 0, speedBps: 0, status: "", error: "", message: "", queue: [] },
  };
  const llama = { updating: false, lastUpdate: "last update: 2026-10-16 ok", info: { version: "0.7.0-dev", commit: "83f938941", describe: "b11708", commitDate: "2026-10-17", subject: "server: fix slot reuse", branch: "master", dirty: false, behind: 0, fetched: "2026-10-18 09:12", binary: "/home/me/llama.cpp/build-latest/bin/llama-server", builtAt: "2026-10-17 21:40", buildDir: "build-latest", buildType: "Release", cudaOn: true, cudaArch: "89", gpu: "NVIDIA GeForce RTX 4070 Ti SUPER", repoDir: "/home/me/llama.cpp" }, procs: running ? [{ pid: 749941, uptime: "01:23:45", listen: "127.0.0.1:8090", model: "Qwen3.8-27B-UD-IQ4_XS", managed: true, cmd: "llama-server --model /m/Qwen3.8-27B-UD-IQ4_XS.gguf --port 8090" }] : [] };
  const E = (id, name, family, quant, bytes, paramsB, extra) => Object.assign({ id, name, family, repo: "unsloth/" + family, file: name.replace(/\W+/g, "-") + ".gguf", quant, bytes, paramsB, vision: false, mtp: false }, extra || {});
  const picks = [
    { role: "quality", label: "Best quality for this machine", rec: { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("a", "Qwen3.8 27B (UD-IQ3_XXS, vision)", "Qwen3.8-27B-GGUF", "UD-IQ3_XXS", 11862468192, 27, { vision: true }) } },
    { role: "vision", label: "Vision (understands images)", rec: { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("b", "Qwen3.5 9B (Q8_0, vision)", "Qwen3.5-9B-GGUF", "Q8_0", 10445668128, 9, { vision: true }) } },
    { role: "context", label: "Longest context", rec: { fit: "gpu", why: "fits 16.0 GiB VRAM; up to ~212k context", entry: E("c", "Ornith 1.5 9B (Q8_0, vision, MTP)", "Ornith-1.5-9B-GGUF", "Q8_0", 10707765056, 9, { vision: true, mtp: true }) } },
    { role: "light", label: "Lightweight and fast", rec: { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("d", "Llama 3.2 3B Instruct (Q4_K_M)", "Llama-3.2-3B-GGUF", "Q4_K_M", 2019377696, 3) } },
  ];
  const catalog = { vramMB: 16376, ramMB: 31800, picks, fitting: 11, total: 14, info: { source: "remote", updated: "2026-10-18", count: 33 }, diskFree: 412 * 1024 * 1024 * 1024, diskTotal: 2048 * 1024 * 1024 * 1024, modelsDir: "/home/me/LlamaHerd/models" };
  const full = { vramMB: 16376, ramMB: 31800, info: catalog.info, entries: [
    { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("e1", "Qwen3.8 27B (UD-IQ4_XS, vision)", "Qwen3.8-27B-GGUF", "UD-IQ4_XS", 15180453472, 27, { vision: true, family: "qwen3.8-27b", moe: false }) },
    { fit: "gpu", why: "fits 16.0 GiB VRAM with ~51k context", entry: E("e2", "Qwen3.8 27B (UD-IQ3_XXS, vision)", "Qwen3.8-27B-GGUF", "UD-IQ3_XXS", 11862468192, 27, { vision: true, family: "qwen3.8-27b" }) },
    { fit: "none", why: "needs ~27.0 GiB VRAM or ~25.5 GiB RAM", entry: E("e3", "Qwen3.8 27B (UD-Q6_K, vision)", "Qwen3.8-27B-GGUF", "UD-Q6_K", 22911284832, 27, { vision: true, family: "qwen3.8-27b" }) },
    { fit: "hybrid", why: "~64% of the weights fit in 16.0 GiB VRAM, the rest runs from RAM (slower)", entry: E("e4", "Qwen3-Coder-Next (UD-IQ1_M)", "Qwen3-Coder-Next-GGUF", "UD-IQ1_M", 21701693440, 80, { family: "qwen3-coder-next", moe: true }) },
    { fit: "none", why: "needs ~42.0 GiB VRAM or ~39.0 GiB RAM", entry: E("e5", "Qwen3-Coder-Next (UD-IQ4_XS)", "Qwen3-Coder-Next-GGUF", "UD-IQ4_XS", 38429272064, 80, { family: "qwen3-coder-next", moe: true }) },
  ] };
  const repoFiles = { repo: "unsloth/Qwen3.5-9B-GGUF", license: "apache-2.0", gated: params.get("gated") ? "manual" : "no",
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
    DetectServerBin: () => Promise.resolve({ path: "/home/me/llama.cpp/build/bin/llama-server", version: "0.7.0-dev (b11708)" }),
    DiagnosticsText: () => Promise.resolve("LlamaHerd mock diagnostics\nstatus: running"),
    CheckImport: (paths) => Promise.resolve(paths.map((p, i) => i === 0
      ? { path: p, size: 15180453472, duplicate: true, existingAlias: models[0].alias, existingPath: models[0].path }
      : { path: p, size: 5680000000, duplicate: false })),
    ImportModels: (paths) => Promise.resolve({ items: paths.map((p) => ({ source: p, status: "imported", alias: "x" })) }),
  }, { get: (t, k) => (k in t ? t[k] : noop) }) } };
  window.runtime = { EventsOn: () => {}, OnFileDrop: (cb) => { window.__mockFileDrop = cb; } };
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
    if (modal === "import") document.getElementById("importOpen").click();
    if (modal === "catalog") { document.getElementById("dlOpen").click(); setTimeout(() => document.getElementById("dlBrowse").click(), 50); }
    if (modal === "repo") { document.getElementById("dlOpen").click(); document.getElementById("dlRef").value = "https://huggingface.co/unsloth/Qwen3.5-9B-GGUF"; document.getElementById("dlFind").click(); }
    const click = (sel, text) => { const n = Array.from(document.querySelectorAll(sel)).find((x) => !text || x.textContent.trim().startsWith(text)); if (n) n.click(); return n; };
    if (params.get("view") === "all") click("#crumbs button", "All");
    const menu = params.get("menu");
    if (menu === "more") document.getElementById("moreBtn").click();
    if (params.get("notif") === "1") document.getElementById("notifBtn").click();
    if (menu === "model") document.querySelector("#modelList li:not(.folder)").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 560 }));
    if (menu === "folder") document.querySelector("#modelList li.folder").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 520 }));
    const dialog = params.get("dialog");
    if (dialog === "rename") { document.querySelector("#modelList li:not(.folder)").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 560 })); click("#ctxMenu button", "Rename"); }
    if (dialog === "move") { document.querySelector("#modelList li:not(.folder)").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 560 })); click("#ctxMenu button", "Move"); }
    if (dialog === "details") { document.querySelector("#modelList li:not(.folder)").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 560 })); click("#ctxMenu button", "Details"); }
  }, 150));
})();

// ---- UI smoke test -------------------------------------------------------
// `?selftest=main` / `?selftest=failed` runs a few assertions in the real
// frontend and appends the result to the DOM. scripts/uishot/smoke.sh reads it
// with Chromium --dump-dom, so no npm/Playwright runtime is required.
(function () {
  const params = new URLSearchParams(location.search);
  const mode = params.get("selftest");
  if (!mode) return;
  window.addEventListener("load", () => setTimeout(() => {
    const checks = [];
    const ok = (name, pass) => checks.push({ name, pass: !!pass });
    const text = (id) => {
      const n = document.getElementById(id);
      return n ? n.textContent || "" : "";
    };
    const byId = (id) => document.getElementById(id);

    if (mode === "failed") {
      ok("status word", text("statusWord").includes("NOT RESPONDING"));
      ok("banner visible", byId("procBanner") && !byId("procBanner").classList.contains("hidden"));
      ok("banner title", text("procBannerTitle").includes("not responding"));
      ok("recent log tail", text("procBannerLog").includes("failed to load model"));
      ok("retry button", !!byId("procRetry"));
      ok("stop button", !!byId("procStop"));
    } else {
      ok("running status", text("statusWord").includes("RUNNING"));
      ok("model rows", document.querySelectorAll("#modelList li:not(.folder)").length > 0);
      ok("benchmark rows", document.querySelectorAll("#benchBody tr").length === 3);
      ok("benchmark current row", !!document.querySelector("#benchBody tr.current"));
      ok("benchmark fastest first", (() => {
        const cells = Array.from(document.querySelectorAll("#benchBody tr td:nth-child(5)"));
        const values = cells.map((c) => parseFloat(c.textContent) || 0);
        return values.every((v, i) => i === 0 || values[i - 1] >= v);
      })());

      ok("detect button", !!byId("frBinDetect") && !!byId("pathBinDetect"));
      ok("diagnostics button", !!byId("diagCopy"));

      // Secondary actions live behind one "⋯" menu.
      byId("moreBtn").click();
      ok("more menu open", byId("moreMenu") && !byId("moreMenu").hidden);
      ok("more menu items", !!byId("moreDiag") && !!byId("moreUpdate") && !!byId("morePaths") && !!byId("themeBtn"));
      byId("moreBtn").click();
      ok("more menu closes", byId("moreMenu").hidden);
      ok("notification center button", !!byId("notifBtn"));
      // Any toast also lands in the notification center.
      byId("rescanBtn").click();

      // Download queue: two waiting jobs, pause/resume/retry controls.
      if (byId("dlOpen")) byId("dlOpen").click();
      ok("download queue rows", document.querySelectorAll("#dlQueue .dl-queue-row").length === 2);
      ok("download pause button", !!byId("dlPause") && !byId("dlPause").hidden);
      ok("download resume defined", !!byId("dlResume"));
      ok("download remove buttons", document.querySelectorAll("#dlQueue .dl-queue-row button").length === 2);
      if (byId("dlClose")) byId("dlClose").click();

      const openBtn = byId("importOpen");
      ok("import button", !!openBtn);
      if (openBtn) openBtn.click();
      ok("import dialog", byId("importModal") && !byId("importModal").hidden);
      ok("models dir shown", text("importModelsDir").includes("models"));
      const mm = byId("importHasMMProj");
      if (mm) mm.click();
      ok("mmproj picker", byId("importMMProjRow") && !byId("importMMProjRow").hidden);
      const cancel = byId("importCancel");
      if (cancel) cancel.click();
      ok("import closes", byId("importModal").hidden);

      // Provenance in the details drawer.
      const first = document.querySelector("#modelList li:not(.folder)");
      if (first) first.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: 150, clientY: 560 }));
      const det = Array.from(document.querySelectorAll("#ctxMenu button")).find((b) => b.textContent.trim().startsWith("Details"));
      if (det) det.click();
      ok("details licence", text("detailsBody").includes("apache-2.0") && text("detailsBody").includes("unsloth/Qwen3.8-27B-GGUF"));
      ok("details source link", byId("detailsOpenSource") && !byId("detailsOpenSource").hidden);
      if (byId("detailsClose")) byId("detailsClose").click();

      // Drag-and-drop onto the Models panel opens the dialog with a preview.
      ok("drop handler", typeof window.__mockFileDrop === "function");
      if (window.__mockFileDrop) window.__mockFileDrop(10, 10, ["/in/Qwen3.8-27B-UD-IQ4_XS.gguf", "/in/new-model.gguf", "/in/notes.txt"]);
      ok("drop opens import", !byId("importModal").hidden);
      ok("drop fills paths", (byId("importModelPath").value.match(/\.gguf/g) || []).length === 2);
    }

    const finish = () => {
    const failed = checks.filter((c) => !c.pass);
    const box = document.createElement("div");
    box.id = "selftest";
    box.setAttribute("data-checks", String(checks.length));
    box.textContent = failed.length
      ? "SELFTEST FAIL: " + failed.map((c) => c.name).join(", ")
      : "SELFTEST PASS (" + checks.length + " checks)";
    document.body.appendChild(box);
    };
    if (mode === "failed") { finish(); return; }
    setTimeout(() => {
      ok("duplicate preview", document.querySelectorAll("#importList .import-duplicate").length === 1 &&
        document.querySelectorAll("#importList .import-new").length === 1);
      ok("allow-duplicate toggle", !byId("importAllowDupRow").hidden);
      ok("notification center populated", document.querySelectorAll("#notifList .notif-item").length > 0);
      ok("notification badge", byId("notifCount") && !byId("notifCount").hidden);
      byId("notifBtn").click();
      ok("notification panel opens", byId("notifPanel") && !byId("notifPanel").hidden);
      finish();
    }, 100);
  }, 700));
})();
