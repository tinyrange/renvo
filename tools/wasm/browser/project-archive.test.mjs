import assert from "node:assert/strict";
import test from "node:test";
import { decodeProjectZip, normalizeProjectPath, encodeProjectZip } from "./project-archive.mjs";

test("project paths stay inside the workspace", () => {
  assert.equal(normalizeProjectPath("./cmd/app/main.go"), "cmd/app/main.go");
  assert.throws(() => normalizeProjectPath("../secret"), /escape/);
});

test("project ZIP round trips UTF-8 files", () => {
  const files = { "main.go": "package main\n", "assets/message.txt": "héllo\n" };
  assert.deepEqual(decodeProjectZip(encodeProjectZip(files)), files);
});
