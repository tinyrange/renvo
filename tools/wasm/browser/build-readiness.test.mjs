import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { buildReadiness } from "./build-readiness.mjs";

const readyBase = { compilerReady: true, editorReady: true, building: false, currentRevision: 7, validatedRevision: 7 };

test("build is enabled only for the currently validated revision", () => {
  assert.equal(buildReadiness({ ...readyBase, state: "success" }).ready, true);
  assert.equal(buildReadiness({ ...readyBase, state: "success", currentRevision: 8 }).ready, false);
  assert.equal(buildReadiness({ ...readyBase, state: "checking" }).label, "Checking…");
});

test("known failures make the build action explain what to do", () => {
  const result = buildReadiness({ ...readyBase, state: "failure" });
  assert.equal(result.ready, false);
  assert.equal(result.label, "Fix errors");
  assert.match(result.title, /reported errors/);
});

test("ready builds use the target action label", () => {
  const download = buildReadiness({ ...readyBase, state: "success", readyLabel: "Download" });
  const flash = buildReadiness({ ...readyBase, state: "success", readyLabel: "JTAG load" });
  assert.equal(download.label, "Download");
  assert.equal(download.title, "Download (Ctrl+Enter)");
  assert.equal(flash.label, "JTAG load");
});

test("the IDE validates through the complete compiler pipeline and reuses the artifact", async () => {
  const [app, worker, buildScript] = await Promise.all([
    readFile(new URL("app.mjs", import.meta.url), "utf8"),
    readFile(new URL("worker.mjs", import.meta.url), "utf8"),
    readFile(new URL("../build-browser.sh", import.meta.url), "utf8"),
  ]);
  assert.match(app, /type: "validate"/);
  assert.match(app, /function publishValidatedBuild/);
  assert.match(app, /buildValidationRevision === buildRevision/);
  assert.match(worker, /request\.type !== "compile" && request\.type !== "validate"/);
  assert.match(worker, /result\.type = "validation-result"/);
  assert.match(buildScript, /browser\/build-readiness\.mjs/);
});

test("target actions replace the artifact panel", async () => {
  const [app, page] = await Promise.all([
    readFile(new URL("app.mjs", import.meta.url), "utf8"),
    readFile(new URL("index.html", import.meta.url), "utf8"),
  ]);
  assert.match(app, /function primaryTargetAction[\s\S]*device === "esp32"[\s\S]*runArtifact/);
  assert.match(app, /function secondaryTargetAction[\s\S]*device === "esp32"[\s\S]*downloadValidatedArtifact/);
  assert.match(app, /function downloadValidatedArtifact/);
  assert.doesNotMatch(page, /data-panel="artifacts"|id="artifacts"/);
});
