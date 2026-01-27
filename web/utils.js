// Utility Functions
// Common utilities for file operations, downloads, and string manipulation

// File Download Functions

export function downloadFile(filename, dataUri) {
  const link = document.createElement("a");
  link.href = dataUri;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

export function downloadTextFile(filename, content) {
  const blob = new Blob([content], { type: "text/plain" });
  const url = URL.createObjectURL(blob);
  downloadFile(filename, url);
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function downloadBinaryFile(filename, buffer) {
  const blob = new Blob([buffer], { type: "application/octet-stream" });
  const url = URL.createObjectURL(blob);
  downloadFile(filename, url);
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

// Filename Sanitization

export function sanitizeFilename(name) {
  return (
    String(name)
      .replace(/\\s+/g, "_")
      .replace(/[^a-zA-Z0-9_-]/g, "")
      .replace(/_+/g, "_")
      .replace(/^_+|_+$/g, "")
      || "geometry"
  );
}
