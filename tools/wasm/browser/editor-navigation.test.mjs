import assert from "node:assert/strict";
import test from "node:test";

import { installEditorOpener } from "./editor-navigation.mjs";

test("cross-file definitions load and open the target model", async () => {
  let opener;
  const monaco = {
    editor: {
      registerEditorOpener(value) {
        opener = value;
        return { dispose() {} };
      },
    },
  };
  const calls = [];
  const range = { startLineNumber: 171, startColumn: 18, endLineNumber: 171, endColumn: 26 };
  const host = {
    cleanPath: (path) => path.replace(/^\//, ""),
    ensureSourceModel: async (name) => ({ name }),
    openFile: (name) => calls.push(["open", name]),
    editor: {
      setSelection: (value) => calls.push(["select", value]),
      revealRangeInCenter: (value) => calls.push(["reveal", value]),
      setPosition: () => assert.fail("range was treated as a position"),
      revealPositionInCenter: () => assert.fail("range was treated as a position"),
      focus: () => calls.push(["focus"]),
    },
  };

  installEditorOpener(monaco, host);
  const handled = await opener.openCodeEditor({}, { path: "/device/sensor/bme688/bme688.go" }, range);

  assert.equal(handled, true);
  assert.deepEqual(calls, [
    ["open", "device/sensor/bme688/bme688.go"],
    ["select", range],
    ["reveal", range],
    ["focus"],
  ]);
});

test("cross-file definitions decline missing source models", async () => {
  let opener;
  const monaco = {
    editor: {
      registerEditorOpener(value) {
        opener = value;
        return { dispose() {} };
      },
    },
  };
  const host = {
    cleanPath: () => "missing.go",
    ensureSourceModel: async () => undefined,
    openFile: () => assert.fail("missing source was opened"),
    editor: {},
  };

  installEditorOpener(monaco, host);
  assert.equal(await opener.openCodeEditor({}, { path: "/missing.go" }), false);
});
