const testName = /^Test(?:$|[^a-z])/;

export function generateBrowserTestProject(files) {
  const goFiles = Object.entries(files).filter(([name]) => name.endsWith(".go"));
  const nativeFiles = Object.entries(files).filter(([name]) => /\.(?:c|h)$/i.test(name));
  const tests = [];
  const generated = {};
  let packageName = "";
  for (const [name, source] of goFiles) {
    const match = /^\s*package\s+([A-Za-z_]\w*)/m.exec(source);
    if (!match) continue;
    if (!packageName) packageName = match[1];
    if (match[1] !== packageName) throw new Error("External test packages are not supported in the browser runner.");
    for (const declaration of source.matchAll(/\bfunc\s+(Test[A-Za-z0-9_]*)\s*\(\s*(?:[A-Za-z_]\w*\s+)?\*\s*(?:[A-Za-z_]\w*\s*\.\s*)?T\s*\)/g)) {
      if (testName.test(declaration[1])) tests.push(declaration[1]);
    }
    let rewritten = source.slice(0, match.index) + source.slice(match.index).replace(/^\s*package\s+[A-Za-z_]\w*/m, "package main");
    if (!name.endsWith("_test.go") && packageName === "main") rewritten = rewritten.replace(/\bfunc\s+main\s*\(/, "func renvoProgramMain(");
    const base = name.split("/").pop().replace(/_test\.go$/, "_testsrc.go");
    generated[`renvo_tests/${base}`] = rewritten;
  }
  for (const [name, source] of nativeFiles) generated[`renvo_tests/${name}`] = source;
  tests.sort();
  if (!tests.length) throw new Error("No TestXxx functions were found.");
  generated["renvo_tests/renvo_testmain.go"] = `package main

import "testing"

func main() {
	failed := false
${tests.map((name) => `\tif !testing.RunTest(${JSON.stringify(name)}, ${name}) { failed = true }`).join("\n")}
	testing.Finish(failed)
}
`;
  return { files: generated, tests };
}
