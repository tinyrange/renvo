import {
  ESPRESSIF_VENDOR_ID, USB_SERIAL_JTAG_PRODUCT_ID,
  selectAuthorizedESPUSBDevice,
} from "./esp-webusb.mjs";

const JTAG_IR_IDCODE = 0x01;
const JTAG_IR_DTMCS = 0x10;
const JTAG_IR_DMI = 0x11;

const DMI_DATA0 = 0x04;
const DMI_DMCONTROL = 0x10;
const DMI_DMSTATUS = 0x11;
const DMI_ABSTRACTCS = 0x16;
const DMI_COMMAND = 0x17;
const DMI_PROGRAM0 = 0x20;
const DMI_PROGRAM1 = 0x21;
const DMI_SBCS = 0x38;
const DMI_SBADDRESS0 = 0x39;
const DMI_SBDATA0 = 0x3c;

const delay = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));

export function supportsESPWebUSBJTAG(target) {
  return target === "esp32c6/riscv32" || target === "esp32c6-jtag/riscv32";
}

export async function requestESPUSBJTAG(target) {
  if (!supportsESPWebUSBJTAG(target)) throw new Error(`JTAG WebUSB does not support ${target}`);
  if (!globalThis.navigator?.usb) {
    throw new Error("WebUSB is unavailable. Open Renvo over HTTPS in a Chromium-based browser.");
  }
  const permitted = await navigator.usb.getDevices?.() || [];
  let device = selectAuthorizedESPUSBDevice(permitted);
  if (!device) {
    device = await navigator.usb.requestDevice({ filters: [{
      vendorId: ESPRESSIF_VENDOR_ID,
      productId: USB_SERIAL_JTAG_PRODUCT_ID,
    }] });
  }
  return new ESPWebUSBJTAG(device, navigator.usb);
}

// ESPWebUSBJTAG ports the repository's direct ESP32-C6 USB/JTAG debugger to
// WebUSB. It claims only the vendor JTAG interface, so desktop CDC drivers may
// continue owning interfaces 0 and 1 without blocking deployment.
export class ESPWebUSBJTAG {
  constructor(device, usb = globalThis.navigator?.usb, options = {}) {
    this.transport = "webusb-jtag";
    this.device = device;
    this.usb = usb;
    this.initializeTarget = options.initialize || ((debug) => debug.initialize());
    this.opened = false;
    this.disconnected = false;
    this.abits = 0;
    this.idleClocks = 0;
    this.handleDisconnect = (event) => {
      if (event.device !== this.device) return;
      this.disconnected = true;
      this.opened = false;
    };
    this.usb?.addEventListener?.("disconnect", this.handleDisconnect);
  }

  getInfo() {
    return { usbVendorId: this.device.vendorId, usbProductId: this.device.productId };
  }

  canReopen() { return !this.disconnected; }

  diagnostics() {
    return `WebUSB JTAG interface ${this.jtag?.iface?.interfaceNumber ?? "?"}, endpoints OUT ${this.jtag?.bulkOut?.endpointNumber ?? "?"}/IN ${this.jtag?.bulkIn?.endpointNumber ?? "?"}`;
  }

  async open() {
    if (this.opened) return;
    this.disconnected = false;
    try {
      if (!this.device.opened) await this.device.open();
      const selected = findJTAGConfiguration(this.device.configurations);
      if (!selected) throw new Error("device does not expose the ESP vendor JTAG interface");
      if (this.device.configuration?.configurationValue !== selected.configuration.configurationValue) {
        await this.device.selectConfiguration(selected.configuration.configurationValue);
      }
      this.jtag = findJTAGConfiguration([this.device.configuration]) || selected;
      await this.device.claimInterface(this.jtag.iface.interfaceNumber);
      if (this.jtag.alternate.alternateSetting !== 0) {
        await this.device.selectAlternateInterface(this.jtag.iface.interfaceNumber, this.jtag.alternate.alternateSetting);
      }
      this.opened = true;
      await this.initializeTarget(this);
    } catch (error) {
      await this.close();
      throw new Error(`could not open ESP WebUSB JTAG transport: ${error.message || error}`);
    }
  }

  async close() {
    this.opened = false;
    this.abits = 0;
    this.idleClocks = 0;
    try { if (this.device.opened) await this.device.close(); } catch {}
    this.usb?.removeEventListener?.("disconnect", this.handleDisconnect);
  }

  async initialize() {
    const commands = new JTAGCommands();
    for (let index = 0; index < 6; index++) commands.clock(0, 0, 1);
    commands.clock(0, 0, 0);
    await this.execute(commands);
    const id = await this.scanDR(0n, 32, 32);
    if (Number(id) !== 0x0000dc25) throw new Error(`JTAG TAP ID 0x${Number(id).toString(16)} is not ESP32-C6`);
    await this.scanIR(JTAG_IR_DTMCS);
    const dtmcs = Number(await this.scanDR(0n, 32, 32));
    if ((dtmcs & 15) !== 1) throw new Error(`unsupported RISC-V debug transport version ${dtmcs & 15}`);
    this.abits = (dtmcs >>> 4) & 0x3f;
    this.idleClocks = (dtmcs >>> 12) & 7;
    if (this.abits < 6 || this.abits > 16) throw new Error(`invalid RISC-V DMI address width ${this.abits}`);
    await this.scanIR(JTAG_IR_DTMCS);
    await this.scanDR(1n << 16n, 32, 0);
    await this.scanIR(JTAG_IR_DMI);
    await this.dmiRead(DMI_DMSTATUS);
  }

  async halt() {
    // A freshly enumerated C6 commonly has allhavereset/anyhavereset latched.
    // Acknowledge that state together with haltreq or the hart may continue
    // running while dmstatus remains 0xc0ca2.
    await this.dmiWrite(DMI_DMCONTROL, 0x90000001);
    let lastStatus = 0;
    for (let attempt = 0; attempt < 100; attempt++) {
      lastStatus = await this.dmiRead(DMI_DMSTATUS);
      if ((lastStatus & (1 << 9)) !== 0) return;
      await delay(1);
    }
    throw new Error(`ESP32-C6 did not halt (dmstatus 0x${lastStatus.toString(16)})`);
  }

  async resume() {
    await this.dmiWrite(DMI_DMCONTROL, 0x40000001);
    for (let attempt = 0; attempt < 100; attempt++) {
      const status = await this.dmiRead(DMI_DMSTATUS);
      if ((status & (1 << 17)) !== 0) {
        await this.dmiWrite(DMI_DMCONTROL, 1);
        return;
      }
      await delay(1);
    }
    throw new Error("ESP32-C6 did not resume");
  }

  async setPC(address) { await this.writeAbstractRegister(0x7b1, address); }

  async fenceI() {
    await this.dmiWrite(DMI_PROGRAM0, 0x0000100f);
    await this.dmiWrite(DMI_PROGRAM1, 0x00100073);
    await this.dmiWrite(DMI_COMMAND, 0x00240000);
    await this.waitAbstractCommand();
  }

  async writeAbstractRegister(register, value) {
    await this.dmiWrite(DMI_DATA0, value);
    await this.dmiWrite(DMI_COMMAND, 0x00230000 | register);
    await this.waitAbstractCommand();
  }

  async waitAbstractCommand() {
    for (let attempt = 0; attempt < 100; attempt++) {
      const state = await this.dmiRead(DMI_ABSTRACTCS);
      if ((state & (1 << 12)) !== 0) continue;
      const commandError = (state >>> 8) & 7;
      if (commandError !== 0) {
        await this.dmiWrite(DMI_ABSTRACTCS, 7 << 8);
        throw new Error(`RISC-V abstract command failed (cmderr ${commandError})`);
      }
      return;
    }
    throw new Error("RISC-V abstract command timed out");
  }

  async writeMemory(address, source) {
    const data = bytes(source);
    if (data.length === 0) return;
    if ((address & 3) !== 0 || (data.length & 3) !== 0) throw new Error("JTAG memory writes must be word aligned");
    if (data.length > 0xffffffff - address) throw new Error("JTAG memory write range overflows the address space");
    const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
    const wordsPerBatch = 128;
    for (let batchAt = 0; batchAt < data.length; batchAt += wordsPerBatch * 4) {
      const batchEnd = Math.min(data.length, batchAt + wordsPerBatch * 4);
      const requests = [
        { address: DMI_SBCS, data: 0x00050000, operation: 2 },
        { address: DMI_SBADDRESS0, data: (address + batchAt) >>> 0, operation: 2 },
      ];
      for (let offset = batchAt; offset < batchEnd; offset += 4) {
        requests.push({ address: DMI_SBDATA0, data: view.getUint32(offset, true), operation: 2 });
      }
      await this.dmiBatch(requests);
    }
    const state = await this.dmiRead(DMI_SBCS);
    if ((state & (1 << 22)) !== 0 || (state & (7 << 12)) !== 0) {
      await this.dmiWrite(DMI_SBCS, (1 << 22) | (7 << 12));
      throw new Error("RISC-V system-bus memory write failed");
    }
  }

  async dmiWrite(address, data) {
    await this.dmiBatch([{ address, data, operation: 2 }]);
  }

  async dmiRead(address) {
    const request = BigInt(address) << 34n | 1n;
    let lastResponse = 0n;
    for (let attempt = 0; attempt < 100; attempt++) {
      await this.scanIR(JTAG_IR_DMI);
      const bits = 34 + this.abits;
      const commands = new JTAGCommands();
      commands.scanDR(request, bits, bits);
      const idleClocks = Math.max(1, this.idleClocks + attempt);
      for (let idle = 0; idle < idleClocks; idle++) commands.clock(0, 0, 0);
      commands.scanDR(0n, bits, bits);
      const responseData = await this.execute(commands);
      const response = capturedBits(responseData, bits, bits);
      lastResponse = response;
      const status = Number(response & 3n);
      if (status === 0) {
        if (idleClocks > this.idleClocks) this.idleClocks = idleClocks;
        return Number(response >> 2n) >>> 0;
      }
      if (status === 3) {
        await this.resetDMI();
        continue;
      }
      throw new Error("RISC-V DMI read failed");
    }
    throw new Error(`RISC-V DMI remained busy (last response data 0x${Number(lastResponse >> 2n).toString(16)})`);
  }

  async resetDMI() {
    await this.scanIR(JTAG_IR_DTMCS);
    await this.scanDR(1n << 16n, 32, 0);
    await this.scanIR(JTAG_IR_DMI);
  }

  async dmiBatch(requests) {
    if (!requests.length) return;
    for (let attempt = 0; attempt < 100; attempt++) {
      await this.scanIR(JTAG_IR_DMI);
      const commands = new JTAGCommands();
      for (let index = 0; index <= requests.length; index++) {
        let value = 0n;
        if (index < requests.length) {
          const request = requests[index];
          value = BigInt(request.address) << 34n |
            BigInt(request.data >>> 0) << 2n | BigInt(request.operation);
        }
        commands.scanDR(value, 34 + this.abits, 2);
        for (let idle = 0; idle < Math.max(1, this.idleClocks); idle++) commands.clock(0, 0, 0);
      }
      const response = await this.execute(commands);
      let busy = false;
      for (let index = 1; index <= requests.length; index++) {
        const status = Number(capturedBits(response, index * 2, 2));
        if (status === 3) { busy = true; break; }
        if (status !== 0) throw new Error("RISC-V DMI write failed");
      }
      if (!busy) return;
      await this.resetDMI();
      this.idleClocks++;
    }
    throw new Error("RISC-V DMI remained busy during batched write");
  }

  async scanIR(value) {
    const commands = new JTAGCommands();
    commands.clock(0, 0, 1);
    commands.clock(0, 0, 1);
    commands.clock(0, 0, 0);
    commands.clock(0, 0, 0);
    commands.shift(BigInt(value), 5, 5);
    commands.clock(0, 0, 1);
    commands.clock(0, 0, 0);
    return capturedBits(await this.execute(commands), 0, 5);
  }

  async scanDR(value, bitCount, captureCount) {
    const commands = new JTAGCommands();
    commands.scanDR(value, bitCount, captureCount);
    return capturedBits(await this.execute(commands), 0, captureCount);
  }

  async execute(commands) {
    if (!this.opened) throw new Error("WebUSB JTAG device is closed");
    commands.nibble(0x0a);
    if ((commands.nibbles & 1) !== 0) commands.nibble(0x0a);
    let written = 0;
    const output = Uint8Array.from(commands.data);
    while (written < output.length) {
      const result = await this.device.transferOut(this.jtag.bulkOut.endpointNumber, output.subarray(written));
      if (result.status === "stall") {
        await this.device.clearHalt("out", this.jtag.bulkOut.endpointNumber);
        continue;
      }
      if (result.status !== "ok") throw new Error(`USB/JTAG OUT transfer returned ${result.status}`);
      if (!result.bytesWritten) throw new Error("USB/JTAG write made no progress");
      written += result.bytesWritten;
    }
    const expected = Math.ceil(commands.captures / 8);
    const response = new Uint8Array(expected);
    let offset = 0;
    let empty = 0;
    while (offset < expected) {
      const length = Math.max(this.jtag.bulkIn.packetSize || 64, expected - offset);
      const result = await this.device.transferIn(this.jtag.bulkIn.endpointNumber, length);
      if (result.status === "stall") {
        await this.device.clearHalt("in", this.jtag.bulkIn.endpointNumber);
        continue;
      }
      if (result.status !== "ok") throw new Error(`USB/JTAG IN transfer returned ${result.status}`);
      const part = result.data ? new Uint8Array(result.data.buffer, result.data.byteOffset, result.data.byteLength) : new Uint8Array();
      if (!part.length) {
        if (++empty >= 100) throw new Error("USB/JTAG returned an empty response");
        await delay(1);
        continue;
      }
      empty = 0;
      const count = Math.min(part.length, expected - offset);
      response.set(part.subarray(0, count), offset);
      offset += count;
    }
    return response;
  }
}

export class ESPJTAGHotReloadSession {
  constructor(debug, { progress = () => {} } = {}) {
    this.debug = debug;
    this.progress = progress;
    this.previous = undefined;
  }

  reset() { this.previous = undefined; }

  async update(elf) {
    await this.debug.open?.();
    const next = parseJTAGLoadImage(elf);
    const patches = planJTAGPatches(this.previous, next);
    const report = { entry: next.entry, patchCount: 0, bytesWritten: 0, unchanged: false };
    if (!patches.length && this.previous?.entry === next.entry) {
      report.unchanged = true;
      return report;
    }
    const total = patches.reduce((sum, patch) => sum + patch.data.length, 0);
    await this.debug.halt();
    for (const patch of patches) {
      await this.debug.writeMemory(patch.address, patch.data);
      report.patchCount++;
      report.bytesWritten += patch.data.length;
      this.progress(total ? report.bytesWritten / total : 1);
    }
    await this.debug.fenceI();
    await this.debug.setPC(next.entry);
    await this.debug.resume();
    this.previous = cloneLoadImage(next);
    return report;
  }

  async close() { await this.debug.close?.(); }
}

export function parseJTAGLoadImage(source) {
  const data = bytes(source);
  if (data.length < 52 || data[0] !== 0x7f || data[1] !== 0x45 || data[2] !== 0x4c || data[3] !== 0x46 || data[4] !== 1 || data[5] !== 1) {
    throw new Error("JTAG hot reload requires a little-endian ELF32 file");
  }
  const view = new DataView(data.buffer, data.byteOffset, data.byteLength);
  if (view.getUint16(18, true) !== 243) throw new Error("JTAG hot reload requires a RISC-V ELF");
  const entry = view.getUint32(24, true);
  if (!c6RAMAddress(entry, 1)) throw new Error("ELF entry is not in ESP32-C6 SRAM; compile with target esp32c6-jtag/riscv32");
  const programAt = view.getUint32(28, true);
  const programSize = view.getUint16(42, true);
  const programCount = view.getUint16(44, true);
  if (programSize < 32 || programCount < 1 || programAt + programSize * programCount > data.length) {
    throw new Error("ELF program table is invalid");
  }
  const segments = [];
  for (let index = 0; index < programCount; index++) {
    const at = programAt + index * programSize;
    if (view.getUint32(at, true) !== 1) continue;
    const offset = view.getUint32(at + 4, true);
    const address = view.getUint32(at + 8, true);
    const fileSize = view.getUint32(at + 16, true);
    const memorySize = view.getUint32(at + 20, true);
    const flags = view.getUint32(at + 24, true);
    if (memorySize < fileSize || offset + fileSize > data.length) throw new Error("ELF load segment is invalid");
    if (!memorySize) continue;
    if (!c6RAMAddress(address, memorySize)) throw new Error("ELF contains a load segment outside ESP32-C6 SRAM");
    segments.push({
      address, memorySize, executable: (flags & 1) !== 0, writable: (flags & 2) !== 0,
      data: data.slice(offset, offset + fileSize),
    });
  }
  if (!segments.length) throw new Error("ELF has no loadable segments");
  segments.sort((left, right) => left.address - right.address);
  for (let index = 1; index < segments.length; index++) {
    if (segments[index].address < segments[index - 1].address + segments[index - 1].memorySize) {
      throw new Error("ELF load segments overlap");
    }
  }
  return { entry, segments };
}

export function planJTAGPatches(previous, next) {
  const patches = [];
  for (const segment of next.segments) {
    const dataSize = (segment.data.length + 3) & ~3;
    if (!previous) {
      if (dataSize) {
        const data = new Uint8Array(dataSize);
        data.set(segment.data);
        patches.push({ address: segment.address, data });
      }
      continue;
    }
    let start = -1;
    let lastChanged = -1;
    for (let offset = 0; offset < dataSize; offset += 4) {
      let changed = false;
      for (let byteIndex = 0; byteIndex < 4; byteIndex++) {
        const at = offset + byteIndex;
        const nextByte = at < segment.data.length ? segment.data[at] : 0;
        const old = loadImageByte(previous, segment.address + at);
        if (!old.found || old.value !== nextByte) changed = true;
      }
      if (changed) {
        if (start < 0) start = offset;
        else if (lastChanged >= 0 && offset - lastChanged > 20) {
          appendSegmentPatch(patches, segment, start, lastChanged + 4);
          start = offset;
        }
        lastChanged = offset;
      }
    }
    if (start >= 0) appendSegmentPatch(patches, segment, start, lastChanged + 4);
  }
  return patches;
}

class JTAGCommands {
  constructor() { this.data = []; this.nibbles = 0; this.captures = 0; }
  nibble(value) {
    if ((this.nibbles & 1) === 0) this.data.push((value & 15) << 4);
    else this.data[this.data.length - 1] |= value & 15;
    this.nibbles++;
  }
  clock(capture, tdi, tms) {
    this.nibble((capture & 1) << 2 | (tms & 1) << 1 | (tdi & 1));
    if (capture) this.captures++;
  }
  shift(value, bitCount, captureCount) {
    for (let bit = 0; bit < bitCount; bit++) {
      this.clock(bit < captureCount ? 1 : 0, Number(value >> BigInt(bit) & 1n), bit + 1 === bitCount ? 1 : 0);
    }
  }
  scanDR(value, bitCount, captureCount) {
    this.clock(0, 0, 1);
    this.clock(0, 0, 0);
    this.clock(0, 0, 0);
    this.shift(value, bitCount, captureCount);
    this.clock(0, 0, 1);
    this.clock(0, 0, 0);
  }
}

function findJTAGConfiguration(configurations) {
  for (const configuration of configurations || []) {
    for (const iface of configuration.interfaces || []) {
      for (const alternate of iface.alternates || []) {
        const bulkIn = alternate.endpoints?.find((endpoint) => endpoint.type === "bulk" && endpoint.direction === "in");
        const bulkOut = alternate.endpoints?.find((endpoint) => endpoint.type === "bulk" && endpoint.direction === "out");
        if (alternate.interfaceClass === 0xff && alternate.interfaceSubclass === 0xff && alternate.interfaceProtocol === 1 && bulkIn && bulkOut) {
          return { configuration, iface, alternate, bulkIn, bulkOut };
        }
      }
    }
  }
  return undefined;
}

function c6RAMAddress(address, size) {
  return size > 0 && address >= 0x40800000 && address < 0x40880000 && size <= 0x40880000 - address;
}

function appendSegmentPatch(patches, segment, start, end) {
  const data = new Uint8Array(end - start);
  if (start < segment.data.length) data.set(segment.data.subarray(start, Math.min(end, segment.data.length)));
  patches.push({ address: segment.address + start, data });
}

function loadImageByte(image, address) {
  for (const segment of image?.segments || []) {
    const paddedSize = (segment.data.length + 3) & ~3;
    if (address >= segment.address && address < segment.address + paddedSize) {
      const offset = address - segment.address;
      return { value: offset < segment.data.length ? segment.data[offset] : 0, found: true };
    }
  }
  return { value: 0, found: false };
}

function cloneLoadImage(image) {
  return { entry: image.entry, segments: image.segments.map((segment) => ({ ...segment, data: segment.data.slice() })) };
}

function capturedBits(data, offset, count) {
  let result = 0n;
  for (let bit = 0; bit < count; bit++) {
    const at = offset + bit;
    if (at >> 3 < data.length && (data[at >> 3] & 1 << (at & 7)) !== 0) result |= 1n << BigInt(bit);
  }
  return result;
}

function bytes(value) {
  return value instanceof Uint8Array ? value : new Uint8Array(value);
}
