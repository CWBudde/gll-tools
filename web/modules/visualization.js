export function createVisualizationController({
  getCurrentData,
  getCachedArrayBalloon,
  formatFrequency,
  formatAngle,
  computePolarSlices,
  getBalloonGrid,
  getResponseWithSymmetry,
  escapeHtml,
}) {
  // Chart and scene state
  let polarChart = null;
  let polarChartInitialized = false;
  let balloonRenderer = null;
  let balloonScene = null;
  let balloonCamera = null;
  let balloonGroup = null;
  let balloonMesh = null;
  let balloonFrameId = null;
  let balloonResizeBound = false;
  let balloonPointerState = null;
  let coverageLines = [];
  const balloonMaxCache = new WeakMap();
  const arrayBalloonMaxCache = new WeakMap();
  const polarSliderMax = 1000;

  // Chart.js plugin for polar compass labels
  const polarCompassPlugin = {
    id: "polarCompass",
    afterDraw(chart) {
      const scale = chart.scales?.r;
      if (!scale) return;
      const { xCenter, yCenter, drawingArea } = scale;
      const ctx = chart.ctx;
      const sideOffset = 40; // Horizontal offset for Front/Back labels
      const vertOffset = 28; // Vertical offset for Top/Bottom labels

      ctx.save();
      ctx.font = "bold 12px sans-serif";

      // IMPORTANT: Front/on-axis is on the RIGHT (startAngle=90).
      // Both slices are great circles through the front-back axis:
      //   Horizontal (blue): Front → Right → Back → Left  (meridian 90°/270°)
      //   Vertical   (red):  Front → Top   → Back → Bottom (meridian 0°/180°)
      //
      // Chart positions:
      //   Right  = Front (shared)
      //   Left   = Back  (shared)
      //   Top    = Right (H, blue) / Top    (V, red)
      //   Bottom = Left  (H, blue) / Bottom (V, red)

      // Right = Front, Left = Back (shared by both slices).
      ctx.fillStyle = "#334155";
      ctx.textBaseline = "middle";
      ctx.textAlign = "left";
      ctx.fillText("Front", xCenter + drawingArea + sideOffset, yCenter);
      ctx.textAlign = "right";
      ctx.fillText("Back", xCenter - drawingArea - sideOffset, yCenter);

      // Top of chart: Right (horizontal, blue) / Top (vertical, red).
      ctx.textAlign = "center";
      ctx.textBaseline = "bottom";
      ctx.fillStyle = "#2563eb";
      ctx.fillText("Right", xCenter - 18, yCenter - drawingArea - vertOffset);
      ctx.fillStyle = "#dc2626";
      ctx.fillText("Top", xCenter + 18, yCenter - drawingArea - vertOffset);

      // Bottom of chart: Left (horizontal, blue) / Bottom (vertical, red).
      ctx.textBaseline = "top";
      ctx.fillStyle = "#2563eb";
      ctx.fillText("Left", xCenter - 22, yCenter + drawingArea + vertOffset);
      ctx.fillStyle = "#dc2626";
      ctx.fillText("Bottom", xCenter + 22, yCenter + drawingArea + vertOffset);

      ctx.restore();
    },
  };

  function updatePolarOptions() {
    // Populate polar frequency options from cached array balloon data
    const freqSelect = document.getElementById("polar-frequency");
    if (!freqSelect) {
      return;
    }

    const cached = getCachedArrayBalloon();
    if (!cached?.frequencies?.length) {
      // Fall back to single-source if no array data
      const currentData = getCurrentData();
      const sources = currentData?.database?.source_definitions || [];
      const sourcesWithResponses = sources.filter(
        (s) => s.responses && s.responses.length > 0,
      );
      if (!sourcesWithResponses.length) {
        freqSelect.innerHTML = "";
        updateGlobalSliderState(null);
        document.getElementById("polar-meta").innerHTML =
          '<div class="empty-state">No polar data available. Build a configuration and recalculate.</div>';
        return;
      }
      // Use first source as frequency reference
      const source = sourcesWithResponses[0];
      const sampleResponse = source?.responses?.[0];
      const frequencies = sampleResponse?.frequencies || [];
      const previousIndex = parseInt(freqSelect.value);
      freqSelect.innerHTML = frequencies
        .map((f, i) => `<option value="${i}">${formatFrequency(f)}</option>`)
        .join("");
      const nextIndex =
        !isNaN(previousIndex) && previousIndex < frequencies.length
          ? previousIndex
          : findNearestFrequencyIndex(frequencies, 1000);
      freqSelect.value = String(nextIndex);
      updateGlobalSliderState(frequencies);
      updateGlobalSliderFromIndex(nextIndex, frequencies);
      updatePolarChart();
      return;
    }

    const frequencies = cached.frequencies;
    const previousIndex = parseInt(freqSelect.value);
    freqSelect.innerHTML = frequencies
      .map((f, i) => `<option value="${i}">${formatFrequency(f)}</option>`)
      .join("");

    const nextIndex =
      !isNaN(previousIndex) && previousIndex < frequencies.length
        ? previousIndex
        : findNearestFrequencyIndex(frequencies, 1000);
    freqSelect.value = String(nextIndex);
    updateGlobalSliderState(frequencies);
    updateGlobalSliderFromIndex(nextIndex, frequencies);

    updatePolarChart();
  }

  function updateBalloonSourceOptions() {
    // Source selects are now populated by setupSourceControls() in app.js
  }

  function updateBalloonOptions() {
    // Populate balloon frequency options from cached array balloon data
    const freqSelect = document.getElementById("balloon-frequency");
    if (!freqSelect) {
      return;
    }

    const cached = getCachedArrayBalloon();
    if (!cached?.frequencies?.length) {
      freqSelect.innerHTML = "";
      updateGlobalSliderState(null);
      updateBalloonPlaceholder(true);
      updateBalloonMeta(null);
      return;
    }

    const frequencies = cached.frequencies;
    const previousIndex = parseInt(freqSelect.value);
    freqSelect.innerHTML = frequencies
      .map((f, i) => `<option value="${i}">${formatFrequency(f)}</option>`)
      .join("");

    const nextIndex =
      !isNaN(previousIndex) && previousIndex < frequencies.length
        ? previousIndex
        : findNearestFrequencyIndex(frequencies, 1000);
    freqSelect.value = String(nextIndex);
    updateGlobalSliderState(frequencies);
    updateGlobalSliderFromIndex(nextIndex, frequencies);
    updateBalloonVisualization();
  }

  function extractArrayPolarSlices(cached, freqIndex) {
    // Extract horizontal and vertical slices from cached array balloon grid
    const { merCount, parCount } = cached.grid;
    const stepDeg = 10;
    const angles = buildPolarAngles(stepDeg);
    const labels = angles.map(formatPolarLabel);
    const horizontalLevels = [];
    const verticalLevels = [];

    // Grid indexing: result[m * parCount + p] for meridian m, parallel p
    // Meridian 90° = index merCount * 90/360 = merCount/4
    // Meridian 270° = index merCount * 270/360 = 3*merCount/4
    // Meridian 0° = index 0
    // Meridian 180° = index merCount/2
    const merRight = Math.round((merCount * 90) / 360);
    const merLeft = Math.round((merCount * 270) / 360);
    const merTop = 0;
    const merBottom = Math.round((merCount * 180) / 360);

    for (const angle of angles) {
      const parallelDeg = Math.abs(angle);
      // Parallel index: 0=0°, parCount-1=180°
      const pIdx = Math.min(
        parCount - 1,
        Math.round((parallelDeg / 180) * (parCount - 1)),
      );

      // Horizontal slice (right/left plane)
      const hMerIdx = angle >= 0 ? merRight : merLeft;
      const hResult = cached.results[hMerIdx * parCount + pIdx];
      horizontalLevels.push(hResult?.level?.[freqIndex] ?? null);

      // Vertical slice (top/bottom plane)
      const vMerIdx = angle >= 0 ? merTop : merBottom;
      const vResult = cached.results[vMerIdx * parCount + pIdx];
      verticalLevels.push(vResult?.level?.[freqIndex] ?? null);
    }

    return { labels, horizontalLevels, verticalLevels };
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

  function formatPolarLabel(angle) {
    const normalized = ((angle + 180) % 360) - 180;
    if (Math.abs(normalized) === 180) {
      return "±180°";
    }
    if (Math.abs(normalized) < 1e-6) {
      return "0°";
    }
    return `${formatAngle(normalized)}°`;
  }

  function updatePolarChart() {
    // Render polar directivity chart from cached array balloon data
    const freqSelect = document.getElementById("polar-frequency");
    if (!freqSelect) {
      return;
    }

    const freqIndex = parseInt(freqSelect.value);
    if (isNaN(freqIndex)) {
      return;
    }

    const cached = getCachedArrayBalloon();
    let slices;
    let frequency;
    let frequencies;
    let isFallback = false;

    if (cached?.frequencies?.length) {
      frequencies = cached.frequencies;
      frequency = frequencies[freqIndex];
      const extracted = extractArrayPolarSlices(cached, freqIndex);
      slices = {
        labels: extracted.labels,
        horizontal: { levels: extracted.horizontalLevels, meridianDeg: 90 },
        vertical: {
          levels: extracted.verticalLevels,
          meridianDeg: 0,
          maxParallel: 180,
          canMirrorParallel: false,
        },
        meta: {
          symmetryName: "Array",
          frontHalfOnly: false,
          usesOnAxis: false,
          stepDeg: 10,
        },
      };
    } else {
      // Fallback to single-source
      const currentData = getCurrentData();
      const sources = currentData?.database?.source_definitions || [];
      const sourcesWithResponses = sources.filter(
        (s) => s.responses && s.responses.length > 0,
      );
      if (!sourcesWithResponses.length) return;
      const source = sourcesWithResponses[0];
      const sampleResponse = source?.responses?.[0];
      frequencies = sampleResponse?.frequencies || [];
      frequency = frequencies[freqIndex];
      slices = computePolarSlices(source, freqIndex);
      isFallback = true;
    }

    if (!slices) return;

    updatePolarSliderFromIndex(freqIndex, frequencies);
    updatePolarFrequencyValue(frequency);

    const ctx = document.getElementById("polar-chart").getContext("2d");
    if (polarChart) {
      // Replace existing chart instance
      polarChart.destroy();
    }

    // Check if normalization is enabled
    const normalizeCheckbox = document.getElementById("polar-normalize");
    const normalize = normalizeCheckbox?.checked ?? false;

    // Apply normalization if enabled
    let horizontalLevels = slices.horizontal.levels;
    let verticalLevels = slices.vertical.levels;

    if (normalize) {
      // Normalize to each slice's max level
      const hMax = Math.max(
        ...horizontalLevels.filter((v) => v !== null && !isNaN(v)),
      );
      const vMax = Math.max(
        ...verticalLevels.filter((v) => v !== null && !isNaN(v)),
      );

      horizontalLevels = horizontalLevels.map((v) =>
        v !== null && !isNaN(v) ? v - hMax : v,
      );
      verticalLevels = verticalLevels.map((v) =>
        v !== null && !isNaN(v) ? v - vMax : v,
      );
    }

    const levelRange = computeLevelRange([
      ...horizontalLevels,
      ...verticalLevels,
    ]);
    const suggestedMax =
      levelRange.max !== null ? levelRange.max + 3 : undefined;
    const suggestedMin =
      levelRange.max !== null ? levelRange.max - 40 : undefined;

    polarChart = new Chart(ctx, {
      // Create Chart.js radar chart
      type: "radar",
      plugins: [polarCompassPlugin],
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
        animation: polarChartInitialized ? false : { duration: 700 },
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
            // Chart.js radar: 0°=top, 90°=right, 180°=bottom, 270°=left.
            // IMPORTANT: Directivity polar convention — Front/on-axis is on
            // the RIGHT of the plot.  startAngle=90 rotates the first label
            // (0°) from the default top position to the right.
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

    polarChartInitialized = true;
    updatePolarMeta(slices, frequency);
  }

  function handleGlobalSliderInput(e) {
    // Sync all frequency controls from the global slider
    const sliderValue = Number(e.target.value);
    const frequencyData = getGlobalFrequencyData();
    if (!frequencyData) {
      return;
    }

    const targetFrequency = sliderValueToFrequency(sliderValue, frequencyData);
    const freqIndex = findNearestFrequencyIndex(
      frequencyData.frequencies,
      targetFrequency,
    );

    // Update both frequency selects
    const polarFreqSelect = document.getElementById("polar-frequency");
    const balloonFreqSelect = document.getElementById("balloon-frequency");
    if (polarFreqSelect) {
      polarFreqSelect.value = String(freqIndex);
    }
    if (balloonFreqSelect) {
      balloonFreqSelect.value = String(freqIndex);
    }

    // Update both visualizations
    updatePolarChart();
    updateBalloonVisualization();
  }

  function handlePolarSliderInput(e) {
    handleGlobalSliderInput(e);
  }

  function handleBalloonSliderInput(e) {
    handleGlobalSliderInput(e);
  }

  function handleBalloonRangeInput(e) {
    // Update dB range for balloon visualization
    const value = Number(e.target.value);
    const label = document.getElementById("balloon-range-value");
    label.textContent = Number.isFinite(value) ? String(value) : "-";
    updateBalloonVisualization();
  }

  function handleBalloonScaleInput(e) {
    // Update size scale for balloon visualization
    const value = Number(e.target.value);
    const label = document.getElementById("balloon-scale-value");
    label.textContent = Number.isFinite(value) ? `${value.toFixed(1)}×` : "-";
    updateBalloonVisualization();
  }

  function handleBalloonAutorotateToggle(e) {
    // Toggle auto-rotation of balloon mesh
    if (balloonGroup) {
      balloonGroup.userData.autoRotate = !!e.target.checked;
    }
  }

  function updateGlobalSliderState(frequencies) {
    // Enable/disable global frequency slider
    const slider = document.getElementById("global-frequency-slider");
    if (!slider) return;

    if (!frequencies || frequencies.length === 0) {
      slider.disabled = true;
      slider.value = "0";
      updateGlobalFrequencyValue(null);
      return;
    }

    slider.disabled = false;
    slider.min = "0";
    slider.max = String(polarSliderMax);
    slider.step = "1";
  }

  function updatePolarSliderState(frequencies) {
    updateGlobalSliderState(frequencies);
  }

  function updateBalloonSliderState(frequencies) {
    updateGlobalSliderState(frequencies);
  }

  function updateGlobalSliderFromIndex(freqIndex, frequencies) {
    // Move global slider based on selected frequency
    const slider = document.getElementById("global-frequency-slider");
    if (!slider) return;

    const frequencyData = getGlobalFrequencyData(frequencies);
    if (!frequencyData || frequencyData.logRange === 0) {
      updateGlobalSliderState(null);
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

    updateGlobalFrequencyValue(freqValue);
  }

  function updatePolarSliderFromIndex(freqIndex, frequencies) {
    updateGlobalSliderFromIndex(freqIndex, frequencies);
  }

  function updateBalloonSliderFromIndex(freqIndex, frequencies) {
    updateGlobalSliderFromIndex(freqIndex, frequencies);
  }

  function updateGlobalFrequencyValue(frequency) {
    // Update frequency label text
    const value = document.getElementById("global-frequency-value");
    if (value) {
      value.textContent = frequency ? formatFrequency(frequency) : "-";
    }
  }

  function updatePolarFrequencyValue(frequency) {
    updateGlobalFrequencyValue(frequency);
  }

  function getBalloonFrequencyData(frequenciesOverride) {
    return getGlobalFrequencyData(frequenciesOverride);
  }

  function updateBalloonFrequencyValue(frequency) {
    updateGlobalFrequencyValue(frequency);
  }

  function getGlobalFrequencyData(frequenciesOverride) {
    // Compute log range used for slider mapping
    // Prefer cached array balloon frequencies
    const cached = getCachedArrayBalloon();
    if (cached?.frequencies?.length) {
      const frequencies = frequenciesOverride || cached.frequencies;
      if (!frequencies.length) return null;
      const logMin = Math.log10(frequencies[0]);
      const logMax = Math.log10(frequencies[frequencies.length - 1]);
      return { frequencies, logMin, logRange: logMax - logMin };
    }

    // Fallback to first source with responses
    const currentData = getCurrentData();
    const sources = currentData?.database?.source_definitions || [];
    const sourcesWithResponses = sources.filter(
      (s) => s.responses && s.responses.length > 0,
    );
    if (!sourcesWithResponses.length) return null;

    const source = sourcesWithResponses[0];
    const sampleResponse = source?.responses?.[0];
    const frequencies =
      frequenciesOverride || sampleResponse?.frequencies || [];
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

  function getPolarFrequencyData(frequenciesOverride) {
    return getGlobalFrequencyData(frequenciesOverride);
  }

  function sliderValueToFrequency(value, frequencyData) {
    // Convert slider position to frequency
    const ratio = value / polarSliderMax;
    return Math.pow(10, frequencyData.logMin + frequencyData.logRange * ratio);
  }

  function findNearestFrequencyIndex(frequencies, targetFrequency) {
    // Find closest frequency index to a target value
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
    // Compute min/max ignoring nulls
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

  function updatePolarMeta(slices, frequency) {
    // Populate metadata chips for polar chart
    const meta = document.getElementById("polar-meta");
    if (!meta) return;

    const chips = [];
    if (frequency) {
      chips.push(
        `<span class="chip">Frequency ${formatFrequency(frequency)}</span>`,
      );
    }
    if (slices?.meta?.symmetryName) {
      chips.push(
        `<span class="chip">Symmetry ${slices.meta.symmetryName}</span>`,
      );
    }
    if (slices?.meta?.frontHalfOnly) {
      chips.push('<span class="chip">Front half only</span>');
    }
    if (slices?.meta?.usesOnAxis) {
      chips.push('<span class="chip">On-axis + directivity</span>');
    }
    chips.push(
      `<span class="chip">Horizontal meridian ${formatAngle(slices.horizontal.meridianDeg ?? 90)}°</span>`,
    );
    chips.push(
      `<span class="chip">Vertical meridian ${formatAngle(slices.vertical.meridianDeg ?? 0)}°</span>`,
    );
    chips.push(
      `<span class="chip">Vertical range 0-${formatAngle(slices.vertical.maxParallel ?? 180)}°${slices.vertical.canMirrorParallel ? " (mirrored)" : ""}</span>`,
    );
    chips.push(
      `<span class="chip">Step ${formatAngle(slices.meta.stepDeg ?? 10)}°</span>`,
    );

    meta.innerHTML = chips.join("");
  }

  function updateBalloonPlaceholder(show) {
    // Show or hide balloon placeholder
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
    // Initialize Three.js scene for balloon visualization
    const container = document.getElementById("balloon-viewer");
    if (!container || typeof THREE === "undefined") {
      return false;
    }

    if (balloonRenderer && balloonScene && balloonCamera && balloonGroup) {
      // Scene already initialized
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
    // Drive balloon render loop
    if (!balloonRenderer || !balloonScene || !balloonCamera) {
      return;
    }

    if (balloonFrameId) {
      cancelAnimationFrame(balloonFrameId);
    }

    const animate = () => {
      balloonFrameId = requestAnimationFrame(animate);
      if (balloonGroup && balloonGroup.userData.autoRotate) {
        // Auto-rotate balloon group
        balloonGroup.rotation.y += 0.0035;
      }
      balloonRenderer.render(balloonScene, balloonCamera);
    };

    animate();
  }

  function initBalloonPointerControls(target) {
    // Pointer controls for rotating and zooming balloon
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
      // Begin drag rotation
      state.dragging = true;
      state.lastX = event.clientX;
      state.lastY = event.clientY;
      target.setPointerCapture?.(event.pointerId);
    };

    const onPointerMove = (event) => {
      // Apply drag rotation
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
      // End drag
      state.dragging = false;
      target.releasePointerCapture?.(event.pointerId);
    };

    const onWheel = (event) => {
      // Zoom camera in/out
      if (!balloonCamera) return;
      event.preventDefault();
      const delta = Math.sign(event.deltaY) * 0.2;
      const nextZ = Math.max(
        1.2,
        Math.min(6, balloonCamera.position.z + delta),
      );
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
    // Resize renderer and camera
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
    // Dispose balloon renderer and scene
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

  function buildArrayBalloonGeometry(cached, freqIndex, dbRange, scale, normalize) {
    // Build balloon geometry from cached array response grid
    const { merCount, parCount } = cached.grid;
    const meridianStep = 360 / merCount;
    const parallelStep = 180 / (parCount - 1);

    const levels = [];
    let maxLevel = null;
    let minLevel = null;

    // Grid is stored as merCount * parCount, with meridian-major ordering
    for (let p = 0; p < parCount; p++) {
      for (let m = 0; m < merCount; m++) {
        const result = cached.results[m * parCount + p];
        const level = result?.level?.[freqIndex];
        if (level !== null && level !== undefined && !isNaN(level)) {
          if (maxLevel === null || level > maxLevel) maxLevel = level;
          if (minLevel === null || level < minLevel) minLevel = level;
        }
        levels.push(level ?? null);
      }
    }

    if (maxLevel === null) return null;

    const globalMaxLevel = getArrayGlobalMaxLevel(cached);
    const displayMax = normalize ? maxLevel : globalMaxLevel;
    if (displayMax === null) return null;
    const displayMin = displayMax - dbRange;
    const baseRadius = 0.3 * scale;
    const amplitude = 0.9 * scale;

    const positions = [];
    const colors = [];
    const color = new THREE.Color();
    let vertexIndex = 0;

    for (let p = 0; p < parCount; p++) {
      const parallelDeg = p * parallelStep;
      const phi = (parallelDeg * Math.PI) / 180;
      for (let m = 0; m < merCount; m++) {
        const azimuthDeg = m * meridianStep;
        const theta = (azimuthDeg * Math.PI) / 180;
        const rawLevel = levels[vertexIndex];
        const level = rawLevel !== null && !isNaN(rawLevel) ? rawLevel : null;
        const normalized =
          level === null ? null : Math.min(Math.max((level - displayMin) / dbRange, 0), 1);
        const radius = normalized !== null ? baseRadius + amplitude * normalized : baseRadius;

        positions.push(
          radius * Math.sin(phi) * Math.cos(theta),
          radius * Math.sin(phi) * Math.sin(theta),
          radius * Math.cos(phi),
        );

        if (normalized !== null) {
          color.setHSL(0.67 * (1 - normalized), 0.85, 0.5);
        } else {
          color.setRGB(0.5, 0.5, 0.5);
        }
        colors.push(color.r, color.g, color.b);
        vertexIndex++;
      }
    }

    // Build triangle indices
    const indices = [];
    for (let p = 0; p < parCount - 1; p++) {
      for (let m = 0; m < merCount; m++) {
        const m1 = (m + 1) % merCount;
        const i00 = p * merCount + m;
        const i01 = p * merCount + m1;
        const i10 = (p + 1) * merCount + m;
        const i11 = (p + 1) * merCount + m1;
        indices.push(i00, i10, i01);
        indices.push(i01, i10, i11);
      }
    }

    const geometry = new THREE.BufferGeometry();
    geometry.setIndex(indices);
    geometry.setAttribute("position", new THREE.Float32BufferAttribute(positions, 3));
    geometry.setAttribute("color", new THREE.Float32BufferAttribute(colors, 3));
    geometry.computeVertexNormals();

    return {
      geometry,
      stats: {
        frequency: cached.frequencies[freqIndex],
        minLevel,
        maxLevel,
        displayMin,
        displayMax,
        dbRange,
        meridianCount: merCount,
        parallelCount: parCount,
        symmetry: 0,
        symmetryName: "Array",
        normalized: normalize,
      },
    };
  }

  function getArrayGlobalMaxLevel(cached) {
    if (!cached) return null;
    const existing = arrayBalloonMaxCache.get(cached);
    if (existing !== undefined) {
      return existing;
    }

    let maxLevel = null;
    for (const result of cached.results || []) {
      for (const value of result?.level || []) {
        if (value === null || value === undefined || Number.isNaN(value)) {
          continue;
        }
        if (maxLevel === null || value > maxLevel) {
          maxLevel = value;
        }
      }
    }

    arrayBalloonMaxCache.set(cached, maxLevel);
    return maxLevel;
  }

  function updateBalloonVisualization() {
    // Compute and render the 3D balloon mesh from cached array data or single source
    const freqSelect = document.getElementById("balloon-frequency");
    if (!freqSelect) {
      return;
    }

    const freqIndex = parseInt(freqSelect.value);
    if (isNaN(freqIndex)) {
      updateBalloonPlaceholder(true);
      updateBalloonMeta(null);
      return;
    }

    const cached = getCachedArrayBalloon();

    if (cached?.frequencies?.length) {
      // Array-driven balloon
      const frequency = cached.frequencies[freqIndex];
      if (!frequency) {
        updateBalloonPlaceholder(true);
        updateBalloonMeta(null);
        return;
      }

      if (typeof THREE === "undefined") {
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

      updateBalloonSliderFromIndex(freqIndex, cached.frequencies);
      updateBalloonFrequencyValue(frequency);

      const rangeValue = Number(document.getElementById("balloon-range")?.value);
      const scaleValue = Number(document.getElementById("balloon-scale")?.value);
      const normalizeCheckbox = document.getElementById("balloon-normalize");
      const dbRange = Number.isFinite(rangeValue) ? rangeValue : 40;
      const scale = Number.isFinite(scaleValue) ? scaleValue : 1;
      const normalize = normalizeCheckbox?.checked ?? false;

      const geometryData = buildArrayBalloonGeometry(cached, freqIndex, dbRange, scale, normalize);
      renderBalloonGeometry(geometryData);
      return;
    }

    // Fallback to single-source balloon (legacy path)
    const currentData = getCurrentData();
    const sources = currentData?.database?.source_definitions || [];
    const sourcesWithResponses = sources.filter(
      (s) => s.responses && s.responses.length > 0,
    );

    if (!sourcesWithResponses.length) {
      updateBalloonPlaceholder(true);
      updateBalloonMeta(null);
      return;
    }

    const source = sourcesWithResponses[0];
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
    const normalizeCheckbox = document.getElementById("balloon-normalize");
    const dbRange = Number.isFinite(rangeValue) ? rangeValue : 40;
    const scale = Number.isFinite(scaleValue) ? scaleValue : 1;
    const normalize = normalizeCheckbox?.checked ?? false;

    const geometryData = buildBalloonGeometry(
      source,
      grid,
      ang,
      freqIndex,
      dbRange,
      scale,
      normalize,
    );

    renderBalloonGeometry(geometryData, source, grid, ang, freqIndex, dbRange, scale, normalize);
  }

  function renderBalloonGeometry(geometryData, source, grid, ang, freqIndex, dbRange, scale, normalize) {
    if (!geometryData) {
      // No geometry to render
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
    balloonGroup?.add(balloonMesh);

    // Remove old coverage lines
    for (const line of coverageLines) {
      balloonGroup?.remove(line);
      line.geometry?.dispose();
      line.material?.dispose();
    }
    coverageLines = [];

    // Draw coverage contour lines (only for single-source mode with grid data)
    if (source && grid && ang && document.getElementById("balloon-coverage")?.checked) {
      const contours = buildCoverageContours(
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
          balloonGroup?.add(line);
          coverageLines.push(line);
        }
      }
    }

    updateBalloonMeta(stats);
  }

  function getSourceGlobalMaxLevel(source) {
    // Cache max SPL level for normalization reference
    if (!source) return null;
    const cached = balloonMaxCache.get(source);
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
    balloonMaxCache.set(source, maxLevel);
    return maxLevel;
  }

  function buildCoverageContours(
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

    // On-axis reference level
    const onAxisResp = getResponseWithSymmetry(source, grid, 0, 0);
    const onAxisLevel = onAxisResp?.level?.[freqIndex];
    if (onAxisLevel == null || Number.isNaN(onAxisLevel)) return null;

    const globalMaxLevel = getSourceGlobalMaxLevel(source);
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
            // Interpolate between p-1 and p
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
            // Compute 3D position with same deformation as mesh
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

  function buildBalloonGeometry(
    source,
    grid,
    ang,
    freqIndex,
    dbRange,
    scale,
    normalize = false,
  ) {
    // Build vertex grid and mesh for balloon visualization
    const meridianStep = ang.meridian_step;
    const parallelStep = ang.parallel_step;

    if (!meridianStep || !parallelStep) {
      return null;
    }

    // Use full-sphere grid for rendering; getResponseWithSymmetry handles mirroring.
    const meridianCount = Math.max(
      3,
      grid?.fullMeridianCount || Math.round(360 / meridianStep),
    );
    const parallelCount = Math.max(
      2,
      grid?.fullParallelCount || Math.round(180 / parallelStep) + 1,
    );
    // Full sphere always wraps meridian.
    const wrapMeridian = true;

    const levels = [];
    const positions = [];
    const colors = [];
    const color = new THREE.Color();

    let maxLevel = null;
    let minLevel = null;

    for (let p = 0; p < parallelCount; p += 1) {
      // Sweep parallels
      const parallelDeg = p * parallelStep;
      for (let m = 0; m < meridianCount; m += 1) {
        // Sweep meridians
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
      // No levels found
      if (window?.GLL_DEBUG_BALLOON) {
        console.warn("[Balloon] No level data found for frequency index", {
          freqIndex,
          responseCount: source?.responses?.length || 0,
        });
      }
      return null;
    }

    const globalMaxLevel = getSourceGlobalMaxLevel(source);
    // Normalize against either local or global max
    const displayMax = normalize ? maxLevel : globalMaxLevel;
    if (displayMax === null) {
      return null;
    }
    const displayMin = displayMax - dbRange;
    const baseRadius = 0.3 * scale;
    const amplitude = 0.9 * scale;

    let vertexIndex = 0;
    for (let p = 0; p < parallelCount; p += 1) {
      // Build vertex positions and colors
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
          normalized === null
            ? baseRadius
            : baseRadius + amplitude * normalized;

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
      // Build triangle indices between parallels
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
    // Upload position and color attributes
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

  function updateBalloonMeta(stats) {
    // Populate metadata chips for balloon view
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
        `<span class="chip">Level ${stats.displayMin.toFixed(1)} to ${stats.displayMax.toFixed(1)} dB${stats.normalized ? " (normalized)" : ""}</span>`,
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
    // Update min/max labels in legend
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

  function resetVisualization() {
    // Tear down charts and 3D views
    if (polarChart) {
      polarChart.destroy();
      polarChart = null;
    }
    polarChartInitialized = false;
    destroyBalloonScene();
  }

  return {
    updatePolarOptions,
    updateBalloonSourceOptions,
    updateBalloonOptions,
    updatePolarChart,
    updateBalloonVisualization,
    handleGlobalSliderInput,
    handlePolarSliderInput,
    handleBalloonSliderInput,
    handleBalloonRangeInput,
    handleBalloonScaleInput,
    handleBalloonAutorotateToggle,
    handleBalloonResize,
    resetVisualization,
  };
}
