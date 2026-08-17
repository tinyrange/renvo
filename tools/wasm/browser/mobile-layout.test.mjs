import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const root = new URL("./", import.meta.url);

test("phone workspace exposes the file, target, editor, and console flow", async () => {
  const [html, css, app] = await Promise.all([
    readFile(new URL("index.html", root), "utf8"),
    readFile(new URL("styles.css", root), "utf8"),
    readFile(new URL("app.mjs", root), "utf8"),
  ]);

  for (const view of ["files", "editor", "console"]) {
    assert.match(html, new RegExp(`data-mobile-view="${view}"`));
  }
  assert.match(html, /id="mobile-target-button"/);
  assert.match(html, /id="copy-to-playground"/);
  assert.match(html, /id="mobile-target-view"/);
  assert.match(html, /data-mobile-transport="webusb"/);
  assert.match(html, /data-mobile-transport="webserial"/);
  assert.match(html, /id="mobile-flash-view"/);
  assert.match(html, /data-panel="plotter"/);
  assert.match(html, /id="serial-plotter-canvas"/);
  assert.match(html, /id="toggle-plotter-size"/);
  assert.match(html, /interactive-widget=resizes-content/);
  assert.match(css, /@media \(max-width: 680px\)/);
  assert.match(css, /\.ide\[data-mobile-view="files"\] \.sidebar/);
  assert.match(css, /\.ide\[data-mobile-view="target"\] \.mobile-target-view/);
  assert.match(css, /\.ide\[data-mobile-view="editor"\] \.workbench/);
  assert.match(css, /\.ide\[data-mobile-view="console"\] \.workbench/);
  assert.match(css, /\.mobile-flash-view \{[\s\S]*position: fixed/);
  assert.match(css, /\.workbench\.plotter-expanded \{[\s\S]*grid-template-rows: 34px 0 minmax\(0, 1fr\)/);
  assert.match(app, /showMobileView\("editor"\)/);
  assert.match(app, /copyActiveFileToPlayground/);
  assert.match(app, /models\.get\("main\.go"\)/);
  assert.doesNotMatch(app, /PLAYGROUND_COPY_PREFIX/);
  assert.match(app, /openMobileFlashView\("Select a device"\)/);
  assert.match(app, /appendSerialText/);
  assert.match(app, /setPlotterExpanded/);
  assert.doesNotMatch(css, /--mobile-viewport-height/);
  assert.doesNotMatch(app, /--mobile-viewport-height/);
});
