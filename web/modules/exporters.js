// Geometry Export Formats
// Provides functions to convert GLL geometry data to various 3D file formats

// XED Format (EASE native format)
export function buildXedContent(geometry, options) {
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

export function buildSequentialEdgePairs(vertexCount) {
  const pairs = [];
  for (let i = 0; i + 1 < vertexCount; i += 2) {
    pairs.push([i + 1, i + 2]);
  }
  return pairs;
}

function formatXedLine(point, precision) {
  return `${formatXedNumber(point.x, precision)} ${formatXedNumber(point.y, precision)} ${formatXedNumber(point.z, precision)}`;
}

function formatXedNumber(value, precision) {
  const rounded = Number(value).toFixed(precision);
  return rounded.replace(/\.?0+$/, "");
}

// STL Binary Format
export function buildStlBinary(geometry, options) {
  const vertices = Array.isArray(geometry.vertices) ? geometry.vertices : [];
  const faces = Array.isArray(geometry.faces) ? geometry.faces : [];
  const edges = Array.isArray(geometry.edges) ? geometry.edges : [];

  if (faces.length === 0) {
    if (vertices.length === 0 && edges.length === 0) {
      return new Uint8Array(0);
    }
    return buildEmptyStlBinary();
  }

  // Collect all triangles with symmetry applied
  const triangles = [];
  for (const face of faces) {
    const faceVertices = face.vertices || [];
    if (faceVertices.length < 3) continue;

    // Triangulate face using fan triangulation
    const tris = triangulateFace(faceVertices);
    for (const tri of tris) {
      const v1 = resolveGeometryVertex(geometry, vertices, tri[0], 1);
      const v2 = resolveGeometryVertex(geometry, vertices, tri[1], 1);
      const v3 = resolveGeometryVertex(geometry, vertices, tri[2], 1);
      if (v1 && v2 && v3) {
        triangles.push([v1, v2, v3]);
      }
    }
  }

  if (triangles.length === 0) {
    return new Uint8Array(0);
  }

  // Calculate buffer size: 80-byte header + 4-byte count + (50 bytes × triangle count)
  const bufferSize = 80 + 4 + triangles.length * 50;
  const buffer = new ArrayBuffer(bufferSize);
  const view = new DataView(buffer);

  // Write 80-byte header (can be any data, typically zeros or description)
  const headerText = "Binary STL from GLL Viewer";
  for (let i = 0; i < headerText.length && i < 80; i++) {
    view.setUint8(i, headerText.charCodeAt(i));
  }

  // Write triangle count (uint32 little-endian at offset 80)
  view.setUint32(80, triangles.length, true);

  // Write triangles
  let offset = 84;
  for (const [v1, v2, v3] of triangles) {
    // Calculate normal using cross product: (v2-v1) × (v3-v1)
    const normal = calculateNormal(v1, v2, v3);

    // Normal vector (3 × float32)
    view.setFloat32(offset, normal.x, true);
    view.setFloat32(offset + 4, normal.y, true);
    view.setFloat32(offset + 8, normal.z, true);
    offset += 12;

    // Vertex 1 (3 × float32)
    view.setFloat32(offset, v1.x, true);
    view.setFloat32(offset + 4, v1.y, true);
    view.setFloat32(offset + 8, v1.z, true);
    offset += 12;

    // Vertex 2 (3 × float32)
    view.setFloat32(offset, v2.x, true);
    view.setFloat32(offset + 4, v2.y, true);
    view.setFloat32(offset + 8, v2.z, true);
    offset += 12;

    // Vertex 3 (3 × float32)
    view.setFloat32(offset, v3.x, true);
    view.setFloat32(offset + 4, v3.y, true);
    view.setFloat32(offset + 8, v3.z, true);
    offset += 12;

    // Attribute byte count (uint16, typically 0)
    view.setUint16(offset, 0, true);
    offset += 2;
  }

  return new Uint8Array(buffer);
}

function buildEmptyStlBinary() {
  const buffer = new ArrayBuffer(84);
  const view = new DataView(buffer);
  const headerText = "Binary STL from GLL Viewer";
  for (let i = 0; i < headerText.length && i < 80; i++) {
    view.setUint8(i, headerText.charCodeAt(i));
  }
  view.setUint32(80, 0, true);
  return new Uint8Array(buffer);
}

// OBJ Format (Wavefront)
export function buildObjContent(geometry, options) {
  const vertices = Array.isArray(geometry.vertices) ? geometry.vertices : [];
  const faces = Array.isArray(geometry.faces) ? geometry.faces : [];
  const edges = Array.isArray(geometry.edges) ? geometry.edges : [];

  if (vertices.length === 0) {
    return "";
  }

  const lines = ["# Wavefront OBJ from GLL Viewer", ""];

  // Build vertex list with symmetry applied
  const vertexMap = new Map(); // Maps original index to obj index
  let objIndex = 1;

  // Process all vertices referenced by faces (preferred) or edges (fallback)
  const referencedIndices = new Set();
  const useFaces = faces.length > 0;
  if (useFaces) {
    for (const face of faces) {
      const faceVertices = face.vertices || [];
      for (const idx of faceVertices) {
        referencedIndices.add(idx);
        // Also add mirrored vertex if negative
        if (idx < 0 && geometry.is_symmetric) {
          referencedIndices.add(Math.abs(idx));
        }
      }
    }
  } else if (edges.length > 0) {
    for (const edge of edges) {
      const indices = [edge.v1, edge.v2];
      for (const idx of indices) {
        referencedIndices.add(idx);
        if (idx < 0 && geometry.is_symmetric) {
          referencedIndices.add(Math.abs(idx));
        }
      }
    }
  } else {
    for (let i = 1; i <= vertices.length; i += 1) {
      referencedIndices.add(i);
    }
  }

  // Write vertices
  for (const index of Array.from(referencedIndices).sort(
    (a, b) => Math.abs(a) - Math.abs(b),
  )) {
    const vertex = resolveGeometryVertex(geometry, vertices, index, 1);
    if (vertex) {
      lines.push(
        `v ${vertex.x.toFixed(6)} ${vertex.y.toFixed(6)} ${vertex.z.toFixed(6)}`,
      );
      vertexMap.set(index, objIndex++);
    }
  }

  lines.push("");

  if (useFaces) {
    // Write faces
    for (const face of faces) {
      const faceVertices = face.vertices || [];
      if (faceVertices.length < 3) continue;

      const objIndices = faceVertices
        .map((idx) => vertexMap.get(idx))
        .filter((idx) => idx !== undefined);

      if (objIndices.length >= 3) {
        lines.push(`f ${objIndices.join(" ")}`);
      }
    }
  } else if (edges.length > 0) {
    // Write edges as line segments
    for (const edge of edges) {
      const v1 = vertexMap.get(edge.v1);
      const v2 = vertexMap.get(edge.v2);
      if (v1 != null && v2 != null) {
        lines.push(`l ${v1} ${v2}`);
      }
    }
  }

  return lines.join("\n") + "\n";
}

// Shared Geometry Utilities

export function resolveGeometryVertex(geometry, vertices, index, scale) {
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

export function triangulateFace(vertexIndices) {
  const triangles = [];
  if (vertexIndices.length < 3) return triangles;

  // Fan triangulation: (v0, v1, v2), (v0, v2, v3), (v0, v3, v4), ...
  const v0 = vertexIndices[0];
  for (let i = 1; i + 1 < vertexIndices.length; i++) {
    triangles.push([v0, vertexIndices[i], vertexIndices[i + 1]]);
  }

  return triangles;
}

export function calculateNormal(v1, v2, v3) {
  // Edge vectors
  const e1x = v2.x - v1.x;
  const e1y = v2.y - v1.y;
  const e1z = v2.z - v1.z;

  const e2x = v3.x - v1.x;
  const e2y = v3.y - v1.y;
  const e2z = v3.z - v1.z;

  // Cross product: e1 × e2
  const nx = e1y * e2z - e1z * e2y;
  const ny = e1z * e2x - e1x * e2z;
  const nz = e1x * e2y - e1y * e2x;

  // Normalize
  const length = Math.sqrt(nx * nx + ny * ny + nz * nz);
  if (length < 1e-10) {
    return { x: 0, y: 0, z: 1 };
  }

  return {
    x: nx / length,
    y: ny / length,
    z: nz / length,
  };
}

export function hasGeometryData(geometry) {
  if (!geometry) return false;
  const hasEdges = Array.isArray(geometry.edges) && geometry.edges.length > 0;
  const hasVertices =
    Array.isArray(geometry.vertices) && geometry.vertices.length > 1;
  return hasEdges || hasVertices;
}

// Filter Spectrum Export Formats

const RAD_TO_DEG = 180 / Math.PI;

// FRD Format (Frequency Response Data)
// Standard text format: frequency_hz  level_db  phase_deg
export function buildFrdContent(frequencies, levels, phases) {
  const lines = [];
  for (let i = 0; i < frequencies.length; i++) {
    const freq = frequencies[i];
    const level = levels[i] ?? 0;
    const phaseDeg = phases?.[i] != null ? phases[i] * RAD_TO_DEG : 0;
    lines.push(
      `${freq.toFixed(3)}  ${level.toFixed(2)}  ${phaseDeg.toFixed(2)}`,
    );
  }
  return lines.join("\n") + "\n";
}

// CSV Format for filter responses
export function buildCsvFilterContent(frequencies, levels, phases) {
  const lines = ["Frequency (Hz),Level (dB),Phase (deg)"];
  for (let i = 0; i < frequencies.length; i++) {
    const freq = frequencies[i];
    const level = levels[i] ?? 0;
    const phaseDeg = phases?.[i] != null ? phases[i] * RAD_TO_DEG : 0;
    lines.push(`${freq.toFixed(3)},${level.toFixed(2)},${phaseDeg.toFixed(2)}`);
  }
  return lines.join("\n") + "\n";
}

// XGFB Format (XGLL Generic Filter Bank)
// Human-readable text format mirroring the binary GenericFilterBank structure.
// Phase values in radians (matching GLL internal format).
export function buildXgfbContent(filterDef) {
  const bank = filterDef?.filter;
  if (!bank) return "";

  const lines = [
    "# XGFB - Generic Filter Bank",
    "# Exported from gll-tools",
    "",
    "[FilterBank]",
    `Bypass = ${!!bank.bypass}`,
    `InvertPolarity = ${!!bank.invert_polarity}`,
    `MuteInput = ${!!bank.mute_input}`,
    `Gain = ${(bank.gain || 0).toFixed(2)}`,
    `Delay = ${(bank.delay || 0).toFixed(6)}`,
  ];

  const filters = bank.filters || [];
  for (let i = 0; i < filters.length; i++) {
    const f = filters[i];
    lines.push("");
    lines.push(`[Filter.${i}]`);
    lines.push(`Kind = ${formatXgfbKind(f.kind)}`);
    if (f.label) lines.push(`Label = "${f.label}"`);
    if (f.key) lines.push(`Key = "${f.key}"`);
    lines.push(`Bypass = ${!!f.bypass}`);
    lines.push(`InvertPolarity = ${!!f.invert_polarity}`);
    lines.push(`Gain = ${(f.gain || 0).toFixed(2)}`);
    lines.push(`Delay = ${(f.delay || 0).toFixed(6)}`);

    if (f.kind === 0 && f.log_spectrum) {
      const s = f.log_spectrum;
      lines.push(`BandsPerOctave = ${s.bands_per_octave}`);
      lines.push(`LowestFrequency = ${s.lowest_frequency}`);
      lines.push(`NumberOfBands = ${s.number_of_bands}`);
      if (s.delay) lines.push(`SpectrumDelay = ${s.delay.toFixed(6)}`);
      lines.push("");
      lines.push(`[Filter.${i}.Spectrum]`);
      lines.push("# Freq(Hz)  Level(dB)  Phase(rad)");
      const count = s.level?.length || s.phase?.length || 0;
      for (let j = 0; j < count; j++) {
        const freq = s.lowest_frequency * Math.pow(2, j / s.bands_per_octave);
        const level = s.level?.[j] ?? 0;
        const phase = s.phase?.[j] ?? 0;
        lines.push(
          `${freq.toFixed(6)}  ${level.toFixed(4)}  ${phase.toFixed(6)}`,
        );
      }
    } else if (f.kind === 1 && f.iir_params) {
      const p = f.iir_params;
      lines.push(`FilterType = ${formatXgfbIIRType(p.filter_type)}`);
      lines.push(`FilterShape = ${formatXgfbIIRShape(p.filter_shape)}`);
      lines.push(`Order = ${p.order}`);
      lines.push(`FreqCritInHz = ${p.freq_crit_hz}`);
      lines.push(`Alignment = ${p.alignment}`);
      lines.push(`QFactor = ${(p.q_factor || 0).toFixed(6)}`);
      lines.push(
        `ParametricGainIndB = ${(p.parametric_gain_db || 0).toFixed(3)}`,
      );
    } else if (f.kind === 2 && f.fir_data) {
      const d = f.fir_data;
      lines.push(`IsTimeResponse = ${!!d.is_time_response}`);
      lines.push(`IsComplex = ${!!d.is_complex}`);
      lines.push(`IsEven = ${!!d.is_even}`);
      lines.push(`SampleRate = ${d.sample_rate}`);
      const dataIRM = d.data_irm || [];
      const dataDIP = d.data_dip || [];
      const count = Math.max(dataIRM.length, dataDIP.length);
      lines.push(`DataCount = ${count}`);
      lines.push("");
      lines.push(`[Filter.${i}.Data]`);
      lines.push(
        d.is_time_response
          ? "# Index  Real  Imaginary"
          : "# Index  Magnitude  Phase",
      );
      for (let j = 0; j < count; j++) {
        const val1 = dataIRM[j] ?? 0;
        const val2 = dataDIP[j] ?? 0;
        lines.push(`${j}  ${val1}  ${val2}`);
      }
    }
  }

  return lines.join("\n") + "\n";
}

function formatXgfbKind(kind) {
  switch (kind) {
    case 0:
      return "LogSpectrum";
    case 1:
      return "IIR";
    case 2:
      return "FIR";
    default:
      return `Unknown(${kind})`;
  }
}

function formatXgfbIIRType(type) {
  const names = [
    "LowPass",
    "HighPass",
    "AllPass",
    "Peak",
    "PeakSym",
    "LowShelf",
    "HighShelf",
  ];
  return names[type] ?? `Unknown(${type})`;
}

function formatXgfbIIRShape(shape) {
  const names = ["Butterworth", "LinkwitzRiley", "Bessel", "SallenKey"];
  return names[shape] ?? `Unknown(${shape})`;
}
