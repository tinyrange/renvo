import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const styles = await readFile(new URL("./styles.css", import.meta.url), "utf8");
const app = await readFile(new URL("./app.mjs", import.meta.url), "utf8");
const plotter = await readFile(new URL("./serial-plotter.mjs", import.meta.url), "utf8");

function contrast(first, second) {
  const luminance = (hex) => {
    const channels = hex.match(/[0-9a-f]{2}/gi).map((value) => Number.parseInt(value, 16) / 255);
    const [red, green, blue] = channels.map((value) => value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4);
    return 0.2126 * red + 0.7152 * green + 0.0722 * blue;
  };
  const values = [luminance(first), luminance(second)].sort((left, right) => right - left);
  return (values[0] + 0.05) / (values[1] + 0.05);
}

test("theme keeps its nature palette as its anchors", () => {
  for (const color of ["#596f3f", "#dccde8", "#23001e", "#45503b", "#b6244f"]) {
    assert.match(styles, new RegExp(color));
  }
});

test("theme text and actions retain readable contrast", () => {
  assert.ok(contrast("#d8cae2", "#1b0718") >= 7);
  assert.ok(contrast("#f4ebf7", "#596f3f") >= 4.5);
  assert.ok(contrast("#a894ac", "#160512") >= 4.5);
});

test("Monaco and the plotter use the Renvo theme", () => {
  assert.match(app, /"editor\.background": "#1b0718"/);
  assert.match(app, /"editor\.selectionBackground": "#596f3f99"/);
  assert.match(app, /token: "comment", foreground: "8D9E7E"/);
  assert.match(plotter, /context\.fillStyle = "#12010f"/);

  const retired = /#(?:181818|1e1e1e|2a2d2e|37373d|007fd4|75beff|4ec9b0|c586c0|f48771)\b/i;
  assert.doesNotMatch(styles, retired);
  assert.doesNotMatch(app, retired);
  assert.doesNotMatch(plotter, retired);
});
