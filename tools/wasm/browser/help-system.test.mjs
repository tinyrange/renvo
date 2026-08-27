import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
const index = await readFile(new URL("./index.html", import.meta.url), "utf8");

test("documentation opens in a native workbench tab and copies a complete page", () => {
  assert.match(index, /id="help-heading"/);
  assert.match(index, /id="help-view"/);
  assert.match(index, /id="copy-help-page"[^>]*>Copy docs</);
  assert.match(app, /function openHelpPage\(importPath\)/);
  assert.match(app, /openFiles\.includes\(activeHelp\)/);
  assert.match(app, /navigator\.clipboard\.writeText\(helpPageText\(page\)\)/);
  assert.match(app, /\["Constants", page\.constants\].*\["Types", page\.types\]/s);
  assert.match(app, /monaco\.editor\.colorize\(source, "go", \{ tabSize: 4 \}\)/);
  assert.match(app, /element\.textContent = source/);
  assert.match(app, /function openHelpSource\(page, entry\)/);
  assert.match(app, /editor\.revealPositionInCenter\(position\)/);
});

test("sidebar examples expose a read-only file tree and replace only after confirmation", () => {
  assert.match(index, /id="examples-heading"/);
  assert.match(index, /id="sidebar-examples"/);
  assert.match(app, /function renderSidebarExamples\(catalog = standardCatalog\)/);
  assert.match(app, /files\.replaceChildren\(\.\.\.entry\.item\.files\.map/);
  assert.match(app, /button\.addEventListener\("click", \(\) => viewExample\(entry, file\)\)/);
  assert.match(app, /toggle\.setAttribute\("aria-expanded"/);
  assert.match(app, /use\.textContent = "Use"/);
  assert.match(app, /accept: "Replace project"/);
  assert.match(app, /if \(!accepted\).*return;/);
});

test("the primary Examples action opens the hardware-first browser on every viewport", () => {
  assert.match(index, /id="browse-examples"/);
  assert.match(index, /id="example-dialog"/);
  assert.match(index, /<strong>What are you using\?<\/strong>/);
  assert.match(app, /querySelector\("#browse-examples"\)\.addEventListener\("click", openExampleBrowser\)/);
  assert.doesNotMatch(app, /#browse-examples[\s\S]{0,160}openSidebarExamples/);
});

test("external views round-trip through URL deep links", () => {
  assert.match(app, /const viewParameterNames = \["help", "source", "example", "file"\]/);
  assert.match(app, /history\.pushState\(\{ renvoView: true \}, "", url\)/);
  assert.match(app, /window\.addEventListener\("popstate", \(\) => \{ if \(deepLinksReady\) restoreDeepLink\(\); \}\)/);
  assert.match(app, /setViewDeepLink\("example", example\.importPath, example\.file\)/);
  assert.match(app, /setViewDeepLink\("source", name\)/);
  assert.match(app, /setViewDeepLink\("help", importPath\)/);
});

test("structured compiler errors suppress generic failure summaries", () => {
  assert.match(app, /if \(!problems\.some\(\(problem\) => problem\.code\)\) return problems/);
  assert.match(app, /frontend compilation\|build/);
});
