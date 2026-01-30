// GLL Viewer - WebAssembly Application
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
  buildFrdContent,
  buildCsvFilterContent,
  buildXgfbContent,
} from "./modules/exporters.js";

if (window) {
  window.THREE = Object.assign({}, THREE, { OrbitControls });
}

let wasmReady = false;
let currentData = null;
let currentFileBytes = null;
let chart = null;
let sourceResponseCharts = new Map();
let filterGroupResponseCharts = new Map();
let sourceResponseChartInitialized = new Set();
let filterGroupChartInitialized = new Set();
let combinedChart = null;
let combinedChartInitialized = false;
let responseChartInitialized = false;
let combinedListenersBound = false;
let activeConfig = null; // { elements: [{ box_type_key, position: {x,y,z}, angles: {x,y,z}, gain }] }

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

const visualization = createVisualizationController({
  getCurrentData: () => currentData,
  formatFrequency,
  formatAngle,
  computePolarSlices,
  getBalloonGrid,
  getResponseWithSymmetry,
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
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function applyTheme(mode) {
  const theme = mode === "auto" ? getSystemTheme() : mode;
  if (theme === "dark") {
    document.documentElement.setAttribute("data-theme", "dark");
  } else {
    document.documentElement.removeAttribute("data-theme");
  }
  geometry?.updateTheme?.();
}

function updateThemeToggleButton() {
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
  const currentIndex = THEME_MODES.indexOf(currentThemeMode);
  const nextIndex = (currentIndex + 1) % THEME_MODES.length;
  currentThemeMode = THEME_MODES[nextIndex];

  applyTheme(currentThemeMode);
  updateThemeToggleButton();
  localStorage.setItem(THEME_KEY, currentThemeMode);
}

function initTheme() {
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
  initTheme();
  initDOMElements();
  await initWasm();
  setupEventListeners();
  restoreCardStates();
});

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

  const globalSource = document.getElementById("global-source");
  if (globalSource) {
    globalSource.addEventListener("change", handleGlobalSourceChange);
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
  const globalSlider = document.getElementById("global-frequency-slider");
  if (globalSlider) {
    globalSlider.addEventListener(
      "input",
      visualization.handleGlobalSliderInput,
    );
  }
  const globalNormalize = document.getElementById("global-normalize");
  if (globalNormalize) {
    globalNormalize.addEventListener("change", () => {
      visualization.updatePolarChart();
      visualization.updateBalloonVisualization();
    });
  }
  const polarFrequency = document.getElementById("polar-frequency");
  if (polarFrequency) {
    polarFrequency.addEventListener("change", visualization.updatePolarChart);
  }
  const balloonSource = document.getElementById("balloon-source");
  if (balloonSource) {
    balloonSource.addEventListener(
      "change",
      visualization.updateBalloonOptions,
    );
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
  const balloonAutorotate = document.getElementById("balloon-autorotate");
  if (balloonAutorotate) {
    balloonAutorotate.addEventListener(
      "change",
      visualization.handleBalloonAutorotateToggle,
    );
  }
  window.addEventListener("resize", () => geometry.handleGeometryResize());
}

function handleDragOver(e) {
  e.preventDefault();
  dropZone.classList.add("drag-over");
}

function handleDragLeave(e) {
  e.preventDefault();
  dropZone.classList.remove("drag-over");
}

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
  filterGroupResponseCharts.forEach((localChart) => localChart.destroy());
  filterGroupResponseCharts = new Map();
  sourceResponseChartInitialized = new Set();
  filterGroupChartInitialized = new Set();
  resetFilterChart();
  visualization.resetVisualization();
  geometry.resetGeometry();
  responseChartInitialized = false;
  combinedChartInitialized = false;
  activeConfig = null;
  updateConfigEditorHint("");
  results.classList.add("hidden");
  loading.classList.add("hidden");
  error.classList.add("hidden");
  dropZone.classList.remove("hidden");
  fileInput.value = "";
}

function displayResults() {
  loading.classList.add("hidden");
  dropZone.classList.add("hidden");
  error.classList.add("hidden");
  results.classList.remove("hidden");

  displayOverview();
  displaySources();
  displayConfig();
  displayConfigurations();
  displayResources();
  setupConfigEditor();
  setupGlobalSourceControls();
  setupCombinedResponseControls();
  setupPolarControls();
  setupBalloonControls();
  setupGeometryControls();

  // Switch to overview tab
  switchTab("overview");
}

function handleGlobalSourceChange(e) {
  const value = e?.target?.value;
  if (value === undefined || value === null) {
    return;
  }
  syncSourceSelectors(value);
  visualization.updatePolarOptions();
  visualization.updateBalloonOptions();
}

function setupGlobalSourceControls() {
  const globalSelect = document.getElementById("global-source");
  if (!globalSelect) {
    return;
  }

  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );

  if (!sourcesWithResponses.length) {
    globalSelect.innerHTML =
      '<option value="">No response data available</option>';
    globalSelect.disabled = true;
    return;
  }

  globalSelect.disabled = false;
  globalSelect.innerHTML = sourcesWithResponses
    .map(
      (src, i) =>
        `<option value="${i}">${escapeHtml(src.definition?.label || src.key)}</option>`,
    )
    .join("");
  syncSourceSelectors(globalSelect.value);
}

function syncSourceSelectors(value) {
  const balloonSelect = document.getElementById("balloon-source");
  if (balloonSelect) {
    balloonSelect.value = String(value);
  }
}

function switchTab(tabName) {
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
    updateCombinedResponseChart();
    visualization.updatePolarChart();
    visualization.updateBalloonVisualization();
    visualization.handleBalloonResize();
  }
  if (tabName === "geometry" && currentData) {
    requestAnimationFrame(() => {
      geometry.initInlineViewers();
      geometry.handleGeometryResize();
    });
  }
}

function displayOverview() {
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
  const sourcesList = document.getElementById("sources-list");
  const sources = currentData.database?.source_definitions || [];
  const placementsByDef = buildSourcePlacementsMap(
    currentData.database?.box_types || [],
  );

  if (sources.length === 0) {
    sourcesList.innerHTML =
      '<div class="empty-state">No source definitions found</div>';
    return;
  }

  sourcesList.innerHTML = sources
    .map((src, sourceIndex) => {
      const def = src.definition || {};
      const balloon = def.balloon_data;
      const placements = placementsByDef.get(src.key) || [];
      const responseCount = src.responses?.length || 0;
      const placementHtml = placements.length
        ? `
                    <div class="source-detail">
                        <strong>Placements:</strong>
                        <div class="source-placement-list">
                            ${placements
                              .map((placement) => {
                                const source = placement.source || {};
                                const boxLabel =
                                  placement.box?.label || "Unknown";
                                const boxKey = placement.box?.key || "";
                                const sourceLabel = source.label || "Source";
                                const sourceKey = source.key || "";
                                return `
                                    <div class="source-placement">
                                        <div class="source-placement-header">
                                            Box: ${escapeHtml(boxLabel)}${boxKey ? ` (${escapeHtml(boxKey)})` : ""}
                                        </div>
                                        <div class="source-placement-detail">
                                            Source: ${escapeHtml(sourceLabel)}${sourceKey ? ` (${escapeHtml(sourceKey)})` : ""}
                                        </div>
                                        <div class="source-placement-detail">
                                            Position: ${formatPosition(source.position)} mm
                                        </div>
                                        <div class="source-placement-detail">
                                            Angles: H ${formatAngleDegrees(source.angles?.x)}, V ${formatAngleDegrees(source.angles?.y)}, R ${formatAngleDegrees(source.angles?.z)}
                                        </div>
                                    </div>
                                `;
                              })
                              .join("")}
                        </div>
                    </div>
                `
        : "";
      const responseOptions = responseCount
        ? Array.from({ length: responseCount }, (_, i) => {
            const angle = computeResponseAngles(src, i);
            const angleLabel = angle
              ? ` • Az ${formatAngle(angle.meridianDeg)}° / Off ${formatAngle(angle.parallelDeg)}°`
              : "";
            return `<option value="${i}">Response ${i + 1}${angleLabel}</option>`;
          }).join("")
        : "";
      const responseSection = responseCount
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
                  <div class="source-details">
                    <div class="source-detail">
                        <strong>Bandwidth:</strong>
                        ${formatFrequency(def.nominal_bandwidth_from)} - ${formatFrequency(def.nominal_bandwidth_to)}
                    </div>
                    <div class="source-detail">
                        <strong>Data Type:</strong>
                        ${formatDataType(def.data_type)}
                    </div>
                    ${
                      balloon
                        ? `
                    <div class="source-detail">
                        <strong>Responses:</strong>
                        ${responseCount}
                    </div>
                    <div class="source-detail">
                        <strong>Resolution:</strong>
                        ${balloon.angular_resolution?.meridian_step || 0}° × ${balloon.angular_resolution?.parallel_step || 0}°
                    </div>
                    `
                        : ""
                    }
                    ${placementHtml}
                  </div>
                  ${responseSection}
                </div>
            </div>
        `;
    })
    .join("");

  wireSourceResponseControls();
}

function wireSourceResponseControls() {
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

function renderSourceResponseChart(sourceIndex) {
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

  const shouldAnimate = !sourceResponseChartInitialized.has(sourceIndex);

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
        x: buildLogFrequencyScale(
          frequencyData.minFrequency,
          frequencyData.maxFrequency,
          "Frequency",
        ),
        y: {
          type: "linear",
          display: true,
          position: "left",
          title: {
            display: true,
            text: "Level (dB)",
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
  if (!meta) return;
  meta.innerHTML = buildResponseMetaHtml(source, responseIndex);
}

function handleSourceResponseAngleInput(sourceIndex) {
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

  const azimuthDeg = normalizeAzimuthForGrid(
    Number(azSlider.value),
    ang,
  );
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

function buildSourcePlacementsMap(boxTypes) {
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

  // Auto-select default configuration
  autoSelectDefaultConfig();

  // Demo configuration if no setup data exists
  ensureDemoConfiguration();
}

function displayConnectors(db) {
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
  return (rad * 180) / Math.PI;
}

function toRad(deg) {
  return (deg * Math.PI) / 180;
}

function setupConfigEditor() {
  const addBtn = document.getElementById("config-add-element");
  const clearBtn = document.getElementById("config-clear");
  const applyBtn = document.getElementById("config-apply");

  if (addBtn) {
    addBtn.onclick = () => addConfigEditorRow();
  }
  if (clearBtn) {
    clearBtn.onclick = () => {
      document.getElementById("config-editor-body").innerHTML = "";
      activeConfig = null;
      updateConfigEditorHint("Add elements to build a configuration.");
    };
  }
  if (applyBtn) {
    applyBtn.onclick = () => applyConfigFromEditor();
  }
}

function getBoxTypeOptions() {
  const boxTypes = currentData?.database?.box_types || [];
  return boxTypes
    .map(
      (bt) =>
        `<option value="${escapeHtml(bt.key)}">${escapeHtml(bt.label || bt.key)}</option>`,
    )
    .join("");
}

function addConfigEditorRow(boxTypeKey, pos, angles, gain) {
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
    <td><input type="number" class="cfg-x" value="${pos?.x ?? 0}" step="1"></td>
    <td><input type="number" class="cfg-y" value="${pos?.y ?? 0}" step="1"></td>
    <td><input type="number" class="cfg-z" value="${pos?.z ?? 0}" step="1"></td>
    <td><input type="number" class="cfg-h" value="${formatNumber(toDeg(angles?.x ?? 0))}" step="0.5"></td>
    <td><input type="number" class="cfg-v" value="${formatNumber(toDeg(angles?.y ?? 0))}" step="0.5"></td>
    <td><input type="number" class="cfg-r" value="${formatNumber(toDeg(angles?.z ?? 0))}" step="0.5"></td>
    <td><input type="number" class="cfg-gain" value="${gain ?? 0}" step="0.5"></td>
    <td><button class="btn-remove" onclick="this.closest('tr').remove()">X</button></td>
  `;
  tbody.appendChild(row);
}

// Load a cluster setup into the editor
window.loadClusterSetupToEditor = function (clusterIndex) {
  const clusterSetups = currentData?.database?.cluster_setups || [];
  const cs = clusterSetups[clusterIndex];
  if (!cs) return;

  const tbody = document.getElementById("config-editor-body");
  if (tbody) tbody.innerHTML = "";
  updateConfigEditorHint("");

  for (const box of cs.setup?.boxes || []) {
    addConfigEditorRow(box.box_type_key, box.position, box.angles, 0);
  }
};

function applyConfigFromEditor() {
  const tbody = document.getElementById("config-editor-body");
  if (!tbody) return;

  const rows = tbody.querySelectorAll("tr");
  const elements = [];
  for (const row of rows) {
    const boxTypeKey = row.querySelector(".cfg-box-type")?.value;
    const x = parseFloat(row.querySelector(".cfg-x")?.value) || 0;
    const y = parseFloat(row.querySelector(".cfg-y")?.value) || 0;
    const z = parseFloat(row.querySelector(".cfg-z")?.value) || 0;
    const h = parseFloat(row.querySelector(".cfg-h")?.value) || 0;
    const v = parseFloat(row.querySelector(".cfg-v")?.value) || 0;
    const r = parseFloat(row.querySelector(".cfg-r")?.value) || 0;
    const gain = parseFloat(row.querySelector(".cfg-gain")?.value) || 0;
    if (boxTypeKey) {
      elements.push({
        box_type_key: boxTypeKey,
        position: { x, y, z },
        angles: { x: toRad(h), y: toRad(v), z: toRad(r) },
        gain,
      });
    }
  }

  if (elements.length === 0) {
    activeConfig = null;
    updateConfigEditorHint("Add elements to build a configuration.");
  } else {
    activeConfig = { elements, label: `Config (${elements.length} boxes)` };
    updateConfigEditorHint("");
  }

  updateCombinedResponseChart();
}

function autoSelectDefaultConfig() {
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
  // For LineArray (type 0), full support needs Frame/Connector parsing (not yet implemented)
}

function ensureDemoConfiguration() {
  const db = currentData?.database;
  if (!db) return;

  const clusterSetups = db.cluster_setups || [];
  const connectors = db.connectors || [];
  if (clusterSetups.length || connectors.length) {
    return;
  }

  if (activeConfig?.elements?.length) {
    return;
  }

  const boxTypes = db.box_types || [];
  if (!boxTypes.length) {
    updateConfigEditorHint("No box types available for demo configuration.");
    return;
  }

  const demoBox = boxTypes.find((box) => {
    const placements = box?.source_placements || [];
    const sources = box?.sources || [];
    return placements.length > 0 || sources.length > 0;
  });

  if (!demoBox) {
    updateConfigEditorHint("No box types with sources available for demo.");
    return;
  }

  const spacingMm = 500;
  const splayDeg = 5;
  const elements = Array.from({ length: 4 }, (_, i) => ({
    box_type_key: demoBox.key,
    position: { x: 0, y: i * spacingMm, z: 0 },
    angles: { x: 0, y: toRad(-splayDeg * i), z: 0 },
    gain: 0,
  }));

  activeConfig = {
    elements,
    label: "Demo Config (4 boxes)",
    isDemo: true,
  };

  if (buildElementsFromConfig(activeConfig).length === 0) {
    activeConfig = null;
    updateConfigEditorHint(
      "Demo configuration could not be generated from available sources.",
    );
    return;
  }

  const tbody = document.getElementById("config-editor-body");
  if (tbody) {
    tbody.innerHTML = "";
    elements.forEach((elem) =>
      addConfigEditorRow(
        elem.box_type_key,
        elem.position,
        elem.angles,
        elem.gain,
      ),
    );
  }

  updateConfigEditorHint(
    "Demo configuration created (no setup data found in file).",
  );
}

function updateConfigEditorHint(message) {
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
        // Combine box-level config position with source placement position
        allElements.push({
          source_key: sourceKey,
          position: {
            x:
              (Number(elem.position?.x) || 0) / 1000 +
              (Number(pPos.x) || 0) / 1000,
            y:
              (Number(elem.position?.y) || 0) / 1000 +
              (Number(pPos.y) || 0) / 1000,
            z:
              (Number(elem.position?.z) || 0) / 1000 +
              (Number(pPos.z) || 0) / 1000,
          },
          angles: {
            x: (Number(elem.angles?.x) || 0) + toRadiansMaybe(pAngles.x),
            y: (Number(elem.angles?.y) || 0) + toRadiansMaybe(pAngles.y),
            z: (Number(elem.angles?.z) || 0) + toRadiansMaybe(pAngles.z),
          },
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
          gain: elem.gain || 0,
        });
      }
    }
  }

  return allElements;
}

function wireFilterGroupResponses() {
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
    const message = response.error || "Failed to compute filter response";
    updateFilterGroupResponseMeta(groupIndex, message);
    setFilterGroupResponsePlaceholder(groupIndex, message);
    destroyFilterGroupChart(groupIndex);
    return;
  }

  if (!response.frequencies?.length || !response.level?.length) {
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

  const scales = {
    x: buildLogFrequencyScale(
      frequencyData.minFrequency,
      frequencyData.maxFrequency,
      "Frequency",
    ),
    y: {
      type: "linear",
      display: true,
      position: "left",
      title: {
        display: true,
        text: "Level (dB)",
      },
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
  const existingChart = filterGroupResponseCharts.get(groupIndex);
  if (existingChart) {
    existingChart.destroy();
    filterGroupResponseCharts.delete(groupIndex);
  }
}

function buildFilterGroupMetaChips(group, filterDef, response) {
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
  if (!box) {
    return "";
  }

  const key = box.key ? escapeHtml(box.key) : "-";
  const sources = Array.isArray(box.sources) && box.sources.length > 0
    ? box.sources.map((src) => escapeHtml(src)).join(", ")
    : "-";
  const weightValue = formatNumber(box.weight, 2);
  const weight = weightValue === "-" ? "-" : `${weightValue} kg`;
  const vAngleValue = formatNumber(box.vertical_opening_angle, 1);
  const vAngle = vAngleValue === "-" ? "-" : `${vAngleValue}°`;
  const hAngleValue = formatNumber(box.horizontal_opening_angle, 1);
  const hAngle = hAngleValue === "-" ? "-" : `${hAngleValue}°`;

  return `
        <div class="config-item-detail">
            Key: ${key} • Sources: ${sources} • Weight: ${weight} • Vertical Opening Angle: ${vAngle} • Horizontal Opening Angle: ${hAngle}
        </div>
    `;
}

function formatGeometryActions(kind, index, geometry, label) {
  if (!hasGeometryData(geometry)) {
    return "";
  }
  const filename = sanitizeFilename(label || `${kind}-${index + 1}`);
  return `
        <div class="config-item-actions">
            <div class="dropdown-container">
                <button class="btn-download btn-geom-export" data-geom-type="${kind}" data-geom-index="${index}" data-geom-filename="${escapeHtml(filename)}">
                    Download <span class="dropdown-icon">▼</span>
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
  if (!hasGeometryData(caseGeometry)) {
    return "";
  }
  const id = `geometry-inline-${kind}-${index}`;
  return `
        <div class="inline-geometry-viewer" id="${id}" data-geom-kind="${kind}" data-geom-index="${index}">
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
  document.querySelectorAll(".dropdown-item").forEach((item) => {
    if (item.dataset.bound === "true") {
      return;
    }
    item.dataset.bound = "true";
    item.addEventListener("click", (e) => {
      e.stopPropagation();
      const format = item.dataset.format;
      const button = item
        .closest(".dropdown-container")
        .querySelector(".btn-geom-export");
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

      const label = dataItem?.label || dataItem?.key || `${kind}-${index + 1}`;
      const filename = datasetFilename || sanitizeFilename(label);

      // Generate and download based on format
      if (format === "xed") {
        const content = buildXedContent(geometry, { units: "m", precision: 6 });
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
    resourcesTab.classList.toggle("hidden", !hasResources);
    resourcesContent.classList.toggle("hidden", !hasResources);
    if (!hasResources && resourcesTab.classList.contains("active")) {
      switchTab("overview");
    }
  }

  if (!hasResources) {
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
  // Remove Windows path separators and get base name
  return path.replace(/\\/g, "/").split("/").pop() || path;
}

function setupResponseControls() {
  const sourceSelect = document.getElementById("response-source");
  const sources = currentData.database?.source_definitions || [];

  // Filter sources that have responses
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );

  if (sourcesWithResponses.length === 0) {
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

  const globalSelect = document.getElementById("global-source");
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
  visualization.updatePolarOptions();
}

function setupBalloonControls() {
  const rangeValue = document.getElementById("balloon-range");
  const scaleValue = document.getElementById("balloon-scale");
  if (rangeValue) {
    visualization.handleBalloonRangeInput({ target: rangeValue });
  }
  if (scaleValue) {
    visualization.handleBalloonScaleInput({ target: scaleValue });
  }
  visualization.updateBalloonSourceOptions();
  visualization.updateBalloonOptions();
}

function setupGeometryControls() {
  geometry.resetGeometry();
  // Inline viewers are initialized when the geometry tab is shown
}

function setupCombinedResponseControls() {
  const boxSelect = document.getElementById("combined-box");
  const groupSelect = document.getElementById("combined-filter-group");
  const filterSelect = document.getElementById("combined-filter");
  const gainInput = document.getElementById("combined-gain");
  const distanceInput = document.getElementById("combined-distance");
  const phaseSelect = document.getElementById("combined-phase-mode");
  if (!boxSelect || !groupSelect || !filterSelect) {
    return;
  }

  if (!combinedListenersBound) {
    boxSelect.addEventListener("change", updateCombinedResponseChart);
    groupSelect.addEventListener("change", updateCombinedFilterOptions);
    filterSelect.addEventListener("change", updateCombinedResponseChart);
    gainInput?.addEventListener("input", updateCombinedResponseChart);
    distanceInput?.addEventListener("input", updateCombinedResponseChart);
    phaseSelect?.addEventListener("change", updateCombinedResponseChart);
    combinedListenersBound = true;
  }

  const boxTypes = currentData?.database?.box_types || [];
  if (!boxTypes.length) {
    boxSelect.innerHTML = '<option value="">No box types</option>';
    boxSelect.disabled = true;
  } else {
    boxSelect.disabled = false;
    boxSelect.innerHTML = boxTypes
      .map(
        (box, i) =>
          `<option value="${i}">${escapeHtml(box.label || box.key || `Box ${i + 1}`)}</option>`,
      )
      .join("");
  }

  const groups = currentData?.database?.filter_groups || [];
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
  const sourceSelect = document.getElementById("response-source");
  const indexSelect = document.getElementById("response-index");
  const onAxisToggle = document.getElementById("response-use-onaxis");
  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );

  const sourceIndex = parseInt(sourceSelect.value);
  const globalSelect = document.getElementById("global-source");
  if (globalSelect && !Number.isNaN(sourceIndex)) {
    globalSelect.value = String(sourceIndex);
  }
  if (isNaN(sourceIndex) || sourceIndex >= sourcesWithResponses.length) {
    indexSelect.innerHTML = "";
    updateResponseAngleControls(null, null);
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const responseCount = source.responses?.length || 0;

  indexSelect.innerHTML = Array.from({ length: responseCount }, (_, i) => {
    const angle = computeResponseAngles(source, i);
    const angleLabel = angle
      ? ` • Az ${formatAngle(angle.meridianDeg)}° / Off ${formatAngle(angle.parallelDeg)}°`
      : "";
    return `<option value="${i}">Response ${i + 1}${angleLabel}</option>`;
  }).join("");

  if (onAxisToggle) {
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
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const response = source?.responses?.[respIndex];

  if (!response) {
    updateResponseAngleControls(null, null);
    return;
  }

  updateResponseAngleControls(source, respIndex);
  updateResponseMeta(source, respIndex);

  // Update chart
  const ctx = document.getElementById("response-chart").getContext("2d");

  if (chart) {
    chart.destroy();
  }

  const onAxis = source?.definition?.on_axis_spectrum;
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
    return;
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
        x: buildLogFrequencyScale(
          frequencyData.minFrequency,
          frequencyData.maxFrequency,
          "Frequency",
        ),
        y: {
          type: "linear",
          display: true,
          position: "left",
          title: {
            display: true,
            text: "Level (dB)",
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
  const boxSelect = document.getElementById("combined-box");
  const groupSelect = document.getElementById("combined-filter-group");
  const filterSelect = document.getElementById("combined-filter");
  const gainInput = document.getElementById("combined-gain");
  const distanceInput = document.getElementById("combined-distance");
  const phaseSelect = document.getElementById("combined-phase-mode");
  const meta = document.getElementById("combined-response-meta");
  const ctx = document
    .getElementById("combined-response-chart")
    ?.getContext("2d");

  if (!boxSelect || !ctx || !meta) {
    return;
  }

  if (!currentFileBytes || typeof computeArrayResponse !== "function") {
    meta.innerHTML =
      '<span class="chip">Array response helper not available</span>';
    destroyCombinedChart();
    return;
  }

  const boxTypes = currentData?.database?.box_types || [];
  let elements;
  let box;

  if (activeConfig) {
    // Use active configuration from editor/cluster setup
    elements = buildElementsFromConfig(activeConfig);
    box = null;
  } else {
    const boxIndex = parseInt(boxSelect.value);
    if (Number.isNaN(boxIndex) || boxIndex >= boxTypes.length) {
      meta.innerHTML = '<span class="chip">Select a box type</span>';
      destroyCombinedChart();
      return;
    }
    box = boxTypes[boxIndex];
    elements = buildCombinedElements(box);
  }

  if (elements.length) {
    const validSources = new Set(
      (currentData?.database?.source_definitions || []).map((s) => s.key),
    );
    elements = elements.filter((elem) => validSources.has(elem.source_key));
  }

  if (!elements.length) {
    meta.innerHTML =
      '<span class="chip">No valid sources found for this configuration</span>';
    destroyCombinedChart();
    return;
  }

  const gainOffset = parseFloat(gainInput?.value) || 0;
  const receiverDistance = parseFloat(distanceInput?.value) || 1;
  const arrayPayload = JSON.stringify({
    elements,
    receiver: { x: 0, y: 0, z: Math.max(receiverDistance, 0.1) },
    air_props: {
      temperature: 20,
      humidity: 0.5,
      speed: 0,
      air_atten_on: false,
    },
  });

  let arrayResponse;
  try {
    const responseJSON = computeArrayResponse(currentFileBytes, arrayPayload);
    arrayResponse = JSON.parse(responseJSON);
  } catch (err) {
    meta.innerHTML =
      '<span class="chip">Failed to compute array response</span>';
    destroyCombinedChart();
    return;
  }

  if (!arrayResponse.success) {
    meta.innerHTML = `<span class="chip">${escapeHtml(arrayResponse.error || "Failed to compute array response")}</span>`;
    destroyCombinedChart();
    return;
  }

  let combinedLevel = arrayResponse.level?.slice() || [];
  let combinedPhase = arrayResponse.phase?.slice() || [];
  let filterMessage = null;
  let filterLabel = null;
  let groupLabel = null;
  const groupIndex = parseInt(groupSelect?.value);
  const filterIndex = parseInt(filterSelect?.value);

  if (!Number.isNaN(groupIndex) && !filterSelect?.disabled) {
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

  if (Number.isFinite(gainOffset) && gainOffset !== 0) {
    combinedLevel = combinedLevel.map((value) => value + gainOffset);
  }

  if (phaseSelect) {
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
    combinedLevel,
  );
  if (!frequencyData) {
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
    phaseSeries = getPhaseSeries(
      phaseMode,
      arrayResponse.frequencies,
      combinedPhase,
      unwrapPhase(combinedPhase),
    );
  }

  if (combinedChart) {
    combinedChart.destroy();
  }

  combinedChart = new Chart(ctx, {
    type: "line",
    data: {
      datasets: [
        {
          label: "Level (dB)",
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
        x: buildLogFrequencyScale(
          frequencyData.minFrequency,
          frequencyData.maxFrequency,
          "Frequency",
        ),
        y: {
          type: "linear",
          display: true,
          position: "left",
          title: {
            display: true,
            text: "Level (dB)",
          },
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
  const configBox = activeConfig
    ? { label: activeConfig.label || `Config (${activeConfig.elements.length} boxes)` }
    : box;
  updateCombinedResponseMeta(
    meta,
    configBox,
    elements.length,
    gainOffset,
    receiverDistance,
    groupLabel,
    filterLabel,
    filterMessage,
  );
}

function computeCombinedFilterResponse(groupIndex, filterIndex) {
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
  groupLabel,
  filterLabel,
  filterMessage,
) {
  const chips = [];
  if (box) {
    chips.push(
      `<span class="chip">${escapeHtml(box.label || box.key || "Box")}</span>`,
    );
  }
  chips.push(`<span class="chip">${elementCount} sources</span>`);
  chips.push(
    `<span class="chip">Receiver ${receiverDistance.toFixed(1)} m</span>`,
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
  if (combinedChart) {
    combinedChart.destroy();
    combinedChart = null;
  }
}

function applyDelayToPhase(phaseValues, frequencies, delaySeconds) {
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
  const azimuthDeg = normalizeAzimuthForGrid(
    Number(azSlider.value),
    ang,
  );
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

// Get response at given azimuth/parallel angles, applying symmetry mirroring
function getResponseWithSymmetry(source, grid, azimuthDeg, parallelDeg) {
  const responses = source?.responses || [];
  const ang = source?.definition?.balloon_data?.angular_resolution;

  if (!responses.length || !ang || !grid) {
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

  // Convert angles to grid indices
  const meridianIdx = Math.round(lookupAzimuth / ang.meridian_step);
  const parallelIdx = Math.round(lookupParallel / ang.parallel_step);

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
    return ((azimuthDeg - 90) % 360 + 360) % 360;
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
  if (!grid || !grid.meridianCount || !grid.parallelCount) {
    return null;
  }

  let localMeridianIdx = meridianIdx;
  let localParallelIdx = null;

  if (overrides) {
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
    renderSourceResponseChart(idx);
  } else {
    content.style.display = "none";
    toggle.textContent = "▶";
    card.classList.remove("expanded");
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
