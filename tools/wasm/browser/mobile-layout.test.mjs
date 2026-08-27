import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const root = new URL("./", import.meta.url);

test("phone workspace exposes the project, editor, and device flow", async () => {
  const [html, css, app, worker] = await Promise.all([
    readFile(new URL("index.html", root), "utf8"),
    readFile(new URL("styles.css", root), "utf8"),
    readFile(new URL("app.mjs", root), "utf8"),
    readFile(new URL("worker.mjs", root), "utf8"),
  ]);

  for (const view of ["files", "editor", "device"]) {
    assert.match(html, new RegExp(`data-mobile-view="${view}"`));
  }
  assert.match(html, /id="mobile-target-button"/);
  assert.match(html, /id="copy-to-playground"/);
  assert.match(html, /id="project-file-input"/);
  assert.match(html, /id="project-action-menu"/);
  assert.doesNotMatch(html, /id="build-scope"/);
  assert.match(html, /id="browse-examples"/);
  assert.match(html, /id="example-dialog"/);
  assert.match(html, /id="example-board-filter"/);
  assert.match(html, /Start with your hardware/);
  assert.match(html, /Choose a board or computer/);
  assert.match(html, /id="example-target-filter"/);
  assert.match(html, /id="mobile-setup-progress"/);
  assert.doesNotMatch(html, /Initializing compiler/);
  for (const step of ["workspace", "catalog", "editor", "compiler"]) {
    assert.match(html, new RegExp(`data-setup-step="${step}"`));
  }
  assert.match(html, /id="new-file-dialog"/);
  assert.match(html, /data-project-action="new"/);
  assert.match(html, /id="new-project-dialog"/);
  assert.match(html, /name="project-kind" value="go"/);
  assert.match(html, /name="project-kind" value="c"/);
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
  assert.match(html, /id="mobile-device-build"/);
  assert.match(html, /id="mobile-device-output"/);
  assert.match(html, /data-mobile-transport="webusb"/);
  assert.match(html, /data-mobile-transport="webserial"/);
  assert.match(html, /id="mobile-flash-view"/);
  assert.match(html, /id="mobile-flash-detail"/);
  for (const step of ["usb", "check", "firmware", "load", "run"]) {
    assert.match(html, new RegExp(`data-deploy-step="${step}"`));
  }
  assert.match(html, /id="device-permission-dialog"/);
  assert.match(html, /id="device-webusb-status"/);
  assert.match(html, /id="device-webserial-status"/);
  assert.match(html, /id="pico-monitor-build"/);
  assert.match(html, /Compile &amp; download monitor/);
  assert.match(html, /shared RP2040\/RP2350 UF2/);
  assert.match(html, /data-panel="plotter"/);
  assert.match(html, /id="serial-plotter-canvas"/);
  assert.match(html, /id="toggle-plotter-size"/);
  assert.match(html, /interactive-widget=resizes-content/);
  assert.match(css, /@media \(max-width: 680px\)/);
  assert.match(css, /data-device-class="tablet"/);
  assert.match(css, /\.ide\[data-mobile-view="files"\] \.sidebar/);
  assert.match(css, /\.ide\[data-mobile-view="device"\] \.mobile-target-view/);
  assert.match(css, /\.ide\[data-mobile-view="editor"\] \.workbench/);
  assert.match(css, /\.mobile-flash-view \{[\s\S]*position: fixed/);
  assert.match(css, /\.workbench\.plotter-expanded \{[\s\S]*grid-template-rows: 38px 0 minmax\(0, 1fr\)/);
  assert.match(app, /showMobileView\("editor"\)/);
  assert.match(app, /copyActiveFileToPlayground/);
  assert.match(app, /const destination = activeFile\.split\("\/"\)\.pop\(\)/);
  assert.doesNotMatch(app, /PLAYGROUND_COPY_PREFIX/);
  assert.match(app, /if \(mobileDeploymentActive\) openMobileFlashView/);
  assert.match(app, /function startMobileDeployment/);
  assert.match(app, /function receiveCompileProgress/);
  assert.match(app, /arenaSize: entry\.item\.arenaSize \|\| ""/);
  assert.match(app, /Tap to see device load details/);
  assert.match(worker, /type: "compile-progress"/);
  assert.match(worker, /Downloading the board compiler/);
  assert.match(app, /appendSerialText/);
  assert.match(app, /setPlotterExpanded/);
  assert.match(app, /chooseESPTransportAvailability/);
  assert.match(app, /function requestDevicePermission/);
  assert.match(app, /renvo\.devicePermissionExplained\.v1/);
  assert.match(app, /renvo\.devicePermissionExplained\.rp2\.v1/);
  assert.match(app, /function compilePicoMonitor/);
  assert.match(app, /\.\/cmd\/renvopico-monitor/);
  assert.match(app, /maybeOpenMobileExamples/);
  assert.match(app, /exampleCatalogPromise = catalogPromise/);
  assert.match(app, /function selectInitialExampleBoard/);
  assert.match(app, /function catalogComputers/);
  assert.match(app, /example-machine-group/);
  assert.match(app, /exampleBoardSelectionTouched/);
  assert.match(app, /elements\.exampleResults\.scrollTop = 0/);
  assert.match(app, /setSetupStep\("compiler", "active"/);
  assert.doesNotMatch(html, /id="mobile-build"/);
  assert.match(app, /runTests/);
  assert.match(app, /requestedAvailable \? requestedTarget : restoredAvailable \? restoredTargetName/);
  assert.match(app, /selectedTarget\.tags.*target\.tags/);
  assert.match(app, /event\.metaKey[\s\S]*event\.key\.toLowerCase\(\) === "s"[\s\S]*saveAndDeploy\(\)/);
  assert.match(app, /function saveAndDeploy\(\)[\s\S]*isBoardTarget\(selectedTarget\)[\s\S]*runArtifact\(\)/);
  assert.match(app, /openSnapshotDialog/);
  assert.match(app, /function createNewWorkspaceFile[\s\S]*setBuildPackage\("\."\)/);
  assert.match(app, /function createNewProject[\s\S]*initialCFiles/);
  assert.match(app, /activeBuildLanguage\(\) === "c"[\s\S]*result\.unshift\("cc"\)/);
  assert.match(app, /async function loadCLibrary[\s\S]*stdlibFiles\.set\(`libc\/\$\{file\}`/);
  assert.match(app, /catalog\.module[\s\S]*stdlibFiles\.set\("go\.mod"/);
  assert.match(app, /setBuildPackage\(item\.root, item\.target, button, item\.language\)/);
  assert.match(app, /item\.language === "c"[\s\S]*file === "main\.c"/);
  assert.match(app, /language: projectLanguage/);
  assert.match(app, /function syncBuildScope/);
  assert.match(app, /async function handleExampleAction/);
  assert.match(app, /target: selectedTarget\?\.name \|\| restoredTargetName/);
  assert.match(app, /\\\.\(\?:go\|c\|h\|rtg\)\$/);
  assert.doesNotMatch(app, /\b(?:prompt|confirm)\s*\(/);
  assert.doesNotMatch(css, /--mobile-viewport-height/);
  assert.doesNotMatch(app, /--mobile-viewport-height/);
});
