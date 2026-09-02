import { readFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { fileURLToPath, pathToFileURL } from "node:url";

const bundleRoot = path.resolve(process.argv[2] || "dist/pages");
const exampleFilter = process.argv[3] || "";
const bundleURL = pathToFileURL(bundleRoot + path.sep);
const standardRoot = new URL("stdlib/", bundleURL);

function bytes(data) {
  return data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength);
}

async function readJSON(url) {
  return JSON.parse(await readFile(fileURLToPath(url), "utf8"));
}

const targetCatalog = await readJSON(new URL("targets.json", bundleURL));
const standardCatalog = await readJSON(new URL("catalog.json", standardRoot));
let workerURL = new URL("worker.mjs", bundleURL);
try { await readFile(fileURLToPath(workerURL)); }
catch (error) {
  if (error?.code !== "ENOENT") throw error;
  workerURL = new URL("browser/worker.mjs", bundleURL);
}

const originalFetch = globalThis.fetch;
globalThis.fetch = async (input, init) => {
  const url = new URL(typeof input === "string" || input instanceof URL ? input : input.url);
  if (url.protocol !== "file:") return originalFetch(input, init);
  try {
    const data = await readFile(fileURLToPath(url));
    const type = url.pathname.endsWith(".wasm") ? "application/wasm" : "application/octet-stream";
    return new Response(data, { status: 200, headers: { "content-type": type } });
  } catch (error) {
    if (error?.code === "ENOENT") return new Response("not found", { status: 404 });
    throw error;
  }
};

let messageHandler;
let requestID = 0;
const pending = new Map();
globalThis.self = {
  addEventListener(type, handler) {
    if (type === "message") messageHandler = handler;
  },
  postMessage(message) {
    const key = message.id === undefined ? "init" : message.id;
    const request = pending.get(key);
    if (!request || message.type !== request.type) return;
    pending.delete(key);
    request.resolve(message);
  },
};

await import(workerURL);
if (!messageHandler) throw new Error("browser worker did not install its message handler");

function request(message, responseType) {
  const key = message.id === undefined ? "init" : message.id;
  return new Promise((resolve, reject) => {
    pending.set(key, { type: responseType, resolve, reject });
    Promise.resolve(messageHandler({ data: message })).catch((error) => {
      pending.delete(key);
      reject(error);
    });
  });
}

const ready = await request({
  type: "init",
  compiler: new URL(targetCatalog.compiler, bundleURL).href,
  linker: new URL(targetCatalog.linker, bundleURL).href,
  languageService: new URL(targetCatalog.languageService, bundleURL).href,
  backendJIT: new URL(targetCatalog.backendJIT, bundleURL).href,
  vmBackend: new URL(targetCatalog.vmBackend, bundleURL).href,
  terminalCompiler: new URL(targetCatalog.terminalCompiler, bundleURL).href,
  formatter: "",
}, "ready");
if (ready.type !== "ready") throw new Error("browser compiler did not become ready");

async function addFile(files, name, url) {
  if (files.has(name)) return;
  files.set(name, bytes(await readFile(fileURLToPath(url))));
}

async function addPackage(files, importPath, loading = new Set()) {
  let name = importPath;
  if (name.startsWith("renvo.dev/std/")) name = name.slice("renvo.dev/std/".length);
  const platform = standardCatalog.platforms?.[importPath];
  const item = platform || standardCatalog.packages?.[name];
  if (!item) return;
  const key = platform ? importPath : name;
  if (loading.has(key)) return;
  loading.add(key);
  const root = platform ? `module/${item.root}` : `src/${name}`;
  const prefix = platform ? item.root : `std/${name}`;
  for (const file of item.files || []) {
    await addFile(files, `${prefix}/${file}`, new URL(`${root}/${file}`, standardRoot));
  }
  for (const dependency of item.imports || []) await addPackage(files, dependency, loading);
}

async function checkParsedImportContext() {
  const source = `package main

import "os"

func main() {
	os.WriteFile("hello.txt", []byte("Hello, World"), os.
}
`;
  const files = new Map([
    ["main.go", bytes(new TextEncoder().encode(source))],
    ["go.mod", bytes(new TextEncoder().encode("module renvo.dev\n\ngo 1.20\n"))],
  ]);
  const result = await request({
    type: "imports", id: ++requestID, files: workerFiles(files), workspaceRevision: 1,
    target: "wasi/wasm32", tags: ["wasi", "wasip1", "wasm", "wasm32"], file: "main.go",
    offset: new TextEncoder().encode(source.slice(0, source.indexOf("os.\n") + 3)).length,
    language: "go", packageAt: ".",
  }, "language-result");
  if (!result.output.split("\n").includes("I\t\tos") || result.output.split("\n").some((line) => line.startsWith("P\t")) ||
      !result.output.split("\n").includes("Q\tos\t\t" + String(source.indexOf("os.\n") + 3))) {
    throw new Error(`parsed import context confused call strings with imports: ${JSON.stringify(result.output)}`);
  }
  process.stdout.write("PASS parser-owned import context after string arguments\n");
}

async function exampleFiles(importPath, item) {
  const files = new Map();
  for (const file of item.files || []) {
    await addFile(files, file, new URL(`module/${item.root}/${file}`, standardRoot));
  }
  for (const [name, data] of [...files]) {
    if (!name.endsWith(".rbe")) continue;
    const source = new TextDecoder().decode(data);
    const section = /^[ \t]*@stdlib[ \t]+"([^"\r\n]+)"[ \t]*\r?\n([\s\S]*?)^[ \t]*@endstdlib[ \t]*(?:\r?\n|$)/gm;
    for (const match of source.matchAll(section)) {
      files.set(`std/${match[1]}`, bytes(new TextEncoder().encode(match[2])));
    }
  }
  if ([...files.keys()].some((name) => name.endsWith(".go")) && !files.has("go.mod")) {
    files.set("go.mod", bytes(new TextEncoder().encode(standardCatalog.module)));
  }
  for (const dependency of item.imports || []) await addPackage(files, dependency);
  if (item.language === "c") {
    for (const file of standardCatalog.libc || []) {
      await addFile(files, `libc/${file}`, new URL(`libc/${file}`, standardRoot));
    }
    await addPackage(files, "unsafe");
  }
  return files;
}

function workerFiles(files) {
  return [...files].map(([name, data]) => ({ name, data: data.slice(0) }));
}

async function compileStarterCObject() {
  const target = targetCatalog.targets.find((candidate) => candidate.name === "linux/amd64");
  if (!target?.cBackend) throw new Error("browser bundle has no Linux C backend");
  const files = new Map([["main.c", bytes(new TextEncoder().encode(`#include <stdio.h>
int main(void) {
    printf("Hello from Renvo C!\\n");
    return 0;
}
`))]]);
  for (const file of standardCatalog.libc || []) {
    await addFile(files, `libc/${file}`, new URL(`libc/${file}`, standardRoot));
  }
  const args = ["cc"];
  for (const tag of target.tags || []) args.push("-tags", tag);
  args.push("-t", target.name, "-s", "-c", "main.c", "-o", "main.o");
  const result = await request({
    type: "compile", id: ++requestID, args, files: workerFiles(files),
    backend: new URL(target.cBackend, bundleURL).href,
    backendTarget: target.backendTarget || target.name, backendFormat: target.backendFormat || "wasm",
  }, "result");
  if (result.exitCode !== 0) {
    const diagnostic = [result.stderr, result.stdout].filter(Boolean).join("\n").trim();
    throw new Error(`starter C object failed:\n${diagnostic}`);
  }
  const artifact = result.files.find((file) => file.name === "main.o");
  if (!artifact || artifact.data.byteLength === 0) throw new Error("starter C object produced no main.o");
  const header = new Uint8Array(artifact.data, 0, Math.min(18, artifact.data.byteLength));
  if (header.length < 18 || header[0] !== 0x7f || header[1] !== 0x45 || header[2] !== 0x4c || header[3] !== 0x46 ||
      new DataView(header.buffer, header.byteOffset, header.byteLength).getUint16(16, true) !== 1) {
    throw new Error("starter C object is not an ELF relocatable object");
  }
  process.stdout.write("PASS starter C object (linux/amd64)\n");
}

async function compileMacOSArtifact() {
  const target = targetCatalog.targets.find((candidate) => candidate.name === "darwin/arm64");
  if (!target?.backend) throw new Error("browser bundle has no macOS backend");
  const files = new Map([
    ["main.go", bytes(new TextEncoder().encode('package main\n\nfunc main() { print("PASS\\n") }\n'))],
    ["go.mod", bytes(new TextEncoder().encode("module renvo.dev\n\ngo 1.20\n"))],
  ]);
  const args = [];
  for (const tag of target.tags || []) args.push("-tags", tag);
  args.push("-t", target.name, "-s", "-o", target.output, ".");
  const result = await request({
    type: "compile", id: ++requestID, args, files: workerFiles(files),
    backend: new URL(target.backend, bundleURL).href,
    backendTarget: target.backendTarget || target.name,
    backendFormat: target.backendFormat || "wasm",
  }, "result");
  if (result.exitCode !== 0) throw new Error(result.stderr || "macOS browser build failed");
  const artifact = result.files.find((file) => file.name === target.output);
  const magic = artifact && new Uint8Array(artifact.data, 0, Math.min(4, artifact.data.byteLength));
  if (!magic || magic.length < 4 || magic[0] !== 0xcf || magic[1] !== 0xfa || magic[2] !== 0xed || magic[3] !== 0xfe) {
    throw new Error("macOS browser build did not produce an arm64 Mach-O executable");
  }
  process.stdout.write("PASS native artifact (darwin/arm64)\n");
}

async function compileTerminalCLIArtifact() {
  if (!targetCatalog.terminalCompiler) throw new Error("browser bundle has no full terminal compiler");
  const encoder = new TextEncoder();
  const files = new Map([
    ["main.go", bytes(encoder.encode(`package main

import "os"

func main() {
	if os.WriteFile("hello.txt", []byte("Hello, World"), os.ModePerm) != nil {
		print("FAIL\\n")
		return
	}
	print("PASS\\n")
}
`))],
    ["go.mod", bytes(encoder.encode("module renvo.dev\n\ngo 1.20\n"))],
    ["Makefile", bytes(encoder.encode("all: terminal-app\nterminal-app: main.go go.mod\n\trenvo -t darwin/arm64 -s -o terminal-app .\n"))],
  ]);
  const workspaceNames = [...files.keys()];
  await addPackage(files, "os");
  const internalNames = [...files.keys()].filter((name) => !workspaceNames.includes(name));
  const help = await request({ type: "terminal", id: ++requestID, args: ["--help"], files: workerFiles(files), workspaceNames, internalNames }, "result");
  if (help.exitCode !== 0 || !help.stdout.includes("renvo make") || !help.stdout.includes("-backend")) {
    throw new Error(help.stderr || "full terminal Renvo help is incomplete");
  }
  const made = await request({ type: "terminal", id: ++requestID, args: ["make"], files: workerFiles(files), workspaceNames, internalNames }, "result");
  const artifact = made.files.find((file) => file.name === "terminal-app");
  const magic = artifact && new Uint8Array(artifact.data, 0, Math.min(4, artifact.data.byteLength));
  if (made.exitCode !== 0 || !made.filesystemComplete || !magic || magic[0] !== 0xcf || magic[1] !== 0xfa || magic[2] !== 0xed || magic[3] !== 0xfe) {
    throw new Error(made.stderr || "terminal Renvo make did not retain its Mach-O output");
  }
  const compiled = await request({
    type: "terminal", id: ++requestID, args: ["-t", "wasi/wasm32", "-s", "-o", "app.wasm", "."],
    files: workerFiles(files), workspaceNames, internalNames,
  }, "result");
  const app = compiled.files.find((file) => file.name === "app.wasm");
  if (compiled.exitCode !== 0 || !app) throw new Error(compiled.stderr || "terminal Renvo did not produce a WASI app");
  const runFiles = new Map(files); runFiles.set("app.wasm", app.data.slice(0));
  const ran = await request({
    type: "terminal-run", id: ++requestID, name: "app.wasm", args: [], files: workerFiles(runFiles),
    workspaceNames, internalNames, directories: [],
  }, "terminal-run-result");
  if (ran.exitCode !== 0 || ran.stdout !== "PASS\n" || !ran.filesystemComplete) {
    throw new Error(ran.stderr || `terminal WASI run returned ${JSON.stringify(ran.stdout)}`);
  }
  const written = ran.files.find((file) => file.name === "hello.txt");
  if (!written || new TextDecoder().decode(written.data) !== "Hello, World") throw new Error("terminal WASI run did not retain os.WriteFile output");
  process.stdout.write("PASS full terminal CLI, virtual output filesystem, and WASI execution\n");
}

async function prepareProjectTargets(files) {
  const definitions = [...files.keys()].filter((name) =>
    (name.endsWith(".rtg") || name.endsWith(".rbe")) && !name.includes("/"));
  const targets = new Map();
  for (const definition of definitions) {
    const inspected = await request({
      type: "backend-inspect", id: ++requestID, definition, files: workerFiles(files),
    }, "backend-result");
    if (inspected.exitCode !== 0) throw new Error(inspected.stderr.trim() || `could not inspect ${definition}`);
    const manifests = JSON.parse(inspected.stdout);
    for (const candidate of manifests) {
      const prepared = await request({
        type: "backend-prepare", id: ++requestID, definition, target: candidate.name, files: workerFiles(files),
      }, "backend-result");
      if (prepared.exitCode !== 0 || !prepared.compiler) {
        throw new Error(prepared.stderr.trim() || `could not prepare ${candidate.name}`);
      }
      const manifest = JSON.parse(prepared.stdout);
      targets.set(manifest.name, {
        ...manifest,
        backend: URL.createObjectURL(new Blob([prepared.compiler], { type: "application/octet-stream" })),
        backendFormat: "vm32",
        projectDefinition: definition,
      });
    }
  }
  return targets;
}

function exampleTargets(item) {
  const names = [
    ...(item.boards || []).map((board) => board.target),
    ...(item.computers || []).map((computer) => computer.target),
    item.target,
  ].filter(Boolean);
  return [...new Set(names)];
}

const published = Object.entries(standardCatalog.platforms || {})
  .filter(([importPath, item]) => item.main && !item.hidden &&
    (!exampleFilter || importPath.includes(exampleFilter)))
  .sort(([left], [right]) => left.localeCompare(right));
let compiled = 0;
let monitorCompiled = 0;
const failures = [];
await checkParsedImportContext();
await compileStarterCObject();
await compileMacOSArtifact();
await compileTerminalCLIArtifact();
for (const [importPath, item] of published) {
  const files = await exampleFiles(importPath, item);
  const projectTargets = await prepareProjectTargets(files);
  for (const targetName of exampleTargets(item)) {
    const target = targetCatalog.targets.find((candidate) => candidate.name === targetName) || projectTargets.get(targetName);
    if (!target) throw new Error(`${importPath} publishes unavailable target ${targetName}`);
    const targetFiles = new Map(files);
    for (const library of target.libraryFiles || []) {
      targetFiles.set(`std/${library.name}`, bytes(new TextEncoder().encode(library.source)));
    }
    const args = [];
    if (item.language === "c") args.push("cc");
    for (const tag of target.tags || []) args.push("-tags", tag);
    if (target.definition) {
      args.push("-target-definition", target.definition, "-target-version", String(target.descriptorVersion));
    }
    args.push("-t", target.frontendTarget || target.name);
    if (item.arenaSize) args.push("-arena-size", String(item.arenaSize));
    args.push("-s", "-o", target.output || "app", ".");
    const backendPath = item.language === "c" && target.cBackend ? target.cBackend : target.backend;
    const backend = backendPath.includes(":") ? backendPath : new URL(backendPath, bundleURL).href;
    const started = performance.now();
    const result = await request({
      type: "compile", id: ++requestID, args, files: workerFiles(targetFiles),
      backend, backendTarget: target.backendTarget || target.name, backendFormat: target.backendFormat || "wasm",
      rtgDefinition: target.rtgDefinition ? new URL(target.rtgDefinition, bundleURL).href : "",
      rtgDefinitionName: target.rtgDefinitionName || target.projectDefinition || "",
      rtgImports: (target.rtgImports || []).map((entry) => ({
        ...entry, source: new URL(entry.source, bundleURL).href,
      })),
    }, "result");
    if (result.exitCode !== 0) {
      const diagnostic = [result.stderr, result.stdout].filter(Boolean).join("\n").trim();
      failures.push(`${importPath} (${targetName}) failed:\n${diagnostic}`);
      process.stdout.write(`FAIL ${importPath} (${targetName})\n`);
      continue;
    }
    const artifact = result.files.find((file) => file.name === (target.output || "app"));
    if (!artifact || artifact.data.byteLength === 0) {
      throw new Error(`${importPath} (${targetName}) produced no ${target.output || "app"}`);
    }
    compiled++;
    process.stdout.write(`PASS ${importPath} (${targetName}) ${(performance.now() - started).toFixed(0)} ms\n`);

    if (target.device === "rp2" && item.language !== "c") {
      const debugTarget = targetCatalog.targets.find((candidate) => candidate.name === "rp2-debug/thumb");
      if (!debugTarget) throw new Error("browser bundle has no rp2-debug/thumb target");
      const debugArgs = [];
      for (const tag of new Set([...(target.tags || []), ...(debugTarget.tags || [])])) debugArgs.push("-tags", tag);
      if (debugTarget.definition) {
        debugArgs.push("-target-definition", debugTarget.definition,
          "-target-version", String(debugTarget.descriptorVersion));
      }
      debugArgs.push("-t", debugTarget.frontendTarget || debugTarget.name,
        "-s", "-o", debugTarget.output || "app.elf", ".");
      const debugBackend = new URL(debugTarget.backend, bundleURL).href;
      const debugResult = await request({
        type: "compile", id: ++requestID, args: debugArgs, files: workerFiles(files),
        backend: debugBackend, backendTarget: debugTarget.backendTarget || debugTarget.name,
        backendFormat: debugTarget.backendFormat || "wasm",
        rtgDefinition: debugTarget.rtgDefinition ? new URL(debugTarget.rtgDefinition, bundleURL).href : "",
        rtgDefinitionName: debugTarget.rtgDefinitionName || debugTarget.projectDefinition || "",
        rtgImports: (debugTarget.rtgImports || []).map((entry) => ({
          ...entry, source: new URL(entry.source, bundleURL).href,
        })),
      }, "result");
      if (debugResult.exitCode !== 0) {
        const diagnostic = [debugResult.stderr, debugResult.stdout].filter(Boolean).join("\n").trim();
        failures.push(`${importPath} (${targetName}, monitor load) failed:\n${diagnostic}`);
        process.stdout.write(`FAIL ${importPath} (${targetName}, monitor load)\n`);
      } else {
        const debugArtifact = debugResult.files.find((file) => file.name === (debugTarget.output || "app.elf"));
        if (!debugArtifact || debugArtifact.data.byteLength === 0) {
          throw new Error(`${importPath} (${targetName}, monitor load) produced no ${debugTarget.output || "app.elf"}`);
        }
        monitorCompiled++;
        process.stdout.write(`PASS ${importPath} (${targetName}, monitor load)\n`);
      }
    }
  }
}

process.stdout.write(`Compiled ${compiled} published browser example/target combinations.\n`);
process.stdout.write(`Compiled ${monitorCompiled} RP2 monitor-load combinations.\n`);
if (failures.length) throw new Error(failures.join("\n\n"));
