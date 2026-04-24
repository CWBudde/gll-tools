// GLL Viewer - WebAssembly Application
// Three.js core + OrbitControls for 3D previews
import * as THREE from "three";
import { OrbitControls } from "https://cdn.jsdelivr.net/npm/three@0.159.0/examples/jsm/controls/OrbitControls.js";
import {
  buildFrequencyPoints,
  buildLogFrequencyScale,
  buildZoomPluginOptions,
  getPhaseSeries,
  unwrapPhase,
} from "./modules/charting.js";
import { resetFilterChart } from "./modules/filters.js";
import { createVisualizationController } from "./modules/visualization.js";
import { createGeometryController } from "./modules/geometry.js";
import {
  downloadTextFile,
  downloadBinaryFile,
  sanitizeFilename,
} from "./utils.js";
import {
  buildXedContent,
  buildStlBinary,
  buildObjContent,
  hasGeometryData,
  resolveGeometryVertex,
  buildSequentialEdgePairs,
  triangulateFace,
  buildFrdContent,
  buildCsvFilterContent,
  buildXgfbContent,
} from "./modules/exporters.js";

// Expose Three.js for modules that rely on global access
if (window) {
  window.THREE = Object.assign({}, THREE, { OrbitControls });
}

// Runtime state for parsed data and charts
let wasmReady = false;
let currentData = null;
let currentFileBytes = null;
let chart = null;
let sourceResponseCharts = new Map();
let sourcePolarCharts = new Map();
let sourceImpedanceCharts = new Map();
let sourceBalloonStates = new Map();
let filterGroupResponseCharts = new Map();
let sourceResponseChartInitialized = new Set();
let sourcePolarChartInitialized = new Set();
let filterGroupChartInitialized = new Set();
let combinedChart = null;
let combinedChartInitialized = false;
let responseChartInitialized = false;
let combinedListenersBound = false;
let activeConfig = null; // { elements: [{ box_type_key, position: {x,y,z}, angles: {x,y,z}, gain }] }
let arrayViewer = null;
let arrayViewerNeedsUpdate = false;
let cachedArrayBalloon = null; // { frequencies, results: [{ level, phase }], grid: { merCount, parCount } }
let configDirty = false;
let arrayVisualizationState = {
  status: "empty",
  error: null,
  computedAt: null,
  sourceCount: 0,
  receiverDistance: null,
  airAttenOn: false,
};

const ARRAY_BALLOON_MER_COUNT = 72;
const ARRAY_BALLOON_PAR_COUNT = 37;

// Theme management
const THEME_KEY = "gll-viewer-theme";
const THEME_MODES = ["auto", "light", "dark"];
let currentThemeMode = "auto";

// DOM Elements (initialized in DOMContentLoaded)
let dropZone = null;
let fileInput = null;
let loading = null;
let error = null;
let errorMessage = null;
let results = null;
let fileName = null;
let clearBtn = null;

// Controllers encapsulate visualization and geometry logic
const visualization = createVisualizationController({
  getCurrentData: () => currentData,
  getCachedArrayBalloon: () => cachedArrayBalloon,
  formatFrequency,
  formatAngle,
  computePolarSlices,
  getBalloonGrid,
  getResponseWithSymmetry,
  getArrayVisualizationState,
  escapeHtml,
});
const geometry = createGeometryController({
  getCurrentData: () => currentData,
  formatNumber,
  hasGeometryData,
  resolveGeometryVertex,
  buildSequentialEdgePairs,
});

// Initialize WASM
async function initWasm() {
  const go = new Go();
  const result = await WebAssembly.instantiateStreaming(
    fetch("gll.wasm"),
    go.importObject,
  );
  go.run(result.instance);
  wasmReady = true;
}

// Initialize DOM elements
function initDOMElements() {
  dropZone = document.getElementById("drop-zone");
  fileInput = document.getElementById("file-input");
  loading = document.getElementById("loading");
  error = document.getElementById("error");
  errorMessage = document.getElementById("error-message");
  results = document.getElementById("results");
  fileName = document.getElementById("file-name");
  clearBtn = document.getElementById("clear-btn");
}

// Theme functions
function getSystemTheme() {
  // Match OS-level color scheme
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function applyTheme(mode) {
  // Resolve "auto" to actual theme and update DOM
  const theme = mode === "auto" ? getSystemTheme() : mode;
  if (theme === "dark") {
    document.documentElement.setAttribute("data-theme", "dark");
  } else {
    document.documentElement.removeAttribute("data-theme");
  }
  geometry?.updateTheme?.();
  if (arrayViewer?.grid) {
    applyGridTheme(arrayViewer.grid, readArrayThemeColors());
  }
}

function getDarkModeGridColor() {
  return document.documentElement.getAttribute("data-theme") === "dark"
    ? "rgba(148, 163, 184, 0.25)"
    : null;
}

function updateThemeToggleButton() {
  // Reflect the current theme mode in the toggle UI
  const themeToggle = document.getElementById("theme-toggle");
  const themeLabel = themeToggle?.querySelector(".theme-toggle-label");
  const themeIcon = themeToggle?.querySelector(".theme-toggle-icon");

  if (!themeLabel || !themeIcon) return;

  const labels = { auto: "Auto", light: "Light", dark: "Dark" };
  themeLabel.textContent = labels[currentThemeMode];

  // Update icon based on current mode
  if (currentThemeMode === "auto") {
    themeIcon.innerHTML = `
      <circle cx="12" cy="12" r="5"></circle>
      <line x1="12" y1="1" x2="12" y2="3"></line>
      <line x1="12" y1="21" x2="12" y2="23"></line>
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
      <line x1="1" y1="12" x2="3" y2="12"></line>
      <line x1="21" y1="12" x2="23" y2="12"></line>
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
    `;
  } else if (currentThemeMode === "light") {
    themeIcon.innerHTML = `
      <circle cx="12" cy="12" r="5"></circle>
      <line x1="12" y1="1" x2="12" y2="3"></line>
      <line x1="12" y1="21" x2="12" y2="23"></line>
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
      <line x1="1" y1="12" x2="3" y2="12"></line>
      <line x1="21" y1="12" x2="23" y2="12"></line>
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
    `;
  } else {
    themeIcon.innerHTML = `
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
    `;
  }
}

function cycleTheme() {
  // Rotate through auto → light → dark
  const currentIndex = THEME_MODES.indexOf(currentThemeMode);
  const nextIndex = (currentIndex + 1) % THEME_MODES.length;
  currentThemeMode = THEME_MODES[nextIndex];

  applyTheme(currentThemeMode);
  updateThemeToggleButton();
  localStorage.setItem(THEME_KEY, currentThemeMode);
}

function initTheme() {
  // Load stored preference and wire system theme changes
  // Load saved theme preference
  const savedTheme = localStorage.getItem(THEME_KEY);
  if (savedTheme && THEME_MODES.includes(savedTheme)) {
    currentThemeMode = savedTheme;
  }

  // Apply theme
  applyTheme(currentThemeMode);
  updateThemeToggleButton();

  // Listen for system theme changes
  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", () => {
      if (currentThemeMode === "auto") {
        applyTheme("auto");
      }
    });
}

// Initialize
document.addEventListener("DOMContentLoaded", async () => {
  // Boot sequence: theme, DOM, WASM, then UI bindings
  initTheme();
  initDOMElements();
  await initWasm();
  setupEventListeners();
  restoreCardStates();
});

// Register UI event handlers for controls and visualizations
function setupEventListeners() {
  // Theme toggle
  const themeToggle = document.getElementById("theme-toggle");
  if (themeToggle) {
    themeToggle.addEventListener("click", cycleTheme);
  }

  // Drop zone events
  dropZone.addEventListener("click", () => fileInput.click());
  dropZone.addEventListener("dragover", handleDragOver);
  dropZone.addEventListener("dragleave", handleDragLeave);
  dropZone.addEventListener("drop", handleDrop);
  fileInput.addEventListener("change", handleFileSelect);

  // Tab switching
  document.querySelectorAll(".tab").forEach((tab) => {
    tab.addEventListener("click", () => switchTab(tab.dataset.tab));
  });

  // Clear button
  clearBtn.addEventListener("click", clearResults);

  // Normalize toggles (synced)
  const balloonNormalize = document.getElementById("balloon-normalize");
  if (balloonNormalize) {
    balloonNormalize.addEventListener("change", (e) => {
      const polarNorm = document.getElementById("polar-normalize");
      if (polarNorm) polarNorm.checked = e.target.checked;
      visualization.updateBalloonVisualization();
      visualization.updatePolarChart();
    });
  }
  const polarNormalize = document.getElementById("polar-normalize");
  if (polarNormalize) {
    polarNormalize.addEventListener("change", (e) => {
      const balloonNorm = document.getElementById("balloon-normalize");
      if (balloonNorm) balloonNorm.checked = e.target.checked;
      visualization.updatePolarChart();
      visualization.updateBalloonVisualization();
    });
  }

  // Response controls
  const responseSource = document.getElementById("response-source");
  if (responseSource) {
    responseSource.addEventListener("change", updateResponseOptions);
  }
  const responseIndex = document.getElementById("response-index");
  if (responseIndex) {
    responseIndex.addEventListener("change", updateResponseChart);
  }
  const responsePhase = document.getElementById("response-phase-mode");
  if (responsePhase) {
    responsePhase.addEventListener("change", updateResponseChart);
  }
  const responseOnAxis = document.getElementById("response-use-onaxis");
  if (responseOnAxis) {
    responseOnAxis.addEventListener("change", updateResponseChart);
  }
  const responseAz = document.getElementById("response-azimuth-slider");
  if (responseAz) {
    responseAz.addEventListener("input", handleResponseAngleInput);
  }
  const responseEl = document.getElementById("response-elevation-slider");
  if (responseEl) {
    responseEl.addEventListener("input", handleResponseAngleInput);
  }
  const polarFrequency = document.getElementById("polar-frequency");
  if (polarFrequency) {
    polarFrequency.addEventListener("change", visualization.updatePolarChart);
  }
  const balloonFrequency = document.getElementById("balloon-frequency");
  if (balloonFrequency) {
    balloonFrequency.addEventListener(
      "change",
      visualization.updateBalloonVisualization,
    );
  }
  const balloonRange = document.getElementById("balloon-range");
  if (balloonRange) {
    balloonRange.addEventListener(
      "input",
      visualization.handleBalloonRangeInput,
    );
  }
  const balloonScale = document.getElementById("balloon-scale");
  if (balloonScale) {
    balloonScale.addEventListener(
      "input",
      visualization.handleBalloonScaleInput,
    );
  }
  const balloonWireframe = document.getElementById("balloon-wireframe");
  if (balloonWireframe) {
    balloonWireframe.addEventListener(
      "change",
      visualization.updateBalloonVisualization,
    );
  }
  const balloonCoverage = document.getElementById("balloon-coverage");
  if (balloonCoverage) {
    balloonCoverage.addEventListener(
      "change",
      visualization.updateBalloonVisualization,
    );
  }
  const balloonAutorotate = document.getElementById("balloon-autorotate");
  if (balloonAutorotate) {
    balloonAutorotate.addEventListener(
      "change",
      visualization.handleBalloonAutorotateToggle,
    );
  }
  window.addEventListener("resize", () => {
    geometry.handleGeometryResize();
    handleArrayViewerResize();
  });
}

// Drag/drop UI state handlers
function handleDragOver(e) {
  e.preventDefault();
  dropZone.classList.add("drag-over");
}

function handleDragLeave(e) {
  e.preventDefault();
  dropZone.classList.remove("drag-over");
}

// Accept dropped file and trigger parsing
function handleDrop(e) {
  e.preventDefault();
  dropZone.classList.remove("drag-over");
  const files = e.dataTransfer.files;
  if (files.length > 0) {
    processFile(files[0]);
  }
}

function handleFileSelect(e) {
  const files = e.target.files;
  if (files.length > 0) {
    processFile(files[0]);
  }
}

// Validate selected file and parse with WASM
async function processFile(file) {
  if (!file.name.toLowerCase().endsWith(".gll")) {
    showError("Please select a .gll file");
    return;
  }

  if (!wasmReady) {
    showError("WASM module not ready. Please wait and try again.");
    return;
  }

  showLoading();

  try {
    const arrayBuffer = await file.arrayBuffer();
    const uint8Array = new Uint8Array(arrayBuffer);
    currentFileBytes = uint8Array;

    // Call WASM parser
    const resultJson = parseGLL(uint8Array);
    const result = JSON.parse(resultJson);

    if (!result.success) {
      throw new Error(result.error);
    }

    currentData = result.data;
    fileName.textContent = file.name;
    displayResults();
  } catch (err) {
    currentFileBytes = null;
    showError("Failed to parse file: " + err.message);
  }
}

// UI helpers for loading/error states
function showLoading() {
  dropZone.classList.add("hidden");
  error.classList.add("hidden");
  results.classList.add("hidden");
  loading.classList.remove("hidden");
}

function showError(msg) {
  loading.classList.add("hidden");
  dropZone.classList.remove("hidden");
  error.classList.remove("hidden");
  errorMessage.textContent = msg;
}

// Reset state and tear down charts/visuals
function clearResults() {
  currentData = null;
  currentFileBytes = null;
  if (chart) {
    chart.destroy();
    chart = null;
  }
  if (combinedChart) {
    combinedChart.destroy();
    combinedChart = null;
  }
  sourceResponseCharts.forEach((localChart) => localChart.destroy());
  sourceResponseCharts = new Map();
  sourcePolarCharts.forEach((localChart) => localChart.destroy());
  sourcePolarCharts = new Map();
  sourceImpedanceCharts.forEach((localChart) => localChart.destroy());
  sourceImpedanceCharts = new Map();
  destroySourceBalloonStates();
  sourceBalloonStates = new Map();
  filterGroupResponseCharts.forEach((localChart) => localChart.destroy());
  filterGroupResponseCharts = new Map();
  sourceResponseChartInitialized = new Set();
  sourcePolarChartInitialized = new Set();
  filterGroupChartInitialized = new Set();
  resetFilterChart();
  visualization.resetVisualization();
  geometry.resetGeometry();
  resetArrayViewer();
  responseChartInitialized = false;
  combinedChartInitialized = false;
  activeConfig = null;
  cachedArrayBalloon = null;
  configDirty = false;
  setArrayVisualizationState({
    status: "empty",
    error: null,
    computedAt: null,
    sourceCount: 0,
    receiverDistance: null,
    airAttenOn: false,
  });
  updateConfigEditorHint("");
  results.classList.add("hidden");
  loading.classList.add("hidden");
  error.classList.add("hidden");
  dropZone.classList.remove("hidden");
  fileInput.value = "";
}

function displayResults() {
  // Show results panel and populate all tabs
  loading.classList.add("hidden");
  dropZone.classList.add("hidden");
  error.classList.add("hidden");
  results.classList.remove("hidden");

  displayOverview();
  displaySources();
  displayConfig();
  displayConfigurations();
  displayTransformers();
  displayResources();
  setupConfigEditor();
  setupSimulationParams();
  setupCombinedResponseControls();
  setupPolarControls();
  setupBalloonControls();
  setupGeometryControls();

  // Switch to overview tab
  switchTab("overview");
}

function setupSimulationParams() {
  // Wire simulation parameter controls
  const distance = document.getElementById("sim-receiver-distance");
  const temperature = document.getElementById("sim-temperature");
  const humidity = document.getElementById("sim-humidity");
  const pressure = document.getElementById("sim-pressure");
  const airAtten = document.getElementById("sim-air-atten");
  const autoRecalc = document.getElementById("sim-auto-recalc");
  const recalcBtn = document.getElementById("sim-recalculate");

  const onParamChange = () => {
    markConfigDirty();
  };

  for (const el of [distance, temperature, humidity, pressure]) {
    if (el) el.addEventListener("input", onParamChange);
  }
  if (airAtten) airAtten.addEventListener("change", onParamChange);

  if (autoRecalc) {
    autoRecalc.addEventListener("change", () => {
      const btn = document.getElementById("sim-recalculate");
      if (autoRecalc.checked) {
        if (btn) btn.classList.add("hidden");
        if (configDirty) triggerFullRecalculation();
      } else {
        updateRecalcButtonVisibility();
      }
    });
  }

  if (recalcBtn) {
    recalcBtn.addEventListener("click", () => {
      triggerFullRecalculation();
    });
  }
}

function readNumberInput(id, fallback) {
  const value = parseFloat(document.getElementById(id)?.value);
  return Number.isFinite(value) ? value : fallback;
}

function getSimulationParams() {
  return {
    receiverDistance: Math.max(0.1, readNumberInput("sim-receiver-distance", 10)),
    temperature: readNumberInput("sim-temperature", 20),
    humidity: readNumberInput("sim-humidity", 50) / 100,
    pressure: readNumberInput("sim-pressure", 101.325),
    airAttenOn: document.getElementById("sim-air-atten")?.checked || false,
  };
}

function isVisualizationTabActive() {
  return document.getElementById("tab-visualization")?.classList.contains("active");
}

function setArrayVisualizationState(next) {
  arrayVisualizationState = {
    ...arrayVisualizationState,
    ...next,
  };
  updateVisualizationStateUI();
}

function getArrayVisualizationState() {
  const hasCache = !!cachedArrayBalloon?.frequencies?.length;
  const hasActiveConfig = !!activeConfig?.elements?.length;
  const autoRecalc = document.getElementById("sim-auto-recalc")?.checked ?? true;
  const stale = hasCache && configDirty;
  const usable = hasCache && !configDirty && arrayVisualizationState.status !== "error";

  let status = arrayVisualizationState.status;
  if (recalcInProgress) {
    status = "computing";
  } else if (stale) {
    status = "stale";
  } else if (usable) {
    status = "fresh";
  } else if (!hasActiveConfig) {
    status = "empty";
  } else if (configDirty) {
    status = "pending";
  }

  return {
    ...arrayVisualizationState,
    status,
    hasActiveConfig,
    hasCache,
    stale,
    usable,
    autoRecalc,
  };
}

function buildArrayStateChips(state = getArrayVisualizationState(), includeDetails = true) {
  const chips = [];
  switch (state.status) {
    case "fresh":
      chips.push('<span class="chip chip-success">Array data fresh</span>');
      break;
    case "stale":
      chips.push('<span class="chip chip-warning">Array data stale</span>');
      chips.push('<span class="chip">Recalculate to update charts</span>');
      break;
    case "computing":
      chips.push('<span class="chip chip-info">Computing array data</span>');
      break;
    case "error":
      chips.push(
        `<span class="chip chip-error">${escapeHtml(state.error || "Array computation failed")}</span>`,
      );
      break;
    case "pending":
      chips.push('<span class="chip chip-warning">Array data not computed</span>');
      break;
  }

  if (includeDetails && (state.status === "fresh" || state.status === "stale")) {
    if (Number.isFinite(state.sourceCount)) {
      chips.push(`<span class="chip">${state.sourceCount} sources</span>`);
    }
    if (Number.isFinite(state.receiverDistance)) {
      chips.push(
        `<span class="chip">Receiver ${state.receiverDistance.toFixed(1)} m</span>`,
      );
    }
    chips.push(
      `<span class="chip">Air ${state.airAttenOn ? "on" : "off"}</span>`,
    );
  }

  return chips;
}

function setChartControlsDisabled(disabled) {
  const ids = [
    "polar-frequency",
    "polar-normalize",
    "balloon-frequency",
    "balloon-normalize",
    "balloon-range",
    "balloon-scale",
    "balloon-wireframe",
    "balloon-coverage",
    "balloon-autorotate",
    "combined-filter-group",
    "combined-filter",
    "combined-phase-mode",
    "combined-normalize",
  ];
  for (const id of ids) {
    const el = document.getElementById(id);
    if (el) el.disabled = disabled;
  }
}

function updateVisualizationStateUI() {
  if (!isVisualizationTabActive()) return;
  const state = getArrayVisualizationState();
  const disableArrayControls =
    state.status === "computing" ||
    state.status === "stale" ||
    state.status === "pending" ||
    state.status === "error";
  setChartControlsDisabled(disableArrayControls);

  if (state.status === "stale" || state.status === "pending" || state.status === "error") {
    const chips = buildArrayStateChips(state).join("");
    for (const id of ["polar-meta", "balloon-meta", "combined-response-meta"]) {
      const meta = document.getElementById(id);
      if (meta) meta.innerHTML = chips;
    }
  }
}

function markConfigDirty() {
  configDirty = true;
  updateVisualizationStateUI();
  // Only trigger computation when the visualization tab is visible.
  // Otherwise the heavy WASM call will be deferred until the tab is opened.
  if (!isVisualizationTabActive()) return;
  const autoRecalc = document.getElementById("sim-auto-recalc");
  if (autoRecalc?.checked) {
    triggerFullRecalculation();
  } else {
    updateRecalcButtonVisibility();
  }
}

function updateRecalcButtonVisibility() {
  const btn = document.getElementById("sim-recalculate");
  if (!btn) return;
  const autoRecalc = document.getElementById("sim-auto-recalc");
  if (autoRecalc?.checked || !configDirty) {
    btn.classList.add("hidden");
  } else {
    btn.classList.remove("hidden");
    btn.classList.add("sim-recalculate-dirty");
  }
}

let recalcInProgress = false;

async function triggerFullRecalculation() {
  if (recalcInProgress) return;
  recalcInProgress = true;
  configDirty = false;
  setArrayVisualizationState({
    status: "computing",
    error: null,
  });
  const btn = document.getElementById("sim-recalculate");
  if (btn) {
    btn.classList.add("hidden");
    btn.classList.remove("sim-recalculate-dirty");
  }

  setSimulationProgress(0, "Preparing array response...");

  try {
    // Yield to let the progress UI render before WASM starts.
    await nextPaint();

    const balloonComputed = await computeArrayBalloonGrid((completed, total) => {
      const pointLabel =
        total > 0
          ? ` (${completed.toLocaleString()} of ${total.toLocaleString()} points)`
          : "";
      setSimulationProgress(
        total > 0 ? (completed / total) * 100 : 0,
        `Computing array response${pointLabel}...`,
      );
    });

    setSimulationProgress(100, "Updating visualizations...");
    await nextPaint();

    updateCombinedResponseChart();
    if (balloonComputed) {
      visualization.updateBalloonOptions();
      visualization.updatePolarOptions();
      visualization.updateBalloonVisualization();
      visualization.updatePolarChart();
    } else {
      updateVisualizationStateUI();
    }
  } finally {
    hideSimulationProgress();
    recalcInProgress = false;
    updateRecalcButtonVisibility();
    updateVisualizationStateUI();
    const autoRecalc = document.getElementById("sim-auto-recalc");
    if (configDirty && autoRecalc?.checked && isVisualizationTabActive()) {
      triggerFullRecalculation();
    }
  }
}

function nextPaint() {
  return new Promise((resolve) =>
    requestAnimationFrame(() => requestAnimationFrame(resolve)),
  );
}

function setSimulationProgress(percent, messageText) {
  const overlay = document.getElementById("sim-computing-overlay");
  const message = document.getElementById("sim-computing-message");
  const percentLabel = document.getElementById("sim-computing-percent");
  const progress = document.querySelector(".sim-progress");
  const bar = document.getElementById("sim-progress-bar");
  const value = Math.max(
    0,
    Math.min(100, Number.isFinite(percent) ? percent : 0),
  );

  if (overlay) overlay.classList.remove("hidden");
  if (message) message.textContent = messageText;
  if (percentLabel) percentLabel.textContent = `${Math.round(value)}%`;
  if (progress) {
    progress.setAttribute("aria-valuenow", String(Math.round(value)));
  }
  if (bar) bar.style.width = `${value}%`;
}

function hideSimulationProgress() {
  const overlay = document.getElementById("sim-computing-overlay");
  if (overlay) overlay.classList.add("hidden");
}

function generateSphericalGrid(distance, merCount, parCount) {
  // Generate receivers on a spherical grid matching GLL balloon convention
  // meridian 0-355° (merCount steps), parallel 0-180° (parCount steps)
  const receivers = [];
  for (let m = 0; m < merCount; m++) {
    const azimuthDeg = (m * 360) / merCount;
    const azimuthRad = (azimuthDeg * Math.PI) / 180;
    for (let p = 0; p < parCount; p++) {
      const parallelDeg = (p * 180) / (parCount - 1);
      const parallelRad = (parallelDeg * Math.PI) / 180;
      // Acoustics engine convention: X = firing axis (on-axis direction).
      // Meridian rotates around X:
      //   0°=top (+Z), 90°=right (+Y), 180°=bottom (-Z), 270°=left (-Y)
      const x = distance * Math.cos(parallelRad);
      const y = distance * Math.sin(parallelRad) * Math.sin(azimuthRad);
      const z = distance * Math.sin(parallelRad) * Math.cos(azimuthRad);
      receivers.push({ x, y, z });
    }
  }
  return receivers;
}

async function computeArrayBalloonGrid(onProgress) {
  // Compute array response at a full spherical grid for balloon/polar
  if (
    !activeConfig ||
    !currentFileBytes ||
    (typeof window.computeArrayBalloonAsync !== "function" &&
      typeof window.computeArrayBalloon !== "function")
  ) {
    cachedArrayBalloon = null;
    setArrayVisualizationState({
      status: !activeConfig ? "empty" : "error",
      error: !activeConfig ? null : "Array balloon helper not available",
      computedAt: null,
      sourceCount: 0,
    });
    return false;
  }

  const elements = buildElementsFromConfig(activeConfig);
  if (!elements.length) {
    cachedArrayBalloon = null;
    setArrayVisualizationState({
      status: "pending",
      error: "No valid sources found for this configuration",
      computedAt: null,
      sourceCount: 0,
    });
    return false;
  }

  // Filter to valid sources
  const validSources = new Set(
    (currentData?.database?.source_definitions || []).map((s) => s.key),
  );
  const validElements = elements.filter((elem) =>
    validSources.has(elem.source_key),
  );
  if (!validElements.length) {
    cachedArrayBalloon = null;
    setArrayVisualizationState({
      status: "pending",
      error: "No valid sources found for this configuration",
      computedAt: null,
      sourceCount: 0,
    });
    return false;
  }

  const sim = getSimulationParams();
  const merCount = ARRAY_BALLOON_MER_COUNT; // 5° steps
  const parCount = ARRAY_BALLOON_PAR_COUNT; // 5° steps
  const receivers = generateSphericalGrid(
    sim.receiverDistance,
    merCount,
    parCount,
  );

  const payload = JSON.stringify({
    elements: validElements,
    receivers,
    air_props: {
      temperature: sim.temperature,
      humidity: sim.humidity,
      pressure: sim.pressure,
      speed: 0,
      air_atten_on: sim.airAttenOn,
    },
  });

  try {
    const result =
      typeof window.computeArrayBalloonAsync === "function"
        ? await computeArrayBalloonGridAsync(payload, onProgress)
        : JSON.parse(window.computeArrayBalloon(currentFileBytes, payload));
    if (result.success) {
      cachedArrayBalloon = {
        frequencies: result.frequencies,
        results: result.results,
        grid: { merCount, parCount },
        receiverDistance: sim.receiverDistance,
      };
      setArrayVisualizationState({
        status: "fresh",
        error: null,
        computedAt: Date.now(),
        sourceCount: validElements.length,
        receiverDistance: sim.receiverDistance,
        airAttenOn: sim.airAttenOn,
      });
      return true;
    } else {
      cachedArrayBalloon = null;
      setArrayVisualizationState({
        status: "error",
        error: result.error || "Failed to compute array balloon",
        computedAt: null,
        sourceCount: validElements.length,
        receiverDistance: sim.receiverDistance,
        airAttenOn: sim.airAttenOn,
      });
      return false;
    }
  } catch (err) {
    cachedArrayBalloon = null;
    setArrayVisualizationState({
      status: "error",
      error: err?.message || "Failed to compute array balloon",
      computedAt: null,
      sourceCount: validElements.length,
      receiverDistance: sim.receiverDistance,
      airAttenOn: sim.airAttenOn,
    });
    return false;
  }
}

function computeArrayBalloonGridAsync(payload, onProgress) {
  return new Promise((resolve, reject) => {
    let started;
    try {
      started = JSON.parse(
        window.computeArrayBalloonAsync(
          currentFileBytes,
          payload,
          (eventJSON) => {
            let event;
            try {
              event = JSON.parse(eventJSON);
            } catch (err) {
              reject(err);
              return;
            }

            if (event.type === "progress") {
              onProgress?.(event.completed || 0, event.total || 0);
              return;
            }

            if (event.type === "complete") {
              if (event.success && event.result) {
                resolve(event.result);
              } else {
                resolve({
                  success: false,
                  error:
                    event.error ||
                    event.result?.error ||
                    "Failed to compute array response",
                });
              }
            }
          },
        ),
      );
    } catch (err) {
      reject(err);
      return;
    }

    if (!started.success) {
      resolve({
        success: false,
        error: started.error || "Failed to start array response computation",
      });
    }
  });
}

function switchTab(tabName) {
  // Toggle active tab button and content panel
  // Update tab buttons
  document.querySelectorAll(".tab").forEach((tab) => {
    tab.classList.toggle("active", tab.dataset.tab === tabName);
  });

  // Update tab content
  document.querySelectorAll(".tab-content").forEach((content) => {
    content.classList.toggle("active", content.id === `tab-${tabName}`);
  });

  // Initialize chart when switching to visualization tab
  if (tabName === "visualization" && currentData) {
    const autoRecalc = document.getElementById("sim-auto-recalc");
    if (
      (configDirty || !cachedArrayBalloon) &&
      activeConfig &&
      !recalcInProgress &&
      autoRecalc?.checked
    ) {
      triggerFullRecalculation();
    } else if (!recalcInProgress) {
      updateCombinedResponseChart();
      visualization.updatePolarChart();
      visualization.updateBalloonVisualization();
      updateVisualizationStateUI();
    }
    visualization.handleBalloonResize();
    updateArrayViewer();
    handleArrayViewerResize();
  }
  if (tabName === "geometry" && currentData) {
    requestAnimationFrame(() => {
      geometry.initInlineViewers();
      geometry.handleGeometryResize();
    });
  }
}

function displayOverview() {
  // Render system metadata and header information
  const data = currentData;

  // System info
  const systemInfo = document.getElementById("system-info");
  systemInfo.innerHTML = createTableRows([
    ["Label", data.gen_system.label],
    ["Key", data.gen_system.key],
    ["Type", formatSystemType(data.gen_system.type)],
    ["Version", data.gen_system.version],
    ["Company", data.gen_system.company],
  ]);

  // Metadata
  const metadataInfo = document.getElementById("metadata-info");
  const metadataRows = [];
  if (data.metadata.description)
    metadataRows.push(["Description", data.metadata.description]);
  if (data.gen_system.website_text)
    metadataRows.push(["Website", data.gen_system.website_text]);
  if (data.gen_system.email_text)
    metadataRows.push(["Email", data.gen_system.email_text]);
  if (data.gen_system.copyright_text)
    metadataRows.push(["Copyright", data.gen_system.copyright_text]);
  if (data.gen_system.support_text)
    metadataRows.push(["Support", data.gen_system.support_text]);
  metadataInfo.innerHTML =
    metadataRows.length > 0
      ? createTableRows(metadataRows)
      : '<tr><td colspan="2" class="empty-state">No metadata available</td></tr>';

  // Header info
  const headerInfo = document.getElementById("header-info");
  headerInfo.innerHTML = createTableRows([
    ["Magic", data.header.magic],
    ["Format ID", data.header.format_id],
    ["Format Version", data.header.format_version],
    ["Sub Version", data.header.sub_version],
  ]);
}

function displaySources() {
  // Build source cards and response controls
  const sourcesList = document.getElementById("sources-list");
  const sources = currentData.database?.source_definitions || [];

  // Empty state for missing sources
  if (sources.length === 0) {
    sourcesList.innerHTML =
      '<div class="empty-state">No source definitions found</div>';
    return;
  }

  // Build source cards markup
  sourcesList.innerHTML = sources
    .map((src, sourceIndex) => {
      const def = src.definition || {};
      const balloon = def.balloon_data;
      const responseCount = src.responses?.length || 0;
      const directivityLabel = formatDirectivityType(def.directivity_type);
      const onAxisPoints = def.on_axis_spectrum?.level?.length || 0;
      const impedancePoints = def.impedance?.level?.length || 0;
      const hasResponses = responseCount > 0;
      const hasImpedance = impedancePoints > 0;
      const identityDetails = [
        renderSourceDetail("Company", def.company_label),
        renderSourceDetail("Description", def.description, { wide: true }),
      ]
        .filter(Boolean)
        .join("");

      const coverageDetails = [
        renderSourceDetail(
          "Bandwidth",
          `${formatFrequency(def.nominal_bandwidth_from)} - ${formatFrequency(def.nominal_bandwidth_to)}`,
        ),
        renderSourceDetail("Data type", formatDataType(def.data_type)),
        renderSourceDetail(
          "Responses",
          responseCount ? String(responseCount) : "-",
        ),
        balloon
          ? renderSourceDetail(
              "Resolution",
              `${balloon.angular_resolution?.meridian_step || 0}° × ${balloon.angular_resolution?.parallel_step || 0}°`,
            )
          : "",
        renderSourceDetail(
          "Rated coverage",
          formatCoverageAngles(
            def.rated_horizontal_angle,
            def.rated_vertical_angle,
          ),
        ),
        renderSourceDetail("Directivity model", directivityLabel),
      ]
        .filter(Boolean)
        .join("");

      const electricalDetails = [
        renderSourceDetail(
          "On-axis level",
          formatGainNumber(def.on_axis_level),
        ),
        renderSourceDetail(
          "On-axis response",
          onAxisPoints ? `${onAxisPoints} points` : "-",
        ),
        renderSourceDetail("Rated impedance", formatOhms(def.rated_impedance)),
        renderSourceDetail(
          "Impedance data",
          impedancePoints ? `${impedancePoints} points` : "-",
        ),
        renderSourceDetail("Max voltage", formatVoltage(def.max_voltage)),
      ]
        .filter(Boolean)
        .join("");

      const measurementDetails = [
        renderSourceDetail(
          "Measured voltage",
          formatVoltage(def.measured_voltage),
        ),
        renderSourceDetail(
          "Measured distance",
          formatDistance(def.measured_distance),
        ),
        renderSourceDetail(
          "Measured gain",
          formatGainNumber(def.measured_gain_in_db),
        ),
        renderSourceDetail("Temperature", formatTemperature(def.temperature)),
        renderSourceDetail("Humidity", formatPercent(def.humidity)),
        renderSourceDetail(
          "Atmospheric pressure",
          formatPressure(def.atmospheric_pressure),
        ),
      ]
        .filter(Boolean)
        .join("");

      const detailsSections = [
        buildSourceSectionRow([
          buildSourceSection("Identity", identityDetails, { column: "left" }),
          buildSourceSection("Electrical", electricalDetails, {
            column: "right",
          }),
        ]),
        buildSourceSection("Coverage", coverageDetails),
        buildSourceSection("Measurement", measurementDetails),
      ]
        .filter(Boolean)
        .join("");

      // Response select options with angle labels
      const responseOptions = responseCount
        ? Array.from({ length: responseCount }, (_, i) => {
            const angle = computeResponseAngles(src, i);
            const angleLabel = angle
              ? ` • Az ${formatAngle(angle.meridianDeg)}° / Off ${formatAngle(angle.parallelDeg)}°`
              : "";
            return `<option value="${i}">Response ${i + 1}${angleLabel}</option>`;
          }).join("")
        : "";

      const responsePanel = hasResponses
        ? `
              <div class="source-response">
                <div class="response-controls source-response-controls">
                  <div class="response-controls-row">
                    <label>
                      Response:
                      <select id="source-response-index-${sourceIndex}">
                        ${responseOptions}
                      </select>
                    </label>
                    <label>
                      Phase:
                      <select id="source-response-phase-${sourceIndex}">
                        <option value="unwrapped" selected>Unwrapped</option>
                        <option value="wrapped">Wrapped</option>
                        <option value="group-delay">Group delay</option>
                      </select>
                    </label>
                    <label class="response-toggle">
                      <input
                        id="source-response-normalized-${sourceIndex}"
                        type="checkbox"
                      />
                      Normalized
                    </label>
                    <div class="dropdown-container source-export-dropdown">
                      <button class="btn-download btn-source-export" data-source-idx="${sourceIndex}">
                        Export <span class="dropdown-icon">▼</span>
                      </button>
                      <div class="dropdown-menu">
                        <button class="dropdown-item" data-format="frd">Response .frd</button>
                        <button class="dropdown-item" data-format="csv">Response .csv</button>
                      </div>
                    </div>
                  </div>
                  <div class="response-controls-row">
                    <label class="response-slider">
                      Azimuth:
                      <input
                        id="source-response-azimuth-${sourceIndex}"
                        type="range"
                        min="0"
                        max="360"
                        step="1"
                        value="0"
                      />
                      <span
                        id="source-response-azimuth-value-${sourceIndex}"
                        class="angle-value"
                        >-</span
                      >
                    </label>
                    <label class="response-slider">
                      Off-axis:
                      <input
                        id="source-response-elevation-${sourceIndex}"
                        type="range"
                        min="0"
                        max="180"
                        step="1"
                        value="0"
                      />
                      <span
                        id="source-response-elevation-value-${sourceIndex}"
                        class="angle-value"
                        >-</span
                      >
                    </label>
                  </div>
                </div>
                <div class="source-response-chart">
                  <canvas id="source-response-chart-${sourceIndex}"></canvas>
                </div>
                <div
                  id="source-response-meta-${sourceIndex}"
                  class="response-meta"
                ></div>
              </div>
            `
        : '<div class="empty-state">No frequency response data available</div>';

      const polarPanel = hasResponses
        ? `
            <div class="source-polar-controls">
              <label>
                Frequency:
                <select id="source-polar-frequency-${sourceIndex}"></select>
              </label>
              <label class="response-toggle">
                <input id="source-polar-normalize-${sourceIndex}" type="checkbox" />
                Normalize
              </label>
            </div>
            <div class="source-polar-chart">
              <canvas id="source-polar-chart-${sourceIndex}"></canvas>
            </div>
            <div id="source-polar-meta-${sourceIndex}" class="response-meta"></div>
          `
        : '<div class="empty-state">No polar data available</div>';

      const balloonPanel = hasResponses
        ? `
            <div class="balloon-controls source-balloon-controls">
              <div class="source-balloon-row">
                <label>
                  Frequency:
                  <select id="source-balloon-frequency-${sourceIndex}"></select>
                </label>
                <label class="balloon-slider">
                  <input
                    id="source-balloon-frequency-slider-${sourceIndex}"
                    type="range"
                    min="0"
                    max="0"
                    step="1"
                    value="0"
                  />
                  <span id="source-balloon-frequency-value-${sourceIndex}" class="frequency-value">-</span>
                </label>
              </div>
              <div class="source-balloon-row">
                <label class="balloon-slider">
                  Range (dB):
                  <input
                    id="source-balloon-range-${sourceIndex}"
                    type="range"
                    min="20"
                    max="80"
                    step="5"
                    value="40"
                  />
                  <span id="source-balloon-range-value-${sourceIndex}" class="frequency-value">40</span>
                </label>
                <label class="balloon-slider">
                  Scale:
                  <input
                    id="source-balloon-scale-${sourceIndex}"
                    type="range"
                    min="0.6"
                    max="1.6"
                    step="0.1"
                    value="1"
                  />
                  <span id="source-balloon-scale-value-${sourceIndex}" class="frequency-value">1.0×</span>
                </label>
                <label class="balloon-toggle">
                  <input id="source-balloon-wireframe-${sourceIndex}" type="checkbox" />
                  Wireframe
                </label>
                <label class="balloon-toggle">
                  <input id="source-balloon-coverage-${sourceIndex}" type="checkbox" checked />
                  Coverage
                </label>
                <label class="balloon-toggle">
                  <input id="source-balloon-autorotate-${sourceIndex}" type="checkbox" checked />
                  Auto-rotate
                </label>
                <label class="balloon-toggle">
                  <input id="source-balloon-normalize-${sourceIndex}" type="checkbox" />
                  Normalize
                </label>
              </div>
            </div>
            <div id="source-balloon-viewer-${sourceIndex}" class="balloon-viewer">
              <div id="source-balloon-placeholder-${sourceIndex}" class="empty-state">
                No 3D balloon data available
              </div>
              <div class="balloon-legend">
                <div class="balloon-legend-title">SPL (dB)</div>
                <div class="balloon-legend-bar"></div>
                <div class="balloon-legend-labels">
                  <span>Low</span>
                  <span>High</span>
                </div>
              </div>
            </div>
            <div id="source-balloon-meta-${sourceIndex}" class="response-meta"></div>
          `
        : '<div class="empty-state">No balloon data available</div>';

      const impedancePanel = hasImpedance
        ? `
            <div class="source-impedance-controls">
              <label>
                Phase:
                <select id="source-impedance-phase-${sourceIndex}">
                  <option value="unwrapped" selected>Unwrapped</option>
                  <option value="wrapped">Wrapped</option>
                  <option value="group-delay">Group delay</option>
                </select>
              </label>
            </div>
            <div class="source-impedance-chart">
              <canvas id="source-impedance-chart-${sourceIndex}"></canvas>
            </div>
            <div id="source-impedance-meta-${sourceIndex}" class="response-meta"></div>
          `
        : '<div class="empty-state">No impedance data available</div>';

      const plotTabs = `
          <div class="source-plot-tabs" role="tablist">
            <button class="source-plot-tab active" data-plot="response" type="button">Response</button>
            <button class="source-plot-tab" data-plot="polar" type="button" ${hasResponses ? "" : "disabled"}>Polar</button>
            <button class="source-plot-tab" data-plot="balloon" type="button" ${hasResponses ? "" : "disabled"}>Balloon</button>
            <button class="source-plot-tab" data-plot="impedance" type="button" ${hasImpedance ? "" : "disabled"}>Impedance</button>
          </div>
        `;

      const plotPanels = `
          ${plotTabs}
          <div class="source-plot-panel" data-plot="response">
            ${responsePanel}
          </div>
          <div class="source-plot-panel" data-plot="polar">
            ${polarPanel}
          </div>
          <div class="source-plot-panel" data-plot="balloon">
            ${balloonPanel}
          </div>
          <div class="source-plot-panel" data-plot="impedance">
            ${impedancePanel}
          </div>
        `;
      return `
            <div class="source-card source-collapsible" data-source-idx="${sourceIndex}">
                <div class="source-header source-header-toggle" onclick="toggleSource(${sourceIndex})">
                    <div class="source-title">
                      <span class="source-toggle">▶</span>
                      <span class="source-label">${escapeHtml(def.label || "Unknown")}</span>
                    </div>
                    <span class="source-key">${escapeHtml(src.key)}</span>
                </div>
                <div class="source-content" style="display: none;">
                  ${detailsSections}
                  ${plotPanels}
                </div>
            </div>
        `;
    })
    .join("");

  wireSourceResponseControls();
  wireSourcePlotControls();
  wireSourceExportDropdowns();
}

function renderSourceDetail(label, value, options = {}) {
  if (value === null || value === undefined || value === "" || value === "-") {
    return "";
  }
  const classes = ["source-detail"];
  if (options.wide) classes.push("source-detail--wide");
  return `<div class="${classes.join(" ")}"><strong>${escapeHtml(label)}:</strong><span>${escapeHtml(String(value))}</span></div>`;
}

function buildSourceSection(title, content, options = {}) {
  if (!content) return "";
  const columnClass = options.column
    ? ` source-section--${options.column}`
    : "";
  return `
    <div class="source-section${columnClass}">
      <h4 class="source-section-title">${escapeHtml(title)}</h4>
      <div class="source-details">
        ${content}
      </div>
    </div>
  `;
}

function buildSourceSectionRow(sections) {
  const rowSections = sections.filter(Boolean);
  if (!rowSections.length) return "";
  return `
    <div class="source-section-row">
      ${rowSections.join("")}
    </div>
  `;
}

function wireSourceResponseControls() {
  // Attach listeners for per-source response widgets
  const cards = document.querySelectorAll(".source-card[data-source-idx]");
  cards.forEach((card) => {
    const sourceIndex = Number(card.dataset.sourceIdx);
    const elements = getSourceResponseElements(sourceIndex);
    if (!elements.indexSelect) {
      return;
    }

    elements.indexSelect.addEventListener("change", () =>
      renderSourceResponseChart(sourceIndex),
    );
    elements.phaseSelect?.addEventListener("change", () =>
      renderSourceResponseChart(sourceIndex),
    );
    elements.normalizedToggle?.addEventListener("change", () =>
      renderSourceResponseChart(sourceIndex),
    );
    elements.azSlider?.addEventListener("input", () =>
      handleSourceResponseAngleInput(sourceIndex),
    );
    elements.elSlider?.addEventListener("input", () =>
      handleSourceResponseAngleInput(sourceIndex),
    );
  });
}

function getSourceResponseElements(sourceIndex) {
  // Resolve DOM nodes for a source card
  return {
    indexSelect: document.getElementById(
      `source-response-index-${sourceIndex}`,
    ),
    phaseSelect: document.getElementById(
      `source-response-phase-${sourceIndex}`,
    ),
    normalizedToggle: document.getElementById(
      `source-response-normalized-${sourceIndex}`,
    ),
    azSlider: document.getElementById(`source-response-azimuth-${sourceIndex}`),
    elSlider: document.getElementById(
      `source-response-elevation-${sourceIndex}`,
    ),
    azValue: document.getElementById(
      `source-response-azimuth-value-${sourceIndex}`,
    ),
    elValue: document.getElementById(
      `source-response-elevation-value-${sourceIndex}`,
    ),
    chartCanvas: document.getElementById(
      `source-response-chart-${sourceIndex}`,
    ),
    meta: document.getElementById(`source-response-meta-${sourceIndex}`),
  };
}

function getSourcePlotElements(sourceIndex) {
  return {
    plotTabs: document.querySelectorAll(
      `.source-card[data-source-idx="${sourceIndex}"] .source-plot-tab`,
    ),
    panels: document.querySelectorAll(
      `.source-card[data-source-idx="${sourceIndex}"] .source-plot-panel`,
    ),
    polarSelect: document.getElementById(
      `source-polar-frequency-${sourceIndex}`,
    ),
    polarNormalize: document.getElementById(
      `source-polar-normalize-${sourceIndex}`,
    ),
    polarCanvas: document.getElementById(`source-polar-chart-${sourceIndex}`),
    polarMeta: document.getElementById(`source-polar-meta-${sourceIndex}`),
    balloonSelect: document.getElementById(
      `source-balloon-frequency-${sourceIndex}`,
    ),
    balloonSlider: document.getElementById(
      `source-balloon-frequency-slider-${sourceIndex}`,
    ),
    balloonSliderValue: document.getElementById(
      `source-balloon-frequency-value-${sourceIndex}`,
    ),
    balloonRange: document.getElementById(
      `source-balloon-range-${sourceIndex}`,
    ),
    balloonRangeValue: document.getElementById(
      `source-balloon-range-value-${sourceIndex}`,
    ),
    balloonScale: document.getElementById(
      `source-balloon-scale-${sourceIndex}`,
    ),
    balloonScaleValue: document.getElementById(
      `source-balloon-scale-value-${sourceIndex}`,
    ),
    balloonWireframe: document.getElementById(
      `source-balloon-wireframe-${sourceIndex}`,
    ),
    balloonCoverage: document.getElementById(
      `source-balloon-coverage-${sourceIndex}`,
    ),
    balloonAutorotate: document.getElementById(
      `source-balloon-autorotate-${sourceIndex}`,
    ),
    balloonNormalize: document.getElementById(
      `source-balloon-normalize-${sourceIndex}`,
    ),
    balloonViewer: document.getElementById(
      `source-balloon-viewer-${sourceIndex}`,
    ),
    balloonPlaceholder: document.getElementById(
      `source-balloon-placeholder-${sourceIndex}`,
    ),
    balloonMeta: document.getElementById(`source-balloon-meta-${sourceIndex}`),
    impedancePhase: document.getElementById(
      `source-impedance-phase-${sourceIndex}`,
    ),
    impedanceCanvas: document.getElementById(
      `source-impedance-chart-${sourceIndex}`,
    ),
    impedanceMeta: document.getElementById(
      `source-impedance-meta-${sourceIndex}`,
    ),
  };
}

function setSourcePlotPanel(sourceIndex, plot) {
  const elements = getSourcePlotElements(sourceIndex);
  elements.panels.forEach((panel) => {
    const isActive = panel.dataset.plot === plot;
    panel.classList.toggle("active", isActive);
  });
  elements.plotTabs.forEach((tab) => {
    tab.classList.toggle("active", tab.dataset.plot === plot);
  });

  if (plot !== "balloon") {
    stopSourceBalloonLoop(sourceIndex);
  }
}

function renderSourcePlot(sourceIndex) {
  const elements = getSourcePlotElements(sourceIndex);
  const activeTab =
    Array.from(elements.plotTabs || []).find((tab) =>
      tab.classList.contains("active"),
    ) || elements.plotTabs?.[0];
  const plot = activeTab?.dataset.plot || "response";
  setSourcePlotPanel(sourceIndex, plot);

  if (plot === "polar") {
    renderSourcePolarChart(sourceIndex);
  } else if (plot === "balloon") {
    renderSourceBalloon(sourceIndex);
  } else if (plot === "impedance") {
    renderSourceImpedanceChart(sourceIndex);
  } else {
    renderSourceResponseChart(sourceIndex);
  }
}

function wireSourcePlotControls() {
  const cards = document.querySelectorAll(".source-card[data-source-idx]");
  cards.forEach((card) => {
    const sourceIndex = Number(card.dataset.sourceIdx);
    const elements = getSourcePlotElements(sourceIndex);

    elements.plotTabs.forEach((tab) => {
      tab.addEventListener("click", () => {
        if (tab.disabled) return;
        elements.plotTabs.forEach((item) =>
          item.classList.toggle("active", item === tab),
        );
        renderSourcePlot(sourceIndex);
      });
    });

    if (elements.polarSelect) {
      elements.polarSelect.addEventListener("change", () =>
        renderSourcePolarChart(sourceIndex),
      );
    }
    elements.polarNormalize?.addEventListener("change", () =>
      renderSourcePolarChart(sourceIndex),
    );

    if (elements.balloonSelect) {
      elements.balloonSelect.addEventListener("change", () =>
        renderSourceBalloon(sourceIndex),
      );
    }
    if (elements.balloonSelect && elements.balloonSlider) {
      elements.balloonSelect.addEventListener("change", () => {
        elements.balloonSlider.value = elements.balloonSelect.value;
      });
    }
    elements.balloonSlider?.addEventListener("input", () =>
      handleSourceBalloonFrequencySlider(sourceIndex),
    );
    elements.balloonRange?.addEventListener("input", () =>
      handleSourceBalloonRangeInput(sourceIndex),
    );
    elements.balloonScale?.addEventListener("input", () =>
      handleSourceBalloonScaleInput(sourceIndex),
    );
    elements.balloonWireframe?.addEventListener("change", () =>
      renderSourceBalloon(sourceIndex),
    );
    elements.balloonCoverage?.addEventListener("change", () =>
      renderSourceBalloon(sourceIndex),
    );
    elements.balloonAutorotate?.addEventListener("change", () =>
      renderSourceBalloon(sourceIndex),
    );
    elements.balloonNormalize?.addEventListener("change", () =>
      renderSourceBalloon(sourceIndex),
    );

    elements.impedancePhase?.addEventListener("change", () =>
      renderSourceImpedanceChart(sourceIndex),
    );

    const defaultTab = elements.plotTabs?.[0];
    if (defaultTab) {
      elements.plotTabs.forEach((item) =>
        item.classList.toggle("active", item === defaultTab),
      );
      setSourcePlotPanel(sourceIndex, defaultTab.dataset.plot || "response");
    }
  });
}

function wireSourceExportDropdowns() {
  // Attach export dropdown behavior for source responses
  document.querySelectorAll(".btn-source-export").forEach((button) => {
    if (button.dataset.bound === "true") return;
    button.dataset.bound = "true";
    button.addEventListener("click", (e) => {
      e.stopPropagation();
      const container = button.closest(".dropdown-container");
      const wasOpen = container.classList.contains("show");
      document.querySelectorAll(".dropdown-container.show").forEach((other) => {
        other.classList.remove("show");
      });
      if (!wasOpen) container.classList.add("show");
    });
  });

  document
    .querySelectorAll(".source-export-dropdown .dropdown-item")
    .forEach((item) => {
      if (item.dataset.bound === "true") return;
      item.dataset.bound = "true";
      item.addEventListener("click", (e) => {
        e.stopPropagation();
        const format = item.dataset.format;
        const button = item
          .closest(".dropdown-container")
          .querySelector(".btn-source-export");
        const sourceIdx = Number(button.dataset.sourceIdx);
        const exportData = buildSourceResponseExportData(sourceIdx);
        if (!exportData) return;

        const { frequencies, level, phase, responseIndex } = exportData;
        if (!frequencies?.length) return;

        const basename = buildSourceExportBasename(sourceIdx, responseIndex);
        if (format === "frd") {
          const content = buildFrdContent(frequencies, level, phase);
          downloadTextFile(`${basename}.frd`, content);
        } else if (format === "csv") {
          const content = buildCsvFilterContent(frequencies, level, phase);
          downloadTextFile(`${basename}.csv`, content);
        }

        item.closest(".dropdown-container").classList.remove("show");
      });
    });
}

function buildSourceExportBasename(sourceIndex, responseIndex) {
  const source = currentData?.database?.source_definitions?.[sourceIndex];
  const gllName = sanitizeFilename(
    currentData?.gen_system?.model ||
      currentData?.gen_system?.manufacturer ||
      "gll",
  );
  const sourceLabel = sanitizeFilename(
    source?.definition?.label || source?.key || `source-${sourceIndex + 1}`,
  );
  const respLabel =
    responseIndex !== null && responseIndex !== undefined
      ? `resp-${responseIndex + 1}`
      : "response";
  return `${gllName}_${sourceLabel}_${respLabel}`;
}

function buildSourceResponseExportData(sourceIndex) {
  const source = currentData?.database?.source_definitions?.[sourceIndex];
  if (!source) return null;
  const elements = getSourceResponseElements(sourceIndex);
  const responseIndex = parseInt(elements.indexSelect?.value);
  if (Number.isNaN(responseIndex)) return null;
  const response = source?.responses?.[responseIndex];
  if (!response) return null;

  const onAxis = source?.definition?.on_axis_spectrum;
  const onAxisFreqs = buildLogFrequencies(
    onAxis?.definition,
    onAxis?.level?.length,
  );
  const useNormalized = !!elements.normalizedToggle?.checked;
  const canCombineOnAxis =
    !useNormalized &&
    onAxis &&
    Array.isArray(onAxis.level) &&
    Array.isArray(onAxisFreqs) &&
    response.frequencies.length === onAxisFreqs.length &&
    response.level.length === onAxis.level.length &&
    frequenciesMatch(response.frequencies, onAxisFreqs);
  const levelSeries = canCombineOnAxis
    ? response.level.map((value, i) => value + onAxis.level[i])
    : response.level;

  const phaseMode = elements.phaseSelect?.value || "unwrapped";
  const rawPhase = response.phase || [];
  const onAxisPhase = onAxis?.phase || [];
  const canCombinePhase =
    canCombineOnAxis &&
    Array.isArray(onAxisPhase) &&
    rawPhase.length > 0 &&
    rawPhase.length === onAxisPhase.length;
  const combinedPhase = canCombinePhase
    ? rawPhase.map((value, i) => value + onAxisPhase[i])
    : rawPhase;
  const responseDelay = Number.isFinite(response.delay) ? response.delay : 0;
  const onAxisDelay =
    canCombinePhase && Number.isFinite(onAxis?.delay) ? onAxis.delay : 0;
  const combinedDelay = responseDelay + onAxisDelay;
  const delayAdjustedPhase = applyDelayToPhase(
    combinedPhase,
    response.frequencies,
    combinedDelay,
  );
  const unwrappedPhase = unwrapPhase(delayAdjustedPhase);
  const phaseSeries = getPhaseSeries(
    phaseMode,
    response.frequencies,
    delayAdjustedPhase,
    unwrappedPhase,
  );

  return {
    frequencies: response.frequencies,
    level: levelSeries,
    phase: phaseSeries.values,
    responseIndex,
  };
}

function renderSourceResponseChart(sourceIndex) {
  // Build chart for the selected source response
  const source = currentData.database?.source_definitions?.[sourceIndex];
  if (!source) {
    return;
  }

  const elements = getSourceResponseElements(sourceIndex);
  const responseIndex = parseInt(elements.indexSelect?.value);
  if (Number.isNaN(responseIndex)) {
    return;
  }

  const response = source?.responses?.[responseIndex];
  if (!response || !elements.chartCanvas) {
    return;
  }

  updateSourceResponseAngleControls(source, responseIndex, elements);
  updateSourceResponseMeta(elements.meta, source, responseIndex);

  // Optional on-axis normalization / combination
  const onAxis = source?.definition?.on_axis_spectrum;
  if (elements.normalizedToggle) {
    const hasOnAxis =
      !!onAxis && Array.isArray(onAxis.level) && onAxis.level.length > 0;
    elements.normalizedToggle.disabled = !hasOnAxis;
    if (!hasOnAxis) {
      elements.normalizedToggle.checked = false;
    }
  }
  const onAxisFreqs = buildLogFrequencies(
    onAxis?.definition,
    onAxis?.level?.length,
  );
  const useNormalized = !!elements.normalizedToggle?.checked;
  const canCombineOnAxis =
    !useNormalized &&
    onAxis &&
    Array.isArray(onAxis.level) &&
    Array.isArray(onAxisFreqs) &&
    response.frequencies.length === onAxisFreqs.length &&
    response.level.length === onAxis.level.length &&
    frequenciesMatch(response.frequencies, onAxisFreqs);
  const levelSeries = canCombineOnAxis
    ? response.level.map((value, i) => value + onAxis.level[i])
    : response.level;

  // Phase handling with optional on-axis merge and delay
  const phaseMode = elements.phaseSelect?.value || "unwrapped";
  const rawPhase = response.phase || [];
  const onAxisPhase = onAxis?.phase || [];
  const canCombinePhase =
    canCombineOnAxis &&
    Array.isArray(onAxisPhase) &&
    rawPhase.length > 0 &&
    rawPhase.length === onAxisPhase.length;
  const combinedPhase = canCombinePhase
    ? rawPhase.map((value, i) => value + onAxisPhase[i])
    : rawPhase;
  const responseDelay = Number.isFinite(response.delay) ? response.delay : 0;
  const onAxisDelay =
    canCombinePhase && Number.isFinite(onAxis?.delay) ? onAxis.delay : 0;
  const combinedDelay = responseDelay + onAxisDelay;
  const delayAdjustedPhase = applyDelayToPhase(
    combinedPhase,
    response.frequencies,
    combinedDelay,
  );
  const unwrappedPhase = unwrapPhase(delayAdjustedPhase);
  const phaseSeries = getPhaseSeries(
    phaseMode,
    response.frequencies,
    delayAdjustedPhase,
    unwrappedPhase,
  );
  const phaseLabel = canCombinePhase
    ? `${phaseSeries.label} (on-axis + directivity)`
    : phaseSeries.label;

  // Build chart datasets for level/phase
  const frequencyData = buildFrequencyPoints(response.frequencies, levelSeries);
  const phaseData = buildFrequencyPoints(
    response.frequencies,
    phaseSeries.values,
  );
  if (!frequencyData) {
    return;
  }

  const ctx = elements.chartCanvas.getContext("2d");
  const existingChart = sourceResponseCharts.get(sourceIndex);
  if (existingChart) {
    existingChart.destroy();
  }

  // Animate only on first render for this source
  const shouldAnimate = !sourceResponseChartInitialized.has(sourceIndex);

  const gridColor = getDarkModeGridColor();
  const xScale = buildLogFrequencyScale(
    frequencyData.minFrequency,
    frequencyData.maxFrequency,
    "Frequency",
  );
  if (gridColor) {
    xScale.grid = { ...(xScale.grid || {}), color: gridColor };
  }

  const localChart = new Chart(ctx, {
    type: "line",
    data: {
      datasets: [
        {
          label: canCombineOnAxis
            ? "Level (dB, on-axis + directivity)"
            : "Level (dB)",
          data: frequencyData.points,
          borderColor: "#2563eb",
          backgroundColor: "rgba(37, 99, 235, 0.1)",
          fill: true,
          tension: 0.3,
          pointRadius: 0,
          yAxisID: "y",
        },
        {
          label: phaseLabel,
          data: phaseData ? phaseData.points : [],
          borderColor: "#dc2626",
          backgroundColor: "transparent",
          tension: 0.3,
          pointRadius: 0,
          yAxisID: "y1",
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: shouldAnimate ? { duration: 700 } : false,
      interaction: {
        mode: "index",
        intersect: false,
      },
      scales: {
        x: xScale,
        y: {
          type: "linear",
          display: true,
          position: "left",
          title: {
            display: true,
            text: "Level (dB)",
          },
          grid: gridColor ? { color: gridColor } : undefined,
        },
        y1: {
          type: "linear",
          display: true,
          position: "right",
          title: {
            display: true,
            text: phaseSeries.axisTitle,
          },
          grid: {
            drawOnChartArea: false,
          },
        },
      },
      plugins: {
        legend: {
          position: "top",
        },
        tooltip: {
          callbacks: {
            title: (items) => {
              const value = items?.[0]?.parsed?.x;
              return value ? formatFrequency(value) : "";
            },
          },
        },
        zoom: buildZoomPluginOptions(
          frequencyData.minFrequency,
          frequencyData.maxFrequency,
        ),
      },
    },
  });

  sourceResponseCharts.set(sourceIndex, localChart);
  sourceResponseChartInitialized.add(sourceIndex);
}

function updateSourceResponseAngleControls(source, responseIndex, elements) {
  // Configure sliders based on available balloon grid
  const azSlider = elements.azSlider;
  const elSlider = elements.elSlider;
  const azValue = elements.azValue;
  const elValue = elements.elValue;

  if (!azSlider || !elSlider || !azValue || !elValue) {
    return;
  }

  const ang = source?.definition?.balloon_data?.angular_resolution;
  const grid = source ? getBalloonGrid(source) : null;

  if (!ang || !grid || !grid.meridianCount || !grid.parallelCount) {
    azSlider.disabled = true;
    elSlider.disabled = true;
    azSlider.value = "0";
    elSlider.value = "0";
    azValue.textContent = "-";
    elValue.textContent = "-";
    return;
  }

  const azMax = (grid.meridianCount - 1) * ang.meridian_step;
  const elMax = (grid.parallelCount - 1) * ang.parallel_step;

  azSlider.disabled = false;
  elSlider.disabled = false;
  azSlider.min = "0";
  azSlider.max = String(azMax);
  azSlider.step = String(ang.meridian_step);
  elSlider.min = "0";
  elSlider.max = String(elMax);
  elSlider.step = String(ang.parallel_step);

  if (responseIndex === null || responseIndex === undefined) {
    return;
  }

  const angle = computeResponseAngles(source, responseIndex);
  if (!angle) {
    azValue.textContent = "-";
    elValue.textContent = "-";
    return;
  }

  azSlider.value = String(angle.meridianDeg);
  elSlider.value = String(angle.parallelDeg);
  azValue.textContent = `${formatAngle(angle.meridianDeg)}°`;
  elValue.textContent = `${formatAngle(angle.parallelDeg)}°`;
}

function updateSourceResponseMeta(meta, source, responseIndex) {
  // Update per-response metadata chips
  if (!meta) return;
  meta.innerHTML = buildResponseMetaHtml(source, responseIndex);
}

function handleSourceResponseAngleInput(sourceIndex) {
  // Map slider angles back to response index
  const source = currentData.database?.source_definitions?.[sourceIndex];
  if (!source) {
    return;
  }
  const ang = source?.definition?.balloon_data?.angular_resolution;
  const grid = getBalloonGrid(source);
  if (!ang || !grid) {
    return;
  }

  const elements = getSourceResponseElements(sourceIndex);
  const azSlider = elements.azSlider;
  const elSlider = elements.elSlider;
  if (!azSlider || !elSlider) {
    return;
  }

  const azimuthDeg = normalizeAzimuthForGrid(Number(azSlider.value), ang);
  const elevationDeg = Number(elSlider.value);
  const meridianIdx = Math.round(azimuthDeg / ang.meridian_step);
  const parallelIdx = Math.round(elevationDeg / ang.parallel_step);
  const responseIndex = getResponseIndex(grid, meridianIdx, null, ang, {
    parallelIdx,
  });
  if (responseIndex === null) {
    return;
  }

  const indexSelect = elements.indexSelect;
  if (!indexSelect) {
    return;
  }

  setResponseSelectValue(indexSelect, source, responseIndex);
  renderSourceResponseChart(sourceIndex);
}

const sourcePolarCompassPlugin = {
  id: "sourcePolarCompass",
  afterDraw(chart) {
    const scale = chart.scales?.r;
    if (!scale) return;
    const { xCenter, yCenter, drawingArea } = scale;
    const ctx = chart.ctx;
    const sideOffset = 36;
    const vertOffset = 24;

    ctx.save();
    ctx.font = "bold 11px sans-serif";

    ctx.fillStyle = "#334155";
    ctx.textBaseline = "middle";
    ctx.textAlign = "left";
    ctx.fillText("Front", xCenter + drawingArea + sideOffset, yCenter);
    ctx.textAlign = "right";
    ctx.fillText("Back", xCenter - drawingArea - sideOffset, yCenter);

    ctx.textAlign = "center";
    ctx.textBaseline = "bottom";
    ctx.fillStyle = "#2563eb";
    ctx.fillText("Right", xCenter - 16, yCenter - drawingArea - vertOffset);
    ctx.fillStyle = "#dc2626";
    ctx.fillText("Top", xCenter + 16, yCenter - drawingArea - vertOffset);

    ctx.textBaseline = "top";
    ctx.fillStyle = "#2563eb";
    ctx.fillText("Left", xCenter - 18, yCenter + drawingArea + vertOffset);
    ctx.fillStyle = "#dc2626";
    ctx.fillText("Bottom", xCenter + 18, yCenter + drawingArea + vertOffset);

    ctx.restore();
  },
};

function findNearestFrequencyIndex(frequencies, targetFrequency) {
  let closestIndex = 0;
  let closestDistance = Infinity;
  frequencies.forEach((freq, index) => {
    const distance = Math.abs(freq - targetFrequency);
    if (distance < closestDistance) {
      closestDistance = distance;
      closestIndex = index;
    }
  });
  return closestIndex;
}

function computeLevelRange(levels) {
  let min = null;
  let max = null;
  for (const value of levels) {
    if (value === null || value === undefined || Number.isNaN(value)) {
      continue;
    }
    if (min === null || value < min) min = value;
    if (max === null || value > max) max = value;
  }
  return { min, max };
}

function updateSourcePolarMeta(meta, slices, frequency) {
  if (!meta) return;
  meta.innerHTML = "";
}

function renderSourcePolarChart(sourceIndex) {
  const source = currentData.database?.source_definitions?.[sourceIndex];
  if (!source) return;
  const sampleResponse = source?.responses?.[0];
  const frequencies = sampleResponse?.frequencies || [];

  const elements = getSourcePlotElements(sourceIndex);
  if (!elements.polarSelect || !elements.polarCanvas) return;

  if (!frequencies.length) {
    if (elements.polarMeta) {
      elements.polarMeta.innerHTML =
        '<div class="empty-state">No polar data available</div>';
    }
    return;
  }

  if (elements.polarSelect.options.length === 0) {
    elements.polarSelect.innerHTML = frequencies
      .map((f, i) => `<option value="${i}">${formatFrequency(f)}</option>`)
      .join("");
    const nextIndex = findNearestFrequencyIndex(frequencies, 1000);
    elements.polarSelect.value = String(nextIndex);
  }

  const freqIndex = parseInt(elements.polarSelect.value);
  if (Number.isNaN(freqIndex) || freqIndex >= frequencies.length) {
    return;
  }
  const frequency = frequencies[freqIndex];
  const slices = computePolarSlices(source, freqIndex);
  if (!slices) return;

  const normalize = elements.polarNormalize?.checked ?? false;
  let horizontalLevels = slices.horizontal.levels;
  let verticalLevels = slices.vertical.levels;

  if (normalize) {
    const hMax = Math.max(
      ...horizontalLevels.filter((v) => v !== null && !Number.isNaN(v)),
    );
    const vMax = Math.max(
      ...verticalLevels.filter((v) => v !== null && !Number.isNaN(v)),
    );
    horizontalLevels = horizontalLevels.map((v) =>
      v !== null && !Number.isNaN(v) ? v - hMax : v,
    );
    verticalLevels = verticalLevels.map((v) =>
      v !== null && !Number.isNaN(v) ? v - vMax : v,
    );
  }

  const levelRange = computeLevelRange([
    ...horizontalLevels,
    ...verticalLevels,
  ]);
  const suggestedMax = levelRange.max !== null ? levelRange.max + 3 : undefined;
  const suggestedMin =
    levelRange.max !== null ? levelRange.max - 40 : undefined;

  const ctx = elements.polarCanvas.getContext("2d");
  const existing = sourcePolarCharts.get(sourceIndex);
  if (existing) {
    existing.destroy();
  }

  const localChart = new Chart(ctx, {
    type: "radar",
    plugins: [sourcePolarCompassPlugin],
    data: {
      labels: slices.labels,
      datasets: [
        {
          label: `Horizontal @ ${formatFrequency(frequency)}${normalize ? " (normalized)" : ""}`,
          data: horizontalLevels,
          borderColor: "#2563eb",
          backgroundColor: "rgba(37, 99, 235, 0.12)",
          pointRadius: 0,
          borderWidth: 2,
          fill: true,
          tension: 0.2,
        },
        {
          label: `Vertical @ ${formatFrequency(frequency)}${normalize ? " (normalized)" : ""}`,
          data: verticalLevels,
          borderColor: "#dc2626",
          backgroundColor: "rgba(220, 38, 38, 0.12)",
          pointRadius: 0,
          borderWidth: 2,
          fill: true,
          tension: 0.2,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: sourcePolarChartInitialized.has(sourceIndex)
        ? false
        : { duration: 700 },
      layout: {
        padding: { top: 30, bottom: 30, left: 30, right: 30 },
      },
      plugins: {
        legend: {
          position: "top",
        },
        tooltip: {
          callbacks: {
            title: (items) => {
              const label = items?.[0]?.label;
              return label ? `Angle ${label}` : "";
            },
            label: (item) => {
              if (item?.raw === null || item?.raw === undefined) {
                return `${item.dataset?.label || "Level"}: -`;
              }
              return `${item.dataset?.label || "Level"}: ${item.raw.toFixed(1)} dB`;
            },
          },
        },
      },
      scales: {
        r: {
          suggestedMin,
          suggestedMax,
          startAngle: 90,
          ticks: {
            backdropColor: "transparent",
            color: "#64748b",
          },
          grid: {
            color: "rgba(148, 163, 184, 0.25)",
          },
          angleLines: {
            color: "rgba(148, 163, 184, 0.25)",
          },
          pointLabels: {
            color: "#64748b",
            font: {
              size: 10,
            },
          },
        },
      },
    },
  });

  sourcePolarCharts.set(sourceIndex, localChart);
  sourcePolarChartInitialized.add(sourceIndex);
  updateSourcePolarMeta(elements.polarMeta, slices, frequency);
}

function getSourceBalloonState(sourceIndex) {
  if (!sourceBalloonStates.has(sourceIndex)) {
    sourceBalloonStates.set(sourceIndex, {
      renderer: null,
      scene: null,
      camera: null,
      group: null,
      mesh: null,
      coverageLines: [],
      controls: null,
      frameId: null,
      active: false,
    });
  }
  return sourceBalloonStates.get(sourceIndex);
}

const sourceBalloonMaxCache = new WeakMap();

function getSourceBalloonMaxLevel(source) {
  if (!source) return null;
  const cached = sourceBalloonMaxCache.get(source);
  if (cached !== undefined) {
    return cached;
  }
  let maxLevel = null;
  const responses = source.responses || [];
  for (const resp of responses) {
    const levels = resp?.level || [];
    for (let i = 0; i < levels.length; i += 1) {
      const value = levels[i];
      if (value === null || value === undefined || Number.isNaN(value)) {
        continue;
      }
      if (maxLevel === null || value > maxLevel) {
        maxLevel = value;
      }
    }
  }
  sourceBalloonMaxCache.set(source, maxLevel);
  return maxLevel;
}

function initSourceBalloonScene(state, container, autorotate) {
  if (!container || typeof THREE === "undefined") {
    return false;
  }
  if (state.renderer && state.scene && state.camera && state.group) {
    return true;
  }

  if (state.renderer && state.renderer.domElement?.parentNode) {
    state.renderer.domElement.parentNode.removeChild(state.renderer.domElement);
  }

  state.renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
  state.renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  state.renderer.setClearColor(0x000000, 0);
  container.appendChild(state.renderer.domElement);

  state.scene = new THREE.Scene();
  state.camera = new THREE.PerspectiveCamera(45, 1, 0.1, 100);
  state.camera.position.set(0, 0.6, 2.6);
  state.camera.lookAt(0, 0, 0);

  state.group = new THREE.Group();
  state.group.userData.autoRotate = !!autorotate;

  const ambient = new THREE.AmbientLight(0xffffff, 0.65);
  const keyLight = new THREE.DirectionalLight(0xffffff, 0.85);
  keyLight.position.set(2, 2, 3);
  state.scene.add(ambient, keyLight);

  const reference = new THREE.Mesh(
    new THREE.SphereGeometry(1, 24, 16),
    new THREE.MeshBasicMaterial({
      color: 0x94a3b8,
      wireframe: true,
      opacity: 0.2,
      transparent: true,
    }),
  );
  state.group.add(reference);
  state.group.add(new THREE.AxesHelper(1.2));
  state.scene.add(state.group);

  state.controls = new OrbitControls(state.camera, state.renderer.domElement);
  state.controls.enableDamping = true;
  state.controls.enablePan = false;
  state.controls.minDistance = 1.2;
  state.controls.maxDistance = 6;

  return true;
}

function updateSourceBalloonPlaceholder(elements, show) {
  if (!elements.balloonViewer) return;
  if (elements.balloonPlaceholder) {
    elements.balloonPlaceholder.classList.toggle("hidden", !show);
  }
}

function updateSourceBalloonLegend(elements, stats) {
  if (!elements.balloonViewer) return;
  const legend = elements.balloonViewer.querySelector(".balloon-legend");
  if (!legend) return;
  const labels = legend.querySelectorAll(".balloon-legend-labels span");
  if (!stats || labels.length < 2 || stats.displayMin === undefined) {
    labels.forEach((label, idx) => {
      label.textContent = idx === 0 ? "Low" : "High";
    });
    return;
  }
  labels[0].textContent = `${stats.displayMin.toFixed(1)} dB`;
  labels[1].textContent = `${stats.displayMax.toFixed(1)} dB`;
}

function updateSourceBalloonMeta(elements, stats) {
  const meta = elements.balloonMeta;
  if (!meta) return;
  meta.innerHTML = "";
  updateSourceBalloonLegend(elements, stats);
}

function buildSourceBalloonGeometry(
  source,
  grid,
  ang,
  freqIndex,
  dbRange,
  scale,
  normalize = false,
) {
  const meridianStep = ang.meridian_step;
  const parallelStep = ang.parallel_step;
  if (!meridianStep || !parallelStep) {
    return null;
  }

  const meridianCount = Math.max(
    3,
    grid?.fullMeridianCount || Math.round(360 / meridianStep),
  );
  const parallelCount = Math.max(
    2,
    grid?.fullParallelCount || Math.round(180 / parallelStep) + 1,
  );
  const wrapMeridian = true;

  const levels = [];
  const positions = [];
  const colors = [];
  const color = new THREE.Color();

  let maxLevel = null;
  let minLevel = null;

  for (let p = 0; p < parallelCount; p += 1) {
    const parallelDeg = p * parallelStep;
    for (let m = 0; m < meridianCount; m += 1) {
      const azimuthDeg = m * meridianStep;
      const response = getResponseWithSymmetry(
        source,
        grid,
        azimuthDeg,
        parallelDeg,
      );
      const level = response?.level?.[freqIndex];
      if (level !== null && level !== undefined && !Number.isNaN(level)) {
        if (maxLevel === null || level > maxLevel) maxLevel = level;
        if (minLevel === null || level < minLevel) minLevel = level;
      }
      levels.push(level ?? null);
    }
  }

  if (maxLevel === null) {
    return null;
  }

  const globalMaxLevel = getSourceBalloonMaxLevel(source);
  const displayMax = normalize ? maxLevel : globalMaxLevel;
  if (displayMax === null) {
    return null;
  }
  const displayMin = displayMax - dbRange;
  const baseRadius = 0.3 * scale;
  const amplitude = 0.9 * scale;

  let vertexIndex = 0;
  for (let p = 0; p < parallelCount; p += 1) {
    const parallelDeg = p * parallelStep;
    const phi = (parallelDeg * Math.PI) / 180;
    for (let m = 0; m < meridianCount; m += 1) {
      const azimuthDeg = m * meridianStep;
      const theta = (azimuthDeg * Math.PI) / 180;
      const rawLevel = levels[vertexIndex];
      const level =
        rawLevel !== null && rawLevel !== undefined && !Number.isNaN(rawLevel)
          ? rawLevel
          : null;
      const normalized =
        level === null
          ? null
          : Math.min(Math.max((level - displayMin) / dbRange, 0), 1);
      const radius =
        normalized === null ? baseRadius : baseRadius + amplitude * normalized;

      const sinPhi = Math.sin(phi);
      const x = radius * sinPhi * Math.cos(theta);
      const y = radius * sinPhi * Math.sin(theta);
      const z = radius * Math.cos(phi);
      positions.push(x, y, z);

      if (normalized === null) {
        color.setRGB(0.65, 0.65, 0.65);
      } else {
        const hue = (1 - normalized) * 0.66;
        color.setHSL(hue, 0.75, 0.5);
      }
      colors.push(color.r, color.g, color.b);
      vertexIndex += 1;
    }
  }

  const indices = [];
  for (let p = 0; p < parallelCount - 1; p += 1) {
    const meridianLimit = wrapMeridian ? meridianCount : meridianCount - 1;
    for (let m = 0; m < meridianLimit; m += 1) {
      const nextM = wrapMeridian ? (m + 1) % meridianCount : m + 1;
      if (nextM >= meridianCount) {
        continue;
      }
      const a = p * meridianCount + m;
      const b = p * meridianCount + nextM;
      const c = (p + 1) * meridianCount + m;
      const d = (p + 1) * meridianCount + nextM;
      indices.push(a, c, b);
      indices.push(b, c, d);
    }
  }

  const geometry = new THREE.BufferGeometry();
  geometry.setIndex(indices);
  geometry.setAttribute(
    "position",
    new THREE.Float32BufferAttribute(positions, 3),
  );
  geometry.setAttribute("color", new THREE.Float32BufferAttribute(colors, 3));
  geometry.computeVertexNormals();

  return {
    geometry,
    stats: {
      frequency: source.responses?.[0]?.frequencies?.[freqIndex],
      minLevel,
      maxLevel,
      displayMin,
      displayMax,
      dbRange,
      meridianCount,
      parallelCount,
      symmetry: grid.symmetry,
      symmetryName: grid.symmetryName,
      normalized: normalize,
    },
  };
}

function buildSourceCoverageContours(
  source,
  grid,
  ang,
  freqIndex,
  dbRange,
  scale,
  normalize = false,
) {
  const meridianStep = ang.meridian_step;
  const parallelStep = ang.parallel_step;
  if (!meridianStep || !parallelStep) return null;

  const meridianCount = Math.max(
    3,
    grid?.fullMeridianCount || Math.round(360 / meridianStep),
  );
  const parallelCount = Math.max(
    2,
    grid?.fullParallelCount || Math.round(180 / parallelStep) + 1,
  );

  const onAxisResp = getResponseWithSymmetry(source, grid, 0, 0);
  const onAxisLevel = onAxisResp?.level?.[freqIndex];
  if (onAxisLevel == null || Number.isNaN(onAxisLevel)) return null;

  const globalMaxLevel = getSourceBalloonMaxLevel(source);
  const displayMax = normalize ? onAxisLevel : globalMaxLevel;
  if (displayMax === null) return null;
  const displayMin = displayMax - dbRange;
  const baseRadius = 0.3 * scale;
  const amplitude = 0.9 * scale;

  const thresholds = [3, 6, 9];
  const contours = [];

  for (const cm of thresholds) {
    const points = [];
    for (let m = 0; m < meridianCount; m++) {
      const azimuthDeg = m * meridianStep;
      for (let p = 1; p < parallelCount; p++) {
        const pDeg = p * parallelStep;
        const resp = getResponseWithSymmetry(source, grid, azimuthDeg, pDeg);
        const level = resp?.level?.[freqIndex];
        if (level == null || Number.isNaN(level)) continue;
        const drop = onAxisLevel - level;
        if (drop > cm) {
          const prevDeg = (p - 1) * parallelStep;
          const prevResp = getResponseWithSymmetry(
            source,
            grid,
            azimuthDeg,
            prevDeg,
          );
          const prevLevel = prevResp?.level?.[freqIndex];
          let contourDeg;
          if (prevLevel == null || Number.isNaN(prevLevel)) {
            contourDeg = pDeg;
          } else {
            const prevDrop = onAxisLevel - prevLevel;
            const frac = (cm - prevDrop) / (drop - prevDrop);
            contourDeg = prevDeg + frac * parallelStep;
          }
          const contourLevel = onAxisLevel - cm;
          const norm = Math.min(
            Math.max((contourLevel - displayMin) / dbRange, 0),
            1,
          );
          const radius = baseRadius + amplitude * norm;
          const phi = (contourDeg * Math.PI) / 180;
          const theta = (azimuthDeg * Math.PI) / 180;
          points.push(
            new THREE.Vector3(
              radius * Math.sin(phi) * Math.cos(theta),
              radius * Math.sin(phi) * Math.sin(theta),
              radius * Math.cos(phi),
            ),
          );
          break;
        }
      }
    }
    if (points.length > 2) {
      points.push(points[0].clone());
      contours.push({ threshold: cm, points });
    }
  }

  return contours.length > 0 ? contours : null;
}

function handleSourceBalloonRangeInput(sourceIndex) {
  const elements = getSourcePlotElements(sourceIndex);
  if (!elements.balloonRange || !elements.balloonRangeValue) return;
  elements.balloonRangeValue.textContent = elements.balloonRange.value;
  renderSourceBalloon(sourceIndex);
}

function handleSourceBalloonFrequencySlider(sourceIndex) {
  const elements = getSourcePlotElements(sourceIndex);
  if (!elements.balloonSlider || !elements.balloonSelect) return;
  elements.balloonSelect.value = String(elements.balloonSlider.value);
  renderSourceBalloon(sourceIndex);
}

function handleSourceBalloonScaleInput(sourceIndex) {
  const elements = getSourcePlotElements(sourceIndex);
  if (!elements.balloonScale || !elements.balloonScaleValue) return;
  elements.balloonScaleValue.textContent = `${Number(elements.balloonScale.value).toFixed(1)}×`;
  renderSourceBalloon(sourceIndex);
}

function renderSourceBalloon(sourceIndex) {
  const source = currentData.database?.source_definitions?.[sourceIndex];
  if (!source) return;
  const elements = getSourcePlotElements(sourceIndex);

  const balloon = source?.definition?.balloon_data;
  const ang = balloon?.angular_resolution;
  const grid = getBalloonGrid(source);
  if (!balloon || !ang || !grid) {
    updateSourceBalloonPlaceholder(elements, true);
    updateSourceBalloonMeta(elements, null);
    return;
  }

  if (!elements.balloonViewer || !elements.balloonSelect) {
    return;
  }

  const sampleResponse = source?.responses?.[0];
  const frequencies = sampleResponse?.frequencies || [];
  if (!frequencies.length) {
    updateSourceBalloonPlaceholder(elements, true);
    updateSourceBalloonMeta(elements, null);
    return;
  }

  if (elements.balloonSelect.options.length === 0) {
    elements.balloonSelect.innerHTML = frequencies
      .map((f, i) => `<option value="${i}">${formatFrequency(f)}</option>`)
      .join("");
    const nextIndex = findNearestFrequencyIndex(frequencies, 1000);
    elements.balloonSelect.value = String(nextIndex);
  }

  const freqIndex = parseInt(elements.balloonSelect.value);
  if (Number.isNaN(freqIndex) || freqIndex >= frequencies.length) {
    updateSourceBalloonPlaceholder(elements, true);
    updateSourceBalloonMeta(elements, null);
    return;
  }

  if (elements.balloonSlider) {
    elements.balloonSlider.max = String(Math.max(0, frequencies.length - 1));
    elements.balloonSlider.value = String(freqIndex);
  }
  if (elements.balloonSliderValue) {
    elements.balloonSliderValue.textContent = formatFrequency(
      frequencies[freqIndex],
    );
  }
  if (elements.balloonRangeValue && elements.balloonRange) {
    elements.balloonRangeValue.textContent = elements.balloonRange.value;
  }
  if (elements.balloonScaleValue && elements.balloonScale) {
    elements.balloonScaleValue.textContent = `${Number(elements.balloonScale.value).toFixed(1)}×`;
  }

  const state = getSourceBalloonState(sourceIndex);
  const autorotate = elements.balloonAutorotate?.checked ?? true;
  const sceneReady = initSourceBalloonScene(
    state,
    elements.balloonViewer,
    autorotate,
  );
  if (!sceneReady) {
    updateSourceBalloonPlaceholder(elements, true);
    updateSourceBalloonMeta(elements, null);
    return;
  }

  const width = elements.balloonViewer.clientWidth;
  const height = elements.balloonViewer.clientHeight;
  if (width > 0 && height > 0) {
    state.renderer.setSize(width, height);
    state.camera.aspect = width / height;
    state.camera.updateProjectionMatrix();
  }

  updateSourceBalloonPlaceholder(elements, false);
  state.group.userData.autoRotate = autorotate;

  const rangeValue = Number(elements.balloonRange?.value);
  const scaleValue = Number(elements.balloonScale?.value);
  const normalize = elements.balloonNormalize?.checked ?? false;
  const dbRange = Number.isFinite(rangeValue) ? rangeValue : 40;
  const scale = Number.isFinite(scaleValue) ? scaleValue : 1;

  const geometryData = buildSourceBalloonGeometry(
    source,
    grid,
    ang,
    freqIndex,
    dbRange,
    scale,
    normalize,
  );

  if (!geometryData) {
    if (state.mesh) {
      state.group?.remove(state.mesh);
      state.mesh.geometry?.dispose?.();
      state.mesh.material?.dispose?.();
      state.mesh = null;
    }
    if (state.coverageLines?.length) {
      state.coverageLines.forEach((line) => {
        state.group?.remove(line);
        line.geometry?.dispose?.();
        line.material?.dispose?.();
      });
      state.coverageLines = [];
    }
    updateSourceBalloonPlaceholder(elements, true);
    updateSourceBalloonMeta(elements, null);
    return;
  }

  const { geometry, stats } = geometryData;

  if (state.mesh) {
    state.group?.remove(state.mesh);
    state.mesh.geometry?.dispose?.();
    state.mesh.material?.dispose?.();
  }

  const wireframe = !!elements.balloonWireframe?.checked;
  const material = new THREE.MeshStandardMaterial({
    vertexColors: true,
    flatShading: false,
    metalness: 0.1,
    roughness: 0.65,
    wireframe,
  });

  state.mesh = new THREE.Mesh(geometry, material);
  state.group?.add(state.mesh);

  for (const line of state.coverageLines || []) {
    state.group?.remove(line);
    line.geometry?.dispose?.();
    line.material?.dispose?.();
  }
  state.coverageLines = [];

  if (elements.balloonCoverage?.checked) {
    const contours = buildSourceCoverageContours(
      source,
      grid,
      ang,
      freqIndex,
      dbRange,
      scale,
      normalize,
    );
    if (contours) {
      const colorMap = { 3: 0xffffff, 6: 0xffff00, 9: 0xff4444 };
      for (const c of contours) {
        const geom = new THREE.BufferGeometry().setFromPoints(c.points);
        const mat = new THREE.LineBasicMaterial({
          color: colorMap[c.threshold] || 0xffffff,
          linewidth: 2,
          depthTest: true,
        });
        const line = new THREE.Line(geom, mat);
        state.group?.add(line);
        state.coverageLines.push(line);
      }
    }
  }

  updateSourceBalloonMeta(elements, stats);
  startSourceBalloonLoop(sourceIndex);
}

function startSourceBalloonLoop(sourceIndex) {
  const state = getSourceBalloonState(sourceIndex);
  if (!state || !state.renderer || !state.scene || !state.camera) return;
  if (state.frameId) {
    cancelAnimationFrame(state.frameId);
  }
  state.active = true;
  const animate = () => {
    if (!state.active) return;
    state.frameId = requestAnimationFrame(animate);
    if (state.group && state.group.userData.autoRotate) {
      state.group.rotation.y += 0.0035;
    }
    state.controls?.update();
    state.renderer.render(state.scene, state.camera);
  };
  animate();
}

function stopSourceBalloonLoop(sourceIndex) {
  const state = sourceBalloonStates.get(sourceIndex);
  if (!state) return;
  state.active = false;
  if (state.frameId) {
    cancelAnimationFrame(state.frameId);
    state.frameId = null;
  }
}

function destroySourceBalloonStates() {
  sourceBalloonStates.forEach((state) => {
    if (state.frameId) {
      cancelAnimationFrame(state.frameId);
    }
    if (state.mesh) {
      state.mesh.geometry?.dispose?.();
      state.mesh.material?.dispose?.();
      state.mesh = null;
    }
    if (state.coverageLines?.length) {
      state.coverageLines.forEach((line) => {
        line.geometry?.dispose?.();
        line.material?.dispose?.();
      });
      state.coverageLines = [];
    }
    if (state.renderer) {
      state.renderer.dispose();
      if (state.renderer.domElement?.parentNode) {
        state.renderer.domElement.parentNode.removeChild(
          state.renderer.domElement,
        );
      }
    }
    state.controls?.dispose?.();
  });
}

function renderSourceImpedanceChart(sourceIndex) {
  const source = currentData.database?.source_definitions?.[sourceIndex];
  if (!source) return;
  const impedance = source?.definition?.impedance;
  if (!impedance || !Array.isArray(impedance.level)) return;

  const elements = getSourcePlotElements(sourceIndex);
  if (!elements.impedanceCanvas) return;

  const frequencies = buildLogFrequencies(
    impedance.definition,
    impedance.level.length,
  );
  if (!frequencies) {
    return;
  }

  const delay = Number.isFinite(impedance.delay) ? impedance.delay : 0;
  const rawPhase = impedance.phase || [];
  const delayAdjustedPhase = applyDelayToPhase(rawPhase, frequencies, delay);
  const unwrappedPhase = unwrapPhase(delayAdjustedPhase);
  const phaseMode = elements.impedancePhase?.value || "unwrapped";
  const phaseSeries = getPhaseSeries(
    phaseMode,
    frequencies,
    delayAdjustedPhase,
    unwrappedPhase,
  );

  const levelSeries = buildFrequencyPoints(frequencies, impedance.level);
  const phaseData = buildFrequencyPoints(frequencies, phaseSeries.values);
  if (!levelSeries) return;

  const ctx = elements.impedanceCanvas.getContext("2d");
  const existing = sourceImpedanceCharts.get(sourceIndex);
  if (existing) {
    existing.destroy();
  }

  const localChart = new Chart(ctx, {
    type: "line",
    data: {
      datasets: [
        {
          label: "Impedance (level)",
          data: levelSeries.points,
          borderColor: "#0f766e",
          backgroundColor: "rgba(15, 118, 110, 0.1)",
          fill: true,
          tension: 0.3,
          pointRadius: 0,
          yAxisID: "y",
        },
        {
          label: phaseSeries.label,
          data: phaseData ? phaseData.points : [],
          borderColor: "#f97316",
          backgroundColor: "transparent",
          tension: 0.3,
          pointRadius: 0,
          yAxisID: "y1",
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: { duration: 600 },
      interaction: {
        mode: "index",
        intersect: false,
      },
      scales: {
        x: buildLogFrequencyScale(
          levelSeries.minFrequency,
          levelSeries.maxFrequency,
          "Frequency",
        ),
        y: {
          type: "linear",
          display: true,
          position: "left",
          title: {
            display: true,
            text: "Level",
          },
        },
        y1: {
          type: "linear",
          display: true,
          position: "right",
          title: {
            display: true,
            text: phaseSeries.axisTitle,
          },
          grid: {
            drawOnChartArea: false,
          },
        },
      },
    },
  });

  sourceImpedanceCharts.set(sourceIndex, localChart);

  if (elements.impedanceMeta) {
    const chips = [];
    chips.push(`<span class="chip">${impedance.level.length} points</span>`);
    chips.push(
      `<span class="chip">${formatFrequency(frequencies[0])} - ${formatFrequency(frequencies[frequencies.length - 1])}</span>`,
    );
    if (delay) {
      chips.push(`<span class="chip">Delay ${formatDelay(delay)}</span>`);
    }
    elements.impedanceMeta.innerHTML = chips.join("");
  }
}

function buildSourcePlacementsMap(boxTypes) {
  // Group placements by source definition key
  const map = new Map();
  boxTypes.forEach((box) => {
    const placements = box?.source_placements || [];
    placements.forEach((placement) => {
      const defKey = placement?.source_def_key;
      if (!defKey) return;
      const list = map.get(defKey) || [];
      list.push({ box, source: placement });
      map.set(defKey, list);
    });
  });
  return map;
}

function displayConfig() {
  // Render box types, frames, filters, limits, and warnings
  const db = currentData.database;

  // Box Types
  const boxTypesList = document.getElementById("box-types-list");
  const boxTypes = db?.box_types || [];
  if (boxTypes.length === 0) {
    boxTypesList.innerHTML =
      '<div class="empty-state">No box types defined</div>';
  } else {
    boxTypesList.innerHTML = boxTypes
      .map(
        (box, index) => `
            <div class="config-item">
                <div class="config-item-header-row">
                    <div class="config-item-header">${escapeHtml(box.label)}</div>
                    ${formatGeometryActions("box", index, box.case_geometry, box.label || box.key)}
                </div>
                ${formatBoxTypeDetail(box)}
                ${formatGeometryDetail(box.case_geometry)}
                ${formatInlineGeometryViewer("box", index, box.case_geometry)}
            </div>
        `,
      )
      .join("");
  }

  // Frames
  const framesList = document.getElementById("frames-list");
  const frames = db?.frames || [];
  if (frames.length === 0) {
    framesList.innerHTML = '<div class="empty-state">No frames defined</div>';
  } else {
    framesList.innerHTML = frames
      .map(
        (frame, index) => `
            <div class="config-item">
                <div class="config-item-header-row">
                    <div class="config-item-header">${escapeHtml(frame.label)}</div>
                    ${formatGeometryActions(
                      "frame",
                      index,
                      frame.case_geometry,
                      frame.label || frame.key,
                    )}
                </div>
                ${formatGeometryDetail(frame.case_geometry)}
                ${formatInlineGeometryViewer("frame", index, frame.case_geometry)}
            </div>
        `,
      )
      .join("");
  }

  // Filter Groups
  const filterGroupsList = document.getElementById("filter-groups-list");
  const filterGroups = db?.filter_groups || [];
  if (filterGroups.length === 0) {
    filterGroupsList.innerHTML =
      '<div class="empty-state">No filter groups defined</div>';
  } else {
    filterGroupsList.innerHTML = filterGroups
      .map(
        (fg, fgIdx) => `
            <div class="filter-group" data-group-idx="${fgIdx}">
                <div class="filter-group-header" onclick="toggleFilterGroup(${fgIdx})">
                    <span class="filter-group-toggle">▶</span>
                    <span class="filter-group-title">${escapeHtml(fg.label)}</span>
                    <span class="filter-group-meta">
                        ${fg.filters?.length ? `${fg.filters.length} filter(s)` : ""}
                        ${fg.is_overridable ? " • Overridable" : ""}
                    </span>
                </div>
                <div class="filter-group-content" style="display: none;">
                    <div class="filter-group-key">Key: ${escapeHtml(fg.key)}</div>
                    <div class="filter-group-response">
                      <div class="filter-response-controls">
                        <label>
                          Filter:
                          <select id="filter-group-response-filter-${fgIdx}">
                            ${
                              fg.filters?.length
                                ? fg.filters
                                    .map(
                                      (filter, i) =>
                                        `<option value="${i}">${escapeHtml(filter.label || filter.key || `Filter ${i + 1}`)}</option>`,
                                    )
                                    .join("")
                                : '<option value="">No filters</option>'
                            }
                          </select>
                        </label>
                        <label>
                          Phase:
                          <select id="filter-group-response-phase-${fgIdx}">
                            <option value="unwrapped" selected>Unwrapped</option>
                            <option value="wrapped">Wrapped</option>
                            <option value="group-delay">Group delay</option>
                          </select>
                        </label>
                        <div class="dropdown-container filter-export-dropdown">
                          <button class="btn-download btn-filter-export" data-group-idx="${fgIdx}">
                            Export <span class="dropdown-icon">▼</span>
                          </button>
                          <div class="dropdown-menu">
                            <button class="dropdown-item" data-format="frd">Combined .frd</button>
                            <button class="dropdown-item" data-format="csv">Combined .csv</button>
                            <button class="dropdown-item" data-format="xgfb">Filter Bank .xgfb</button>
                          </div>
                        </div>
                      </div>
                      <div class="filter-response-chart-container">
                        <div
                          id="filter-group-response-placeholder-${fgIdx}"
                          class="filter-response-placeholder"
                        >
                          No filter response data loaded
                        </div>
                        <canvas id="filter-group-response-chart-${fgIdx}"></canvas>
                      </div>
                      <div
                        id="filter-group-response-meta-${fgIdx}"
                        class="response-meta"
                      ></div>
                    </div>
                    ${
                      fg.filters?.length
                        ? fg.filters
                            .map(
                              (f) => `
                            <div class="filter-definition">
                                <div class="filter-def-header">${escapeHtml(f.label || f.key)}</div>
                                ${f.key && f.key !== f.label ? `<div class="filter-def-key">Key: ${escapeHtml(f.key)}</div>` : ""}
                                ${renderFilterDetails(f)}
                            </div>
                        `,
                            )
                            .join("")
                        : '<div class="filter-empty">No filters in group</div>'
                    }
                </div>
            </div>
        `,
      )
      .join("");
  }
  wireFilterGroupResponses();
  wireFilterExportDropdowns();

  // Render limits section
  // Limits
  const limitsList = document.getElementById("limits-list");
  const limits = db?.limits || [];
  if (limits.length === 0) {
    limitsList.innerHTML = '<div class="empty-state">No limits defined</div>';
  } else {
    limitsList.innerHTML = limits
      .map(
        (limit) => `
            <div class="config-item">
                <div class="config-item-header">${formatLimitType(limit.type)}</div>
                <div class="config-item-detail">
                    Value: ${limit.limit_value}
                    ${limit.box_type ? ` • Box: ${escapeHtml(limit.box_type)}` : ""}
                    ${limit.frame ? ` • Frame: ${escapeHtml(limit.frame)}` : ""}
                </div>
            </div>
        `,
      )
      .join("");
  }

  // Render warnings section
  // Warnings
  const warningsList = document.getElementById("warnings-list");
  const warnings = db?.warnings || [];
  if (warnings.length === 0) {
    warningsList.innerHTML =
      '<div class="empty-state">No warnings defined</div>';
  } else {
    warningsList.innerHTML = warnings
      .map(
        (warning) => `
            <div class="config-item">
                <div class="config-item-header">${formatWarningType(warning.type)}</div>
                <div class="config-item-detail">
                    ${escapeHtml(warning.text || "")}
                    ${warning.limit_value ? ` (Value: ${warning.limit_value})` : ""}
                </div>
            </div>
        `,
      )
      .join("");
  }

  wireGeometryDownloads();

  // Update card counts
  document.getElementById("limits-count").textContent =
    limits.length > 0 ? limits.length : "";
  document.getElementById("warnings-count").textContent =
    warnings.length > 0 ? warnings.length : "";
}

function displayConfigurations() {
  // Show cluster setups, connectors, and defaults
  const db = currentData?.database;
  const clusterSetups = db?.cluster_setups || [];

  // Cluster Setups
  const clusterList = document.getElementById("cluster-setups-list");
  if (clusterList) {
    if (clusterSetups.length === 0) {
      clusterList.innerHTML =
        '<div class="empty-state">No cluster setups defined</div>';
    } else {
      clusterList.innerHTML = clusterSetups
        .map(
          (cs, idx) => `
          <div class="cluster-setup-item" data-cluster-index="${idx}">
            <div class="cluster-setup-header">
              <h4>${escapeHtml(cs.label || cs.key)}</h4>
              <button class="btn-use-config" onclick="loadClusterSetupToEditor(${idx})">Load into Editor</button>
            </div>
            ${cs.setup?.description ? `<div style="color:var(--text-muted);font-size:0.8125rem;margin-bottom:0.5rem">${escapeHtml(cs.setup.description)}</div>` : ""}
            <div class="cluster-setup-boxes">
              <table>
                <thead><tr>
                  <th>Label</th><th>Box Type</th>
                  <th>X (mm)</th><th>Y (mm)</th><th>Z (mm)</th>
                  <th>H (&deg;)</th><th>V (&deg;)</th><th>R (&deg;)</th>
                </tr></thead>
                <tbody>
                  ${(cs.setup?.boxes || [])
                    .map(
                      (box) => `<tr>
                    <td>${escapeHtml(box.label || "")}</td>
                    <td>${escapeHtml(box.box_type_key || "")}</td>
                    <td>${formatNumber(box.position?.x || 0)}</td>
                    <td>${formatNumber(box.position?.y || 0)}</td>
                    <td>${formatNumber(box.position?.z || 0)}</td>
                    <td>${formatNumber(toDeg(box.angles?.x || 0))}</td>
                    <td>${formatNumber(toDeg(box.angles?.y || 0))}</td>
                    <td>${formatNumber(toDeg(box.angles?.z || 0))}</td>
                  </tr>`,
                    )
                    .join("")}
                </tbody>
              </table>
            </div>
          </div>`,
        )
        .join("");
    }
  }

  // Connectors
  displayConnectors(db);

  // Populate frame selector for auto-generation
  populateFrameSelect();

  // Auto-select default configuration
  autoSelectDefaultConfig();

  // Default configuration if no setup data exists
  ensureDefaultConfiguration();
}

function displayTransformers() {
  // Render transformer metadata and tap settings
  const transformersList = document.getElementById("transformers-list");
  if (!transformersList) return;

  const transformers = currentData?.database?.transformers || [];
  if (transformers.length === 0) {
    transformersList.innerHTML =
      '<div class="empty-state">No transformer settings defined</div>';
    return;
  }

  transformersList.innerHTML = transformers
    .map((transformer) => {
      const taps = transformer.tap_settings || [];
      const maxPower = transformer.max_power;
      const tapRows = taps.length
        ? `<table class="transformer-table">
            <thead>
              <tr>
                <th>Tap</th>
                <th>Key</th>
                <th>Power Ratio</th>
                <th>Power (W)</th>
              </tr>
            </thead>
            <tbody>
              ${taps
                .map((tap) => {
                  const ratio = Number.isFinite(tap.power_ratio)
                    ? tap.power_ratio
                    : null;
                  const power =
                    Number.isFinite(maxPower) && Number.isFinite(ratio)
                      ? maxPower * ratio
                      : null;
                  return `<tr>
                    <td>${escapeHtml(tap.label || "(unnamed)")}</td>
                    <td>${escapeHtml(tap.key || "—")}</td>
                    <td>${ratio === null ? "—" : formatNumber(ratio, 2)}</td>
                    <td>${power === null ? "—" : formatNumber(power, 1)}</td>
                  </tr>`;
                })
                .join("")}
            </tbody>
          </table>`
        : '<div class="empty-state">No tap settings defined</div>';

      return `
        <div class="config-item">
          <div class="config-item-header">${escapeHtml(transformer.label || "Transformer")}</div>
          <div class="config-item-detail">
            ${transformer.key ? `Key: ${escapeHtml(transformer.key)} • ` : ""}
            Max Power: ${formatNumber(transformer.max_power, 1)} W
            • Net Voltage: ${formatNumber(transformer.net_voltage, 1)} V
            • Impedance: ${formatNumber(transformer.lspk_impedance, 1)} Ω
          </div>
          <div class="transformer-taps">
            ${tapRows}
          </div>
        </div>
      `;
    })
    .join("");
}

function displayConnectors(db) {
  // Render connector table grouped by frame
  const connectors = db?.connectors || [];
  const connectorsEl = document.getElementById("connectors-list");
  if (!connectorsEl) return;

  if (connectors.length === 0) {
    connectorsEl.innerHTML =
      '<div class="empty-state">No connectors defined</div>';
    return;
  }

  // Group connectors by frame
  const byFrame = new Map();
  for (const c of connectors) {
    const frame = c.frame || "(no frame)";
    if (!byFrame.has(frame)) byFrame.set(frame, []);
    byFrame.get(frame).push(c);
  }

  let html = "";
  for (const [frame, conns] of byFrame) {
    html += `<div class="connector-frame-group">`;
    html += `<h4 class="connector-frame-label">${escapeHtml(frame)}</h4>`;
    html += `<table><thead><tr>
      <th>Upper Box</th><th>Lower Box</th><th>Splay Angles</th>
    </tr></thead><tbody>`;
    for (const c of conns) {
      const angles = (c.angles || [])
        .map((a) => `${a.value.toFixed(1)}\u00B0`)
        .join(", ");
      html += `<tr>
        <td>${escapeHtml(c.upper_box)}</td>
        <td>${escapeHtml(c.lower_box)}</td>
        <td class="connector-angles">${angles || "\u2014"}</td>
      </tr>`;
    }
    html += "</tbody></table></div>";
  }

  connectorsEl.innerHTML = html;
}

function toDeg(rad) {
  // Convert radians to degrees
  return (rad * 180) / Math.PI;
}

function toRad(deg) {
  // Convert degrees to radians
  return (deg * Math.PI) / 180;
}

function populateFrameSelect() {
  const select = document.getElementById("config-frame-select");
  if (!select) return;
  const frames = currentData?.database?.frames || [];
  if (frames.length === 0) {
    select.innerHTML = '<option value="">No frames</option>';
    select.disabled = true;
    return;
  }
  select.disabled = false;
  select.innerHTML = frames
    .map(
      (f, i) =>
        `<option value="${i}">${escapeHtml(f.label || f.key)} (${f.type_flown ? "flown" : "GS"})</option>`,
    )
    .join("");
}

function autoGenerateFromFrame() {
  const select = document.getElementById("config-frame-select");
  const db = currentData?.database;
  if (!select || !db) return;
  const frameIndex = parseInt(select.value, 10);
  const frames = db.frames || [];
  const frame = frames[frameIndex];
  if (!frame) return;

  const config = buildFrameConfig(db, frame);
  if (!config || !config.rows.length) {
    updateConfigEditorHint(
      `No valid configuration could be generated for frame "${frame.label || frame.key}".`,
    );
    return;
  }

  const tbody = document.getElementById("config-editor-body");
  if (tbody) {
    tbody.innerHTML = "";
    config.rows.forEach((row) =>
      addConfigEditorRow(row.box_type_key, row.splay_deg),
    );
  }

  const elements = buildArrayElementsFromRows(config.rows);
  activeConfig = {
    elements,
    label: config.label,
    isDemo: false,
  };

  updateConfigEditorHint(config.label);
  scheduleArrayViewerUpdate();
  markConfigDirty();
}

let configApplyTimer = null;

function scheduleConfigApply() {
  // Debounce config application so rapid edits don't trigger many recomputations
  clearTimeout(configApplyTimer);
  configApplyTimer = setTimeout(() => {
    applyConfigFromEditor();
  }, 400);
}

function setupConfigEditor() {
  // Wire config editor buttons
  const addBtn = document.getElementById("config-add-element");
  const clearBtn = document.getElementById("config-clear");
  const autogenBtn = document.getElementById("config-autogen");

  if (autogenBtn) {
    autogenBtn.onclick = () => autoGenerateFromFrame();
  }
  if (addBtn) {
    addBtn.onclick = () => addConfigEditorRow();
  }
  if (clearBtn) {
    clearBtn.onclick = () => {
      document.getElementById("config-editor-body").innerHTML = "";
      activeConfig = null;
      cachedArrayBalloon = null;
      setArrayVisualizationState({
        status: "empty",
        error: null,
        computedAt: null,
        sourceCount: 0,
      });
      updateConfigEditorHint("Add elements to build a configuration.");
      scheduleArrayViewerUpdate();
      markConfigDirty();
    };
  }

  const table = document.getElementById("config-editor-table");
  if (table) {
    table.addEventListener("click", (event) => {
      const removeButton = event.target.closest(".btn-remove");
      if (!removeButton) return;

      const row = removeButton.closest("tr");
      if (!row) return;

      row.remove();
      applyConfigFromEditor();
    });
    table.addEventListener("input", () => {
      scheduleArrayViewerUpdate();
      scheduleConfigApply();
    });
    table.addEventListener("change", () => {
      scheduleArrayViewerUpdate();
      scheduleConfigApply();
    });
  }
}

function getBoxTypeOptions() {
  // Build select options for box types
  const boxTypes = currentData?.database?.box_types || [];
  return boxTypes
    .map(
      (bt) =>
        `<option value="${escapeHtml(bt.key)}">${escapeHtml(bt.label || bt.key)}</option>`,
    )
    .join("");
}

function addConfigEditorRow(boxTypeKey, splayDeg) {
  // Insert an editable config row into the table
  const tbody = document.getElementById("config-editor-body");
  if (!tbody) return;
  const boxTypes = currentData?.database?.box_types || [];
  const row = document.createElement("tr");
  const options = boxTypes
    .map(
      (bt) =>
        `<option value="${escapeHtml(bt.key)}"${bt.key === boxTypeKey ? " selected" : ""}>${escapeHtml(bt.label || bt.key)}</option>`,
    )
    .join("");
  row.innerHTML = `
    <td><select class="cfg-box-type">${options}</select></td>
    <td><input type="number" class="cfg-splay" value="${formatNumber(splayDeg ?? 0)}" step="0.5"></td>
    <td><button type="button" class="btn-remove">X</button></td>
  `;
  tbody.appendChild(row);
  scheduleArrayViewerUpdate();
  scheduleConfigApply();
}

function readConfigEditorRows() {
  // Parse editor table rows into splay-only rows
  const tbody = document.getElementById("config-editor-body");
  if (!tbody) return [];

  const rows = tbody.querySelectorAll("tr");
  const elements = [];
  for (const row of rows) {
    const boxTypeKey = row.querySelector(".cfg-box-type")?.value;
    const splay = parseFloat(row.querySelector(".cfg-splay")?.value) || 0;
    if (boxTypeKey) {
      elements.push({
        box_type_key: boxTypeKey,
        splay_deg: splay,
      });
    }
  }
  return elements;
}

// Load a cluster setup into the editor
window.loadClusterSetupToEditor = function (clusterIndex) {
  // Populate editor rows from a stored cluster setup
  const clusterSetups = currentData?.database?.cluster_setups || [];
  const cs = clusterSetups[clusterIndex];
  if (!cs) return;

  const tbody = document.getElementById("config-editor-body");
  if (tbody) tbody.innerHTML = "";
  updateConfigEditorHint("");

  for (const box of cs.setup?.boxes || []) {
    const splay = toDeg(box.angles?.y || 0);
    addConfigEditorRow(box.box_type_key, -splay);
  }
  scheduleArrayViewerUpdate();
};

function applyConfigFromEditor() {
  // Convert editor rows into active configuration
  const rows = readConfigEditorRows();
  const elements = buildArrayElementsFromRows(rows);

  if (elements.length === 0) {
    activeConfig = null;
    cachedArrayBalloon = null;
    setArrayVisualizationState({
      status: "empty",
      error: null,
      computedAt: null,
      sourceCount: 0,
    });
    updateConfigEditorHint("Add elements to build a configuration.");
  } else {
    activeConfig = { elements, label: `Config (${elements.length} boxes)` };
    updateConfigEditorHint("");
  }

  scheduleArrayViewerUpdate();
  markConfigDirty();
}

function autoSelectDefaultConfig() {
  // Choose a reasonable default configuration per system type
  const db = currentData?.database;
  const systemType = currentData?.gen_system?.type;

  // type 1 = Cluster, type 0 = LineArray, type 2 = Loudspeaker
  if (systemType === 1) {
    // Cluster: use first cluster setup
    const clusterSetups = db?.cluster_setups || [];
    if (clusterSetups.length > 0) {
      const cs = clusterSetups[0];
      const elements = (cs.setup?.boxes || []).map((box) => ({
        box_type_key: box.box_type_key,
        position: box.position || { x: 0, y: 0, z: 0 },
        angles: box.angles || { x: 0, y: 0, z: 0 },
        gain: 0,
      }));
      if (elements.length > 0) {
        activeConfig = { elements };
        activeConfig.label = `Config (${elements.length} boxes)`;
        // Also populate editor
        window.loadClusterSetupToEditor(0);
        updateConfigEditorHint("");
      }
    }
  }
  // For Loudspeaker (type 2), the existing box-type dropdown works fine
  // For LineArray (type 0), frame-based config is built in ensureDefaultConfiguration
}

function findConnectableBoxTypes(db, previousKey, frameKey, isFlown) {
  // Mimic legacy ArrayBuilder.DetermineNextPossibleBoxTypes:
  // Find box types that can connect after previousKey for the given frame.
  const connectors = db?.connectors || [];
  const boxTypes = db?.box_types || [];
  const boxKeys = new Set(boxTypes.map((bt) => bt.key));
  const result = [];
  for (const bt of boxTypes) {
    const candidateKey = bt.key;
    // For flown: previous=upper, candidate=lower
    // For ground-stacked: previous=lower, candidate=upper
    const upper = isFlown ? previousKey : candidateKey;
    const lower = isFlown ? candidateKey : previousKey;
    const found = connectors.some(
      (c) =>
        c.upper_box === upper &&
        c.lower_box === lower &&
        (c.frame === frameKey || c.frame === ""),
    );
    if (found) result.push(candidateKey);
  }
  return result;
}

function getDefaultSplayAngles(count) {
  // Legacy default splay pattern: [0, 0, 0, 0, 2, 4, 6, 8] degrees
  const defaults = [0, 0, 0, 0, 2, 4, 6, 8];
  const result = [];
  for (let i = 0; i < count; i++) {
    result.push(
      i < defaults.length ? defaults[i] : defaults[defaults.length - 1],
    );
  }
  return result;
}

function buildFrameConfig(db, frame) {
  // Build a default config for a specific frame, mirroring legacy SwitchToFrame logic.
  const connectors = db?.connectors || [];
  const boxTypes = db?.box_types || [];
  if (!connectors.length || !boxTypes.length) return null;

  const isFlown = frame.type_flown;
  const frameKey = frame.key;
  const desiredCount = isFlown ? 8 : 4;

  // TODO: when limits are parsed, clamp desiredCount to [min, max]
  const count = desiredCount;

  const splayAngles = getDefaultSplayAngles(count);
  const rows = [];

  // Build elements iteratively like legacy _E001 method
  let previousKey = frameKey; // First element connects to frame
  for (let i = 0; i < count; i++) {
    const remaining = count - i;
    const possible = findConnectableBoxTypes(
      db,
      previousKey,
      frameKey,
      isFlown,
    );
    if (possible.length === 0) {
      if (rows.length === 0) return null; // Frame has no valid connections
      break; // Partial config is ok
    }
    // Prefer same box type as previous if compatible, otherwise first match
    const boxKey = possible.includes(previousKey) ? previousKey : possible[0];
    rows.push({ box_type_key: boxKey, splay_deg: splayAngles[i] });
    previousKey = boxKey;
  }

  if (rows.length === 0) return null;

  const typeLabel = isFlown ? "flown" : "ground-stacked";
  return {
    rows,
    label: `${frame.label} (${rows.length} boxes, ${typeLabel})`,
    frameKey,
    isDemo: false,
  };
}

function buildConnectorDefaultConfig(db) {
  // Build a default array config by trying each frame in order (like legacy SwitchToDefaultConfig).
  const frames = db?.frames || [];
  const connectors = db?.connectors || [];
  const boxTypes = db?.box_types || [];
  if (!connectors.length || !boxTypes.length) return null;

  // Try each frame, return first one that produces a valid config
  for (const frame of frames) {
    const config = buildFrameConfig(db, frame);
    if (config && config.rows.length > 0) return config;
  }

  return null;
}

function buildFallbackDefaultConfig(db) {
  // Create a demo config when no setup data exists
  const boxTypes = db?.box_types || [];
  if (!boxTypes.length) {
    updateConfigEditorHint("No box types available for demo configuration.");
    return null;
  }

  const demoBox = boxTypes.find((box) => {
    const placements = box?.source_placements || [];
    const sources = box?.sources || [];
    return placements.length > 0 || sources.length > 0;
  });

  if (!demoBox) {
    updateConfigEditorHint("No box types with sources available for demo.");
    return null;
  }

  const splayDeg = 5;
  const rows = Array.from({ length: 4 }, (_, i) => ({
    box_type_key: demoBox.key,
    splay_deg: splayDeg * Math.max(0, i - 1),
  }));

  return {
    rows,
    label: "Demo Config (4 boxes)",
    isDemo: true,
  };
}

function ensureDefaultConfiguration() {
  // Ensure the editor always starts with a basic configuration
  const db = currentData?.database;
  if (!db) return;

  if (activeConfig?.elements?.length) {
    scheduleArrayViewerUpdate();
    return;
  }

  let config = buildConnectorDefaultConfig(db);
  let hint = config
    ? `Default configuration for frame "${config.frameKey || "?"}".`
    : "";
  if (!config) {
    config = buildFallbackDefaultConfig(db);
    hint = "Demo configuration created (no setup data found in file).";
  }

  if (!config) return;

  const tbody = document.getElementById("config-editor-body");
  if (tbody) {
    tbody.innerHTML = "";
    config.rows.forEach((row) =>
      addConfigEditorRow(row.box_type_key, row.splay_deg),
    );
  }

  const elements = buildArrayElementsFromRows(config.rows);
  if (!elements.length || buildElementsFromConfig({ elements }).length === 0) {
    activeConfig = null;
    updateConfigEditorHint(
      "Demo configuration could not be generated from available sources.",
    );
    scheduleArrayViewerUpdate();
    return;
  }

  activeConfig = {
    elements,
    label: config.label,
    isDemo: config.isDemo,
  };

  updateConfigEditorHint(hint);
  scheduleArrayViewerUpdate();
  markConfigDirty();
}

function updateConfigEditorHint(message) {
  // Show or hide the hint under the editor
  const hint = document.getElementById("config-editor-hint");
  if (!hint) return;
  if (message) {
    hint.textContent = message;
    hint.classList.remove("hidden");
  } else {
    hint.textContent = "";
    hint.classList.add("hidden");
  }
}

function buildElementsFromConfig(config) {
  // Expand a config into concrete source elements
  if (!config?.elements?.length) return [];

  const boxTypes = currentData?.database?.box_types || [];
  const allElements = [];

  for (const elem of config.elements) {
    const boxType = boxTypes.find((bt) => bt.key === elem.box_type_key);
    if (!boxType) continue;

    const placements = boxType.source_placements || [];
    if (placements.length > 0) {
      for (const placement of placements) {
        const sourceKey = placement?.source_def_key;
        if (!sourceKey) continue;
        const pPos = placement.position || {};
        const pAngles = placement.angles || {};
        const elemAngles = {
          x: Number(elem.angles?.x) || 0,
          y: Number(elem.angles?.y) || 0,
          z: Number(elem.angles?.z) || 0,
        };
        const placementAngles = {
          x: toRadiansMaybe(pAngles.x),
          y: toRadiansMaybe(pAngles.y),
          z: toRadiansMaybe(pAngles.z),
        };
        const rotatedPlacement = rotateVectorFromAngles(elemAngles, pPos);
        const orientationMatrix = composeRotationMatrices(
          buildRotationMatrix(elemAngles),
          buildRotationMatrix(placementAngles),
        );
        const filterGroupKeys = getFilterGroupKeysForSource(boxType, sourceKey);

        // Combine box-level config position with rotated source placement position.
        allElements.push({
          source_key: sourceKey,
          position: {
            x:
              (Number(elem.position?.x) || 0) / 1000 +
              (Number(rotatedPlacement.x) || 0) / 1000,
            y:
              (Number(elem.position?.y) || 0) / 1000 +
              (Number(rotatedPlacement.y) || 0) / 1000,
            z:
              (Number(elem.position?.z) || 0) / 1000 +
              (Number(rotatedPlacement.z) || 0) / 1000,
          },
          angles: {
            x: elemAngles.x + placementAngles.x,
            y: elemAngles.y + placementAngles.y,
            z: elemAngles.z + placementAngles.z,
          },
          orientation_matrix: orientationMatrix,
          filter_group_keys: filterGroupKeys,
          gain: elem.gain || 0,
        });
      }
    } else {
      // Fallback: use box sources directly
      const sources = boxType.sources || [];
      for (const key of sources) {
        allElements.push({
          source_key: key,
          position: {
            x: (Number(elem.position?.x) || 0) / 1000,
            y: (Number(elem.position?.y) || 0) / 1000,
            z: (Number(elem.position?.z) || 0) / 1000,
          },
          angles: {
            x: Number(elem.angles?.x) || 0,
            y: Number(elem.angles?.y) || 0,
            z: Number(elem.angles?.z) || 0,
          },
          orientation_matrix: buildRotationMatrix(elem.angles),
          filter_group_keys: getFilterGroupKeysForSource(boxType, key),
          gain: elem.gain || 0,
        });
      }
    }
  }

  return allElements;
}

function scheduleArrayViewerUpdate() {
  // Debounce array viewer redraws
  if (arrayViewerNeedsUpdate) return;
  arrayViewerNeedsUpdate = true;
  requestAnimationFrame(() => {
    arrayViewerNeedsUpdate = false;
    updateArrayViewer();
  });
}

function toViewPoint(point) {
  // Convert GLL Z-up to Three.js Y-up
  if (!point) return null;
  const x = Number(point.x);
  const y = Number(point.y);
  const z = Number(point.z);
  if (![x, y, z].every(Number.isFinite)) return null;
  return { x, y: z, z: y };
}

function buildArrayElementsFromRows(rows) {
  // Expand splay-only rows into positioned elements
  if (!rows?.length) return [];
  const boxTypes = currentData?.database?.box_types || [];
  const elements = [];
  const position = { x: 0, y: 0, z: 0 };
  let cumulativeSplayDeg = 0;

  rows.forEach((row, index) => {
    const boxType = boxTypes.find((bt) => bt.key === row.box_type_key);
    if (!boxType) return;
    const splayDeg = Number(row.splay_deg) || 0;
    if (index > 0) {
      cumulativeSplayDeg += splayDeg;
    }
    const angles = { x: 0, y: toRad(-cumulativeSplayDeg), z: 0 };
    elements.push({
      box_type_key: boxType.key,
      position: { x: position.x, y: position.y, z: position.z },
      angles,
      gain: 0,
    });

    const pivot = boxType?.next_pivot;
    let step = null;
    if (
      pivot &&
      [pivot.x, pivot.y, pivot.z].every((value) => Number.isFinite(value))
    ) {
      step = { x: pivot.x, y: pivot.y, z: pivot.z };
    } else {
      const height = estimateBoxHeightMm(boxType);
      step = { x: 0, y: 0, z: height };
    }

    const rotatedStep = rotateVectorFromAngles(angles, step);
    position.x += rotatedStep.x;
    position.y -= rotatedStep.y;
    position.z += rotatedStep.z;
  });

  return elements;
}

function getFilterGroupKeysForSource(boxType, sourceKey) {
  if (!boxType?.input_config?.inputs?.length || !sourceKey) {
    return [];
  }

  const keys = [];
  for (const input of boxType.input_config.inputs) {
    const links = input?.source_links || [];
    for (const link of links) {
      if (link?.source_key === sourceKey && link?.filter_grp_key) {
        keys.push(link.filter_grp_key);
      }
    }
  }

  return [...new Set(keys)];
}

function rotateVectorFromAngles(angles, vector) {
  // Rotate vector by GLL Euler angles (H,V,R)
  const matrix = buildRotationMatrix(angles);

  const x = Number(vector?.x) || 0;
  const y = Number(vector?.y) || 0;
  const z = Number(vector?.z) || 0;

  return {
    x: matrix[0] * x + matrix[1] * y + matrix[2] * z,
    y: matrix[3] * x + matrix[4] * y + matrix[5] * z,
    z: matrix[6] * x + matrix[7] * y + matrix[8] * z,
  };
}

function buildRotationMatrix(angles) {
  const h = Number(angles?.x) || 0;
  const v = Number(angles?.y) || 0;
  const r = Number(angles?.z) || 0;
  const sh = Math.sin(h);
  const ch = Math.cos(h);
  const sv = Math.sin(v);
  const cv = Math.cos(v);
  const sr = Math.sin(r);
  const cr = Math.cos(r);

  return [
    ch * cr - sv * sh * sr,
    sh * cr + sv * ch * sr,
    cv * sr,
    -cv * sh,
    cv * ch,
    -sv,
    -ch * sr - sv * sh * cr,
    -sh * sr + sv * ch * cr,
    cv * cr,
  ];
}

function composeRotationMatrices(left, right) {
  if (!Array.isArray(left) || left.length !== 9) return right;
  if (!Array.isArray(right) || right.length !== 9) return left;

  return [
    left[0] * right[0] + left[1] * right[3] + left[2] * right[6],
    left[0] * right[1] + left[1] * right[4] + left[2] * right[7],
    left[0] * right[2] + left[1] * right[5] + left[2] * right[8],
    left[3] * right[0] + left[4] * right[3] + left[5] * right[6],
    left[3] * right[1] + left[4] * right[4] + left[5] * right[7],
    left[3] * right[2] + left[4] * right[5] + left[5] * right[8],
    left[6] * right[0] + left[7] * right[3] + left[8] * right[6],
    left[6] * right[1] + left[7] * right[4] + left[8] * right[7],
    left[6] * right[2] + left[7] * right[5] + left[8] * right[8],
  ];
}

function toViewQuaternion(angles) {
  // Convert GLL Euler angles to view quaternion (forward vector only)
  if (!angles || typeof THREE === "undefined") return null;
  const matrix = buildRotationMatrix(angles);
  const rotMatrix = new THREE.Matrix4().set(
    matrix[0], matrix[1], matrix[2], 0,
    matrix[3], matrix[4], matrix[5], 0,
    matrix[6], matrix[7], matrix[8], 0,
    0, 0, 0, 1,
  );

  const forward = new THREE.Vector3(0, 1, 0);
  forward.applyMatrix4(rotMatrix);
  const forwardView = new THREE.Vector3(forward.x, forward.z, forward.y);
  if (forwardView.lengthSq() === 0) return null;
  forwardView.normalize();
  const quat = new THREE.Quaternion();
  quat.setFromUnitVectors(new THREE.Vector3(0, 1, 0), forwardView);
  return quat;
}

function toViewQuaternionNoSwap(angles) {
  // Convert GLL Euler angles without Y/Z swap (for already-swapped geometry)
  if (!angles || typeof THREE === "undefined") return null;
  const matrix = buildRotationMatrix(angles);
  const rotMatrix = new THREE.Matrix4().set(
    matrix[0], matrix[1], matrix[2], 0,
    matrix[3], matrix[4], matrix[5], 0,
    matrix[6], matrix[7], matrix[8], 0,
    0, 0, 0, 1,
  );

  const forward = new THREE.Vector3(0, 1, 0);
  forward.applyMatrix4(rotMatrix);
  if (forward.lengthSq() === 0) return null;
  forward.normalize();
  const quat = new THREE.Quaternion();
  quat.setFromUnitVectors(new THREE.Vector3(0, 1, 0), forward);
  return quat;
}

function readThemeVar(name, fallback) {
  // Read CSS variable with fallback
  if (typeof window === "undefined" || !document?.documentElement) {
    return fallback;
  }
  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
  return value || fallback;
}

function readArrayThemeColors() {
  // Collect theme-specific colors for array viewer
  return {
    gridMajor: readThemeVar("--geom-grid-major", "#94a3b8"),
    gridMinor: readThemeVar("--geom-grid-minor", "#e2e8f0"),
    gridOpacity: parseFloat(readThemeVar("--geom-grid-opacity", "0.45")),
  };
}

function applyGridTheme(grid, colors) {
  // Apply CSS-driven grid styling
  if (!grid || !colors) return;
  const materials = Array.isArray(grid.material)
    ? grid.material
    : [grid.material];
  materials.forEach((material, index) => {
    if (!material) return;
    material.transparent = true;
    material.opacity = Number.isFinite(colors.gridOpacity)
      ? colors.gridOpacity
      : 0.45;
    const color = index === 0 ? colors.gridMajor : colors.gridMinor;
    if (color && material.color?.setStyle) {
      material.color.setStyle(color);
    } else if (color && material.color?.set) {
      material.color.set(color);
    }
    material.needsUpdate = true;
  });
}

function computeGeometryBounds(geometry, transform = null) {
  // Compute geometry bounds in local or transformed space
  const vertices = geometry?.vertices || [];
  if (!geometry || vertices.length === 0) return null;

  let minX = Infinity;
  let minY = Infinity;
  let minZ = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  let maxZ = -Infinity;

  for (let i = 1; i <= vertices.length; i += 1) {
    const vertex = resolveGeometryVertex(geometry, vertices, i, 1);
    if (!vertex) continue;
    const point = transform ? transform(vertex) : vertex;
    if (!point) continue;
    minX = Math.min(minX, point.x);
    minY = Math.min(minY, point.y);
    minZ = Math.min(minZ, point.z);
    maxX = Math.max(maxX, point.x);
    maxY = Math.max(maxY, point.y);
    maxZ = Math.max(maxZ, point.z);
  }

  if (!Number.isFinite(minX) || !Number.isFinite(maxX)) return null;
  return { minX, minY, minZ, maxX, maxY, maxZ };
}

function estimateBoxHeightMm(boxType) {
  // Estimate vertical height for spacing (Z axis)
  const bounds = computeGeometryBounds(boxType?.case_geometry);
  if (bounds) {
    return Math.max(bounds.maxZ - bounds.minZ, 1);
  }
  const pivot = boxType?.next_pivot;
  if (pivot && Number.isFinite(pivot.z) && Math.abs(pivot.z) > 0) {
    return Math.abs(pivot.z);
  }
  return 500;
}

function estimateBoxSizeMm(boxType) {
  // Estimate box size for fallback rendering
  const bounds = computeGeometryBounds(boxType?.case_geometry);
  if (bounds) {
    return {
      x: Math.max(bounds.maxX - bounds.minX, 1),
      y: Math.max(bounds.maxY - bounds.minY, 1),
      z: Math.max(bounds.maxZ - bounds.minZ, 1),
    };
  }
  const height = estimateBoxHeightMm(boxType);
  return {
    x: height * 0.6,
    y: height * 0.4,
    z: height,
  };
}

function buildBoxMeshesFromGeometry(geometry, color) {
  // Build mesh/line objects from case geometry
  const vertices = geometry?.vertices || [];
  const faces = geometry?.faces || [];
  const edges = geometry?.edges || [];
  let bounds = null;

  const updateBounds = (point) => {
    if (!point) return;
    if (!bounds) {
      bounds = {
        minX: point.x,
        minY: point.y,
        minZ: point.z,
        maxX: point.x,
        maxY: point.y,
        maxZ: point.z,
      };
      return;
    }
    bounds.minX = Math.min(bounds.minX, point.x);
    bounds.minY = Math.min(bounds.minY, point.y);
    bounds.minZ = Math.min(bounds.minZ, point.z);
    bounds.maxX = Math.max(bounds.maxX, point.x);
    bounds.maxY = Math.max(bounds.maxY, point.y);
    bounds.maxZ = Math.max(bounds.maxZ, point.z);
  };

  let mesh = null;
  if (faces.length > 0) {
    const positions = [];
    for (const face of faces) {
      const indices = Array.isArray(face.vertices) ? face.vertices : [];
      if (indices.length < 3) continue;
      const triangles = triangulateFace(indices);
      for (const tri of triangles) {
        for (const idx of tri) {
          const vertex = resolveGeometryVertex(geometry, vertices, idx, 1);
          const point = vertex ? toViewPoint(vertex) : null;
          if (!point) continue;
          positions.push(point.x, point.y, point.z);
          updateBounds(point);
        }
      }
    }
    if (positions.length > 0) {
      const buffer = new THREE.BufferGeometry();
      buffer.setAttribute(
        "position",
        new THREE.Float32BufferAttribute(positions, 3),
      );
      buffer.computeBoundingBox();
      buffer.computeBoundingSphere();
      buffer.computeVertexNormals();
      const material = new THREE.MeshStandardMaterial({
        color,
        transparent: true,
        opacity: 0.85,
        roughness: 0.5,
        metalness: 0.08,
        side: THREE.DoubleSide,
      });
      mesh = new THREE.Mesh(buffer, material);
    }
  }

  let lines = null;
  if (edges.length > 0) {
    const positions = [];
    for (const edge of edges) {
      const v1 = resolveGeometryVertex(geometry, vertices, edge.v1, 1);
      const v2 = resolveGeometryVertex(geometry, vertices, edge.v2, 1);
      const p1 = v1 ? toViewPoint(v1) : null;
      const p2 = v2 ? toViewPoint(v2) : null;
      if (!p1 || !p2) continue;
      positions.push(p1.x, p1.y, p1.z, p2.x, p2.y, p2.z);
      updateBounds(p1);
      updateBounds(p2);
    }
    if (positions.length > 0) {
      const buffer = new THREE.BufferGeometry();
      buffer.setAttribute(
        "position",
        new THREE.Float32BufferAttribute(positions, 3),
      );
      buffer.computeBoundingBox();
      buffer.computeBoundingSphere();
      const material = new THREE.LineBasicMaterial({
        color,
        transparent: true,
        opacity: 0.7,
      });
      lines = new THREE.LineSegments(buffer, material);
    }
  }

  return { mesh, lines, bounds };
}

function initArrayViewer() {
  // Initialize array viewer scene if needed
  if (arrayViewer || typeof THREE === "undefined") return;
  const container = document.getElementById("array-view-canvas");
  if (!container) return;

  const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  renderer.setSize(container.clientWidth || 1, container.clientHeight || 1);
  renderer.setClearColor(0x000000, 0);
  container.appendChild(renderer.domElement);

  const scene = new THREE.Scene();
  const camera = new THREE.PerspectiveCamera(
    45,
    (container.clientWidth || 1) / (container.clientHeight || 1),
    0.01,
    2000,
  );
  camera.position.set(0, 0.4, 2.2);
  camera.lookAt(0, 0, 0);

  const controls = new OrbitControls(camera, renderer.domElement);
  controls.enableDamping = true;
  controls.dampingFactor = 0.08;
  controls.screenSpacePanning = true;
  controls.enablePan = true;
  controls.enableZoom = true;
  controls.rotateSpeed = 0.6;
  controls.panSpeed = 0.9;
  controls.autoRotate = true;
  controls.minDistance = 0.2;
  controls.maxDistance = 25;
  controls.mouseButtons = {
    LEFT: THREE.MOUSE.ROTATE,
    MIDDLE: THREE.MOUSE.DOLLY,
    RIGHT: THREE.MOUSE.PAN,
  };

  const ambient = new THREE.AmbientLight(0xffffff, 0.7);
  const keyLight = new THREE.DirectionalLight(0xffffff, 0.85);
  keyLight.position.set(2.5, 2.5, 2);
  scene.add(ambient, keyLight);

  const grid = new THREE.GridHelper(2, 12, 0x94a3b8, 0xe2e8f0);
  applyGridTheme(grid, readArrayThemeColors());
  const axes = new THREE.AxesHelper(0.8);
  axes.material.transparent = true;
  axes.material.opacity = 0.5;
  scene.add(grid, axes);

  const group = new THREE.Group();
  scene.add(group);

  const viewer = {
    renderer,
    scene,
    camera,
    controls,
    group,
    grid,
    frameId: null,
    container,
  };

  const animate = () => {
    viewer.frameId = requestAnimationFrame(animate);
    viewer.controls.update();
    viewer.renderer.render(viewer.scene, viewer.camera);
  };
  animate();

  arrayViewer = viewer;

  const autoToggle = document.getElementById("array-view-autorotate");
  if (autoToggle) {
    autoToggle.addEventListener("change", (e) => {
      if (!arrayViewer?.controls) return;
      arrayViewer.controls.autoRotate = !!e.target.checked;
    });
  }
  const facesToggle = document.getElementById("array-view-faces");
  if (facesToggle) {
    facesToggle.addEventListener("change", () => scheduleArrayViewerUpdate());
  }
  const edgesToggle = document.getElementById("array-view-edges");
  if (edgesToggle) {
    edgesToggle.addEventListener("change", () => scheduleArrayViewerUpdate());
  }
}

function resetArrayViewer() {
  // Destroy array viewer instance
  if (!arrayViewer) return;
  if (arrayViewer.frameId) cancelAnimationFrame(arrayViewer.frameId);
  if (arrayViewer.controls?.dispose) arrayViewer.controls.dispose();
  if (arrayViewer.renderer) {
    arrayViewer.renderer.dispose();
    if (arrayViewer.renderer.domElement?.parentNode) {
      arrayViewer.renderer.domElement.parentNode.removeChild(
        arrayViewer.renderer.domElement,
      );
    }
  }
  arrayViewer = null;
}

function handleArrayViewerResize() {
  // Resize array viewer renderer
  if (!arrayViewer) return;
  const width = arrayViewer.container.clientWidth || 1;
  const height = arrayViewer.container.clientHeight || 1;
  arrayViewer.renderer.setSize(width, height);
  arrayViewer.camera.aspect = width / height;
  arrayViewer.camera.updateProjectionMatrix();
  arrayViewer.controls.update();
}

function updateArrayViewer() {
  // Render array configuration in 3D
  const container = document.getElementById("array-view-canvas");
  const placeholder = document.getElementById("array-view-placeholder");
  if (!container) return;

  const rows = readConfigEditorRows();
  const elements = buildArrayElementsFromRows(rows);
  const disposeGroup = (group) => {
    if (!group) return;
    group.traverse((obj) => {
      if (obj.geometry?.dispose) obj.geometry.dispose();
      if (obj.material) {
        if (Array.isArray(obj.material)) {
          obj.material.forEach((mat) => mat?.dispose?.());
        } else if (obj.material?.dispose) {
          obj.material.dispose();
        }
      }
    });
    group.clear();
  };

  if (!elements.length) {
    if (placeholder) placeholder.style.display = "flex";
    disposeGroup(arrayViewer?.group);
    return;
  }

  initArrayViewer();
  if (!arrayViewer) return;

  if (placeholder) placeholder.style.display = "none";

  const showFaces =
    document.getElementById("array-view-faces")?.checked ?? true;
  const showEdges =
    document.getElementById("array-view-edges")?.checked ?? true;

  disposeGroup(arrayViewer.group);

  const boxTypes = currentData?.database?.box_types || [];
  elements.forEach((elem, index) => {
    const boxType = boxTypes.find((bt) => bt.key === elem.box_type_key);
    const color = new THREE.Color().setHSL((index * 0.16) % 1, 0.55, 0.55);

    const boxGroup = new THREE.Group();
    const geometry = boxType?.case_geometry;

    if (geometry && (geometry.faces?.length || geometry.edges?.length)) {
      const { mesh, lines } = buildBoxMeshesFromGeometry(geometry, color);
      if (mesh && showFaces) boxGroup.add(mesh);
      if (lines && showEdges) boxGroup.add(lines);
    } else {
      const size = estimateBoxSizeMm(boxType);
      const viewSize = {
        x: size.x,
        y: size.z,
        z: size.y,
      };
      const boxGeom = new THREE.BoxGeometry(viewSize.x, viewSize.y, viewSize.z);
      const material = new THREE.MeshStandardMaterial({
        color,
        transparent: true,
        opacity: 0.6,
        roughness: 0.5,
        metalness: 0.08,
      });
      if (showFaces) {
        boxGroup.add(new THREE.Mesh(boxGeom, material));
      }
      if (showEdges) {
        const edges = new THREE.EdgesGeometry(boxGeom);
        const lineMat = new THREE.LineBasicMaterial({
          color: color.clone().multiplyScalar(0.6),
          transparent: true,
          opacity: 0.6,
        });
        boxGroup.add(new THREE.LineSegments(edges, lineMat));
      }
    }

    const viewPos = toViewPoint(elem.position);
    if (viewPos) {
      boxGroup.position.set(viewPos.x, viewPos.y, viewPos.z);
    }
    const quat = toViewQuaternionNoSwap(elem.angles);
    if (quat) {
      boxGroup.setRotationFromQuaternion(quat);
    }

    arrayViewer.group.add(boxGroup);
  });

  arrayViewer.group.updateMatrixWorld(true);
  const bounds = new THREE.Box3().setFromObject(arrayViewer.group);
  if (!bounds.isEmpty()) {
    const size = new THREE.Vector3();
    const center = new THREE.Vector3();
    bounds.getSize(size);
    bounds.getCenter(center);
    const maxDim = Math.max(size.x, size.y, size.z);
    const targetSize = 1.6;
    const scale = maxDim > 0 ? targetSize / maxDim : 1;
    arrayViewer.group.scale.setScalar(scale);
    arrayViewer.group.position.set(
      -center.x * scale,
      -center.y * scale,
      -center.z * scale,
    );

    const cameraDistance = Math.max(targetSize * 1.4, 1.2);
    arrayViewer.camera.position.set(0, targetSize * 0.5, cameraDistance);
  }

  arrayViewer.controls.target.set(0, 0, 0);
  arrayViewer.controls.update();
  applyGridTheme(arrayViewer.grid, readArrayThemeColors());
}

function wireFilterGroupResponses() {
  // Bind filter group chart controls
  const groups = currentData?.database?.filter_groups || [];
  groups.forEach((group, groupIndex) => {
    const elements = getFilterGroupResponseElements(groupIndex);
    if (!elements.filterSelect) {
      return;
    }

    const hasFilters = !!group?.filters?.length;
    elements.filterSelect.disabled = !hasFilters;
    elements.phaseSelect.disabled = !hasFilters;

    elements.filterSelect.addEventListener("change", () =>
      renderFilterGroupResponse(groupIndex),
    );
    elements.phaseSelect.addEventListener("change", () =>
      renderFilterGroupResponse(groupIndex),
    );
  });
}

function wireFilterExportDropdowns() {
  // Attach export dropdown behavior for filter groups
  document.querySelectorAll(".btn-filter-export").forEach((button) => {
    if (button.dataset.bound === "true") return;
    button.dataset.bound = "true";
    button.addEventListener("click", (e) => {
      e.stopPropagation();
      const container = button.closest(".dropdown-container");
      const wasOpen = container.classList.contains("show");
      document.querySelectorAll(".dropdown-container.show").forEach((other) => {
        other.classList.remove("show");
      });
      if (!wasOpen) container.classList.add("show");
    });
  });

  document
    .querySelectorAll(".filter-export-dropdown .dropdown-item")
    .forEach((item) => {
      if (item.dataset.bound === "true") return;
      item.dataset.bound = "true";
      item.addEventListener("click", (e) => {
        e.stopPropagation();
        const format = item.dataset.format;
        const button = item
          .closest(".dropdown-container")
          .querySelector(".btn-filter-export");
        const groupIdx = Number(button.dataset.groupIdx);
        const filterSelect = document.getElementById(
          `filter-group-response-filter-${groupIdx}`,
        );
        const filterIdx = filterSelect ? parseInt(filterSelect.value) : 0;

        const groups = currentData?.database?.filter_groups || [];
        const group = groups[groupIdx];
        if (!group) return;

        const filterDef = group.filters?.[filterIdx];
        const gllName = sanitizeFilename(
          currentData?.gen_system?.model ||
            currentData?.gen_system?.manufacturer ||
            "gll",
        );
        const groupLabel = sanitizeFilename(
          group.label || group.key || `group-${groupIdx}`,
        );
        const filterLabel = sanitizeFilename(
          filterDef?.label || filterDef?.key || `filter-${filterIdx}`,
        );
        const basename = `${gllName}_${groupLabel}_${filterLabel}`;

        if (format === "xgfb") {
          if (!filterDef) return;
          const content = buildXgfbContent(filterDef);
          if (content) downloadTextFile(`${basename}.xgfb`, content);
        } else {
          const response = computeCombinedFilterResponse(groupIdx, filterIdx);
          if (response.error || !response.frequencies?.length) return;
          if (format === "frd") {
            const content = buildFrdContent(
              response.frequencies,
              response.level,
              response.phase,
            );
            downloadTextFile(`${basename}.frd`, content);
          } else if (format === "csv") {
            const content = buildCsvFilterContent(
              response.frequencies,
              response.level,
              response.phase,
            );
            downloadTextFile(`${basename}.csv`, content);
          }
        }

        item.closest(".dropdown-container").classList.remove("show");
      });
    });
}

function getFilterGroupResponseElements(groupIndex) {
  // Resolve DOM nodes for filter group charts
  return {
    filterSelect: document.getElementById(
      `filter-group-response-filter-${groupIndex}`,
    ),
    phaseSelect: document.getElementById(
      `filter-group-response-phase-${groupIndex}`,
    ),
    placeholder: document.getElementById(
      `filter-group-response-placeholder-${groupIndex}`,
    ),
    chartCanvas: document.getElementById(
      `filter-group-response-chart-${groupIndex}`,
    ),
    meta: document.getElementById(`filter-group-response-meta-${groupIndex}`),
  };
}

function renderFilterGroupResponse(groupIndex) {
  // Compute and render the filter group response chart
  const data = currentData;
  const bytes = currentFileBytes;
  const groups = data?.database?.filter_groups || [];
  const group = groups[groupIndex];

  const elements = getFilterGroupResponseElements(groupIndex);
  if (!elements.filterSelect || !elements.chartCanvas) {
    return;
  }

  const filterIndex = parseInt(elements.filterSelect.value);
  if (!group || Number.isNaN(filterIndex)) {
    // Invalid selection or missing group
    updateFilterGroupResponseMeta(
      groupIndex,
      "No filter response data available",
    );
    setFilterGroupResponsePlaceholder(
      groupIndex,
      "No filter response data available",
    );
    destroyFilterGroupChart(groupIndex);
    return;
  }

  if (!bytes || typeof computeFilterResponse !== "function") {
    // Missing WASM helper for filter response
    updateFilterGroupResponseMeta(
      groupIndex,
      "Filter response helper not available",
    );
    setFilterGroupResponsePlaceholder(
      groupIndex,
      "Filter response helper not available",
    );
    destroyFilterGroupChart(groupIndex);
    return;
  }

  const payload = JSON.stringify({
    group_index: groupIndex,
    filter_index: filterIndex,
  });
  let response;
  try {
    // Delegate response computation to WASM
    const responseJSON = computeFilterResponse(bytes, payload);
    response = JSON.parse(responseJSON);
  } catch (err) {
    updateFilterGroupResponseMeta(
      groupIndex,
      "Failed to compute filter response",
    );
    setFilterGroupResponsePlaceholder(
      groupIndex,
      "Failed to compute filter response",
    );
    destroyFilterGroupChart(groupIndex);
    return;
  }

  if (!response.success) {
    // Surface computation errors
    const message = response.error || "Failed to compute filter response";
    updateFilterGroupResponseMeta(groupIndex, message);
    setFilterGroupResponsePlaceholder(groupIndex, message);
    destroyFilterGroupChart(groupIndex);
    return;
  }

  if (!response.frequencies?.length || !response.level?.length) {
    // Response missing usable data
    const message = response.message || "No filter response data available";
    updateFilterGroupResponseMeta(
      groupIndex,
      message,
      buildFilterGroupMetaChips(group, group?.filters?.[filterIndex], response),
    );
    setFilterGroupResponsePlaceholder(groupIndex, message);
    destroyFilterGroupChart(groupIndex);
    return;
  }

  if (elements.phaseSelect) {
    // Disable phase selector when no phase data exists
    const hasPhase = Array.isArray(response.phase) && response.phase.length > 0;
    elements.phaseSelect.disabled = !hasPhase;
    if (!hasPhase) {
      elements.phaseSelect.value = "unwrapped";
    }
  }

  const frequencyData = buildFrequencyPoints(
    response.frequencies,
    response.level,
  );
  if (!frequencyData) {
    // Abort if points cannot be built
    updateFilterGroupResponseMeta(
      groupIndex,
      "No filter response data available",
    );
    setFilterGroupResponsePlaceholder(
      groupIndex,
      "No filter response data available",
    );
    destroyFilterGroupChart(groupIndex);
    return;
  }

  setFilterGroupResponsePlaceholder(groupIndex, "");
  // Compose chart datasets
  const datasets = [
    {
      label: "Level (dB)",
      data: frequencyData.points,
      borderColor: "#0ea5e9",
      backgroundColor: "rgba(14, 165, 233, 0.12)",
      fill: true,
      tension: 0.3,
      pointRadius: 0,
      yAxisID: "y",
    },
  ];

  let phaseAxisTitle = "Phase (rad)";
  if (response.phase && response.phase.length === response.level.length) {
    // Add phase series when available
    const phaseMode = elements.phaseSelect?.value || "unwrapped";
    const phaseSeries = getPhaseSeries(
      phaseMode,
      response.frequencies,
      response.phase,
      unwrapPhase(response.phase),
    );
    phaseAxisTitle = phaseSeries.axisTitle;
    const phaseData = buildFrequencyPoints(
      response.frequencies,
      phaseSeries.values,
    );
    if (phaseData) {
      datasets.push({
        label: phaseSeries.label,
        data: phaseData.points,
        borderColor: "#f97316",
        backgroundColor: "transparent",
        tension: 0.3,
        pointRadius: 0,
        yAxisID: "y1",
      });
    }
  }

  const gridColor = getDarkModeGridColor();
  const xScale = buildLogFrequencyScale(
    frequencyData.minFrequency,
    frequencyData.maxFrequency,
    "Frequency",
  );
  if (gridColor) {
    xScale.grid = { ...(xScale.grid || {}), color: gridColor };
  }

  const scales = {
    x: xScale,
    y: {
      type: "linear",
      display: true,
      position: "left",
      title: {
        display: true,
        text: "Level (dB)",
      },
      grid: gridColor ? { color: gridColor } : undefined,
    },
  };

  if (response.phase) {
    scales.y1 = {
      type: "linear",
      display: true,
      position: "right",
      title: {
        display: true,
        text: phaseAxisTitle,
      },
      grid: {
        drawOnChartArea: false,
      },
    };
  }

  const ctx = elements.chartCanvas.getContext("2d");
  destroyFilterGroupChart(groupIndex);
  // Create chart instance
  const localChart = new Chart(ctx, {
    type: "line",
    data: {
      datasets,
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: filterGroupChartInitialized.has(groupIndex)
        ? false
        : { duration: 700 },
      interaction: {
        mode: "index",
        intersect: false,
      },
      scales,
      plugins: {
        legend: {
          position: "top",
        },
        tooltip: {
          callbacks: {
            title: (items) => {
              const value = items?.[0]?.parsed?.x;
              return value ? formatFrequency(value) : "";
            },
          },
        },
        zoom: buildZoomPluginOptions(
          frequencyData.minFrequency,
          frequencyData.maxFrequency,
        ),
      },
    },
  });

  filterGroupResponseCharts.set(groupIndex, localChart);
  filterGroupChartInitialized.add(groupIndex);
  updateFilterGroupResponseMeta(
    groupIndex,
    response.message,
    buildFilterGroupMetaChips(group, group?.filters?.[filterIndex], response),
  );
}

function destroyFilterGroupChart(groupIndex) {
  // Tear down any existing chart for this group
  const existingChart = filterGroupResponseCharts.get(groupIndex);
  if (existingChart) {
    existingChart.destroy();
    filterGroupResponseCharts.delete(groupIndex);
  }
}

function buildFilterGroupMetaChips(group, filterDef, response) {
  // Build metadata chips describing response details
  const chips = [];
  if (group) {
    chips.push(
      `<span class="chip">${escapeHtml(group.label || group.key || "Filter Group")}</span>`,
    );
  }
  if (filterDef) {
    chips.push(
      `<span class="chip">${escapeHtml(filterDef.label || filterDef.key || "Filter")}</span>`,
    );
  }
  if (Number.isFinite(response.used_filters)) {
    const kind = response.filter_kind || "LogSpectrum";
    chips.push(
      `<span class="chip">${escapeHtml(kind)} ${response.used_filters}</span>`,
    );
  }
  if (Number.isFinite(response.sample_rate)) {
    chips.push(
      `<span class="chip">SR ${response.sample_rate.toFixed(0)} Hz</span>`,
    );
  }
  if (Number.isFinite(response.point_count)) {
    chips.push(`<span class="chip">${response.point_count} points</span>`);
  }
  if (response.is_complex) {
    chips.push('<span class="chip">Complex</span>');
  }
  if (
    Number.isFinite(response.skipped_filters) &&
    response.skipped_filters > 0
  ) {
    chips.push(`<span class="chip">Skipped ${response.skipped_filters}</span>`);
  }
  if (
    Number.isFinite(response.mismatched_filters) &&
    response.mismatched_filters > 0
  ) {
    chips.push(
      `<span class="chip">Mismatched ${response.mismatched_filters}</span>`,
    );
  }
  if (response.bypassed) {
    chips.push('<span class="chip">Bypassed</span>');
  }
  return chips;
}

function updateFilterGroupResponseMeta(groupIndex, message, chips = null) {
  // Update metadata chip list beneath the chart
  const meta = document.getElementById(
    `filter-group-response-meta-${groupIndex}`,
  );
  if (!meta) return;

  const parts = Array.isArray(chips) && chips.length > 0 ? chips.slice() : [];
  if (message) {
    parts.push(`<span class="chip">${escapeHtml(message)}</span>`);
  }
  if (parts.length === 0) {
    meta.innerHTML =
      '<span class="chip">No filter response data available</span>';
    return;
  }
  meta.innerHTML = parts.join("");
}

function setFilterGroupResponsePlaceholder(groupIndex, message) {
  // Show or hide the chart placeholder
  const placeholder = document.getElementById(
    `filter-group-response-placeholder-${groupIndex}`,
  );
  if (!placeholder) return;
  if (message) {
    placeholder.textContent = message;
    placeholder.style.display = "flex";
    return;
  }
  placeholder.style.display = "none";
}

function formatGeometryDetail(geometry) {
  // Render a brief geometry summary line
  if (!geometry) {
    return "";
  }

  const vertexCount = geometry.vertices?.length || 0;
  const edgeCount = geometry.edges?.length || 0;
  const faceCount = geometry.faces?.length || 0;
  const symmetry = geometry.is_symmetric
    ? `Symmetric @ X=${formatNumber(geometry.symmetry_axis, 3)}`
    : "Asymmetric";

  return `
        <div class="config-item-detail">
            Geometry: ${vertexCount} vertices • ${edgeCount} edges • ${faceCount} faces • ${symmetry}
        </div>
    `;
}

function formatBoxTypeDetail(box) {
  // Render box metadata with safe sanitization
  if (!box) {
    return "";
  }

  const sanitizeDisplayText = (value) => {
    if (value === null || value === undefined) return "";
    const raw = String(value);
    const cleaned = raw.replace(/[\x00-\x1F\x7F]/g, "").trim();
    if (!cleaned) return "";
    const maxLen = 120;
    return cleaned.length > maxLen
      ? `${cleaned.slice(0, maxLen - 3)}...`
      : cleaned;
  };

  const formatSafeList = (values) => {
    // Sanitize and join a list of labels
    if (!Array.isArray(values) || values.length === 0) return "-";
    const items = values
      .map((value) => sanitizeDisplayText(value))
      .filter((value) => value);
    if (items.length === 0) return "-";
    return items.map((value) => escapeHtml(value)).join(", ");
  };

  const keyText = sanitizeDisplayText(box.key);
  const key = keyText ? escapeHtml(keyText) : "-";
  const sources = formatSafeList(box.sources);
  const placements =
    Array.isArray(box.source_placements) && box.source_placements.length > 0
      ? box.source_placements
          .map((placement) => {
            const label = sanitizeDisplayText(
              placement?.label || placement?.key,
            );
            const defKey = sanitizeDisplayText(placement?.source_def_key);
            if (label && defKey) {
              return `${escapeHtml(label)} (${escapeHtml(defKey)})`;
            }
            if (label) {
              return escapeHtml(label);
            }
            if (defKey) {
              return escapeHtml(defKey);
            }
            return "-";
          })
          .join(", ")
      : "-";
  const weightValue = formatNumber(box.weight, 2);
  const weight = weightValue === "-" ? "-" : `${weightValue} kg`;
  const vAngleValue = formatNumber(box.vertical_opening_angle, 1);
  const vAngle = vAngleValue === "-" ? "-" : `${vAngleValue}°`;
  const hAngleValue = formatNumber(box.horizontal_opening_angle, 1);
  const hAngle = hAngleValue === "-" ? "-" : `${hAngleValue}°`;

  return `
        <div class="config-item-detail">
            Key: ${key} • Weight: ${weight} • Vertical Opening Angle: ${vAngle} • Horizontal Opening Angle: ${hAngle}
        </div>
        <div class="config-item-detail">
            Sources: ${sources} • Source Placements: ${placements}
        </div>
    `;
}

function formatGeometryActions(kind, index, geometry, label) {
  // Build export dropdown for geometry assets
  if (!hasGeometryData(geometry)) {
    return "";
  }
  const filename = sanitizeFilename(label || `${kind}-${index + 1}`);
  return `
        <div class="config-item-actions">
            <div class="dropdown-container">
                <button class="btn-download btn-geom-export" data-geom-type="${kind}" data-geom-index="${index}" data-geom-filename="${escapeHtml(filename)}">
                    Download 3D Model <span class="dropdown-icon">▼</span>
                </button>
                <div class="dropdown-menu">
                    <button class="dropdown-item" data-format="xed">${escapeHtml(filename)}.xed</button>
                    <button class="dropdown-item" data-format="stl">${escapeHtml(filename)}.stl</button>
                    <button class="dropdown-item" data-format="obj">${escapeHtml(filename)}.obj</button>
                </div>
            </div>
        </div>
    `;
}

function formatInlineGeometryViewer(kind, index, caseGeometry) {
  // Inline geometry viewer markup for a config item
  if (!hasGeometryData(caseGeometry)) {
    return "";
  }
  const id = `geometry-inline-${kind}-${index}`;
  return `
        <div class="inline-geometry-viewer" id="${id}" data-geom-kind="${kind}" data-geom-index="${index}">
            <div class="inline-geometry-toolbar">
                <div class="inline-geometry-controls">
                    <label class="geometry-toggle">
                        <input type="checkbox" class="inline-geom-faces" checked /> Faces
                    </label>
                    <label class="geometry-toggle">
                        <input type="checkbox" class="inline-geom-edges" checked /> Edges
                    </label>
                    <label class="geometry-toggle">
                        <input type="checkbox" class="inline-geom-autorotate" checked /> Auto-rotate
                    </label>
                    <label class="geometry-toggle">
                        <input type="checkbox" class="inline-geom-center-ref" checked /> Center reference
                    </label>
                    <label class="geometry-toggle">
                        <input type="checkbox" class="inline-geom-sources" /> Sources
                    </label>
                </div>
                <div class="inline-geometry-legend">
                    <label class="geometry-toggle">
                        <input type="checkbox" class="inline-geom-ref" checked />
                        <span class="geom-legend-swatch swatch-ref"></span>
                        Reference Point
                    </label>
                    <label class="geometry-toggle">
                        <input type="checkbox" class="inline-geom-com" checked />
                        <span class="geom-legend-swatch swatch-com"></span>
                        Center of Mass
                    </label>
                    <label class="geometry-toggle">
                        <input type="checkbox" class="inline-geom-pivot" />
                        <span class="geom-legend-swatch swatch-pivot"></span>
                        Next Pivot
                    </label>
                </div>
            </div>
            <div class="inline-geometry-canvas"></div>
        </div>
    `;
}

function wireGeometryDownloads() {
  // Wire dropdown toggle buttons
  document.querySelectorAll(".btn-geom-export").forEach((button) => {
    if (button.dataset.bound === "true") {
      return;
    }
    button.dataset.bound = "true";
    button.addEventListener("click", (e) => {
      e.stopPropagation();
      const container = button.closest(".dropdown-container");
      const wasOpen = container.classList.contains("show");

      // Close all other dropdowns
      document.querySelectorAll(".dropdown-container.show").forEach((other) => {
        other.classList.remove("show");
      });

      // Toggle current dropdown
      if (!wasOpen) {
        container.classList.add("show");
      }
    });
  });

  // Wire dropdown menu items
  document.querySelectorAll(".dropdown-container").forEach((container) => {
    const button = container.querySelector(".btn-geom-export");
    if (!button) return;
    container.querySelectorAll(".dropdown-item").forEach((item) => {
      if (item.dataset.bound === "true") {
        return;
      }
      item.dataset.bound = "true";
      item.addEventListener("click", (e) => {
        e.stopPropagation();
        const format = item.dataset.format;
        const kind = button.dataset.geomType;
        const index = Number(button.dataset.geomIndex);
        const datasetFilename = button.dataset.geomFilename;
        const db = currentData?.database;
        const dataItem =
          kind === "frame" ? db?.frames?.[index] : db?.box_types?.[index];
        const geometry = dataItem?.case_geometry;

        if (!hasGeometryData(geometry)) {
          return;
        }

        const label =
          dataItem?.label || dataItem?.key || `${kind}-${index + 1}`;
        const filename = datasetFilename || sanitizeFilename(label);

        // Generate and download based on format
        if (format === "xed") {
          const content = buildXedContent(geometry, {
            units: "m",
            precision: 6,
          });
          downloadTextFile(`${filename}.xed`, content);
        } else if (format === "stl") {
          const buffer = buildStlBinary(geometry, {});
          if (buffer.length > 0) {
            downloadBinaryFile(`${filename}.stl`, buffer);
          }
        } else if (format === "obj") {
          const content = buildObjContent(geometry, {});
          if (content) {
            downloadTextFile(`${filename}.obj`, content);
          }
        }

        // Close dropdown
        item.closest(".dropdown-container").classList.remove("show");
      });
    });
  });

  // Close dropdowns when clicking outside
  document.addEventListener("click", () => {
    document
      .querySelectorAll(".dropdown-container.show")
      .forEach((container) => {
        container.classList.remove("show");
      });
  });
}

function displayResources() {
  // Include Files (PDFs, documentation, technical drawings)
  const resourcesTab = document.querySelector('.tab[data-tab="resources"]');
  const resourcesContent = document.getElementById("tab-resources");
  const includeFilesList = document.getElementById("include-files-list");
  const includeFiles = currentData.database?.include_files || [];
  const dataFiles = currentData.database?.data_files || [];
  const resources = currentData.resources || [];
  const dataFileNames = new Set(
    (dataFiles || []).map((f) => cleanFilename(f.filename).toLowerCase()),
  );
  const filteredResources = resources.filter((res) => {
    // Skip resources that are already represented elsewhere
    if (res.type === "PNG" && res.name) {
      return !dataFileNames.has(cleanFilename(res.name).toLowerCase());
    }
    if (res.type === "ZLIB") {
      const label = String(res.name || "").toLowerCase();
      if (label.startsWith("pdf-") || label.startsWith("font-")) {
        return false;
      }
    }
    return true;
  });
  const hasResources =
    includeFiles.length > 0 ||
    dataFiles.length > 0 ||
    filteredResources.length > 0;
  if (resourcesTab && resourcesContent) {
    // Hide resources tab when empty
    resourcesTab.classList.toggle("hidden", !hasResources);
    resourcesContent.classList.toggle("hidden", !hasResources);
    if (!hasResources && resourcesTab.classList.contains("active")) {
      switchTab("overview");
    }
  }

  if (!hasResources) {
    // Empty-state for all resource lists
    includeFilesList.innerHTML =
      '<div class="empty-state">No documentation files found</div>';
    const dataFilesList = document.getElementById("data-files-list");
    if (dataFilesList) {
      dataFilesList.innerHTML =
        '<div class="empty-state">No data files found</div>';
    }
    return;
  }

  if (includeFiles.length === 0) {
    // Render documentation list
    includeFilesList.innerHTML =
      '<div class="empty-state">No documentation files found</div>';
  } else {
    includeFilesList.innerHTML = includeFiles
      .map((file) => {
        const isPdf = file.filename.toLowerCase().endsWith(".pdf");
        const icon = isPdf ? "📄" : "📁";
        return `
                <div class="resource-item resource-item--doc">
                    <div class="resource-meta">
                        <span class="resource-icon">${icon}</span>
                        <div class="resource-details">
                            <span class="resource-label">${escapeHtml(file.label || "Document")}</span>
                            <span class="resource-name">${escapeHtml(file.filename)}</span>
                        </div>
                    </div>
                    <div class="resource-actions">
                        <span class="resource-size">${formatBytes(file.size)}</span>
                        ${
                          file.data_uri
                            ? `
                            <button class="btn-download" onclick="downloadFile('${escapeHtml(file.filename)}', '${file.data_uri}')">
                                Download
                            </button>
                        `
                            : ""
                        }
                    </div>
                </div>
            `;
      })
      .join("");
  }

  // Data Files (images, geometry)
  const dataFilesList = document.getElementById("data-files-list");
  // Render data files list
  if (dataFiles.length === 0) {
    dataFilesList.innerHTML =
      '<div class="empty-state">No data files found</div>';
  } else {
    dataFilesList.innerHTML = dataFiles
      .map((file) => {
        const isImage = /\.(png|jpg|jpeg|gif)$/i.test(file.filename);
        const icon = isImage ? "🖼️" : "📦";
        return `
                <div class="resource-item ${isImage && file.data_uri ? "resource-item--image" : ""}">
                    <div class="resource-meta">
                        <span class="resource-icon">${icon}</span>
                        <span class="resource-name">${escapeHtml(file.filename)}</span>
                    </div>
                    <div class="resource-actions">
                        <span class="resource-size">${formatBytes(file.size)}</span>
                        ${
                          file.data_uri
                            ? `
                            <button class="btn-download" onclick="downloadFile('${escapeHtml(cleanFilename(file.filename))}', '${file.data_uri}')">
                                Download
                            </button>
                        `
                            : ""
                        }
                    </div>
                    ${
                      isImage && file.data_uri
                        ? `
                        <div class="resource-preview">
                            <img src="${file.data_uri}" alt="${escapeHtml(file.filename)}" loading="lazy" />
                        </div>
                    `
                        : ""
                    }
                </div>
            `;
      })
      .join("");
  }

  const resourcesCard = document.getElementById("other-resources-card");
  const resourcesList = document.getElementById("resources-list");
  if (resourcesCard && resourcesList) {
    // Render remaining embedded resources
    resourcesCard.classList.toggle("hidden", filteredResources.length === 0);
    if (filteredResources.length > 0) {
      resourcesList.innerHTML = filteredResources
        .map((res) => {
          const hasPreview = res.type === "PNG" && res.data_uri;
          return `
                <div class="resource-item ${hasPreview ? "resource-item--image" : ""}">
                    <div class="resource-meta">
                        <span class="resource-name">${escapeHtml(res.name || "Unnamed")}</span>
                        <div class="resource-info">
                            <span class="resource-type">${res.type}</span>
                            <span>${formatBytes(res.size)}</span>
                        </div>
                    </div>
                    ${
                      hasPreview
                        ? `
                        <div class="resource-preview">
                            <img src="${res.data_uri}" alt="${escapeHtml(res.name || "Embedded PNG")}" loading="lazy" />
                        </div>
                    `
                        : ""
                    }
                </div>
            `;
        })
        .join("");
    } else {
      resourcesList.innerHTML = "";
    }
  }
}

function cleanFilename(path) {
  // Normalize to basename for downloads
  // Remove Windows path separators and get base name
  return path.replace(/\\/g, "/").split("/").pop() || path;
}

function setupResponseControls() {
  // Populate response controls for single-response view
  const sourceSelect = document.getElementById("response-source");
  const sources = currentData.database?.source_definitions || [];

  // Filter sources that have responses
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );

  if (sourcesWithResponses.length === 0) {
    // No response data available
    sourceSelect.innerHTML =
      '<option value="">No response data available</option>';
    document.getElementById("response-index").innerHTML = "";
    document.getElementById("response-meta").innerHTML =
      '<div class="empty-state">No frequency response data loaded</div>';
    return;
  }

  sourceSelect.innerHTML = sourcesWithResponses
    .map(
      (src, i) =>
        `<option value="${i}">${escapeHtml(src.definition?.label || src.key)}</option>`,
    )
    .join("");

  const globalSelect = document.getElementById("balloon-source");
  const defaultIndex = parseInt(globalSelect?.value);
  if (
    !Number.isNaN(defaultIndex) &&
    defaultIndex < sourcesWithResponses.length
  ) {
    sourceSelect.value = String(defaultIndex);
  }

  updateResponseOptions();
}

function setupPolarControls() {
  // Initialize polar chart controls
  visualization.updatePolarOptions();
}

function setupBalloonControls() {
  // Initialize balloon sliders and options
  const rangeValue = document.getElementById("balloon-range");
  const scaleValue = document.getElementById("balloon-scale");
  if (rangeValue) {
    visualization.handleBalloonRangeInput({ target: rangeValue });
  }
  if (scaleValue) {
    visualization.handleBalloonScaleInput({ target: scaleValue });
  }
  visualization.updateBalloonOptions();
}

function setupGeometryControls() {
  // Reset geometry view state
  geometry.resetGeometry();
  // Inline viewers are initialized when the geometry tab is shown
}

function setupCombinedResponseControls() {
  // Wire combined response controls and populate options
  const groupSelect = document.getElementById("combined-filter-group");
  const filterSelect = document.getElementById("combined-filter");
  const phaseSelect = document.getElementById("combined-phase-mode");
  const normalizeToggle = document.getElementById("combined-normalize");
  if (!groupSelect || !filterSelect) {
    return;
  }

  if (!combinedListenersBound) {
    // Bind listeners once per session
    groupSelect.addEventListener("change", updateCombinedFilterOptions);
    filterSelect.addEventListener("change", updateCombinedResponseChart);
    phaseSelect?.addEventListener("change", updateCombinedResponseChart);
    normalizeToggle?.addEventListener("change", updateCombinedResponseChart);
    combinedListenersBound = true;
  }

  const groups = currentData?.database?.filter_groups || [];
  // Populate filter group dropdown
  if (!groups.length) {
    groupSelect.innerHTML = '<option value="">No filter groups</option>';
    groupSelect.disabled = true;
  } else {
    groupSelect.disabled = false;
    groupSelect.innerHTML = [
      '<option value="">None</option>',
      ...groups.map(
        (group, i) =>
          `<option value="${i}">${escapeHtml(group.label || group.key || `Group ${i + 1}`)}</option>`,
      ),
    ].join("");
  }

  updateCombinedFilterOptions();
}

function updateCombinedFilterOptions() {
  // Populate filter dropdown for selected group
  const groupSelect = document.getElementById("combined-filter-group");
  const filterSelect = document.getElementById("combined-filter");
  if (!groupSelect || !filterSelect) {
    return;
  }

  const groups = currentData?.database?.filter_groups || [];
  const groupIndex = parseInt(groupSelect.value);
  if (Number.isNaN(groupIndex)) {
    filterSelect.innerHTML = '<option value="">None</option>';
    filterSelect.disabled = true;
    updateCombinedResponseChart();
    return;
  }

  const group = groups[groupIndex];
  const filters = group?.filters || [];
  if (!filters.length) {
    filterSelect.innerHTML = '<option value="">No filters</option>';
    filterSelect.disabled = true;
    updateCombinedResponseChart();
    return;
  }

  filterSelect.disabled = false;
  filterSelect.innerHTML = filters
    .map(
      (filter, i) =>
        `<option value="${i}">${escapeHtml(filter.label || filter.key || `Filter ${i + 1}`)}</option>`,
    )
    .join("");

  updateCombinedResponseChart();
}

function updateResponseOptions() {
  // Update response list for selected source
  const sourceSelect = document.getElementById("response-source");
  const indexSelect = document.getElementById("response-index");
  const onAxisToggle = document.getElementById("response-use-onaxis");
  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );

  const sourceIndex = parseInt(sourceSelect.value);
  const globalSelect = document.getElementById("balloon-source");
  if (globalSelect && !Number.isNaN(sourceIndex)) {
    globalSelect.value = String(sourceIndex);
  }
  if (isNaN(sourceIndex) || sourceIndex >= sourcesWithResponses.length) {
    // Clear options when selection is invalid
    indexSelect.innerHTML = "";
    updateResponseAngleControls(null, null);
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const responseCount = source.responses?.length || 0;

  // Build response option list with angle labels
  indexSelect.innerHTML = Array.from({ length: responseCount }, (_, i) => {
    const angle = computeResponseAngles(source, i);
    const angleLabel = angle
      ? ` • Az ${formatAngle(angle.meridianDeg)}° / Off ${formatAngle(angle.parallelDeg)}°`
      : "";
    return `<option value="${i}">Response ${i + 1}${angleLabel}</option>`;
  }).join("");

  if (onAxisToggle) {
    // Enable normalization only when on-axis data exists
    const onAxis = source?.definition?.on_axis_spectrum;
    const hasOnAxis =
      !!onAxis && Array.isArray(onAxis.level) && onAxis.level.length > 0;
    onAxisToggle.disabled = !hasOnAxis;
    if (!hasOnAxis) {
      onAxisToggle.checked = false;
    }
  }

  updateResponseChart();
  visualization.updatePolarOptions();
  visualization.updateBalloonOptions();
}

function updateResponseChart() {
  // Render the single-source response chart
  const sourceSelect = document.getElementById("response-source");
  const indexSelect = document.getElementById("response-index");
  const onAxisToggle = document.getElementById("response-use-onaxis");
  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );

  const sourceIndex = parseInt(sourceSelect.value);
  const respIndex = parseInt(indexSelect.value);

  if (isNaN(sourceIndex) || isNaN(respIndex)) {
    // Invalid selection
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const response = source?.responses?.[respIndex];

  if (!response) {
    // Missing response data
    updateResponseAngleControls(null, null);
    return;
  }

  updateResponseAngleControls(source, respIndex);
  updateResponseMeta(source, respIndex);

  // Update chart
  const ctx = document.getElementById("response-chart").getContext("2d");

  if (chart) {
    // Replace previous chart instance
    chart.destroy();
  }

  const onAxis = source?.definition?.on_axis_spectrum;
  // Combine on-axis level data when available
  const onAxisFreqs = buildLogFrequencies(
    onAxis?.definition,
    onAxis?.level?.length,
  );
  const canCombineOnAxis =
    !onAxisToggle?.checked &&
    onAxis &&
    Array.isArray(onAxis.level) &&
    Array.isArray(onAxisFreqs) &&
    response.frequencies.length === onAxisFreqs.length &&
    response.level.length === onAxis.level.length &&
    frequenciesMatch(response.frequencies, onAxisFreqs);
  const levelSeries = canCombineOnAxis
    ? response.level.map((value, i) => value + onAxis.level[i])
    : response.level;

  const phaseMode =
    document.getElementById("response-phase-mode")?.value || "unwrapped";
  // Combine phase and apply delay if needed
  const rawPhase = response.phase || [];
  const onAxisPhase = onAxis?.phase || [];
  const canCombinePhase =
    canCombineOnAxis &&
    Array.isArray(onAxisPhase) &&
    rawPhase.length > 0 &&
    rawPhase.length === onAxisPhase.length;
  const combinedPhase = canCombinePhase
    ? rawPhase.map((value, i) => value + onAxisPhase[i])
    : rawPhase;
  const responseDelay = Number.isFinite(response.delay) ? response.delay : 0;
  const onAxisDelay =
    canCombinePhase && Number.isFinite(onAxis?.delay) ? onAxis.delay : 0;
  const combinedDelay = responseDelay + onAxisDelay;
  const delayAdjustedPhase = applyDelayToPhase(
    combinedPhase,
    response.frequencies,
    combinedDelay,
  );
  const unwrappedPhase = unwrapPhase(delayAdjustedPhase);
  const phaseSeries = getPhaseSeries(
    phaseMode,
    response.frequencies,
    delayAdjustedPhase,
    unwrappedPhase,
  );
  const phaseLabel = canCombinePhase
    ? `${phaseSeries.label} (on-axis + directivity)`
    : phaseSeries.label;

  const frequencyData = buildFrequencyPoints(response.frequencies, levelSeries);
  const phaseData = buildFrequencyPoints(
    response.frequencies,
    phaseSeries.values,
  );
  if (!frequencyData) {
    // Abort if chart points cannot be built
    return;
  }

  const responseGridColor = getDarkModeGridColor();
  const responseXScale = buildLogFrequencyScale(
    frequencyData.minFrequency,
    frequencyData.maxFrequency,
    "Frequency",
  );
  if (responseGridColor) {
    responseXScale.grid = {
      ...(responseXScale.grid || {}),
      color: responseGridColor,
    };
  }

  chart = new Chart(ctx, {
    type: "line",
    data: {
      datasets: [
        {
          label: canCombineOnAxis
            ? "Level (dB, on-axis + directivity)"
            : "Level (dB)",
          data: frequencyData.points,
          borderColor: "#2563eb",
          backgroundColor: "rgba(37, 99, 235, 0.1)",
          fill: true,
          tension: 0.3,
          pointRadius: 0,
          yAxisID: "y",
        },
        {
          label: phaseLabel,
          data: phaseData ? phaseData.points : [],
          borderColor: "#dc2626",
          backgroundColor: "transparent",
          tension: 0.3,
          pointRadius: 0,
          yAxisID: "y1",
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: responseChartInitialized ? false : { duration: 700 },
      interaction: {
        mode: "index",
        intersect: false,
      },
      scales: {
        x: responseXScale,
        y: {
          type: "linear",
          display: true,
          position: "left",
          title: {
            display: true,
            text: "Level (dB)",
          },
          grid: responseGridColor ? { color: responseGridColor } : undefined,
        },
        y1: {
          type: "linear",
          display: true,
          position: "right",
          title: {
            display: true,
            text: phaseSeries.axisTitle,
          },
          grid: {
            drawOnChartArea: false,
          },
        },
      },
      plugins: {
        legend: {
          position: "top",
        },
        tooltip: {
          callbacks: {
            title: (items) => {
              const value = items?.[0]?.parsed?.x;
              return value ? formatFrequency(value) : "";
            },
          },
        },
        zoom: buildZoomPluginOptions(
          frequencyData.minFrequency,
          frequencyData.maxFrequency,
        ),
      },
    },
  });

  responseChartInitialized = true;
  document.getElementById("response-meta").classList.remove("empty-state");
}

function updateCombinedResponseChart() {
  // Compute and render combined array response
  const groupSelect = document.getElementById("combined-filter-group");
  const filterSelect = document.getElementById("combined-filter");
  const phaseSelect = document.getElementById("combined-phase-mode");
  const normalizeToggle = document.getElementById("combined-normalize");
  const meta = document.getElementById("combined-response-meta");
  const ctx = document
    .getElementById("combined-response-chart")
    ?.getContext("2d");

  if (!ctx || !meta) {
    // Missing DOM nodes
    return;
  }

  const state = getArrayVisualizationState();
  if (
    activeConfig &&
    (state.status === "stale" ||
      state.status === "pending" ||
      state.status === "computing" ||
      state.status === "error")
  ) {
    meta.innerHTML = buildArrayStateChips(state).join("");
    return;
  }

  if (!currentFileBytes || typeof computeArrayResponse !== "function") {
    // Missing WASM helper
    meta.innerHTML =
      '<span class="chip">Array response helper not available</span>';
    destroyCombinedChart();
    return;
  }

  if (!activeConfig) {
    meta.innerHTML =
      '<span class="chip">Build a configuration above to see combined response</span>';
    destroyCombinedChart();
    return;
  }

  let elements = buildElementsFromConfig(activeConfig);

  if (elements.length) {
    // Filter elements to valid sources
    const validSources = new Set(
      (currentData?.database?.source_definitions || []).map((s) => s.key),
    );
    elements = elements.filter((elem) => validSources.has(elem.source_key));
  }

  if (!elements.length) {
    // Nothing to compute
    meta.innerHTML =
      '<span class="chip">No valid sources found for this configuration</span>';
    destroyCombinedChart();
    return;
  }

  const sim = getSimulationParams();
  // Prepare payload for array response (on-axis point)
  const arrayPayload = JSON.stringify({
    elements,
    receiver: { x: sim.receiverDistance, y: 0, z: 0 },
    air_props: {
      temperature: sim.temperature,
      humidity: sim.humidity,
      pressure: sim.pressure,
      speed: 0,
      air_atten_on: sim.airAttenOn,
    },
  });

  let arrayResponse;
  try {
    // Delegate array response computation to WASM
    const responseJSON = computeArrayResponse(currentFileBytes, arrayPayload);
    arrayResponse = JSON.parse(responseJSON);
  } catch (err) {
    meta.innerHTML =
      '<span class="chip">Failed to compute array response</span>';
    destroyCombinedChart();
    return;
  }

  if (!arrayResponse.success) {
    // Surface computation errors
    meta.innerHTML = `<span class="chip">${escapeHtml(arrayResponse.error || "Failed to compute array response")}</span>`;
    destroyCombinedChart();
    return;
  }

  let combinedLevel = arrayResponse.level?.slice() || [];
  let combinedPhase = arrayResponse.phase?.slice() || [];
  const normalizedMode = !!normalizeToggle?.checked;
  // Optional filter group application
  let filterMessage = null;
  let filterLabel = null;
  let groupLabel = null;
  const groupIndex = parseInt(groupSelect?.value);
  const filterIndex = parseInt(filterSelect?.value);

  if (!Number.isNaN(groupIndex) && !filterSelect?.disabled) {
    // Apply filter response to combined array response
    const groups = currentData?.database?.filter_groups || [];
    const group = groups[groupIndex];
    groupLabel = group?.label || group?.key || "Filter Group";
    if (!Number.isNaN(filterIndex)) {
      const filterDef = group?.filters?.[filterIndex];
      filterLabel = filterDef?.label || filterDef?.key || "Filter";
    }
    const filterResponse = computeCombinedFilterResponse(
      groupIndex,
      filterIndex,
    );
    if (filterResponse?.error) {
      filterMessage = filterResponse.error;
    } else if (
      filterResponse?.frequencies?.length &&
      frequenciesMatch(arrayResponse.frequencies, filterResponse.frequencies)
    ) {
      combinedLevel = combinedLevel.map(
        (value, i) => value + filterResponse.level[i],
      );
      if (
        Array.isArray(filterResponse.phase) &&
        filterResponse.phase.length === combinedPhase.length
      ) {
        combinedPhase = combinedPhase.map(
          (value, i) => value + filterResponse.phase[i],
        );
      } else {
        combinedPhase = combinedPhase.length ? combinedPhase : [];
      }
    } else {
      filterMessage = "Filter response grid mismatch";
    }
  } else if (!Number.isNaN(groupIndex) && filterSelect?.disabled) {
    filterMessage = "No filters in group";
  }

  // gainOffset removed - gain is now per-element in config

  if (phaseSelect) {
    // Disable phase selector when phase data is missing
    const hasPhase =
      Array.isArray(combinedPhase) &&
      combinedPhase.length === combinedLevel.length;
    phaseSelect.disabled = !hasPhase;
    if (!hasPhase) {
      phaseSelect.value = "unwrapped";
    }
  }

  const frequencyData = buildFrequencyPoints(
    arrayResponse.frequencies,
    normalizedMode ? normalizeLevelSeries(combinedLevel) : combinedLevel,
  );
  if (!frequencyData) {
    // No data to chart
    meta.innerHTML = '<span class="chip">No combined response data</span>';
    destroyCombinedChart();
    return;
  }

  const phaseMode = phaseSelect?.value || "unwrapped";
  let phaseSeries = null;
  if (
    Array.isArray(combinedPhase) &&
    combinedPhase.length === combinedLevel.length
  ) {
    // Build phase series when available
    phaseSeries = getPhaseSeries(
      phaseMode,
      arrayResponse.frequencies,
      combinedPhase,
      unwrapPhase(combinedPhase),
    );
  }

  if (combinedChart) {
    // Replace existing chart instance
    combinedChart.destroy();
  }

  const combinedGridColor = getDarkModeGridColor();
  const combinedXScale = buildLogFrequencyScale(
    frequencyData.minFrequency,
    frequencyData.maxFrequency,
    "Frequency",
  );
  if (combinedGridColor) {
    combinedXScale.grid = {
      ...(combinedXScale.grid || {}),
      color: combinedGridColor,
    };
  }

  combinedChart = new Chart(ctx, {
    type: "line",
    data: {
      datasets: [
        {
          label: normalizedMode ? "Level (dB, normalized)" : "Level (dB SPL)",
          data: frequencyData.points,
          borderColor: "#2563eb",
          backgroundColor: "rgba(37, 99, 235, 0.1)",
          fill: true,
          tension: 0.3,
          pointRadius: 0,
          yAxisID: "y",
        },
        ...(phaseSeries
          ? [
              {
                label: phaseSeries.label,
                data:
                  buildFrequencyPoints(
                    arrayResponse.frequencies,
                    phaseSeries.values,
                  )?.points || [],
                borderColor: "#dc2626",
                backgroundColor: "transparent",
                tension: 0.3,
                pointRadius: 0,
                yAxisID: "y1",
              },
            ]
          : []),
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: combinedChartInitialized ? false : { duration: 700 },
      interaction: {
        mode: "index",
        intersect: false,
      },
      scales: {
        x: combinedXScale,
        y: {
          type: "linear",
          display: true,
          position: "left",
          title: {
            display: true,
            text: normalizedMode ? "Level (dB re max)" : "Level (dB SPL)",
          },
          grid: combinedGridColor ? { color: combinedGridColor } : undefined,
        },
        y1: phaseSeries
          ? {
              type: "linear",
              display: true,
              position: "right",
              title: {
                display: true,
                text: phaseSeries.axisTitle,
              },
              grid: {
                drawOnChartArea: false,
              },
            }
          : undefined,
      },
      plugins: {
        legend: {
          position: "top",
        },
        tooltip: {
          callbacks: {
            title: (items) => {
              const value = items?.[0]?.parsed?.x;
              return value ? formatFrequency(value) : "";
            },
          },
        },
        zoom: buildZoomPluginOptions(
          frequencyData.minFrequency,
          frequencyData.maxFrequency,
        ),
      },
    },
  });

  combinedChartInitialized = true;
  // Update metadata chips
  const configBox = {
    label:
      activeConfig.label ||
      `Config (${activeConfig.elements.length} boxes)`,
  };
  updateCombinedResponseMeta(
    meta,
    configBox,
    elements.length,
    0,
    sim.receiverDistance,
    sim.airAttenOn,
    groupLabel,
    filterLabel,
    filterMessage,
    normalizedMode,
  );
}

function normalizeLevelSeries(levels) {
  const finite = levels.filter((value) => Number.isFinite(value));
  if (!finite.length) return levels;
  const maxLevel = Math.max(...finite);
  return levels.map((value) => (Number.isFinite(value) ? value - maxLevel : value));
}

function computeCombinedFilterResponse(groupIndex, filterIndex) {
  // Request filter response from WASM and validate
  if (!currentFileBytes || typeof computeFilterResponse !== "function") {
    return { error: "Filter response helper not available" };
  }
  if (Number.isNaN(filterIndex)) {
    return { error: "Select a filter" };
  }

  const payload = JSON.stringify({
    group_index: groupIndex,
    filter_index: filterIndex,
  });
  try {
    const responseJSON = computeFilterResponse(currentFileBytes, payload);
    const response = JSON.parse(responseJSON);
    if (!response.success) {
      return { error: response.error || "Failed to compute filter response" };
    }
    if (!response.frequencies?.length || !response.level?.length) {
      return { error: response.message || "No filter response data" };
    }
    return response;
  } catch (err) {
    return { error: "Failed to compute filter response" };
  }
}

function buildCombinedElements(box) {
  // Build elements from placements or fallback sources
  const placements = box?.source_placements || [];
  if (placements.length > 0) {
    return placements
      .map((placement) => {
        const sourceKey = placement?.source_def_key;
        if (!sourceKey) {
          return null;
        }
        const pos = placement.position || {};
        const angles = placement.angles || {};
        return {
          source_key: sourceKey,
          position: {
            x: (Number(pos.x) || 0) / 1000,
            y: (Number(pos.y) || 0) / 1000,
            z: (Number(pos.z) || 0) / 1000,
          },
          angles: {
            x: toRadiansMaybe(angles.x),
            y: toRadiansMaybe(angles.y),
            z: toRadiansMaybe(angles.z),
          },
          gain: 0,
        };
      })
      .filter(Boolean);
  }

  const sources = box?.sources || [];
  return sources.map((key) => ({
    source_key: key,
    position: { x: 0, y: 0, z: 0 },
    angles: { x: 0, y: 0, z: 0 },
    gain: 0,
  }));
}

function updateCombinedResponseMeta(
  meta,
  box,
  elementCount,
  gainOffset,
  receiverDistance,
  airAttenOn,
  groupLabel,
  filterLabel,
  filterMessage,
  normalizedMode,
) {
  // Compose metadata chip list for combined response
  const chips = buildArrayStateChips(getArrayVisualizationState(), false);
  if (box) {
    chips.push(
      `<span class="chip">${escapeHtml(box.label || box.key || "Box")}</span>`,
    );
  }
  chips.push(`<span class="chip">${elementCount} sources</span>`);
  chips.push(
    `<span class="chip">Receiver ${receiverDistance.toFixed(1)} m</span>`,
  );
  chips.push(`<span class="chip">Air ${airAttenOn ? "on" : "off"}</span>`);
  chips.push(
    `<span class="chip">${normalizedMode ? "Normalized shape" : "Absolute SPL"}</span>`,
  );
  if (Number.isFinite(gainOffset) && gainOffset !== 0) {
    chips.push(`<span class="chip">Gain ${gainOffset.toFixed(1)} dB</span>`);
  }
  if (groupLabel) {
    chips.push(`<span class="chip">${escapeHtml(groupLabel)}</span>`);
  }
  if (filterLabel) {
    chips.push(`<span class="chip">${escapeHtml(filterLabel)}</span>`);
  }
  if (filterMessage) {
    chips.push(`<span class="chip">${escapeHtml(filterMessage)}</span>`);
  }
  meta.innerHTML = chips.join("");
}

function destroyCombinedChart() {
  // Tear down the combined response chart
  if (combinedChart) {
    combinedChart.destroy();
    combinedChart = null;
  }
}

function applyDelayToPhase(phaseValues, frequencies, delaySeconds) {
  // Shift phase by delay (seconds)
  if (!delaySeconds) {
    return phaseValues;
  }
  if (!Array.isArray(phaseValues) || !Array.isArray(frequencies)) {
    return phaseValues;
  }
  if (phaseValues.length !== frequencies.length) {
    return phaseValues;
  }
  const factor = 2 * Math.PI * delaySeconds;
  return phaseValues.map((value, i) => value - frequencies[i] * factor);
}

function handleResponseIndexInput(e) {
  // Slider-driven response selection
  const sourceSelect = document.getElementById("response-source");
  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );
  const sourceIndex = parseInt(sourceSelect.value);

  if (isNaN(sourceIndex) || sourceIndex >= sourcesWithResponses.length) {
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const responseCount = source.responses?.length || 0;
  if (!responseCount) {
    return;
  }

  const sliderValue = Number(e.target.value);
  const responseIndex = Math.max(0, Math.min(responseCount - 1, sliderValue));
  const indexSelect = document.getElementById("response-index");
  setResponseSelectValue(indexSelect, source, responseIndex);
  updateResponseChart();
}

function updateResponseAngleControls(source, responseIndex) {
  // Configure response angle sliders based on grid
  const azSlider = document.getElementById("response-azimuth-slider");
  const elSlider = document.getElementById("response-elevation-slider");
  const azValue = document.getElementById("response-azimuth-value");
  const elValue = document.getElementById("response-elevation-value");

  const ang = source?.definition?.balloon_data?.angular_resolution;
  const grid = source ? getBalloonGrid(source) : null;

  if (!ang || !grid || !grid.meridianCount || !grid.parallelCount) {
    azSlider.disabled = true;
    elSlider.disabled = true;
    azSlider.value = "0";
    elSlider.value = "0";
    azValue.textContent = "-";
    elValue.textContent = "-";
    return;
  }

  const azMax = (grid.meridianCount - 1) * ang.meridian_step;
  const elMax = (grid.parallelCount - 1) * ang.parallel_step;

  azSlider.disabled = false;
  elSlider.disabled = false;
  azSlider.min = "0";
  azSlider.max = String(azMax);
  azSlider.step = String(ang.meridian_step);
  elSlider.min = "0";
  elSlider.max = String(elMax);
  elSlider.step = String(ang.parallel_step);

  if (responseIndex === null || responseIndex === undefined) {
    return;
  }

  const angle = computeResponseAngles(source, responseIndex);
  if (!angle) {
    azValue.textContent = "-";
    elValue.textContent = "-";
    return;
  }

  azSlider.value = String(angle.meridianDeg);
  elSlider.value = String(angle.parallelDeg);
  azValue.textContent = `${formatAngle(angle.meridianDeg)}°`;
  elValue.textContent = `${formatAngle(angle.parallelDeg)}°`;
}

function handleResponseAngleInput() {
  // Update response index based on slider angles
  const sourceSelect = document.getElementById("response-source");
  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );
  const sourceIndex = parseInt(sourceSelect.value);

  if (isNaN(sourceIndex) || sourceIndex >= sourcesWithResponses.length) {
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const ang = source?.definition?.balloon_data?.angular_resolution;
  const grid = getBalloonGrid(source);
  if (!ang || !grid) {
    return;
  }

  const azSlider = document.getElementById("response-azimuth-slider");
  const elSlider = document.getElementById("response-elevation-slider");
  const azimuthDeg = normalizeAzimuthForGrid(Number(azSlider.value), ang);
  const elevationDeg = Number(elSlider.value);
  const meridianIdx = Math.round(azimuthDeg / ang.meridian_step);
  const parallelIdx = Math.round(elevationDeg / ang.parallel_step);
  const responseIndex = getResponseIndex(grid, meridianIdx, null, ang, {
    parallelIdx,
  });
  if (responseIndex === null) {
    return;
  }

  const indexSelect = document.getElementById("response-index");
  setResponseSelectValue(indexSelect, source, responseIndex);
  updateResponseChart();
}

function setResponseSelectValue(indexSelect, source, responseIndex) {
  // Ensure select contains and selects response index
  const existingCustom = indexSelect.querySelector(
    'option[data-custom="true"]',
  );
  if (existingCustom) {
    existingCustom.remove();
  }

  const hasOption = Array.from(indexSelect.options).some(
    (option) => Number(option.value) === responseIndex,
  );

  if (!hasOption) {
    const angle = computeResponseAngles(source, responseIndex);
    const angleLabel = angle
      ? ` • Az ${formatAngle(angle.meridianDeg)}° / Off ${formatAngle(angle.parallelDeg)}°`
      : "";
    const option = document.createElement("option");
    option.value = String(responseIndex);
    option.dataset.custom = "true";
    option.textContent = `Response ${responseIndex + 1}${angleLabel}`;
    indexSelect.appendChild(option);
  }

  indexSelect.value = String(responseIndex);
}

// Helper functions
function createTableRows(rows) {
  return rows
    .map(
      ([label, value]) => `
        <tr>
            <th>${escapeHtml(label)}</th>
            <td>${escapeHtml(String(value ?? "-"))}</td>
        </tr>
    `,
    )
    .join("");
}

function escapeHtml(str) {
  if (typeof str !== "string") return str;
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function formatSystemType(type) {
  const types = {
    0: "Line Array",
    1: "Cluster",
    2: "Loudspeaker",
  };
  return types[type] || `Unknown (${type})`;
}

function formatDataType(type) {
  const types = {
    0: "High Resolution",
    1: "1/3 Octave",
    2: "Octave",
  };
  return types[type] || `Unknown (${type})`;
}

function formatDirectivityType(type) {
  if (type === null || type === undefined || Number.isNaN(type)) return "";
  const types = {
    0: "Point",
    1: "Line",
    2: "Circular Piston",
    3: "Rectangular Piston",
  };
  return types[type] || `Unknown (${type})`;
}

function formatGainNumber(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return formatGain(Number(value));
}

function formatVoltage(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return `${formatNumber(value, 2)} V`;
}

function formatDistance(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return `${formatNumber(value, 2)} m`;
}

function formatTemperature(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return `${formatNumber(value, 1)} °C`;
}

function formatPercent(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return `${formatNumber(value, 1)} %`;
}

function formatPressure(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return `${formatNumber(value, 2)} kPa`;
}

function formatOhms(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return `${formatNumber(value, 2)} ohm`;
}

function formatCoverageAngles(horizontal, vertical) {
  const hValid = !(
    horizontal === null ||
    horizontal === undefined ||
    Number.isNaN(horizontal)
  );
  const vValid = !(
    vertical === null ||
    vertical === undefined ||
    Number.isNaN(vertical)
  );
  if (!hValid && !vValid) return "-";
  const hText = hValid ? `${formatAngle(horizontal)}°` : "-";
  const vText = vValid ? `${formatAngle(vertical)}°` : "-";
  return `H ${hText} / V ${vText}`;
}

function formatLimitType(type) {
  const types = {
    0: "Max Count",
    1: "Max Count Type",
    2: "Max Weight (kg)",
    4: "Max Tilt Angle",
    5: "Min Tilt Angle",
    6: "Min Count",
  };
  return types[type] || `Limit Type ${type}`;
}

function formatWarningType(type) {
  const types = {
    0: "Max Count Warning",
    1: "Min Count Warning",
    2: "Max Weight Warning",
    3: "Max Tilt Warning",
    4: "Min Tilt Warning",
  };
  return types[type] || `Warning Type ${type}`;
}

function formatFilterKind(kind) {
  const kinds = {
    0: "LogSpectrum",
    1: "IIR",
    2: "FIR",
  };
  return kinds[kind] || `Unknown (${kind})`;
}

function formatIIRFilterType(type) {
  const types = {
    0: "Low-Pass",
    1: "High-Pass",
    2: "All-Pass",
    3: "Peak",
    4: "Peak (Sym)",
    5: "Low-Shelf",
    6: "High-Shelf",
  };
  return types[type] || `Type ${type}`;
}

function formatIIRFilterShape(shape) {
  const shapes = {
    0: "Butterworth",
    1: "Linkwitz-Riley",
    2: "Bessel",
    3: "Sallen-Key",
  };
  return shapes[shape] || `Shape ${shape}`;
}

function formatFilterAlign(align) {
  const alignments = {
    0: "None",
    1: "-3 dB",
    2: "-6 dB",
    3: "Phase-Matched",
  };
  return alignments[align] || `Align ${align}`;
}

function formatGain(db) {
  if (db === 0) return "0 dB";
  return (db > 0 ? "+" : "") + db.toFixed(1) + " dB";
}

function formatDelay(seconds) {
  if (!seconds || seconds === 0) return "-";
  if (seconds >= 0.001) {
    return (seconds * 1000).toFixed(2) + " ms";
  }
  return (seconds * 1000000).toFixed(1) + " µs";
}

function formatSampleRate(hz) {
  if (!hz || hz === 0) return "-";
  return (hz / 1000).toFixed(hz % 1000 === 0 ? 0 : 1) + " kHz";
}

function renderFilterDetails(filter) {
  // Render filter bank details for UI
  if (!filter) return "";

  const bank = filter.filter;
  if (!bank) return '<div class="filter-detail">No filter data</div>';

  let html = '<div class="filter-bank">';

  // Bank-level settings
  const bankFlags = [];
  if (bank.bypass) bankFlags.push("Bypassed");
  if (bank.invert_polarity) bankFlags.push("Inverted");
  if (bank.mute_input) bankFlags.push("Muted");

  html += '<div class="filter-bank-header">';
  html += `<span class="filter-bank-gain">Gain: ${formatGain(bank.gain || 0)}</span>`;
  if (bank.delay) html += ` • <span>Delay: ${formatDelay(bank.delay)}</span>`;
  if (bankFlags.length > 0)
    html += ` • <span class="filter-flags">${bankFlags.join(", ")}</span>`;
  html += "</div>";

  // Individual filters in the bank
  if (bank.filters && bank.filters.length > 0) {
    html += '<div class="filter-chain">';
    bank.filters.forEach((f) => {
      html += renderSingleFilter(f);
    });
    html += "</div>";
  } else {
    html += '<div class="filter-empty">No filters in chain</div>';
  }

  html += "</div>";
  return html;
}

function renderSingleFilter(f) {
  // Render a single filter summary row
  const kindName = formatFilterKind(f.kind);
  const flags = [];
  if (f.bypass) flags.push("Bypassed");
  if (f.invert_polarity) flags.push("Inverted");

  let html = `<div class="filter-item filter-kind-${kindName.toLowerCase()}">`;
  html += `<div class="filter-item-header">`;
  html += `<span class="filter-kind-badge">${kindName}</span>`;
  if (f.label)
    html += ` <span class="filter-label">${escapeHtml(f.label)}</span>`;
  html += "</div>";

  html += '<div class="filter-item-params">';

  // Common parameters
  const params = [];
  if (f.gain !== 0) params.push(`Gain: ${formatGain(f.gain)}`);
  if (f.delay) params.push(`Delay: ${formatDelay(f.delay)}`);
  if (flags.length > 0) params.push(flags.join(", "));

  // Type-specific parameters
  if (f.kind === 1 && f.iir_params) {
    // IIR filter
    const p = f.iir_params;
    params.push(formatIIRFilterType(p.filter_type));
    params.push(formatIIRFilterShape(p.filter_shape));
    params.push(`Order: ${p.order}`);
    params.push(`Freq: ${formatFrequency(p.freq_crit_hz)}`);
    if (p.filter_shape === 3 && p.q_factor) {
      params.push(`Q: ${p.q_factor.toFixed(2)}`);
    }
    if (p.alignment !== 0) {
      params.push(`Align: ${formatFilterAlign(p.alignment)}`);
    }
    if (p.parametric_gain_db !== 0) {
      params.push(`Param Gain: ${formatGain(p.parametric_gain_db)}`);
    }
  } else if (f.kind === 2 && f.fir_data) {
    // FIR filter
    const d = f.fir_data;
    params.push(d.is_time_response ? "Time Domain" : "Freq Domain");
    if (d.is_complex) params.push("Complex");
    if (d.sample_rate) params.push(`SR: ${formatSampleRate(d.sample_rate)}`);
    if (d.data_irm) params.push(`${d.data_irm.length} coefficients`);
  } else if (f.kind === 0 && f.log_spectrum) {
    // LogSpectrum filter
    const s = f.log_spectrum;
    if (s.lowest_frequency && s.number_of_bands && s.bands_per_octave) {
      const highestFreq =
        s.lowest_frequency *
        Math.pow(2, s.number_of_bands / s.bands_per_octave);
      params.push(
        `${formatFrequency(s.lowest_frequency)} - ${formatFrequency(highestFreq)}`,
      );
      params.push(`${s.number_of_bands} bands`);
      params.push(`${s.bands_per_octave}/oct`);
    }
    if (s.delay) params.push(`Delay: ${formatDelay(s.delay)}`);
  }

  html += params.join(" • ");
  html += "</div></div>";
  return html;
}

function formatFrequency(hz) {
  if (!hz || hz === 0) return "-";
  if (hz >= 1000) {
    return (hz / 1000).toFixed(hz % 1000 === 0 ? 0 : 1) + " kHz";
  }
  return hz.toFixed(0) + " Hz";
}

function formatAngle(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  const formatted = Number(value).toFixed(1);
  return formatted.endsWith(".0") ? formatted.slice(0, -2) : formatted;
}

function normalizeAngleDegrees(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return null;
  const absVal = Math.abs(value);
  if (absVal > Math.PI * 2 + 1e-6) {
    return value;
  }
  return (value * 180) / Math.PI;
}

function toRadiansMaybe(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return 0;
  const absVal = Math.abs(value);
  if (absVal > Math.PI * 2 + 1e-6) {
    return (value * Math.PI) / 180;
  }
  return value;
}

function formatAngleDegrees(value) {
  const deg = normalizeAngleDegrees(value);
  if (deg === null) return "-";
  return `${formatAngle(deg)}°`;
}

function formatPosition(position) {
  if (!position) return "-";
  return [
    formatNumber(position.x, 1),
    formatNumber(position.y, 1),
    formatNumber(position.z, 1),
  ].join(", ");
}

function computeResponseAngles(source, responseIndex) {
  const balloon = source?.definition?.balloon_data;
  const ang = balloon?.angular_resolution;
  if (!ang || !ang.meridian_step || !ang.parallel_step) {
    return null;
  }

  const grid = getBalloonGrid(source);
  if (!grid || !grid.meridianCount || !grid.parallelCount) {
    return null;
  }

  const indices = responseIndexToBalloonIndices(responseIndex, grid);
  if (!indices) {
    return null;
  }
  const { meridianIdx, parallelIdx } = indices;

  if (parallelIdx >= grid.parallelCount) {
    return null;
  }

  let meridianDeg = meridianIdx * ang.meridian_step;
  if (ang.symmetry === 2) {
    meridianDeg = (meridianDeg + 90) % 360;
  }

  return {
    meridianDeg,
    parallelDeg: parallelIdx * ang.parallel_step,
    meridianIdx,
    parallelIdx,
    meridianCount: grid.meridianCount,
    parallelCount: grid.parallelCount,
  };
}

function updateResponseMeta(source, responseIndex) {
  const meta = document.getElementById("response-meta");
  if (!meta) return;
  meta.innerHTML = buildResponseMetaHtml(source, responseIndex);
}

function buildResponseMetaHtml(source, responseIndex) {
  const responseCount = source?.responses?.length || 0;
  const angle = computeResponseAngles(source, responseIndex);
  const angRes = source?.definition?.balloon_data?.angular_resolution;

  const chips = [];
  chips.push(
    `<span class="chip">Response ${responseIndex + 1} of ${responseCount}</span>`,
  );

  if (angle) {
    chips.push(
      `<span class="chip">Azimuth ${formatAngle(angle.meridianDeg)}°</span>`,
    );
    chips.push(
      `<span class="chip">Off-axis ${formatAngle(angle.parallelDeg)}°</span>`,
    );
  } else {
    chips.push('<span class="chip">Angle data unavailable</span>');
  }

  if (angRes?.meridian_step && angRes?.parallel_step) {
    chips.push(
      `<span class="chip">Grid ${formatAngle(angRes.meridian_step)}° × ${formatAngle(angRes.parallel_step)}°</span>`,
    );
  }

  return chips.join("");
}

function buildPolarAngles(stepDeg) {
  const angles = [0];
  for (let angle = -stepDeg; angle >= -180; angle -= stepDeg) {
    angles.push(angle);
  }
  for (let angle = 180 - stepDeg; angle > 0; angle -= stepDeg) {
    angles.push(angle);
  }
  return angles;
}

function formatPolarLabel(angleDeg) {
  const normalized = ((angleDeg + 180) % 360) - 180;
  if (Math.abs(normalized) === 180) {
    return "±180°";
  }
  if (Math.abs(normalized) < 1e-6) {
    return "0°";
  }
  return `${formatAngle(normalized)}°`;
}

function computePolarSlices(source, freqIndex) {
  // Build horizontal/vertical polar slices for a frequency index
  const grid = getBalloonGrid(source);
  if (!grid) {
    return null;
  }

  const stepDeg = 10;
  const angles = buildPolarAngles(stepDeg);
  const labels = angles.map(formatPolarLabel);
  const horizontalLevels = [];
  const verticalLevels = [];

  const maxParallel = grid.measuredParallelDeg;
  const canMirrorParallel = grid.symmetry === 2 || grid.symmetry === 3;
  const onAxis = source?.definition?.on_axis_spectrum;
  const onAxisFreqs = buildLogFrequencies(
    onAxis?.definition,
    onAxis?.level?.length,
  );
  const sampleResponse = source?.responses?.[0];
  const canCombineOnAxis =
    onAxis &&
    Array.isArray(onAxis.level) &&
    Array.isArray(onAxisFreqs) &&
    sampleResponse &&
    sampleResponse.frequencies.length === onAxisFreqs.length &&
    sampleResponse.level.length === onAxis.level.length &&
    frequenciesMatch(sampleResponse.frequencies, onAxisFreqs);

  // Both slices are great circles through the front-back axis (parallel 0-180).
  // GLL coordinate system: meridian = rotation around the firing axis,
  // parallel = angle from the firing axis (colatitude).
  // Meridian 0° = "top" of the speaker, 90° = "right", 180° = "bottom", 270° = "left".
  //
  // Horizontal slice (Front-Right-Back-Left plane):
  //   positive chart angles → meridian=90°  (right), parallel = angle
  //   negative chart angles → meridian=270° (left),  parallel = |angle|
  //
  // Vertical slice (Front-Top-Back-Bottom plane):
  //   positive chart angles → meridian=0°   (top),    parallel = angle
  //   negative chart angles → meridian=180° (bottom), parallel = |angle|

  angles.forEach((angle) => {
    // Horizontal slice sample
    // Horizontal slice: great circle through front-right-back-left.
    const hParallelDeg = Math.abs(angle);
    const hMeridianDeg = angle >= 0 ? 90 : 270;

    const horizontalResponse = getResponseWithSymmetry(
      source,
      grid,
      hMeridianDeg,
      hParallelDeg,
    );
    const horizontalLevel = horizontalResponse?.level?.[freqIndex];
    horizontalLevels.push(
      canCombineOnAxis && Number.isFinite(horizontalLevel)
        ? horizontalLevel + onAxis.level[freqIndex]
        : (horizontalLevel ?? null),
    );

    // Vertical slice: great circle through front-top-back-bottom.
    const vParallelDeg = Math.abs(angle);
    const vMeridianDeg = angle >= 0 ? 0 : 180;

    const verticalResponse = getResponseWithSymmetry(
      source,
      grid,
      vMeridianDeg,
      vParallelDeg,
    );
    const verticalLevel = verticalResponse?.level?.[freqIndex];
    verticalLevels.push(
      canCombineOnAxis && Number.isFinite(verticalLevel)
        ? verticalLevel + onAxis.level[freqIndex]
        : (verticalLevel ?? null),
    );
  });

  return {
    labels,
    horizontal: {
      levels: horizontalLevels,
      // Great circle through front-right-back-left (meridian 90°/270°).
      meridianDeg: 90,
    },
    vertical: {
      levels: verticalLevels,
      // Great circle through front-top-back-bottom (meridian 0°/180°).
      meridianDeg: 0,
      maxParallel,
      canMirrorParallel,
    },
    meta: {
      usesOnAxis: !!canCombineOnAxis,
      symmetry: grid.symmetry,
      symmetryName: grid.symmetryName,
      frontHalfOnly: grid.frontHalfOnly,
      measuredMeridianDeg: grid.measuredMeridianDeg,
      measuredParallelDeg: grid.measuredParallelDeg,
      stepDeg,
    },
  };
}

function getBalloonGrid(source) {
  // Calculate balloon grid dimensions and symmetry
  const balloon = source?.definition?.balloon_data;
  const ang = balloon?.angular_resolution;
  if (!ang || !ang.meridian_step || !ang.parallel_step) {
    return null;
  }

  // Get symmetry type and front_half_only flag
  const symmetry = ang?.symmetry ?? 0;
  const frontHalfOnly = !!ang?.front_half_only;
  const symmetryNames = ["None", "Vertical", "Horizontal", "Quarter", "Axial"];

  // Full-sphere grid dimensions (for rendering loops).
  const fullMeridianCount = Math.max(1, Math.round(360 / ang.meridian_step));
  const fullParallelCount = Math.max(
    1,
    Math.round(180 / ang.parallel_step) + 1,
  );

  // Measured (stored) grid dimensions based on symmetry and front_half_only.
  // See docs/response.md section 5 for derivation.
  const responseCount = source?.responses?.length || 0;
  let meridianCount;
  switch (symmetry) {
    case 4: // Axial — rotational symmetry, single meridian
      meridianCount = 1;
      break;
    case 3: // Quarter — 0° to 90° measured
      meridianCount = Math.max(1, Math.round(90 / ang.meridian_step) + 1);
      break;
    case 1: // Vertical — 0° to 180° measured
    case 2: // Horizontal — 0° to 180° measured
      meridianCount = Math.max(1, Math.round(180 / ang.meridian_step) + 1);
      break;
    default: // None — full 360° (no +1, wraps around)
      meridianCount = fullMeridianCount;
      break;
  }
  let parallelCount = frontHalfOnly
    ? Math.max(1, Math.round(90 / ang.parallel_step) + 1)
    : fullParallelCount;

  // Validate against actual response count (accounting for pole deduplication).
  // At parallel=0 all meridians collapse to 1 stored point (saves merCount-1).
  // At parallel=max (if full sphere) same collapse applies.
  const expectedWithDedup =
    meridianCount * parallelCount -
    (meridianCount > 1 ? meridianCount - 1 : 0) -
    (!frontHalfOnly && meridianCount > 1 ? meridianCount - 1 : 0);

  if (
    responseCount > 0 &&
    responseCount !== meridianCount * parallelCount &&
    responseCount !== expectedWithDedup
  ) {
    // Fallback: back-calculate from responseCount if symmetry-based calc doesn't match.
    if (window?.GLL_DEBUG_BALLOON) {
      console.warn("[Balloon Grid] Symmetry-based grid mismatch", {
        symmetry,
        computed: `${meridianCount}×${parallelCount}=${meridianCount * parallelCount}`,
        withDedup: expectedWithDedup,
        actual: responseCount,
      });
    }
    // Try using responseCount / meridianCount as parallelCount
    const altParallel = Math.ceil(responseCount / meridianCount);
    if (altParallel > 0 && altParallel <= fullParallelCount) {
      parallelCount = altParallel;
    }
  }

  // Calculate actual measured angular ranges
  const measuredMeridianDeg = (meridianCount - 1) * ang.meridian_step;
  const measuredParallelDeg = (parallelCount - 1) * ang.parallel_step;

  const grid = {
    meridianCount,
    parallelCount,
    symmetry,
    frontHalfOnly,
    meridianRange: 360,
    parallelRange: 180,
    measuredMeridianDeg, // Actual measured azimuth range
    measuredParallelDeg, // Actual measured parallel range
    meridianStep: ang.meridian_step,
    parallelStep: ang.parallel_step,
    responseCount,
    fullMeridianCount,
    fullParallelCount,
    symmetryName: symmetryNames[symmetry] || "Unknown",
    // Legacy compatibility
    measuredAzimuthRange: measuredMeridianDeg,
    measuredElevationRange: measuredParallelDeg,
  };

  // Debug logging (toggle for development)
  if (window?.GLL_DEBUG_BALLOON) {
    console.log("[Balloon Grid]", {
      symmetry: grid.symmetry,
      symmetryName: grid.symmetryName,
      responseCount,
      calculatedGrid: `${meridianCount} × ${parallelCount} = ${
        meridianCount * parallelCount
      }`,
      measuredRanges: {
        meridian: `0-${measuredMeridianDeg}°`,
        parallel: `0-${measuredParallelDeg}°`,
      },
      resolution: `${ang.meridian_step}° × ${ang.parallel_step}°`,
    });
  }

  return grid;
}

function mapAnglesBySymmetry(grid, azimuthDeg, parallelDeg) {
  if (!grid) {
    return null;
  }

  let lookupAzimuth = ((azimuthDeg % 360) + 360) % 360;
  let lookupParallel = parallelDeg;

  // Symmetry codes: 0=None, 1=Vertical, 2=Horizontal, 3=Quarter, 4=Axial
  const symmetry = grid.symmetry ?? 0;
  const canMirrorParallel = symmetry === 2 || symmetry === 3;

  if (symmetry === 4) {
    // Axial: rotational symmetry around the vertical axis.
    lookupAzimuth = 0;
  } else if (symmetry === 3) {
    // Quarter: fold 360° into 0-90° by successive mirroring.
    if (lookupAzimuth >= 270) {
      lookupAzimuth = 360 - lookupAzimuth;
    } else if (lookupAzimuth >= 180) {
      lookupAzimuth = lookupAzimuth - 180;
    } else if (lookupAzimuth >= 90) {
      lookupAzimuth = 180 - lookupAzimuth;
    }
  } else if (symmetry === 1) {
    // Vertical: mirror front/back across 180°.
    if (lookupAzimuth >= 180) {
      lookupAzimuth = 360 - lookupAzimuth;
    }
  } else if (symmetry === 2) {
    // Horizontal: rotate by -90° then mirror around that axis.
    lookupAzimuth = lookupAzimuth - 90;
    if (lookupAzimuth < 0) {
      lookupAzimuth = -lookupAzimuth;
    } else if (lookupAzimuth >= 180) {
      lookupAzimuth = 360 - lookupAzimuth;
    }
  }

  if (lookupParallel < 0 || lookupParallel > 180) {
    return null;
  }

  if (grid.frontHalfOnly && lookupParallel > 90) {
    return null;
  }

  if (lookupParallel > grid.measuredParallelDeg) {
    // Mirror elevation if symmetry allows it
    if (canMirrorParallel) {
      // Mirror elevation across the horizontal plane (parallel=90°).
      const mirrored = 180 - lookupParallel;
      if (mirrored <= grid.measuredParallelDeg) {
        lookupParallel = mirrored;
      } else {
        return null;
      }
    } else {
      return null;
    }
  }

  return { meridianDeg: lookupAzimuth, parallelDeg: lookupParallel };
}

// Get response at given azimuth/parallel angles, applying symmetry mirroring
// and bilinear interpolation in the complex-pressure domain.
function getResponseWithSymmetry(source, grid, azimuthDeg, parallelDeg) {
  const responses = source?.responses || [];
  const ang = source?.definition?.balloon_data?.angular_resolution;

  if (!responses.length || !ang || !grid) {
    return null;
  }

  const mapped = mapAnglesBySymmetry(grid, azimuthDeg, parallelDeg);
  if (!mapped) {
    return null;
  }

  const meridianIdx = mapped.meridianDeg / ang.meridian_step;
  const parallelIdx = mapped.parallelDeg / ang.parallel_step;

  return interpolateResponseAtGrid(source, grid, meridianIdx, parallelIdx);
}

function interpolateResponseAtGrid(source, grid, meridianIdx, parallelIdx) {
  const responses = source?.responses || [];
  if (!responses.length || !grid) {
    return null;
  }

  let meridianIdx0 = Math.floor(meridianIdx);
  let meridianIdx1 = Math.ceil(meridianIdx);
  if (grid.symmetry === 0 && grid.meridianCount > 1) {
    meridianIdx0 =
      ((meridianIdx0 % grid.meridianCount) + grid.meridianCount) %
      grid.meridianCount;
    meridianIdx1 =
      ((meridianIdx1 % grid.meridianCount) + grid.meridianCount) %
      grid.meridianCount;
  } else {
    meridianIdx0 = clampIndex(meridianIdx0, grid.meridianCount);
    meridianIdx1 = clampIndex(meridianIdx1, grid.meridianCount);
  }

  const parallelIdx0 = clampIndex(Math.floor(parallelIdx), grid.parallelCount);
  const parallelIdx1 = clampIndex(Math.ceil(parallelIdx), grid.parallelCount);

  const r00 = getResponseAtGridPoint(source, grid, meridianIdx0, parallelIdx0);
  const r01 = getResponseAtGridPoint(source, grid, meridianIdx1, parallelIdx0);
  const r10 = getResponseAtGridPoint(source, grid, meridianIdx0, parallelIdx1);
  const r11 = getResponseAtGridPoint(source, grid, meridianIdx1, parallelIdx1);

  if (!Array.isArray(r00?.level) || r00.level.length === 0) {
    return null;
  }

  if (meridianIdx0 === meridianIdx1 && parallelIdx0 === parallelIdx1) {
    return r00;
  }

  const [w00, w01, w10, w11] = bilinearWeights(meridianIdx, parallelIdx);
  const level = [];
  const phase = [];
  for (let band = 0; band < r00.level.length; band += 1) {
    const interpolated = interpolateComplexPressureBand(
      band,
      w00,
      w01,
      w10,
      w11,
      r00,
      r01,
      r10,
      r11,
    );
    level.push(interpolated.level);
    phase.push(interpolated.phase);
  }

  return {
    frequencies: r00.frequencies,
    level,
    phase,
    delay: r00.delay || 0,
  };
}

function clampIndex(index, count) {
  if (index < 0) return 0;
  if (index >= count) return count - 1;
  return index;
}

function bilinearWeights(meridianIdx, parallelIdx) {
  const meridianFrac = meridianIdx - Math.floor(meridianIdx);
  const parallelFrac = parallelIdx - Math.floor(parallelIdx);
  return [
    (1 - meridianFrac) * (1 - parallelFrac),
    meridianFrac * (1 - parallelFrac),
    (1 - meridianFrac) * parallelFrac,
    meridianFrac * parallelFrac,
  ];
}

function interpolateComplexPressureBand(
  band,
  w00,
  w01,
  w10,
  w11,
  r00,
  r01,
  r10,
  r11,
) {
  const [real00, imag00] = weightedComplexPressure(w00, r00, band);
  const [real01, imag01] = weightedComplexPressure(w01, r01, band);
  const [real10, imag10] = weightedComplexPressure(w10, r10, band);
  const [real11, imag11] = weightedComplexPressure(w11, r11, band);

  const realSum = real00 + real01 + real10 + real11;
  const imagSum = imag00 + imag01 + imag10 + imag11;
  if (Math.abs(realSum) < 1e-12 && Math.abs(imagSum) < 1e-12) {
    return { level: -200, phase: 0 };
  }

  return {
    level: 20 * Math.log10(Math.hypot(realSum, imagSum)),
    phase: Math.atan2(imagSum, realSum),
  };
}

function weightedComplexPressure(weight, response, band) {
  const level = response?.level?.[band];
  if (!weight || !Number.isFinite(level)) {
    return [0, 0];
  }

  const phase = Number.isFinite(response?.phase?.[band])
    ? response.phase[band]
    : 0;
  const magnitude = weight * Math.pow(10, level / 20);
  return [magnitude * Math.cos(phase), magnitude * Math.sin(phase)];
}

function getResponseAtGridPoint(source, grid, meridianIdx, parallelIdx) {
  const responses = source?.responses || [];
  if (!responses.length || !grid) {
    return null;
  }

  // Bounds check
  if (
    meridianIdx < 0 ||
    meridianIdx >= grid.meridianCount ||
    parallelIdx < 0 ||
    parallelIdx >= grid.parallelCount
  ) {
    return null;
  }

  // Calculate response index using meridian-major order with pole deduplication.
  // Storage layout:
  //   mer=0: all parallelCount entries (indices 0 .. parCount-1)
  //   mer=1..N: skip front pole (par=0) and back pole (par=last, unless frontHalfOnly)
  // At poles, all meridians collapse to meridian=0's entry.
  const responseIndex = balloonResponseIndex(
    meridianIdx,
    parallelIdx,
    grid.meridianCount,
    grid.parallelCount,
    grid.frontHalfOnly,
  );

  if (
    responseIndex !== null &&
    responseIndex >= 0 &&
    responseIndex < responses.length
  ) {
    return responses[responseIndex];
  }

  return null;
}

// Compute the flat array index for a (meridianIdx, parallelIdx) pair,
// matching the EASE legacy storage order: meridian-major with pole dedup.
function balloonResponseIndex(
  meridianIdx,
  parallelIdx,
  meridianCount,
  parallelCount,
  frontHalfOnly,
) {
  const lastParIdx = parallelCount - 1;
  const isFrontPole = parallelIdx === 0;
  const isBackPole = parallelIdx === lastParIdx && !frontHalfOnly;

  // Poles are stored only once (at meridian=0).
  if (isFrontPole || isBackPole) {
    // mer=0 stores all parallels sequentially: index = parallelIdx
    return parallelIdx;
  }

  if (meridianIdx === 0) {
    // First meridian stores all parallels (including poles): index = parallelIdx
    return parallelIdx;
  }

  // Subsequent meridians skip both poles (or just front pole if frontHalfOnly).
  // mer=0 contributes parallelCount entries.
  // Each subsequent meridian contributes (parallelCount - 2) entries (full sphere)
  // or (parallelCount - 1) entries (front-half only, no back pole to skip).
  const skippedPerMer = frontHalfOnly ? 1 : 2;
  const pointsPerMer = parallelCount - skippedPerMer;

  return parallelCount + (meridianIdx - 1) * pointsPerMer + (parallelIdx - 1);
}

function responseIndexToBalloonIndices(responseIndex, grid) {
  // Map response index back to meridian/parallel indices
  if (!grid || !grid.meridianCount || !grid.parallelCount) {
    return null;
  }
  if (!Number.isFinite(responseIndex) || responseIndex < 0) {
    return null;
  }

  const meridianCount = grid.meridianCount;
  const parallelCount = grid.parallelCount;
  const responseCount = grid.responseCount || meridianCount * parallelCount;

  if (responseIndex >= responseCount) {
    return null;
  }

  if (responseIndex < parallelCount) {
    // First meridian includes poles and all parallels
    return {
      meridianIdx: 0,
      parallelIdx: responseIndex,
    };
  }

  const pointsPerMer = parallelCount - (grid.frontHalfOnly ? 1 : 2);
  if (pointsPerMer <= 0) {
    return null;
  }

  const offset = responseIndex - parallelCount;
  const meridianIdx = Math.floor(offset / pointsPerMer) + 1;
  const parallelIdx = (offset % pointsPerMer) + 1;

  if (
    meridianIdx < 0 ||
    meridianIdx >= meridianCount ||
    parallelIdx < 0 ||
    parallelIdx >= parallelCount
  ) {
    return null;
  }

  return { meridianIdx, parallelIdx };
}

function normalizeAzimuthForGrid(azimuthDeg, ang) {
  if (!Number.isFinite(azimuthDeg)) {
    return azimuthDeg;
  }
  if (ang?.symmetry === 2) {
    return (((azimuthDeg - 90) % 360) + 360) % 360;
  }
  return azimuthDeg;
}

function getResponseIndex(
  grid,
  meridianIdx,
  parallelDeg,
  ang,
  overrides = null,
) {
  // Compute response index for given angles or overrides
  if (!grid || !grid.meridianCount || !grid.parallelCount) {
    return null;
  }

  let localMeridianIdx = meridianIdx;
  let localParallelIdx = null;

  if (overrides) {
    // Allow explicit overrides for indices and azimuth
    if (overrides.parallelIdx !== undefined) {
      localParallelIdx = overrides.parallelIdx;
    }
    if (overrides.azimuthDeg !== undefined) {
      const maxMeridianDeg = (grid.meridianCount - 1) * ang.meridian_step;
      if (overrides.azimuthDeg > maxMeridianDeg + ang.meridian_step * 0.5) {
        return null;
      }
      const rawIdx = Math.round(overrides.azimuthDeg / ang.meridian_step);
      localMeridianIdx = Math.min(grid.meridianCount - 1, Math.max(0, rawIdx));
    }
  }

  if (localParallelIdx === null) {
    // Derive parallel index from degrees
    if (parallelDeg === null || parallelDeg === undefined) {
      return null;
    }
    const maxParallelDeg = (grid.parallelCount - 1) * ang.parallel_step;
    if (parallelDeg > maxParallelDeg + ang.parallel_step * 0.5) {
      return null;
    }
    localParallelIdx = Math.min(
      grid.parallelCount - 1,
      Math.max(0, Math.round(parallelDeg / ang.parallel_step)),
    );
  }

  if (localMeridianIdx === null || localMeridianIdx === undefined) {
    return null;
  }

  if (localMeridianIdx < 0 || localMeridianIdx >= grid.meridianCount) {
    return null;
  }

  if (localParallelIdx < 0 || localParallelIdx >= grid.parallelCount) {
    return null;
  }

  const responseIndex = balloonResponseIndex(
    localMeridianIdx,
    localParallelIdx,
    grid.meridianCount,
    grid.parallelCount,
    grid.frontHalfOnly,
  );
  if (
    responseIndex === null ||
    (grid.responseCount && responseIndex >= grid.responseCount)
  ) {
    return null;
  }

  return responseIndex;
}

function averageLevels(a, b) {
  const hasA = a !== null && a !== undefined && !Number.isNaN(a);
  const hasB = b !== null && b !== undefined && !Number.isNaN(b);
  if (hasA && hasB) {
    return (a + b) / 2;
  }
  if (hasA) return a;
  if (hasB) return b;
  return null;
}

function buildLogFrequencies(definition, countOverride) {
  if (!definition) return null;
  const bandsPerOctave = Number(definition.bands_per_octave);
  const startFreq = Number(definition.start_freq);
  const pointCount = Number(definition.point_count);
  if (!bandsPerOctave || !startFreq) return null;
  const count =
    Number.isFinite(countOverride) && countOverride > 0
      ? countOverride
      : pointCount;
  if (!count || count <= 0) return null;
  return Array.from(
    { length: count },
    (_, i) => startFreq * Math.pow(2, i / bandsPerOctave),
  );
}

function frequenciesMatch(a, b) {
  if (!Array.isArray(a) || !Array.isArray(b) || a.length !== b.length) {
    return false;
  }
  const tol = 1e-3;
  for (let i = 0; i < a.length; i += 1) {
    const av = a[i];
    const bv = b[i];
    if (!Number.isFinite(av) || !Number.isFinite(bv)) {
      return false;
    }
    const rel = Math.abs(av - bv) / Math.max(1, Math.abs(bv));
    if (rel > tol) {
      return false;
    }
  }
  return true;
}

function formatNumber(value, digits) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return Number(value).toFixed(digits);
}

function formatBytes(bytes) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024;
    i++;
  }
  return bytes.toFixed(i === 0 ? 0 : 1) + " " + units[i];
}

// Filter group toggle for expandable display
// eslint-disable-next-line no-unused-vars
function toggleFilterGroup(idx) {
  const group = document.querySelector(
    `.filter-group[data-group-idx="${idx}"]`,
  );
  if (!group) return;

  const content = group.querySelector(".filter-group-content");
  const toggle = group.querySelector(".filter-group-toggle");

  if (content.style.display === "none") {
    content.style.display = "block";
    toggle.textContent = "▼";
    group.classList.add("expanded");
    renderFilterGroupResponse(idx);
  } else {
    content.style.display = "none";
    toggle.textContent = "▶";
    group.classList.remove("expanded");
  }
}

// Source toggle for expandable response display
// eslint-disable-next-line no-unused-vars
function toggleSource(idx) {
  const card = document.querySelector(`.source-card[data-source-idx="${idx}"]`);
  if (!card) return;

  const content = card.querySelector(".source-content");
  const toggle = card.querySelector(".source-toggle");

  if (content.style.display === "none") {
    content.style.display = "block";
    toggle.textContent = "▼";
    card.classList.add("expanded");
    renderSourcePlot(idx);
  } else {
    content.style.display = "none";
    toggle.textContent = "▶";
    card.classList.remove("expanded");
    stopSourceBalloonLoop(idx);
  }
}

window.toggleSource = toggleSource;
window.toggleFilterGroup = toggleFilterGroup;
window.toggleCard = toggleCard;

// Collapsible card toggle with localStorage persistence
// eslint-disable-next-line no-unused-vars
function toggleCard(cardId) {
  const card = document.querySelector(
    `.card-collapsible[data-card="${cardId}"]`,
  );
  if (!card) return;

  const isCollapsed = card.classList.toggle("collapsed");

  // Persist state to localStorage
  const cardStates = JSON.parse(
    localStorage.getItem("gll-card-states") || "{}",
  );
  cardStates[cardId] = isCollapsed;
  localStorage.setItem("gll-card-states", JSON.stringify(cardStates));
}

// Restore card collapsed states from localStorage
function restoreCardStates() {
  const cardStates = JSON.parse(
    localStorage.getItem("gll-card-states") || "{}",
  );
  Object.entries(cardStates).forEach(([cardId, isCollapsed]) => {
    if (isCollapsed) {
      const card = document.querySelector(
        `.card-collapsible[data-card="${cardId}"]`,
      );
      if (card) card.classList.add("collapsed");
    }
  });
}

// Expose functions to global scope for inline event handlers
window.toggleFilterGroup = toggleFilterGroup;
window.toggleCard = toggleCard;
