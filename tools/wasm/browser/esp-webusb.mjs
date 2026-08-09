// First-party WebUSB adapter for the native Espressif USB Serial/JTAG device.
// This implements the small SerialPort-shaped contract used by ESPWebSerial;
// the ROM-loader and image code therefore remains shared. The USB layout and
// CDC requests are documented by Espressif and the USB-IF:
// https://www.espressif.com/sites/default/files/documentation/esp32-c6_technical_reference_manual_en.pdf
// https://www.usb.org/document-library/class-definitions-communication-devices-12

const ESPRESSIF_VENDOR_ID = 0x303a;
const USB_SERIAL_JTAG_PRODUCT_ID = 0x1001;
const CDC_CONTROL_CLASS = 0x02;
const CDC_DATA_CLASS = 0x0a;
const SET_LINE_CODING = 0x20;
const SET_CONTROL_LINE_STATE = 0x22;

export function supportsESPWebUSB(target) {
  return target === "esp32c6/riscv32" || target === "esp32s3/xtensa_lx7";
}

export function preferredESPTransport({ saved, android, webSerial, webUSB }) {
  const preferred = saved === "webserial" || saved === "webusb"
    ? saved : android ? "webusb" : "webserial";
  const fallback = preferred === "webusb" ? "webserial" : "webusb";
  const available = { webserial: webSerial, webusb: webUSB };
  return available[preferred] ? preferred : available[fallback] ? fallback : preferred;
}

export function supportsESPWebUSBPlatform({ platform = "", userAgent = "" } = {}) {
  return platform === "Android" || /Android/i.test(userAgent);
}

export function selectAuthorizedESPUSBDevice(devices) {
  const permitted = (devices || []).filter((device) =>
    device.vendorId === ESPRESSIF_VENDOR_ID && device.productId === USB_SERIAL_JTAG_PRODUCT_ID);
  if (permitted.length === 1) return permitted[0];
  if (permitted.length > 1 && permitted.every((device) =>
    device.serialNumber && device.serialNumber === permitted[0].serialNumber)) {
    return permitted.find((device) => device.opened) || permitted[permitted.length - 1];
  }
  return undefined;
}

export async function requestESPUSBPort(target) {
  if (!supportsESPWebUSB(target)) throw new Error(`unsupported ESP target ${target}`);
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
  return new ESPWebUSBPort(device, navigator.usb);
}

export class ESPWebUSBPort {
  constructor(device, usb = globalThis.navigator?.usb) {
    this.transport = "webusb";
    this.device = device;
    this.usb = usb;
    this.readable = null;
    this.writable = null;
    this.claimedInterfaces = [];
    this.generation = 0;
    this.disconnected = false;
    this.signals = { dataTerminalReady: false, requestToSend: false };
    this.bytesRead = 0;
    this.bytesWritten = 0;
    this.controlRequests = 0;
    this.handleDisconnect = (event) => {
      if (event.device !== this.device) return;
      this.disconnected = true;
      this.generation++;
      try { this.readController?.error(new DOMException("ESP WebUSB device disconnected", "NetworkError")); } catch {}
      this.readable = null;
      this.writable = null;
    };
    this.startListening();
  }

  getInfo() {
    return { usbVendorId: this.device.vendorId, usbProductId: this.device.productId };
  }

  canReopen() {
    return !this.disconnected;
  }

  diagnostics() {
    const input = this.cdc?.bulkIn?.endpointNumber;
    const output = this.cdc?.bulkOut?.endpointNumber;
    return `WebUSB CDC interface ${this.cdc?.data?.interfaceNumber ?? "?"}, endpoints OUT ${output ?? "?"}/IN ${input ?? "?"}, ${this.bytesWritten} bytes sent, ${this.bytesRead} bytes received, ${this.controlRequests} control requests`;
  }

  async open({ baudRate = 115200 } = {}) {
    if (this.readable || this.writable) return;
    this.startListening();
    this.disconnected = false;
    const generation = ++this.generation;
    try {
      if (!this.device.opened) await this.device.open();
      const selected = findCDCConfiguration(this.device.configurations);
      if (!selected) throw new Error("device does not expose CDC control and bulk data interfaces");
      if (this.device.configuration?.configurationValue !== selected.configuration.configurationValue) {
        await this.device.selectConfiguration(selected.configuration.configurationValue);
      }
      const cdc = findCDCConfiguration([this.device.configuration]);
      if (!cdc) throw new Error("selected USB configuration no longer exposes the CDC interfaces");
      this.cdc = cdc;

      // CDC class requests target the control interface, so merely claiming
      // the bulk-data interface is insufficient. Desktop kernels normally own
      // this interface; those platforms must use WebSerial instead.
      try { await this.claim(cdc.control.interfaceNumber, cdc.controlAlternate.alternateSetting); }
      catch (error) {
        throw new Error(`CDC control interface ${cdc.control.interfaceNumber} is owned by the operating system; use WebSerial on desktop (${error.message || error})`);
      }
      await this.claim(cdc.data.interfaceNumber, cdc.dataAlternate.alternateSetting);
      await this.setLineCoding(baudRate);

      this.readable = new ReadableStream({
        start: (controller) => { this.readController = controller; },
        pull: (controller) => this.readPacket(controller, generation),
        cancel: () => { if (generation === this.generation) this.generation++; },
      }, { highWaterMark: 0 });
      this.writable = new WritableStream({
        write: (chunk) => this.writeAll(chunk, generation),
      });
    } catch (error) {
      await this.close();
      throw new Error(`could not open ESP WebUSB transport: ${error.message || error}`);
    }
  }

  async claim(interfaceNumber, alternateSetting) {
    await this.device.claimInterface(interfaceNumber);
    this.claimedInterfaces.push(interfaceNumber);
    if (alternateSetting !== 0) await this.device.selectAlternateInterface(interfaceNumber, alternateSetting);
  }

  async setLineCoding(baudRate) {
    const data = new Uint8Array(7);
    new DataView(data.buffer).setUint32(0, baudRate, true);
    data[4] = 0;
    data[5] = 0;
    data[6] = 8;
    await this.controlTransfer(SET_LINE_CODING, 0, data);
  }

  async setSignals(next) {
    if (next.dataTerminalReady !== undefined) this.signals.dataTerminalReady = next.dataTerminalReady;
    if (next.requestToSend !== undefined) this.signals.requestToSend = next.requestToSend;
    const value = (this.signals.dataTerminalReady ? 1 : 0) |
      (this.signals.requestToSend ? 2 : 0);
    await this.controlTransfer(SET_CONTROL_LINE_STATE, value);
  }

  async controlTransfer(request, value, data) {
    const result = await this.device.controlTransferOut({
      requestType: "class", recipient: "interface", request, value,
      index: this.cdc.control.interfaceNumber,
    }, data);
    if (result.status !== "ok") throw new Error(`CDC control request 0x${request.toString(16)} returned ${result.status}`);
    if (data && result.bytesWritten !== data.byteLength) {
      throw new Error(`CDC control request wrote ${result.bytesWritten || 0}/${data.byteLength} bytes`);
    }
    this.controlRequests++;
  }

  async readPacket(controller, generation) {
    if (generation !== this.generation || this.disconnected) return;
    try {
      const endpoint = this.cdc.bulkIn;
      const result = await this.device.transferIn(endpoint.endpointNumber, endpoint.packetSize || 64);
      if (generation !== this.generation) return;
      if (result.status === "stall") {
        await this.device.clearHalt("in", endpoint.endpointNumber);
        return;
      }
      if (result.status !== "ok") throw new Error(`bulk IN transfer returned ${result.status}`);
      if (result.data?.byteLength) {
        const data = new Uint8Array(result.data.buffer, result.data.byteOffset, result.data.byteLength);
        this.bytesRead += data.byteLength;
        controller.enqueue(data.slice());
      }
    } catch (error) {
      if (generation === this.generation && !this.disconnected) controller.error(error);
    }
  }

  async writeAll(source, generation) {
    const data = source instanceof Uint8Array ? source : new Uint8Array(source);
    let offset = 0;
    while (offset < data.byteLength) {
      if (generation !== this.generation || this.disconnected) throw new Error("ESP WebUSB device disconnected");
      const endpoint = this.cdc.bulkOut.endpointNumber;
      const result = await this.device.transferOut(endpoint, data.subarray(offset));
      if (result.status === "stall") {
        await this.device.clearHalt("out", endpoint);
        continue;
      }
      if (result.status !== "ok") throw new Error(`bulk OUT transfer returned ${result.status}`);
      if (!result.bytesWritten) throw new Error("bulk OUT transfer wrote no data");
      offset += result.bytesWritten;
      this.bytesWritten += result.bytesWritten;
    }
  }

  async close() {
    this.generation++;
    this.readController = undefined;
    this.readable = null;
    this.writable = null;
    // USBDevice.close() releases every claimed interface and aborts pending
    // transfers. Do it before waiting on the stream reader: Android otherwise
    // can leave the bulk IN request alive and keep the CDC interface claimed.
    this.claimedInterfaces = [];
    try { if (this.device.opened) await this.device.close(); } catch {}
    this.usb?.removeEventListener?.("disconnect", this.handleDisconnect);
    this.listening = false;
  }

  startListening() {
    if (this.listening) return;
    this.usb?.addEventListener?.("disconnect", this.handleDisconnect);
    this.listening = true;
  }
}

function findCDCConfiguration(configurations) {
  for (const configuration of configurations || []) {
    let control;
    let controlAlternate;
    let data;
    let dataAlternate;
    let bulkIn;
    let bulkOut;
    for (const iface of configuration.interfaces || []) {
      for (const alternate of iface.alternates || []) {
        if (!control && alternate.interfaceClass === CDC_CONTROL_CLASS) {
          control = iface;
          controlAlternate = alternate;
        }
        const nextIn = alternate.endpoints?.find((endpoint) => endpoint.type === "bulk" && endpoint.direction === "in");
        const nextOut = alternate.endpoints?.find((endpoint) => endpoint.type === "bulk" && endpoint.direction === "out");
        if (!data && alternate.interfaceClass === CDC_DATA_CLASS && nextIn && nextOut) {
          data = iface;
          dataAlternate = alternate;
          bulkIn = nextIn;
          bulkOut = nextOut;
        }
      }
    }
    if (control && data) return { configuration, control, controlAlternate, data, dataAlternate, bulkIn, bulkOut };
  }
  return undefined;
}
