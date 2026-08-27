import { parsePicoDebugImage, planPicoPatches } from "./pico-cmsis-dap.mjs";

const RENVO_VENDOR_ID = 0xcafe;
const RENVO_MONITOR_PRODUCT_ID = 0x4021;
const COMMAND_INFO = 1;
const COMMAND_BEGIN = 2;
const COMMAND_WRITE = 3;
const COMMAND_COMMIT = 4;
const PACKET_SIZE = 64;
const WRITE_HEADER_SIZE = 12;

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

  getInfo() { return { usbVendorId: this.device.vendorId, usbProductId: this.device.productId }; }
  canReopen() { return !this.disconnected; }

  async open() {
    if (this.opened) return;
    this.disconnected = false;
    try {
      if (!this.device.opened) await this.device.open();
      const selected = findMonitorInterface(this.device.configurations);
      if (!selected) throw new Error("device does not expose the Renvo reload interface");
      if (this.device.configuration?.configurationValue !== selected.configuration.configurationValue) {
        await this.device.selectConfiguration(selected.configuration.configurationValue);
      }
      const active = findMonitorInterface([this.device.configuration]) || selected;
      await this.device.claimInterface(active.iface.interfaceNumber);
      if (active.alternate.alternateSetting !== 0) {
        await this.device.selectAlternateInterface(active.iface.interfaceNumber, active.alternate.alternateSetting);
      }
      this.bulkOut = active.bulkOut;
      this.bulkIn = active.bulkIn;
      this.opened = true;
      await this.command(COMMAND_INFO);
    } catch (error) {
      await this.close();
      throw new Error(`could not open the Renvo Pico monitor: ${error.message || error}`);
    }
  }

  async close() {
    this.opened = false;
    try { if (this.device.opened) await this.device.close(); } catch {}
    this.usb?.removeEventListener?.("disconnect", this.handleDisconnect);
  }

  async command(operation, address = 0, payload = new Uint8Array()) {
    if (!this.opened && operation !== COMMAND_INFO) throw new Error("Pico monitor is not open");
    if (payload.length > PACKET_SIZE - WRITE_HEADER_SIZE) throw new Error("Pico monitor write packet is too large");
    const packet = new Uint8Array(WRITE_HEADER_SIZE + payload.length);
    packet.set([0x52, 0x4e, 0x56, 0x32, operation]);
    new DataView(packet.buffer).setUint32(8, address >>> 0, true);
    packet.set(payload, WRITE_HEADER_SIZE);
    const sent = await this.device.transferOut(this.bulkOut.endpointNumber, packet);
    if (sent.status && sent.status !== "ok") throw new Error(`Pico monitor USB write ${sent.status}`);
    const received = await this.device.transferIn(this.bulkIn.endpointNumber, PACKET_SIZE);
    if (received.status && received.status !== "ok") throw new Error(`Pico monitor USB read ${received.status}`);
    const response = new Uint8Array(received.data.buffer, received.data.byteOffset, received.data.byteLength);
    if (response.length < WRITE_HEADER_SIZE || response[0] !== 0x52 || response[1] !== 0x4e ||
        response[2] !== 0x56 || response[3] !== 0x32 || response[4] !== operation) {
      throw new Error("Pico monitor returned an invalid response");
    }
    if (response[5] !== 0) throw new Error(`Pico monitor rejected command ${operation} (status ${response[5]})`);
    return response;
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
      return { entry: image.entry, patchCount: 0, bytesWritten: 0, unchanged: true };
    }
    await this.monitor.open();
    await this.monitor.command(COMMAND_BEGIN);
    const total = patches.reduce((sum, patch) => sum + patch.data.length, 0);
    let written = 0;
    let packets = 0;
    for (const patch of patches) {
      for (let offset = 0; offset < patch.data.length; offset += PACKET_SIZE - WRITE_HEADER_SIZE) {
        const part = patch.data.subarray(offset, offset + PACKET_SIZE - WRITE_HEADER_SIZE);
        await this.monitor.command(COMMAND_WRITE, patch.address + offset, part);
        written += part.length;
        packets++;
        this.progress(total ? written / total : 1);
      }
    }
    await this.monitor.command(COMMAND_COMMIT, image.entry);
    this.previous = image;
    this.progress(1);
    return { entry: image.entry, patchCount: packets, bytesWritten: written, unchanged: false };
  }

  close() { return this.monitor.close(); }
}

function isRenvoMonitor(device) {
  return device.vendorId === RENVO_VENDOR_ID && device.productId === RENVO_MONITOR_PRODUCT_ID;
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
