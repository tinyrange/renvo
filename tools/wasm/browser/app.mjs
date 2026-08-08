const command = document.querySelector("#command");
const source = document.querySelector("#source");
const compile = document.querySelector("#compile");
const diagnostics = document.querySelector("#diagnostics");
const outputs = document.querySelector("#outputs");
const parameters = new URLSearchParams(location.search);
const compiler = new URL(parameters.get("compiler") || "./renvo.wasm", location.href).href;
const backend = new URL(parameters.get("backend") || "./renvo-backend.wasm", location.href).href;
const worker = new Worker(new URL("./worker.mjs", import.meta.url), { type: "module" });

worker.addEventListener("message", (event) => {
  const result = event.data;
  if (result.type === "ready") {
    compile.disabled = false;
    compile.textContent = "Compile";
    diagnostics.textContent = "Compiler ready.";
    if (parameters.has("run")) compile.click();
    return;
  }
  if (result.type !== "result") return;
  compile.disabled = false;
  compile.textContent = "Compile";
  const phases = result.backendMilliseconds > 0 ?
    ` (${result.frontendMilliseconds.toFixed(1)} ms frontend + ${result.backendMilliseconds.toFixed(1)} ms backend)` : "";
  const timing = `${result.elapsedMilliseconds.toFixed(1)} ms${phases}, ${(result.linearMemoryBytes / 1048576).toFixed(1)} MiB linear memory`;
  diagnostics.textContent = (result.stderr || result.stdout || (result.exitCode === 0 ? "Build succeeded." : "Build failed.")) + `\n${timing}`;
  outputs.replaceChildren();
  for (const file of result.files) {
    const link = document.createElement("a");
    const blob = new Blob([file.data], { type: "application/octet-stream" });
    link.href = URL.createObjectURL(blob);
    link.download = file.name.split("/").pop();
    link.textContent = `Download ${file.name} (${formatBytes(file.data.byteLength)})`;
    outputs.append(link);
  }
});

worker.addEventListener("error", (event) => {
  compile.disabled = true;
  compile.textContent = "Compiler unavailable";
  diagnostics.textContent = event.message;
});

compile.addEventListener("click", () => {
  let args;
  try {
    args = splitArguments(command.value);
  } catch (error) {
    diagnostics.textContent = error.message;
    return;
  }
  const data = new TextEncoder().encode(source.value);
  const moduleFile = new TextEncoder().encode("module playground\n\ngo 1.20\n");
  compile.disabled = true;
  compile.textContent = "Compiling…";
  diagnostics.textContent = "";
  outputs.replaceChildren();
  worker.postMessage({ type: "compile", args, files: [{ name: "main.go", data }, { name: "go.mod", data: moduleFile }] }, [data.buffer, moduleFile.buffer]);
});

worker.postMessage({ type: "init", compiler, backend });

function splitArguments(text) {
  const args = [];
  let value = "";
  let quote = "";
  let escaped = false;
  let active = false;
  for (const character of text) {
    if (escaped) {
      value += character;
      escaped = false;
      active = true;
    } else if (character === "\\" && quote !== "'") {
      escaped = true;
      active = true;
    } else if (quote) {
      if (character === quote) quote = "";
      else value += character;
      active = true;
    } else if (character === "'" || character === '"') {
      quote = character;
      active = true;
    } else if (/\s/.test(character)) {
      if (active) {
        args.push(value);
        value = "";
        active = false;
      }
    } else {
      value += character;
      active = true;
    }
  }
  if (escaped || quote) throw new Error("Unterminated quote or escape in compiler arguments.");
  if (active) args.push(value);
  return args;
}

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / 1048576).toFixed(1)} MiB`;
}
