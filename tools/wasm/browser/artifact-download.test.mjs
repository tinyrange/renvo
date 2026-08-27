import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("artifact downloads preserve the raw output", async () => {
  const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
  const start = app.indexOf("function downloadValidatedArtifact()");
  const end = app.indexOf("function publishValidatedBuild()", start);
  const download = app.slice(start, end);
  assert.match(download, /const data = artifact\.data/);
  assert.match(download, /const filename = artifact\.name\.split\("\/"\)\.pop\(\)/);
  assert.match(download, /link\.download = filename/);
  assert.doesNotMatch(download, /encodeProjectZip|application\/zip|\.zip/);
});

test("small downloads avoid blob URLs", async () => {
  const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
  assert.match(app, /const inlineDownloadLimit = 16 \* 1024 \* 1024/);
  assert.match(app, /return `data:\$\{type\};base64,\$\{btoa\(chunks\.join\(""\)\)\}`/);
});
