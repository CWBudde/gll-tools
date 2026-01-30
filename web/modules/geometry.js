export function createGeometryController({
  getCurrentData,
  formatNumber,
  hasGeometryData,
  resolveGeometryVertex,
  buildSequentialEdgePairs,
}) {
  // Map of "kind-index" -> instance state
  const instances = new Map();
  let themeColors = readThemeColors();

  function getThemeVar(name, fallback) {
    if (typeof window === "undefined" || !document?.documentElement) {
      return fallback;
    }
    const value = getComputedStyle(document.documentElement)
      .getPropertyValue(name)
      .trim();
    return value || fallback;
  }

  function readThemeColors() {
    return {
      gridMajor: getThemeVar("--geom-grid-major", "#94a3b8"),
      gridMinor: getThemeVar("--geom-grid-minor", "#e2e8f0"),
      gridOpacity: parseFloat(getThemeVar("--geom-grid-opacity", "0.45")),
      edgeFallback: getThemeVar("--geom-edge-default", "#475569"),
      faceFallback: getThemeVar("--geom-face-default", "#60a5fa"),
    };
  }

  function normalizePoint(point) {
    if (!point) return null;
    const x = Number(point.x);
    const y = Number(point.y);
    const z = Number(point.z);
    if (![x, y, z].every(Number.isFinite)) {
      return null;
    }
    return { x, y, z };
  }

  function toViewPoint(point) {
    const p = normalizePoint(point);
    if (!p) return null;
    // Most likely fix: GLL geometry is Z-up; swap Y/Z for Three.js (Y-up).
    return { x: p.x, y: p.z, z: p.y };
  }

  function applyGridTheme(grid, colors) {
    if (!grid || !colors) return;
    const materials = Array.isArray(grid.material) ? grid.material : [grid.material];
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

  function createInstance(container, caseGeometry, item) {
    if (!container || typeof THREE === "undefined") return null;

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.setSize(container.clientWidth, container.clientHeight);
    renderer.setClearColor(0x000000, 0);
    container.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(
      42,
      container.clientWidth / container.clientHeight,
      0.01,
      200,
    );
    camera.position.set(0, 0.4, 2.2);
    camera.lookAt(0, 0, 0);

    const group = new THREE.Group();
    group.userData.autoRotate = true;
    const orbit = {
      theta: Math.PI * 0.65,
      phi: Math.PI * 0.35,
      radius: 2,
      target: new THREE.Vector3(0, 0, 0),
    };

    const controls = THREE.OrbitControls
      ? new THREE.OrbitControls(camera, renderer.domElement)
      : null;
    if (controls) {
      controls.enableDamping = true;
      controls.dampingFactor = 0.08;
      controls.screenSpacePanning = true;
      controls.enablePan = true;
      controls.enableZoom = true;
      controls.enableKeys = true;
      controls.keyPanSpeed = 20;
      controls.minDistance = 0.25;
      controls.maxDistance = 25;
      controls.rotateSpeed = 0.6;
      controls.panSpeed = 0.9;
      controls.autoRotate = true;
      controls.mouseButtons = {
        LEFT: THREE.MOUSE.ROTATE,
        MIDDLE: THREE.MOUSE.DOLLY,
        RIGHT: THREE.MOUSE.PAN,
      };
      controls.target.copy(orbit.target);
      controls.listenToKeyEvents(window);
      controls.update();
    }

    const ambient = new THREE.AmbientLight(0xffffff, 0.7);
    const keyLight = new THREE.DirectionalLight(0xffffff, 0.85);
    keyLight.position.set(2.5, 2.5, 2);
    scene.add(ambient, keyLight);

    const grid = new THREE.GridHelper(2, 12, 0x94a3b8, 0xe2e8f0);
    applyGridTheme(grid, themeColors);
    scene.add(grid);

    const axes = new THREE.AxesHelper(0.8);
    axes.material.transparent = true;
    axes.material.opacity = 0.5;
    scene.add(axes);
    scene.add(group);

    const inst = {
      renderer,
      scene,
      camera,
      group,
      orbit,
      controls,
      mesh: null,
      lines: null,
      markers: null,
      frameId: null,
      container,
      caseGeometry,
      item,
      grid,
      showFaces: true,
      showEdges: true,
    };

    if (!controls) {
      initPointerControls(inst, renderer.domElement);
    }
    populateGeometry(inst, true, true);
    startAnimation(inst);
    return inst;
  }

  function updateCameraFromOrbit(inst) {
    if (!inst.camera || !inst.orbit) return;
    const { radius, phi, theta, target } = inst.orbit;
    const sinPhi = Math.sin(phi);
    const cosPhi = Math.cos(phi);
    const sinTheta = Math.sin(theta);
    const cosTheta = Math.cos(theta);
    inst.camera.position.set(
      target.x + radius * sinPhi * cosTheta,
      target.y + radius * cosPhi,
      target.z + radius * sinPhi * sinTheta,
    );
    inst.camera.lookAt(target);
  }

  function startAnimation(inst) {
    if (!inst.renderer || !inst.scene || !inst.camera) return;
    if (inst.frameId) cancelAnimationFrame(inst.frameId);

    const animate = () => {
      inst.frameId = requestAnimationFrame(animate);
      if (inst.controls) {
        inst.controls.update();
      } else if (inst.orbit && inst.group?.userData.autoRotate) {
        inst.orbit.theta += 0.0035;
        updateCameraFromOrbit(inst);
      }
      inst.renderer.render(inst.scene, inst.camera);
    };
    animate();
  }

  function initPointerControls(inst, target) {
    if (!target) return;
    const state = {
      dragging: false,
      lastX: 0,
      lastY: 0,
      mode: null,
    };

    target.addEventListener("pointerdown", (e) => {
      if (e.button === 2) {
        state.mode = "pan";
      } else if (e.button === 0) {
        state.mode = "rotate";
      } else {
        return;
      }
      state.dragging = true;
      state.lastX = e.clientX;
      state.lastY = e.clientY;
      target.setPointerCapture?.(e.pointerId);
    });
    target.addEventListener("pointermove", (e) => {
      if (!state.dragging || !inst.orbit) return;
      const dx = e.clientX - state.lastX;
      const dy = e.clientY - state.lastY;
      state.lastX = e.clientX;
      state.lastY = e.clientY;
      if (state.mode === "pan" && inst.camera) {
        const distance = inst.orbit.radius;
        const vFov = (inst.camera.fov * Math.PI) / 180;
        const viewHeight = 2 * Math.tan(vFov / 2) * distance;
        const viewWidth = viewHeight * inst.camera.aspect;
        const deltaX = (dx / Math.max(1, target.clientWidth)) * viewWidth;
        const deltaY = (dy / Math.max(1, target.clientHeight)) * viewHeight;

        const forward = new THREE.Vector3();
        inst.camera.getWorldDirection(forward);
        const right = new THREE.Vector3().crossVectors(forward, inst.camera.up).normalize();
        const up = new THREE.Vector3().copy(inst.camera.up).normalize();

        inst.orbit.target.addScaledVector(right, -deltaX);
        inst.orbit.target.addScaledVector(up, deltaY);
        updateCameraFromOrbit(inst);
      } else if (state.mode === "rotate") {
        inst.orbit.theta -= dx * 0.006;
        inst.orbit.phi -= dy * 0.006;
        inst.orbit.phi = Math.max(0.05, Math.min(Math.PI - 0.05, inst.orbit.phi));
        updateCameraFromOrbit(inst);
      }
    });
    const onUp = (e) => {
      state.dragging = false;
      state.mode = null;
      target.releasePointerCapture?.(e.pointerId);
    };
    target.addEventListener("pointerup", onUp);
    target.addEventListener("pointerleave", onUp);
    target.addEventListener("contextmenu", (e) => e.preventDefault());
    target.addEventListener(
      "wheel",
      (e) => {
        if (!inst.camera || !inst.orbit) return;
        e.preventDefault();
        const delta = Math.sign(e.deltaY) * 0.2;
        inst.orbit.radius = Math.max(0.25, Math.min(25, inst.orbit.radius + delta));
        updateCameraFromOrbit(inst);
      },
      { passive: false },
    );
  }

  function populateGeometry(inst, showFaces, showEdges) {
    const geometry = inst.caseGeometry;
    if (!hasGeometryData(geometry)) return;

    const geometryData = buildCaseGeometryData(geometry, { showFaces, showEdges });
    if (!geometryData || (!geometryData.mesh && !geometryData.lines)) return;

    inst.showFaces = showFaces;
    inst.showEdges = showEdges;

    // Remove old
    if (inst.mesh) {
      inst.group.remove(inst.mesh);
      inst.mesh.geometry?.dispose?.();
      inst.mesh.material?.dispose?.();
      inst.mesh = null;
    }
    if (inst.lines) {
      inst.group.remove(inst.lines);
      inst.lines.geometry?.dispose?.();
      inst.lines.material?.dispose?.();
      inst.lines = null;
    }
    if (inst.markers) {
      inst.group.remove(inst.markers);
      inst.markers.traverse((child) => {
        child.geometry?.dispose?.();
        child.material?.dispose?.();
      });
      inst.markers = null;
    }

    inst.group.rotation.set(0, 0, 0);
    inst.group.scale.set(1, 1, 1);

    if (geometryData.mesh) {
      const material = new THREE.MeshStandardMaterial({
        vertexColors: true,
        flatShading: true,
        metalness: 0.05,
        roughness: 0.75,
        side: THREE.DoubleSide,
      });
      inst.mesh = new THREE.Mesh(geometryData.mesh, material);
      inst.group.add(inst.mesh);
    }

    if (geometryData.lines) {
      const lineMaterial = new THREE.LineBasicMaterial({
        vertexColors: true,
        transparent: true,
        opacity: 0.9,
      });
      inst.lines = new THREE.LineSegments(geometryData.lines, lineMaterial);
      inst.group.add(inst.lines);
    }

    if (geometryData.bounds && inst.camera && inst.group) {
      const bounds = geometryData.bounds;
      const center = {
        x: (bounds.minX + bounds.maxX) / 2,
        y: (bounds.minY + bounds.maxY) / 2,
        z: (bounds.minZ + bounds.maxZ) / 2,
      };
      inst.group.position.set(-center.x, -center.y, -center.z);

      const size =
        geometryData.size ??
        Math.max(
          bounds.maxX - bounds.minX,
          bounds.maxY - bounds.minY,
          bounds.maxZ - bounds.minZ,
        );
      const targetSize = 1.2;
      const scaleFactor = size > 0 ? targetSize / size : 1;
      inst.group.scale.setScalar(scaleFactor);

      const markerRadiusWorld = 0.03;
      const markerRadiusRaw = scaleFactor > 0
        ? markerRadiusWorld / scaleFactor
        : markerRadiusWorld;

      const markers = new THREE.Group();
      const refPoint = toViewPoint(inst.item?.reference_point);
      const comPoint = toViewPoint(inst.item?.center_of_mass);

      if (refPoint) {
        const refGeom = new THREE.SphereGeometry(markerRadiusRaw, 16, 16);
        const refMat = new THREE.MeshBasicMaterial({ color: 0xef4444 });
        const refMesh = new THREE.Mesh(refGeom, refMat);
        refMesh.position.set(refPoint.x, refPoint.y, refPoint.z);
        markers.add(refMesh);
      }

      if (comPoint) {
        const comGeom = new THREE.SphereGeometry(markerRadiusRaw, 16, 16);
        const comMat = new THREE.MeshBasicMaterial({ color: 0x22c55e });
        const comMesh = new THREE.Mesh(comGeom, comMat);
        comMesh.position.set(comPoint.x, comPoint.y, comPoint.z);
        markers.add(comMesh);
      }

      if (markers.children.length > 0) {
        inst.markers = markers;
        inst.group.add(markers);
      }

      const scaledSize = Math.max(size * scaleFactor, 0.2);
      const radius = Math.max(scaledSize * 0.5, 0.1);
      const fov = (inst.camera.fov * Math.PI) / 180;
      const distance = radius / Math.tan(fov / 2);
      if (inst.controls) {
        inst.controls.target.set(0, 0, 0);
        inst.camera.position.set(0, 0.4, Math.max(distance * 1.15, 0.5));
        inst.controls.update();
      } else if (inst.orbit) {
        inst.orbit.target.set(0, 0, 0);
        inst.orbit.radius = Math.max(distance * 1.15, 0.5);
        inst.orbit.theta = Math.PI * 0.65;
        inst.orbit.phi = Math.PI * 0.38;
        updateCameraFromOrbit(inst);
      }
    }

    // Meta removed (duplicate of header info)
  }

  function destroyInstance(inst) {
    if (inst.frameId) {
      cancelAnimationFrame(inst.frameId);
      inst.frameId = null;
    }
    if (inst.mesh) {
      inst.mesh.geometry?.dispose?.();
      inst.mesh.material?.dispose?.();
    }
    if (inst.lines) {
      inst.lines.geometry?.dispose?.();
      inst.lines.material?.dispose?.();
    }
    if (inst.controls?.dispose) {
      inst.controls.dispose();
    }
    if (inst.renderer) {
      inst.renderer.dispose();
      if (inst.renderer.domElement?.parentNode) {
        inst.renderer.domElement.parentNode.removeChild(inst.renderer.domElement);
      }
    }
  }

  function resizeInstance(inst) {
    if (!inst.renderer || !inst.camera) return;
    const width = inst.container.clientWidth || 1;
    const height = inst.container.clientHeight || 1;
    inst.renderer.setSize(width, height);
    inst.camera.aspect = width / height;
    inst.camera.updateProjectionMatrix();
    if (inst.controls) {
      inst.controls.update();
    } else {
      updateCameraFromOrbit(inst);
    }
  }

  // --- Public API ---

  function initInlineViewers() {
    const viewers = document.querySelectorAll(".inline-geometry-viewer");
    viewers.forEach((viewer) => {
      const kind = viewer.dataset.geomKind;
      const index = parseInt(viewer.dataset.geomIndex);
      const key = `${kind}-${index}`;

      // Already initialized
      if (instances.has(key)) return;

      const db = getCurrentData()?.database;
      const list =
        kind === "frame"
          ? db?.frames || []
          : kind === "box"
            ? db?.box_types || []
            : [];
      const item = list[index];
      const caseGeometry = item?.case_geometry;
      if (!hasGeometryData(caseGeometry)) return;

      const canvas = viewer.querySelector(".inline-geometry-canvas");
      if (!canvas) return;

      const inst = createInstance(canvas, caseGeometry, item);
      if (!inst) return;
      instances.set(key, inst);

      // Wire controls
      const facesCheck = viewer.querySelector(".inline-geom-faces");
      const edgesCheck = viewer.querySelector(".inline-geom-edges");
      const autoCheck = viewer.querySelector(".inline-geom-autorotate");

      const onToggle = () => {
        populateGeometry(
          inst,
          facesCheck?.checked ?? true,
          edgesCheck?.checked ?? true,
        );
      };
      if (facesCheck) facesCheck.addEventListener("change", onToggle);
      if (edgesCheck) edgesCheck.addEventListener("change", onToggle);
      if (autoCheck) {
        autoCheck.addEventListener("change", (e) => {
          if (inst.controls) {
            inst.controls.autoRotate = !!e.target.checked;
          } else {
            inst.group.userData.autoRotate = !!e.target.checked;
          }
        });
      }
    });
  }

  function handleGeometryResize() {
    for (const inst of instances.values()) {
      resizeInstance(inst);
    }
  }

  function resetGeometry() {
    for (const inst of instances.values()) {
      destroyInstance(inst);
    }
    instances.clear();
  }

  // --- Geometry building (unchanged logic) ---

  function buildCaseGeometryData(geometry, options) {
    if (!geometry) return null;

    const scale = 1;
    const bounds = {
      minX: Infinity,
      minY: Infinity,
      minZ: Infinity,
      maxX: -Infinity,
      maxY: -Infinity,
      maxZ: -Infinity,
    };
    let hasBounds = false;

    const addBounds = (point) => {
      if (!point) return;
      hasBounds = true;
      bounds.minX = Math.min(bounds.minX, point.x);
      bounds.minY = Math.min(bounds.minY, point.y);
      bounds.minZ = Math.min(bounds.minZ, point.z);
      bounds.maxX = Math.max(bounds.maxX, point.x);
      bounds.maxY = Math.max(bounds.maxY, point.y);
      bounds.maxZ = Math.max(bounds.maxZ, point.z);
    };

    let lineGeometry = null;
    if (options?.showEdges) {
      const edges = Array.isArray(geometry.edges) ? geometry.edges : [];
      const vertices = Array.isArray(geometry.vertices) ? geometry.vertices : [];
      const edgePairs =
        edges.length > 0
          ? edges.map((edge) => ({
              v1: edge.v1,
              v2: edge.v2,
              color: edge.color,
            }))
          : buildSequentialEdgePairs(vertices.length).map(([v1, v2]) => ({
              v1,
              v2,
              color: null,
            }));

      const positions = [];
      const colors = [];
      const color = new THREE.Color();

      edgePairs.forEach((edge) => {
        const p1 = toViewPoint(
          resolveGeometryVertex(geometry, vertices, edge.v1, scale),
        );
        const p2 = toViewPoint(
          resolveGeometryVertex(geometry, vertices, edge.v2, scale),
        );
        if (!p1 || !p2) return;
        positions.push(p1.x, p1.y, p1.z, p2.x, p2.y, p2.z);
        addBounds(p1);
        addBounds(p2);
        const edgeColor = colorFromInt(edge.color, themeColors.edgeFallback, color);
        colors.push(
          edgeColor.r,
          edgeColor.g,
          edgeColor.b,
          edgeColor.r,
          edgeColor.g,
          edgeColor.b,
        );
      });

      if (positions.length > 0) {
        lineGeometry = new THREE.BufferGeometry();
        lineGeometry.setAttribute(
          "position",
          new THREE.Float32BufferAttribute(positions, 3),
        );
        lineGeometry.setAttribute(
          "color",
          new THREE.Float32BufferAttribute(colors, 3),
        );
      }
    }

    let meshGeometry = null;
    if (options?.showFaces && Array.isArray(geometry.faces)) {
      const faces = geometry.faces;
      const vertices = Array.isArray(geometry.vertices) ? geometry.vertices : [];
      const positions = [];
      const colors = [];
      const color = new THREE.Color();

      faces.forEach((face) => {
        const indices = Array.isArray(face.vertices) ? face.vertices : [];
        if (indices.length < 3) return;
        const faceColor = colorFromInt(face.color, themeColors.faceFallback, color);
        for (let i = 1; i + 1 < indices.length; i += 1) {
          const p0 = toViewPoint(
            resolveGeometryVertex(geometry, vertices, indices[0], scale),
          );
          const p1 = toViewPoint(
            resolveGeometryVertex(geometry, vertices, indices[i], scale),
          );
          const p2 = toViewPoint(
            resolveGeometryVertex(geometry, vertices, indices[i + 1], scale),
          );
          if (!p0 || !p1 || !p2) continue;
          positions.push(
            p0.x,
            p0.y,
            p0.z,
            p1.x,
            p1.y,
            p1.z,
            p2.x,
            p2.y,
            p2.z,
          );
          addBounds(p0);
          addBounds(p1);
          addBounds(p2);
          for (let c = 0; c < 3; c += 1) {
            colors.push(faceColor.r, faceColor.g, faceColor.b);
          }
        }
      });

      if (positions.length > 0) {
        meshGeometry = new THREE.BufferGeometry();
        meshGeometry.setAttribute(
          "position",
          new THREE.Float32BufferAttribute(positions, 3),
        );
        meshGeometry.setAttribute(
          "color",
          new THREE.Float32BufferAttribute(colors, 3),
        );
        meshGeometry.computeVertexNormals();
      }
    }

    if (!hasBounds) {
      return null;
    }

    return {
      bounds,
      size: Math.max(
        bounds.maxX - bounds.minX,
        bounds.maxY - bounds.minY,
        bounds.maxZ - bounds.minZ,
      ),
      lines: lineGeometry,
      mesh: meshGeometry,
    };
  }

  function colorFromInt(value, fallback, target) {
    const color = target || new THREE.Color();
    const fallbackColor = new THREE.Color(fallback ?? 0x64748b);
    if (!Number.isFinite(value)) {
      color.copy(fallbackColor);
      return color;
    }
    const raw = value >>> 0;
    const r = (raw >> 16) & 0xff;
    const g = (raw >> 8) & 0xff;
    const b = raw & 0xff;
    color.setRGB(r / 255, g / 255, b / 255);
    return color;
  }

  return {
    initInlineViewers,
    handleGeometryResize,
    resetGeometry,
    updateTheme() {
      themeColors = readThemeColors();
      instances.forEach((inst) => {
        applyGridTheme(inst.grid, themeColors);
        if (inst.mesh || inst.lines) {
          populateGeometry(
            inst,
            inst.showFaces ?? true,
            inst.showEdges ?? true,
          );
        }
      });
    },
  };
}
