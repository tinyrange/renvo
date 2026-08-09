// First-party ESP ROM-loader support for Renvo. This module has no runtime
// dependencies and contains no third-party source. Its wire behavior was
// checked against Espressif's Apache-2.0 esptool-js at commit c2956c5:
// https://github.com/espressif/esptool-js/tree/c2956c5aac35d7ef24d614b734590263662eb8d8
// Protocol and image layout are also documented by Espressif at:
// https://docs.espressif.com/projects/esptool/en/latest/esp32h2/advanced-topics/serial-protocol.html
// https://docs.espressif.com/projects/esptool/en/release-v4/esp32/advanced-topics/firmware-image-format.html

const FLASH_OFFSET = 0x10000;
const FLASH_BLOCK = 0x400;
const CHECKSUM_MAGIC = 0xef;
const encoder = new TextEncoder();
const decoder = new TextDecoder();

const targets = {
  "esp32c6/riscv32": {
    chipID: 13, machine: 243, flashConfig: 0x20, magic: 0x2ce0806f,
    flashAddress(address) { return address >= 0x42000000 && address < 0x43000000; },
  },
  "esp32s3/xtensa_lx7": {
    chipID: 9, machine: 94, flashConfig: 0x3f, magic: 0x09,
    flashAddress(address) {
      return (address >= 0x3c000000 && address < 0x3d000000) ||
        (address >= 0x42000000 && address < 0x44000000);
    },
  },
};

export function supportsESPWebSerial(target) { return Boolean(targets[target]); }

export async function requestESPPort(target) {
  if (!supportsESPWebSerial(target)) throw new Error(`unsupported ESP target ${target}`);
  if (!globalThis.navigator?.serial) {
    throw new Error("WebSerial is unavailable. Open Renvo over HTTPS in a Chromium-based browser.");
  }
  return navigator.serial.requestPort({ filters: [{ usbVendorId: 0x303a }] });
}

export async function elfToESPImage(elfSource, targetName) {
  const target = targets[targetName];
  if (!target) throw new Error(`unsupported ESP target ${targetName}`);
  const elf = bytes(elfSource);
  const parsed = parseELF(elf);
  if (parsed.machine !== target.machine) {
    throw new Error(`ELF machine ${parsed.machine} does not match ${targetName}`);
  }

  const appdesc = [];
  const ram = [];
  const flash = [];
  for (const section of parsed.sections) {
    if (section.name === ".flash.appdesc") appdesc.push(section);
    else if (target.flashAddress(section.address)) flash.push(section);
    else ram.push(section);
  }
  ram.sort((left, right) => left.address - right.address);
  flash.sort((left, right) => left.address - right.address);

  const segments = [];
  let imageLength = 24;
  const append = (address, data) => {
    const padded = pad(data, 4, 0);
    segments.push({ address, data: padded });
    imageLength += 8 + padded.length;
  };
  for (const section of appdesc) append(section.address, section.data);
  for (const section of ram) append(section.address, section.data);
  for (const section of flash) {
    const desired = (section.address - FLASH_OFFSET) & 0xffff;
    if (((imageLength + 8) & 0xffff) !== desired) {
      const padding = (desired - imageLength - 16) & 0xffff;
      append(0, new Uint8Array(padding));
    }
    append(section.address, section.data);
  }
  if (!segments.length || segments.length > 255) throw new Error("ELF has an invalid number of loadable sections");

  const output = new ByteBuilder();
  output.byte(0xe9).byte(segments.length).byte(0x02).byte(target.flashConfig).u32(parsed.entry);
  output.byte(0xee).byte(0).byte(0).byte(0);
  output.u16(target.chipID).byte(0).byte(0).byte(0).byte(0xff).byte(0xff);
  output.byte(0).byte(0).byte(0).byte(0).byte(1);
  let checksum = CHECKSUM_MAGIC;
  for (const segment of segments) {
    output.u32(segment.address).u32(segment.data.length).data(segment.data);
    for (const value of segment.data) checksum ^= value;
  }
  while ((output.length & 15) !== 15) output.byte(0);
  output.byte(checksum);
  const imageWithoutHash = output.finish();
  const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", imageWithoutHash));
  return concat(imageWithoutHash, digest);
}

export class ESPWebSerial {
  constructor(port, { log = () => {}, serial = () => {}, progress = () => {} } = {}) {
    this.port = port;
    this.log = log;
    this.serial = serial;
    this.progress = progress;
    this.frames = [];
    this.frameWaiters = [];
    this.monitoring = false;
    this.frame = [];
    this.escaped = false;
    this.insideFrame = false;
    this.closed = false;
    this.reading = false;
    this.reader = undefined;
    this.dtr = false;
    this.webUSB = port.transport === "webusb";
  }

  async open() {
    if (!this.port.readable) {
      let lastError;
      for (let attempt = 0; attempt < 6 && !this.port.readable; attempt++) {
        try {
          await this.port.open({ baudRate: 115200 });
        } catch (error) {
          lastError = error;
          // Navigation, a failed flash, or USB re-enumeration can leave an
          // internally open handle whose readable/writable streams are null.
          // Closing is harmless when it is already closed, and gives the OS
          // time to release the native USB interface before retrying.
          try { await this.port.close(); } catch {}
          await delay(200 + attempt * 200);
        }
      }
      if (!this.port.readable) throw lastError || new Error("failed to open the ESP serial port");
    }
    if (!this.reading) {
      this.reading = true;
      this.readLoop().catch((error) => {
        if (!this.closed) this.log(`Serial read stopped: ${error.message || error}`);
      }).finally(() => { this.reading = false; });
    }
  }

  async flash(elf, targetName) {
    this.monitoring = false;
    this.discardFrames();
    const image = await elfToESPImage(elf, targetName);
    this.log(`Prepared ${formatBytes(image.length)} ESP app image`);
    await this.open();
    await this.connectBootloader();
    const chipMagic = await this.command(0x0a, words(0x40001000), 3000);
    if (chipMagic !== targets[targetName].magic) {
      throw new Error(`connected ESP chip 0x${chipMagic.toString(16)} does not match ${targetName}`);
    }
    this.log(`ROM loader connected (${targetName})`);
    await this.command(0x0d, words(0), 3000);
    const blocks = Math.ceil(image.length / FLASH_BLOCK);
    await this.command(0x02, words(image.length, blocks, FLASH_BLOCK, FLASH_OFFSET, 0), 10000);
    for (let sequence = 0; sequence < blocks; sequence++) {
      const block = new Uint8Array(FLASH_BLOCK);
      block.fill(0xff);
      block.set(image.subarray(sequence * FLASH_BLOCK, (sequence + 1) * FLASH_BLOCK));
      let checksum = CHECKSUM_MAGIC;
      for (const value of block) checksum ^= value;
      await this.command(0x03, concat(words(block.length, sequence, 0, 0), block), 5000, checksum);
      this.progress((sequence + 1) / blocks);
    }
    this.log(`Wrote ${formatBytes(image.length)} at 0x${FLASH_OFFSET.toString(16)}`);
    // Leave the ROM loader idle, then reset explicitly. Asking FLASH_END to
    // reboot can tear down native USB before the browser has released the
    // download-mode control signals, which puts C6 boards straight back into
    // the loader on re-enumeration.
    await this.command(0x04, words(1), 3000);
    await this.startApplication(targetName);
    this.log("Application started; serial monitor attached");
  }

  async connectBootloader() {
    const nativeUSB = this.port.getInfo?.().usbProductId === 0x1001;
    const resets = this.webUSB ? [
      ["USB/JTAG", () => this.usbJTAGReset()],
      ["USB/JTAG retry", async () => { await delay(300); await this.usbJTAGReset(); }],
    ] : nativeUSB ? [
      ["USB/JTAG", () => this.usbJTAGReset()],
      ["classic", () => this.classicReset(50)],
      ["classic delayed", () => this.classicReset(550)],
    ] : [
      ["classic", () => this.classicReset(50)],
      ["classic delayed", () => this.classicReset(550)],
      ["USB/JTAG", () => this.usbJTAGReset()],
    ];
    let lastError;
    for (const [name, reset] of resets) {
      this.log(`Resetting board into the ROM loader (${name})…`);
      await reset();
      this.discardFrames();
      try {
        if (this.webUSB) await delay(250);
        await this.synchronize(this.webUSB ? 800 : 300);
        return;
      } catch (error) {
        lastError = error;
      }
    }
    const diagnostics = this.port.diagnostics?.();
    throw new Error(`could not synchronize with the ESP ROM loader: ${lastError?.message || "timeout"}${diagnostics ? `; ${diagnostics}` : ""}`);
  }

  async usbJTAGReset() {
    await this.setRTS(false);
    await this.setDTR(false);
    await delay(100);
    await this.setDTR(true);
    await this.setRTS(false);
    await delay(100);
    await this.setRTS(true);
    await this.setDTR(false);
    if (!this.webUSB) await this.setRTS(true);
    await delay(100);
    await this.setRTS(false);
    await this.setDTR(false);
  }

  async classicReset(resetDelay) {
    await this.setDTR(false);
    await this.setRTS(true);
    await delay(100);
    await this.setDTR(true);
    await this.setRTS(false);
    await delay(resetDelay);
    await this.setDTR(false);
  }

  async startApplication(targetName) {
    this.log("Restarting board into the application…");
    await this.setDTR(false);
    await this.setRTS(true);
    await delay(200);
    await this.setRTS(false);
    await this.setDTR(false);
    this.monitoring = true;
    // The C6 native USB endpoint needs to remain open while reset settles;
    // this mirrors examples/m5nanoc6/flash.sh. S3 also benefits from waiting
    // for the boot ROM and second-stage loader before reporting completion.
    await delay(targetName === "esp32c6/riscv32" ? 1000 : 500);
  }

  async synchronize(timeout = 300) {
    const payload = new Uint8Array(36);
    payload.set([0x07, 0x07, 0x12, 0x20]);
    payload.fill(0x55, 4);
    let lastError;
    for (let attempt = 0; attempt < 5; attempt++) {
      try {
        await this.command(0x08, payload, timeout);
        return;
      } catch (error) {
        lastError = error;
        this.discardFrames();
        await delay(80);
      }
    }
    throw new Error(`could not synchronize with the ESP ROM loader: ${lastError?.message || "timeout"}`);
  }

  async command(opcode, payload, timeout, checksum = 0) {
    const request = new ByteBuilder();
    request.byte(0).byte(opcode).u16(payload.length).u32(checksum).data(payload);
    const writer = this.port.writable.getWriter();
    try { await writer.write(slip(request.finish())); } finally { writer.releaseLock(); }
    const expires = performance.now() + timeout;
    while (true) {
      const remaining = expires - performance.now();
      if (remaining <= 0) throw new Error(`ESP command 0x${opcode.toString(16)} timed out`);
      const frame = await this.nextFrame(remaining);
      if (frame.length < 8 || frame[0] !== 1 || frame[1] !== opcode) continue;
      const body = frame.subarray(8);
      if (body.length >= 2 && body[body.length - 2] !== 0) {
        throw new Error(`ESP command 0x${opcode.toString(16)} failed (status ${body[body.length - 2]}, error ${body[body.length - 1]})`);
      }
      return new DataView(frame.buffer, frame.byteOffset + 4, 4).getUint32(0, true);
    }
  }

  async setDTR(state) {
    this.dtr = state;
    await this.port.setSignals({ dataTerminalReady: state });
  }

  async setRTS(state) {
    await this.port.setSignals({ requestToSend: state });
    // Windows' usbser.sys may not send an RTS-only control-line change. A
    // no-op DTR update commits the combined state, matching Espressif's
    // WebSerial transport workaround.
    if (!this.webUSB) await this.setDTR(this.dtr);
  }

  async readLoop() {
    const reader = this.port.readable.getReader();
    this.reader = reader;
    try {
      while (!this.closed) {
        const { value, done } = await reader.read();
        if (done) break;
        if (this.monitoring) this.serial(decoder.decode(value, { stream: true }));
        else this.accept(value);
      }
    } finally {
      this.reader = undefined;
      reader.releaseLock();
    }
  }

  accept(data) {
    for (const value of data) {
      if (value === 0xc0) {
        if (this.insideFrame && this.frame.length) this.pushFrame(Uint8Array.from(this.frame));
        this.insideFrame = true; this.frame = []; this.escaped = false;
      } else if (!this.insideFrame) {
        continue;
      } else if (this.escaped) {
        if (value === 0xdc) this.frame.push(0xc0);
        else if (value === 0xdd) this.frame.push(0xdb);
        else { this.frame.push(0xdb); this.frame.push(value); }
        this.escaped = false;
      } else if (value === 0xdb) {
        this.escaped = true;
      } else {
        this.frame.push(value);
      }
    }
  }

  pushFrame(frame) {
    const waiter = this.frameWaiters.shift();
    if (waiter) waiter.resolve(frame);
    else this.frames.push(frame);
  }

  nextFrame(timeout) {
    if (this.frames.length) return Promise.resolve(this.frames.shift());
    return new Promise((resolve, reject) => {
      const waiter = { resolve: (frame) => { clearTimeout(timer); resolve(frame); }, reject };
      const timer = setTimeout(() => {
        const at = this.frameWaiters.indexOf(waiter);
        if (at >= 0) this.frameWaiters.splice(at, 1);
        reject(new Error("serial response timed out"));
      }, timeout);
      this.frameWaiters.push(waiter);
    });
  }

  discardFrames() {
    this.frames.length = 0;
    this.frame.length = 0;
    this.insideFrame = false;
    this.escaped = false;
  }

  async close() {
    this.closed = true;
    if (this.webUSB) {
      // A WebUSB reader may be blocked in transferIn(). Closing the USBDevice
      // first aborts that transfer and releases the claimed CDC interfaces.
      try { await this.port.close(); } catch {}
      try { await this.reader?.cancel(); } catch {}
    } else {
      try { await this.reader?.cancel(); } catch {}
      try { await this.port.close(); } catch {}
    }
  }
}

function parseELF(source) {
  if (source.length < 52 || source[0] !== 0x7f || source[1] !== 0x45 || source[2] !== 0x4c || source[3] !== 0x46) {
    throw new Error("compiler output is not an ELF file");
  }
  if (source[4] !== 1 || source[5] !== 1) throw new Error("ESP flashing requires a little-endian ELF32 file");
  const view = new DataView(source.buffer, source.byteOffset, source.byteLength);
  const entry = view.getUint32(24, true);
  const machine = view.getUint16(18, true);
  const sectionAt = view.getUint32(32, true);
  const sectionSize = view.getUint16(46, true);
  const sectionCount = view.getUint16(48, true);
  const namesIndex = view.getUint16(50, true);
  if (sectionSize < 40 || namesIndex >= sectionCount || sectionAt + sectionSize * sectionCount > source.length) {
    throw new Error("ELF section table is invalid");
  }
  const header = (index) => sectionAt + index * sectionSize;
  const namesHeader = header(namesIndex);
  const namesAt = view.getUint32(namesHeader + 16, true);
  const namesSize = view.getUint32(namesHeader + 20, true);
  if (namesAt + namesSize > source.length) throw new Error("ELF section names are invalid");
  const sections = [];
  for (let index = 0; index < sectionCount; index++) {
    const at = header(index);
    const type = view.getUint32(at + 4, true);
    const flags = view.getUint32(at + 8, true);
    const offset = view.getUint32(at + 16, true);
    const size = view.getUint32(at + 20, true);
    if (type !== 1 || !(flags & 2) || size === 0) continue;
    if (offset + size > source.length) throw new Error("ELF section data is invalid");
    const nameOffset = view.getUint32(at, true);
    let end = namesAt + nameOffset;
    while (end < namesAt + namesSize && source[end] !== 0) end++;
    const name = decoder.decode(source.subarray(namesAt + nameOffset, end));
    sections.push({ name, address: view.getUint32(at + 12, true), data: source.slice(offset, offset + size) });
  }
  return { entry, machine, sections };
}

function slip(data) {
  const output = [0xc0];
  for (const value of data) {
    if (value === 0xc0) output.push(0xdb, 0xdc);
    else if (value === 0xdb) output.push(0xdb, 0xdd);
    else output.push(value);
  }
  output.push(0xc0);
  return Uint8Array.from(output);
}

function words(...values) {
  const output = new Uint8Array(values.length * 4);
  const view = new DataView(output.buffer);
  values.forEach((value, index) => view.setUint32(index * 4, value, true));
  return output;
}

function concat(...values) {
  const length = values.reduce((total, value) => total + value.length, 0);
  const output = new Uint8Array(length);
  let offset = 0;
  for (const value of values) { output.set(value, offset); offset += value.length; }
  return output;
}

function pad(data, alignment, value) {
  const length = Math.ceil(data.length / alignment) * alignment;
  if (length === data.length) return data;
  const output = new Uint8Array(length); output.fill(value); output.set(data); return output;
}

function bytes(value) {
  if (value instanceof Uint8Array) return value;
  if (value instanceof ArrayBuffer) return new Uint8Array(value);
  return new Uint8Array(value.buffer, value.byteOffset, value.byteLength);
}

class ByteBuilder {
  constructor() { this.values = []; }
  get length() { return this.values.length; }
  byte(value) { this.values.push(value & 255); return this; }
  u16(value) { return this.byte(value).byte(value >>> 8); }
  u32(value) { return this.u16(value).u16(value >>> 16); }
  data(value) { for (const byte of value) this.values.push(byte); return this; }
  finish() { return Uint8Array.from(this.values); }
}

function delay(milliseconds) { return new Promise((resolve) => setTimeout(resolve, milliseconds)); }
function formatBytes(value) { return value < 1048576 ? `${(value / 1024).toFixed(1)} KiB` : `${(value / 1048576).toFixed(2)} MiB`; }
