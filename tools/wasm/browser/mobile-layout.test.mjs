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
  assert.match(html, /id="project-file-input"/);
  assert.match(html, /id="project-action-menu"/);
  assert.match(html, /id="build-scope"/);
  assert.match(html, /id="new-file-dialog"/);
  assert.match(html, /id="new-file-kind"/);
  assert.doesNotMatch(html, /class="activity-bar"/);
  assert.match(html, /id="snapshot-dialog"/);
  assert.match(html, /id="text-dialog"/);
  assert.match(html, /id="confirm-dialog"/);
  assert.match(html, /id="open-editor-tabs"/);
  assert.match(html, /id="format-file"/);
  assert.match(html, /data-panel="tests"/);
  assert.match(html, /data-panel="preview"/);
  assert.match(html, /id="mobile-target-view"/);
  assert.match(html, /data-mobile-transport="webusb"/);
  assert.match(html, /data-mobile-transport="webserial"/);
  assert.match(html, /id="mobile-flash-view"/);
  assert.match(html, /data-panel="plotter"/);
  assert.match(html, /id="serial-plotter-canvas"/);
  assert.match(html, /id="toggle-plotter-size"/);
  assert.match(html, /interactive-widget=resizes-content/);
  assert.match(css, /@media \(max-width: 680px\)/);
  assert.match(css, /data-device-class="tablet"/);
  assert.match(css, /\.ide\[data-mobile-view="files"\] \.sidebar/);
  assert.match(css, /\.ide\[data-mobile-view="target"\] \.mobile-target-view/);
  assert.match(css, /\.ide\[data-mobile-view="editor"\] \.workbench/);
  assert.match(css, /\.ide\[data-mobile-view="console"\] \.workbench/);
  assert.match(css, /\.mobile-flash-view \{[\s\S]*position: fixed/);
  assert.match(css, /\.workbench\.plotter-expanded \{[\s\S]*grid-template-rows: 38px 0 minmax\(0, 1fr\)/);
  assert.match(app, /showMobileView\("editor"\)/);
  assert.match(app, /copyActiveFileToPlayground/);
  assert.match(app, /const destination = activeFile\.split\("\/"\)\.pop\(\)/);
  assert.doesNotMatch(app, /PLAYGROUND_COPY_PREFIX/);
  assert.match(app, /openMobileFlashView\("Select a device"\)/);
  assert.match(app, /appendSerialText/);
  assert.match(app, /setPlotterExpanded/);
  assert.match(app, /chooseESPTransportAvailability/);
  assert.match(app, /runTests/);
  assert.match(app, /event\.metaKey[\s\S]*event\.key\.toLowerCase\(\) === "s"[\s\S]*saveAndDeploy\(\)/);
  assert.match(app, /function saveAndDeploy\(\)[\s\S]*selectedTarget\?\.device === "esp32"[\s\S]*runArtifact\(\)/);
  assert.match(app, /openSnapshotDialog/);
  assert.match(app, /function createNewWorkspaceFile[\s\S]*setBuildPackage\("\."\)/);
  assert.match(app, /function syncBuildScope/);
  assert.match(app, /target: selectedTarget\?\.name \|\| restoredTargetName/);
  assert.match(app, /\\\.\(\?:go\|c\|h\|rtg\)\$/);
  assert.doesNotMatch(app, /\b(?:prompt|confirm)\s*\(/);
  assert.doesNotMatch(css, /--mobile-viewport-height/);
  assert.doesNotMatch(app, /--mobile-viewport-height/);
});
