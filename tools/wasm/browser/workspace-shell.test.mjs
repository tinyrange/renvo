import assert from "node:assert/strict";
import test from "node:test";
import fs from "node:fs";

const html = fs.readFileSync(new URL("./index.html", import.meta.url), "utf8");
const app = fs.readFileSync(new URL("./app.mjs", import.meta.url), "utf8");
const styles = fs.readFileSync(new URL("./styles.css", import.meta.url), "utf8");

test("the desktop shell starts at a Renvo welcome screen without a command bar input", () => {
  assert.match(html, /id="project-name">Renvo</);
  assert.match(html, /id="start-screen"/);
  assert.match(html, /data-start-action="new-go"/);
  assert.doesNotMatch(html, /data-start-action="folder"/);
  assert.doesNotMatch(html, /project-directory-input|data-project-action="directory"/);
  assert.match(html, /id="command"[^>]*type="hidden"/);
  assert.doesNotMatch(html, /class="command-input"/);
  assert.match(styles, /\.editor-tabs \{[^}]*grid-row: 1;/);
  assert.match(styles, /\.start-grid \{[^}]*repeat\(3,/);
});

test("the explorer context menu creates files and folders", () => {
  assert.match(html, /data-file-action="new-file"/);
  assert.match(html, /data-file-action="new-folder"/);
  assert.match(app, /function createWorkspaceFolder\(parent = ""\)/);
  assert.match(app, /workspaceFolders = new Set/);
  assert.match(styles, /\.folder-row/);
});

test("browser preview opens as a right-hand workbench column", () => {
  assert.match(html, /class="preview-pane" id="preview-pane"/);
  assert.match(app, /classList\.add\("preview-open"\)/);
  assert.match(styles, /\.workbench\.preview-open \{ grid-template-columns:/);
  assert.match(styles, /\.preview-pane \{[^}]*grid-column: 2;/);
});

test("snapshots use a create card and saved-history region", () => {
  assert.match(html, /class="snapshot-layout"/);
  assert.match(html, /class="snapshot-create-card"/);
  assert.match(html, /class="snapshot-history"/);
});

test("long target lists can be filtered on desktop and mobile", () => {
  assert.match(html, /id="target-search"/);
  assert.match(html, /id="mobile-target-search"/);
  assert.match(app, /function filterTargetList\(container, query\)/);
  assert.match(app, /dataset\.search/);
});

test("workspace regions can be resized and generated binaries open in a hex viewer", () => {
  assert.match(html, /id="sidebar-resizer"[^>]*role="separator"/);
  assert.match(html, /id="panel-resizer"[^>]*role="separator"/);
  assert.match(html, /id="hex-view"/);
  assert.match(app, /function installWorkspaceResizers\(\)/);
  assert.match(app, /setPointerCapture/);
  assert.match(app, /function openBinaryFile\(name\)/);
  assert.match(app, /formatHexPage\(data, activeBinaryOffset, hexPageSize\)/);
  assert.doesNotMatch(app, /button\.addEventListener\("click", \(\) => downloadArtifact\(\{ name, data: generatedFiles/);
  assert.match(styles, /\.sidebar-resizer[^}]*cursor: ew-resize/);
  assert.match(styles, /\.panel-resizer[^}]*cursor: ns-resize/);
});

test("Go selector completion keeps working inside incomplete calls", () => {
  assert.match(app, /tabCompletion: "on"/);
  assert.match(app, /requestLanguage\("imports", model, offset, false\)/);
  assert.match(app, /catalogSelectorCompletions\(standardCatalog, parsed\.selector, parsed\.imports\)/);
  assert.match(app, /if \(catalogCompletion\)/);
  assert.doesNotMatch(app, /function importContextAt|function scanImports/);
});

test("Renvo quick open owns the global shortcuts and indexes Monaco, catalogs, and project definitions", () => {
  assert.match(html, /id="quick-open"/);
  assert.match(app, /function handleQuickOpenShortcut\(event\)/);
  assert.match(app, /if \(key === "p"\) value = event\.shiftKey \? ">" : ""/);
  assert.match(app, /editor\.getSupportedActions\(\)/);
  assert.match(app, /catalogFileItems\(standardCatalog\)/);
  assert.match(app, /quickOpenSymbolItems\(\[\.\.\.editableFiles\]\.sort\(\)\)/);
  assert.doesNotMatch(app, /editor\.trigger\("global", "editor\.action\.quickCommand"/);
  assert.match(app, /event\.code === "Backquote"/);
  assert.match(app, /showPanel\("terminal"\)/);
});

test("board actions run a user-visible flash command and stream its lifecycle in the terminal", () => {
  assert.match(app, /if \(isBoardTarget\(selectedTarget\)\) return submitTerminalCommand\(flashCommand\(elements\.flashTransport\.value\)\)/);
  assert.match(app, /if \(command === "flash"\) \{\s*await executeFlashCommand\(args\)/);
  assert.match(app, /function appendTerminalActivity\(text\)/);
  assert.match(app, /if \(terminalFlashActive\) writeFlashProgress\(message\.message\)/);
  assert.match(app, /writeFlashProgress\(`\$\{verb\} firmware: \$\{percent\}%`\)/);
  assert.match(app, /Firmware build complete in/);
  assert.match(app, /const resumesAfterBuild = await runArtifactWithMode\(false\);\s*if \(resumesAfterBuild\) await completion/);
  assert.match(app, /if \(building \|\| stale\) \{[\s\S]*?return true;\s*\}/);
  assert.match(app, /requestAnimationFrame\(\(\) => \{\s*terminalFitAddon\?\.fit\(\);\s*renderTerminalInput\(\)/);
  assert.doesNotMatch(app, /monitor \? "monitor-load" : jtag \? "jtag-load"/);
});

test("board application output has a serial tab separate from flash activity", () => {
  assert.match(html, /data-panel="serial"[^>]*>Serial</);
  assert.match(html, /id="serial-output"[^>]*data-panel-view="serial"/);
  assert.match(app, /function appendSerialText\(text\) \{\s*if \(elements\.serialOutput\.textContent/);
  assert.match(app, /elements\.serialOutput\.textContent \+= text/);
  assert.doesNotMatch(app, /function appendSerialText\(text\) \{\s*elements\.terminalOutput\.textContent \+= text/);
  assert.match(app, /elements\.mobileDeviceOutput\.textContent = serial/);
});

test("WASI Run and Ctrl+S build into the virtual filesystem before execution", () => {
  assert.match(app, /if \(isWASITarget\(\)\) return submitTerminalCommand\("run"\)/);
  assert.match(app, /if \(!args\.length\) \{\s*await runCurrentWASIProject\(\)/);
  assert.match(app, /const result = await runTerminalArguments\(buildArgs\)/);
  assert.match(app, /await runTerminalWASI\(output, args\)/);
  assert.match(app, /writeTerminalStage\("build", `renvo /);
  assert.match(app, /writeTerminalStage\("run", `\.\/\$\{output\}/);
  assert.doesNotMatch(app, /writeTerminal\(`\$ (?:renvo|\.\/)/);
  assert.match(app, /if \(isBoardTarget\(selectedTarget\) \|\| isWASITarget\(\)\) runArtifact\(\)/);
  assert.match(app, /elements\.run\.classList\.toggle\("primary-action", runIsPrimary\)/);
  assert.match(app, /elements\.compile\.classList\.toggle\("secondary-action", runIsPrimary\)/);
});
