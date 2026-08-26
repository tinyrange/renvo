import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("DOS executables offer a ZIP download and retain the raw artifact", async () => {
  const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
  assert.match(app, /\/\\\.\(\?:com\|exe\)\$\/i\.test\(filename\)/);
  assert.match(app, /encodeProjectZip\(\{ \[filename\]: file\.data \}\)/);
  assert.match(app, /zipLink\.textContent = "Download ZIP"/);
  assert.match(app, /rawLink\.textContent = "Raw"/);
});
