const decoder = new TextDecoder();

function uint32(data, offset, littleEndian) {
  if (offset + 4 > data.byteLength) return 0;
  return new DataView(data.buffer, data.byteOffset, data.byteLength).getUint32(offset, littleEndian);
}

export function isTextData(data) {
  const sample = data.subarray(0, Math.min(data.byteLength, 4096));
  for (const value of sample) if (value === 0 || value < 9 || value > 13 && value < 32) return false;
  return true;
}

export function describeFile(name, data) {
  if (!data) return "cannot open: No such file";
  if (data.length >= 8 && data[0] === 0x00 && data[1] === 0x61 && data[2] === 0x73 && data[3] === 0x6d) {
    return `WebAssembly (WASM) binary module, version ${uint32(data, 4, true)}`;
  }
  if (data.length >= 20 && data[0] === 0x7f && data[1] === 0x45 && data[2] === 0x4c && data[3] === 0x46) {
    const machine = new DataView(data.buffer, data.byteOffset, data.byteLength).getUint16(18, data[5] === 1);
    const machines = { 3: "Intel 80386", 40: "ARM", 62: "x86-64", 183: "AArch64", 243: "RISC-V" };
    return `ELF ${data[4] === 2 ? "64-bit" : "32-bit"} ${data[5] === 2 ? "big-endian" : "little-endian"} executable${machines[machine] ? `, ${machines[machine]}` : ""}`;
  }
  const magic = uint32(data, 0, false);
  if ([0xcffaedfe, 0xfeedfacf].includes(magic)) return "Mach-O 64-bit executable";
  if ([0xcefaedfe, 0xfeedface].includes(magic)) return "Mach-O 32-bit executable";
  if ([0xcafebabe, 0xbebafeca].includes(magic)) return "Mach-O universal binary";
  if (data.length >= 2 && data[0] === 0x4d && data[1] === 0x5a) {
    const peAt = uint32(data, 0x3c, true);
    return peAt && peAt + 4 <= data.length && uint32(data, peAt, false) === 0x50450000 ? "PE executable" : "DOS executable";
  }
  if (data.length >= 4 && data[0] === 0x50 && data[1] === 0x4b && [0x03, 0x05, 0x07].includes(data[2])) return "ZIP archive data";
  if (isTextData(data)) {
    const type = name.endsWith(".go") ? "Go source" : /\.[ch]$/.test(name) ? "C source" : name.endsWith(".json") ? "JSON text" : "UTF-8 text";
    return `${type}, ${data.byteLength} bytes`;
  }
  return `data, ${data.byteLength} bytes`;
}

export function formatHexPage(data, offset = 0, pageSize = 4096, width = 16) {
  const start = Math.max(0, Math.min(offset, Math.max(0, data.byteLength - 1)));
  const end = Math.min(data.byteLength, start + pageSize);
  const digits = Math.max(8, (data.byteLength - 1).toString(16).length);
  const lines = [];
  for (let at = start; at < end; at += width) {
    const row = data.subarray(at, Math.min(end, at + width));
    const hex = [...row].map((value) => value.toString(16).padStart(2, "0")).join(" ").padEnd(width * 3 - 1);
    const ascii = [...row].map((value) => value >= 32 && value < 127 ? String.fromCharCode(value) : ".").join("");
    lines.push(`${at.toString(16).padStart(digits, "0")}  ${hex}  |${ascii.padEnd(width)}|`);
  }
  return lines.join("\n");
}

export function textPreview(data, limit = 80) {
  return decoder.decode(data.subarray(0, Math.min(data.length, limit)));
}
