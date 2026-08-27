import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { RTG_LANGUAGE_ID, registerRTGLanguage } from "./rtg-language.mjs";

test("registers RTG syntax and editor behavior", () => {
  const calls = {};
  const monaco = { languages: {
    getLanguages: () => [],
    register: (value) => { calls.registration = value; },
    setLanguageConfiguration: (id, value) => { calls.configuration = { id, value }; },
    setMonarchTokensProvider: (id, value) => { calls.tokens = { id, value }; },
  } };
  registerRTGLanguage(monaco);
  assert.equal(RTG_LANGUAGE_ID, "renvo-rtg");
  assert.deepEqual(calls.registration.extensions, [".rtg"]);
  assert.equal(calls.configuration.value.comments.lineComment, "#");
  assert.ok(calls.tokens.value.declarationKeywords.includes("target"));
  assert.ok(calls.tokens.value.blockKeywords.includes("instructions"));
  assert.ok(calls.tokens.value.goKeywords.includes("func"));
});

test("keeps an existing RTG registration", () => {
  let registered = false;
  registerRTGLanguage({ languages: {
    getLanguages: () => [{ id: RTG_LANGUAGE_ID }],
    register: () => { registered = true; },
  } });
  assert.equal(registered, false);
});

test("project RTG targets use the browser JIT and VM backend pipeline", async () => {
  const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
  const worker = await readFile(new URL("./worker.mjs", import.meta.url), "utf8");
  assert.match(app, /if \(name\.endsWith\("\.rtg"\) \|\| name\.endsWith\("\.rtgasm"\)\) return RTG_LANGUAGE_ID/);
  assert.match(app, /type, id, definition, target, files: payload\.files/);
  assert.match(app, /backendFormat: buildTarget\.backendFormat \|\| "wasm"/);
  assert.match(app, /function starterCommand\(\)[\s\S]*selectedTarget\?\.output/);
  assert.match(app, /command: starterCommand\(\)/);
  assert.match(app, /backendRoots: backendDefinition \? \[backendDefinition\] : \[\]/);
  assert.match(app, /if \(backendDefinition\) await useProjectBackend\(backendDefinition\)/);
  assert.match(worker, /request\.type === "backend-inspect" \|\| request\.type === "backend-prepare"/);
  assert.match(worker, /request\.backendFormat === "vm32"/);
  assert.match(worker, /"-evaluate-unit", plan\.temporary/);
  assert.match(app, /rtgDefinition: buildTarget\.rtgDefinition \? new URL\(buildTarget\.rtgDefinition, catalogUrl\)\.href : ""/);
  assert.match(app, /rtgImports: \(buildTarget\.rtgImports \|\| \[\]\)\.map/);
  assert.match(worker, /request\.rtgImports \|\| \[\]/);
  assert.match(worker, /fd_sync: \(\) => 0/);
  assert.match(worker, /path_create_directory: .*pathCreateDirectory/);
  assert.match(worker, /path_rename:/);
});
