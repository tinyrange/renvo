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

test("the Web IDE terminal runs Makefiles against the worker filesystem", () => {
  const html = fs.readFileSync(new URL("./index.html", import.meta.url), "utf8");
  const app = fs.readFileSync(new URL("./app.mjs", import.meta.url), "utf8");
  const worker = fs.readFileSync(new URL("./worker.mjs", import.meta.url), "utf8");
  assert.match(html, /id="terminal-command"/);
  assert.match(app, /function runTerminalCommand\(\)/);
  assert.match(worker, /runMakePipeline/);
  assert.match(worker, /new Map\(request\.files/);
});
