const encoder = new TextEncoder();
const decoder = new TextDecoder();

export function normalizeProjectPath(value) {
  const parts = [];
  for (const raw of String(value || "").replaceAll("\\", "/").split("/")) {
    const part = raw.trim();
    if (!part || part === ".") continue;
    if (part === "..") {
      if (!parts.length) throw new Error("Project files cannot escape the workspace.");
      parts.pop();
      continue;
    }
    if (/\0/.test(part)) throw new Error("Project file names cannot contain NUL bytes.");
    parts.push(part);
  }
  if (!parts.length) throw new Error("Enter a project file name.");
  return parts.join("/");
}

export function encodeProjectZip(files) {
  const entries = Object.entries(files).map(([name, source]) => ({
    name: normalizeProjectPath(name),
    nameBytes: encoder.encode(normalizeProjectPath(name)),
    data: typeof source === "string" ? encoder.encode(source) : new Uint8Array(source),
  })).sort((left, right) => left.name.localeCompare(right.name));
  const local = [];
  const central = [];
  let offset = 0;
  for (const entry of entries) {
    const crc = crc32(entry.data);
    const header = new Uint8Array(30 + entry.nameBytes.length);
    const view = new DataView(header.buffer);
    view.setUint32(0, 0x04034b50, true);
    view.setUint16(4, 20, true);
    view.setUint16(6, 0x0800, true);
    view.setUint32(14, crc, true);
    view.setUint32(18, entry.data.length, true);
    view.setUint32(22, entry.data.length, true);
    view.setUint16(26, entry.nameBytes.length, true);
    header.set(entry.nameBytes, 30);
    local.push(header, entry.data);

    const directory = new Uint8Array(46 + entry.nameBytes.length);
    const directoryView = new DataView(directory.buffer);
    directoryView.setUint32(0, 0x02014b50, true);
    directoryView.setUint16(4, 20, true);
    directoryView.setUint16(6, 20, true);
    directoryView.setUint16(8, 0x0800, true);
    directoryView.setUint32(16, crc, true);
    directoryView.setUint32(20, entry.data.length, true);
    directoryView.setUint32(24, entry.data.length, true);
    directoryView.setUint16(28, entry.nameBytes.length, true);
    directoryView.setUint32(42, offset, true);
    directory.set(entry.nameBytes, 46);
    central.push(directory);
    offset += header.length + entry.data.length;
  }
  const centralSize = central.reduce((sum, value) => sum + value.length, 0);
  const end = new Uint8Array(22);
  const endView = new DataView(end.buffer);
  endView.setUint32(0, 0x06054b50, true);
  endView.setUint16(8, entries.length, true);
  endView.setUint16(10, entries.length, true);
  endView.setUint32(12, centralSize, true);
  endView.setUint32(16, offset, true);
  return concatBytes([...local, ...central, end]);
}

export function decodeProjectZip(source) {
  const bytes = source instanceof Uint8Array ? source : new Uint8Array(source);
  const files = {};
  let at = 0;
  while (at + 4 <= bytes.length) {
    const view = new DataView(bytes.buffer, bytes.byteOffset + at, bytes.length - at);
    const signature = view.getUint32(0, true);
    if (signature !== 0x04034b50) break;
    if (at + 30 > bytes.length) throw new Error("The project ZIP is truncated.");
    const flags = view.getUint16(6, true);
    const method = view.getUint16(8, true);
    const size = view.getUint32(18, true);
    const nameLength = view.getUint16(26, true);
    const extraLength = view.getUint16(28, true);
    if (flags & 0x0008) throw new Error("ZIP data descriptors are not supported; export the project without compression.");
    if (method !== 0) throw new Error("Compressed ZIP entries are not supported yet; use a Renvo-exported ZIP or import a directory.");
    const dataAt = at + 30 + nameLength + extraLength;
    const end = dataAt + size;
    if (end > bytes.length) throw new Error("The project ZIP is truncated.");
    const name = normalizeProjectPath(decoder.decode(bytes.subarray(at + 30, at + 30 + nameLength)));
    if (!name.endsWith("/")) files[name] = decoder.decode(bytes.subarray(dataAt, end));
    at = end;
  }
  if (!Object.keys(files).length) throw new Error("The ZIP does not contain project files.");
  return files;
}

export function encodeSharedProject(files) {
  const bytes = encoder.encode(JSON.stringify({ version: 1, files }));
  let binary = "";
  for (let at = 0; at < bytes.length; at += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(at, at + 0x8000));
  }
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}

export function decodeSharedProject(value) {
  const padded = value.replaceAll("-", "+").replaceAll("_", "/") + "===".slice((value.length + 3) % 4);
  const binary = atob(padded);
  const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
  const project = JSON.parse(decoder.decode(bytes));
  if (project?.version !== 1 || !project.files || typeof project.files !== "object") throw new Error("Unsupported shared project.");
  const files = {};
  for (const [name, source] of Object.entries(project.files)) files[normalizeProjectPath(name)] = String(source);
  return files;
}

function concatBytes(parts) {
  const result = new Uint8Array(parts.reduce((sum, value) => sum + value.length, 0));
  let at = 0;
  for (const part of parts) { result.set(part, at); at += part.length; }
  return result;
}

function crc32(bytes) {
  let crc = 0xffffffff;
  for (const byte of bytes) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0);
  }
  return (crc ^ 0xffffffff) >>> 0;
}
