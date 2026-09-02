import assert from "node:assert/strict";
import test from "node:test";

import { flashCommand, parseFlashArguments } from "./flash-command.mjs";

test("flash commands round trip toolbar transport choices", () => {
  assert.equal(flashCommand("webusb"), "flash --transport webusb");
  assert.deepEqual(parseFlashArguments(["--transport", "webusb"]), { help: false, transport: "webusb" });
  assert.deepEqual(parseFlashArguments(["--transport=webserial"]), { help: false, transport: "webserial" });
});

test("flash command rejects unsupported arguments and transports", () => {
  assert.throws(() => parseFlashArguments(["board.bin"]), /unsupported argument/);
  assert.throws(() => parseFlashArguments(["--transport", "bluetooth"]), /unsupported transport/);
  assert.deepEqual(parseFlashArguments(["--help"]), { help: true, transport: "" });
});
