export function detectDeviceProfile(input = {}) {
  const userAgent = input.userAgent || "";
  const platform = input.platform || "";
  const mobileHint = Boolean(input.mobile);
  const touchPoints = Number(input.maxTouchPoints || 0);
  const coarsePointer = Boolean(input.coarsePointer);
  const width = Number(input.width || 0);
  const shortSide = Number(input.shortSide || width);
  const android = /Android/i.test(`${platform} ${userAgent}`);
  const ios = /iPad|iPhone|iPod/i.test(`${platform} ${userAgent}`) || platform === "MacIntel" && touchPoints > 1;
  const mobileCapable = mobileHint || android || ios || coarsePointer && touchPoints > 0;
  const phone = mobileCapable && (shortSide > 0 ? shortSide <= 700 : width <= 700);
  const tablet = mobileCapable && !phone;
  return { android, ios, mobileCapable, phone, tablet, deviceClass: phone ? "phone" : tablet ? "tablet" : "desktop" };
}

export function chooseESPTransportAvailability({ profile, webSerial, webUSB }) {
  // WebUSB can use the ESP32-C6 vendor JTAG interface even when a desktop
  // kernel owns the CDC interfaces. Keep it available on every WebUSB host.
  return { webserial: Boolean(webSerial), webusb: Boolean(webUSB) };
}
