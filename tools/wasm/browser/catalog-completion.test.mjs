import assert from "node:assert/strict";
import test from "node:test";

import { catalogSelectorCompletions } from "./catalog-completion.mjs";

test("catalog completion answers an incomplete imported package selector immediately", () => {
  const source = `package main

import "os"

func main() {
	os.WriteFile("hello.txt", []byte("Hello, World"), os.
}`;
  const catalog = { packages: { os: { docs: {
    constants: [{ name: "ModePerm", signature: "const ModePerm FileMode = 0o777" }],
    functions: [{ name: "WriteFile", signature: "func WriteFile(name string, data []byte, perm FileMode) error" }],
    variables: [{ name: "Stdout", signature: "var Stdout *File" }],
    types: [{ name: "FileMode", signature: "type FileMode uint32" }],
  } } } };
  const result = catalogSelectorCompletions(catalog, { base: "os", prefix: "" }, [{ name: "os", importPath: "os" }]);
  assert.deepEqual(result.items.map((item) => item.name), ["ModePerm", "Stdout", "WriteFile", "FileMode"]);
});

test("catalog completion respects aliases and selector prefixes", () => {
  const source = `package main\nimport host "os"\nvar _ = host.Wri`;
  const catalog = { packages: { os: { docs: { functions: [{ name: "WriteFile" }, { name: "ReadFile" }] } } } };
  const result = catalogSelectorCompletions(catalog, { base: "host", prefix: "Wri" }, [{ name: "host", importPath: "os" }]);
  assert.equal(result.importPath, "os");
  assert.deepEqual(result.items.map((item) => item.name), ["WriteFile"]);
});
