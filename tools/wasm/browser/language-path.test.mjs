import assert from "node:assert/strict";
import test from "node:test";

import { cleanLanguagePath, isCLibrarySourcePath, sourceImportPath } from "./language-path.mjs";

test("language-service paths map to catalog and workspace models", () => {
  const models = new Map([["main.go", {}]]);
  const initial = { "main.go": "" };
  assert.equal(cleanLanguagePath("/workspace/device/i2c/i2c.go", models, initial), "device/i2c/i2c.go");
  assert.equal(cleanLanguagePath("/workspace/std/fmt/fmt.go", models, initial), "std/fmt/fmt.go");
  assert.equal(cleanLanguagePath("/workspace/libc/src/stdio.c", models, initial), "libc/src/stdio.c");
  assert.equal(cleanLanguagePath("/workspace/examples/demo/main.go", models, initial), "examples/demo/main.go");
  assert.equal(cleanLanguagePath("/workspace/main.go", models, initial), "main.go");
});

test("catalog source paths resolve outside the playground module", () => {
  const catalog = {
    packages: { fmt: { files: ["fmt.go"] } },
    platforms: {
      "renvo.dev/device/clock": { root: "device/clock", files: ["clock.go"] },
      "renvo.dev/device/board": { root: "device/board", files: ["m5atoms3lite_impl.go"] },
    },
  };
  assert.equal(sourceImportPath("/workspace/std/fmt/fmt.go", catalog), "fmt");
  assert.equal(sourceImportPath("/workspace/device/clock/clock.go", catalog), "renvo.dev/device/clock");
  assert.equal(sourceImportPath("/workspace/device/board/m5atoms3lite_impl.go", catalog), "renvo.dev/device/board");
});

test("bundled C library headers and implementations are navigable sources", () => {
  const catalog = { libc: ["include/stdio.h", "src/stdio.c"] };
  assert.equal(isCLibrarySourcePath("/workspace/libc/include/stdio.h", catalog), true);
  assert.equal(isCLibrarySourcePath("/workspace/libc/src/stdio.c", catalog), true);
  assert.equal(isCLibrarySourcePath("/workspace/libc/src/missing.c", catalog), false);
});
