import assert from "node:assert/strict";
import test from "node:test";
import { generateBrowserTestProject } from "./test-project.mjs";

test("browser test projects rewrite package files and create a runner", () => {
  const result = generateBrowserTestProject({
    "calc.go": "package calc\nfunc add(a, b int) int { return a+b }\n",
    "calc_test.go": "package calc\nimport \"testing\"\nfunc TestAdd(t *testing.T) { if add(1,2)!=3 { t.Fail() } }\n",
    "native/add.c": "int add_native(int a, int b) { return a + b; }\n",
    "native/add.h": "int add_native(int a, int b);\n",
  });
  assert.deepEqual(result.tests, ["TestAdd"]);
  assert.match(result.files["renvo_tests/calc.go"], /^package main/m);
  assert.match(result.files["renvo_tests/renvo_testmain.go"], /testing\.RunTest\("TestAdd", TestAdd\)/);
  assert.equal(result.files["renvo_tests/native/add.c"], "int add_native(int a, int b) { return a + b; }\n");
  assert.equal(result.files["renvo_tests/native/add.h"], "int add_native(int a, int b);\n");
});
