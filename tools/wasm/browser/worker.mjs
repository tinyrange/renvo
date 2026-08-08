let frontendModule;
let backendModule;
let compilerError;
const encoder = new TextEncoder();
const decoder = new TextDecoder();

self.addEventListener("message", async (event) => {
  const request = event.data;
  if (request.type === "init") {
    try {
      [frontendModule, backendModule] = await Promise.all([
        loadModule(request.compiler, "frontend"),
        loadModule(request.backend, "backend"),
      ]);
      self.postMessage({ type: "ready" });
    } catch (error) {
      compilerError = error;
      throw error;
    }
    return;
  }
  if (request.type !== "compile") return;
  if (!frontendModule || !backendModule) {
    self.postMessage({ type: "result", exitCode: 1, stdout: "", stderr: String(compilerError || "compiler is not ready"), files: [], elapsedMilliseconds: 0, linearMemoryBytes: 0 });
    return;
  }
  const result = await runPipeline(request);
  self.postMessage(result, result.files.map((file) => file.data));
});

async function loadModule(url, name) {
  const response = await fetch(url);
  if (!response.ok) throw new Error(`could not load ${name}: HTTP ${response.status}`);
  const fallback = response.clone();
  try {
    return await WebAssembly.compileStreaming(response);
  } catch {
    return await WebAssembly.compile(await fallback.arrayBuffer());
  }
}

async function runPipeline(request) {
  const files = new Map(request.files.map((file) => [clean(file.name), new Uint8Array(file.data)]));
  const inputNames = new Set(files.keys());
  const context = { memory: null, maxLinearMemoryBytes: 0, files, fds: new Map(), nextFd: 4, stdout: [], stderr: [] };
  const plan = pipelineArguments(request.args, files);
  const started = performance.now();
  const frontendStarted = performance.now();
  let exitCode = await runModule(frontendModule, context, ["renvo", ...plan.frontend]);
  const frontendMilliseconds = performance.now() - frontendStarted;
  let backendMilliseconds = 0;
  if (exitCode === 0 && plan.backend) {
    const backendStarted = performance.now();
    exitCode = await runModule(backendModule, context, ["renvo-wasi-backend", ...plan.backend]);
    backendMilliseconds = performance.now() - backendStarted;
  }
  if (plan.backend) files.delete(plan.temporary);
  const outputs = [];
  for (const [name, data] of files) {
    if (inputNames.has(name)) continue;
    const copy = data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength);
    outputs.push({ name, data: copy });
  }
  outputs.sort((left, right) => left.name.localeCompare(right.name));
  return {
    type: "result",
    exitCode,
    stdout: decodeParts(context.stdout),
    stderr: decodeParts(context.stderr),
    files: outputs,
    elapsedMilliseconds: performance.now() - started,
    frontendMilliseconds,
    backendMilliseconds,
    linearMemoryBytes: context.maxLinearMemoryBytes,
  };
}

async function runModule(module, context, args) {
  context.fds = new Map();
  context.nextFd = 4;
  context.memory = null;
  let exitCode = 0;
  try {
    const instance = await WebAssembly.instantiate(module, wasiImports(context, args));
    context.memory = instance.exports.memory;
    try {
      instance.exports._start();
    } catch (error) {
      if (!error || error.renvoExitCode === undefined) throw error;
      exitCode = error.renvoExitCode;
    }
    context.maxLinearMemoryBytes = Math.max(context.maxLinearMemoryBytes, context.memory?.buffer.byteLength || 0);
  } catch (error) {
    exitCode = 1;
    context.stderr.push(encoder.encode(String(error) + "\n"));
  }
  return exitCode;
}

function pipelineArguments(args, files) {
  const frontend = [];
  let output = "";
  let outputAt = -1;
  let arenaSize = "";
  let strip = false;
  let emitUnit = false;
  for (let i = 0; i < args.length; i++) {
    if (args[i] === "-o" && i + 1 < args.length) {
      output = args[++i];
      frontend.push("-o", output);
      outputAt = frontend.length - 1;
    } else if (args[i] === "-arena-size" && i + 1 < args.length) {
      arenaSize = args[++i];
    } else {
      if (args[i] === "-s") strip = true;
      if (args[i] === "-emit-unit") emitUnit = true;
      frontend.push(args[i]);
    }
  }
  if (emitUnit || outputAt < 0) return { frontend, backend: null, temporary: "" };
  let temporary = ".renvo/frontend.unit";
  for (let suffix = 1; files.has(temporary); suffix++) temporary = `.renvo/frontend-${suffix}.unit`;
  frontend[outputAt] = temporary;
  const backend = [];
  if (strip) backend.push("-s");
  if (arenaSize) backend.push("-arena-size", arenaSize);
  backend.push("-o", output, temporary);
  return { frontend, backend, temporary };
}

function wasiImports(context, args) {
  const env = ["PWD=."];
  return { wasi_snapshot_preview1: {
    fd_write: (fd, iovecsAt, count, writtenAt) => fdWrite(context, fd, iovecsAt, count, writtenAt),
    fd_read: (fd, iovecsAt, count, readAt) => fdRead(context, fd, iovecsAt, count, readAt),
    fd_pread: (fd, iovecsAt, count, offset, readAt) => fdRead(context, fd, iovecsAt, count, readAt, Number(offset)),
    fd_pwrite: (fd, iovecsAt, count, offset, writtenAt) => fdWrite(context, fd, iovecsAt, count, writtenAt, Number(offset)),
    path_open: (...values) => pathOpen(context, ...values),
    fd_close: (fd) => { context.fds.delete(fd); return 0; },
    fd_fdstat_get: (fd, at) => fdStat(context, fd, at),
    fd_readdir: (fd, at, length, cookie, usedAt) => fdReaddir(context, fd, at, length, cookie, usedAt),
    args_sizes_get: (countAt, sizeAt) => writeStringSizes(context, args, countAt, sizeAt),
    args_get: (pointersAt, dataAt) => writeStrings(context, args, pointersAt, dataAt),
    environ_sizes_get: (countAt, sizeAt) => writeStringSizes(context, env, countAt, sizeAt),
    environ_get: (pointersAt, dataAt) => writeStrings(context, env, pointersAt, dataAt),
    proc_exit: (code) => { throw { renvoExitCode: code }; },
  } };
}

function view(context) { return new DataView(context.memory.buffer); }

function iovecs(context, at, count) {
  const data = view(context);
  const result = [];
  for (let i = 0; i < count; i++) {
    const pointer = data.getUint32(at + i * 8, true);
    const length = data.getUint32(at + i * 8 + 4, true);
    result.push(new Uint8Array(context.memory.buffer, pointer, length));
  }
  return result;
}

function fdWrite(context, fd, at, count, writtenAt, explicitOffset) {
  const parts = iovecs(context, at, count);
  let total = 0;
  for (const part of parts) total += part.length;
  view(context).setUint32(writtenAt, total, true);
  if (fd === 1 || fd === 2) {
    const target = fd === 1 ? context.stdout : context.stderr;
    for (const part of parts) target.push(part.slice());
    return 0;
  }
  const entry = context.fds.get(fd);
  if (!entry || entry.directory) return 8;
  let offset = explicitOffset === undefined ? entry.offset : explicitOffset;
  const old = context.files.get(entry.path) || new Uint8Array(0);
  const next = new Uint8Array(Math.max(old.length, offset + total));
  next.set(old);
  for (const part of parts) { next.set(part, offset); offset += part.length; }
  context.files.set(entry.path, next);
  if (explicitOffset === undefined) entry.offset = offset;
  return 0;
}

function fdRead(context, fd, at, count, readAt, explicitOffset) {
  const entry = context.fds.get(fd);
  if (!entry || entry.directory) return 8;
  const source = context.files.get(entry.path) || new Uint8Array(0);
  let offset = explicitOffset === undefined ? entry.offset : explicitOffset;
  let total = 0;
  for (const part of iovecs(context, at, count)) {
    const size = Math.max(0, Math.min(part.length, source.length - offset));
    part.set(source.subarray(offset, offset + size));
    offset += size;
    total += size;
    if (size < part.length) break;
  }
  if (explicitOffset === undefined) entry.offset = offset;
  view(context).setUint32(readAt, total, true);
  return 0;
}

function pathOpen(context, _dirfd, _dirflags, pathAt, pathLength, oflags, _rightsBase, _rightsInherit, _fdflags, resultAt) {
  const name = clean(decoder.decode(new Uint8Array(context.memory.buffer, pathAt, pathLength)));
  const directory = isDirectory(context, name);
  const create = (oflags & 1) !== 0;
  if (!directory && !context.files.has(name) && !create) return 44;
  if (create && !context.files.has(name)) context.files.set(name, new Uint8Array(0));
  if (!directory && (oflags & 8) !== 0) context.files.set(name, new Uint8Array(0));
  const fd = context.nextFd++;
  context.fds.set(fd, { path: name, directory, offset: 0 });
  view(context).setUint32(resultAt, fd, true);
  return 0;
}

function fdStat(context, fd, at) {
  const entry = context.fds.get(fd);
  if (!entry && fd !== 3 && fd > 2) return 8;
  new Uint8Array(context.memory.buffer, at, 24).fill(0);
  new Uint8Array(context.memory.buffer)[at] = fd === 3 || entry?.directory ? 3 : 4;
  return 0;
}

function fdReaddir(context, fd, at, length, cookie, usedAt) {
  const entry = fd === 3 ? { path: ".", directory: true } : context.fds.get(fd);
  if (!entry?.directory) return 54;
  const names = directoryEntries(context, entry.path);
  const destination = new Uint8Array(context.memory.buffer, at, length);
  let used = 0;
  for (let i = Number(cookie); i < names.length; i++) {
    const name = encoder.encode(names[i].name);
    const size = 24 + name.length;
    if (used + size > length) break;
    const record = new DataView(destination.buffer, destination.byteOffset + used, size);
    record.setBigUint64(0, BigInt(i + 1), true);
    record.setBigUint64(8, BigInt(i + 1), true);
    record.setUint32(16, name.length, true);
    record.setUint8(20, names[i].directory ? 3 : 4);
    destination.set(name, used + 24);
    used += size;
  }
  view(context).setUint32(usedAt, used, true);
  return 0;
}

function writeStringSizes(context, values, countAt, sizeAt) {
  let size = 0;
  for (const value of values) size += encoder.encode(value).length + 1;
  view(context).setUint32(countAt, values.length, true);
  view(context).setUint32(sizeAt, size, true);
  return 0;
}

function writeStrings(context, values, pointersAt, dataAt) {
  const data = view(context);
  const bytes = new Uint8Array(context.memory.buffer);
  let at = dataAt;
  for (let i = 0; i < values.length; i++) {
    data.setUint32(pointersAt + i * 4, at, true);
    const encoded = encoder.encode(values[i]);
    bytes.set(encoded, at);
    at += encoded.length;
    bytes[at++] = 0;
  }
  return 0;
}

function clean(name) {
  const parts = [];
  for (const part of name.replaceAll("\\", "/").split("/")) {
    if (!part || part === ".") continue;
    if (part === "..") parts.pop();
    else parts.push(part);
  }
  return parts.join("/") || ".";
}

function isDirectory(context, name) {
  if (name === ".") return true;
  const prefix = name + "/";
  for (const path of context.files.keys()) if (path.startsWith(prefix)) return true;
  return false;
}

function directoryEntries(context, directory) {
  const prefix = directory === "." ? "" : directory + "/";
  const names = new Map();
  for (const path of context.files.keys()) {
    if (!path.startsWith(prefix)) continue;
    const rest = path.slice(prefix.length);
    const slash = rest.indexOf("/");
    const name = slash < 0 ? rest : rest.slice(0, slash);
    if (name) names.set(name, slash >= 0);
  }
  return Array.from(names, ([name, directory]) => ({ name, directory })).sort((left, right) => left.name.localeCompare(right.name));
}

function decodeParts(parts) {
  let result = "";
  for (const part of parts) result += decoder.decode(part, { stream: true });
  return result + decoder.decode();
}
