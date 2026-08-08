#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

function usage() {
  console.error("usage: node tools/wasm/compile.mjs [--workspace DIR] FRONTEND.wasm BACKEND.wasm -- [compiler arguments...]");
}

const args = process.argv.slice(2);
let workspace = process.cwd();
if (args[0] === "--workspace" && args.length >= 2) {
  args.shift();
  workspace = path.resolve(args.shift());
}
if (args.length < 2) {
  usage();
  process.exit(2);
}
const frontend = path.resolve(args.shift());
const backend = path.resolve(args.shift());
if (args[0] === "--") args.shift();

let output = "";
let emitUnit = false;
let strip = false;
let arenaSize = "";
const frontendArgs = [];
for (let i = 0; i < args.length; i++) {
  if (args[i] === "-o" && i + 1 < args.length) {
    output = args[++i];
    frontendArgs.push("-o", output);
  } else if (args[i] === "-arena-size" && i + 1 < args.length) {
    arenaSize = args[++i];
  } else {
    if (args[i] === "-emit-unit") emitUnit = true;
    if (args[i] === "-s") strip = true;
    frontendArgs.push(args[i]);
  }
}
if (!output) {
  console.error("renvo: missing output path (-o)");
  process.exit(2);
}

const runner = path.join(path.dirname(fileURLToPath(import.meta.url)), "run.mjs");
if (emitUnit) {
  process.exitCode = run(frontend, frontendArgs);
} else {
  const temporary = `.renvo-wasm-${process.pid}-${Date.now()}.unit`;
  const outputAt = frontendArgs.indexOf("-o") + 1;
  frontendArgs[outputAt] = temporary;
  try {
    const frontendExit = run(frontend, frontendArgs);
    if (frontendExit !== 0) {
      process.exitCode = frontendExit;
    } else {
      const backendArgs = [];
      if (strip) backendArgs.push("-s");
      if (arenaSize) backendArgs.push("-arena-size", arenaSize);
      backendArgs.push("-o", output, temporary);
      process.exitCode = run(backend, backendArgs);
    }
  } finally {
    fs.rmSync(path.join(workspace, temporary), { force: true });
  }
}

function run(module, moduleArgs) {
  const child = spawnSync(process.execPath, [runner, "--workspace", workspace, module, "--", ...moduleArgs], {
    cwd: workspace,
    stdio: "inherit",
  });
  return child.status ?? 1;
}
