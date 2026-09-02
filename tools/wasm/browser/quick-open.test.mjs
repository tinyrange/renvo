import assert from "node:assert/strict";
import test from "node:test";

import { catalogFileItems, filterQuickOpenItems, fuzzyScore, quickOpenQuery } from "./quick-open.mjs";

test("recognizes VS Code quick-open prefixes", () => {
  assert.deepEqual(quickOpenQuery("> format"), { mode: "commands", query: "format", prefix: ">" });
  assert.deepEqual(quickOpenQuery("@main"), { mode: "symbols", query: "main", prefix: "@" });
  assert.deepEqual(quickOpenQuery("# Widget"), { mode: "definitions", query: "Widget", prefix: "#" });
  assert.deepEqual(quickOpenQuery(":42:7"), { mode: "line", query: "42:7", prefix: ":" });
  assert.equal(quickOpenQuery("main.go").mode, "files");
});

test("fuzzy filtering favors direct filename matches", () => {
  const items = [
    { label: "src/main.go", detail: "Project" },
    { label: "std/math/bits.go", detail: "Standard library" },
    { label: "examples/blink/main.go", detail: "Sample" },
  ];
  assert.deepEqual(filterQuickOpenItems(items, "blink main").map((item) => item.label), ["examples/blink/main.go"]);
  assert.ok(fuzzyScore("main.go", "main") < fuzzyScore("examples/main.go", "main"));
});

test("catalog index includes standard library, platform, sample, and libc metadata", () => {
  const items = catalogFileItems({
    packages: { fmt: { files: ["print.go"] } },
    platforms: {
      "renvo.dev/examples/blink": { root: "examples/blink", main: true, files: ["main.go"] },
      "renvo.dev/device/gpio": { root: "device/gpio", files: ["gpio.go"] },
    },
    libc: ["include/stdio.h"],
  });
  assert.deepEqual(items, [
    { path: "std/fmt/print.go", source: "Standard library" },
    { path: "examples/blink/main.go", source: "Sample", importPath: "renvo.dev/examples/blink" },
    { path: "device/gpio/gpio.go", source: "Platform library", importPath: "renvo.dev/device/gpio" },
    { path: "libc/include/stdio.h", source: "C standard library" },
  ]);
});
