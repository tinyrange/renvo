import assert from "node:assert/strict";
import test from "node:test";

import { terminalCompletionChoices, terminalPathCompletions } from "./terminal-completion.mjs";

const files = ["main.go", "app.wasm", "src/main.go", "src/tools/check.wasm", "assets/icon.bin"];
const folders = ["empty", "src/generated"];

test("dot-slash completes every virtual filesystem entry", () => {
  assert.deepEqual(terminalCompletionChoices("./", files, folders).candidates, [
    "./app.wasm", "./assets/", "./empty/", "./main.go", "./src/",
  ]);
  assert.deepEqual(terminalCompletionChoices("./s", files, folders).candidates, ["./src/"]);
  assert.deepEqual(terminalPathCompletions(files, folders, "./src/"), [
    "./src/generated/", "./src/main.go", "./src/tools/",
  ]);
});

test("path completion includes generated files and implied directories", () => {
  assert.deepEqual(terminalCompletionChoices("file assets/", files, folders).candidates, ["assets/icon.bin"]);
  assert.deepEqual(terminalCompletionChoices("cat empty/", files, folders).candidates, []);
  assert.deepEqual(terminalCompletionChoices("run ./", files, folders).candidates, ["./app.wasm", "./src/"]);
});

test("Renvo options and target completion retain shell context", () => {
  assert.deepEqual(terminalCompletionChoices("renvo -t wa", files, folders, ["wasi/wasm32", "darwin/arm64"]).candidates, ["wasi/wasm32"]);
  assert.ok(terminalCompletionChoices("renvo -o ./", files, folders).candidates.includes("./main.go"));
});

test("flash command completes its transport options", () => {
  assert.deepEqual(terminalCompletionChoices("flash --transport web", files, folders).candidates, ["webserial", "webusb"]);
  assert.deepEqual(terminalCompletionChoices("flash --h", files, folders).candidates, ["--help"]);
});
