import {
  buildFrequencyPoints,
  buildLogFrequencyScale,
  buildZoomPluginOptions,
  getPhaseSeries,
  unwrapPhase,
} from "./charting.js";

let filterChart = null;
let filterChartInitialized = false;
let filterContext = null;
let filterListenersBound = false;

export function setupFilterControls(context) {
  filterContext = context;
  const groupSelect = document.getElementById("filter-group-select");
  const defSelect = document.getElementById("filter-def-select");
  const phaseSelect = document.getElementById("filter-phase-mode");
  if (!groupSelect || !defSelect) {
    return;
  }

  if (!filterListenersBound) {
    groupSelect.addEventListener("change", updateFilterDefinitionOptions);
    defSelect.addEventListener("change", updateFilterChart);
    if (phaseSelect) {
      phaseSelect.addEventListener("change", updateFilterChart);
    }
    filterListenersBound = true;
  }

  const data = filterContext?.getData?.();
  const groups = data?.database?.filter_groups || [];
  if (!groups.length) {
    groupSelect.innerHTML = '<option value="">No filter groups</option>';
    groupSelect.disabled = true;
    defSelect.innerHTML = "";
    defSelect.disabled = true;
    updateFilterMeta("No filter response data available");
    setFilterPlaceholder("No filter response data available");
    return;
  }

  groupSelect.disabled = false;
  groupSelect.innerHTML = groups
    .map(
      (group, i) =>
        `<option value="${i}">${escapeHtml(group.label || group.key || `Group ${i + 1}`)}</option>`,
    )
    .join("");

  updateFilterDefinitionOptions();
}

export function updateFilterDefinitionOptions() {
  const groupSelect = document.getElementById("filter-group-select");
  const defSelect = document.getElementById("filter-def-select");
  if (!groupSelect || !defSelect) {
    return;
  }

  const data = filterContext?.getData?.();
  const groups = data?.database?.filter_groups || [];
  const groupIndex = parseInt(groupSelect.value);
  if (isNaN(groupIndex) || groupIndex >= groups.length) {
    defSelect.innerHTML = "";
    defSelect.disabled = true;
    setFilterPlaceholder("Select a filter group to view response data");
    updateFilterChart();
    return;
  }

  const group = groups[groupIndex];
  const defs = group?.filters || [];
  if (!defs.length) {
    defSelect.innerHTML = '<option value="">No filters</option>';
    defSelect.disabled = true;
    setFilterPlaceholder("No filters available in this group");
    updateFilterChart();
    return;
  }

  defSelect.disabled = false;
  defSelect.innerHTML = defs
    .map(
      (filter, i) =>
        `<option value="${i}">${escapeHtml(filter.label || filter.key || `Filter ${i + 1}`)}</option>`,
    )
    .join("");

  updateFilterChart();
}

export function updateFilterChart() {
  const groupSelect = document.getElementById("filter-group-select");
  const defSelect = document.getElementById("filter-def-select");
  if (!groupSelect || !defSelect) {
    return;
  }

  const data = filterContext?.getData?.();
  const bytes = filterContext?.getBytes?.();
  const groups = data?.database?.filter_groups || [];
  const groupIndex = parseInt(groupSelect.value);
  const defIndex = parseInt(defSelect.value);

  if (
    isNaN(groupIndex) ||
    isNaN(defIndex) ||
    groupIndex >= groups.length
  ) {
    updateFilterMeta("No filter response data available");
    setFilterPlaceholder("No filter response data available");
    destroyFilterChart();
    return;
  }

  if (!bytes || typeof computeFilterResponse !== "function") {
    updateFilterMeta("Filter response helper not available");
    setFilterPlaceholder("Filter response helper not available");
    destroyFilterChart();
    return;
  }

  const group = groups[groupIndex];
  const filterDef = group?.filters?.[defIndex];

  const payload = JSON.stringify({
    group_index: groupIndex,
    filter_index: defIndex,
  });
  let response;
  try {
    const responseJSON = computeFilterResponse(bytes, payload);
    response = JSON.parse(responseJSON);
  } catch (err) {
    updateFilterMeta("Failed to compute filter response");
    setFilterPlaceholder("Failed to compute filter response");
    destroyFilterChart();
    return;
  }
  if (!response.success) {
    updateFilterMeta(response.error || "Failed to compute filter response");
    setFilterPlaceholder(response.error || "Failed to compute filter response");
    destroyFilterChart();
    return;
  }

  if (!response.frequencies?.length || !response.level?.length) {
    const message =
      response.message || "No filter response data available";
    updateFilterMeta(
      message,
      buildFilterMetaChips(group, filterDef, response),
    );
    setFilterPlaceholder(message);
    destroyFilterChart();
    return;
  }

  const phaseSelect = document.getElementById("filter-phase-mode");
  if (phaseSelect) {
    const hasPhase = Array.isArray(response.phase) && response.phase.length > 0;
    phaseSelect.disabled = !hasPhase;
    if (!hasPhase) {
      phaseSelect.value = "unwrapped";
    }
  }

  const ctx = document.getElementById("filter-response-chart").getContext("2d");
  if (filterChart) {
    filterChart.destroy();
  }

  const frequencyData = buildFrequencyPoints(
    response.frequencies,
    response.level,
  );
  if (!frequencyData) {
    updateFilterMeta("No filter response data available");
    setFilterPlaceholder("No filter response data available");
    destroyFilterChart();
    return;
  }

  setFilterPlaceholder("");
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
    const phaseMode = phaseSelect?.value || "unwrapped";
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

  filterChart = new Chart(ctx, {
    type: "line",
    data: {
      datasets,
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: filterChartInitialized ? false : { duration: 700 },
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

  filterChartInitialized = true;
  updateFilterMeta(response.message, buildFilterMetaChips(group, filterDef, response));
}

export function resetFilterChart() {
  destroyFilterChart();
  filterChartInitialized = false;
}

function destroyFilterChart() {
  if (filterChart) {
    filterChart.destroy();
    filterChart = null;
  }
}

function buildFilterMetaChips(group, filterDef, response) {
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
    chips.push(`<span class="chip">${escapeHtml(kind)} ${response.used_filters}</span>`);
  }
  if (Number.isFinite(response.sample_rate)) {
    chips.push(`<span class="chip">SR ${response.sample_rate.toFixed(0)} Hz</span>`);
  }
  if (Number.isFinite(response.point_count)) {
    chips.push(`<span class="chip">${response.point_count} points</span>`);
  }
  if (response.is_complex) {
    chips.push('<span class="chip">Complex</span>');
  }
  if (Number.isFinite(response.skipped_filters) && response.skipped_filters > 0) {
    chips.push(`<span class="chip">Skipped ${response.skipped_filters}</span>`);
  }
  if (Number.isFinite(response.mismatched_filters) && response.mismatched_filters > 0) {
    chips.push(`<span class="chip">Mismatched ${response.mismatched_filters}</span>`);
  }
  if (response.bypassed) {
    chips.push('<span class="chip">Bypassed</span>');
  }
  return chips;
}

function updateFilterMeta(message, chips = null) {
  const meta = document.getElementById("filter-response-meta");
  if (!meta) return;

  const parts = Array.isArray(chips) && chips.length > 0 ? chips.slice() : [];
  if (message) {
    parts.push(`<span class="chip">${escapeHtml(message)}</span>`);
  }
  if (parts.length === 0) {
    meta.innerHTML = '<span class="chip">No filter response data available</span>';
    return;
  }
  meta.innerHTML = parts.join("");
}

function setFilterPlaceholder(message) {
  const placeholder = document.getElementById("filter-response-placeholder");
  if (!placeholder) return;
  if (message) {
    placeholder.textContent = message;
    placeholder.style.display = "flex";
    return;
  }
  placeholder.style.display = "none";
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

function formatFrequency(hz) {
  if (!hz || hz === 0) return "-";
  if (hz >= 1000) {
    return (hz / 1000).toFixed(hz % 1000 === 0 ? 0 : 1) + " kHz";
  }
  return hz.toFixed(0) + " Hz";
}
