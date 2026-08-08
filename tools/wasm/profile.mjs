#!/usr/bin/env node

import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

function usage() {
  console.error("usage: node tools/wasm/profile.mjs FRONTEND.wasm [--backend BACKEND.wasm] [--workspace DIR] [--json] [--check]");
}

const args = process.argv.slice(2);
if (args.length === 0) {
  usage();
  process.exit(2);
}
const compiler = path.resolve(args.shift());
let workspace = process.cwd();
let backend = "";
let jsonOnly = false;
let check = false;
while (args.length > 0) {
  const option = args.shift();
  if (option === "--workspace" && args.length > 0) {
    workspace = path.resolve(args.shift());
  } else if (option === "--backend" && args.length > 0) {
    backend = path.resolve(args.shift());
  } else if (option === "--json") {
    jsonOnly = true;
  } else if (option === "--check") {
    check = true;
  } else {
    usage();
    process.exit(2);
  }
}

const runner = path.join(path.dirname(fileURLToPath(import.meta.url)), "run.mjs");
const temporary = fs.mkdtempSync(path.join(workspace, ".renvo-wasm-profile-"));
const temporaryRelative = path.relative(workspace, temporary);
const cases = [
  { name: "tiny", args: ["-o", `${temporaryRelative}/tiny.unit`, "./tools/wasm/testdata"] },
  { name: "repl", args: ["-o", `${temporaryRelative}/repl.unit`, "./cmd/renvorepl"] },
  { name: "selfhost", args: ["-tags", "renvo_wasi_frontend", "-o", `${temporaryRelative}/renvo.unit`, "./cmd/renvowasi"] },
];

const results = [];
try {
  for (const test of cases) {
    const child = spawnSync(process.execPath, [runner, "--workspace", workspace, "--profile", compiler, "--", ...test.args], {
      cwd: workspace,
      encoding: "utf8",
      maxBuffer: 16 * 1024 * 1024,
    });
    const profileLine = child.stderr.split("\n").find((line) => line.startsWith("RENVO_WASM_PROFILE "));
    if (child.status !== 0 || !profileLine) {
      process.stderr.write(child.stderr);
      process.stderr.write(child.stdout);
      throw new Error(`${test.name} exited with status ${child.status}`);
    }
    const profile = JSON.parse(profileLine.slice("RENVO_WASM_PROFILE ".length));
    const output = test.args[test.args.indexOf("-o") + 1];
    profile.outputBytes = fs.statSync(path.join(workspace, output)).size;
    if (backend) {
      const executable = `${temporaryRelative}/${test.name}.wasm`;
      const backendArgs = [runner, "--workspace", workspace, "--profile", backend, "--", "-s", "-arena-size", String(160 * 1024 * 1024), "-o", executable, output];
      const backendChild = spawnSync(process.execPath, backendArgs, {
        cwd: workspace,
        encoding: "utf8",
        maxBuffer: 16 * 1024 * 1024,
      });
      const backendLine = backendChild.stderr.split("\n").find((line) => line.startsWith("RENVO_WASM_PROFILE "));
      if (backendChild.status !== 0 || !backendLine) {
        process.stderr.write(backendChild.stderr);
        process.stderr.write(backendChild.stdout);
        throw new Error(`${test.name} backend exited with status ${backendChild.status}`);
      }
      const backendProfile = JSON.parse(backendLine.slice("RENVO_WASM_PROFILE ".length));
      profile.backendExecuteMilliseconds = backendProfile.executeMilliseconds;
      profile.backendModuleCompileMilliseconds = backendProfile.moduleCompileMilliseconds;
      profile.backendCpuUserMilliseconds = backendProfile.cpuUserMilliseconds;
      profile.backendCpuSystemMilliseconds = backendProfile.cpuSystemMilliseconds;
      profile.backendMaxRssBytes = backendProfile.maxRssBytes;
      profile.backendLinearMemoryBytes = backendProfile.linearMemoryBytes;
      profile.executableBytes = fs.statSync(path.join(workspace, executable)).size;
      profile.pipelineMilliseconds = profile.executeMilliseconds + backendProfile.executeMilliseconds;
    }
    results.push({ name: test.name, ...profile });
  }
} finally {
  fs.rmSync(temporary, { recursive: true, force: true });
}

if (check) {
  const sizeLimit = 2 * 1024 * 1024;
  const selfhostLimit = 1000;
  const compilerBytes = fs.statSync(compiler).size;
  const selfhost = results.find((result) => result.name === "selfhost");
  const selfhostMilliseconds = selfhost.pipelineMilliseconds ?? selfhost.executeMilliseconds;
  if (compilerBytes > sizeLimit || selfhostMilliseconds > selfhostLimit) {
    console.error(`WASM gate failed: frontend=${compilerBytes}/${sizeLimit} bytes selfhost=${selfhostMilliseconds.toFixed(1)}/${selfhostLimit} ms`);
    process.exitCode = 1;
  }
}

if (jsonOnly) {
  console.log(JSON.stringify({ node: process.version, platform: `${process.platform}/${process.arch}`, results }, null, 2));
} else {
  console.log(`Node ${process.version} on ${process.platform}/${process.arch}`);
  console.log("case              execute     CPU       peak RSS   linear memory  output");
  for (const result of results) {
    const cpu = result.cpuUserMilliseconds + result.cpuSystemMilliseconds +
      (result.backendCpuUserMilliseconds ?? 0) + (result.backendCpuSystemMilliseconds ?? 0);
    const rss = Math.max(result.maxRssBytes, result.backendMaxRssBytes ?? 0);
    const linear = Math.max(result.linearMemoryBytes, result.backendLinearMemoryBytes ?? 0);
    const outputBytes = result.executableBytes ?? result.outputBytes;
    console.log(
      result.name.padEnd(17) +
      `${(result.pipelineMilliseconds ?? result.executeMilliseconds).toFixed(1)} ms`.padStart(10) +
      `${cpu.toFixed(1)} ms`.padStart(10) +
      `${formatBytes(rss)}`.padStart(11) +
      `${formatBytes(linear)}`.padStart(16) +
      `${formatBytes(outputBytes)}`.padStart(9),
    );
  }
  if (backend) {
    console.log(`\nPipeline execute time is frontend + backend; backend module ${formatBytes(fs.statSync(backend).size)}.`);
  }
  console.log("\nUse --json for phase timings, memory counters, and WASI syscall counts.");
}

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
}
