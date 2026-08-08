#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { performance } from "node:perf_hooks";
import { WASI } from "node:wasi";

function usage() {
  console.error("usage: node tools/wasm/run.mjs [--workspace DIR] [--profile] COMPILER.wasm [--] [compiler arguments...]");
}

function parseArguments(args) {
  let workspace = process.cwd();
  let profile = false;
  let at = 0;
  for (; at < args.length; at++) {
    if (args[at] === "--workspace" && at + 1 < args.length) {
      workspace = args[++at];
      continue;
    }
    if (args[at] === "--profile") {
      profile = true;
      continue;
    }
    break;
  }
  if (at >= args.length) return null;
  const compiler = args[at++];
  if (args[at] === "--") at++;
  return { compiler, compilerArgs: args.slice(at), profile, workspace };
}

const options = parseArguments(process.argv.slice(2));
if (!options) {
  usage();
  process.exit(2);
}

const compilerPath = path.resolve(options.compiler);
const workspace = path.resolve(options.workspace);
const cpuStarted = process.cpuUsage();
const started = performance.now();
const binary = fs.readFileSync(compilerPath);
const loaded = performance.now();
const module = await WebAssembly.compile(binary);
const compiled = performance.now();
const wasi = new WASI({
  version: "preview1",
  args: ["renvo", ...options.compilerArgs],
  env: { PWD: "." },
  preopens: { ".": workspace },
  returnOnExit: true,
});
const imports = wasi.getImportObject();
const syscallCounts = {};
if (options.profile) {
  for (const [name, original] of Object.entries(imports.wasi_snapshot_preview1)) {
    imports.wasi_snapshot_preview1[name] = (...args) => {
      syscallCounts[name] = (syscallCounts[name] ?? 0) + 1;
      return original(...args);
    };
  }
}
const instance = await WebAssembly.instantiate(module, imports);
const instantiated = performance.now();
const exitCode = wasi.start(instance);
const finished = performance.now();

if (options.profile) {
  const cpu = process.cpuUsage(cpuStarted);
  const resources = process.resourceUsage();
  const memory = process.memoryUsage();
  console.error("RENVO_WASM_PROFILE " + JSON.stringify({
    exitCode,
    wasmBytes: binary.length,
    linearMemoryBytes: instance.exports.memory?.buffer.byteLength ?? 0,
    loadMilliseconds: loaded - started,
    moduleCompileMilliseconds: compiled - loaded,
    instantiateMilliseconds: instantiated - compiled,
    executeMilliseconds: finished - instantiated,
    totalMilliseconds: finished - started,
    cpuUserMilliseconds: cpu.user / 1000,
    cpuSystemMilliseconds: cpu.system / 1000,
    maxRssBytes: resources.maxRSS * 1024,
    heapUsedBytes: memory.heapUsed,
    externalBytes: memory.external,
    syscallCounts,
  }));
}

process.exitCode = exitCode;
