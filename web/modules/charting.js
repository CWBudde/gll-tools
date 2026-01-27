export function buildFrequencyPoints(frequencies, values) {
  if (!Array.isArray(frequencies) || !Array.isArray(values)) {
    return null;
  }
  if (frequencies.length === 0 || values.length === 0) {
    return null;
  }
  if (frequencies.length !== values.length) {
    return null;
  }
  const points = frequencies.map((freq, i) => ({
    x: freq,
    y: values[i],
  }));
  const minFrequency = Math.min(...frequencies);
  const maxFrequency = Math.max(...frequencies);
  if (!Number.isFinite(minFrequency) || !Number.isFinite(maxFrequency)) {
    return null;
  }
  return { points, minFrequency, maxFrequency };
}

export function buildLogFrequencyScale(minFrequency, maxFrequency, label) {
  return {
    type: "logarithmic",
    title: {
      display: true,
      text: label,
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
  };
}

export function buildZoomPluginOptions(minFrequency, maxFrequency) {
  return {
    limits: {
      x: {
        min: minFrequency,
        max: maxFrequency,
      },
    },
    pan: {
      enabled: true,
      mode: "x",
      modifierKey: "shift",
    },
    zoom: {
      wheel: {
        enabled: true,
        modifierKey: "ctrl",
      },
      pinch: {
        enabled: true,
      },
      mode: "x",
    },
  };
}

export function getPhaseSeries(mode, frequencies, phase, unwrapped) {
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

export function unwrapPhase(phase) {
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

export function wrapPhase(value) {
  if (value === null || value === undefined) return null;
  const twoPi = 2 * Math.PI;
  const wrapped = ((((value + Math.PI) % twoPi) + twoPi) % twoPi) - Math.PI;
  return wrapped;
}

export function computeGroupDelayMs(frequencies, phaseUnwrapped) {
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

export function buildLogTicks(min, max) {
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

function formatFrequencyShort(hz) {
  if (!hz || hz === 0) return "-";
  if (hz >= 1000) {
    return (hz / 1000).toFixed(1) + "k";
  }
  return hz.toFixed(0);
}

function isPowerOfTen(value) {
  if (!value || value <= 0) return false;
  const exponent = Math.log10(value);
  return Number.isInteger(exponent);
}

function formatNumber(value, digits) {
  if (value === null || value === undefined || Number.isNaN(value)) return "-";
  return Number(value).toFixed(digits);
}
