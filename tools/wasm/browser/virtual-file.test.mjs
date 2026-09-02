import assert from "node:assert/strict";
import test from "node:test";

import { describeFile, formatHexPage, isTextData } from "./virtual-file.mjs";

test("file descriptions identify WASM and native executable formats", () => {
  assert.equal(describeFile("app.wasm", Uint8Array.from([0, 0x61, 0x73, 0x6d, 1, 0, 0, 0])), "WebAssembly (WASM) binary module, version 1");
  assert.equal(describeFile("app", Uint8Array.from([0xcf, 0xfa, 0xed, 0xfe])), "Mach-O 64-bit executable");
  assert.equal(describeFile("readme.txt", new TextEncoder().encode("hello\n")), "UTF-8 text, 6 bytes");
});

test("hex pages include offsets, bytes, and printable characters", () => {
  const output = formatHexPage(Uint8Array.from([0, 0x41, 0x42, 0xff]));
  assert.match(output, /^00000000  00 41 42 ff/);
  assert.match(output, /\|\.AB\./);
  assert.equal(isTextData(Uint8Array.from([0, 1, 2])), false);
});
