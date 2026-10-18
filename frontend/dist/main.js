"use strict";

(function () {
  const $ = (id) => document.getElementById(id);

  const el = {
    statusHead: $("statusHead"),
    statusDot: $("statusDot"),
    statusWord: $("statusWord"),
    kv: {
      pid: $("kv-pid"),
      listen: $("kv-listen"),
      wslip: $("kv-wslip"),
      model: $("kv-model"),
      context: $("kv-context"),
      gpulayers: $("kv-gpulayers"),
      tps: $("kv-tps"),
      vram: $("kv-vram"),
      cpuram: $("kv-cpuram"),
      uptime: $("kv-uptime"),
    },
    modelList: $("modelList"),
    crumbs: $("crumbs"),
    visionChk: $("visionChk"),
    visionHint: $("visionHint"),
    mtpChk: $("mtpChk"),
    mtpHint: $("mtpHint"),
    nglAll: $("nglAll"),
    sectVision: $("sectVision"),
    sectSpec: $("sectSpec"),
    statusInfo: $("statusInfo"),
    logsGroup: $("logsGroup"),
    logsTitle: $("logsTitle"),
    presetSel: $("presetSel"),
    saveBtn: $("saveBtn"),
    loadBtn: $("loadBtn"),
    startBtn: $("startBtn"),
    stopBtn: $("stopBtn"),
    configPath: $("configPath"),
    tabBtnBasic: $("tabBtnBasic"),
    tabBtnAdvanced: $("tabBtnAdvanced"),
    tabBtnLlama: $("tabBtnLlama"),
    tabLlama: $("tabLlama"),
    llamaInfo: $("llamaInfo"),
    llamaProcs: $("llamaProcs"),
    llamaLast: $("llamaLast"),
    llamaRefresh: $("llamaRefresh"),
    llamaCheck: $("llamaCheck"),
    llamaUpdate: $("llamaUpdate"),
    llamaCancel: $("llamaCancel"),
    llamaKill: $("llamaKill"),
    tabBasic: $("tabBasic"),
    tabAdvanced: $("tabAdvanced"),
    cmdline: $("cmdline"),
    logView: $("logView"),
      warnBanner: $("warnBanner"),
    vramEstimate: $("vramEstimate"),
    doctorBanner: $("doctorBanner"),
    doctorList: $("doctorList"),
    doctorRefresh: $("doctorRefresh"),
    doctorSummary: $("doctorSummary"),
    firstRun: $("firstRun"),
    frBin: $("frBin"),
    frModels: $("frModels"),
    frPath: $("frPath"),
    frMsg: $("frMsg"),
    frSave: $("frSave"),
    dlOpen: $("dlOpen"),
    dlModal: $("dlModal"),
    dlSummary: $("dlSummary"),
    dlList: $("dlList"),
    dlRef: $("dlRef"),
    dlFind: $("dlFind"),
    dlMsg: $("dlMsg"),
    dlCatalogBox: $("dlCatalogBox"),
    dlHidden: $("dlHidden"),
    dlBrowse: $("dlBrowse"),
    dlBadge: $("dlBadge"),
    catModal: $("catModal"),
    catSummary: $("catSummary"),
    catList: $("catList"),
    catClose: $("catClose"),
    dlFiles: $("dlFiles"),
    dlFilesHead: $("dlFilesHead"),
    dlOptions: $("dlOptions"),
    dlFileList: $("dlFileList"),
    dlClose: $("dlClose"),
    dlToken: $("dlToken"),
    dlTokenSave: $("dlTokenSave"),
    dlTokenClear: $("dlTokenClear"),
    dlTokenMsg: $("dlTokenMsg"),
    dlProgress: $("dlProgress"),
    dlBarInner: $("dlBarInner"),
    dlProgressText: $("dlProgressText"),
    dlCancel: $("dlCancel"),
    frDownload: $("frDownload")
  };

  let snap = null;
  let bootPrinted = false;
  let uptimeSec = 0;
  let lastStatus = "";

  // ---- bindings ----

  function app() {
    return (window.go && window.go.main && window.go.main.App) || null;
  }

  function call(method, ...args) {
    const a = app();
    if (!a) {
      appendLog("bindings unavailable (run inside wails)");
      return;
    }
    Promise.resolve(a[method](...args)).catch((err) =>
      appendLog(`error: ${err && err.message ? err.message : err}`)
    );
  }

  function runtime() {
    return window.runtime || null;
  }

  // ---- log ----

  function appendLog(line) {
    const div = document.createElement("div");
    if (String(line).startsWith("server:")) div.className = "log-server";
    div.textContent = String(line);
    el.logView.appendChild(div);
    el.logView.scrollTop = el.logView.scrollHeight;
  }

  // ---- form ----

  function pull(cfg, keys) {
    const out = {};
    keys.forEach((k) => { out[k] = cfg && cfg[k] !== undefined ? cfg[k] : ""; });
    return out;
  }

  const BASIC_KEYS = [
    "model", "alias", "host", "port", "context", "gpuLayers", "parallelSlots",
    "flashAttention", "kCache", "vCache", "jinja", "reasoningFmt",
  ];
  const ADV_KEYS = [
    "specType", "draftNMax", "draftNgl", "draftTypeK", "draftTypeV", "draftModel",
    "mmproj", "noMMProjOffload", "fit", "perf", "sleepIdle", "cacheRAM",
    "mlock", "noMmap", "cpuThreads", "ubatch", "batch", "overrideTensor",
  ];

  function fillForm() {
    if (!snap || !snap.config) return;
    const basic = pull(snap.config.basic, BASIC_KEYS);
    const adv = pull(snap.config.advanced, ADV_KEYS);
    const values = Object.assign({}, basic, adv);
    document.querySelectorAll("[data-key]").forEach((node) => {
      if (!(node.dataset.key in values)) return;
      const v = values[node.dataset.key];
      if (node.type === "checkbox") {
        if (node.checked !== !!v) node.checked = !!v;
        return;
      }
      const next = String(v);
      if (String(node.value) !== next) node.value = next;
    });
    syncSpecState();
  }

  function collectConfig() {
    const cfg = { basic: {}, advanced: {} };
    document.querySelectorAll("[data-key]").forEach((node) => {
      const group = node.dataset.group;
      const key = node.dataset.key;
      if (node.type === "checkbox") {
        cfg[group][key] = node.checked;
      } else if (node.type === "number") {
        const n = node.value === "" ? 0 : parseInt(node.value, 10);
        cfg[group][key] = Number.isNaN(n) ? 0 : n;
      } else {
        cfg[group][key] = node.value;
      }
    });
    return cfg;
  }

  // The three Basic checkboxes are views over real fields:
  //   Vision  <-> advanced.mmproj != ""       (path comes from the model's library meta)
  //   MTP     <-> advanced.specType == draft-mtp
  //   ngl all <-> basic.gpuLayers == "all"
  let lastNgl = "";

  function selectedModel() {
    return snap && (snap.models || [])[snap.selectedModel];
  }

  function syncSpecState() {
    const specOff = $("f-specType").value === "none";
    ["draftModel", "draftNMax", "draftNgl", "draftTypeK", "draftTypeV"].forEach((key) => {
      const node = $(`f-${key}`);
      if (node) node.disabled = specOff;
    });

    const ngl = $("f-gpuLayers");
    const all = ngl.value.trim() === "all";
    if (!all && ngl.value.trim() !== "") lastNgl = ngl.value.trim();
    el.nglAll.checked = all;
    ngl.disabled = all;

    const m = selectedModel();
    const mmproj = $("f-mmproj").value;
    const canVision = !!(m && m.mmproj) || mmproj !== "";
    el.visionChk.checked = mmproj !== "";
    el.visionChk.disabled = !canVision;
    el.visionHint.textContent = !canVision ? "no mmproj found for this model"
      : mmproj ? mmproj.split("/").pop() : (m && m.mmproj ? m.mmproj.split("/").pop() : "");
    el.visionHint.title = mmproj || (m && m.mmproj) || "";

    const mtpOn = $("f-specType").value === "draft-mtp";
    el.mtpChk.checked = mtpOn;
    if (mtpOn) {
      el.mtpHint.textContent = `depth ${$("f-draftNMax").value || 0}`;
    } else {
      let hint = m && m.mtp ? "embedded MTP head found" : "no embedded MTP head in this model";
      if (m && m.mtpDraft) hint += "; draft sidecar available";
      el.mtpHint.textContent = hint;
    }

    // Advanced shows detail settings only for features that are switched on.
    el.sectVision.hidden = mmproj === "";
    el.sectSpec.hidden = specOff;
  }

  function onVisionToggle() {
    const m = selectedModel();
    $("f-mmproj").value = el.visionChk.checked ? (m && m.mmproj) || "" : "";
    if (!el.visionChk.checked) $("f-noMMProjOffload").checked = false;
    onFormChange();
  }

  function onMtpToggle() {
    $("f-specType").value = el.mtpChk.checked ? "draft-mtp" : "none";
    if (el.mtpChk.checked && !(parseInt($("f-draftNMax").value, 10) > 0)) {
      $("f-draftNMax").value = 2;
    }
    onFormChange();
  }

  function onNglAllToggle() {
    const ngl = $("f-gpuLayers");
    if (el.nglAll.checked) {
      if (ngl.value.trim() !== "all") lastNgl = ngl.value.trim() || lastNgl;
      ngl.value = "all";
    } else {
      ngl.value = lastNgl || "99";
    }
    onFormChange();
  }

  function onFormChange() {
    syncSpecState();
    call("UpdateConfig", collectConfig());
  }

  function wireForm() {
    el.visionChk.addEventListener("change", onVisionToggle);
    el.mtpChk.addEventListener("change", onMtpToggle);
    el.nglAll.addEventListener("change", onNglAllToggle);
    [el.tabBasic, el.tabAdvanced].forEach((panel) => {
      // Only real config fields (data-key) trigger a sync. The Vision / MTP /
      // "all" checkboxes are views over those fields and have their own
      // handlers; reacting to their `input` event here would reset them to
      // the old derived state before their `change` handler runs.
      ["input", "change"].forEach((evt) =>
        panel.addEventListener(evt, (e) => {
          if (e.target && e.target.dataset && e.target.dataset.key) onFormChange();
        })
      );
    });
  }

  // ---- render ----

  const ST_WORD = {
    starting: "STARTING",
    stopping: "STOPPING",
    running: "RUNNING",
    stopped: "STOPPED",
  };

  const ST_CLASS = {
    starting: ["st-starting", "st-word-starting"],
    stopping: ["st-stopping", "st-word-stopping"],
    running: ["st-running", "st-word-running"],
    stopped: ["st-stopped", "st-word-stopped"],
    crashed: ["st-crashed", "st-word-crashed"],
  };

  function parseUptime(uptime) {
    const m = /^(\d+):(\d{2}):(\d{2})$/.exec(String(uptime || ""));
    if (!m) return 0;
    return Number(m[1]) * 3600 + Number(m[2]) * 60 + Number(m[3]);
  }

  function fmtUptime(total) {
    const h = Math.floor(total / 3600);
    const m = Math.floor((total % 3600) / 60);
    const s = total % 60;
    const p = (n) => String(n).padStart(2, "0");
    return `${p(h)}:${p(m)}:${p(s)}`;
  }

  function renderStatus() {
    const status = snap.status || "stopped";
    const busy = status === "starting" || status === "stopping";
    const cl = ST_CLASS[status] || ST_CLASS.stopped;
    el.statusHead.className = cl[0];
    el.statusDot.className = busy ? "dot pulse" : "dot";
    el.statusWord.className = cl[1];
    el.statusWord.textContent =
      status === "crashed" ? `CRASHED exit ${snap.crashExit}` : ST_WORD[status] || status.toUpperCase();

    if (status !== lastStatus && !logsPinned && (status === "starting" || status === "crashed")) {
      setLogsCollapsed(false, false);
    }
    lastStatus = status;

    const srv = snap.server || {};
    el.kv.pid.textContent = srv.pid || "—";
    el.kv.listen.textContent = srv.listen || "—";
    el.kv.wslip.textContent = srv.wslIp || "—";
    el.kv.model.textContent = srv.model || "—";
    el.kv.context.textContent = srv.context || "—";
    el.kv.gpulayers.textContent = srv.gpuLayers || "—";
    el.kv.tps.textContent = srv.tokPerSec || "—";
    el.kv.tps.title = srv.tokPerSecTip || "";
    el.kv.vram.textContent = srv.vram || "—";
    el.kv.cpuram.textContent = srv.cpuRam || "—";
    el.kv.uptime.textContent = srv.uptime || "—";

    el.startBtn.disabled =
      status === "running" || status === "starting" || status === "stopping";
    el.stopBtn.disabled = status !== "running";
  }

  // Category drill-down: browsePath is the folder being viewed (e.g.
  // ["Qwen","3.8"]). Folders = next category level, items = models placed
  // exactly here. Uncategorized models live under "Uncategorized".
  let browsePath = null;
  let lastSelected = -2;
  const UNCAT = "Uncategorized";
  const catOf = (m) => (m.category && m.category.length ? m.category : [UNCAT]);
  const hasPrefix = (c, p) => p.every((x, i) => c[i] === x);

  function renderModels() {
    const models = snap.models || [];
    // Jump to the selected model's folder when the selection changes.
    if (browsePath === null || lastSelected !== snap.selectedModel) {
      const sm = models[snap.selectedModel];
      if (browsePath === null || sm) browsePath = sm ? catOf(sm) : [];
      lastSelected = snap.selectedModel;
    }

    el.crumbs.innerHTML = "";
    const addCrumb = (label, path) => {
      if (el.crumbs.childNodes.length) el.crumbs.appendChild(document.createTextNode(" › "));
      const b = document.createElement("button");
      b.type = "button";
      b.textContent = label;
      b.addEventListener("click", () => { browsePath = path; renderModels(); });
      el.crumbs.appendChild(b);
    };
    addCrumb("All", []);
    browsePath.forEach((c, i) => addCrumb(c, browsePath.slice(0, i + 1)));

    el.modelList.innerHTML = "";
    const folders = new Map();
    const here = [];
    models.forEach((m, i) => {
      const c = catOf(m);
      if (!hasPrefix(c, browsePath)) return;
      if (c.length > browsePath.length) {
        const name = c[browsePath.length];
        folders.set(name, (folders.get(name) || 0) + 1);
      } else {
        here.push(i);
      }
    });
    // Empty folders (created in the UI) exist only in snap.categories.
    (snap.categories || []).forEach((c) => {
      if (c.length > browsePath.length && hasPrefix(c, browsePath)) {
        const name = c[browsePath.length];
        if (!folders.has(name)) folders.set(name, 0);
      }
    });
    folders.forEach((count, name) => {
      const li = document.createElement("li");
      li.className = "folder";
      const label = document.createElement("span");
      label.textContent = "▸ " + name;
      const chip = document.createElement("span");
      chip.className = "chip";
      chip.textContent = count;
      li.appendChild(label);
      li.appendChild(chip);
      const fpath = browsePath.concat(name);
      li.appendChild(menuButton((x, y) => openFolderMenu(x, y, fpath)));
      li.addEventListener("contextmenu", (e) => { e.preventDefault(); openFolderMenu(e.clientX, e.clientY, fpath); });
      li.addEventListener("click", () => { browsePath = fpath; renderModels(); });
      el.modelList.appendChild(li);
    });
    here.forEach((i) => {
      const model = models[i];
      const li = document.createElement("li");
      li.title = model.path || "";
      const name = document.createElement("span");
      name.textContent = model.display || model.path || "—";
      li.appendChild(name);
      [model.mmproj ? "vision" : "", model.mtp ? "MTP" : "", model.mtpDraft ? "MTP-draft" : ""].forEach((t) => {
        if (!t) return;
        const chip = document.createElement("span");
        chip.className = "chip";
        chip.textContent = t;
        li.appendChild(chip);
      });
      li.appendChild(menuButton((x, y) => openModelMenu(x, y, model)));
      li.addEventListener("contextmenu", (e) => { e.preventDefault(); openModelMenu(e.clientX, e.clientY, model); });
      if (i === snap.selectedModel) li.classList.add("selected");
      li.addEventListener("click", () => call("SelectModel", i));
      el.modelList.appendChild(li);
    });
  }

  function renderPreset() {
    const names = snap.presetNames || [];
    el.presetSel.innerHTML = "";
    names.forEach((name) => {
      const opt = document.createElement("option");
      opt.value = name;
      opt.textContent = name;
      el.presetSel.appendChild(opt);
    });
    const selected = names[snap.selectedPreset];
    if (selected) el.presetSel.value = selected;
  }

  function renderConfigPath() {
    let path = snap.configPath || "configs/—.json";
    if (snap.dirty) path += " *";
    el.configPath.textContent = path;
    el.configPath.title = path;
  }

  const DOC_LABEL = { ok: "OK", warn: "WARN", fail: "FAIL", info: "INFO", skip: "SKIP" };

  function renderDoctor() {
    const rep = snap.doctor || {};
    const checks = rep.checks || [];
    el.doctorList.innerHTML = "";
    checks.forEach((c) => {
      const row = document.createElement("div");
      row.className = "doc-row doc-" + (c.status || "info");
      const head = document.createElement("div");
      head.className = "doc-head";
      const badge = document.createElement("span");
      badge.className = "doc-badge";
      badge.textContent = DOC_LABEL[c.status] || String(c.status || "").toUpperCase();
      const title = document.createElement("span");
      title.className = "doc-title";
      title.textContent = c.title || c.id || "";
      head.appendChild(badge);
      head.appendChild(title);
      row.appendChild(head);
      if (c.detail) {
        const d = document.createElement("div");
        d.className = "doc-detail";
        d.textContent = c.detail;
        row.appendChild(d);
      }
      if (c.fix) {
        const f = document.createElement("pre");
        f.className = "doc-fix";
        f.textContent = c.fix;
        row.appendChild(f);
      }
      el.doctorList.appendChild(row);
    });
    el.doctorSummary.textContent = rep.summary || "";

    const bad = checks.filter((c) => c.status === "fail" || c.status === "warn");
    if (bad.length) {
      el.doctorBanner.innerHTML = "";
      const title = document.createElement("div");
      title.className = "doc-banner-title";
      title.textContent = "Environment check: " + (rep.summary || "");
      el.doctorBanner.appendChild(title);
      bad.forEach((c) => {
        const line = document.createElement("div");
        line.className = "doc-banner-line doc-" + c.status;
        line.textContent = (DOC_LABEL[c.status] || c.status) + ": " + (c.title || "") +
          (c.detail ? " — " + c.detail : "");
        el.doctorBanner.appendChild(line);
      });
      const hint = document.createElement("div");
      hint.className = "doc-banner-hint";
      hint.textContent = "Open the llama.cpp tab for fixes, then Re-check environment.";
      el.doctorBanner.appendChild(hint);
      el.doctorBanner.classList.remove("hidden");
    } else {
      el.doctorBanner.innerHTML = "";
      el.doctorBanner.classList.add("hidden");
    }
  }

  // ---- download / onboarding ----

  function fmtBytes(n) {
    n = Number(n) || 0;
    if (n < 1024) return n + " B";
    const u = ["KiB", "MiB", "GiB", "TiB"];
    let i = -1;
    do { n /= 1024; i++; } while (n >= 1024 && i < u.length - 1);
    return n.toFixed(1) + " " + u[i];
  }

  const FIT_LABEL = { gpu: "fits GPU", hybrid: "GPU + RAM", cpu: "CPU", none: "too large" };
  const famName = (e) => (e.name || e.id || "").replace(/\s*\(.*\)\s*$/, "");

  // One catalog row: name, size, tags and a Download button. Curated entries name
  // their exact file and never pull sidecars they did not ask for (vision
  // projector, MTP draft).
  function catalogRow(r, opts) {
    const e = r.entry || {};
    const row = document.createElement("div");
    row.className = "dl-row dl-" + (r.fit || "cpu");
    const left = document.createElement("div");
    const title = document.createElement("div");
    title.className = "dl-name";
    if (opts && opts.quantTitle) {
      const q = document.createElement("span");
      q.className = "chip chip-quant";
      q.textContent = e.quant || "";
      title.appendChild(q);
      title.appendChild(document.createTextNode(" " + (e.file || "").split("/").pop()));
    } else {
      title.textContent = (e.starter ? "★ " : "") + (e.name || e.id);
    }
    const fit = document.createElement("span");
    fit.className = "dl-fit";
    fit.textContent = FIT_LABEL[r.fit] || r.fit;
    title.appendChild(fit);
    const meta = document.createElement("div");
    meta.className = "dl-meta";
    const tags = [e.vision ? "vision" : "", e.mtp ? "MTP" : ""].filter(Boolean).join(" · ");
    meta.textContent = `${fmtBytes(e.bytes)}${tags ? " · " + tags : ""} · ${r.why || ""}`;
    left.appendChild(title);
    left.appendChild(meta);
    row.appendChild(left);
    const btn = document.createElement("button");
    btn.className = "tiny";
    btn.textContent = "Download";
    btn.addEventListener("click", () => {
      el.dlRef.value = e.repo || "";
      call("StartDownload", e.repo || "", e.file || "", e.mmproj || "none", "none");
      el.catModal.hidden = true;
      el.dlModal.hidden = false;
    });
    row.appendChild(btn);
    return row;
  }

  function renderDownloadCatalog(cat) {
    const picks = (cat && cat.picks) || [];
    const info = (cat && cat.info) || {};
    el.dlSummary.textContent =
      `Recommended for this machine — VRAM ${fmtBytes((cat.vramMB || 0) * 1048576)}, RAM ${fmtBytes((cat.ramMB || 0) * 1048576)}`;
    el.dlList.innerHTML = "";
    picks.forEach((p) => {
      const role = document.createElement("div");
      role.className = "dl-role";
      role.textContent = p.label;
      el.dlList.appendChild(role);
      el.dlList.appendChild(catalogRow(p.rec));
    });
    el.dlHidden.textContent = `${cat.fitting || 0} of ${cat.total || 0} models fit this machine` +
      (info.updated ? ` · catalog ${info.updated}` : "");
  }

  function openFullCatalog() {
    const a = app();
    if (!a) return;
    el.catModal.hidden = false;
    el.catList.textContent = "Loading…";
    Promise.resolve(a.GetFullCatalog()).then((c) => {
      const info = c.info || {};
      el.catSummary.textContent = `VRAM ${fmtBytes((c.vramMB || 0) * 1048576)}, RAM ${fmtBytes((c.ramMB || 0) * 1048576)} · ` +
        `${(c.entries || []).length} files · catalog ${info.updated || "?"} (${info.source || "embedded"})`;
      el.catList.innerHTML = "";
      let last = null;
      (c.entries || []).forEach((r) => {
        const fam = r.entry.family || r.entry.id;
        if (fam !== last) {
          last = fam;
          const h = document.createElement("div");
          h.className = "cat-fam";
          h.textContent = famName(r.entry);
          const small = document.createElement("small");
          small.textContent = `${r.entry.paramsB}B${r.entry.moe ? " MoE" : ""}`;
          h.appendChild(small);
          el.catList.appendChild(h);
        }
        el.catList.appendChild(catalogRow(r, { quantTitle: true }));
      });
    }).catch((err) => { el.catList.textContent = "error: " + (err && err.message ? err.message : err); });
  }

  // ---- Hugging Face repo browser: paste a link, pick a file ----

  let lookupSeq = 0;

  function findRepoFiles() {
    const ref = el.dlRef.value.trim();
    if (!ref) { el.dlMsg.textContent = "Enter a repo (owner/name) or a Hugging Face link."; return; }
    const a = app();
    if (!a) return;
    const seq = ++lookupSeq;
    el.dlMsg.textContent = "Looking up files…";
    el.dlFind.disabled = true;
    Promise.resolve(a.ListRepoFiles(ref)).then((l) => {
      if (seq !== lookupSeq) return;
      el.dlMsg.textContent = "";
      renderRepoFiles(l);
    }).catch((err) => {
      if (seq !== lookupSeq) return;
      el.dlFiles.hidden = true;
      el.dlMsg.textContent = err && err.message ? err.message : String(err);
    }).finally(() => { el.dlFind.disabled = false; });
  }

  function sizeLabel(n) { return fmtBytes(n); }

  function renderRepoFiles(l) {
    el.dlFiles.hidden = false;
    el.dlCatalogBox.open = false; // the list below is the focus now
    const models = l.models || [], mms = l.mmprojs || [], mtps = l.mtpDrafts || [];
    el.dlFilesHead.textContent = `${l.repo} — ${models.length} model file(s)`;
    el.dlOptions.innerHTML = "";
    el.dlFileList.innerHTML = "";
    if (!models.length) {
      el.dlFileList.textContent = "No .gguf model files in this repository.";
      return;
    }

    // Optional sidecars: a switch + (when several) a picker.
    const extra = (kind, items, label, hint, defaultOn) => {
      if (!items.length) return null;
      const wrap = document.createElement("div");
      wrap.className = "dl-opt";
      const sw = document.createElement("input");
      sw.type = "checkbox";
      sw.checked = defaultOn;
      const txt = document.createElement("div");
      txt.className = "dl-opt-text";
      txt.innerHTML = "";
      const t = document.createElement("div");
      t.className = "dl-opt-title";
      t.textContent = label;
      const h = document.createElement("div");
      h.className = "field-hint";
      h.textContent = hint;
      txt.appendChild(t);
      txt.appendChild(h);
      const sel = document.createElement("select");
      items.forEach((f) => {
        const o = document.createElement("option");
        o.value = f.path;
        o.textContent = `${f.name} · ${sizeLabel(f.size)}`;
        sel.appendChild(o);
      });
      sel.hidden = items.length === 1;
      wrap.appendChild(sw);
      wrap.appendChild(txt);
      wrap.appendChild(sel);
      el.dlOptions.appendChild(wrap);
      sw.addEventListener("change", refreshButtons);
      sel.addEventListener("change", refreshButtons);
      return { sw, sel, items };
    };
    const vision = extra("vision", mms, "Include the vision projector (images)", "Needed for image input; adds the size shown.", true);
    const mtp = extra("mtp", mtps, "Include the MTP draft sidecar", "For speculative decoding; most builds do not need it.", false);

    const sidecarBytes = () => {
      let n = 0;
      [vision, mtp].forEach((x) => {
        if (x && x.sw.checked) n += (x.items.find((f) => f.path === x.sel.value) || {}).size || 0;
      });
      return n;
    };
    const buttons = [];
    function refreshButtons() {
      buttons.forEach(({ btn, m }) => { btn.textContent = `Download · ${sizeLabel(m.size + sidecarBytes())}`; });
    }

    models.forEach((m) => {
      const row = document.createElement("div");
      row.className = "dl-row";
      const left = document.createElement("div");
      const title = document.createElement("div");
      title.className = "dl-name";
      if (m.quant) {
        const q = document.createElement("span");
        q.className = "chip chip-quant";
        q.textContent = m.quant;
        title.appendChild(q);
      }
      title.appendChild(document.createTextNode(" " + m.name));
      const meta = document.createElement("div");
      meta.className = "dl-meta";
      meta.textContent = m.parts > 1 ? `${sizeLabel(m.size)} · ${m.parts} parts — multi-part files are not downloadable yet` : sizeLabel(m.size);
      left.appendChild(title);
      left.appendChild(meta);
      row.appendChild(left);
      const btn = document.createElement("button");
      btn.className = "tiny";
      btn.disabled = m.parts > 1;
      btn.addEventListener("click", () => {
        call("StartDownload", l.repo, m.path,
          vision && vision.sw.checked ? vision.sel.value : "none",
          mtp && mtp.sw.checked ? mtp.sel.value : "none");
      });
      row.appendChild(btn);
      buttons.push({ btn, m });
      el.dlFileList.appendChild(row);
    });
    refreshButtons();
  }

  function openDownload() {
    el.dlModal.hidden = false;
    const a = app();
    if (!a) return;
    Promise.resolve(a.GetCatalog()).then(renderDownloadCatalog).catch(() => {});
    // Pull the newest catalog from the repository, then redraw (best effort).
    Promise.resolve(a.RefreshCatalog()).then(renderDownloadCatalog).catch(() => {});
    Promise.resolve(a.GetHFTokenStatus()).then((s) => {
      el.dlTokenMsg.textContent = s && s.set ? "token configured (" + s.source + ")" : "";
    }).catch(() => {});
  }

  function renderDownload() {
    const d = (snap && snap.download) || {};
    const active = d.active || d.status === "downloading" || d.status === "verifying";
    if (!active && d.status !== "done" && d.status !== "error" && d.status !== "cancelled") {
      el.dlProgress.hidden = true;
      return;
    }
    el.dlProgress.hidden = false;
    const pct = Math.max(0, Math.min(100, d.percent || 0));
    el.dlBarInner.style.width = pct.toFixed(1) + "%";
    let txt = `${d.status || ""} ${d.file || ""} — ${pct.toFixed(1)}%  ${fmtBytes(d.received)}/${fmtBytes(d.total)}  ${fmtBytes(d.speedBps)}/s`;
    if (d.error) txt = "error: " + d.error;
    else if (d.message) txt = d.message;
    el.dlProgressText.textContent = txt;
    el.dlCancel.hidden = !active;
  }

  // Status-bar badge: downloads keep running when the dialog is closed.
  let badgeDoneUntil = 0;
  let badgeLastStatus = "";

  function renderDownloadBadge() {
    const d = (snap && snap.download) || {};
    const active = d.active || d.status === "downloading" || d.status === "verifying";
    const now = Date.now();
    if (badgeLastStatus !== d.status) {
      if (d.status === "done" || d.status === "error") badgeDoneUntil = now + 20000;
      badgeLastStatus = d.status;
    }
    const b = el.dlBadge;
    b.className = "dl-badge";
    b.innerHTML = "";
    const name = (d.file || "").split("/").pop();
    if (active) {
      const pct = Math.max(0, Math.min(100, d.percent || 0));
      b.textContent = `⬇ ${name} · ${pct.toFixed(0)}%` + (d.speedBps ? ` · ${fmtBytes(d.speedBps)}/s` : "");
      const bar = document.createElement("span");
      bar.className = "bar";
      bar.style.width = pct + "%";
      b.appendChild(bar);
      b.hidden = false;
    } else if (d.status === "error" && now < badgeDoneUntil) {
      b.className = "dl-badge error";
      b.textContent = "⚠ download failed — click for details";
      b.hidden = false;
    } else if (d.status === "done" && now < badgeDoneUntil) {
      b.className = "dl-badge done";
      b.textContent = `✓ ${name} installed`;
      b.hidden = false;
    } else {
      b.hidden = true;
    }
  }
  setInterval(renderDownloadBadge, 2000); // lets the done/error badge expire

  function render() {
    if (!snap) return;
    renderDownloadBadge();
    if (!bootPrinted) {
      bootPrinted = true;
      (snap.bootLines || []).forEach((line) => appendLog(line));
    }
    uptimeSec = parseUptime((snap.server || {}).uptime);
    renderStatus();
    renderModels();
    renderPreset();
    renderStatusBar();
    renderConfigPath();
    fillForm();
    el.cmdline.textContent = snap.commandLine || "";
    el.cmdline.title = `${snap.commandLine || ""} — preview only, never executed`;
    el.vramEstimate.textContent = snap.vramEstimate || "";
    renderVramBar(snap.vramEstimate || "");
    if (snap.warning) {
      el.warnBanner.textContent = "⚠ " + snap.warning;
      el.warnBanner.classList.remove("hidden");
    } else {
      el.warnBanner.textContent = "";
      el.warnBanner.classList.add("hidden");
    }
    renderDoctor();
    renderDownload();
    el.firstRun.hidden = !snap.firstRun;
    if (snap.firstRun) {
      el.frPath.textContent = snap.userConfigPath || "";
      if (!el.frBin.value) el.frBin.value = snap.serverBin || "";
      if (!el.frModels.value) el.frModels.value = snap.modelsDir || "";
    }
  }

  // ---- tabs ----

  function showTab(which) {
    const tabs = {
      basic: [el.tabBtnBasic, el.tabBasic],
      advanced: [el.tabBtnAdvanced, el.tabAdvanced],
      llama: [el.tabBtnLlama, el.tabLlama],
    };
    Object.entries(tabs).forEach(([name, [btn, panel]]) => {
      btn.classList.toggle("active", name === which);
      panel.classList.toggle("active", name === which);
    });
    if (which === "llama") refreshLlama();
  }

  // ---- library editing: context menu, rename / move / category dialogs ----

  const ctxMenu = $("ctxMenu");

  function menuButton(open) {
    const b = document.createElement("button");
    b.type = "button";
    b.className = "row-menu";
    b.textContent = "⋯";
    b.title = "Actions";
    b.addEventListener("click", (e) => {
      e.stopPropagation();
      const r = b.getBoundingClientRect();
      open(r.left, r.bottom + 2);
    });
    return b;
  }

  function closeCtxMenu() { ctxMenu.hidden = true; ctxMenu.innerHTML = ""; }

  function showCtxMenu(x, y, items) {
    ctxMenu.innerHTML = "";
    items.forEach((it) => {
      if (it === "-") { ctxMenu.appendChild(document.createElement("hr")); return; }
      const b = document.createElement("button");
      b.type = "button";
      b.textContent = it.label;
      b.disabled = !!it.disabled;
      if (it.danger) b.className = "danger";
      b.addEventListener("click", () => { closeCtxMenu(); it.run(); });
      ctxMenu.appendChild(b);
    });
    ctxMenu.hidden = false;
    // keep the menu inside the window
    const w = ctxMenu.offsetWidth, h = ctxMenu.offsetHeight;
    ctxMenu.style.left = Math.max(8, Math.min(x, window.innerWidth - w - 8)) + "px";
    ctxMenu.style.top = Math.max(8, Math.min(y, window.innerHeight - h - 8)) + "px";
  }

  function openModelMenu(x, y, model) {
    const categorized = model.category && model.category.length > 0;
    showCtxMenu(x, y, [
      { label: "Rename…", run: () => renameModelDialog(model) },
      { label: "Move to category…", run: () => moveModelDialog(model) },
      { label: "Remove from category", disabled: !categorized, run: () => libCall("SetModelCategory", model.path, []) },
    ]);
  }

  function openFolderMenu(x, y, path) {
    showCtxMenu(x, y, [
      { label: "Rename…", run: () => renameFolderDialog(path) },
      { label: "New subcategory…", run: () => newFolderDialog(path) },
      "-",
      { label: "Delete category", danger: true, run: () => deleteFolder(path) },
    ]);
  }

  document.addEventListener("click", (e) => { if (!ctxMenu.hidden && !ctxMenu.contains(e.target)) closeCtxMenu(); });
  document.addEventListener("keydown", (e) => { if (e.key === "Escape") { closeCtxMenu(); closeEdit(); } });
  window.addEventListener("blur", closeCtxMenu);
  $("modelList").addEventListener("scroll", closeCtxMenu);

  // Run a bound library method; failures surface in the log (and as a rejection
  // for dialogs that want to show the message inline).
  function libCall(method, ...args) {
    return Promise.resolve(app()[method](...args)).catch((err) => {
      const msg = err && err.message ? err.message : String(err);
      appendLog("error: " + msg);
      throw new Error(msg);
    });
  }

  const edit = {
    modal: $("editModal"), title: $("editTitle"), text: $("editText"),
    fields: $("editFields"), msg: $("editMsg"), ok: $("editOk"), cancel: $("editCancel"),
  };
  let editSubmit = null;

  function closeEdit() { edit.modal.hidden = true; editSubmit = null; }

  // spec: {title, text, submit, fields:[{key,label,hint,value,options?,mono?}], onSubmit(values)}
  function openEdit(spec) {
    edit.title.textContent = spec.title;
    edit.text.textContent = spec.text || "";
    edit.text.hidden = !spec.text;
    edit.msg.textContent = "";
    edit.ok.textContent = spec.submit || "Save";
    edit.fields.innerHTML = "";
    const inputs = {};
    spec.fields.forEach((f) => {
      const wrap = document.createElement("div");
      wrap.className = "field";
      const lab = document.createElement("label");
      lab.className = "fl";
      lab.textContent = f.label;
      let inp;
      if (f.options) {
        inp = document.createElement("select");
        f.options.forEach((o) => {
          const op = document.createElement("option");
          op.value = o.value; op.textContent = o.label;
          inp.appendChild(op);
        });
      } else {
        inp = document.createElement("input");
        inp.spellcheck = false;
        if (f.placeholder) inp.placeholder = f.placeholder;
      }
      inp.value = f.value || "";
      lab.htmlFor = inp.id = "edit-" + f.key;
      wrap.appendChild(lab);
      wrap.appendChild(inp);
      if (f.hint) {
        const h = document.createElement("div");
        h.className = "field-hint";
        h.textContent = f.hint;
        wrap.appendChild(h);
      }
      edit.fields.appendChild(wrap);
      inputs[f.key] = inp;
      if (f.onChange) inp.addEventListener("change", () => f.onChange(inputs));
    });
    const values = () => Object.fromEntries(Object.entries(inputs).map(([k, v]) => [k, v.value]));
    editSubmit = async () => {
      edit.ok.disabled = true;
      try {
        await spec.onSubmit(values());
        closeEdit();
      } catch (e) {
        edit.msg.textContent = e && e.message ? e.message : String(e);
      } finally {
        edit.ok.disabled = false;
      }
    };
    edit.modal.hidden = false;
    const first = Object.values(inputs)[0];
    if (first) { first.focus(); if (first.select) first.select(); }
    edit.fields.onkeydown = (e) => { if (e.key === "Enter" && editSubmit) { e.preventDefault(); editSubmit(); } };
  }

  edit.ok.addEventListener("click", () => { if (editSubmit) editSubmit(); });
  edit.cancel.addEventListener("click", closeEdit);
  edit.modal.addEventListener("mousedown", (e) => { if (e.target === edit.modal) closeEdit(); });

  function renameModelDialog(model) {
    openEdit({
      title: "Rename model",
      text: "Only the display name and alias change; the .gguf file is not touched.",
      fields: [
        { key: "name", label: "Display name", value: model.display, hint: "Shown in the model list. Leave empty to use the file name." },
        { key: "alias", label: "Alias", value: model.alias, hint: "Used for --alias, the CLI and the API. Its configs/<alias>.json is renamed with it; clients using the old alias must be updated." },
      ],
      onSubmit: (v) => libCall("RenameModel", model.path, v.name, v.alias),
    });
  }

  // All category paths that exist (model placements, their ancestors, empty folders).
  function allCategoryPaths() {
    const seen = new Map();
    const add = (c) => {
      for (let i = 1; i <= c.length; i++) seen.set(c.slice(0, i).join("\u001f"), c.slice(0, i));
    };
    (snap.models || []).forEach((m) => { if (m.category && m.category.length) add(m.category); });
    (snap.categories || []).forEach(add);
    return Array.from(seen.values()).sort((a, b) => a.join("›").localeCompare(b.join("›")));
  }

  const parsePath = (text) => text.split(/[/›]/).map((x) => x.trim()).filter(Boolean);

  function moveModelDialog(model) {
    const NEW = "\u0001new";
    const cur = (model.category || []).join("\u001f");
    const options = [{ value: "", label: "Uncategorized" }]
      .concat(allCategoryPaths().map((p) => ({ value: p.join("\u001f"), label: p.join(" › ") })))
      .concat([{ value: NEW, label: "New category…" }]);
    openEdit({
      title: "Move to category",
      text: model.display,
      submit: "Move",
      fields: [
        { key: "cat", label: "Category", value: cur, options, onChange: (inp) => {
            inp.newpath.parentElement.hidden = inp.cat.value !== NEW;
            if (!inp.newpath.parentElement.hidden) inp.newpath.focus();
          } },
        { key: "newpath", label: "New category path", placeholder: "Qwen / 3.8", hint: "Separate levels with / or ›. Missing levels are created." },
      ],
      onSubmit: (v) => {
        const path = v.cat === NEW ? parsePath(v.newpath) : (v.cat ? v.cat.split("\u001f") : []);
        if (v.cat === NEW && !path.length) return Promise.reject(new Error("Enter a category path."));
        return libCall("SetModelCategory", model.path, path);
      },
    });
    const np = $("edit-newpath");
    if (np) np.parentElement.hidden = true;
  }

  function newFolderDialog(parent) {
    openEdit({
      title: "New category",
      text: parent.length ? "Inside " + parent.join(" › ") : "At the top level",
      submit: "Create",
      fields: [{ key: "name", label: "Name", placeholder: "e.g. Coding models" }],
      onSubmit: (v) => libCall("CreateCategory", parent.concat(v.name.trim())).then(() => {
        browsePath = parent.concat(v.name.trim());
        renderModels();
      }),
    });
  }

  function renameFolderDialog(path) {
    openEdit({
      title: "Rename category",
      text: path.join(" › "),
      fields: [{ key: "name", label: "New name", value: path[path.length - 1] }],
      onSubmit: (v) => libCall("RenameCategory", path, v.name).then(() => {
        const nn = v.name.trim();
        if (browsePath && hasPrefix(browsePath, path)) {
          browsePath = browsePath.slice();
          browsePath[path.length - 1] = nn;
        }
      }),
    });
  }

  function deleteFolder(path) {
    const n = (snap.models || []).filter((m) => m.category && hasPrefix(m.category, path)).length;
    const parent = path.length > 1 ? path.slice(0, -1).join(" › ") : UNCAT;
    const msg = `Delete the category "${path.join(" › ")}"?` +
      (n ? `\n\nIts ${n} model(s) are NOT deleted; they move to "${parent}".` : "");
    if (!window.confirm(msg)) return;
    libCall("DeleteCategory", path).then(() => {
      if (browsePath && hasPrefix(browsePath, path)) browsePath = path.slice(0, -1);
      renderModels();
    }).catch(() => {});
  }

  // ---- VRAM bar: parses the "(85%)" part of the estimate text ----

  function renderVramBar(text) {
    const m = /\((\d+)%\)/.exec(text);
    const fill = $("vramFill");
    const pct = m ? Math.min(100, parseInt(m[1], 10)) : 0;
    fill.style.width = pct + "%";
    fill.className = pct >= 90 ? "bad" : pct >= 75 ? "warn" : "";
    $("vramBar").style.visibility = m ? "visible" : "hidden";
  }

  // ---- theme: auto (system) -> light -> dark ----

  const THEMES = ["auto", "light", "dark"];

  function applyTheme(t) {
    const root = document.documentElement;
    if (t === "light" || t === "dark") root.dataset.theme = t;
    else delete root.dataset.theme;
    $("themeBtn").title = "Theme: " + t + " (click to change)";
    try { localStorage.setItem("theme", t); } catch (e) { /* storage unavailable */ }
  }

  function initTheme() {
    let t = "auto";
    try { t = localStorage.getItem("theme") || "auto"; } catch (e) { /* storage unavailable */ }
    if (!THEMES.includes(t)) t = "auto";
    applyTheme(t);
    $("themeBtn").addEventListener("click", () => {
      const cur = document.documentElement.dataset.theme || "auto";
      applyTheme(THEMES[(THEMES.indexOf(cur) + 1) % THEMES.length]);
    });
  }

  // ---- logs collapse ----

  let logsPinned = false; // true once the user toggled logs themselves

  function setLogsCollapsed(collapsed, persist) {
    el.logsGroup.classList.toggle("collapsed", collapsed);
    el.logsTitle.textContent = (collapsed ? "▸ " : "▾ ") + "Logs";
    if (persist === false) return;
    logsPinned = true;
    try { localStorage.setItem("logsCollapsed", collapsed ? "1" : "0"); } catch (e) { /* storage unavailable */ }
  }

  function initLogsCollapse() {
    // Collapsed by default to give the config form room; it opens by itself when
    // a server starts or crashes (unless the user has chosen a state before).
    let saved = null;
    try { saved = localStorage.getItem("logsCollapsed"); } catch (e) { /* storage unavailable */ }
    if (saved === null) setLogsCollapsed(true, false);
    else { logsPinned = true; setLogsCollapsed(saved === "1", false); }
    const toggle = () => setLogsCollapsed(!el.logsGroup.classList.contains("collapsed"));
    el.logsTitle.addEventListener("click", toggle);
    el.logsTitle.addEventListener("keydown", (e) => {
      if (e.key === "Enter" || e.key === " ") { e.preventDefault(); toggle(); }
    });
  }

  // ---- status bar ----

  function renderStatusBar() {
    const b = (snap.config || {}).basic || {};
    const a = (snap.config || {}).advanced || {};
    const m = selectedModel();
    const srv = snap.server || {};
    const status = snap.status || "stopped";
    const parts = [];
    parts.push(status === "running" && srv.model && srv.model !== "—" ? `running: ${srv.model}` : status);
    parts.push(`model: ${m ? m.display : "—"}`);
    parts.push(`preset: ${(snap.presetNames || [])[snap.selectedPreset] || "—"}${snap.dirty ? "*" : ""}`);
    parts.push(`ctx ${b.context || "—"}`);
    parts.push(`ngl ${b.gpuLayers || "—"}`);
    parts.push(`vision ${a.mmproj ? "on" : "off"}`);
    parts.push(`MTP ${a.specType === "draft-mtp" ? "on" : "off"}`);
    el.statusInfo.textContent = parts.join("  ·  ");
    el.statusInfo.dataset.status = status;
  }

  // ---- llama.cpp tab ----

  let llamaWasUpdating = false;

  function llamaActive() {
    return el.tabLlama.classList.contains("active");
  }

  function refreshLlama(method) {
    const a = app();
    if (!a) return;
    Promise.resolve(a[method || "GetLlamaPanel"]())
      .then(renderLlama)
      .catch((err) => appendLog(`error: ${err && err.message ? err.message : err}`));
  }

  function kvRow(k, v, cls) {
    const kd = document.createElement("div");
    kd.className = "k";
    kd.textContent = k;
    const vd = document.createElement("div");
    vd.className = "v" + (cls ? " " + cls : "");
    vd.textContent = v || "—";
    vd.title = v || "";
    el.llamaInfo.appendChild(kd);
    el.llamaInfo.appendChild(vd);
  }

  function renderLlama(p) {
    if (!p) return;
    const i = p.info || {};
    el.llamaInfo.innerHTML = "";
    if (i.err) kvRow("Error", i.err, "warn");
    kvRow("Version", i.version);
    kvRow("Commit", `${i.commit || "—"}  (${i.describe || "—"})`);
    kvRow("Commit date", i.commitDate);
    kvRow("Last change", i.subject);
    kvRow("Branch", i.branch + (i.dirty ? "  (local modifications)" : ""), i.dirty ? "warn" : "");
    kvRow("Upstream", i.behind < 0 ? "unknown — press Check for updates"
      : i.behind === 0 ? `up to date as of last fetch (${i.fetched || "never"})`
      : `${i.behind} commit(s) behind (fetched ${i.fetched || "never"})`,
      i.behind > 0 ? "warn" : "");
    kvRow("Binary", i.binary);
    kvRow("Built", i.builtAt);
    kvRow("Build", `${i.buildDir || "—"} · ${i.buildType || "—"} · CUDA ${i.cudaOn ? "on" : "off"} · arch ${i.cudaArch || "—"}`);
    kvRow("GPU", i.gpu);
    kvRow("Repo", i.repoDir);

    const procs = p.procs || [];
    el.llamaProcs.innerHTML = "";
    if (!procs.length) {
      const none = document.createElement("div");
      none.className = "none";
      none.textContent = "none running";
      el.llamaProcs.appendChild(none);
    }
    procs.forEach((pr) => {
      const box = document.createElement("div");
      box.className = "proc";
      const head = document.createElement("div");
      head.textContent = `pid ${pr.pid} · up ${pr.uptime} · ${pr.listen || "no port"} · ${pr.model || "?"}` +
        (pr.managed ? " · started by LlamaHerd" : " · external");
      const cmd = document.createElement("code");
      cmd.textContent = pr.cmd;
      cmd.title = pr.cmd;
      box.appendChild(head);
      box.appendChild(cmd);
      el.llamaProcs.appendChild(box);
    });

    el.llamaLast.textContent = p.updating ? "update running — output in Logs" : (p.lastUpdate || "");
    el.llamaUpdate.disabled = p.updating || procs.length > 0;
    el.llamaUpdate.title = procs.length > 0 ? "stop llama-server first (or Force stop all)" : "";
    el.llamaCancel.hidden = !p.updating;
    el.llamaKill.disabled = procs.length === 0;
    el.llamaCheck.disabled = p.updating;
  }

  // ---- boot ----

  function onState(next) {
    snap = next;
    render();
    if (llamaActive() && llamaWasUpdating !== !!snap.llamaUpdating) refreshLlama();
    llamaWasUpdating = !!snap.llamaUpdating;
  }

  function init() {
    if (!app()) {
      appendLog("bindings unavailable (run inside wails)");
      return;
    }

    wireForm();
    initLogsCollapse();
    initTheme();

    $("newCatBtn").addEventListener("click", () => newFolderDialog(browsePath || []));
    el.presetSel.addEventListener("change", () => call("SelectPreset", el.presetSel.value));
    el.saveBtn.addEventListener("click", () => call("SaveConfig"));
    el.loadBtn.addEventListener("click", () => call("LoadConfig"));
    el.startBtn.addEventListener("click", () => call("StartServer"));
    el.stopBtn.addEventListener("click", () => call("StopServer"));
    el.frSave.addEventListener("click", () => {
      const a = app();
      if (!a) return;
      el.frMsg.textContent = "";
      Promise.resolve(a.SetPaths(el.frBin.value.trim(), el.frModels.value.trim()))
        .catch((err) => { el.frMsg.textContent = "error: " + (err && err.message ? err.message : err); });
    });

    el.tabBtnBasic.addEventListener("click", () => showTab("basic"));
    el.tabBtnAdvanced.addEventListener("click", () => showTab("advanced"));
    el.tabBtnLlama.addEventListener("click", () => showTab("llama"));
    el.llamaRefresh.addEventListener("click", () => refreshLlama());
    el.doctorRefresh.addEventListener("click", () => call("RunDoctor"));
    el.dlOpen.addEventListener("click", openDownload);
    el.dlClose.addEventListener("click", () => { el.dlModal.hidden = true; });
    el.dlBrowse.addEventListener("click", openFullCatalog);
    el.catClose.addEventListener("click", () => { el.catModal.hidden = true; });
    el.catModal.addEventListener("mousedown", (e) => { if (e.target === el.catModal) el.catModal.hidden = true; });
    el.dlBadge.addEventListener("click", openDownload);
    el.dlFind.addEventListener("click", findRepoFiles);
    el.dlRef.addEventListener("keydown", (e) => { if (e.key === "Enter") { e.preventDefault(); findRepoFiles(); } });
    // Pasting a link looks it up right away.
    el.dlRef.addEventListener("paste", () => setTimeout(() => { if (el.dlRef.value.includes("/")) findRepoFiles(); }, 0));
    el.dlCancel.addEventListener("click", () => call("CancelDownload"));
    el.dlTokenSave.addEventListener("click", () => {
      const a = app();
      if (!a) return;
      el.dlTokenMsg.textContent = "";
      Promise.resolve(a.SetHFToken(el.dlToken.value.trim()))
        .then(() => { el.dlToken.value = ""; el.dlTokenMsg.textContent = "token saved"; })
        .catch((err) => { el.dlTokenMsg.textContent = "error: " + (err && err.message ? err.message : err); });
    });
    el.dlTokenClear.addEventListener("click", () => {
      const a = app();
      if (!a) return;
      Promise.resolve(a.SetHFToken("")).then(() => { el.dlTokenMsg.textContent = "token cleared"; });
    });
    el.frDownload.addEventListener("click", () => { el.dlModal.hidden = false; openDownload(); });
    el.llamaCheck.addEventListener("click", () => refreshLlama("CheckLlamaUpdates"));
    el.llamaUpdate.addEventListener("click", () => call("UpdateLlamaCpp"));
    el.llamaCancel.addEventListener("click", () => call("CancelLlamaUpdate"));
    el.llamaKill.addEventListener("click", () => {
      if (window.confirm("Force stop ALL llama-server processes, including ones LlamaHerd did not start?")) {
        Promise.resolve(app().ForceStopLlama()).then(() => refreshLlama());
      }
    });
    // Keep the process list live while the tab is open.
    setInterval(() => { if (llamaActive()) refreshLlama(); }, 4000);

    const rt = runtime();
    if (rt && rt.EventsOn) {
      rt.EventsOn("state", onState);
      rt.EventsOn("log", (line) => appendLog(line));
    }

    // Boot state via GetState; if an event snapshot already arrived first,
    // the stale GetState result is discarded.
    Promise.resolve(app().GetState())
      .then((first) => {
        if (!first || snap) return;
        snap = first;
        render();
      })
      .catch((err) =>
        appendLog(`error: ${err && err.message ? err.message : err}`)
      );
  }

  setInterval(() => {
    if (snap && snap.status === "running") {
      uptimeSec += 1;
      el.kv.uptime.textContent = fmtUptime(uptimeSec);
    }
  }, 1000);

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
