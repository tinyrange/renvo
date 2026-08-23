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

test("C models use the complete semantic language-service pipeline", async () => {
  const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
  const worker = await readFile(new URL("./worker.mjs", import.meta.url), "utf8");
  assert.match(app, /const semanticLanguages = \["go", C_LANGUAGE_ID\]/);
  assert.match(app, /registerCompletionItemProvider\(semanticLanguages/);
  assert.match(app, /registerSignatureHelpProvider\(semanticLanguages/);
  assert.match(app, /registerDefinitionProvider\(semanticLanguages/);
  assert.match(app, /registerHoverProvider\(semanticLanguages/);
  assert.match(app, /registerReferenceProvider\(semanticLanguages/);
  assert.match(app, /registerRenameProvider\(semanticLanguages/);
  assert.match(app, /function cIncludeContextAt\(model, position\)/);
  assert.match(app, /name\.startsWith\("libc\/include\/"\)/);
  assert.doesNotMatch(app, /if \(activeBuildLanguage\(\) === "c"\) return \[\]/);
  assert.doesNotMatch(app, /C project · build for diagnostics/);
  assert.match(app, /language: activeBuildLanguage\(\)/);
  assert.match(worker, /args\.push\("-language", request\.language\)/);
});

test("bundled C library sources are visible in the library explorer", async () => {
  const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
  assert.match(app, /heading\.textContent = "C standard library"/);
  assert.match(app, /cLibraryDirectory\(catalog, "include", "Headers"\)/);
  assert.match(app, /cLibraryDirectory\(catalog, "src", "Implementation"\)/);
  assert.match(app, /librarySourceFile\(`libc\/\$\{file\}`/);
});
