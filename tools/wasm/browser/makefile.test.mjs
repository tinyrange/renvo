import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import { parseMakefile, planMakefile } from "./makefile.mjs";

test("browser Makefiles plan Renvo commands in dependency order", () => {
  const file = parseMakefile("CFLAGS = -O2\nall: app\napp: a.o b.o\n\trenvo cc $^ -o $@\na.o: a.c\n\t@renvo cc $(CFLAGS) -c $< -o $@\nb.o: b.c\n\trenvo cc $(CFLAGS) -c $< -o $@\n");
  const plan = planMakefile(file, [], (name) => name === "a.c" || name === "b.c");
  assert.deepEqual(plan.map((item) => item.text), [
    "renvo cc -O2 -c a.c -o a.o", "renvo cc -O2 -c b.c -o b.o", "renvo cc a.o b.o -o app",
  ]);
  assert.equal(plan[0].quiet, true);
});

test("browser Makefiles reject shell commands", () => {
  assert.throws(() => planMakefile(parseMakefile("all:\n\tcc a.c\n")), /must invoke renvo/);
});

test("the Web IDE terminal runs Makefiles against the virtual workspace", () => {
  const html = fs.readFileSync(new URL("./index.html", import.meta.url), "utf8");
  const app = fs.readFileSync(new URL("./app.mjs", import.meta.url), "utf8");
  const worker = fs.readFileSync(new URL("./worker.mjs", import.meta.url), "utf8");
  assert.match(html, /id="terminal-host"/);
  assert.match(html, /@xterm\/xterm@5\.5\.0/);
  assert.match(app, /function executeTerminalLine\(line\)/);
  assert.match(app, /async function runTerminalArguments\(inputArgs\)/);
  assert.match(app, /action: "terminal"/);
  assert.match(app, /workspacePayload\(true\)/);
  assert.match(app, /syncGeneratedFiles\(result\.files\)/);
  assert.match(worker, /terminalCompilerModule/);
  assert.match(worker, /filesystemComplete: true/);
  assert.match(worker, /\["renvo", \.\.\.\(request\.args \|\| \[\]\)\]/);
  assert.match(app, /terminalRenvoArguments\(inputArgs\)/);
  assert.match(app, /command === "file"/);
  assert.match(app, /async function runTerminalWASI\(name, args = \[\]\)/);
  assert.match(app, /command\.endsWith\("\.wasm"\) \|\| command\.startsWith\("\.\/"\)/);
  assert.match(worker, /request\.type === "terminal-run"/);
  assert.match(app, /Ctrl\+A\/E\/U\/K\/W\/L/);
  assert.match(app, /attachCustomKeyEventHandler/);
  assert.match(app, /event\.key !== "Tab"[\s\S]*?event\.preventDefault\(\);[\s\S]*?event\.stopPropagation\(\);[\s\S]*?completeTerminalInput\(\);/);
  assert.match(app, /function longestCommonPrefix\(values\)/);
  assert.match(app, /if \(wasAtEnd\) xtermTerminal\.write\(data\)/);
  assert.match(app, /function replaceTerminalLine\(value\)[^]*oldCursor[^]*renderedLength/);
  assert.doesNotMatch(app, /function replaceTerminalLine\(value\)[^}]*renderTerminalInput\(\)/);
  assert.doesNotMatch(app, /if \(build\.action === "terminal"\)[^]*writeTerminal\(report\)/);
});

test("failed terminal builds resolve without replacing the validated editor build", () => {
  const app = fs.readFileSync(new URL("./app.mjs", import.meta.url), "utf8");
  assert.match(app, /if \(build\.action !== "terminal"[^]*buildValidationState/);
  assert.match(app, /terminalBuildResolve\?\.\(result\)/);
});
