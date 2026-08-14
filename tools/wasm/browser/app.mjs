import { ESPWebSerial, requestESPPort } from "./esp-webserial.mjs";
import { preferredESPTransport, requestESPUSBPort, supportsESPWebUSBPlatform } from "./esp-webusb.mjs";
import { cleanLanguagePath } from "./language-path.mjs";

const MONACO_VERSION = "0.56.0";
const encoder = new TextEncoder();
const decoder = new TextDecoder();
const parameters = new URLSearchParams(location.search);
const browserAssetRoot = new URL(".", import.meta.url);
const bundleRoot = browserAssetRoot.pathname.endsWith("/browser/")
  ? new URL("../", browserAssetRoot)
  : browserAssetRoot;
const MONACO_ROOT = new URL(parameters.get("monaco") || `https://cdn.jsdelivr.net/npm/monaco-editor@${MONACO_VERSION}/min/`, location.href).href.replace(/\/$/, "");
const compilerUrl = new URL(parameters.get("compiler") || "renvo.wasm", bundleRoot).href;
const fallbackBackendUrl = new URL(parameters.get("backend") || "backends/wasi-wasm32.wasm", bundleRoot).href;
const catalogUrl = new URL(parameters.get("catalog") || "targets.json", bundleRoot).href;
const worker = new Worker(new URL("./worker.mjs", import.meta.url), { type: "module" });

const elements = {
  command: document.querySelector("#command"),
  compile: document.querySelector("#compile"),
  run: document.querySelector("#run"),
  flashTransport: document.querySelector("#flash-transport"),
  targetPicker: document.querySelector("#target-picker"),
  targetButton: document.querySelector("#target-button"),
  targetLabel: document.querySelector("#target-label"),
  targetMenu: document.querySelector("#target-menu"),
  runArgs: document.querySelector("#run-args"),
  runStdin: document.querySelector("#run-stdin"),
  terminalOutput: document.querySelector("#terminal-output"),
  compilerStatus: document.querySelector("#compiler-status"),
  languageStatus: document.querySelector("#language-status"),
  cursorStatus: document.querySelector("#cursor-status"),
  memoryStatus: document.querySelector("#memory-status"),
  problemStatus: document.querySelector("#problem-status"),
  problemCount: document.querySelector("#problem-count"),
  artifactCount: document.querySelector("#artifact-count"),
  output: document.querySelector("#output"),
  problems: document.querySelector("#problems"),
  artifacts: document.querySelector("#artifacts"),
  workbench: document.querySelector(".workbench"),
  editorHost: document.querySelector("#editor"),
  stdlibTree: document.querySelector("#stdlib-tree"),
  ide: document.querySelector("#ide"),
  mobileStep: document.querySelector("#mobile-step"),
  mobileContext: document.querySelector("#mobile-context"),
  mobileEditorActions: document.querySelector(".mobile-editor-actions"),
  mobileTargetButton: document.querySelector("#mobile-target-button"),
  mobileBuild: document.querySelector("#mobile-build"),
  mobileRun: document.querySelector("#mobile-run"),
  mobileTargetList: document.querySelector("#mobile-target-list"),
  mobileFlashView: document.querySelector("#mobile-flash-view"),
  mobileFlashState: document.querySelector("#mobile-flash-state"),
  mobileFlashProgress: document.querySelector("#mobile-flash-progress"),
  mobileFlashOutput: document.querySelector("#mobile-flash-output"),
  copyToPlayground: document.querySelector("#copy-to-playground"),
};
const phoneWorkspace = matchMedia("(max-width: 680px)");

const initialFiles = {
  "main.go": `package main

func main() {
	print("Hello from Renvo!\\n")
}
`,
  "go.mod": "module renvo.dev\n\ngo 1.20\n",
};
const savedFiles = loadSavedFiles();
const fileValues = Object.fromEntries(Object.entries(initialFiles).map(([name, source]) => [name, savedFiles[name] ?? source]));
const editableBaselines = new Map(Object.entries(fileValues));
const models = new Map();
const stdlibFiles = new Map();
const loadedStandardPackages = new Set();
const loadingStandardPackages = new Map();
const languageRequests = new Map();
const backendReady = new Set();
let monaco;
let editor;
let activeFile = "main.go";
let compilerReady = false;
let building = false;
let running = false;
let buildRevision = 1;
let pendingBuild;
let runAfterBuild = false;
let artifactUrls = [];
let lastRunnableArtifact;
let espPort;
let espSession;
let espPortTransport;
let selectedTarget;
let targetCatalog;
let standardCatalog;
let standardCatalogPromise;
let analysisTimer;
let languageGeneration = 0;
let latestAnalysisRequestID = 0;
let requestID = 0;
let focusedTargetIndex = -1;
let activeBuildRoot = ".";
let autoBuildPending = parameters.has("run");

setupShell();
boot().catch(showFatalError);

async function boot() {
  const [catalog] = await Promise.all([loadTargetCatalog(), loadMonaco()]);
  targetCatalog = catalog;
  configureTargets(catalog.targets);
  installLanguageProviders();
  const languageService = catalog.languageService ? new URL(catalog.languageService, catalogUrl).href : "";
  await initializeCompiler(languageService);
  scheduleAnalysis(20);
}

async function loadTargetCatalog() {
  try {
    const response = await fetch(catalogUrl);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    const catalog = await response.json();
    if (!Array.isArray(catalog.targets) || !catalog.targets.length) throw new Error("target catalog is empty");
    if (catalog.stdlib) {
      const stdlibUrl = new URL(catalog.stdlib, catalogUrl).href;
      standardCatalogPromise = fetch(stdlibUrl).then((response) => {
        if (!response.ok) throw new Error(`could not load standard library catalog: HTTP ${response.status}`);
        return response.json().then((value) => {
          standardCatalog = { ...value, url: stdlibUrl };
          return standardCatalog;
        });
      });
      standardCatalogPromise.then(renderLibraryCatalog, (error) => {
        elements.stdlibTree.textContent = error.message;
      });
    }
    return catalog;
  } catch (error) {
    elements.output.textContent = `Target catalog unavailable (${error.message}); using the WASI backend from the URL.\n`;
    return { languageService: "", targets: [{
      name: "wasi/wasm32", backendTarget: "wasi/wasm32", backend: fallbackBackendUrl,
      output: "app.wasm", runnable: true, tags: ["wasi", "wasip1", "wasm", "wasm32"],
    }] };
  }
}

function configureTargets(targets) {
  const entries = [];
  const mobileEntries = [];
  let group = "";
  for (let index = 0; index < targets.length; index++) {
    const target = targets[index];
    const nextGroup = targetGroup(target.name);
    if (nextGroup !== group) {
      group = nextGroup;
      const heading = document.createElement("div");
      heading.className = "target-group";
      heading.textContent = group;
      entries.push(heading);
      const mobileHeading = document.createElement("div");
      mobileHeading.className = "mobile-target-group";
      mobileHeading.textContent = group;
      mobileEntries.push(mobileHeading);
    }
    const option = document.createElement("button");
    option.type = "button";
    option.className = "target-option";
    option.id = `target-option-${index}`;
    option.dataset.target = target.name;
    option.dataset.index = String(index);
    option.setAttribute("role", "option");
    option.setAttribute("aria-selected", "false");
    option.textContent = target.name;
    entries.push(option);
    const mobileOption = document.createElement("button");
    mobileOption.type = "button";
    mobileOption.className = "mobile-target-option";
    mobileOption.dataset.target = target.name;
    mobileOption.setAttribute("role", "option");
    mobileOption.setAttribute("aria-selected", "false");
    mobileOption.textContent = target.name;
    mobileOption.addEventListener("click", () => {
      selectTarget(target.name, true);
      showMobileView("editor");
    });
    mobileEntries.push(mobileOption);
  }
  elements.targetMenu.replaceChildren(...entries);
  elements.mobileTargetList.replaceChildren(...mobileEntries);
  const requested = parameters.get("target");
  const initial = targets.some((target) => target.name === requested) ? requested :
    targets.some((target) => target.name === "wasi/wasm32") ? "wasi/wasm32" : targets[0].name;
  selectTarget(initial, false);
}

function targetGroup(name) {
  if (name.startsWith("linux/")) return "Linux";
  if (name.startsWith("windows/")) return "Windows";
  if (name.startsWith("darwin/")) return "macOS";
  if (name.startsWith("wasi/") || name.startsWith("browser/")) return "WebAssembly";
  if (name.startsWith("vm/")) return "Virtual machine";
  if (name.startsWith("esp32")) return "Microcontrollers";
  return "Other";
}

async function loadMonaco() {
  self.MonacoEnvironment = {
    getWorkerUrl() {
      const source = `self.MonacoEnvironment={baseUrl:${JSON.stringify(`${MONACO_ROOT}/`)}};importScripts(${JSON.stringify(`${MONACO_ROOT}/vs/base/worker/workerMain.js`)});`;
      return `data:text/javascript;charset=utf-8,${encodeURIComponent(source)}`;
    },
  };
  await loadScript(`${MONACO_ROOT}/vs/loader.js`);
  window.require.config({ paths: { vs: `${MONACO_ROOT}/vs` } });
  await new Promise((resolve, reject) => window.require(["vs/editor/editor.main"], resolve, reject));
  monaco = window.monaco;
  defineTheme();
  for (const [name, value] of Object.entries(fileValues)) {
    const language = name.endsWith(".go") ? "go" : "plaintext";
    const model = monaco.editor.createModel(value, language, monaco.Uri.parse(`file:///${name}`));
    model.onDidChangeContent(() => handleModelChange(name, model));
    models.set(name, model);
    document.querySelector(`.file[data-file="${name}"]`)?.classList.toggle("modified", value !== editableBaselines.get(name));
  }
  editor = monaco.editor.create(elements.editorHost, {
    model: models.get(activeFile), theme: "renvo-dark", automaticLayout: true,
    fontFamily: "ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace",
    fontSize: 13, lineHeight: 20, tabSize: 4, insertSpaces: false,
    minimap: { enabled: false }, padding: { top: 7, bottom: 6 },
    scrollBeyondLastLine: false, smoothScrolling: true, renderLineHighlight: "all",
    renderWhitespace: "selection", overviewRulerBorder: false, hideCursorInOverviewRuler: true,
    stickyScroll: { enabled: false }, guides: { indentation: true, bracketPairs: false },
    bracketPairColorization: { enabled: true }, lightbulb: { enabled: "off" },
    quickSuggestions: { other: true, comments: false, strings: false },
    suggestOnTriggerCharacters: true, wordBasedSuggestions: "off",
  });
  editor.onDidChangeCursorPosition(({ position }) => {
    elements.cursorStatus.textContent = `Ln ${position.lineNumber}, Col ${position.column}`;
  });
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, compile);
  editor.addCommand(monaco.KeyCode.F5, runArtifact);
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, saveFiles);
  elements.editorHost.querySelector(".editor-loading")?.remove();
  configureEditorForViewport();
  updateReadyState();
  if (!isPhoneWorkspace()) editor.focus();
}

function initializeCompiler(languageService) {
  return new Promise((resolve, reject) => {
    const onReady = (event) => {
      if (event.data.type !== "ready") return;
      worker.removeEventListener("message", onReady);
      compilerReady = true;
      setCompilerStatus("ready", "Compiler ready");
      elements.languageStatus.textContent = languageService ? "Language service ready" : "Language service unavailable";
      updateReadyState();
      resolve();
    };
    worker.addEventListener("message", onReady);
    worker.addEventListener("error", reject, { once: true });
    worker.postMessage({ type: "init", compiler: compilerUrl, languageService });
  });
}

worker.addEventListener("message", (event) => {
  if (event.data.type === "result") renderResult(event.data);
  else if (event.data.type === "run-result") renderRunResult(event.data);
  else if (event.data.type === "language-result") receiveLanguageResult(event.data);
});
worker.addEventListener("error", (event) => showFatalError(new Error(event.message)));

function setupShell() {
  elements.compile.addEventListener("click", compile);
  elements.run.addEventListener("click", runArtifact);
  configureFlashTransports();
  elements.flashTransport.addEventListener("change", changeFlashTransport);
  elements.mobileBuild.addEventListener("click", compile);
  elements.mobileRun.addEventListener("click", runArtifact);
  elements.mobileTargetButton.addEventListener("click", () => {
    showMobileView(elements.ide.dataset.mobileView === "target" ? "editor" : "target");
  });
  elements.copyToPlayground.addEventListener("click", copyActiveFileToPlayground);
  document.querySelectorAll("[data-mobile-transport]").forEach((button) => button.addEventListener("click", () => {
    if (button.disabled) return;
    elements.flashTransport.value = button.dataset.mobileTransport;
    elements.flashTransport.dispatchEvent(new Event("change"));
  }));
  document.querySelector("#mobile-flash-close").addEventListener("click", () => closeMobileFlashView());
  document.querySelectorAll(".mobile-nav button").forEach((button) => button.addEventListener("click", () => {
    showMobileView(button.dataset.mobileView);
  }));
  new MutationObserver(syncMobileFlashOutput).observe(elements.terminalOutput, {
    childList: true, subtree: true, characterData: true,
  });
  phoneWorkspace.addEventListener?.("change", configureMobileWorkspace);
  globalThis.visualViewport?.addEventListener("resize", layoutMobileEditor);
  window.addEventListener("resize", layoutMobileEditor);
  configureMobileWorkspace();
  elements.targetButton.addEventListener("click", () => toggleTargetMenu());
  elements.targetButton.addEventListener("keydown", handleTargetKeydown);
  elements.targetMenu.addEventListener("click", (event) => {
    const option = event.target.closest(".target-option");
    if (option) chooseTargetOption(option);
  });
  elements.targetMenu.addEventListener("pointermove", (event) => {
    const option = event.target.closest(".target-option");
    if (option) setFocusedTarget(Number(option.dataset.index));
  });
  document.addEventListener("pointerdown", (event) => {
    if (!elements.targetPicker.contains(event.target)) closeTargetMenu();
  });
  elements.command.addEventListener("input", markBuildStale);
  elements.command.addEventListener("keydown", (event) => { if (event.key === "Enter") compile(); });
  document.querySelectorAll(".file").forEach((button) => installWorkspaceFileButton(button));
  document.querySelectorAll(".activity[data-view]").forEach((button) => button.addEventListener("click", () => activateView(button.dataset.view)));
  document.querySelectorAll(".panel-tab").forEach((button) => button.addEventListener("click", () => showPanel(button.dataset.panel)));
  document.querySelector("#toggle-panel").addEventListener("click", togglePanel);
  document.querySelector("#close-panel").addEventListener("click", () => elements.workbench.classList.add("panel-hidden"));
  document.querySelector("#clear-output").addEventListener("click", () => { elements.output.textContent = ""; });
  elements.problemStatus.addEventListener("click", () => showPanel("problems"));
  window.addEventListener("keydown", (event) => {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "j") {
      event.preventDefault();
      togglePanel();
    }
  });
  window.addEventListener("pagehide", () => {
    if (espSession) espSession.close().catch(() => {});
    else if (espPort) espPort.close().catch(() => {});
  });
}

function selectTarget(name, updateCommand) {
  const previousTarget = selectedTarget;
  const changed = selectedTarget?.name !== name;
  selectedTarget = targetCatalog?.targets.find((target) => target.name === name);
  if (!selectedTarget) return;
  if (changed && previousTarget?.device === "esp32" && espPort) {
    if (espSession) espSession.close().catch(() => {});
    else espPort.close().catch(() => {});
    espSession = undefined;
    espPort = undefined;
    espPortTransport = undefined;
  }
  elements.targetLabel.textContent = selectedTarget.name;
  elements.targetButton.title = `Build target: ${selectedTarget.name}`;
  const board = selectedTarget.device === "esp32";
  elements.flashTransport.hidden = !board;
  elements.run.title = board ? "Build, flash, and run on the connected ESP board (F5)" : "Run console app (F5)";
  elements.runArgs.closest("label").hidden = board;
  elements.runStdin.closest("label").hidden = board;
  for (const option of elements.targetMenu.querySelectorAll(".target-option")) {
    option.setAttribute("aria-selected", String(option.dataset.target === selectedTarget.name));
  }
  for (const option of elements.mobileTargetList.querySelectorAll(".mobile-target-option")) {
    option.setAttribute("aria-selected", String(option.dataset.target === selectedTarget.name));
  }
  updateMobileHeader();
  lastRunnableArtifact = undefined;
  if (updateCommand) elements.command.value = replaceOutput(elements.command.value, selectedTarget.output);
  if (changed) markBuildStale();
  updateReadyState();
  scheduleAnalysis(20);
}

function toggleTargetMenu(force) {
  const open = force === undefined ? elements.targetMenu.hidden : force;
  if (!open) {
    closeTargetMenu();
    return;
  }
  elements.targetMenu.hidden = false;
  elements.targetPicker.classList.add("open");
  elements.targetButton.setAttribute("aria-expanded", "true");
  const index = targetCatalog.targets.findIndex((target) => target.name === selectedTarget?.name);
  setFocusedTarget(index < 0 ? 0 : index, true);
}

function closeTargetMenu() {
  elements.targetMenu.hidden = true;
  elements.targetPicker.classList.remove("open");
  elements.targetButton.setAttribute("aria-expanded", "false");
  elements.targetButton.removeAttribute("aria-activedescendant");
}

function setFocusedTarget(index, reveal = false) {
  const options = elements.targetMenu.querySelectorAll(".target-option");
  if (!options.length) return;
  focusedTargetIndex = (index + options.length) % options.length;
  for (let i = 0; i < options.length; i++) options[i].classList.toggle("focused", i === focusedTargetIndex);
  const option = options[focusedTargetIndex];
  elements.targetButton.setAttribute("aria-activedescendant", option.id);
  if (reveal) option.scrollIntoView({ block: "nearest" });
}

function chooseTargetOption(option) {
  selectTarget(option.dataset.target, true);
  closeTargetMenu();
  elements.targetButton.focus();
}

function handleTargetKeydown(event) {
  if (event.key === "Escape") {
    closeTargetMenu();
    return;
  }
  if (event.key === "Tab") {
    closeTargetMenu();
    return;
  }
  if (event.key === "Enter" || event.key === " ") {
    event.preventDefault();
    if (elements.targetMenu.hidden) toggleTargetMenu(true);
    else chooseTargetOption(elements.targetMenu.querySelectorAll(".target-option")[focusedTargetIndex]);
    return;
  }
  if (event.key !== "ArrowDown" && event.key !== "ArrowUp" && event.key !== "Home" && event.key !== "End") return;
  event.preventDefault();
  if (elements.targetMenu.hidden) toggleTargetMenu(true);
  const count = targetCatalog.targets.length;
  if (event.key === "Home") setFocusedTarget(0, true);
  else if (event.key === "End") setFocusedTarget(count - 1, true);
  else setFocusedTarget(focusedTargetIndex + (event.key === "ArrowDown" ? 1 : -1), true);
}

async function compile() {
  if (!compilerReady || building || !monaco || !selectedTarget) return;
  const buildTarget = selectedTarget;
  const revision = buildRevision;
  let args;
  try {
    args = splitArguments(elements.command.value);
  } catch (error) {
    runAfterBuild = false;
    updateReadyState();
    renderProblems([{ message: error.message, file: "", line: 0, column: 0 }]);
    showPanel("problems");
    return;
  }
  args = controlledArguments(args, buildTarget);
  saveFiles();
  clearMarkers();
  clearArtifactUrls();
  lastRunnableArtifact = undefined;
  building = true;
  updateReadyState();
  closeTargetMenu();
  const backend = new URL(buildTarget.backend, catalogUrl).href;
  const loading = !backendReady.has(backend);
  setCompilerStatus("busy", loading ? `Loading ${buildTarget.name} backend…` : "Building…");
  elements.output.textContent = `$ renvo ${args.join(" ")}\n`;
  showPanel("output");
  try {
    await ensureWorkspaceDependencies();
    const payload = workspacePayload();
    const id = ++requestID;
    pendingBuild = { id, revision, target: buildTarget, backend };
    worker.postMessage({
      type: "compile", id, args, files: payload.files,
      backend, backendTarget: buildTarget.backendTarget,
    }, payload.transfers);
  } catch (error) {
    building = false;
    pendingBuild = undefined;
    runAfterBuild = false;
    updateReadyState();
    setCompilerStatus("error", "Build failed");
    elements.output.textContent += `${error.message}\n`;
  }
}

function controlledArguments(args, target) {
  const result = [];
  for (let i = 0; i < args.length; i++) {
    if ((args[i] === "-t" || args[i] === "-target-definition" || args[i] === "-target-version") && i + 1 < args.length) {
      i++;
      continue;
    }
    result.push(args[i]);
  }
  result.unshift("-t", target.name);
  for (const tag of target.tags || []) result.unshift("-tags", tag);
  if (target.definition) result.unshift("-target-version", String(target.descriptorVersion), "-target-definition", target.definition);
  return result;
}

function renderResult(result) {
  const build = pendingBuild?.id === result.id ? pendingBuild : {
    revision: buildRevision, target: selectedTarget, backend: new URL(selectedTarget.backend, catalogUrl).href,
  };
  pendingBuild = undefined;
  building = false;
  const phases = result.backendMilliseconds > 0
    ? `${result.frontendMilliseconds.toFixed(1)} ms frontend · ${result.backendMilliseconds.toFixed(1)} ms backend`
    : `${result.frontendMilliseconds.toFixed(1)} ms frontend`;
  const summary = `${result.exitCode === 0 ? "Build succeeded" : "Build failed"} · ${result.elapsedMilliseconds.toFixed(1)} ms · ${phases}`;
  const text = [result.stdout, result.stderr].filter(Boolean).join("");
  elements.output.textContent += `${text}${text && !text.endsWith("\n") ? "\n" : ""}${summary}\n`;
  elements.memoryStatus.textContent = `${(result.linearMemoryBytes / 1048576).toFixed(1)} MiB`;
  setCompilerStatus(result.exitCode === 0 ? "ready" : "error", result.exitCode === 0 ? "Build succeeded" : "Build failed");
  const diagnosticText = result.exitCode === 0 ? "" : [result.stderr, result.stdout].filter(Boolean).join("\n");
  const problems = parseDiagnostics(diagnosticText);
  renderProblems(problems);
  renderArtifacts(result.files);
  if (result.exitCode === 0) {
    backendReady.add(build.backend);
    const artifact = result.files.find((file) => file.name === build.target.output) || result.files[0];
    if ((build.target.runnable || build.target.device === "esp32") && artifact) {
      lastRunnableArtifact = {
        name: artifact.name, data: artifact.data.slice(0),
        revision: build.revision, target: build.target.name,
        buildMilliseconds: result.elapsedMilliseconds,
      };
    }
  }
  const shouldRun = runAfterBuild;
  runAfterBuild = false;
  updateReadyState();
  if (result.exitCode !== 0) showPanel(problems.length ? "problems" : "output");
  else if (result.files.length) showPanel("artifacts");
  if (result.exitCode !== 0 && shouldRun && isPhoneWorkspace()) {
    elements.terminalOutput.textContent = `${text}${text && !text.endsWith("\n") ? "\n" : ""}${summary}\n`;
    setMobileFlashProgress("Build failed", 0);
  }
  if (result.exitCode === 0 && shouldRun) queueMicrotask(resumeArtifactAfterBuild);
}

function runArtifact() {
  return runArtifactWithMode(false);
}

function resumeArtifactAfterBuild() {
  return runArtifactWithMode(true);
}

async function runArtifactWithMode(resumeAfterBuild) {
  if ((!selectedTarget?.runnable && selectedTarget?.device !== "esp32") || running) return;
  const board = selectedTarget.device === "esp32";
  if (board && !resumeAfterBuild) openMobileFlashView("Select a device");
  if (board && !resumeAfterBuild && espPortTransport === "webusb" && espSession) {
    const previousSession = espSession;
    espSession = undefined;
    await previousSession.close();
  }
  const activeESPPort = espPort && (espPort.readable || espPort.writable);
  const reusableESPPort = espPortTransport === "webusb" && espPort?.canReopen?.();
  if (board && !resumeAfterBuild && !activeESPPort && !reusableESPPort) {
    try {
      const previousSession = espSession;
      const previousPort = espPort;
      // Start the permission prompt while the click/key activation is still
      // live; cleanup can safely happen after the user selects the device.
      const transport = elements.flashTransport.value;
      const nextPort = transport === "webusb"
        ? await requestESPUSBPort(selectedTarget.name)
        : await requestESPPort(selectedTarget.name);
      espSession = undefined;
      espPort = undefined;
      if (previousSession) await previousSession.close();
      else if (previousPort) {
        try { await previousPort.close(); } catch {}
      }
      espPort = nextPort;
      espPortTransport = transport;
    } catch (error) {
      elements.terminalOutput.textContent = `${error.message || error}\n`;
      setMobileFlashProgress("Device selection failed", 0);
      showPanel("terminal");
      return;
    }
  }
  if (board && !espPort) {
    elements.terminalOutput.textContent = "The selected ESP device disconnected before flashing. Click Flash & Run again.\n";
    setMobileFlashProgress("Device disconnected", 0);
    showPanel("terminal");
    return;
  }
  let args;
  try {
    args = splitArguments(elements.runArgs.value);
  } catch (error) {
    elements.terminalOutput.textContent = `${error.message}\n`;
    showPanel("terminal");
    return;
  }
  const stale = !lastRunnableArtifact || lastRunnableArtifact.revision !== buildRevision ||
    lastRunnableArtifact.target !== selectedTarget.name;
  if (building || stale) {
    runAfterBuild = true;
    if (board) setMobileFlashProgress("Building…");
    updateReadyState();
    if (!building) compile();
    return;
  }
  running = true;
  updateReadyState();
  showPanel("terminal");
  if (board) {
    openMobileFlashView("Connecting…");
    setMobileFlashProgress("Connecting…");
    const portInfo = espPort.getInfo?.() || {};
    const identity = portInfo.usbVendorId === undefined ? "" :
      ` (USB ${portInfo.usbVendorId.toString(16).padStart(4, "0")}:${(portInfo.usbProductId || 0).toString(16).padStart(4, "0")})`;
    const transportName = espPortTransport === "webusb" ? "WebUSB" : "WebSerial";
    elements.terminalOutput.textContent = `$ flash --transport ${transportName} ${selectedTarget.name}${identity}\n`;
    elements.terminalOutput.textContent += `Build: ${formatElapsed(lastRunnableArtifact.buildMilliseconds)}\n`;
    const flashStarted = performance.now();
    try {
      if (!espSession) espSession = new ESPWebSerial(espPort, {
        log: (message) => { elements.terminalOutput.textContent += `${message}\n`; },
        serial: (text) => { elements.terminalOutput.textContent += text; },
        progress: (value) => {
          elements.run.querySelector("span").textContent = `Flashing ${Math.round(value * 100)}%`;
          setMobileFlashProgress(`Flashing ${Math.round(value * 100)}%`, value);
        },
      });
      await espSession.flash(lastRunnableArtifact.data, selectedTarget.name);
      const flashMilliseconds = performance.now() - flashStarted;
      elements.terminalOutput.textContent += `Flash: ${formatElapsed(flashMilliseconds)} · Build + flash: ${formatElapsed(lastRunnableArtifact.buildMilliseconds + flashMilliseconds)}\n`;
      setMobileFlashProgress("Running — serial console attached", 1);
    } catch (error) {
      const flashMilliseconds = performance.now() - flashStarted;
      elements.terminalOutput.textContent += `Flash failed after ${formatElapsed(flashMilliseconds)}: ${error.message || error}\n`;
      setMobileFlashProgress("Flash failed", 0);
      const failedSession = espSession;
      espSession = undefined;
      try {
        if (failedSession) await failedSession.close();
        else if (espPort) await espPort.close();
      } catch {}
      espPort = undefined;
      espPortTransport = undefined;
    } finally {
      running = false;
      updateReadyState();
    }
    return;
  }
  elements.terminalOutput.textContent = `$ ${lastRunnableArtifact.name}${args.length ? ` ${args.join(" ")}` : ""}\n`;
  const data = lastRunnableArtifact.data.slice(0);
  worker.postMessage({
    type: "run", id: ++requestID, name: lastRunnableArtifact.name,
    data, args, stdin: elements.runStdin.value,
  }, [data]);
}

function renderRunResult(result) {
  running = false;
  const output = `${result.stdout || ""}${result.stderr || ""}`;
  elements.terminalOutput.textContent += `${output}${output && !output.endsWith("\n") ? "\n" : ""}[process exited ${result.exitCode} · ${result.elapsedMilliseconds.toFixed(1)} ms]\n`;
  elements.memoryStatus.textContent = `${(result.linearMemoryBytes / 1048576).toFixed(1)} MiB`;
  updateReadyState();
}

async function ensureWorkspaceDependencies() {
  if (!standardCatalogPromise) return;
  const catalog = await standardCatalogPromise;
  const imports = new Set();
  for (const model of models.values()) {
    if (model.uri.path.endsWith(".go")) for (const name of scanImports(model.getValue())) imports.add(name);
  }
  await Promise.all(Array.from(imports, (name) => loadStandardPackage(name, catalog)));
}

async function loadStandardPackage(importPath, catalog) {
  let name = importPath;
  if (name.startsWith("renvo.dev/std/")) name = name.slice("renvo.dev/std/".length);
  const platform = catalog.platforms?.[importPath];
  const item = platform || catalog.packages?.[name];
  if (!item) return;
  const key = platform ? importPath : name;
  if (loadedStandardPackages.has(key)) return;
  if (loadingStandardPackages.has(key)) return loadingStandardPackages.get(key);
  const loading = (async () => {
    const root = platform ? `module/${item.root}` : `src/${name}`;
    const base = new URL(`${root.split("/").map(encodeURIComponent).join("/")}/`, catalog.url);
    const values = await Promise.all(item.files.map(async (file) => {
      const response = await fetch(new URL(file.split("/").map(encodeURIComponent).join("/"), base));
      if (!response.ok) throw new Error(`could not load library file ${key}/${file}: HTTP ${response.status}`);
      return [file, new Uint8Array(await response.arrayBuffer())];
    }));
    const prefix = platform ? item.root : `std/${name}`;
    for (const [file, data] of values) stdlibFiles.set(`${prefix}/${file}`, data);
    loadedStandardPackages.add(key);
    await Promise.all((item.imports || []).map((dependency) => loadStandardPackage(dependency, catalog)));
  })();
  loadingStandardPackages.set(key, loading);
  try {
    await loading;
  } finally {
    loadingStandardPackages.delete(key);
  }
}

function workspacePayload() {
  const files = [];
  const transfers = [];
  for (const [name, model] of models) {
    const data = encoder.encode(model.getValue());
    files.push({ name, data });
    transfers.push(data.buffer);
  }
  for (const [name, source] of stdlibFiles) {
    if (models.has(name)) continue;
    const data = source.slice();
    files.push({ name, data });
    transfers.push(data.buffer);
  }
  return { files, transfers };
}

function installLanguageProviders() {
  monaco.languages.registerCompletionItemProvider("go", {
    triggerCharacters: [".", '"', "`", "/"],
    provideCompletionItems: async (model, position) => {
      const importContext = importContextAt(model, position);
      if (importContext) {
        const catalog = standardCatalogPromise ? await standardCatalogPromise : { packages: {} };
        const imports = [...Object.keys(catalog.packages || {}), ...Object.keys(catalog.platforms || {})];
        const suggestions = imports.filter((name) => name.startsWith(importContext.prefix)).map((name) => ({
          label: name, kind: monaco.languages.CompletionItemKind.Module,
          detail: "Renvo standard library", insertText: name,
          range: importContext.range,
        }));
        return { suggestions };
      }
      const suggestions = new Map();
      const result = await requestLanguage("complete", model, byteOffset(model, position));
      for (const record of result.filter((record) => record[0] === "C")) {
        suggestions.set(record[1], {
          label: record[1], detail: record[4] || record[2],
          kind: completionKind(Number(record[3])), insertText: record[1],
          documentation: record[5] || undefined,
        });
      }
      return { suggestions: Array.from(suggestions.values()) };
    },
  });
  monaco.languages.registerSignatureHelpProvider("go", {
    signatureHelpTriggerCharacters: ["(", ","],
    signatureHelpRetriggerCharacters: [","],
    provideSignatureHelp: async (model, position) => {
      const records = await requestLanguage("signature", model, byteOffset(model, position));
      const record = records.find((item) => item[0] === "S");
      if (!record) return { value: { signatures: [], activeSignature: 0, activeParameter: 0 }, dispose() {} };
      return { value: {
        signatures: [{ label: record[2], parameters: record.slice(3).map((label) => ({ label })) }],
        activeSignature: 0, activeParameter: Number(record[1]),
      }, dispose() {} };
    },
  });
  monaco.languages.registerDefinitionProvider("go", {
    provideDefinition: async (model, position) => {
      const records = await requestLanguage("definition", model, byteOffset(model, position));
      const record = records.find((item) => item[0] === "L");
      return record ? languageLocation(record) : undefined;
    },
  });
  monaco.languages.registerHoverProvider("go", {
    provideHover: async (model, position) => {
      const records = await requestLanguage("hover", model, byteOffset(model, position));
      const record = records.find((item) => item[0] === "H");
      if (!record) return undefined;
      const start = positionAtByteOffset(model, Number(record[3]));
      const end = positionAtByteOffset(model, Number(record[4]));
      const contents = [{ value: `\`\`\`go\n${record[1]}\n\`\`\`` }];
      if (record[2]) contents.push({ value: record[2] });
      return {
        contents,
        range: new monaco.Range(start.lineNumber, start.column, end.lineNumber, end.column),
      };
    },
  });
  monaco.languages.registerReferenceProvider("go", {
    provideReferences: async (model, position, context) => {
      const records = await requestLanguage("references", model, byteOffset(model, position));
      let locations = (await Promise.all(records.filter((item) => item[0] === "L").map(languageLocation))).filter(Boolean);
      if (!context.includeDeclaration) {
        const definitions = await requestLanguage("definition", model, byteOffset(model, position));
        const definition = definitions.find((item) => item[0] === "L");
        if (definition) locations = locations.filter((location) => !sameLanguageLocation(location, definition));
      }
      return locations;
    },
  });
}

async function languageLocation(record) {
  const name = cleanPath(record[1]);
  const model = await ensureSourceModel(name);
  if (!model) return undefined;
  const start = positionAtByteOffset(model, Number(record[2]));
  const end = positionAtByteOffset(model, Number(record[3]));
  return { uri: model.uri, range: new monaco.Range(start.lineNumber, start.column, end.lineNumber, end.column) };
}

function sameLanguageLocation(location, record) {
  const model = models.get(cleanPath(record[1]));
  if (!model || location.uri.toString() !== model.uri.toString()) return false;
  const start = positionAtByteOffset(model, Number(record[2]));
  const end = positionAtByteOffset(model, Number(record[3]));
  return location.range.startLineNumber === start.lineNumber && location.range.startColumn === start.column &&
    location.range.endLineNumber === end.lineNumber && location.range.endColumn === end.column;
}

async function requestLanguage(mode, model, offset) {
  if (!compilerReady || !targetCatalog?.languageService || !selectedTarget) return [];
  await ensureWorkspaceDependencies();
  const id = ++requestID;
  const payload = workspacePayload();
  const result = new Promise((resolve) => languageRequests.set(id, resolve));
  worker.postMessage({
    type: mode, id, files: payload.files, target: selectedTarget.name,
    tags: selectedTarget.tags || [], file: fileName(model), offset,
    packageAt: languagePackageForModel(model),
  }, payload.transfers);
  return result;
}

function receiveLanguageResult(result) {
  const records = parseProtocol(result.output || "");
  const pending = languageRequests.get(result.id);
  if (pending) {
	languageRequests.delete(result.id);
	if (result.error) elements.languageStatus.textContent = `Language service error: ${result.error.trim()}`;
	pending(records);
    return;
  }
  if (result.mode === "analyze") applyAnalysis(result.id, records, result.error);
}

function scheduleAnalysis(delay = 280) {
  clearTimeout(analysisTimer);
  const generation = ++languageGeneration;
  analysisTimer = setTimeout(() => runAnalysis(generation), delay);
}

async function runAnalysis(generation) {
  if (!compilerReady || !targetCatalog?.languageService || !selectedTarget || generation !== languageGeneration) return;
  elements.languageStatus.textContent = "Checking…";
  try {
    await ensureWorkspaceDependencies();
    if (generation !== languageGeneration) return;
    const payload = workspacePayload();
    const id = ++requestID;
    latestAnalysisRequestID = id;
    worker.postMessage({
      type: "analyze", id, files: payload.files, target: selectedTarget.name,
      tags: selectedTarget.tags || [], file: activeFile, offset: 0,
      packageAt: languagePackageForModel(models.get(activeFile)),
    }, payload.transfers);
    languageRequests.set(id, (records) => applyAnalysis(id, records, ""));
  } catch (error) {
    elements.languageStatus.textContent = "Analysis unavailable";
    renderProblems([{ message: error.message, file: "", line: 0, column: 0 }]);
  }
}

function applyAnalysis(id, records, error) {
  if (id !== latestAnalysisRequestID) return;
  const problems = records.filter((record) => record[0] === "D").map((record) => ({
    file: cleanPath(record[1]), start: Number(record[2]), end: Number(record[3]),
    line: Number(record[4]), column: Number(record[5]), code: record[6], message: record[7],
  }));
  if (error) problems.push({ file: "", line: 0, column: 0, message: error.trim() });
  clearMarkers(false);
  renderProblems(problems);
  elements.languageStatus.textContent = problems.length ? `${problems.length} problem${problems.length === 1 ? "" : "s"}` : "No problems";
}

function parseProtocol(text) {
  return text.split("\n").filter(Boolean).map((line) => {
    const fields = [];
    let field = "";
    let escaped = false;
    for (const character of line) {
      if (escaped) {
        field += character === "n" ? "\n" : character === "r" ? "\r" : character === "t" ? "\t" : character;
        escaped = false;
      } else if (character === "\\") escaped = true;
      else if (character === "\t") { fields.push(field); field = ""; }
      else field += character;
    }
    fields.push(field);
    return fields;
  });
}

function importContextAt(model, position) {
  const offset = model.getOffsetAt(position);
  const source = model.getValue();
  const before = source.slice(0, offset);
  const match = /(["`])([^"`\n]*)$/.exec(before);
  if (!match) return null;
  const quoteAt = offset - match[2].length - 1;
  const prefixSource = before.slice(0, quoteAt);
  const line = prefixSource.slice(prefixSource.lastIndexOf("\n") + 1);
  const importAt = prefixSource.lastIndexOf("import");
  const openAt = prefixSource.lastIndexOf("(");
  const closeAt = prefixSource.lastIndexOf(")");
  if (!/\bimport\s*(?:\w+\s*)?$/.test(line) && !(openAt > closeAt && importAt >= 0 && importAt < openAt)) return null;
  return {
    prefix: match[2],
    range: new monaco.Range(position.lineNumber, position.column - match[2].length, position.lineNumber, position.column),
  };
}

function completionKind(kind) {
  const kinds = monaco.languages.CompletionItemKind;
  return ({ 1: kinds.Variable, 2: kinds.Field, 3: kinds.Method, 4: kinds.Function, 5: kinds.Class, 6: kinds.Module, 7: kinds.Keyword })[kind] || kinds.Text;
}

function byteOffset(model, position) {
  return encoder.encode(model.getValue().slice(0, model.getOffsetAt(position))).length;
}

function fileName(model) { return model.uri.path.replace(/^\//, ""); }

function languagePackageForModel(model) {
  if (!model) return activeBuildRoot;
  const name = fileName(model);
  if (name.startsWith("std/")) return `./${name.slice(0, name.lastIndexOf("/"))}`;
  for (const item of Object.values(targetCatalogSourcePlatforms())) {
    if (name.startsWith(`${item.root}/`)) return `./${item.root}`;
  }
  return activeBuildRoot;
}

function targetCatalogSourcePlatforms() {
  return standardCatalog?.platforms || {};
}

function scanImports(source) {
  const imports = [];
  const direct = /\bimport\s+(?:[._A-Za-z][\w]*\s+)?["`]([^"`]+)["`]/g;
  const grouped = /\bimport\s*\(([\s\S]*?)\)/g;
  for (const match of source.matchAll(direct)) imports.push(match[1]);
  for (const group of source.matchAll(grouped)) {
    for (const match of group[1].matchAll(/(?:^|\s)(?:[._A-Za-z][\w]*\s+)?["`]([^"`]+)["`]/g)) imports.push(match[1]);
  }
  return imports;
}

function openFile(name) {
  const model = models.get(name);
  if (!model || !editor) return;
  activeFile = name;
  editor.setModel(model);
  const editable = isEditableFile(name);
  editor.updateOptions({ readOnly: !editable, readOnlyMessage: { value: "Copy this library source to the playground to edit it." } });
  elements.copyToPlayground.hidden = editable;
  document.querySelectorAll(".file").forEach((item) => item.classList.toggle("active", item.dataset.file === name));
  document.querySelectorAll(".stdlib-file").forEach((item) => item.classList.toggle("active", item.dataset.file === name));
  const tab = document.querySelector(".editor-tab");
  tab.dataset.file = name;
  const icon = tab.querySelector("span:first-child");
  icon.textContent = name.endsWith(".go") ? "Go" : "M";
  icon.className = name.endsWith(".go") ? "go-icon" : "mod-icon";
  tab.querySelector("span:nth-child(2)").textContent = name.split("/").pop();
  tab.title = name;
  updateMobileHeader();
  if (!isPhoneWorkspace()) editor.focus();
}

function isEditableFile(name) {
  return Object.hasOwn(initialFiles, name);
}

function installWorkspaceFileButton(button) {
  if (button.dataset.installed) return;
  button.dataset.installed = "true";
  button.addEventListener("click", () => {
    openFile(button.dataset.file);
    setBuildPackage(".");
    if (isPhoneWorkspace()) showMobileView("editor");
  });
}

function copyActiveFileToPlayground() {
  const source = models.get(activeFile);
  if (!source || isEditableFile(activeFile)) return;
  const main = models.get("main.go");
  if (!main) return;
  main.pushEditOperations([], [{ range: main.getFullModelRange(), text: source.getValue() }], () => null);
  openFile("main.go");
  setBuildPackage(".");
  scheduleAnalysis(20);
  if (isPhoneWorkspace()) showMobileView("editor");
}

function handleModelChange(name, model) {
  document.querySelector(`.file[data-file="${name}"]`)?.classList.toggle("modified", model.getValue() !== editableBaselines.get(name));
  saveFiles();
  markBuildStale();
  if (name.endsWith(".go") || name === "go.mod") scheduleAnalysis();
}

function markBuildStale() {
  buildRevision++;
  updateReadyState();
}

function renderProblems(problems) {
  elements.problemCount.textContent = String(problems.length);
  elements.problemStatus.querySelector("span").textContent = String(problems.length);
  if (!problems.length) {
    elements.problems.innerHTML = '<div class="empty-state">No problems detected.</div>';
    return;
  }
  elements.problems.replaceChildren(...problems.map((problem) => {
    const row = document.createElement("div");
    row.className = "problem-row";
    row.innerHTML = '<svg viewBox="0 0 16 16" aria-hidden="true"><path d="M8 1 15 14H1L8 1Zm0 5v4m0 2v1"/></svg>';
    const message = document.createElement("span");
    message.textContent = problem.code ? `${problem.code}: ${problem.message}` : problem.message;
    const location = document.createElement("span");
    location.className = "problem-location";
    location.textContent = problem.file ? `${problem.file}:${problem.line || 1}:${problem.column || 1}` : "build";
    row.append(message, location);
    if (problem.file) row.addEventListener("click", () => revealProblem(problem));
    return row;
  }));
  const byFile = new Map();
  for (const problem of problems) {
    if (!models.has(problem.file)) continue;
    if (!byFile.has(problem.file)) byFile.set(problem.file, []);
    let start = { lineNumber: problem.line || 1, column: problem.column || 1 };
    let end = { lineNumber: start.lineNumber, column: start.column + 1 };
    if (Number.isFinite(problem.start)) {
      start = positionAtByteOffset(models.get(problem.file), problem.start);
      end = positionAtByteOffset(models.get(problem.file), Math.max(problem.start + 1, problem.end));
    }
    byFile.get(problem.file).push({
      severity: monaco.MarkerSeverity.Error, code: problem.code, message: problem.message,
      startLineNumber: start.lineNumber, startColumn: start.column,
      endLineNumber: end.lineNumber, endColumn: end.column,
    });
  }
  for (const [file, markers] of byFile) monaco.editor.setModelMarkers(models.get(file), "renvo", markers);
}

function positionAtByteOffset(model, offset) {
  const bytes = encoder.encode(model.getValue());
  const units = decoder.decode(bytes.subarray(0, Math.max(0, Math.min(offset, bytes.length)))).length;
  return model.getPositionAt(units);
}

function renderArtifacts(files) {
  elements.artifactCount.textContent = String(files.length);
  if (!files.length) {
    elements.artifacts.innerHTML = '<div class="empty-state">No artifacts produced.</div>';
    return;
  }
  elements.artifacts.replaceChildren(...files.map((file) => {
    const row = document.createElement("div");
    row.className = "artifact-row";
    const name = document.createElement("span"); name.textContent = file.name;
    const size = document.createElement("span"); size.className = "artifact-size"; size.textContent = formatBytes(file.data.byteLength);
    const link = document.createElement("a");
    const url = URL.createObjectURL(new Blob([file.data], { type: file.name.endsWith(".wasm") ? "application/wasm" : "application/octet-stream" }));
    artifactUrls.push(url); link.href = url; link.download = file.name.split("/").pop(); link.textContent = "Download";
    row.append(name, size, link);
    return row;
  }));
}

function parseDiagnostics(stderr) {
  const problems = [];
  for (const rawLine of stderr.split("\n")) {
    const line = rawLine.trim();
    if (!line) continue;
    const match = /^(?:renvo:\s*)?([^:\s]+\.go):(\d+)(?::(\d+))?:\s*(.*)$/.exec(line);
    if (match) problems.push({ file: cleanPath(match[1]), line: Number(match[2]), column: Number(match[3] || 1), message: match[4] });
    else problems.push({ file: "", line: 0, column: 0, message: line.replace(/^renvo:\s*/, "") });
  }
  return problems;
}

function revealProblem(problem) {
  openFile(problem.file);
  const position = Number.isFinite(problem.start) ? positionAtByteOffset(models.get(problem.file), problem.start) : { lineNumber: problem.line || 1, column: problem.column || 1 };
  editor.setPosition(position);
  editor.revealPositionInCenter(position);
}

function showPanel(name) {
  elements.workbench.classList.remove("panel-hidden");
  document.querySelectorAll(".panel-tab").forEach((tab) => tab.classList.toggle("active", tab.dataset.panel === name));
  document.querySelectorAll(".panel-view").forEach((view) => view.classList.toggle("active", view.dataset.panelView === name));
}

function togglePanel() { elements.workbench.classList.toggle("panel-hidden"); }

function activateView(view) {
  document.querySelectorAll(".activity[data-view]").forEach((button) => button.classList.toggle("active", button.dataset.view === view));
  if (view === "output") showPanel("output");
}

function clearMarkers(render = true) {
  if (!monaco) return;
  for (const model of models.values()) monaco.editor.setModelMarkers(model, "renvo", []);
  if (render) renderProblems([]);
}

function setCompilerStatus(state, text) {
  elements.compilerStatus.classList.toggle("status-ready", state === "ready");
  elements.compilerStatus.classList.toggle("status-error", state === "error");
  elements.compilerStatus.querySelector("span:last-child").textContent = text;
}

function updateReadyState() {
  elements.compile.disabled = !compilerReady || !monaco || building;
  elements.compile.querySelector("span").textContent = building ? "Building…" : "Build";
  elements.mobileBuild.disabled = elements.compile.disabled;
  elements.mobileBuild.textContent = building ? "Building…" : "Build";
  elements.targetButton.disabled = building || running;
  const board = selectedTarget?.device === "esp32";
  const executable = selectedTarget?.runnable || board;
  const transportAvailable = !board || !elements.flashTransport.selectedOptions[0]?.disabled;
  elements.run.disabled = !compilerReady || !monaco || !executable || !transportAvailable || running || runAfterBuild;
  elements.flashTransport.disabled = building || running || runAfterBuild;
  elements.run.querySelector("span").textContent = running ? (board ? "Flashing…" : "Running…") :
    runAfterBuild ? (board ? "Flash pending…" : "Run pending…") : (board ? "Flash & Run" : "Run");
  elements.mobileRun.disabled = elements.run.disabled;
  elements.mobileRun.textContent = running ? (board ? "Flashing…" : "Running…") :
    runAfterBuild ? (board ? "Pending…" : "Pending…") : (board ? "Flash" : "Run");
  if (autoBuildPending && compilerReady && monaco && selectedTarget && !building) {
    autoBuildPending = false;
    queueMicrotask(compile);
  }
}

function isPhoneWorkspace() {
  return phoneWorkspace.matches;
}

function configureMobileWorkspace() {
  if (isPhoneWorkspace()) {
    if (!elements.ide.dataset.mobileView) elements.ide.dataset.mobileView = "files";
    showMobileView(elements.ide.dataset.mobileView);
  } else {
    elements.mobileFlashView.hidden = true;
  }
  configureEditorForViewport();
}

function configureEditorForViewport() {
  if (!editor) return;
  editor.updateOptions(isPhoneWorkspace() ? {
    wordWrap: "on", wrappingIndent: "same", fontSize: 14, lineHeight: 22,
    quickSuggestions: { other: false, comments: false, strings: false },
  } : {
    wordWrap: "off", fontSize: 13, lineHeight: 20,
    quickSuggestions: { other: true, comments: false, strings: false },
  });
  requestAnimationFrame(() => editor?.layout());
}

function layoutMobileEditor() {
  if (!isPhoneWorkspace()) return;
  requestAnimationFrame(() => editor?.layout());
}

function showMobileView(view) {
  if (!isPhoneWorkspace()) return;
  elements.mobileFlashView.hidden = true;
  elements.ide.dataset.mobileView = view;
  elements.mobileEditorActions.hidden = view !== "editor";
  document.querySelectorAll(".mobile-nav button").forEach((button) => {
    button.classList.toggle("active", button.dataset.mobileView === view);
  });
  if (view === "console") showPanel("terminal");
  updateMobileHeader();
  if (view === "editor") requestAnimationFrame(() => editor?.layout());
}

function updateMobileHeader() {
  const view = elements.ide.dataset.mobileView || "files";
  const file = activeFile.split("/").pop();
  elements.mobileTargetButton.textContent = view === "target" ? "Done" : "Target";
  elements.mobileTargetButton.title = selectedTarget ? `Select target (currently ${selectedTarget.name})` : "Select target";
  if (view === "files") {
    elements.mobileStep.textContent = "Choose a file";
    elements.mobileContext.textContent = "playground and libraries";
  } else if (view === "target") {
    elements.mobileStep.textContent = "Choose a target";
    elements.mobileContext.textContent = file;
  } else if (view === "editor") {
    elements.mobileStep.textContent = file;
    elements.mobileContext.textContent = selectedTarget?.name || "Choose a target";
  } else {
    elements.mobileStep.textContent = "Console";
    elements.mobileContext.textContent = selectedTarget?.name || "Build and run output";
  }
}

function openMobileFlashView(state = "Preparing…") {
  if (!isPhoneWorkspace()) return;
  elements.mobileFlashView.hidden = false;
  elements.mobileFlashState.textContent = state;
  document.querySelectorAll(".mobile-nav button").forEach((button) => {
    button.classList.toggle("active", button.dataset.mobileView === "console");
  });
  syncMobileFlashOutput();
}

function closeMobileFlashView() {
  elements.mobileFlashView.hidden = true;
  showMobileView("editor");
}

function setMobileFlashProgress(state, value) {
  elements.mobileFlashState.textContent = state;
  if (value === undefined) elements.mobileFlashProgress.removeAttribute("value");
  else elements.mobileFlashProgress.value = Math.max(0, Math.min(1, value));
}

function syncMobileFlashOutput() {
  elements.mobileFlashOutput.textContent = elements.terminalOutput.textContent;
  elements.mobileFlashOutput.scrollTop = elements.mobileFlashOutput.scrollHeight;
}

function configureFlashTransports() {
  const platform = navigator.userAgentData?.platform || navigator.platform || "";
  const android = supportsESPWebUSBPlatform({ platform, userAgent: navigator.userAgent });
  const choices = {
    webserial: Boolean(navigator.serial),
    webusb: Boolean(navigator.usb) && android,
  };
  for (const option of elements.flashTransport.options) {
    option.disabled = !choices[option.value];
    const name = option.value === "webusb" ? "WebUSB (Android)" : "WebSerial";
    option.textContent = `${name}${option.disabled ? " unavailable" : ""}`;
  }
  const saved = localStorage.getItem("renvo.espFlashTransport");
  elements.flashTransport.value = preferredESPTransport({
    saved, android, webSerial: choices.webserial, webUSB: choices.webusb,
  });
  syncMobileTransportPicker();
}

async function changeFlashTransport() {
  localStorage.setItem("renvo.espFlashTransport", elements.flashTransport.value);
  syncMobileTransportPicker();
  if (!espPort || espPortTransport === elements.flashTransport.value) return;
  const session = espSession;
  const port = espPort;
  espSession = undefined;
  espPort = undefined;
  espPortTransport = undefined;
  try {
    if (session) await session.close();
    else await port.close();
  } catch {}
}

function syncMobileTransportPicker() {
  for (const button of document.querySelectorAll("[data-mobile-transport]")) {
    const option = elements.flashTransport.querySelector(`option[value="${button.dataset.mobileTransport}"]`);
    button.disabled = Boolean(option?.disabled);
    button.setAttribute("aria-checked", String(button.dataset.mobileTransport === elements.flashTransport.value));
  }
}

function showFatalError(error) {
  building = false; running = false; runAfterBuild = false; compilerReady = false;
  updateReadyState(); setCompilerStatus("error", "Unavailable");
  elements.output.textContent = `${error.message || error}\n`; showPanel("output");
}

function defineTheme() {
  monaco.editor.defineTheme("renvo-dark", {
    base: "vs-dark", inherit: true,
    rules: [
      { token: "keyword", foreground: "C586C0" }, { token: "type", foreground: "4EC9B0" },
      { token: "string", foreground: "CE9178" }, { token: "number", foreground: "B5CEA8" },
      { token: "comment", foreground: "6A9955" },
    ],
    colors: {
      "editor.background": "#1e1e1e", "editor.foreground": "#d4d4d4",
      "editorLineNumber.foreground": "#858585", "editorLineNumber.activeForeground": "#c6c6c6",
      "editor.lineHighlightBackground": "#2a2d2e66", "editorCursor.foreground": "#aeafad",
      "editor.selectionBackground": "#264f78", "editor.inactiveSelectionBackground": "#3a3d41",
      "editorIndentGuide.background1": "#404040", "editorIndentGuide.activeBackground1": "#707070",
      "editorGutter.background": "#1e1e1e", "scrollbarSlider.background": "#79797966",
      "scrollbarSlider.hoverBackground": "#646464b3",
    },
  });
}

function loadScript(url) {
  return new Promise((resolve, reject) => {
    const script = document.createElement("script"); script.src = url; script.onload = resolve;
    script.onerror = () => reject(new Error(`Could not load Monaco Editor ${MONACO_VERSION}.`));
    document.head.append(script);
  });
}

function saveFiles() {
  if (!models.size) return;
  const values = {};
  for (const [name, model] of models) if (isEditableFile(name)) values[name] = model.getValue();
  try { localStorage.setItem("renvo.playground.files.v1", JSON.stringify(values)); } catch {}
}

function loadSavedFiles() {
  try {
    const value = JSON.parse(localStorage.getItem("renvo.playground.files.v1") || "{}");
    return value && typeof value === "object" && !Array.isArray(value) ? value : {};
  } catch { return {}; }
}

function clearArtifactUrls() { for (const url of artifactUrls) URL.revokeObjectURL(url); artifactUrls = []; }

function splitArguments(text) {
  const args = []; let value = ""; let quote = ""; let escaped = false; let active = false;
  for (const character of text) {
    if (escaped) { value += character; escaped = false; active = true; }
    else if (character === "\\" && quote !== "'") { escaped = true; active = true; }
    else if (quote) { if (character === quote) quote = ""; else value += character; active = true; }
    else if (character === "'" || character === '"') { quote = character; active = true; }
    else if (/\s/.test(character)) { if (active) { args.push(value); value = ""; active = false; } }
    else { value += character; active = true; }
  }
  if (escaped || quote) throw new Error("Unterminated quote or escape in arguments.");
  if (active) args.push(value);
  return args;
}

function replaceOutput(command, output) {
  const args = splitArguments(command); const at = args.indexOf("-o");
  if (at >= 0 && at + 1 < args.length) args[at + 1] = output;
  else args.unshift("-o", output);
  return args.join(" ");
}

function cleanPath(name) {
  return cleanLanguagePath(name, models, initialFiles);
}

function renderLibraryCatalog(catalog) {
  const children = [];
  const appendGroup = (title, entries) => {
    const heading = document.createElement("div");
    heading.className = "library-group"; heading.textContent = title; children.push(heading);
    for (const [name, item] of entries) children.push(libraryPackage(catalog, name, item));
  };
  appendGroup("Standard library", Object.entries(catalog.packages || {}).sort(([left], [right]) => left.localeCompare(right)));
  const platforms = Object.entries(catalog.platforms || {});
  const boards = new Map();
  for (const entry of platforms) {
    const board = entry[1].board;
    if (!board || !entry[1].main) continue;
    if (!boards.has(board)) boards.set(board, []);
    boards.get(board).push(entry);
  }
  if (boards.size) {
    const heading = document.createElement("div");
    heading.className = "library-group"; heading.textContent = "ESP32 platforms"; children.push(heading);
    for (const [board, entries] of [...boards].sort(([left], [right]) => left.localeCompare(right))) {
      const boardHeading = document.createElement("div");
      boardHeading.className = "library-subgroup"; boardHeading.textContent = board; children.push(boardHeading);
      for (const [name, item] of entries.sort(([left], [right]) => left.localeCompare(right))) {
        children.push(libraryPackage(catalog, name, item, item.root.split("/").pop()));
      }
    }
  }
  const forms = platforms.filter(([name]) => !name.includes("/examples/m5"));
  if (forms.length) appendGroup("Frameworks", forms);
  elements.stdlibTree.replaceChildren(...children);
}

function libraryPackage(catalog, importPath, item, label = importPath.replace(/^renvo\.dev\//, "")) {
  const wrapper = document.createElement("div");
  if (item.board) wrapper.className = "library-board-package";
  const button = document.createElement("button");
  button.type = "button"; button.className = "stdlib-package";
  button.textContent = label;
  button.title = item.main ? `${importPath} — click to use as the active app` : importPath;
  const files = document.createElement("div");
  files.className = "stdlib-files"; files.hidden = true;
  button.addEventListener("click", async () => {
    const opening = files.hidden;
    files.hidden = !opening; button.classList.toggle("open", opening);
    if (item.main) setBuildPackage(item.root, item.target, button);
    if (!opening) return;
    if (files.childElementCount) {
      if (item.main) await openPackageEntry(item);
      return;
    }
    button.disabled = true;
    try {
      await loadStandardPackage(importPath, catalog);
      const prefix = item.root || `std/${importPath}`;
      files.replaceChildren(...item.files.filter((file) => file.endsWith(".go")).map((file) => {
        const path = `${prefix}/${file}`;
        const entry = document.createElement("button");
        entry.type = "button"; entry.className = "stdlib-file"; entry.dataset.file = path;
        const icon = document.createElement("span"); icon.className = "go-icon"; icon.textContent = "Go";
        const label = document.createElement("span"); label.textContent = file;
        entry.append(icon, label);
        entry.addEventListener("click", async () => {
          await ensureSourceModel(path);
          openFile(path);
          if (isPhoneWorkspace()) showMobileView("editor");
        });
        return entry;
      }));
      if (item.main) await openPackageEntry(item);
    } catch (error) {
      files.replaceChildren(Object.assign(document.createElement("span"), { className: "tree-loading", textContent: error.message }));
    } finally {
      button.disabled = false;
    }
  });
  wrapper.append(button, files);
  return wrapper;
}

async function openPackageEntry(item) {
  const entryFile = item.files.find((file) => file === "main.go") ||
    item.files.find((file) => file.endsWith(".go"));
  if (!entryFile) return;
  const path = `${item.root}/${entryFile}`;
  await ensureSourceModel(path);
  openFile(path);
}

function setBuildPackage(root, target, button) {
  activeBuildRoot = root === "." ? "." : `./${root}`;
  if (target && target !== selectedTarget?.name) selectTarget(target, true);
  let args;
  try { args = splitArguments(elements.command.value); } catch { return; }
  if (args.length && (args[args.length - 1] === "." || args[args.length - 1].startsWith("./"))) args[args.length - 1] = activeBuildRoot;
  else args.push(activeBuildRoot);
  elements.command.value = args.join(" ");
  document.querySelectorAll(".stdlib-package").forEach((item) => item.classList.toggle("build-root", Boolean(button) && item === button));
  markBuildStale();
}

async function ensureSourceModel(path) {
  const name = cleanPath(path);
  if (models.has(name)) return models.get(name);
  if (!standardCatalogPromise) return undefined;
  const catalog = await standardCatalogPromise;
  let importPath;
  if (name.startsWith("std/")) importPath = name.slice(4, name.lastIndexOf("/"));
  else {
    for (const [candidate, item] of Object.entries(catalog.platforms || {})) {
      if (name.startsWith(`${item.root}/`)) { importPath = candidate; break; }
    }
  }
  if (!importPath) return undefined;
  await loadStandardPackage(importPath, catalog);
  const source = stdlibFiles.get(name);
  if (!source) return undefined;
  const model = monaco.editor.createModel(decoder.decode(source), name.endsWith(".go") ? "go" : "plaintext", monaco.Uri.parse(`file:///${name}`));
  models.set(name, model);
  return model;
}

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / 1048576).toFixed(1)} MiB`;
}

function formatElapsed(milliseconds) {
  return milliseconds < 1000 ? `${milliseconds.toFixed(1)} ms` : `${(milliseconds / 1000).toFixed(2)} s`;
}
