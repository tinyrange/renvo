import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { C_LANGUAGE_ID, registerCLanguage } from "./c-language.mjs";

test("registers a dedicated C tokenizer and editor configuration", () => {
  const calls = {};
  const monaco = { languages: {
    getLanguages: () => [],
    register: (value) => { calls.registration = value; },
    setLanguageConfiguration: (id, value) => { calls.configuration = { id, value }; },
    setMonarchTokensProvider: (id, value) => { calls.tokens = { id, value }; },
  } };
  registerCLanguage(monaco);
  assert.equal(C_LANGUAGE_ID, "renvo-c");
  assert.deepEqual(calls.registration.extensions, [".c", ".h"]);
  assert.equal(calls.configuration.id, C_LANGUAGE_ID);
  assert.equal(calls.tokens.id, C_LANGUAGE_ID);
  assert.ok(calls.tokens.value.keywords.includes("for"));
  assert.ok(calls.tokens.value.typeKeywords.includes("int"));
  assert.ok(calls.tokens.value.tokenizer.comment);
  assert.ok(calls.tokens.value.tokenizer.string);
});

test("does not replace an existing Renvo C registration", () => {
  let registered = false;
  registerCLanguage({ languages: {
    getLanguages: () => [{ id: C_LANGUAGE_ID }],
    register: () => { registered = true; },
  } });
  assert.equal(registered, false);
});

test("catalog source models use extension-based language selection", async () => {
  const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
  assert.match(app, /createModel\(decoder\.decode\(source\), languageForFile\(name\)/);
  assert.match(app, /if \(name\.endsWith\("\.c"\) \|\| name\.endsWith\("\.h"\)\) return C_LANGUAGE_ID/);
});
