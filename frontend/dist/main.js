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
    dlFile: $("dlFile"),
    dlStart: $("dlStart"),
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

    const srv = snap.server || {};
    el.kv.pid.textContent = srv.pid || "—";
    el.kv.listen.textContent = srv.listen || "—";
    el.kv.wslip.textContent = srv.wslIp || "—";
    el.kv.model.textContent = srv.model || "—";
    el.kv.context.textContent = srv.context || "—";
    el.kv.gpulayers.textContent = srv.gpuLayers || "—";
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
      li.addEventListener("click", () => { browsePath = browsePath.concat(name); renderModels(); });
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

  function renderDownloadCatalog(cat) {
    const list = (cat && cat.entries) || [];
    el.dlSummary.textContent =
      `Recommended for this machine — VRAM ${fmtBytes((cat.vramMB || 0) * 1048576)}, RAM ${fmtBytes((cat.ramMB || 0) * 1048576)}`;
    el.dlList.innerHTML = "";
    list.forEach((r) => {
      const e = r.entry || {};
      const row = document.createElement("div");
      row.className = "dl-row dl-" + (r.fit || "too-big");
      const left = document.createElement("div");
      const title = document.createElement("div");
      title.className = "dl-name";
      title.textContent = (e.starter ? "★ " : "") + (e.name || e.id);
      const meta = document.createElement("div");
      meta.className = "dl-meta";
      const tags = [e.vision ? "vision" : "", e.mtp ? "MTP" : ""].filter(Boolean).join(" · ");
      meta.textContent = `${fmtBytes(e.bytes)} · ~${Math.round((e.context || 0) / 1000)}k ctx`
        + `${tags ? " · " + tags : ""} · ${r.why || ""}`;
      left.appendChild(title);
      left.appendChild(meta);
      row.appendChild(left);
      const btn = document.createElement("button");
      btn.className = "tiny";
      btn.textContent = "Download";
      btn.addEventListener("click", () => { el.dlRef.value = e.repo || ""; call("StartDownload", e.repo || "", "", ""); });
      row.appendChild(btn);
      el.dlList.appendChild(row);
    });
  }

  function openDownload() {
    el.dlModal.hidden = false;
    const a = app();
    if (!a) return;
    Promise.resolve(a.GetCatalog()).then(renderDownloadCatalog).catch(() => {});
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

  function render() {
    if (!snap) return;
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

  // ---- logs collapse ----

  function setLogsCollapsed(collapsed) {
    el.logsGroup.classList.toggle("collapsed", collapsed);
    el.logsTitle.textContent = (collapsed ? "▸ " : "▾ ") + "Logs";
    try { localStorage.setItem("logsCollapsed", collapsed ? "1" : "0"); } catch (e) { /* storage unavailable */ }
  }

  function initLogsCollapse() {
    let saved = false;
    try { saved = localStorage.getItem("logsCollapsed") === "1"; } catch (e) { /* storage unavailable */ }
    setLogsCollapsed(saved);
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
    el.dlStart.addEventListener("click", () => {
      call("StartDownload", el.dlRef.value.trim(), el.dlFile.value.trim(), "");
    });
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
