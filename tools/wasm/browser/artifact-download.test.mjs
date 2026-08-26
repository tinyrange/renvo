import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("DOS executables offer a ZIP download and retain the raw artifact", async () => {
  const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
  assert.match(app, /\/\\\.\(\?:com\|exe\)\$\/i\.test\(filename\)/);
  assert.match(app, /createDownloadURL\(encodeProjectZip\(\{ \[filename\]: file\.data \}\), "application\/zip"\)/);
  assert.match(app, /zipLink\.textContent = "Download ZIP"/);
  assert.match(app, /rawLink\.textContent = "Raw"/);
});

test("small downloads avoid blob URLs", async () => {
  const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
  assert.match(app, /const inlineDownloadLimit = 16 \* 1024 \* 1024/);
  assert.match(app, /return `data:\$\{type\};base64,\$\{btoa\(chunks\.join\(""\)\)\}`/);
});
