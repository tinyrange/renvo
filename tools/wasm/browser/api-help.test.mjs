import assert from "node:assert/strict";
import test from "node:test";

import { catalogHelpPages, findAPIHelpReference, installAPIHelpAction } from "./api-help.mjs";

const catalog = {
  builtins: { importPath: "builtin", functions: [{ name: "println" }] },
  packages: {},
  platforms: {
    "renvo.dev/device/i2c": {
      root: "device/i2c",
      docs: {
        name: "i2c",
        functions: [{ name: "New", file: "i2c.go", line: 87 }],
        types: [{
          name: "Device", file: "i2c.go", line: 80,
          methods: [{ name: "ReadAt", file: "i2c.go", line: 159 }],
        }, {
          name: "Memory", file: "i2c.go", line: 190,
          methods: [{ name: "ReadAt", file: "i2c.go", line: 200 }],
        }],
      },
    },
  },
};

test("catalog pages retain the source root used by language-service definitions", () => {
  const page = catalogHelpPages(catalog).find((item) => item.importPath === "renvo.dev/device/i2c");
  assert.equal(page.sourceRoot, "device/i2c");
});

test("API references resolve functions and methods to stable help anchors", () => {
  assert.deepEqual(findAPIHelpReference(catalog, "/device/i2c/i2c.go", 87, "New"), {
    importPath: "renvo.dev/device/i2c", anchor: "doc-functions-new",
  });
  assert.deepEqual(findAPIHelpReference(catalog, "device/i2c/i2c.go", 159, "ReadAt"), {
    importPath: "renvo.dev/device/i2c", anchor: "doc-method-device-readat",
  });
  assert.deepEqual(findAPIHelpReference(catalog, "device/i2c/i2c.go", 200, "ReadAt"), {
    importPath: "renvo.dev/device/i2c", anchor: "doc-method-memory-readat",
  });
  assert.deepEqual(findAPIHelpReference(catalog, "main.go", 1, "println"), {
    importPath: "builtin", anchor: "doc-functions-println",
  });
  assert.equal(findAPIHelpReference(catalog, "main.go", 1, "localValue"), undefined);
});

test("the Monaco context action resolves the model and cursor when invoked", async () => {
  let action;
  const editor = { addAction(value) { action = value; return { dispose() {} }; } };
  const model = { name: "main.go" };
  const position = { lineNumber: 4, column: 9 };
  const calls = [];
  installAPIHelpAction({}, editor, async (...values) => calls.push(values));

  assert.equal(action.label, "Open API Documentation");
  assert.equal(action.contextMenuGroupId, "navigation");
  await action.run({ getModel: () => model, getPosition: () => position });
  assert.deepEqual(calls, [[model, position]]);
});
