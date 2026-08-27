import { parsePicoDebugImage, planPicoPatches } from "./pico-cmsis-dap.mjs";

const RENVO_VENDOR_ID = 0xcafe;
const RENVO_MONITOR_PRODUCT_ID = 0x4021;
const COMMAND_INFO = 1;
const COMMAND_BEGIN = 2;
const COMMAND_WRITE = 3;
const COMMAND_COMMIT = 4;
const COMMAND_WRITE_FAST = 5;
const PROTOCOL_MAJOR = 2;
const PROTOCOL_MINOR = 0;
const CLIENT_VERSION = 1 << 16 | 1 << 8;
const CAPABILITY_FAST_WRITE = 1;
const PACKET_SIZE = 64;
const WRITE_HEADER_SIZE = 12;
const USB_TIMEOUT_MS = 4000;

export async function requestPicoMonitor() {
  if (!globalThis.navigator?.usb) {
    throw new Error("WebUSB is unavailable. Open Renvo over HTTPS in a Chromium-based browser.");
  }
  const permitted = await navigator.usb.getDevices?.() || [];
  let device = permitted.find(isRenvoMonitor);
  if (!device) {
    device = await navigator.usb.requestDevice({ filters: [{ vendorId: RENVO_VENDOR_ID, productId: RENVO_MONITOR_PRODUCT_ID }] });
  }
  return new PicoWebUSBMonitor(device, navigator.usb);
}

export class PicoWebUSBMonitor {
  constructor(device, usb = globalThis.navigator?.usb) {
    this.transport = "webusb-pico-monitor";
    this.device = device;
    this.usb = usb;
    this.opened = false;
    this.disconnected = false;
    this.handleDisconnect = (event) => {
      if (event.device !== this.device) return;
      this.disconnected = true;
      this.opened = false;
    };
    this.usb?.addEventListener?.("disconnect", this.handleDisconnect);
  }

  getInfo() {
    return {
      usbVendorId: this.device.vendorId,
      usbProductId: this.device.productId,
      ...(this.monitorInfo || {}),
    };
  }
  canReopen() { return !this.disconnected; }

  async open() {
    if (this.opened) return;
    this.disconnected = false;
    try {
      if (!this.device.opened) await usbOperation(this.device.open(), "open");
      const selected = findMonitorInterface(this.device.configurations);
      if (!selected) throw new Error("device does not expose the Renvo reload interface");
      if (this.device.configuration?.configurationValue !== selected.configuration.configurationValue) {
        await usbOperation(this.device.selectConfiguration(selected.configuration.configurationValue), "select configuration");
      }
      const active = findMonitorInterface([this.device.configuration]) || selected;
      await usbOperation(this.device.claimInterface(active.iface.interfaceNumber), "claim interface");
      if (active.alternate.alternateSetting !== 0) {
        await usbOperation(this.device.selectAlternateInterface(active.iface.interfaceNumber, active.alternate.alternateSetting), "select interface");
      }
      this.bulkOut = active.bulkOut;
      this.bulkIn = active.bulkIn;
      this.opened = true;
      const info = await this.command(COMMAND_INFO, CLIENT_VERSION);
      this.monitorInfo = parseMonitorInfo(info);
      if (this.monitorInfo.protocolMajor !== PROTOCOL_MAJOR) {
        throw new Error(`monitor protocol ${this.monitorInfo.protocolMajor}.${this.monitorInfo.protocolMinor} is incompatible with browser protocol ${PROTOCOL_MAJOR}.${PROTOCOL_MINOR}; install the current monitor`);
      }
      if (this.monitorInfo.clientVersion !== CLIENT_VERSION) {
        throw new Error("monitor handshake did not echo the browser version; install the current monitor");
      }
      this.fastWrite = (this.monitorInfo.capabilities & CAPABILITY_FAST_WRITE) !== 0;
    } catch (error) {
      await this.close();
      throw new Error(`could not open the Renvo Pico monitor: ${error.message || error}`);
    }
  }

  async close() {
    this.opened = false;
    try { if (this.device.opened) await usbOperation(this.device.close(), "close", 1000); } catch {}
    this.usb?.removeEventListener?.("disconnect", this.handleDisconnect);
  }

  async command(operation, address = 0, payload = new Uint8Array()) {
    if (!this.opened && operation !== COMMAND_INFO) throw new Error("Pico monitor is not open");
    if (payload.length > PACKET_SIZE - WRITE_HEADER_SIZE) throw new Error("Pico monitor write packet is too large");
    const packet = new Uint8Array(WRITE_HEADER_SIZE + payload.length);
    packet.set([0x52, 0x4e, 0x56, 0x32, operation]);
    packet[6] = PROTOCOL_MAJOR;
    packet[7] = PROTOCOL_MINOR;
    new DataView(packet.buffer).setUint32(8, address >>> 0, true);
    packet.set(payload, WRITE_HEADER_SIZE);
    const sent = await usbOperation(this.device.transferOut(this.bulkOut.endpointNumber, packet), `write command ${operation}`);
    if (sent.status && sent.status !== "ok") throw new Error(`Pico monitor USB write ${sent.status}`);
    const received = await usbOperation(this.device.transferIn(this.bulkIn.endpointNumber, PACKET_SIZE), `read command ${operation}`);
    if (received.status && received.status !== "ok") throw new Error(`Pico monitor USB read ${received.status}`);
    const response = new Uint8Array(received.data.buffer, received.data.byteOffset, received.data.byteLength);
    if (response.length < WRITE_HEADER_SIZE || response[0] !== 0x52 || response[1] !== 0x4e ||
        response[2] !== 0x56 || response[3] !== 0x32 || response[4] !== operation) {
      throw new Error("Pico monitor returned an invalid response");
    }
    if (response.length >= 36) this.monitorInfo = parseMonitorInfo(response);
    if (response[5] !== 0) {
      const diagnostic = operation === COMMAND_BEGIN || operation === COMMAND_COMMIT
        ? `; launch stage ${this.monitorInfo?.launchStage ?? "unknown"}, failure ${this.monitorInfo?.launchFailure ?? "unknown"}, FIFO state ${this.monitorInfo?.launchState ?? "unknown"}`
        : "";
      throw new Error(`Pico monitor rejected command ${operation} (status ${response[5]}${diagnostic})`);
    }
    return response;
  }

  async write(address, payload) {
    if (!this.fastWrite) {
      await this.command(COMMAND_WRITE, address, payload);
      return;
    }
    if (!this.opened) throw new Error("Pico monitor is not open");
    if (payload.length > PACKET_SIZE - WRITE_HEADER_SIZE) throw new Error("Pico monitor write packet is too large");
    const packet = new Uint8Array(WRITE_HEADER_SIZE + payload.length);
    packet.set([0x52, 0x4e, 0x56, 0x32, COMMAND_WRITE_FAST]);
    packet[6] = PROTOCOL_MAJOR;
    packet[7] = PROTOCOL_MINOR;
    new DataView(packet.buffer).setUint32(8, address >>> 0, true);
    packet.set(payload, WRITE_HEADER_SIZE);
    const sent = await usbOperation(this.device.transferOut(this.bulkOut.endpointNumber, packet), "write firmware");
    if (sent.status && sent.status !== "ok") throw new Error(`Pico monitor USB write ${sent.status}`);
  }
}

export class PicoMonitorHotReloadSession {
  constructor(monitor, options = {}) {
    this.monitor = monitor;
    this.progress = options.progress || (() => {});
    this.previous = undefined;
  }

  async update(source) {
    const image = parsePicoDebugImage(source);
    const patches = planPicoPatches(this.previous, image);
    if (!patches.length && this.previous) {
      await this.monitor.open();
      await this.monitor.command(COMMAND_INFO, CLIENT_VERSION);
      return { entry: image.entry, patchCount: 0, bytesWritten: 0, unchanged: true, monitorInfo: this.monitor.getInfo?.() };
    }
    await this.monitor.open();
    await this.monitor.command(COMMAND_BEGIN);
    const total = patches.reduce((sum, patch) => sum + patch.data.length, 0);
    let written = 0;
    let packets = 0;
    for (const patch of patches) {
      for (let offset = 0; offset < patch.data.length; offset += PACKET_SIZE - WRITE_HEADER_SIZE) {
        const part = patch.data.subarray(offset, offset + PACKET_SIZE - WRITE_HEADER_SIZE);
        await this.monitor.write(patch.address + offset, part);
        written += part.length;
        packets++;
        this.progress(total ? written / total : 1);
      }
    }
    await this.monitor.command(COMMAND_COMMIT, image.entry);
    // COMMIT is acknowledged only after core 1 has accepted the complete ROM
    // launch sequence. Verify the monitor is still servicing USB before we
    // retain this image as the basis for the next differential update.
    await this.monitor.command(COMMAND_INFO, CLIENT_VERSION);
    this.previous = image;
    this.progress(1);
    return { entry: image.entry, patchCount: packets, bytesWritten: written, unchanged: false, monitorInfo: this.monitor.getInfo?.() };
  }

  close() { return this.monitor.close(); }
}

function isRenvoMonitor(device) {
  return device.vendorId === RENVO_VENDOR_ID && device.productId === RENVO_MONITOR_PRODUCT_ID;
}

function parseMonitorInfo(response) {
  if (response.byteLength < 36) {
    return { protocolMajor: response[6] || 0, protocolMinor: response[7] || 0, clientVersion: 0, capabilities: 0 };
  }
  const view = new DataView(response.buffer, response.byteOffset, response.byteLength);
  return {
    protocolMajor: response[6],
    protocolMinor: response[7],
    generation: view.getUint32(8, true),
    reloadStart: view.getUint32(12, true),
    reloadEnd: view.getUint32(16, true),
    capabilities: view.getUint32(20, true),
    monitorVersion: view.getUint32(24, true),
    chip: view.getUint32(28, true),
    clientVersion: view.getUint32(32, true),
    launchStage: response.byteLength >= 40 ? view.getUint32(36, true) : 0,
    launchEcho: response.byteLength >= 44 ? view.getUint32(40, true) : 0,
    launchState: response.byteLength >= 48 ? view.getUint32(44, true) : 0,
    launchFailure: response.byteLength >= 52 ? view.getUint32(48, true) : 0,
  };
}

function formatVersion(version = 0) {
  version >>>= 0;
  return `${version >>> 16}.${version >>> 8 & 0xff}.${version & 0xff}`;
}

async function usbOperation(promise, action, timeout = USB_TIMEOUT_MS) {
  let timer;
  try {
    return await Promise.race([
      promise,
      new Promise((_, reject) => {
        timer = setTimeout(() => reject(new Error(`Pico monitor USB ${action} timed out after ${timeout} ms`)), timeout);
      }),
    ]);
  } finally {
    clearTimeout(timer);
  }
}

export function formatPicoMonitorInfo(info = {}) {
  const firmware = formatVersion(info.monitorVersion);
  const chip = info.chip === 0x2040 ? "RP2040" : info.chip === 0x2350 ? "RP2350" : `chip 0x${(info.chip || 0).toString(16)}`;
  return `firmware ${firmware} · protocol ${info.protocolMajor || 0}.${info.protocolMinor || 0} · ${chip} · generation ${info.generation || 0}`;
}

function findMonitorInterface(configurations = []) {
  for (const configuration of configurations || []) {
    for (const iface of configuration?.interfaces || []) {
      for (const alternate of iface.alternates || []) {
        if (alternate.interfaceClass !== 0xff || alternate.interfaceSubclass !== 0x52 || alternate.interfaceProtocol !== 1) continue;
        const bulkOut = alternate.endpoints?.find((endpoint) => endpoint.type === "bulk" && endpoint.direction === "out");
        const bulkIn = alternate.endpoints?.find((endpoint) => endpoint.type === "bulk" && endpoint.direction === "in");
        if (bulkOut && bulkIn) return { configuration, iface, alternate, bulkOut, bulkIn };
      }
    }
  }
  return undefined;
}
