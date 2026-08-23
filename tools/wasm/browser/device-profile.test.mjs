import assert from "node:assert/strict";
import test from "node:test";
import { chooseESPTransportAvailability, detectDeviceProfile } from "./device-profile.mjs";

test("large Android tablets retain a mobile device profile", () => {
  const profile = detectDeviceProfile({ platform: "Linux armv8l", userAgent: "Mozilla/5.0 (Linux; Android 15; Tablet)", maxTouchPoints: 10, coarsePointer: true, width: 1280, shortSide: 800 });
  assert.equal(profile.android, true);
  assert.equal(profile.tablet, true);
  assert.deepEqual(chooseESPTransportAvailability({ profile, webSerial: false, webUSB: true }), { webserial: false, webusb: true });
});

test("touch tablets can use WebUSB when user agent reduction hides Android", () => {
  const profile = detectDeviceProfile({ platform: "Linux", userAgent: "Mozilla/5.0", maxTouchPoints: 5, coarsePointer: true, width: 1024, shortSide: 768 });
  assert.equal(profile.tablet, true);
  assert.equal(chooseESPTransportAvailability({ profile, webSerial: false, webUSB: true }).webusb, true);
});

test("desktop WebUSB remains available for the kernel-independent JTAG interface", () => {
  const profile = detectDeviceProfile({ platform: "MacIntel", userAgent: "Mozilla/5.0 (Macintosh)", maxTouchPoints: 0 });
  assert.deepEqual(chooseESPTransportAvailability({ profile, webSerial: true, webUSB: true }), {
    webserial: true, webusb: true,
  });
});
