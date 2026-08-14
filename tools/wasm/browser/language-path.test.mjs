import assert from "node:assert/strict";
import test from "node:test";

import { cleanLanguagePath } from "./language-path.mjs";

test("language-service paths map to catalog and workspace models", () => {
  const models = new Map([["main.go", {}]]);
  const initial = { "main.go": "" };
  assert.equal(cleanLanguagePath("/workspace/device/i2c/i2c.go", models, initial), "device/i2c/i2c.go");
  assert.equal(cleanLanguagePath("/workspace/std/fmt/fmt.go", models, initial), "std/fmt/fmt.go");
  assert.equal(cleanLanguagePath("/workspace/examples/demo/main.go", models, initial), "examples/demo/main.go");
  assert.equal(cleanLanguagePath("/workspace/main.go", models, initial), "main.go");
});
