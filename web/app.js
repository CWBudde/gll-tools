// GLL Viewer - WebAssembly Application

let wasmReady = false;
let currentData = null;
let chart = null;
let polarChart = null;
let balloonRenderer = null;
let balloonScene = null;
let balloonCamera = null;
let balloonGroup = null;
let balloonMesh = null;
let balloonFrameId = null;
let balloonResizeBound = false;
let balloonPointerState = null;
const polarSliderMax = 1000;
let responseChartInitialized = false;
let polarChartInitialized = false;

// DOM Elements (initialized in DOMContentLoaded)
let dropZone = null;
let fileInput = null;
let loading = null;
let error = null;
let errorMessage = null;
let results = null;
let fileName = null;
let clearBtn = null;

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

// Initialize
document.addEventListener("DOMContentLoaded", async () => {
  initDOMElements();
  await initWasm();
  setupEventListeners();
  restoreCardStates();
});

function setupEventListeners() {
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
  const polarPlane = document.getElementById("polar-plane");
  if (polarPlane) {
    polarPlane.addEventListener("change", updatePolarChart);
  }
  const polarFrequency = document.getElementById("polar-frequency");
  if (polarFrequency) {
    polarFrequency.addEventListener("change", updatePolarChart);
  }
  const polarSlider = document.getElementById("polar-frequency-slider");
  if (polarSlider) {
    polarSlider.addEventListener("input", handlePolarSliderInput);
  }
  const balloonSource = document.getElementById("balloon-source");
  if (balloonSource) {
    balloonSource.addEventListener("change", updateBalloonOptions);
  }
  const balloonFrequency = document.getElementById("balloon-frequency");
  if (balloonFrequency) {
    balloonFrequency.addEventListener("change", updateBalloonVisualization);
  }
  const balloonSlider = document.getElementById("balloon-frequency-slider");
  if (balloonSlider) {
    balloonSlider.addEventListener("input", handleBalloonSliderInput);
  }
  const balloonRange = document.getElementById("balloon-range");
  if (balloonRange) {
    balloonRange.addEventListener("input", handleBalloonRangeInput);
  }
  const balloonScale = document.getElementById("balloon-scale");
  if (balloonScale) {
    balloonScale.addEventListener("input", handleBalloonScaleInput);
  }
  const balloonWireframe = document.getElementById("balloon-wireframe");
  if (balloonWireframe) {
    balloonWireframe.addEventListener("change", updateBalloonVisualization);
  }
  const balloonAutorotate = document.getElementById("balloon-autorotate");
  if (balloonAutorotate) {
    balloonAutorotate.addEventListener("change", handleBalloonAutorotateToggle);
  }
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
  if (chart) {
    chart.destroy();
    chart = null;
  }
  if (polarChart) {
    polarChart.destroy();
    polarChart = null;
  }
  destroyBalloonScene();
  responseChartInitialized = false;
  polarChartInitialized = false;
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
  displayResources();
  setupGlobalSourceControls();
  setupResponseControls();
  setupPolarControls();
  setupBalloonControls();

  // Switch to overview tab
  switchTab("overview");
}

function handleGlobalSourceChange(e) {
  const value = e?.target?.value;
  if (value === undefined || value === null) {
    return;
  }
  syncSourceSelectors(value);
  updateResponseOptions();
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

  const responseSelect = document.getElementById("response-source");
  const defaultIndex = parseInt(responseSelect?.value);
  if (
    !Number.isNaN(defaultIndex) &&
    defaultIndex < sourcesWithResponses.length
  ) {
    globalSelect.value = String(defaultIndex);
  }

  syncSourceSelectors(globalSelect.value);
}

function syncSourceSelectors(value) {
  const responseSelect = document.getElementById("response-source");
  if (responseSelect) {
    responseSelect.value = String(value);
  }
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

  // Initialize chart when switching to responses tab
  if (tabName === "responses" && currentData) {
    updateResponseChart();
    updatePolarChart();
    updateBalloonVisualization();
    handleBalloonResize();
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

  if (sources.length === 0) {
    sourcesList.innerHTML =
      '<div class="empty-state">No source definitions found</div>';
    return;
  }

  sourcesList.innerHTML = sources
    .map((src) => {
      const def = src.definition || {};
      const balloon = def.balloon_data;
      return `
            <div class="source-card">
                <div class="source-header">
                    <span class="source-label">${escapeHtml(def.label || "Unknown")}</span>
                    <span class="source-key">${escapeHtml(src.key)}</span>
                </div>
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
                        ${balloon.response_count || 0}
                    </div>
                    <div class="source-detail">
                        <strong>Resolution:</strong>
                        ${balloon.angular_resolution?.meridian_step || 0}° × ${balloon.angular_resolution?.parallel_step || 0}°
                    </div>
                    `
                        : ""
                    }
                </div>
            </div>
        `;
    })
    .join("");
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
                <div class="config-item-header">${escapeHtml(box.label)}</div>
                <div class="config-item-detail">Key: ${escapeHtml(box.key)}</div>
                ${formatGeometryDetail(box.case_geometry)}
                ${formatGeometryActions("box", index, box.case_geometry)}
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
                <div class="config-item-header">${escapeHtml(frame.label)}</div>
                <div class="config-item-detail">Key: ${escapeHtml(frame.key)}</div>
                ${formatGeometryDetail(frame.case_geometry)}
                ${formatGeometryActions("frame", index, frame.case_geometry)}
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
  document.getElementById("box-types-count").textContent =
    boxTypes.length > 0 ? boxTypes.length : "";
  document.getElementById("frames-count").textContent =
    frames.length > 0 ? frames.length : "";
  document.getElementById("filter-groups-count").textContent =
    filterGroups.length > 0 ? filterGroups.length : "";
  document.getElementById("limits-count").textContent =
    limits.length > 0 ? limits.length : "";
  document.getElementById("warnings-count").textContent =
    warnings.length > 0 ? warnings.length : "";
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

function formatGeometryActions(kind, index, geometry) {
  if (!hasGeometryData(geometry)) {
    return "";
  }
  return `
        <div class="config-item-actions">
            <button class="btn-download btn-xed" data-geom-type="${kind}" data-geom-index="${index}">
                Download .xed
            </button>
        </div>
    `;
}

function wireGeometryDownloads() {
  document.querySelectorAll(".btn-xed").forEach((button) => {
    if (button.dataset.bound === "true") {
      return;
    }
    button.dataset.bound = "true";
    button.addEventListener("click", () => {
      const kind = button.dataset.geomType;
      const index = Number(button.dataset.geomIndex);
      const db = currentData?.database;
      const item =
        kind === "frame" ? db?.frames?.[index] : db?.box_types?.[index];
      const geometry = item?.case_geometry;
      if (!hasGeometryData(geometry)) {
        return;
      }
      const label = item?.label || item?.key || `${kind}-${index + 1}`;
      const content = buildXedContent(geometry, { units: "m", precision: 6 });
      downloadTextFile(`${sanitizeFilename(label)}.xed`, content);
    });
  });
}

function hasGeometryData(geometry) {
  if (!geometry) return false;
  const hasEdges = Array.isArray(geometry.edges) && geometry.edges.length > 0;
  const hasVertices =
    Array.isArray(geometry.vertices) && geometry.vertices.length > 1;
  return hasEdges || hasVertices;
}

function buildXedContent(geometry, options) {
  const units = (options?.units || "m").toLowerCase();
  const precision = Number.isFinite(options?.precision) ? options.precision : 6;
  const headerUnits = units === "ft" ? "ft" : "m";
  const scale = units === "ft" ? 1 / (0.3048 * 1000) : 1 / 1000;

  const lines = ["FILE XED", "VERSION 4.0", `UNITS ${headerUnits}`];

  const vertices = Array.isArray(geometry.vertices) ? geometry.vertices : [];
  const edges = Array.isArray(geometry.edges) ? geometry.edges : [];
  const edgePairs =
    edges.length > 0
      ? edges.map((edge) => [edge.v1, edge.v2])
      : buildSequentialEdgePairs(vertices.length);

  for (const [v1, v2] of edgePairs) {
    const p1 = resolveGeometryVertex(geometry, vertices, v1, scale);
    const p2 = resolveGeometryVertex(geometry, vertices, v2, scale);
    if (!p1 || !p2) {
      continue;
    }
    lines.push(formatXedLine(p1, precision));
    lines.push(formatXedLine(p2, precision));
  }

  return lines.join("\n") + "\n";
}

function buildSequentialEdgePairs(vertexCount) {
  const pairs = [];
  for (let i = 0; i + 1 < vertexCount; i += 2) {
    pairs.push([i + 1, i + 2]);
  }
  return pairs;
}

function resolveGeometryVertex(geometry, vertices, index, scale) {
  if (!index || !Number.isFinite(index)) {
    return null;
  }
  const rawIndex = Math.trunc(index);
  const absIndex = Math.abs(rawIndex);
  if (absIndex < 1 || absIndex > vertices.length) {
    return null;
  }
  const vertex = vertices[absIndex - 1];
  if (!vertex) {
    return null;
  }
  let x = Number(vertex.x);
  let y = Number(vertex.y);
  let z = Number(vertex.z);
  if (![x, y, z].every(Number.isFinite)) {
    return null;
  }
  if (rawIndex < 0 && geometry.is_symmetric) {
    const axis = Number(geometry.symmetry_axis) || 0;
    x = 2 * axis - x;
  }
  return {
    x: x * scale,
    y: y * scale,
    z: z * scale,
  };
}

function formatXedLine(point, precision) {
  return `${formatXedNumber(point.x, precision)} ${formatXedNumber(point.y, precision)} ${formatXedNumber(point.z, precision)}`;
}

function formatXedNumber(value, precision) {
  const rounded = Number(value).toFixed(precision);
  return rounded.replace(/\.?0+$/, "");
}

function downloadTextFile(filename, content) {
  const blob = new Blob([content], { type: "text/plain" });
  const url = URL.createObjectURL(blob);
  downloadFile(filename, url);
  URL.revokeObjectURL(url);
}

function sanitizeFilename(name) {
  return (
    String(name)
      .replace(/\\s+/g, "_")
      .replace(/[^a-zA-Z0-9_-]/g, "")
      .replace(/_+/g, "_")
      .replace(/^_+|_+$/g, "")
      .toLowerCase() || "geometry"
  );
}

function displayResources() {
  // Include Files (PDFs, documentation, technical drawings)
  const includeFilesList = document.getElementById("include-files-list");
  const includeFiles = currentData.database?.include_files || [];
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
  const dataFiles = currentData.database?.data_files || [];
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

  // Other Embedded Resources (fonts, compressed data)
  const resourcesList = document.getElementById("resources-list");
  const resources = currentData.resources || [];
  // Filter out PNGs that are also in data_files to avoid duplicates
  const dataFileNames = new Set(
    (dataFiles || []).map((f) => cleanFilename(f.filename).toLowerCase()),
  );
  const filteredResources = resources.filter((res) => {
    if (res.type === "PNG" && res.name) {
      return !dataFileNames.has(cleanFilename(res.name).toLowerCase());
    }
    return true;
  });

  if (filteredResources.length === 0) {
    resourcesList.innerHTML =
      '<div class="empty-state">No additional resources found</div>';
  } else {
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
  }
}

function downloadFile(filename, dataUri) {
  const link = document.createElement("a");
  link.href = dataUri;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
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
  updatePolarOptions();
}

function setupBalloonControls() {
  const rangeValue = document.getElementById("balloon-range");
  const scaleValue = document.getElementById("balloon-scale");
  if (rangeValue) {
    handleBalloonRangeInput({ target: rangeValue });
  }
  if (scaleValue) {
    handleBalloonScaleInput({ target: scaleValue });
  }
  updateBalloonSourceOptions();
  updateBalloonOptions();
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
      ? ` • Az ${formatAngle(angle.meridianDeg)}° / El ${formatAngle(angle.parallelDeg)}°`
      : "";
    return `<option value="${i}">Response ${i + 1}${angleLabel}</option>`;
  }).join("");

  if (onAxisToggle) {
    const onAxis = source?.definition?.on_axis_spectrum;
    const hasOnAxis = !!onAxis && Array.isArray(onAxis.level) && onAxis.level.length > 0;
    onAxisToggle.disabled = !hasOnAxis;
    if (!hasOnAxis) {
      onAxisToggle.checked = false;
    }
  }

  updateResponseChart();
  updatePolarOptions();
  updateBalloonOptions();
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

  const phaseMode =
    document.getElementById("response-phase-mode")?.value || "unwrapped";
  const rawPhase = response.phase || [];
  const unwrappedPhase = unwrapPhase(rawPhase);
  const phaseSeries = getPhaseSeries(
    phaseMode,
    response.frequencies,
    rawPhase,
    unwrappedPhase,
  );

  const onAxis = source?.definition?.on_axis_spectrum;
  const onAxisFreqs = buildLogFrequencies(
    onAxis?.definition,
    onAxis?.level?.length,
  );
  const canCombineOnAxis =
    !!onAxisToggle?.checked &&
    onAxis &&
    Array.isArray(onAxis.level) &&
    Array.isArray(onAxisFreqs) &&
    response.frequencies.length === onAxisFreqs.length &&
    response.level.length === onAxis.level.length &&
    frequenciesMatch(response.frequencies, onAxisFreqs);
  const levelSeries = canCombineOnAxis
    ? response.level.map((value, i) => value + onAxis.level[i])
    : response.level;

  const frequencyPoints = response.frequencies.map((f, i) => ({
    x: f,
    y: levelSeries[i],
  }));
  const phasePoints = response.frequencies.map((f, i) => ({
    x: f,
    y: phaseSeries.values[i],
  }));

  const minFrequency = Math.min(...response.frequencies);
  const maxFrequency = Math.max(...response.frequencies);

  chart = new Chart(ctx, {
    type: "line",
    data: {
      datasets: [
        {
          label: canCombineOnAxis
            ? "Level (dB, on-axis + directivity)"
            : "Level (dB)",
          data: frequencyPoints,
          borderColor: "#2563eb",
          backgroundColor: "rgba(37, 99, 235, 0.1)",
          fill: true,
          tension: 0.3,
          pointRadius: 0,
          yAxisID: "y",
        },
        {
          label: phaseSeries.label,
          data: phasePoints,
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
        x: {
          type: "logarithmic",
          title: {
            display: true,
            text: "Frequency",
          },
          ticks: {
            autoSkip: false,
            callback: (value) => {
              const numericValue = Number(value);
              return isPowerOfTen(numericValue)
                ? formatFrequencyShort(numericValue)
                : "";
            },
          },
          min: minFrequency,
          max: maxFrequency,
          afterBuildTicks: (scale) => {
            scale.ticks = buildLogTicks(scale.min, scale.max);
          },
        },
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
      },
    },
  });

  responseChartInitialized = true;
  document.getElementById("response-meta").classList.remove("empty-state");
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
  const azimuthDeg = Number(azSlider.value);
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
      ? ` • Az ${formatAngle(angle.meridianDeg)}° / El ${formatAngle(angle.parallelDeg)}°`
      : "";
    const option = document.createElement("option");
    option.value = String(responseIndex);
    option.dataset.custom = "true";
    option.textContent = `Response ${responseIndex + 1}${angleLabel}`;
    indexSelect.appendChild(option);
  }

  indexSelect.value = String(responseIndex);
}

function updatePolarOptions() {
  const sourceSelect = document.getElementById("response-source");
  const freqSelect = document.getElementById("polar-frequency");
  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );
  const previousIndex = parseInt(freqSelect.value);

  const sourceIndex = parseInt(sourceSelect.value);
  if (isNaN(sourceIndex) || sourceIndex >= sourcesWithResponses.length) {
    freqSelect.innerHTML = "";
    updatePolarSliderState(null);
    document.getElementById("polar-meta").innerHTML =
      '<div class="empty-state">No polar data available</div>';
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const sampleResponse = source?.responses?.[0];
  const frequencies = sampleResponse?.frequencies || [];

  freqSelect.innerHTML = frequencies
    .map((f, i) => `<option value="${i}">${formatFrequency(f)}</option>`)
    .join("");

  const nextIndex =
    !isNaN(previousIndex) && previousIndex < frequencies.length
      ? previousIndex
      : 0;
  freqSelect.value = String(nextIndex);
  updatePolarSliderState(frequencies);
  updatePolarSliderFromIndex(nextIndex, frequencies);

  updatePolarChart();
}

function updateBalloonSourceOptions() {
  const sourceSelect = document.getElementById("balloon-source");
  const sources = currentData?.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );

  if (!sourcesWithResponses.length) {
    sourceSelect.innerHTML =
      '<option value="">No response data available</option>';
    return;
  }

  const globalSource = document.getElementById("global-source");
  const defaultIndex = parseInt(globalSource?.value);

  sourceSelect.innerHTML = sourcesWithResponses
    .map(
      (src, i) =>
        `<option value="${i}">${escapeHtml(src.definition?.label || src.key)}</option>`,
    )
    .join("");

  if (
    !Number.isNaN(defaultIndex) &&
    defaultIndex < sourcesWithResponses.length
  ) {
    sourceSelect.value = String(defaultIndex);
  }
}

function updateBalloonOptions() {
  const sourceSelect = document.getElementById("balloon-source");
  const freqSelect = document.getElementById("balloon-frequency");
  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );
  const previousIndex = parseInt(freqSelect.value);

  const sourceIndex = parseInt(sourceSelect.value);
  if (isNaN(sourceIndex) || sourceIndex >= sourcesWithResponses.length) {
    freqSelect.innerHTML = "";
    updateBalloonSliderState(null);
    updateBalloonPlaceholder(true);
    updateBalloonMeta(null);
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const sampleResponse = source?.responses?.[0];
  const frequencies = sampleResponse?.frequencies || [];

  freqSelect.innerHTML = frequencies
    .map((f, i) => `<option value="${i}">${formatFrequency(f)}</option>`)
    .join("");

  const nextIndex =
    !isNaN(previousIndex) && previousIndex < frequencies.length
      ? previousIndex
      : 0;
  freqSelect.value = String(nextIndex);
  updateBalloonSliderState(frequencies);
  updateBalloonSliderFromIndex(nextIndex, frequencies);
  updateBalloonVisualization();
}

function updatePolarChart() {
  const sourceSelect = document.getElementById("response-source");
  const freqSelect = document.getElementById("polar-frequency");
  const planeSelect = document.getElementById("polar-plane");
  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );

  const sourceIndex = parseInt(sourceSelect.value);
  const freqIndex = parseInt(freqSelect.value);
  const plane = planeSelect?.value || "horizontal";

  if (
    isNaN(sourceIndex) ||
    isNaN(freqIndex) ||
    sourceIndex >= sourcesWithResponses.length
  ) {
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const sampleResponse = source?.responses?.[0];
  const frequencies = sampleResponse?.frequencies || [];
  const frequency = frequencies[freqIndex];

  const slice = computePolarSlice(source, plane, freqIndex);
  if (!slice) {
    return;
  }

  updatePolarSliderFromIndex(freqIndex, frequencies);
  updatePolarFrequencyValue(frequency);

  const ctx = document.getElementById("polar-chart").getContext("2d");
  if (polarChart) {
    polarChart.destroy();
  }

  const levelRange = computeLevelRange(slice.levels);
  const suggestedMax = levelRange.max !== null ? levelRange.max + 3 : undefined;
  const suggestedMin =
    levelRange.max !== null ? levelRange.max - 40 : undefined;

  polarChart = new Chart(ctx, {
    type: "radar",
    data: {
      labels: slice.labels,
      datasets: [
        {
          label: `${plane === "horizontal" ? "Azimuth" : "Elevation"} @ ${formatFrequency(frequency)}`,
          data: slice.levels,
          borderColor: "#16a34a",
          backgroundColor: "rgba(22, 163, 74, 0.12)",
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
      animation: polarChartInitialized ? false : { duration: 700 },
      plugins: {
        legend: {
          position: "top",
        },
        tooltip: {
          callbacks: {
            title: (items) => {
              const label = items?.[0]?.label;
              return label ? `${slice.axisLabel} ${label}°` : "";
            },
            label: (item) => {
              if (item?.raw === null || item?.raw === undefined) {
                return "Level: -";
              }
              return `Level: ${item.raw.toFixed(1)} dB`;
            },
          },
        },
      },
      scales: {
        r: {
          suggestedMin,
          suggestedMax,
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

  polarChartInitialized = true;
  updatePolarMeta(slice, plane, frequency);
}

function handlePolarSliderInput(e) {
  const sliderValue = Number(e.target.value);
  const frequencyData = getPolarFrequencyData();
  if (!frequencyData) {
    return;
  }

  const targetFrequency = sliderValueToFrequency(sliderValue, frequencyData);
  const freqIndex = findNearestFrequencyIndex(
    frequencyData.frequencies,
    targetFrequency,
  );
  const freqSelect = document.getElementById("polar-frequency");
  freqSelect.value = String(freqIndex);
  updatePolarChart();
}

function handleBalloonSliderInput(e) {
  const sliderValue = Number(e.target.value);
  const frequencyData = getBalloonFrequencyData();
  if (!frequencyData) {
    return;
  }

  const targetFrequency = sliderValueToFrequency(sliderValue, frequencyData);
  const freqIndex = findNearestFrequencyIndex(
    frequencyData.frequencies,
    targetFrequency,
  );
  const freqSelect = document.getElementById("balloon-frequency");
  freqSelect.value = String(freqIndex);
  updateBalloonVisualization();
}

function handleBalloonRangeInput(e) {
  const value = Number(e.target.value);
  const label = document.getElementById("balloon-range-value");
  label.textContent = Number.isFinite(value) ? String(value) : "-";
  updateBalloonVisualization();
}

function handleBalloonScaleInput(e) {
  const value = Number(e.target.value);
  const label = document.getElementById("balloon-scale-value");
  label.textContent = Number.isFinite(value) ? `${value.toFixed(1)}×` : "-";
  updateBalloonVisualization();
}

function handleBalloonAutorotateToggle(e) {
  if (balloonGroup) {
    balloonGroup.userData.autoRotate = !!e.target.checked;
  }
}

function updatePolarSliderState(frequencies) {
  const slider = document.getElementById("polar-frequency-slider");
  if (!frequencies || frequencies.length === 0) {
    slider.disabled = true;
    slider.value = "0";
    updatePolarFrequencyValue(null);
    return;
  }

  slider.disabled = false;
  slider.min = "0";
  slider.max = String(polarSliderMax);
  slider.step = "1";
}

function updateBalloonSliderState(frequencies) {
  const slider = document.getElementById("balloon-frequency-slider");
  if (!frequencies || frequencies.length === 0) {
    slider.disabled = true;
    slider.value = "0";
    updateBalloonFrequencyValue(null);
    return;
  }

  slider.disabled = false;
  slider.min = "0";
  slider.max = String(polarSliderMax);
  slider.step = "1";
}

function updatePolarSliderFromIndex(freqIndex, frequencies) {
  const slider = document.getElementById("polar-frequency-slider");
  const frequencyData = getPolarFrequencyData(frequencies);
  if (!frequencyData || frequencyData.logRange === 0) {
    updatePolarSliderState(null);
    return;
  }

  const freqValue = frequencyData.frequencies[freqIndex];
  if (!freqValue || freqValue <= 0) {
    return;
  }

  const ratio =
    (Math.log10(freqValue) - frequencyData.logMin) / frequencyData.logRange;
  const sliderValue = Math.round(ratio * polarSliderMax);
  slider.value = String(Math.max(0, Math.min(polarSliderMax, sliderValue)));
}

function updateBalloonSliderFromIndex(freqIndex, frequencies) {
  const slider = document.getElementById("balloon-frequency-slider");
  const frequencyData = getBalloonFrequencyData(frequencies);
  if (!frequencyData || frequencyData.logRange === 0) {
    updateBalloonSliderState(null);
    return;
  }

  const freqValue = frequencyData.frequencies[freqIndex];
  if (!freqValue || freqValue <= 0) {
    return;
  }

  const ratio =
    (Math.log10(freqValue) - frequencyData.logMin) / frequencyData.logRange;
  const sliderValue = Math.round(ratio * polarSliderMax);
  slider.value = String(Math.max(0, Math.min(polarSliderMax, sliderValue)));
}

function updatePolarFrequencyValue(frequency) {
  const value = document.getElementById("polar-frequency-value");
  value.textContent = frequency ? formatFrequency(frequency) : "-";
}

function getBalloonFrequencyData(frequenciesOverride) {
  const sourceSelect = document.getElementById("balloon-source");
  const sources = currentData?.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );
  const sourceIndex = parseInt(sourceSelect?.value);
  if (isNaN(sourceIndex) || sourceIndex >= sourcesWithResponses.length) {
    return null;
  }

  const source = sourcesWithResponses[sourceIndex];
  const sampleResponse = source?.responses?.[0];
  const frequencies = frequenciesOverride || sampleResponse?.frequencies || [];
  if (!frequencies.length) {
    return null;
  }

  const positive = frequencies.filter((f) => typeof f === "number" && f > 0);
  if (!positive.length) {
    return null;
  }

  const minFreq = Math.min(...positive);
  const maxFreq = Math.max(...positive);
  const logMin = Math.log10(minFreq);
  const logMax = Math.log10(maxFreq);

  return {
    frequencies,
    logMin,
    logRange: logMax - logMin,
  };
}

function updateBalloonFrequencyValue(frequency) {
  const value = document.getElementById("balloon-frequency-value");
  value.textContent = frequency ? formatFrequency(frequency) : "-";
}

function getPolarFrequencyData(frequenciesOverride) {
  const sourceSelect = document.getElementById("response-source");
  const sources = currentData.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );
  const sourceIndex = parseInt(sourceSelect.value);
  if (isNaN(sourceIndex) || sourceIndex >= sourcesWithResponses.length) {
    return null;
  }

  const source = sourcesWithResponses[sourceIndex];
  const sampleResponse = source?.responses?.[0];
  const frequencies = frequenciesOverride || sampleResponse?.frequencies || [];
  if (!frequencies.length) {
    return null;
  }

  const positive = frequencies.filter((f) => typeof f === "number" && f > 0);
  if (!positive.length) {
    return null;
  }

  const minFreq = Math.min(...positive);
  const maxFreq = Math.max(...positive);
  const logMin = Math.log10(minFreq);
  const logMax = Math.log10(maxFreq);

  return {
    frequencies,
    logMin,
    logRange: logMax - logMin,
  };
}

function sliderValueToFrequency(value, frequencyData) {
  const ratio = value / polarSliderMax;
  return Math.pow(10, frequencyData.logMin + frequencyData.logRange * ratio);
}

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

function formatFrequencyShort(hz) {
  if (!hz || hz === 0) return "-";
  if (hz >= 1000) {
    return (hz / 1000).toFixed(1) + "k";
  }
  return hz.toFixed(0);
}

function formatAngle(value) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  const formatted = Number(value).toFixed(1);
  return formatted.endsWith(".0") ? formatted.slice(0, -2) : formatted;
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

  const meridianIdx = responseIndex % grid.meridianCount;
  const parallelIdx = Math.floor(responseIndex / grid.meridianCount);

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
      `<span class="chip">Elevation ${formatAngle(angle.parallelDeg)}°</span>`,
    );
  } else {
    chips.push('<span class="chip">Angle data unavailable</span>');
  }

  if (angRes?.meridian_step && angRes?.parallel_step) {
    chips.push(
      `<span class="chip">Grid ${formatAngle(angRes.meridian_step)}° × ${formatAngle(angRes.parallel_step)}°</span>`,
    );
  }

  meta.innerHTML = chips.join("");
}

function computePolarSlice(source, plane, freqIndex) {
  const balloon = source?.definition?.balloon_data;
  const ang = balloon?.angular_resolution;
  const grid = getBalloonGrid(source);
  if (!grid) {
    return null;
  }

  const responses = source?.responses || [];
  const labels = [];
  const levels = [];
  const stepDeg = 10;

  if (plane === "vertical") {
    // VERTICAL PLANE: Scan parallel angles (elevation) at fixed meridian (azimuth)
    // Coordinate system: Meridian = azimuth (0-360°), Parallel = elevation (0-180°)
    // TiRAY has: 72 meridians (full 360°) × 10 parallels (only 0-45°)

    const meridianDeg = 0; // Front axis (azimuth = 0°)
    const maxParallel = grid.measuredParallelDeg;
    const canMirrorParallel = grid.symmetry === 2 || grid.symmetry === 3;

    // Create full 360° circle by mirroring the limited parallel range when allowed.
    // Without symmetry, leave the back half open.
    for (let angle = 0; angle < 360; angle += stepDeg) {
      let parallelDeg;

      if (angle <= 90) {
        parallelDeg = (angle / 90) * maxParallel;
      } else if (angle <= 180) {
        parallelDeg = ((180 - angle) / 90) * maxParallel;
      } else if (canMirrorParallel) {
        if (angle <= 270) {
          parallelDeg = ((angle - 180) / 90) * maxParallel;
        } else {
          parallelDeg = ((360 - angle) / 90) * maxParallel;
        }
      } else {
        parallelDeg = null;
      }

      const response =
        parallelDeg === null
          ? null
          : getResponseWithSymmetry(source, grid, meridianDeg, parallelDeg);
      const level = response?.level?.[freqIndex];

      labels.push(formatAngle(angle));
      levels.push(level ?? null);
    }

    return {
      labels,
      levels,
      axisLabel: canMirrorParallel
        ? `Vertical (0° = on-axis, ±${maxParallel.toFixed(0)}° max)`
        : `Vertical (0° = on-axis, 0-${maxParallel.toFixed(0)}° measured)`,
      meta: {
        plane: "vertical",
        frequency: formatFrequency(
          source.responses?.[0]?.frequencies?.[freqIndex],
        ),
        symmetry: grid.symmetry,
        parallelRange: canMirrorParallel
          ? `0-${maxParallel}° (mirrored)`
          : `0-${maxParallel}°`,
        meridianDeg,
        stepDeg,
      },
    };
  }

  // HORIZONTAL PLANE: Scan azimuth at parallel=90° (equator/horizontal)
  // Use on-axis elevation when data does not cover the full 0-180 range.
  const targetParallelDeg = 0;

  // Scan full 360° using symmetry mirroring (0-350° to avoid duplicate at 360°=0°)
  for (let az = 0; az < 360; az += stepDeg) {
    const response = getResponseWithSymmetry(
      source,
      grid,
      az,
      targetParallelDeg,
    );
    const level = response?.level?.[freqIndex];

    labels.push(formatAngle(az));
    levels.push(level ?? null);
  }

  return {
    labels,
    levels,
    axisLabel: "Azimuth (0° = front)",
    meta: {
      plane: "horizontal",
      frequency: formatFrequency(
        source.responses?.[0]?.frequencies?.[freqIndex],
      ),
      symmetry: grid.symmetry,
      measuredRange: `${grid.measuredMeridianDeg}°`,
      maxAzimuth: grid.measuredMeridianDeg,
      parallelDeg: targetParallelDeg,
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

  // Calculate expected grid dimensions from resolution only.
  const fullMeridianCount = Math.max(1, Math.round(360 / ang.meridian_step));
  const fullParallelCount =
    Math.max(1, Math.round(180 / ang.parallel_step) + 1);

  // Back-calculate actual dimensions from response count.
  const responseCount = source?.responses?.length || 0;
  let meridianCount = fullMeridianCount;
  let parallelCount = fullParallelCount;

  if (responseCount > 0) {
    meridianCount = Math.max(1, fullMeridianCount);
    parallelCount = Math.max(1, Math.ceil(responseCount / meridianCount));

    if (parallelCount > fullParallelCount) {
      parallelCount = fullParallelCount;
      meridianCount = Math.max(1, Math.ceil(responseCount / parallelCount));
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

  // Calculate response index (row-major: parallel * meridianCount + meridian)
  const responseIndex = parallelIdx * grid.meridianCount + meridianIdx;

  if (responseIndex >= 0 && responseIndex < responses.length) {
    return responses[responseIndex];
  }

  return null;
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

  const responseIndex =
    localParallelIdx * grid.meridianCount + localMeridianIdx;
  if (grid.responseCount && responseIndex >= grid.responseCount) {
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
  return Array.from({ length: count }, (_, i) =>
    startFreq * Math.pow(2, i / bandsPerOctave),
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

function updatePolarMeta(slice, plane, frequency) {
  const meta = document.getElementById("polar-meta");
  if (!meta) return;

  const chips = [];
  chips.push(
    `<span class="chip">Plane ${plane === "vertical" ? "Vertical" : "Horizontal"}</span>`,
  );
  if (frequency) {
    chips.push(
      `<span class="chip">Frequency ${formatFrequency(frequency)}</span>`,
    );
  }

  if (plane === "vertical") {
    chips.push('<span class="chip">Meridian 0°</span>');
    chips.push('<span class="chip">Elevation 0° top</span>');
    chips.push('<span class="chip">90° / -90° bottom</span>');
  } else {
    chips.push(
      `<span class="chip">Parallel ${formatAngle(slice.meta.parallelDeg)}°</span>`,
    );
    chips.push(
      `<span class="chip">Azimuth 0° to ${formatAngle(slice.meta.maxAzimuth ?? 360)}°</span>`,
    );
  }

  meta.innerHTML = chips.join("");
}

function updateBalloonPlaceholder(show) {
  const container = document.getElementById("balloon-viewer");
  if (!container) return;
  let placeholder = document.getElementById("balloon-placeholder");
  if (!placeholder) {
    placeholder = document.createElement("div");
    placeholder.id = "balloon-placeholder";
    placeholder.className = "empty-state";
    placeholder.textContent = "No 3D balloon data available";
    container.appendChild(placeholder);
  }
  placeholder.classList.toggle("hidden", !show);
}

function initBalloonScene() {
  const container = document.getElementById("balloon-viewer");
  if (!container || typeof THREE === "undefined") {
    return false;
  }

  if (balloonRenderer && balloonScene && balloonCamera && balloonGroup) {
    return true;
  }

  const placeholder = document.getElementById("balloon-placeholder");
  if (balloonRenderer && balloonRenderer.domElement?.parentNode) {
    balloonRenderer.domElement.parentNode.removeChild(
      balloonRenderer.domElement,
    );
  }
  if (!placeholder) {
    const placeholderNode = document.createElement("div");
    placeholderNode.id = "balloon-placeholder";
    placeholderNode.className = "empty-state";
    placeholderNode.textContent = "No 3D balloon data available";
    container.appendChild(placeholderNode);
  }

  balloonRenderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
  balloonRenderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  balloonRenderer.setSize(container.clientWidth, container.clientHeight);
  balloonRenderer.setClearColor(0x000000, 0);
  container.appendChild(balloonRenderer.domElement);

  balloonScene = new THREE.Scene();
  balloonCamera = new THREE.PerspectiveCamera(
    45,
    container.clientWidth / container.clientHeight,
    0.1,
    100,
  );
  balloonCamera.position.set(0, 0.6, 2.6);
  balloonCamera.lookAt(0, 0, 0);
  balloonGroup = new THREE.Group();
  balloonGroup.userData.autoRotate =
    document.getElementById("balloon-autorotate")?.checked ?? true;

  const ambient = new THREE.AmbientLight(0xffffff, 0.65);
  const keyLight = new THREE.DirectionalLight(0xffffff, 0.85);
  keyLight.position.set(2.5, 2.5, 2);
  balloonScene.add(ambient, keyLight);

  const reference = new THREE.Mesh(
    new THREE.SphereGeometry(1, 24, 16),
    new THREE.MeshBasicMaterial({
      color: 0x94a3b8,
      wireframe: true,
      transparent: true,
      opacity: 0.28,
    }),
  );
  reference.name = "reference-sphere";
  balloonGroup.add(reference);

  const axes = new THREE.AxesHelper(1.2);
  axes.material.transparent = true;
  axes.material.opacity = 0.5;
  balloonGroup.add(axes);
  balloonScene.add(balloonGroup);

  initBalloonPointerControls(balloonRenderer.domElement);

  if (!balloonResizeBound) {
    window.addEventListener("resize", handleBalloonResize);
    balloonResizeBound = true;
  }

  startBalloonAnimation();
  return true;
}

function startBalloonAnimation() {
  if (!balloonRenderer || !balloonScene || !balloonCamera) {
    return;
  }

  if (balloonFrameId) {
    cancelAnimationFrame(balloonFrameId);
  }

  const animate = () => {
    balloonFrameId = requestAnimationFrame(animate);
    if (balloonGroup && balloonGroup.userData.autoRotate) {
      balloonGroup.rotation.y += 0.0035;
    }
    balloonRenderer.render(balloonScene, balloonCamera);
  };

  animate();
}

function initBalloonPointerControls(target) {
  if (!target) return;

  if (balloonPointerState?.bound) {
    return;
  }

  const state = {
    bound: true,
    dragging: false,
    lastX: 0,
    lastY: 0,
  };

  const onPointerDown = (event) => {
    state.dragging = true;
    state.lastX = event.clientX;
    state.lastY = event.clientY;
    target.setPointerCapture?.(event.pointerId);
  };

  const onPointerMove = (event) => {
    if (!state.dragging || !balloonGroup) return;
    const dx = event.clientX - state.lastX;
    const dy = event.clientY - state.lastY;
    state.lastX = event.clientX;
    state.lastY = event.clientY;
    balloonGroup.rotation.y += dx * 0.005;
    balloonGroup.rotation.x += dy * 0.005;
    balloonGroup.rotation.x = Math.max(
      -Math.PI / 2.2,
      Math.min(Math.PI / 2.2, balloonGroup.rotation.x),
    );
  };

  const onPointerUp = (event) => {
    state.dragging = false;
    target.releasePointerCapture?.(event.pointerId);
  };

  const onWheel = (event) => {
    if (!balloonCamera) return;
    event.preventDefault();
    const delta = Math.sign(event.deltaY) * 0.2;
    const nextZ = Math.max(1.2, Math.min(6, balloonCamera.position.z + delta));
    balloonCamera.position.z = nextZ;
  };

  target.addEventListener("pointerdown", onPointerDown);
  target.addEventListener("pointermove", onPointerMove);
  target.addEventListener("pointerup", onPointerUp);
  target.addEventListener("pointerleave", onPointerUp);
  target.addEventListener("wheel", onWheel, { passive: false });

  balloonPointerState = state;
}

function handleBalloonResize() {
  if (!balloonRenderer || !balloonCamera) {
    return;
  }
  const container = document.getElementById("balloon-viewer");
  if (!container) return;
  const width = container.clientWidth || 1;
  const height = container.clientHeight || 1;
  balloonRenderer.setSize(width, height);
  balloonCamera.aspect = width / height;
  balloonCamera.updateProjectionMatrix();
}

function destroyBalloonScene() {
  if (balloonFrameId) {
    cancelAnimationFrame(balloonFrameId);
    balloonFrameId = null;
  }

  if (balloonMesh) {
    balloonMesh.geometry?.dispose?.();
    balloonMesh.material?.dispose?.();
    balloonMesh = null;
  }

  if (balloonRenderer) {
    balloonRenderer.dispose();
    if (balloonRenderer.domElement?.parentNode) {
      balloonRenderer.domElement.parentNode.removeChild(
        balloonRenderer.domElement,
      );
    }
  }

  balloonRenderer = null;
  balloonScene = null;
  balloonCamera = null;
  balloonGroup = null;
  updateBalloonPlaceholder(true);
}

function updateBalloonVisualization() {
  const sourceSelect = document.getElementById("balloon-source");
  const freqSelect = document.getElementById("balloon-frequency");
  const sources = currentData?.database?.source_definitions || [];
  const sourcesWithResponses = sources.filter(
    (s) => s.responses && s.responses.length > 0,
  );

  const sourceIndex = parseInt(sourceSelect.value);
  const freqIndex = parseInt(freqSelect.value);

  if (
    isNaN(sourceIndex) ||
    isNaN(freqIndex) ||
    sourceIndex >= sourcesWithResponses.length
  ) {
    updateBalloonPlaceholder(true);
    updateBalloonMeta(null);
    return;
  }

  const source = sourcesWithResponses[sourceIndex];
  const balloon = source?.definition?.balloon_data;
  const ang = balloon?.angular_resolution;
  const grid = getBalloonGrid(source);
  if (!balloon || !ang || !grid) {
    updateBalloonPlaceholder(true);
    updateBalloonMeta(null);
    return;
  }

  if (typeof THREE === "undefined") {
    updateBalloonPlaceholder(true);
    updateBalloonMeta(null);
    return;
  }

  const sampleResponse = source?.responses?.[0];
  const frequencies = sampleResponse?.frequencies || [];
  const frequency = frequencies[freqIndex];
  if (!frequency) {
    updateBalloonPlaceholder(true);
    updateBalloonMeta(null);
    return;
  }

  const sceneReady = initBalloonScene();
  if (!sceneReady) {
    updateBalloonPlaceholder(true);
    updateBalloonMeta(null);
    return;
  }
  handleBalloonResize();
  updateBalloonPlaceholder(false);

  updateBalloonSliderFromIndex(freqIndex, frequencies);
  updateBalloonFrequencyValue(frequency);

  const rangeValue = Number(document.getElementById("balloon-range")?.value);
  const scaleValue = Number(document.getElementById("balloon-scale")?.value);
  const dbRange = Number.isFinite(rangeValue) ? rangeValue : 40;
  const scale = Number.isFinite(scaleValue) ? scaleValue : 1;

  const geometryData = buildBalloonGeometry(
    source,
    grid,
    ang,
    freqIndex,
    dbRange,
    scale,
  );

  if (!geometryData) {
    if (balloonMesh) {
      balloonScene?.remove(balloonMesh);
      balloonMesh.geometry?.dispose?.();
      balloonMesh.material?.dispose?.();
      balloonMesh = null;
    }
    updateBalloonPlaceholder(true);
    updateBalloonMeta(null);
    return;
  }

  const { geometry, stats } = geometryData;

  if (balloonMesh) {
    balloonGroup?.remove(balloonMesh);
    balloonMesh.geometry?.dispose?.();
    balloonMesh.material?.dispose?.();
  }

  const wireframe = !!document.getElementById("balloon-wireframe")?.checked;
  const material = new THREE.MeshStandardMaterial({
    vertexColors: true,
    flatShading: false,
    metalness: 0.1,
    roughness: 0.65,
    wireframe,
  });

  balloonMesh = new THREE.Mesh(geometry, material);
  balloonMesh.rotation.x = Math.PI * 0.15;
  balloonGroup?.add(balloonMesh);

  updateBalloonMeta(stats);
}

function buildBalloonGeometry(source, grid, ang, freqIndex, dbRange, scale) {
  const meridianStep = ang.meridian_step;
  const parallelStep = ang.parallel_step;

  if (!meridianStep || !parallelStep) {
    return null;
  }

  const meridianCount = Math.max(
    3,
    grid?.meridianCount || Math.round(360 / meridianStep),
  );
  const parallelCount = Math.max(
    2,
    grid?.parallelCount || Math.round(180 / parallelStep) + 1,
  );
  const wrapMeridian =
    grid?.measuredMeridianDeg !== undefined
      ? grid.measuredMeridianDeg >= 360 - meridianStep * 1.5
      : true;

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
    if (window?.GLL_DEBUG_BALLOON) {
      console.warn("[Balloon] No level data found for frequency index", {
        freqIndex,
        responseCount: source?.responses?.length || 0,
      });
    }
    return null;
  }

  const displayMin = maxLevel - dbRange;
  const baseRadius = 0.3 * scale;
  const amplitude = 0.9 * scale;

  let vertexIndex = 0;
  for (let p = 0; p < parallelCount; p += 1) {
    const parallelDeg = p * parallelStep;
    const phi = (parallelDeg * Math.PI) / 180;
    for (let m = 0; m < meridianCount; m += 1) {
      const azimuthDeg = m * meridianStep;
      const theta = (azimuthDeg * Math.PI) / 180;
      const level = levels[vertexIndex];
      const normalized =
        level === null || level === undefined || Number.isNaN(level)
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
      displayMax: maxLevel,
      dbRange,
      meridianCount,
      parallelCount,
      symmetry: grid.symmetry,
      symmetryName: grid.symmetryName,
    },
  };
}

function updateBalloonMeta(stats) {
  const meta = document.getElementById("balloon-meta");
  if (!meta) return;
  if (!stats) {
    meta.innerHTML = '<span class="chip">No balloon data</span>';
    updateBalloonLegend(null);
    return;
  }

  const chips = [];
  if (stats.frequency) {
    chips.push(
      `<span class="chip">Frequency ${formatFrequency(stats.frequency)}</span>`,
    );
  }
  if (stats.maxLevel !== null) {
    chips.push(
      `<span class="chip">Level ${stats.displayMin.toFixed(1)} to ${stats.displayMax.toFixed(1)} dB</span>`,
    );
  }
  chips.push(
    `<span class="chip">Grid ${stats.meridianCount} × ${stats.parallelCount}</span>`,
  );
  chips.push(
    `<span class="chip">Symmetry ${stats.symmetryName || "Unknown"}</span>`,
  );
  meta.innerHTML = chips.join("");
  updateBalloonLegend(stats);
}

function updateBalloonLegend(stats) {
  const legend = document.querySelector(".balloon-legend");
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

function getPhaseSeries(mode, frequencies, phase, unwrapped) {
  switch (mode) {
    case "wrapped": {
      const wrapped = phase.map((value) => wrapPhase(value));
      return {
        values: wrapped,
        label: "Phase (rad)",
        axisTitle: "Phase (rad)",
        tableHeader: "Phase (rad)",
        format: (value) => formatNumber(value, 4),
      };
    }
    case "group-delay": {
      const delayMs = computeGroupDelayMs(frequencies, unwrapped);
      return {
        values: delayMs,
        label: "Group Delay (ms)",
        axisTitle: "Group Delay (ms)",
        tableHeader: "Group Delay (ms)",
        format: (value) => formatNumber(value, 3),
      };
    }
    case "unwrapped":
    default:
      return {
        values: unwrapped,
        label: "Phase (rad)",
        axisTitle: "Phase (rad)",
        tableHeader: "Phase (rad)",
        format: (value) => formatNumber(value, 4),
      };
  }
}

function unwrapPhase(phase) {
  if (!Array.isArray(phase) || phase.length === 0) return [];
  const unwrapped = [phase[0]];
  let offset = 0;
  for (let i = 1; i < phase.length; i++) {
    const delta = phase[i] - phase[i - 1];
    if (delta > Math.PI) {
      offset -= 2 * Math.PI;
    } else if (delta < -Math.PI) {
      offset += 2 * Math.PI;
    }
    unwrapped.push(phase[i] + offset);
  }
  return unwrapped;
}

function wrapPhase(value) {
  if (value === null || value === undefined) return null;
  const twoPi = 2 * Math.PI;
  const wrapped = ((((value + Math.PI) % twoPi) + twoPi) % twoPi) - Math.PI;
  return wrapped;
}

function computeGroupDelayMs(frequencies, phaseUnwrapped) {
  if (!Array.isArray(frequencies) || frequencies.length === 0) return [];
  const count = Math.min(frequencies.length, phaseUnwrapped.length);
  const delays = new Array(count);
  const scale = -1 / (2 * Math.PI);

  for (let i = 0; i < count; i++) {
    let dPhi;
    let dF;
    if (i === 0) {
      dPhi = phaseUnwrapped[i + 1] - phaseUnwrapped[i];
      dF = frequencies[i + 1] - frequencies[i];
    } else if (i === count - 1) {
      dPhi = phaseUnwrapped[i] - phaseUnwrapped[i - 1];
      dF = frequencies[i] - frequencies[i - 1];
    } else {
      dPhi = phaseUnwrapped[i + 1] - phaseUnwrapped[i - 1];
      dF = frequencies[i + 1] - frequencies[i - 1];
    }

    if (!dF || dF === 0 || Number.isNaN(dF) || Number.isNaN(dPhi)) {
      delays[i] = null;
      continue;
    }

    const delaySeconds = scale * (dPhi / dF);
    delays[i] = delaySeconds * 1000;
  }

  return delays;
}

function formatNumber(value, digits) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return Number(value).toFixed(digits);
}

function isPowerOfTen(value) {
  if (!value || value <= 0) return false;
  const exponent = Math.log10(value);
  return Number.isInteger(exponent);
}

function buildLogTicks(min, max) {
  if (!min || !max || min <= 0 || max <= 0) return [];
  const ticks = [];
  const startPower = Math.max(1, Math.floor(Math.log10(min)));
  const endPower = Math.ceil(Math.log10(max));

  for (let power = startPower; power <= endPower; power++) {
    const decade = Math.pow(10, power);
    for (let multiplier = 1; multiplier <= 9; multiplier++) {
      const value = multiplier * decade;
      if (value < min || value > max) {
        continue;
      }
      ticks.push({ value });
    }
  }

  return ticks;
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
  } else {
    content.style.display = "none";
    toggle.textContent = "▶";
    group.classList.remove("expanded");
  }
}

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
