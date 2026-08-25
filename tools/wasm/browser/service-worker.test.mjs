import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const source = await readFile(new URL("./service-worker.mjs", import.meta.url), "utf8");

test("versioned assets use the cache before the network", () => {
  assert.match(source, /url\.searchParams\.has\("v"\)/);
  assert.match(source, /caches\.match\(event\.request\).*cached \|\| fetch\(event\.request\)/s);
});
