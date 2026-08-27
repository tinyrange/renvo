import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import { MAKEFILE_LANGUAGE_ID, makefileCompletions, registerMakefileLanguage } from "./makefile-language.mjs";

function mockMonaco() {
  const state = { registered: [], tokens: null, provider: null };
  return { state, languages: {
    CompletionItemKind: { Variable: 1, Function: 2, Keyword: 3, Reference: 4, File: 5 },
    getLanguages: () => [], register: (value) => state.registered.push(value),
    setLanguageConfiguration: () => {}, setMonarchTokensProvider: (_, value) => { state.tokens = value; },
    registerCompletionItemProvider: (_, value) => { state.provider = value; },
  } };
}

function model(source, line) {
  return { getValue: () => source, getLineContent: () => line,
    getWordUntilPosition: (position) => ({ word: "", startColumn: position.column, endColumn: position.column }) };
}

test("registers Makefiles with highlighting and completion", () => {
  const monaco = mockMonaco(); registerMakefileLanguage(monaco, () => ["main.c"]);
  assert.equal(monaco.state.registered[0].id, MAKEFILE_LANGUAGE_ID);
  assert.ok(monaco.state.tokens.tokenizer.root.some(([pattern]) => pattern.test?.("\trenvo cc -c main.c")));
  assert.ok(monaco.state.provider);
});

test("completes declared variables inside expansion", () => {
  const monaco = mockMonaco(), source = "CFLAGS := -O2\nall:\n\trenvo cc $(CF";
  const suggestions = makefileCompletions(monaco, model(source, "\trenvo cc $(CF"), { lineNumber: 3, column: 16 });
  assert.ok(suggestions.some((item) => item.label === "CFLAGS" && item.insertText === "CFLAGS"));
});

test("completes targets and project files in rules", () => {
  const monaco = mockMonaco(), source = "app: main.o\nmain.o: ";
  const suggestions = makefileCompletions(monaco, model(source, "main.o: "), { lineNumber: 2, column: 9 }, ["main.c", "hash.h"]);
  assert.ok(suggestions.some((item) => item.label === "app"));
  assert.ok(suggestions.some((item) => item.label === "main.c" && item.detail === "Project file"));
});

test("the editor assigns Makefiles their dedicated language", () => {
  const app = fs.readFileSync(new URL("./app.mjs", import.meta.url), "utf8");
  assert.match(app, /registerMakefileLanguage/);
  assert.match(app, /base === "Makefile"/);
});

test("browser builds package the Makefile language and evaluator", () => {
  const build = fs.readFileSync(new URL("../build-browser.sh", import.meta.url), "utf8");
  assert.equal(build.match(/tools\/wasm\/browser\/makefile-language\.mjs/g)?.length, 2);
  assert.equal(build.match(/tools\/wasm\/browser\/makefile\.mjs/g)?.length, 2);
});
