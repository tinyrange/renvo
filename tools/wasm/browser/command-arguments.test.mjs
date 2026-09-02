import assert from "node:assert/strict";
import test from "node:test";

import { outputArgument, replaceOutput, splitArguments, terminalRenvoArguments } from "./command-arguments.mjs";

test("target switches replace separated output arguments", () => {
  assert.equal(replaceOutput("-s -o app.wasm .", "app"), "-s -o app .");
});

test("target switches normalize equals output arguments", () => {
  assert.equal(replaceOutput("-s -o=app.wasm .", "app"), "-s -o app .");
});

test("the effective output is the final output argument", () => {
  const args = splitArguments("-o old -s -o=app.wasm .");
  assert.equal(outputArgument(args), "app.wasm");
  assert.equal(replaceOutput("-o old -s -o=app.wasm .", "app"), "-o app -s -o app .");
});

test("terminal builds default to WASI without overriding explicit targets", () => {
  assert.deepEqual(terminalRenvoArguments(["-o", "app.wasm", "."]), ["-t", "wasi/wasm32", "-o", "app.wasm", "."]);
  assert.deepEqual(terminalRenvoArguments(["test", "."]), ["test", "-t", "wasi/wasm32", "."]);
  assert.deepEqual(terminalRenvoArguments(["-t", "darwin/arm64", "-o", "app", "."]), ["-t", "darwin/arm64", "-o", "app", "."]);
  assert.deepEqual(terminalRenvoArguments(["--help"]), ["--help"]);
});
