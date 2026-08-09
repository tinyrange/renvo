import assert from "node:assert/strict";
import test from "node:test";

import { ESPWebSerial, elfToESPImage, supportsESPWebSerial } from "./esp-webserial.mjs";

test("ELF conversion creates a checksummed, aligned C6 app image", async () => {
  const elf = syntheticELF(243, 0x42000100, [
    { name: ".flash.appdesc", address: 0x42010020, data: Uint8Array.from([1, 2, 3, 4]) },
    { name: ".text", address: 0x42000100, data: Uint8Array.from([0xc0, 0xdb, 5, 6]) },
  ]);
  const image = await elfToESPImage(elf, "esp32c6/riscv32");
  assert.equal(image[0], 0xe9);
  assert.equal(image[1], 3); // app descriptor, alignment padding, text
  assert.equal(new DataView(image.buffer).getUint32(4, true), 0x42000100);
  assert.equal(new DataView(image.buffer).getUint16(12, true), 13);

  let at = 24;
  let checksum = 0xef;
  for (let index = 0; index < image[1]; index++) {
    const address = new DataView(image.buffer).getUint32(at, true);
    const size = new DataView(image.buffer).getUint32(at + 4, true);
    if (address === 0x42000100) assert.equal((at + 8) & 0xffff, 0x0100);
    at += 8;
    for (const value of image.subarray(at, at + size)) checksum ^= value;
    at += size;
  }
  while ((at & 15) !== 15) at++;
  assert.equal(image[at], checksum);
  const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", image.subarray(0, at + 1)));
  assert.deepEqual(image.subarray(at + 1), digest);
});

test("ESP targets and ELF machines are checked", async () => {
  assert.equal(supportsESPWebSerial("esp32s3/xtensa_lx7"), true);
  assert.equal(supportsESPWebSerial("linux/amd64"), false);
  await assert.rejects(() => elfToESPImage(syntheticELF(94, 0x40370000, []), "esp32c6/riscv32"), /does not match/);
});

test("ROM loader synchronization works through a WebUSB-shaped stream", async () => {
  let input;
  const readable = new ReadableStream({ start(controller) { input = controller; } });
  const writable = new WritableStream({
    write() {
      input.enqueue(Uint8Array.from([0xc0, 1, 0x08, 2, 0, 0, 0, 0, 0, 0, 0, 0xc0]));
    },
  });
  const port = {
    transport: "webusb", readable, writable,
    async close() { try { input.close(); } catch {} },
  };
  const session = new ESPWebSerial(port);
  await session.open();
  await session.synchronize(100);
  await session.close();
});

test("WebUSB closes the device before cancelling its pending reader", async () => {
  let portClosed = false;
  const port = {
    transport: "webusb",
    async close() { portClosed = true; },
  };
  const session = new ESPWebSerial(port);
  session.reader = {
    async cancel() { assert.equal(portClosed, true); },
  };
  await session.close();
  assert.equal(portClosed, true);
});

function syntheticELF(machine, entry, sections) {
  const names = ["", ...sections.map((section) => section.name), ".shstrtab"];
  const nameOffsets = [];
  let namesLength = 0;
  for (const name of names) { nameOffsets.push(namesLength); namesLength += name.length + 1; }
  const sectionCount = sections.length + 2;
  const sectionTableAt = 52;
  let dataAt = sectionTableAt + sectionCount * 40;
  const dataOffsets = sections.map((section) => { const at = dataAt; dataAt += section.data.length; return at; });
  const namesAt = dataAt;
  const output = new Uint8Array(namesAt + namesLength);
  const view = new DataView(output.buffer);
  output.set([0x7f, 0x45, 0x4c, 0x46, 1, 1, 1]);
  view.setUint16(16, 2, true); view.setUint16(18, machine, true); view.setUint32(20, 1, true);
  view.setUint32(24, entry, true); view.setUint32(32, sectionTableAt, true);
  view.setUint16(40, 52, true); view.setUint16(46, 40, true);
  view.setUint16(48, sectionCount, true); view.setUint16(50, sectionCount - 1, true);
  sections.forEach((section, index) => {
    const at = sectionTableAt + (index + 1) * 40;
    view.setUint32(at, nameOffsets[index + 1], true); view.setUint32(at + 4, 1, true);
    view.setUint32(at + 8, 2, true); view.setUint32(at + 12, section.address, true);
    view.setUint32(at + 16, dataOffsets[index], true); view.setUint32(at + 20, section.data.length, true);
    output.set(section.data, dataOffsets[index]);
  });
  const namesHeader = sectionTableAt + (sectionCount - 1) * 40;
  view.setUint32(namesHeader, nameOffsets[nameOffsets.length - 1], true);
  view.setUint32(namesHeader + 4, 3, true); view.setUint32(namesHeader + 16, namesAt, true);
  view.setUint32(namesHeader + 20, namesLength, true);
  let at = namesAt;
  for (const name of names) { output.set(new TextEncoder().encode(name), at); at += name.length + 1; }
  return output;
}
