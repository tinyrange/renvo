import assert from "node:assert/strict";
import test from "node:test";

import { parsePlotLine, SerialPlotter } from "./serial-plotter.mjs";

test("parses Arduino labelled and unlabelled plot records", () => {
  assert.deepEqual(parsePlotLine("Temperature_C:25.54\tHumidity_pct:52.634,Gas_kOhm:17.8"), [
    { name: "Temperature_C", value: 25.54 },
    { name: "Humidity_pct", value: 52.634 },
    { name: "Gas_kOhm", value: 17.8 },
  ]);
  assert.deepEqual(parsePlotLine("1.5 -2 3e2"), [
    { name: "Value 1", value: 1.5 }, { name: "Value 2", value: -2 }, { name: "Value 3", value: 300 },
  ]);
});

test("rejects ordinary serial prose containing numbers", () => {
  assert.deepEqual(parsePlotLine("temperature=25.54 C, pressure=102935 Pa"), []);
  assert.deepEqual(parsePlotLine("status: ready after 10 seconds"), []);
  assert.deepEqual(parsePlotLine("value:12 trailing prose"), []);
});

test("buffers fragmented lines and bounds each series history", () => {
  const changes = [];
  const plotter = new SerialPlotter({ capacity: 2, onChange: (data) => changes.push(data) });
  assert.equal(plotter.push("A:1 B:"), false);
  assert.equal(plotter.push("2\r\nnoise\nA:3 B:4\nA:5\n"), true);
  const snapshot = plotter.snapshot();
  assert.equal(snapshot.sample, 3);
  assert.deepEqual(snapshot.series[0], { name: "A", points: [{ sample: 2, value: 3 }, { sample: 3, value: 5 }] });
  assert.deepEqual(snapshot.series[1], { name: "B", points: [{ sample: 1, value: 2 }, { sample: 2, value: 4 }] });
  assert.equal(changes.length, 1);
});
