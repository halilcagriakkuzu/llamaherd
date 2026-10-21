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
    presetMenuBtn: $("presetMenuBtn"),
    autoFitBtn: $("autoFitBtn"),
    validation: $("validation"),
    chartVram: $("chartVram"),
    chartRam: $("chartRam"),
    chartSpeed: $("chartSpeed"),
    chartVramVal: $("chartVramVal"),
    chartRamVal: $("chartRamVal"),
    chartSpeedVal: $("chartSpeedVal"),
    metricsHint: $("metricsHint"),
    detailsModal: $("detailsModal"),
    detailsTitle: $("detailsTitle"),
    detailsBody: $("detailsBody"),
    detailsCopy: $("detailsCopy"),
    detailsOpenSource: $("detailsOpenSource"),
    detailsClose: $("detailsClose"),
    detailsMsg: $("detailsMsg"),
    saveBtn: $("saveBtn"),
    loadBtn: $("loadBtn"),
    startBtn: $("startBtn"),
    stopBtn: $("stopBtn"),
    restartBtn: $("restartBtn"),
    chatBtn: $("chatBtn"),
    modelSearch: $("modelSearch"),
    modelFilter: $("modelFilter"),
    rescanBtn: $("rescanBtn"),
    logFilter: $("logFilter"),
    logErrors: $("logErrors"),
    logAuto: $("logAuto"),
    logCopy: $("logCopy"),
    logDownload: $("logDownload"),
    logClear: $("logClear"),
    toasts: $("toasts"),
    moreBtn: $("moreBtn"),
    moreMenu: $("moreMenu"),
    moreDiag: $("moreDiag"),
    moreUpdate: $("moreUpdate"),
    morePaths: $("morePaths"),
    notifBtn: $("notifBtn"),
    notifPanel: $("notifPanel"),
    notifList: $("notifList"),
    notifCount: $("notifCount"),
    notifEmpty: $("notifEmpty"),
    notifClear: $("notifClear"),
    notifEnabled: $("notifEnabled"),
    notifReady: $("notifReady"),
    notifCrash: $("notifCrash"),
    notifDownload: $("notifDownload"),
    notifBenchmark: $("notifBenchmark"),
    notifSave: $("notifSave"),
    notifMsg: $("notifMsg"),
    pathBin: $("pathBin"),
    pathModels: $("pathModels"),
    pathBinBrowse: $("pathBinBrowse"),
    pathModelsBrowse: $("pathModelsBrowse"),
    pathSave: $("pathSave"),
    pathMsg: $("pathMsg"),
    pathBinDetect: $("pathBinDetect"),
    frBinDetect: $("frBinDetect"),
    diagCopy: $("diagCopy"),
    frBinBrowse: $("frBinBrowse"),
    frModelsBrowse: $("frModelsBrowse"),
    dlDisk: $("dlDisk"),
    configPath: $("configPath"),
    tabBtnBasic: $("tabBtnBasic"),
    tabBtnAdvanced: $("tabBtnAdvanced"),
    tabLlama: $("tabLlama"),
    viewLlama: $("viewLlama"),
    viewConfigBtn: $("viewConfigBtn"),
    viewPerfBtn: $("viewPerfBtn"),
    viewLlamaBtn: $("viewLlamaBtn"),
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
    procBanner: $("procBanner"),
    procBannerTitle: $("procBannerTitle"),
    procBannerMsg: $("procBannerMsg"),
    procBannerLog: $("procBannerLog"),
    procRetry: $("procRetry"),
    procStop: $("procStop"),
    benchRun: $("benchRun"),
    benchClear: $("benchClear"),
    benchExport: $("benchExport"),
    benchHint: $("benchHint"),
    benchTable: $("benchTable"),
    benchBody: $("benchBody"),
    benchEmpty: $("benchEmpty"),
    importModal: $("importModal"),
    importModelsDir: $("importModelsDir"),
    importModelPath: $("importModelPath"),
    importModelBrowse: $("importModelBrowse"),
    importList: $("importList"),
    importAllowDupRow: $("importAllowDupRow"),
    importAllowDup: $("importAllowDup"),
    importHasMMProj: $("importHasMMProj"),
    importMMProjRow: $("importMMProjRow"),
    importMMProjPath: $("importMMProjPath"),
    importMMProjBrowse: $("importMMProjBrowse"),
    importMove: $("importMove"),
    importMsg: $("importMsg"),
    importCancel: $("importCancel"),
    importOk: $("importOk"),
    importOpen: $("importOpen"),
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
    dlGated: $("dlGated"),
    dlGatedText: $("dlGatedText"),
    dlGatedOpen: $("dlGatedOpen"),
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
    dlPause: $("dlPause"),
    dlResume: $("dlResume"),
    dlRetry: $("dlRetry"),
    dlQueueWrap: $("dlQueueWrap"),
    dlQueue: $("dlQueue"),
    dlCancel: $("dlCancel"),
    frDownload: $("frDownload")
  };

  let snap = null;
  let bootPrinted = false;
  let uptimeSec = 0;
  let lastStatus = "";
  let lastWarn = "";
  let pathsDirty = false;

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
    Promise.resolve(a[method](...args)).catch((err) => {
      const msg = err && err.message ? err.message : String(err);
      appendLog(`error: ${msg}`);
      toast(msg, "error");
    });
  }

  function runtime() {
    return window.runtime || null;
  }

  // ---- notifications: toasts also land in the session notification center ----

  const NOTIF_MAX = 100;
  let notifications = [];
  try {
    const raw = sessionStorage.getItem("llamaherd.notifications");
    if (raw) notifications = JSON.parse(raw) || [];
  } catch (e) { /* storage unavailable */ }

  function saveNotifications() {
    try { sessionStorage.setItem("llamaherd.notifications", JSON.stringify(notifications)); } catch (e) { /* storage unavailable */ }
  }

  function notify(message, kind) {
    const text = String(message || "").trim();
    if (!text) return;
    notifications.push({ at: Date.now(), message: text, kind: kind || "" });
    if (notifications.length > NOTIF_MAX) notifications = notifications.slice(-NOTIF_MAX);
    saveNotifications();
    renderNotifications();
  }

  function renderNotifications() {
    if (!el.notifList) return;
    el.notifList.innerHTML = "";
    el.notifEmpty.hidden = notifications.length > 0;
    el.notifCount.hidden = notifications.length === 0;
    el.notifCount.textContent = notifications.length > 99 ? "99+" : String(notifications.length);
    notifications.slice().reverse().forEach((n) => {
      const li = document.createElement("li");
      li.className = "notif-item" + (n.kind ? " " + n.kind : "");
      const time = document.createElement("time");
      time.dateTime = new Date(n.at).toISOString();
      time.textContent = new Date(n.at).toLocaleTimeString();
      const msg = document.createElement("span");
      msg.textContent = n.message;
      li.append(time, msg);
      el.notifList.appendChild(li);
    });
  }

  function toggleNotifications(open) {
    const show = open === undefined ? el.notifPanel.hidden : open;
    el.notifPanel.hidden = !show;
    el.notifBtn.setAttribute("aria-expanded", String(!!show));
    if (show) renderNotifications();
  }

  function initNotifications() {
    renderNotifications();
    el.notifBtn.addEventListener("click", (e) => { e.stopPropagation(); toggleNotifications(); });
    el.notifClear.addEventListener("click", () => { notifications = []; saveNotifications(); renderNotifications(); });
    document.addEventListener("click", (e) => {
      if (!el.notifPanel.hidden && !el.notifPanel.contains(e.target) && !el.notifBtn.contains(e.target)) toggleNotifications(false);
      if (!el.moreMenu.hidden && !el.moreMenu.contains(e.target) && !el.moreBtn.contains(e.target)) toggleMore(false);
    });
  }

  function toggleMore(open) {
    const show = open === undefined ? el.moreMenu.hidden : open;
    el.moreMenu.hidden = !show;
    el.moreBtn.setAttribute("aria-expanded", String(!!show));
    if (show) {
      const first = el.moreMenu.querySelector("button");
      if (first) first.focus();
    }
  }

  function initMoreMenu() {
    el.moreBtn.addEventListener("click", (e) => { e.stopPropagation(); toggleMore(); });
    el.moreMenu.addEventListener("keydown", (e) => {
      if (e.key === "Escape") { toggleMore(false); el.moreBtn.focus(); }
    });
    el.moreDiag.addEventListener("click", () => { toggleMore(false); el.diagCopy.click(); });
    el.moreUpdate.addEventListener("click", () => { toggleMore(false); el.llamaUpdate.click(); });
    el.morePaths.addEventListener("click", () => {
      toggleMore(false);
      showView("llama");
      const paths = $("sectPaths");
      if (paths) paths.scrollIntoView({ block: "nearest" });
      el.pathBin.focus();
    });
    $("themeBtn").addEventListener("click", () => toggleMore(false));
  }

  function notifyStatus(status, message) {
    if (status !== lastStatus && (status === "crashed" || status === "failed")) notify(message, "error");
  }

  function loadNotificationSettings() {
    const a = app();
    if (!a || typeof a.NotificationSettings !== "function") return;
    Promise.resolve(a.NotificationSettings()).then((s) => {
      if (!s) return;
      el.notifEnabled.checked = !!s.enabled;
      el.notifReady.checked = !!s.serverReady;
      el.notifCrash.checked = !!s.serverCrash;
      el.notifDownload.checked = !!s.download;
      el.notifBenchmark.checked = !!s.benchmark;
    }).catch(() => {});
  }

  function saveNotificationSettings() {
    const a = app();
    if (!a || typeof a.SetNotificationSettings !== "function") return;
    const s = {
      enabled: el.notifEnabled.checked,
      serverReady: el.notifReady.checked,
      serverCrash: el.notifCrash.checked,
      download: el.notifDownload.checked,
      benchmark: el.notifBenchmark.checked,
    };
    el.notifMsg.textContent = "";
    Promise.resolve(a.SetNotificationSettings(s))
      .then(() => { el.notifMsg.textContent = "saved"; toast("notification settings saved", "ok"); })
      .catch((err) => { el.notifMsg.textContent = "error: " + (err && err.message ? err.message : err); });
  }

  function toast(message, kind) {
    notify(message, kind);
    if (!el.toasts) return;
    const t = document.createElement("div");
    t.className = "toast" + (kind ? " " + kind : "");
    t.textContent = String(message);
    el.toasts.appendChild(t);
    setTimeout(() => t.remove(), kind === "error" ? 8000 : 4000);
  }

  // ---- log ----

  const LOG_MAX = 5000;
  const logLines = [];
  let logAutoScroll = true;

  const logIsError = (text) =>
    /(error|fail|fatal|panic|exception|crashed|exit status [1-9])/i.test(text);

  function logMatches(text) {
    const q = (el.logFilter.value || "").trim().toLowerCase();
    if (q && !text.toLowerCase().includes(q)) return false;
    if (el.logErrors.checked && !logIsError(text)) return false;
    return true;
  }

  function renderLogs() {
    const frag = document.createDocumentFragment();
    logLines.forEach((l) => {
      if (!logMatches(l.text)) return;
      const div = document.createElement("div");
      if (l.cls) div.className = l.cls;
      div.textContent = l.text;
      frag.appendChild(div);
    });
    el.logView.innerHTML = "";
    el.logView.appendChild(frag);
    if (logAutoScroll) el.logView.scrollTop = el.logView.scrollHeight;
  }

  function appendLog(line) {
    const text = String(line);
    logLines.push({ text, cls: text.startsWith("server:") ? "log-server" : "" });
    if (logLines.length > LOG_MAX) logLines.splice(0, logLines.length - LOG_MAX);
    if (!logMatches(text)) return;
    const div = document.createElement("div");
    if (text.startsWith("server:")) div.className = "log-server";
    div.textContent = text;
    el.logView.appendChild(div);
    while (el.logView.childNodes.length > LOG_MAX) el.logView.removeChild(el.logView.firstChild);
    if (logAutoScroll) el.logView.scrollTop = el.logView.scrollHeight;
  }

  function setLogAuto(on) {
    logAutoScroll = on;
    el.logAuto.textContent = "Auto-scroll: " + (on ? "on" : "off");
    if (on) el.logView.scrollTop = el.logView.scrollHeight;
  }

  function wireLogs() {
    let timer = null;
    const redraw = () => { clearTimeout(timer); timer = setTimeout(renderLogs, 120); };
    el.logFilter.addEventListener("input", redraw);
    el.logErrors.addEventListener("change", renderLogs);
    el.logAuto.addEventListener("click", () => setLogAuto(!logAutoScroll));
    el.logView.addEventListener("scroll", () => {
      const nearBottom = el.logView.scrollHeight - el.logView.scrollTop - el.logView.clientHeight < 24;
      if (!nearBottom && logAutoScroll) setLogAuto(false);
    });
    el.logClear.addEventListener("click", () => { logLines.length = 0; el.logView.innerHTML = ""; });
    el.logCopy.addEventListener("click", () => {
      const text = logLines.filter((l) => logMatches(l.text)).map((l) => l.text).join("\n");
      if (!navigator.clipboard) { toast("clipboard unavailable", "warn"); return; }
      navigator.clipboard.writeText(text).then(() => toast("log copied", "ok"), () => toast("could not copy log", "error"));
    });
    el.logDownload.addEventListener("click", () => {
      const text = logLines.filter((l) => logMatches(l.text)).map((l) => l.text).join("\n");
      const blob = new Blob([text], { type: "text/plain" });
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = "llamaherd-" + new Date().toISOString().replace(/[:.]/g, "-") + ".log";
      a.click();
      setTimeout(() => URL.revokeObjectURL(a.href), 1000);
    });
  }

  // ---- form ----

  function pull(cfg, keys) {
    const out = {};
    keys.forEach((k) => { out[k] = cfg && cfg[k] !== undefined ? cfg[k] : ""; });
    return out;
  }

  const BASIC_KEYS = [
    "model", "alias", "host", "port", "context", "gpuLayers", "parallelSlots",
    "flashAttention", "kCache", "vCache", "jinja", "reasoningFmt", "reasoningBudget",
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
      let v = values[node.dataset.key];
      if (node.dataset.key === "reasoningBudget" && Number(v) === 0) v = "";
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
    failed: "NOT RESPONDING",
  };

  const ST_CLASS = {
    starting: ["st-starting", "st-word-starting"],
    stopping: ["st-stopping", "st-word-stopping"],
    running: ["st-running", "st-word-running"],
    stopped: ["st-stopped", "st-word-stopped"],
    crashed: ["st-crashed", "st-word-crashed"],
    failed: ["st-failed", "st-word-failed"],
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

    if (status !== lastStatus) {
      notifyStatus(status, status === "crashed"
        ? `server crashed (exit ${snap.crashExit})`
        : "server is not responding");
    }
    lastStatus = status;
    renderProcBanner();

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
      status === "running" || status === "starting" || status === "stopping" || status === "failed";
    el.stopBtn.disabled = status !== "running" && status !== "starting" && status !== "failed";
    const active = status === "running" || status === "starting" || status === "failed";
    el.restartBtn.hidden = !active;
    el.restartBtn.disabled = status === "stopping" || status === "starting";
    el.chatBtn.disabled = status !== "running";
    el.startBtn.title = active
      ? "llama-server is already running — use Restart to apply form changes"
      : "Start llama-server";
  }

  // The "failed" and "crashed" states explain themselves with the server's own
  // log tail; the user can Retry (restart) or Stop from here.
  function renderProcBanner() {
    const status = (snap && snap.status) || "stopped";
    const bad = status === "failed" || status === "crashed";
    el.procBanner.classList.toggle("hidden", !bad);
    if (!bad) {
      el.procBannerLog.textContent = "";
      return;
    }
    el.procBannerTitle.textContent = status === "failed"
      ? "Server is not responding"
      : `Server crashed (exit ${snap.crashExit})`;
    el.procBannerMsg.textContent = status === "failed"
      ? "It never answered /health. The process is still running: check the last lines, then Retry or Stop."
      : "The last server output is shown below.";
    const tail = (snap.recentLog || []).slice(-20).join("\n");
    el.procBannerLog.textContent = tail;
    el.procBannerLog.hidden = tail === "";
  }

  // Category drill-down: browsePath is the folder being viewed (e.g.
  // ["Qwen","3.8"]). Folders = next category level, items = models placed
  // exactly here. Uncategorized models live under "Uncategorized".
  let browsePath = null;
  let lastSelected = -2;
  let modelQuery = "";
  let modelFilter = "all";
  const UNCAT = "Uncategorized";
  const catOf = (m) => (m.category && m.category.length ? m.category : [UNCAT]);
  const hasPrefix = (c, p) => p.every((x, i) => c[i] === x);

  function modelMatches(m) {
    const q = modelQuery.trim().toLowerCase();
    if (q) {
      const hay = [m.display, m.alias, m.path, (m.category || []).join(" ")].join(" ").toLowerCase();
      if (!hay.includes(q)) return false;
    }
    switch (modelFilter) {
      case "favorites": return !!m.favorite;
      case "recent": return (m.lastUsed || 0) > 0;
      case "vision": return !!m.mmproj;
      case "mtp": return !!(m.mtp || m.mtpDraft);
      case "uncategorized": return !(m.category && m.category.length);
      default: return true;
    }
  }

  function modelRow(i) {
    const model = (snap.models || [])[i];
    const li = document.createElement("li");
    li.title = model.path || "";
    const name = document.createElement("span");
    if (model.favorite) {
      const star = document.createElement("span");
      star.className = "fav-star";
      star.textContent = "★";
      li.appendChild(star);
    }
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
    return li;
  }

  function renderModels() {
    const models = snap.models || [];
    const searching = modelQuery.trim() !== "" || modelFilter !== "all";
    // Jump to the selected model's folder when the selection changes.
    if (!searching && (browsePath === null || lastSelected !== snap.selectedModel)) {
      const sm = models[snap.selectedModel];
      if (browsePath === null || sm) browsePath = sm ? catOf(sm) : [];
      lastSelected = snap.selectedModel;
    }

    el.crumbs.innerHTML = "";
    if (searching) {
      const match = [];
      models.forEach((m, i) => { if (!m.isMMProj && modelMatches(m)) match.push(i); });
      if (modelFilter === "recent") {
        match.sort((x, y) => (models[y].lastUsed || 0) - (models[x].lastUsed || 0));
      }
      const b = document.createElement("span");
      b.className = "crumbs-info";
      b.textContent = `${match.length} match(es)`;
      el.crumbs.appendChild(b);
      el.modelList.innerHTML = "";
      if (!match.length) {
        const empty = document.createElement("li");
        empty.className = "model-empty";
        empty.textContent = "No models match this search.";
        el.modelList.appendChild(empty);
        return;
      }
      match.forEach((i) => el.modelList.appendChild(modelRow(i)));
      return;
    }

    const addCrumb = (label, path) => {
      if (el.crumbs.childNodes.length) el.crumbs.appendChild(document.createTextNode(" › "));
      const b = document.createElement("button");
      b.type = "button";
      b.textContent = label;
      b.addEventListener("click", () => { browsePath = path; renderModels(); });
      el.crumbs.appendChild(b);
    };
    addCrumb("All", []);
    (browsePath || []).forEach((c, i) => addCrumb(c, browsePath.slice(0, i + 1)));

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
    here.forEach((i) => el.modelList.appendChild(modelRow(i)));
  }

  const NEW_PRESET = "\u0001new";

  function renderPreset() {
    const names = snap.presetNames || [];
    el.presetSel.innerHTML = "";
    names.forEach((name) => {
      const opt = document.createElement("option");
      opt.value = name;
      opt.textContent = name;
      el.presetSel.appendChild(opt);
    });
    const sep = document.createElement("option");
    sep.disabled = true;
    sep.textContent = "──────────";
    el.presetSel.appendChild(sep);
    const add = document.createElement("option");
    add.value = NEW_PRESET;
    add.textContent = "<new preset…>";
    el.presetSel.appendChild(add);
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
      hint.textContent = "Open the llama.cpp view for fixes, then Re-check environment.";
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
  const baseName = (p) => String(p || "").split(/[\\/]/).pop();

  function installedModel(e) {
    const file = baseName(e.file);
    if (!file) return null;
    return (snap.models || []).find((m) => m.path && baseName(m.path) === file) || null;
  }

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
    const installed = installedModel(e);
    meta.textContent = `${fmtBytes(e.bytes)}${tags ? " · " + tags : ""} · ${r.why || ""}`;
    if (installed) {
      const chip = document.createElement("span");
      chip.className = "chip";
      chip.textContent = "installed";
      title.appendChild(chip);
    }
    left.appendChild(title);
    left.appendChild(meta);
    row.appendChild(left);
    if (installed) {
      const open = document.createElement("button");
      open.className = "tiny";
      open.textContent = "Open";
      open.title = "Reveal the installed file";
      open.addEventListener("click", () => call("RevealModel", installed.path));
      row.appendChild(open);
    }
    const btn = document.createElement("button");
    btn.className = "tiny";
    btn.textContent = installed ? "Re-download" : "Download";
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
    if (el.dlDisk) {
      const free = fmtBytes(cat.diskFree || 0);
      el.dlDisk.textContent = cat.modelsDir
        ? `Models: ${cat.modelsDir} · ${cat.diskFree ? free + " free" : "free space unknown"}`
        : "";
    }
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

  let lastRepo = "";
  function renderRepoFiles(l) {
    lastRepo = l.repo || "";
    el.dlFiles.hidden = false;
    el.dlCatalogBox.open = false; // the list below is the focus now
    const models = l.models || [], mms = l.mmprojs || [], mtps = l.mtpDrafts || [];
    el.dlFilesHead.textContent = `${l.repo} — ${models.length} model file(s)` + (l.license ? ` · licence ${l.license}` : "");
    el.dlGated.hidden = !(l.gated === "auto" || l.gated === "manual");
    el.dlGatedText.textContent = l.gated === "manual"
      ? "Gated repository: sign in on Hugging Face and request access (the authors approve requests manually), then save a token below before downloading."
      : "Gated repository: sign in on Hugging Face and accept the licence, then save a token below before downloading.";
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
      meta.textContent = m.parts > 1
        ? `${sizeLabel(m.size)} · ${m.parts} shards`
        : sizeLabel(m.size);
      left.appendChild(title);
      left.appendChild(meta);
      row.appendChild(left);
      const btn = document.createElement("button");
      btn.className = "tiny";
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
    const paused = d.paused || d.status === "paused";
    const queue = d.queue || [];
    if (!active && !paused && d.status !== "done" && d.status !== "error" && d.status !== "cancelled") {
      el.dlProgress.hidden = true;
    } else {
      el.dlProgress.hidden = false;
      const pct = Math.max(0, Math.min(100, d.percent || 0));
      el.dlBarInner.style.width = pct.toFixed(1) + "%";
      let txt = `${d.status || ""} ${d.file || ""} — ${pct.toFixed(1)}%  ${fmtBytes(d.received)}/${fmtBytes(d.total)}  ${fmtBytes(d.speedBps)}/s`;
      if (d.shardCount > 1) txt = `shard ${d.shardIndex || 1}/${d.shardCount} · ` + txt;
      if (d.error) txt = "error: " + d.error;
      else if (d.message) txt = d.message;
      el.dlProgressText.textContent = txt;
    }
    el.dlPause.hidden = !active;
    el.dlResume.hidden = !paused;
    el.dlRetry.hidden = !(d.status === "error" || d.status === "cancelled");
    el.dlCancel.hidden = !(active || paused);
    renderDownloadQueue(queue);
  }

  function renderDownloadQueue(queue) {
    el.dlQueue.innerHTML = "";
    el.dlQueueWrap.hidden = queue.length === 0;
    queue.forEach((q, i) => {
      const row = document.createElement("div");
      row.className = "dl-row dl-queue-row";
      const left = document.createElement("div");
      const name = document.createElement("div");
      name.className = "dl-name";
      name.textContent = q.file || q.repo;
      const meta = document.createElement("div");
      meta.className = "dl-meta";
      meta.textContent = `${i + 1}. ${q.repo} · ${q.status || "queued"}${q.attempts ? " · retry" : ""}`;
      left.append(name, meta);
      row.appendChild(left);
      const rm = document.createElement("button");
      rm.className = "tiny";
      rm.textContent = "Remove";
      rm.title = "Remove this waiting download";
      rm.addEventListener("click", () => call("RemoveQueuedDownload", q.id));
      row.appendChild(rm);
      el.dlQueue.appendChild(row);
    });
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
      if (d.status === "done") notify(`download finished: ${(d.file || "").split("/").pop()}`, "ok");
      if (d.status === "error") notify(`download failed: ${d.error || (d.file || "").split("/").pop()}`, "error");
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

  // ---- local model import ----

  function toggleImportMMProj() {
    el.importMMProjRow.hidden = !el.importHasMMProj.checked;
    if (el.importHasMMProj.checked && !el.importMMProjPath.value) el.importMMProjBrowse.focus();
  }

  // Paths are kept in one input, separated by ";" or new lines.
  function importPaths() {
    return el.importModelPath.value.split(/\s*[;\n]\s*/).map((p) => p.trim()).filter(Boolean);
  }

  function renderImportRows(rows) {
    el.importList.textContent = "";
    el.importList.hidden = rows.length === 0;
    rows.forEach((r) => {
      const row = document.createElement("div");
      row.className = "dl-row import-" + r.state;
      const text = document.createElement("div");
      const name = document.createElement("div");
      name.className = "dl-name";
      name.textContent = (r.path || "").split(/[\\/]/).pop();
      name.title = r.path || "";
      const meta = document.createElement("div");
      meta.className = "dl-meta";
      meta.textContent = r.note;
      text.append(name, meta);
      row.appendChild(text);
      if (r.size) {
        const size = document.createElement("span");
        size.className = "dl-meta";
        size.textContent = fmtBytes(r.size);
        row.appendChild(size);
      }
      el.importList.appendChild(row);
    });
  }

  // Preview: which files are new, already in the library or unusable.
  let importCheckSeq = 0;
  function checkImport() {
    const a = app();
    const paths = importPaths();
    const seq = ++importCheckSeq;
    renderImportRows([]);
    el.importAllowDupRow.hidden = true;
    if (!a || !paths.length || typeof a.CheckImport !== "function") return;
    el.importMsg.textContent = "Checking for duplicates…";
    Promise.resolve(a.CheckImport(paths)).then((checks) => {
      if (seq !== importCheckSeq) return;
      checks = checks || [];
      el.importMsg.textContent = "";
      renderImportRows(checks.map((c) => ({
        path: c.path,
        size: c.size,
        state: c.error ? "error" : c.duplicate ? "duplicate" : "new",
        note: c.error || (c.duplicate ? "already imported as " + (c.existingAlias || c.existingPath) : "new"),
      })));
      el.importAllowDupRow.hidden = !checks.some((c) => c.duplicate);
    }).catch((err) => {
      if (seq === importCheckSeq) el.importMsg.textContent = err && err.message ? err.message : String(err);
    });
  }

  function openImport(paths) {
    el.importModal.hidden = false;
    el.importMsg.textContent = "";
    el.importModelsDir.textContent = (snap && snap.modelsDir) || "";
    el.importHasMMProj.checked = false;
    el.importMMProjRow.hidden = true;
    el.importMMProjPath.value = "";
    el.importMove.checked = false;
    el.importAllowDup.checked = false;
    el.importModelPath.value = Array.isArray(paths) ? paths.join("; ") : "";
    checkImport();
    el.importModelBrowse.focus();
  }

  function closeImport() {
    importCheckSeq++;
    el.importModal.hidden = true;
    el.importOk.disabled = false;
  }

  function pickImportFiles() {
    const a = app();
    if (!a || typeof a.PickModelFiles !== "function") return;
    Promise.resolve(a.PickModelFiles()).then((files) => {
      if (!files || !files.length) return;
      el.importModelPath.value = importPaths().concat(files).join("; ");
      checkImport();
    }).catch((err) => toast(err && err.message ? err.message : String(err), "error"));
  }

  // Files dropped onto the Models panel open the dialog prefilled.
  function onFilesDropped(_x, _y, paths) {
    const ggufs = (paths || []).filter((p) => /\.gguf$/i.test(p));
    if (!ggufs.length) {
      toast("drop .gguf model files to import them", "error");
      return;
    }
    if (!el.importModal.hidden) {
      el.importModelPath.value = importPaths().concat(ggufs).join("; ");
      checkImport();
      return;
    }
    openImport(ggufs);
  }

  function doImport() {
    const a = app();
    if (!a) return;
    const paths = importPaths();
    if (!paths.length) {
      el.importMsg.textContent = "Choose one or more .gguf model files.";
      el.importModelBrowse.focus();
      return;
    }
    const mm = el.importHasMMProj.checked ? el.importMMProjPath.value.trim() : "";
    if (el.importHasMMProj.checked && !mm) {
      el.importMsg.textContent = "Choose the vision projector (.gguf) or untick the mmproj box.";
      el.importMMProjBrowse.focus();
      return;
    }
    importCheckSeq++;
    el.importMsg.textContent = "Importing… large files can take a while.";
    el.importOk.disabled = true;
    Promise.resolve(a.ImportModels(paths, mm, el.importMove.checked, el.importAllowDup.checked))
      .then((rep) => {
        const items = (rep && rep.items) || [];
        const count = (st) => items.filter((it) => it.status === st).length;
        const ok = count("imported"), dup = count("duplicate"), failed = items.length - ok - dup;
        const parts = [];
        if (ok) parts.push(ok + " imported");
        if (dup) parts.push(dup + " already in the library");
        if (failed) parts.push(failed + " failed");
        const summary = parts.join(", ") || "nothing imported";
        if (!failed) {
          el.importModelPath.value = "";
          closeImport();
          toast(summary, ok ? "ok" : "");
          return;
        }
        el.importMsg.textContent = summary;
        renderImportRows(items.map((it) => ({
          path: it.source,
          state: it.status,
          note: it.status === "imported" ? "imported as " + it.alias : it.message || it.status,
        })));
      })
      .catch((err) => {
        el.importMsg.textContent = err && err.message ? err.message : String(err);
      })
      .finally(() => { el.importOk.disabled = false; });
  }

  function render() {
    if (!snap) return;
    if (snap.ready === false) {
      el.statusInfo.textContent = "loading — scanning models and presets…";
      return;
    }
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
    renderValidation();
    renderCharts();
    renderBenchmarks();
    // Path fields belong to the llama.cpp tab; never overwrite what the user
    // is typing.
    if (!pathsDirty && document.activeElement !== el.pathBin) el.pathBin.value = snap.serverBin || "";
    if (!pathsDirty && document.activeElement !== el.pathModels) el.pathModels.value = snap.modelsDir || "";
    el.pathBin.title = snap.serverBin || "";
    el.pathModels.title = snap.modelsDir || "";
    el.cmdline.textContent = snap.commandLine || "";
    el.cmdline.title = `${snap.commandLine || ""} — preview only, never executed`;
    el.vramEstimate.textContent = snap.vramEstimate || "";
    renderVramBar(snap.vramEstimate || "");
    if (snap.warning) {
      el.warnBanner.textContent = "⚠ " + snap.warning;
      el.warnBanner.classList.remove("hidden");
      if (snap.warning !== lastWarn) toast(snap.warning, "warn");
    } else {
      el.warnBanner.textContent = "";
      el.warnBanner.classList.add("hidden");
    }
    lastWarn = snap.warning || "";
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
    };
    Object.entries(tabs).forEach(([name, [btn, panel]]) => {
      btn.classList.toggle("active", name === which);
      panel.classList.toggle("active", name === which);
    });
  }

  let currentView = "config";

  function showView(which) {
    if (!["config", "performance", "llama"].includes(which)) which = "config";
    currentView = which;
    document.getElementById("main").dataset.view = which;
    const btns = { config: el.viewConfigBtn, performance: el.viewPerfBtn, llama: el.viewLlamaBtn };
    Object.entries(btns).forEach(([name, btn]) => {
      btn.classList.toggle("active", name === which);
      if (name === which) btn.setAttribute("aria-current", "page");
      else btn.removeAttribute("aria-current");
    });
    if (which === "performance") {
      requestAnimationFrame(() => { if (snap) renderCharts(); });
      if (snap) renderBenchmarks();
    }
    if (which === "llama") { refreshLlama(); loadNotificationSettings(); }
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

  function copyText(text) {
    if (!navigator.clipboard) { toast("clipboard unavailable", "warn"); return; }
    navigator.clipboard.writeText(text).then(() => toast("copied", "ok"), () => toast("could not copy", "error"));
  }

  function deleteModel(model) {
    if (!window.confirm(`Delete the model file?\n\n${model.display}\n${model.path}\n\nThe .gguf file is removed from disk; its preset file is kept.`)) return;
    libCall("DeleteModel", model.path)
      .then(() => toast("model deleted", "ok"))
      .catch(() => {});
  }

  function openModelMenu(x, y, model) {
    const categorized = model.category && model.category.length > 0;
    showCtxMenu(x, y, [
      { label: "Details…", run: () => openModelDetails(model) },
      { label: model.favorite ? "Unpin favorite" : "Pin favorite", run: () => libCall("SetModelFavorite", model.path, !model.favorite).catch(() => {}) },
      { label: "Copy path", run: () => copyText(model.path) },
      { label: "Reveal in file manager", run: () => call("RevealModel", model.path) },
      "-",
      { label: "Rename…", run: () => renameModelDialog(model) },
      { label: "Move to category…", run: () => moveModelDialog(model) },
      { label: "Remove from category", disabled: !categorized, run: () => libCall("SetModelCategory", model.path, []) },
      "-",
      { label: "Delete model file…", danger: true, run: () => deleteModel(model) },
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
      toast(msg, "error");
      throw new Error(msg);
    });
  }

  const edit = {
    modal: $("editModal"), title: $("editTitle"), text: $("editText"),
    fields: $("editFields"), msg: $("editMsg"), ok: $("editOk"), cancel: $("editCancel"),
  };
  let editSubmit = null;
  let editCancel = null;

  function closeEdit() {
    edit.modal.hidden = true;
    editSubmit = null;
    const fn = editCancel;
    editCancel = null;
    if (fn) fn();
  }

  // spec: {title, text, submit, fields:[{key,label,hint,value,options?,mono?}], onSubmit(values), onCancel?}
  function openEdit(spec) {
    editCancel = spec.onCancel || null;
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
        editCancel = null;
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

  // ---- preset editing: new / rename / delete via the Config dropdown ----

  function currentPresetName() {
    const names = (snap && snap.presetNames) || [];
    return names[snap.selectedPreset] || "";
  }

  function newPresetDialog() {
    const alias = (snap && snap.alias) || "this model";
    openEdit({
      title: "New preset",
      text: `Saves the current form as a new preset for ${alias}.`,
      submit: "Create",
      fields: [{
        key: "name", label: "Preset name", placeholder: "e.g. fast, long-context",
        hint: "Must be unique within this model; existing presets are not overwritten.",
      }],
      onCancel: renderPreset,
      onSubmit: (v) => {
        const name = v.name.trim();
        if (!name) return Promise.reject(new Error("Enter a preset name."));
        return libCall("SavePresetAs", name);
      },
    });
  }

  function renamePresetDialog() {
    const old = currentPresetName();
    if (!old) return;
    openEdit({
      title: "Rename preset",
      text: `${(snap && snap.alias) || ""} › ${old}`,
      submit: "Rename",
      fields: [{ key: "name", label: "New name", value: old }],
      onSubmit: (v) => {
        const name = v.name.trim();
        if (!name) return Promise.reject(new Error("Enter a preset name."));
        if (name === old) return Promise.resolve();
        return libCall("RenamePreset", snap.alias, old, name);
      },
    });
  }

  function deletePreset() {
    const name = currentPresetName();
    const names = (snap && snap.presetNames) || [];
    if (!name || names.length <= 1) return;
    if (!window.confirm(`Delete the preset "${name}" from ${snap.alias}?\n\nThe other presets in its config file are kept.`)) return;
    call("RemovePreset", snap.alias, name);
  }

  function duplicatePresetDialog() {
    const old = currentPresetName();
    if (!old) return;
    openEdit({
      title: "Duplicate preset",
      text: `${(snap && snap.alias) || ""} › ${old}`,
      submit: "Create",
      fields: [{ key: "name", label: "New preset name", value: old + " copy" }],
      onSubmit: (v) => {
        const name = v.name.trim();
        if (!name) return Promise.reject(new Error("Enter a preset name."));
        return libCall("SavePresetAs", name);
      },
    });
  }

  function resetPreset() {
    const name = currentPresetName();
    if (!name) return;
    if (!window.confirm(`Reset the preset "${name}" to the auto-fitted model default?\n\nThe current values in this preset are overwritten.`)) return;
    libCall("ResetPreset", name).then(() => toast("preset reset", "ok")).catch(() => {});
  }

  function exportPresets() {
    const a = app();
    if (!a) return;
    Promise.resolve(a.ExportPresets()).then((data) => {
      const blob = new Blob([data], { type: "application/json" });
      const link = document.createElement("a");
      link.href = URL.createObjectURL(blob);
      link.download = ((snap && snap.alias) || "llamaherd") + "-presets.json";
      link.click();
      setTimeout(() => URL.revokeObjectURL(link.href), 1000);
      toast("presets exported", "ok");
    }).catch((err) => toast(err && err.message ? err.message : String(err), "error"));
  }

  function openPresetMenu(x, y) {
    const names = (snap && snap.presetNames) || [];
    showCtxMenu(x, y, [
      { label: "New preset…", run: newPresetDialog },
      { label: "Duplicate preset…", run: duplicatePresetDialog },
      { label: "Rename preset…", run: renamePresetDialog },
      { label: "Reset to model default", run: resetPreset },
      { label: "Export presets…", run: exportPresets },
      { label: "Discard unsaved changes", disabled: !snap.dirty, run: () => call("SelectPreset", currentPresetName()) },
      "-",
      { label: "Delete preset", danger: true, disabled: names.length <= 1, run: deletePreset },
    ]);
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

  // ---- validation: inline form findings from the snapshot ----

  function fieldNode(key) {
    return Array.from(document.querySelectorAll("[data-key]")).find((n) => n.dataset.key === key) || null;
  }

  function renderValidation() {
    const issues = (snap && snap.validation) || [];
    document.querySelectorAll(".field.invalid, .field.warn").forEach((f) => f.classList.remove("invalid", "warn"));
    el.validation.innerHTML = "";
    if (!issues.length) {
      el.validation.hidden = true;
      return;
    }
    el.validation.hidden = false;
    const rank = { warn: 1, error: 2 };
    const perField = {};
    issues.forEach((is) => {
      const item = document.createElement("div");
      item.className = "validation-item " + (is.severity || "warn");
      item.textContent = (is.severity === "error" ? "Error: " : "Warning: ") + is.message;
      item.addEventListener("click", () => {
        const node = fieldNode(is.field);
        if (node) {
          node.scrollIntoView({ block: "center", behavior: "smooth" });
          node.focus();
        }
      });
      el.validation.appendChild(item);
      if (is.field && (!perField[is.field] || rank[is.severity] > rank[perField[is.field]])) {
        perField[is.field] = is.severity;
      }
    });
    Object.entries(perField).forEach(([key, sev]) => {
      const node = fieldNode(key);
      const field = node && node.closest(".field");
      if (field) field.classList.add(sev === "error" ? "invalid" : "warn");
    });
  }

  // ---- performance charts ----

  function fmtMB(mb) {
    const n = Number(mb) || 0;
    if (n <= 0) return "—";
    return n < 1024 ? n + " MB" : (n / 1024).toFixed(1) + " GB";
  }

  function drawChart(canvas, values, color) {
    if (!canvas || canvas.clientWidth === 0) return;
    const cssW = Math.max(80, canvas.clientWidth || 160);
    const cssH = 46;
    const dpr = window.devicePixelRatio || 1;
    canvas.width = Math.round(cssW * dpr);
    canvas.height = Math.round(cssH * dpr);
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, cssW, cssH);
    const pts = values.filter((v) => Number.isFinite(v));
    if (pts.length < 2) {
      ctx.strokeStyle = "rgba(127,127,127,0.25)";
      ctx.beginPath();
      ctx.moveTo(0, cssH - 1);
      ctx.lineTo(cssW, cssH - 1);
      ctx.stroke();
      return;
    }
    let min = Math.min(...pts);
    let max = Math.max(...pts);
    if (max - min < 1e-6) max = min + 1;
    const pad = 4;
    const x = (i) => (i / (pts.length - 1)) * cssW;
    const y = (v) => cssH - pad - ((v - min) / (max - min)) * (cssH - pad * 2);
    ctx.beginPath();
    pts.forEach((v, i) => { if (i === 0) ctx.moveTo(x(i), y(v)); else ctx.lineTo(x(i), y(v)); });
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.6;
    ctx.lineJoin = "round";
    ctx.stroke();
    ctx.lineTo(x(pts.length - 1), cssH);
    ctx.lineTo(x(0), cssH);
    ctx.closePath();
    ctx.fillStyle = color.startsWith("#") ? color + "22"
      : color.replace("rgb(", "rgba(").replace(")", ", 0.12)");
    ctx.fill();
  }

  function renderCharts() {
    if (currentView !== "performance") return; // hidden canvases have no layout
    const pts = (snap && snap.metrics) || [];
    const last = pts.length ? pts[pts.length - 1] : {};
    el.chartVramVal.textContent = fmtMB(last.vramMB);
    el.chartRamVal.textContent = fmtMB(last.ramMB);
    el.chartSpeedVal.textContent = last.tokPerSec > 0 ? last.tokPerSec.toFixed(1) + " tok/s" : "—";
    const style = getComputedStyle(document.documentElement);
    const accent = style.getPropertyValue("--accent").trim() || "#ef6a2d";
    const green = style.getPropertyValue("--green").trim() || "#16a34a";
    const amber = style.getPropertyValue("--amber").trim() || "#c27803";
    drawChart(el.chartVram, pts.map((p) => p.vramMB), accent);
    drawChart(el.chartRam, pts.map((p) => p.ramMB), green);
    drawChart(el.chartSpeed, pts.map((p) => p.tokPerSec), amber);
    if (!pts.length) {
      el.metricsHint.textContent = "collecting — start the server";
      return;
    }
    const gpu = [];
    if (last.gpuUtil > 0) gpu.push("GPU " + last.gpuUtil + "%");
    if (last.gpuTemp > 0) gpu.push(last.gpuTemp + "°C");
    el.metricsHint.textContent = `${pts.length} samples · last 10 min` + (gpu.length ? " · " + gpu.join(" · ") : "");
  }

  // ---- benchmark table ----

  function fmtWhen(sec) {
    if (!sec) return "—";
    const d = new Date(sec * 1000);
    const diff = Date.now() - d.getTime();
    if (diff < 60000) return "just now";
    if (diff < 3600000) return Math.round(diff / 60000) + " min ago";
    if (diff < 86400000) return Math.round(diff / 3600000) + " h ago";
    return d.toLocaleDateString();
  }

  function renderBenchmarks() {
    const rows = ((snap && snap.benchmarks) || []).slice()
      .sort((a, b) => (b.genTokPerSec || 0) - (a.genTokPerSec || 0));
    el.benchTable.hidden = rows.length === 0;
    el.benchEmpty.hidden = rows.length > 0;
    el.benchBody.innerHTML = "";
    const curPreset = currentPresetName();
    const model = selectedModel();
    rows.forEach((b, i) => {
      const tr = document.createElement("tr");
      const samePreset = !!(model && b.modelId === model.id && b.preset === curPreset);
      const current = samePreset && b.fingerprint === snap.configFingerprint;
      const stale = samePreset && b.fingerprint !== snap.configFingerprint;
      if (current) tr.className = "current";
      if (stale) tr.className = "stale";

      const td = (text, cls) => {
        const c = document.createElement("td");
        c.textContent = text;
        if (cls) c.className = cls;
        tr.appendChild(c);
        return c;
      };
      const m = td(b.model || b.alias || "—", "bench-model");
      m.title = b.alias || "";
      const cfgCell = td(b.preset || "—");
      if (b.fingerprint) {
        const fp = document.createElement("code");
        fp.className = "bench-fp";
        fp.textContent = " #" + b.fingerprint.slice(0, 6);
        fp.title = "config fingerprint " + b.fingerprint;
        cfgCell.appendChild(fp);
      }
      td(String(b.context || "—"), "num");
      td(b.gpuLayers || "—", "num");
      td((b.genTokPerSec || 0).toFixed(1), "num" + (i === 0 ? " best" : ""));
      td((b.promptTokPerSec || 0).toFixed(1), "num");
      const pe = td(b.promptEvalMs ? (b.promptEvalMs / 1000).toFixed(2) + "s" : "—", "num");
      pe.title = b.promptEvalMs ? "prompt evaluation from /metrics (delta around the run)" : "no /metrics counters";
      const slots = td(b.slotsActive || b.queueDepth ? `${b.slotsBusy || 0}/${b.slotsActive || 0} · q${b.queueDepth || 0}` : "—", "num");
      slots.title = "busy/active slots and deferred requests";
      td(b.contextTokens ? b.contextTokens.toLocaleString() : (b.kvCacheRatio ? (b.kvCacheRatio * 100).toFixed(0) + "%" : "—"), "num");
      td(fmtMB(b.peakVramMB), "num");
      td(fmtWhen(b.at));
      const st = td("");
      if (current) {
        const chip = document.createElement("span");
        chip.className = "chip";
        chip.textContent = "current";
        st.appendChild(chip);
      } else if (stale) {
        const chip = document.createElement("span");
        chip.className = "chip";
        chip.textContent = "config edited";
        chip.title = "Stored for an older version of this config; re-run to compare.";
        st.appendChild(chip);
      } else if (i === 0) {
        const chip = document.createElement("span");
        chip.className = "chip";
        chip.textContent = "fastest";
        st.appendChild(chip);
      }
      el.benchBody.appendChild(tr);
    });
    const hint = snap.benchmarking
      ? "running — fixed prompt, 128 tokens…"
      : `${rows.length} result(s) · fastest first`;
    el.benchHint.textContent = hint;
    el.benchRun.disabled = snap.status !== "running" || snap.benchmarking;
    el.benchRun.textContent = snap.benchmarking ? "Benchmarking…" : "Benchmark";
    el.benchRun.title = snap.status !== "running"
      ? "Start a server first: the benchmark measures the loaded model + config"
      : "Measure the LOADED model and config with a fixed prompt";
  }

  function runBenchmark() {
    const a = app();
    if (!a) return;
    el.benchRun.disabled = true;
    el.benchHint.textContent = "running — fixed prompt, 128 tokens…";
    Promise.resolve(a.RunBenchmark())
      .then((b) => toast(`${b.model} [${b.preset}]: ${(b.genTokPerSec || 0).toFixed(1)} tok/s`, "ok"))
      .catch((err) => toast(err && err.message ? err.message : String(err), "error"))
      .finally(() => { if (snap) renderBenchmarks(); });
  }

  function clearBenchmarks() {
    if (!window.confirm("Delete every stored benchmark result?")) return;
    libCall("ClearBenchmarks").then(() => { renderBenchmarks(); toast("benchmarks cleared", "ok"); }).catch(() => {});
  }

  function exportBenchmarks() {
    const a = app();
    if (!a) return;
    Promise.resolve(a.BenchmarkExportText("csv")).then((text) => {
      const blob = new Blob([text], { type: "text/csv;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "llamaherd-benchmarks-" + new Date().toISOString().slice(0, 10) + ".csv";
      document.body.appendChild(link);
      link.click();
      link.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      toast("benchmarks exported", "ok");
    }).catch((err) => toast(err && err.message ? err.message : String(err), "error"));
  }

  // ---- model details ----

  let detailsModel = null;

  function detailRow(k, v) {
    if (v === "" || v === undefined || v === null) v = "—";
    const kd = document.createElement("div");
    kd.className = "k";
    kd.textContent = k;
    const vd = document.createElement("div");
    vd.className = "v";
    vd.textContent = v;
    el.detailsBody.appendChild(kd);
    el.detailsBody.appendChild(vd);
  }

  function openModelDetails(model) {
    detailsModel = model;
    el.detailsTitle.textContent = model.display || model.path || "Model details";
    el.detailsBody.innerHTML = "";
    el.detailsMsg.textContent = "";
    const cat = (model.category || []).join(" › ");
    detailRow("Alias", model.alias);
    detailRow("Quant", model.quant);
    detailRow("Architecture", model.arch);
    detailRow("Layers", model.layers ? String(model.layers) : "");
    detailRow("Full-attention layers", model.fullAttnLayers ? String(model.fullAttnLayers) : "");
    detailRow("Trained context", model.trainedContext ? model.trainedContext.toLocaleString() + " tokens" : "");
    detailRow("File size", model.fileSize ? fmtBytes(model.fileSize) : "");
    detailRow("Category", cat);
    detailRow("Vision projector", model.mmproj);
    detailRow("MTP", model.mtp ? "embedded head" : (model.mtpDraft ? "draft sidecar" : ""));
    detailRow("Source", model.source || "local file");
    const p = model.provenance;
    if (p) {
      detailRow("Repository", p.repo);
      detailRow("Licence", p.license ? p.license + (p.licenseName ? " (" + p.licenseName + ")" : "") : "");
    }
    el.detailsOpenSource.hidden = !(p && p.url);
    detailRow("Last used", model.lastUsed ? new Date(model.lastUsed * 1000).toLocaleString() : "never");
    detailRow("Path", model.path);
    el.detailsModal.hidden = false;
  }

  function closeDetails() {
    el.detailsModal.hidden = true;
    detailsModel = null;
  }

  // ---- VRAM bar: parses the "(85%)" part of the estimate text ----

  function renderVramBar(text) {
    const fallback = /\((\d+)%\)/.exec(text);
    const pct = snap && typeof snap.vramPercent === "number"
      ? snap.vramPercent
      : (fallback ? parseInt(fallback[1], 10) : 0);
    const fill = $("vramFill");
    const clamped = Math.max(0, Math.min(100, pct));
    fill.style.width = clamped + "%";
    fill.className = !(snap ? snap.vramFits : true) || clamped >= 90 ? "bad" : clamped >= 75 ? "warn" : "";
    $("vramBar").style.visibility = pct > 0 ? "visible" : "hidden";
  }

  // ---- theme: auto (system) -> light -> dark ----

  const THEMES = ["auto", "light", "dark"];

  function applyTheme(t) {
    const root = document.documentElement;
    if (t === "light" || t === "dark") root.dataset.theme = t;
    else delete root.dataset.theme;
    const b = $("themeBtn");
    b.title = "Theme: " + t + " (click to change)";
    b.textContent = "Theme: " + t;
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

  function setLogsCollapsed(collapsed) {
    el.logsGroup.classList.toggle("collapsed", collapsed);
    el.logsTitle.textContent = (collapsed ? "▸ " : "▾ ") + "Logs";
  }

  function initLogsCollapse() {
    // Always start collapsed; the user opens the accordion when needed.
    setLogsCollapsed(true);
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
    parts.push(status === "running" && srv.model && srv.model !== "—" ? `running: ${srv.model}`
      : status === "failed" ? "not responding" : status);
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
    return currentView === "llama";
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

  // ---- native pickers, shortcuts and focus trap ----

  function pickInto(method, input, ...args) {
    const a = app();
    if (!a || typeof a[method] !== "function") return;
    Promise.resolve(a[method](...args)).then((p) => {
      if (p) {
        input.value = p;
        input.dispatchEvent(new Event("input", { bubbles: true }));
      }
    }).catch((err) => toast(err && err.message ? err.message : String(err), "error"));
  }

  function detectServerBin(input, msgEl) {
    const a = app();
    if (!a) return;
    if (msgEl) msgEl.textContent = "Searching PATH and common build locations…";
    Promise.resolve(a.DetectServerBin()).then((r) => {
      input.value = r.path;
      if (input === el.pathBin) pathsDirty = true;
      if (msgEl) msgEl.textContent = "found " + r.version;
      toast("llama-server: " + r.version, "ok");
    }).catch((err) => {
      const msg = err && err.message ? err.message : String(err);
      if (msgEl) msgEl.textContent = msg;
      toast(msg, "error");
    });
  }

  function savePaths() {
    const a = app();
    if (!a) return;
    el.pathMsg.textContent = "";
    Promise.resolve(a.SetPaths(el.pathBin.value.trim(), el.pathModels.value.trim()))
      .then(() => { el.pathMsg.textContent = "saved"; pathsDirty = false; toast("paths saved", "ok"); })
      .catch((err) => {
        const msg = err && err.message ? err.message : String(err);
        el.pathMsg.textContent = "error: " + msg;
        toast(msg, "error");
      });
  }

  function visibleModal() {
    return Array.from(document.querySelectorAll(".modal")).find((m) => !m.hidden) || null;
  }

  function trapFocus(e) {
    const modal = visibleModal();
    if (!modal) return;
    const nodes = Array.from(modal.querySelectorAll(
      'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])'
    )).filter((n) => n.offsetParent !== null);
    if (!nodes.length) return;
    const first = nodes[0], last = nodes[nodes.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }

  function wireShortcuts() {
    document.addEventListener("keydown", (e) => {
      const mod = e.ctrlKey || e.metaKey;
      if (mod && e.key.toLowerCase() === "s") {
        e.preventDefault();
        call("SaveConfig");
        return;
      }
      if (mod && e.key === "Enter") {
        e.preventDefault();
        if (snap && (snap.status === "running" || snap.status === "starting")) call("RestartServer");
        else call("StartServer");
        return;
      }
      if (mod && e.key.toLowerCase() === "f") {
        e.preventDefault();
        el.modelSearch.focus();
        el.modelSearch.select();
        return;
      }
      if (e.key === "Tab") trapFocus(e);
    });
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
    wireLogs();
    wireShortcuts();
    initLogsCollapse();
    initTheme();
    initNotifications();
    initMoreMenu();
    if (el.tabLlama && el.viewLlama) {
      el.tabLlama.classList.add("active");
      el.viewLlama.appendChild(el.tabLlama);
    }
    el.viewConfigBtn.addEventListener("click", () => showView("config"));
    el.viewPerfBtn.addEventListener("click", () => showView("performance"));
    el.viewLlamaBtn.addEventListener("click", () => showView("llama"));
    showView("config");

    el.modelSearch.addEventListener("input", () => { modelQuery = el.modelSearch.value; renderModels(); });
    el.modelFilter.addEventListener("change", () => { modelFilter = el.modelFilter.value; renderModels(); });
    el.rescanBtn.addEventListener("click", () => {
      const a = app();
      if (!a) return;
      Promise.resolve(a.RescanModels()).then(() => toast("models rescanned", "ok"))
        .catch((err) => toast(err && err.message ? err.message : String(err), "error"));
    });
    $("newCatBtn").addEventListener("click", () => newFolderDialog(browsePath || []));
    el.presetSel.addEventListener("change", () => {
      if (el.presetSel.value === NEW_PRESET) {
        newPresetDialog();
        return;
      }
      call("SelectPreset", el.presetSel.value);
    });
    el.presetMenuBtn.addEventListener("click", () => {
      const r = el.presetMenuBtn.getBoundingClientRect();
      openPresetMenu(r.left, r.bottom + 2);
    });
    el.saveBtn.addEventListener("click", () => call("SaveConfig"));
    el.loadBtn.addEventListener("click", () => call("LoadConfig"));
    el.startBtn.addEventListener("click", () => call("StartServer"));
    el.stopBtn.addEventListener("click", () => call("StopServer"));
    el.restartBtn.addEventListener("click", () => call("RestartServer"));
    el.chatBtn.addEventListener("click", () => call("OpenChat"));
    el.pathBinBrowse.addEventListener("click", () => pickInto("PickServerBinary", el.pathBin));
    el.pathBinDetect.addEventListener("click", () => detectServerBin(el.pathBin, el.pathMsg));
    el.frBinDetect.addEventListener("click", () => detectServerBin(el.frBin, el.frMsg));
    el.diagCopy.addEventListener("click", () => {
      const a = app();
      if (!a) return;
      Promise.resolve(a.DiagnosticsText())
        .then((txt) => copyText(txt))
        .catch((err) => toast(err && err.message ? err.message : String(err), "error"));
    });
    el.pathModelsBrowse.addEventListener("click", () => pickInto("PickModelsDir", el.pathModels));
    el.pathBin.addEventListener("input", () => { pathsDirty = true; });
    el.pathModels.addEventListener("input", () => { pathsDirty = true; });
    el.pathSave.addEventListener("click", savePaths);
    el.notifSave.addEventListener("click", saveNotificationSettings);
    el.autoFitBtn.addEventListener("click", () => {
      const a = app();
      if (!a) return;
      Promise.resolve(a.AutoFitConfig()).then((msg) => toast(msg, "ok"))
        .catch((err) => toast(err && err.message ? err.message : String(err), "error"));
    });
    el.detailsClose.addEventListener("click", closeDetails);
    el.detailsModal.addEventListener("mousedown", (e) => { if (e.target === el.detailsModal) closeDetails(); });
    el.detailsOpenSource.addEventListener("click", () => { if (detailsModel) call("OpenModelSource", detailsModel.alias); });
    el.detailsCopy.addEventListener("click", () => {
      if (detailsModel) copyText(detailsModel.path);
    });
    el.frBinBrowse.addEventListener("click", () => pickInto("PickServerBinary", el.frBin));
    el.frModelsBrowse.addEventListener("click", () => pickInto("PickModelsDir", el.frModels));
    $("mmprojBrowse").addEventListener("click", () => pickInto("PickModelFile", $("f-mmproj"), "Choose a vision projector (.gguf)"));
    $("draftModelBrowse").addEventListener("click", () => pickInto("PickModelFile", $("f-draftModel"), "Choose a draft model (.gguf)"));
    el.frSave.addEventListener("click", () => {
      const a = app();
      if (!a) return;
      el.frMsg.textContent = "";
      Promise.resolve(a.SetPaths(el.frBin.value.trim(), el.frModels.value.trim()))
        .catch((err) => { el.frMsg.textContent = "error: " + (err && err.message ? err.message : err); });
    });

    el.tabBtnBasic.addEventListener("click", () => showTab("basic"));
    el.tabBtnAdvanced.addEventListener("click", () => showTab("advanced"));
    el.llamaRefresh.addEventListener("click", () => refreshLlama());
    el.doctorRefresh.addEventListener("click", () => call("RunDoctor"));
    el.dlOpen.addEventListener("click", openDownload);
    el.importOpen.addEventListener("click", () => openImport());
    el.importModelBrowse.addEventListener("click", pickImportFiles);
    el.importModelPath.addEventListener("change", checkImport);
    el.importMMProjBrowse.addEventListener("click", () => pickInto("PickModelFile", el.importMMProjPath, "Choose the vision projector (.gguf)"));
    el.importHasMMProj.addEventListener("change", toggleImportMMProj);
    el.importCancel.addEventListener("click", closeImport);
    el.importOk.addEventListener("click", doImport);
    el.importModal.addEventListener("mousedown", (e) => { if (e.target === el.importModal) closeImport(); });
    el.benchRun.addEventListener("click", runBenchmark);
    el.benchClear.addEventListener("click", clearBenchmarks);
    el.benchExport.addEventListener("click", exportBenchmarks);
    el.procRetry.addEventListener("click", () => call("RestartServer"));
    el.procStop.addEventListener("click", () => call("StopServer"));
    el.dlClose.addEventListener("click", () => { el.dlModal.hidden = true; });
    el.dlBrowse.addEventListener("click", openFullCatalog);
    el.catClose.addEventListener("click", () => { el.catModal.hidden = true; });
    el.catModal.addEventListener("mousedown", (e) => { if (e.target === el.catModal) el.catModal.hidden = true; });
    el.dlBadge.addEventListener("click", openDownload);
    el.dlFind.addEventListener("click", findRepoFiles);
    el.dlGatedOpen.addEventListener("click", () => { if (lastRepo) call("OpenRepoPage", lastRepo); });
    el.dlRef.addEventListener("keydown", (e) => { if (e.key === "Enter") { e.preventDefault(); findRepoFiles(); } });
    // Pasting a link looks it up right away.
    el.dlRef.addEventListener("paste", () => setTimeout(() => { if (el.dlRef.value.includes("/")) findRepoFiles(); }, 0));
    el.dlCancel.addEventListener("click", () => call("CancelDownload"));
    el.dlPause.addEventListener("click", () => call("PauseDownload"));
    el.dlResume.addEventListener("click", () => call("ResumeDownload"));
    el.dlRetry.addEventListener("click", () => call("RetryDownload"));
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
    if (rt && rt.OnFileDrop) rt.OnFileDrop(onFilesDropped, true);

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

  window.addEventListener("resize", () => {
    if (snap) renderCharts();
  });

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
