import { ESPWebSerial, requestESPPort } from "./esp-webserial.mjs";
import { preferredESPTransport, requestESPUSBPort } from "./esp-webusb.mjs";
import { ESPJTAGHotReloadSession, requestESPUSBJTAG, supportsESPWebUSBJTAG } from "./esp-webusb-jtag.mjs";
import { PicoMonitorHotReloadSession, requestPicoMonitor } from "./pico-webusb-monitor.mjs";
import { installEditorOpener } from "./editor-navigation.mjs";
import { fetchAsset } from "./asset-fetch.mjs";
import { cleanLanguagePath, isCLibrarySourcePath, sourceImportPath } from "./language-path.mjs";
import { SerialPlotter, SerialPlotterView } from "./serial-plotter.mjs";
import { decodeProjectZip, decodeSharedProject, encodeProjectZip, encodeSharedProject, normalizeProjectPath } from "./project-archive.mjs";
import { chooseESPTransportAvailability, detectDeviceProfile } from "./device-profile.mjs";
import { C_LANGUAGE_ID, registerCLanguage } from "./c-language.mjs";
import { RTG_LANGUAGE_ID, registerRTGLanguage } from "./rtg-language.mjs";
import { MAKEFILE_LANGUAGE_ID, registerMakefileLanguage } from "./makefile-language.mjs";
import { generateBrowserTestProject } from "./test-project.mjs";
import { deleteProjectSnapshot, loadCurrentProject, loadPreparedBackends, loadProjectSnapshots, saveCurrentProject, savePreparedBackend, saveProjectSnapshot } from "./workspace-store.mjs";
import { buildReadiness } from "./build-readiness.mjs";
import { hasDownloadableOutput, targetCapabilities, targetCapabilityHint, targetCapabilityTags } from "./target-capabilities.mjs";

const MONACO_VERSION = "0.56.0";
const encoder = new TextEncoder();
const decoder = new TextDecoder();
const parameters = new URLSearchParams(location.search);
const browserAssetRoot = new URL(".", import.meta.url);
const bundleRoot = browserAssetRoot.pathname.endsWith("/browser/")
  ? new URL("../", browserAssetRoot)
  : browserAssetRoot;
const MONACO_ROOT = new URL(parameters.get("monaco") || `https://cdn.jsdelivr.net/npm/monaco-editor@${MONACO_VERSION}/min/`, location.href).href.replace(/\/$/, "");
const compilerURLOverride = parameters.has("compiler") ? new URL(parameters.get("compiler"), bundleRoot).href : "";
const fallbackBackendUrl = new URL(parameters.get("backend") || "backends/wasi-wasm32.wasm", bundleRoot).href;
const catalogUrl = new URL(parameters.get("catalog") || "targets.json", bundleRoot).href;
const worker = new Worker(new URL("./worker.mjs", import.meta.url), { type: "module" });

const elements = {
  command: document.querySelector("#command"),
  compile: document.querySelector("#compile"),
  run: document.querySelector("#run"),
  test: document.querySelector("#test"),
  flashTransport: document.querySelector("#flash-transport"),
  targetPicker: document.querySelector("#target-picker"),
  targetButton: document.querySelector("#target-button"),
  targetLabel: document.querySelector("#target-label"),
  targetMenu: document.querySelector("#target-menu"),
  runArgs: document.querySelector("#run-args"),
  runStdin: document.querySelector("#run-stdin"),
  terminalCommand: document.querySelector("#terminal-command"),
  terminalCommandRun: document.querySelector("#terminal-command-run"),
  terminalOutput: document.querySelector("#terminal-output"),
  plotterLegend: document.querySelector("#plotter-legend"),
  plotterCanvas: document.querySelector("#serial-plotter-canvas"),
  togglePlotterSize: document.querySelector("#toggle-plotter-size"),
  clearOutput: document.querySelector("#clear-output"),
  togglePanel: document.querySelector("#toggle-panel"),
  compilerStatus: document.querySelector("#compiler-status"),
  languageStatus: document.querySelector("#language-status"),
  languageMode: document.querySelector("#language-mode"),
  cursorStatus: document.querySelector("#cursor-status"),
  memoryStatus: document.querySelector("#memory-status"),
  problemStatus: document.querySelector("#problem-status"),
  problemCount: document.querySelector("#problem-count"),
  output: document.querySelector("#output"),
  problems: document.querySelector("#problems"),
  testsOutput: document.querySelector("#tests-output"),
  searchResults: document.querySelector("#search-results"),
  searchQuery: document.querySelector("#search-query"),
  searchMatches: document.querySelector("#search-matches"),
  preview: document.querySelector("#preview"),
  workbench: document.querySelector(".workbench"),
  editorHost: document.querySelector("#editor"),
  helpView: document.querySelector("#help-view"),
  helpTree: document.querySelector("#help-tree"),
  sidebarExamples: document.querySelector("#sidebar-examples"),
  sidebarExampleCount: document.querySelector("#sidebar-example-count"),
  stdlibTree: document.querySelector("#stdlib-tree"),
  fileTree: document.querySelector("#file-tree"),
  openEditorTabs: document.querySelector("#open-editor-tabs"),
  outlineTree: document.querySelector("#outline-tree"),
  projectName: document.querySelector("#project-name"),
  sidebarProjectName: document.querySelector("#sidebar-project-name"),
  projectFileCount: document.querySelector("#project-file-count"),
  toggleSidebar: document.querySelector("#toggle-sidebar"),
  outlineCount: document.querySelector("#outline-count"),
  copyHelpPage: document.querySelector("#copy-help-page"),
  projectMenuButton: document.querySelector("#project-menu"),
  projectActionMenu: document.querySelector("#project-action-menu"),
  fileActionMenu: document.querySelector("#file-action-menu"),
  projectFileInput: document.querySelector("#project-file-input"),
  projectDirectoryInput: document.querySelector("#project-directory-input"),
  backendFileInput: document.querySelector("#backend-file-input"),
  buildMode: document.querySelector("#build-mode"),
  arenaSize: document.querySelector("#arena-size"),
  emitUnit: document.querySelector("#emit-unit"),
  emitImage: document.querySelector("#emit-image"),
  windowsGUI: document.querySelector("#windows-gui"),
  textDialog: document.querySelector("#text-dialog"),
  textDialogTitle: document.querySelector("#text-dialog-title"),
  textDialogLabel: document.querySelector("#text-dialog-label"),
  textDialogInput: document.querySelector("#text-dialog-input"),
  textDialogAccept: document.querySelector("#text-dialog-accept"),
  textDialogError: document.querySelector("#text-dialog-error"),
  newFileDialog: document.querySelector("#new-file-dialog"),
  newFileForm: document.querySelector("#new-file-form"),
  newFileKind: document.querySelector("#new-file-kind"),
  newFilePath: document.querySelector("#new-file-path"),
  newFileHelp: document.querySelector("#new-file-help"),
  newFileError: document.querySelector("#new-file-error"),
  newProjectDialog: document.querySelector("#new-project-dialog"),
  newProjectForm: document.querySelector("#new-project-form"),
  confirmDialog: document.querySelector("#confirm-dialog"),
  confirmDialogTitle: document.querySelector("#confirm-dialog-title"),
  confirmDialogMessage: document.querySelector("#confirm-dialog-message"),
  confirmDialogAccept: document.querySelector("#confirm-dialog-accept"),
  snapshotDialog: document.querySelector("#snapshot-dialog"),
  snapshotName: document.querySelector("#snapshot-name"),
  snapshotList: document.querySelector("#snapshot-list"),
  exampleDialog: document.querySelector("#example-dialog"),
  exampleSearch: document.querySelector("#example-search"),
  exampleBoardFilter: document.querySelector("#example-board-filter"),
  exampleBoardList: document.querySelector("#example-board-list"),
  exampleTargetFilter: document.querySelector("#example-target-filter"),
  exampleResultCount: document.querySelector("#example-result-count"),
  exampleResults: document.querySelector("#example-results"),
  mobileSetup: document.querySelector("#mobile-setup"),
  mobileSetupTitle: document.querySelector("#mobile-setup-title"),
  mobileSetupDetail: document.querySelector("#mobile-setup-detail"),
  mobileSetupProgress: document.querySelector("#mobile-setup-progress"),
  devicePermissionDialog: document.querySelector("#device-permission-dialog"),
  devicePermissionIntro: document.querySelector("#device-permission-intro"),
  devicePermissionNote: document.querySelector("#device-permission-note"),
  devicePermissionAccept: document.querySelector("#device-permission-accept"),
  deviceWebUSBStatus: document.querySelector("#device-webusb-status"),
  picoMonitorInstaller: document.querySelector("#pico-monitor-installer"),
  picoMonitorBuild: document.querySelector("#pico-monitor-build"),
  picoMonitorBuildStatus: document.querySelector("#pico-monitor-build-status"),
  deviceWebSerialStatus: document.querySelector("#device-webserial-status"),
  ide: document.querySelector("#ide"),
  mobileStep: document.querySelector("#mobile-step"),
  mobileContext: document.querySelector("#mobile-context"),
  mobileEditorActions: document.querySelector(".mobile-editor-actions"),
  mobileTargetButton: document.querySelector("#mobile-target-button"),
  mobileDownload: document.querySelector("#mobile-download"),
  mobileRun: document.querySelector("#mobile-run"),
  mobileDeviceBuild: document.querySelector("#mobile-device-build"),
  mobileDeviceRun: document.querySelector("#mobile-device-run"),
  mobileDeviceTarget: document.querySelector("#mobile-device-target"),
  mobileDeviceHint: document.querySelector("#mobile-device-hint"),
  mobileTransportStatus: document.querySelector("#mobile-transport-status"),
  mobileDeviceOutput: document.querySelector("#mobile-device-output"),
  mobileTargetList: document.querySelector("#mobile-target-list"),
  mobileFlashView: document.querySelector("#mobile-flash-view"),
  mobileFlashState: document.querySelector("#mobile-flash-state"),
  mobileFlashProgress: document.querySelector("#mobile-flash-progress"),
  mobileFlashDetail: document.querySelector("#mobile-flash-detail"),
  mobileFlashOutput: document.querySelector("#mobile-flash-output"),
  copyToPlayground: document.querySelector("#copy-to-playground"),
  formatFile: document.querySelector("#format-file"),
  useBackend: document.querySelector("#use-backend"),
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
const initialCFiles = {
  "main.c": `#include <stdio.h>

int main(void) {
    printf("Hello from Renvo C!\\n");
    return 0;
}
`,
};
let fileValues = { ...initialFiles };
const editableFiles = new Set(Object.keys(fileValues));
const editableBaselines = new Map(Object.entries(fileValues));
const models = new Map();
const openFiles = [];
const stdlibFiles = new Map();
const examplePreviewFiles = new Map();
const expandedExamples = new Set();
const loadedStandardPackages = new Set();
const loadingStandardPackages = new Map();
const languageRequests = new Map();
const formatRequests = new Map();
const backendRequests = new Map();
let languageWorkspaceRevision = 1;
let sentLanguageWorkspaceRevision = 0;
const backendReady = new Set();
const prefetchingBackends = new Set();
let monaco;
let editor;
let activeFile = "main.go";
let lastWorkspaceFile = "main.go";
let activeHelp = "";
let deepLinksReady = false;
let applyingDeepLink = false;
let compilerReady = false;
let building = false;
let running = false;
let buildRevision = 1;
let pendingBuild;
let runAfterBuild = false;
const inlineDownloadLimit = 16 * 1024 * 1024;
let lastRunnableArtifact;
let espPort;
let espSession;
let espPortTransport;
let selectedTarget;
let restoredTargetName = "";
let targetCatalog;
let standardCatalog;
let standardCatalogPromise;
let exampleCatalogPromise;
let exampleBoardSelectionTouched = false;
let analysisTimer;
let languageGeneration = 0;
let latestAnalysisRequestID = 0;
let validationTimer;
let validationGeneration = 0;
let pendingValidation;
let buildValidationState = "checking";
let buildValidationRevision = 0;
let validatedBuild;
let requestID = 0;
let focusedTargetIndex = -1;
let activeBuildRoot = ".";
let externalBuildLanguage = "go";
let autoBuildPending = parameters.has("run");
let plotterAutoShown = false;
let projectName = "playground";
let projectLanguage = "go";
let saveTimer;
let testBuild = false;
let testRunning = false;
let previewURL;
let browserHostParts;
let fileMenuTarget = "";
let textDialogResolve;
let confirmDialogResolve;
let devicePermissionResolve;
let mobileDeploymentActive = false;
let mobileDeploymentLabel = "";
let mobileDeploymentStep = "";
let cLibraryPromise;
const customBackendURLs = new Map();
const cachedBackendRecords = new Map();
const projectBackendRoots = new Set();
const plotterView = new SerialPlotterView(elements.plotterCanvas, elements.plotterLegend);
const serialPlotter = new SerialPlotter({ onChange: (data) => plotterView.update(data) });
plotterView.update(serialPlotter.snapshot());

setupShell();
if ("serviceWorker" in navigator && (location.protocol === "https:" || location.hostname === "localhost")) {
  navigator.serviceWorker.register(new URL("./service-worker.mjs", import.meta.url), { type: "module" }).catch(() => {});
}
boot().catch(showFatalError);

async function boot() {
  setSetupStep("workspace", "active", "Opening your saved workspace…");
  setSetupStep("catalog", "active", "Loading boards and examples…");
  const catalogPromise = loadTargetCatalog();
  exampleCatalogPromise = catalogPromise.then(async () => {
    if (!standardCatalogPromise) throw new Error("The example catalog is unavailable.");
    return standardCatalogPromise;
  });
  exampleCatalogPromise.then(
    () => setSetupStep("catalog", "done", "Boards and examples are ready."),
    () => setSetupStep("catalog", "error", "Boards and examples could not be loaded."),
  );
  maybeOpenMobileExamples();
  await restoreProject();
  setSetupStep("workspace", "done", "Workspace open.");
  setSetupStep("editor", "active", "Downloading the code editor…");
  const monacoPromise = loadMonaco().then(() => {
    setSetupStep("editor", "done", "Editor ready.");
  });
  const catalog = await catalogPromise;
  targetCatalog = catalog;
  await installCachedBackends();
  configureTargets(catalog.targets);
  if (elements.exampleDialog.open && standardCatalog) {
    selectInitialExampleBoard();
    renderExampleBrowser();
  }
  await monacoPromise;
  if (elements.exampleDialog.open && standardCatalog) renderExampleBrowser();
  const viewParameters = new URLSearchParams(location.search);
  if (["help", "source", "example"].some((name) => viewParameters.has(name)) && standardCatalogPromise) {
    try { await standardCatalogPromise; } catch {}
  }
  await restoreDeepLink();
  deepLinksReady = true;
  installLanguageProviders();
  const languageService = catalog.languageService ? new URL(catalog.languageService, catalogUrl).href : "";
  const formatter = catalog.formatter ? new URL(catalog.formatter, catalogUrl).href : "";
  const backendJIT = catalog.backendJIT ? new URL(catalog.backendJIT, catalogUrl).href : "";
  const vmBackend = catalog.vmBackend ? new URL(catalog.vmBackend, catalogUrl).href : "";
  const compiler = compilerURLOverride || new URL(catalog.compiler || "renvo.wasm", catalogUrl).href;
  const linker = new URL(catalog.linker || "renvo-linker.wasm", catalogUrl).href;
  setSetupStep("compiler", "active", "Downloading the compiler…");
  await initializeCompiler(compiler, linker, languageService, formatter, backendJIT, vmBackend);
  setSetupStep("compiler", "done", "Compiler ready. Choose an example to continue.");
  prefetchTargetBackend(selectedTarget);
  prefetchExampleBoard();
  await restoreProjectBackends();
  scheduleAnalysis(20);
  scheduleBuildValidation(20);
}

async function loadTargetCatalog() {
  try {
    const response = await fetchAsset(catalogUrl);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    const catalog = await response.json();
    if (!Array.isArray(catalog.targets) || !catalog.targets.length) throw new Error("target catalog is empty");
    if (catalog.stdlib) {
      const stdlibUrl = new URL(catalog.stdlib, catalogUrl).href;
      standardCatalogPromise = fetchAsset(stdlibUrl).then((response) => {
        if (!response.ok) throw new Error(`could not load standard library catalog: HTTP ${response.status}`);
        return response.json().then((value) => {
          standardCatalog = { ...value, url: stdlibUrl };
          return standardCatalog;
        });
      });
      standardCatalogPromise.then((value) => {
        renderLibraryCatalog(value);
        renderHelpCatalog(value);
        renderSidebarExamples(value);
      }, (error) => {
        elements.stdlibTree.textContent = error.message;
        elements.helpTree.textContent = error.message;
        elements.sidebarExamples.textContent = error.message;
      });
    }
    return catalog;
  } catch (error) {
    elements.output.textContent = `Target catalog unavailable (${error.message}); using the WASI backend from the URL.\n`;
    return { languageService: "", targets: [{
      name: "wasi/wasm32", backendTarget: "wasi/wasm32", backend: fallbackBackendUrl,
      cBackend: new URL(parameters.get("cbackend") || "backends/native-c.wasm", bundleRoot).href,
      output: "app.wasm", runnable: true, tags: ["wasi", "wasip1", "wasm", "wasm32"],
    }] };
  }
}

function configureTargets(targets) {
  const visibleTargets = targets.filter((target) => !target.hidden).sort((left, right) => {
    const leftBoard = isBoardTarget(left) ? 0 : 1;
    const rightBoard = isBoardTarget(right) ? 0 : 1;
    return leftBoard - rightBoard;
  });
  const entries = [];
  const mobileEntries = [];
  let group = "";
  for (let index = 0; index < visibleTargets.length; index++) {
    const target = visibleTargets[index];
    const nextGroup = targetGroup(target);
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
    option.textContent = targetDisplayName(target);
    option.title = target.name;
    entries.push(option);
    const mobileOption = document.createElement("button");
    mobileOption.type = "button";
    mobileOption.className = "mobile-target-option";
    mobileOption.dataset.target = target.name;
    mobileOption.setAttribute("role", "option");
    mobileOption.setAttribute("aria-selected", "false");
    const mobileLabel = document.createElement("span");
    mobileLabel.className = "mobile-target-name";
    mobileLabel.textContent = targetDisplayName(target);
    const capabilities = document.createElement("span");
    capabilities.className = "target-capabilities";
    for (const tag of targetCapabilityTags(target)) {
      const badge = document.createElement("span");
      badge.className = "target-capability";
      badge.dataset.capability = tag.name;
      badge.textContent = tag.label;
      capabilities.append(badge);
    }
    const selected = document.createElement("span");
    selected.className = "mobile-target-selected";
    selected.textContent = "Selected";
    selected.setAttribute("aria-hidden", "true");
    mobileOption.append(mobileLabel, capabilities, selected);
    mobileOption.title = target.name;
    mobileOption.addEventListener("click", () => {
      selectTarget(target.name, true);
      updateMobileHeader();
    });
    mobileEntries.push(mobileOption);
  }
  elements.targetMenu.replaceChildren(...entries);
  elements.mobileTargetList.replaceChildren(...mobileEntries);
  const restoredAvailable = restoredTargetName && visibleTargets.some((target) => target.name === restoredTargetName);
  const requested = restoredAvailable ? restoredTargetName : selectedTarget?.name || parameters.get("target");
  const initial = visibleTargets.some((target) => target.name === requested) ? requested :
    visibleTargets.some((target) => target.name === "wasi/wasm32") ? "wasi/wasm32" : visibleTargets[0].name;
  selectTarget(initial, false);
}

function targetGroup(target) {
  if (target.projectBackend) return "Project backends";
  if (isBoardTarget(target)) return "Boards";
  const name = target.name;
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
  registerCLanguage(monaco);
  registerRTGLanguage(monaco);
  registerMakefileLanguage(monaco, () => [...models.keys()]);
  defineTheme();
  for (const [name, value] of Object.entries(fileValues)) {
    createProjectModel(name, value);
  }
  if (!models.has(activeFile)) activeFile = models.keys().next().value || "main.go";
  if (!openFiles.includes(activeFile)) openFiles.push(activeFile);
  renderWorkspaceFiles();
  renderEditorTabs();
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
    hover: { enabled: true, delay: 80, sticky: true },
  });
  installEditorOpener(monaco, {
    cleanPath,
    ensureSourceModel,
    openFile,
    get editor() { return editor; },
  });
  editor.onDidChangeCursorPosition(({ position }) => {
    elements.cursorStatus.textContent = `Ln ${position.lineNumber}, Col ${position.column}`;
  });
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, primaryTargetAction);
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.Enter, runTests);
  editor.addCommand(monaco.KeyCode.F5, runArtifact);
  editor.addCommand(monaco.KeyMod.Shift | monaco.KeyMod.Alt | monaco.KeyCode.KeyF, formatActiveFile);
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyF, searchProject);
  elements.editorHost.querySelector(".editor-loading")?.remove();
  openFile(activeFile);
  configureEditorForViewport();
  updateReadyState();
  if (!isPhoneWorkspace()) editor.focus();
}

function initializeCompiler(compiler, linker, languageService, formatter, backendJIT, vmBackend) {
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
    worker.postMessage({ type: "init", compiler, linker, languageService, formatter, backendJIT, vmBackend });
  });
}

function prefetchTargetBackend(target) {
  if (!compilerReady || !targetCatalog || !isBoardTarget(target)) return;
  let buildTarget = target;
  if (target.device === "rp2") {
    buildTarget = targetCatalog.targets.find((candidate) => candidate.name === "rp2-debug/thumb") || target;
  } else if (elements.flashTransport.value === "webusb" && supportsESPWebUSBJTAG(deviceMachineTarget(target))) {
    buildTarget = targetCatalog.targets.find((candidate) => candidate.name === "esp32c6-jtag/riscv32") || target;
  }
  if (!buildTarget.backend) return;
  const backend = new URL(buildTarget.backend, catalogUrl).href;
  if (backendReady.has(backend) || prefetchingBackends.has(backend)) return;
  prefetchingBackends.add(backend);
  worker.postMessage({
    type: "prefetch-backend", backend,
    backendFormat: buildTarget.backendFormat || "wasm",
  });
}

function receiveBackendPrefetch(message) {
  prefetchingBackends.delete(message.backend);
  if (message.ok) backendReady.add(message.backend);
}

function prefetchExampleBoard() {
  const name = elements.exampleBoardFilter.value;
  if (!name || !standardCatalog) return;
  const board = catalogBoards().find((choice) => choice.name === name);
  if (board) prefetchTargetBackend(targetCatalog?.targets.find((target) => target.name === board.target));
}

worker.addEventListener("message", (event) => {
  if (event.data.type === "result") renderResult(event.data);
  else if (event.data.type === "validation-result") receiveBuildValidation(event.data);
  else if (event.data.type === "run-result") renderRunResult(event.data);
  else if (event.data.type === "language-result") receiveLanguageResult(event.data);
  else if (event.data.type === "format-result") receiveFormatResult(event.data);
  else if (event.data.type === "backend-result") receiveBackendResult(event.data);
  else if (event.data.type === "init-progress") setSetupStep("compiler", "active", event.data.message);
  else if (event.data.type === "compile-progress") receiveCompileProgress(event.data);
  else if (event.data.type === "backend-prefetch") receiveBackendPrefetch(event.data);
  else if (event.data.type === "format-progress") elements.languageStatus.textContent = event.data.message;
});
worker.addEventListener("error", (event) => showFatalError(new Error(event.message)));

function setupShell() {
  elements.compile.addEventListener("click", primaryTargetAction);
  elements.run.addEventListener("click", secondaryTargetAction);
  elements.test.addEventListener("click", runTests);
  elements.terminalCommandRun.addEventListener("click", runTerminalCommand);
  elements.terminalCommand.addEventListener("keydown", (event) => { if (event.key === "Enter") runTerminalCommand(); });
  document.querySelector("#new-file").addEventListener("click", createWorkspaceFile);
  document.querySelector("#browse-examples").addEventListener("click", openExampleBrowser);
  elements.toggleSidebar.addEventListener("click", toggleSidebar);
  elements.projectMenuButton.addEventListener("click", toggleProjectActionMenu);
  elements.projectActionMenu.addEventListener("click", handleProjectAction);
  elements.fileActionMenu.addEventListener("click", handleFileAction);
  document.querySelector("#import-backend").addEventListener("click", () => elements.backendFileInput.click());
  elements.formatFile.addEventListener("click", formatActiveFile);
  elements.useBackend.addEventListener("click", () => useProjectBackend(activeFile));
  elements.copyHelpPage.addEventListener("click", copyActiveHelpPage);
  document.querySelector("#search-project").addEventListener("click", searchProject);
  document.querySelector("#search-form").addEventListener("submit", submitProjectSearch);
  document.querySelector("#advanced-heading").addEventListener("click", toggleAdvancedBuild);
  document.querySelector("#outline-heading").addEventListener("click", toggleOutline);
  document.querySelector("#examples-heading").addEventListener("click", toggleSidebarExamples);
  document.querySelector("#help-heading").addEventListener("click", toggleHelp);
  document.querySelector("#library-heading").addEventListener("click", toggleLibrary);
  elements.projectFileInput.addEventListener("change", () => importProjectFiles(elements.projectFileInput.files));
  elements.projectDirectoryInput.addEventListener("change", () => importProjectFiles(elements.projectDirectoryInput.files, true));
  elements.backendFileInput.addEventListener("change", () => importPreparedBackend(elements.backendFileInput.files));
  for (const control of [elements.buildMode, elements.arenaSize, elements.emitUnit, elements.emitImage, elements.windowsGUI]) {
    control.addEventListener("change", markBuildStale);
    control.addEventListener("input", markBuildStale);
  }
  configureFlashTransports();
  elements.flashTransport.addEventListener("change", changeFlashTransport);
  elements.mobileRun.addEventListener("click", () => {
    if (mobileDeploymentActive) openMobileFlashView(elements.mobileFlashState.textContent);
    else runArtifact();
  });
  elements.mobileDownload.addEventListener("click", downloadValidatedArtifact);
  elements.mobileDeviceBuild.addEventListener("click", primaryTargetAction);
  elements.mobileDeviceRun.addEventListener("click", secondaryTargetAction);
  elements.mobileTargetButton.addEventListener("click", () => {
    showMobileView(elements.ide.dataset.mobileView === "device" ? "editor" : "device");
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
  window.addEventListener("popstate", () => { if (deepLinksReady) restoreDeepLink(); });
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
  elements.command.addEventListener("input", () => { syncBuildRootFromCommand(); markBuildStale(); saveFiles(); });
  elements.command.addEventListener("keydown", (event) => { if (event.key === "Enter") primaryTargetAction(); });
  elements.fileTree.addEventListener("contextmenu", handleWorkspaceFileMenu);
  document.addEventListener("pointerdown", (event) => {
    if (!elements.projectActionMenu.contains(event.target) && event.target !== elements.projectMenuButton) closeProjectActionMenu();
    if (!elements.fileActionMenu.contains(event.target)) closeFileActionMenu();
  });
  window.addEventListener("keydown", (event) => {
    if (event.key === "Escape") { closeProjectActionMenu(); closeFileActionMenu(); }
  });
  elements.textDialog.addEventListener("close", finishTextDialog);
  elements.textDialog.querySelector("form").addEventListener("submit", validateTextDialog);
  elements.newFileForm.addEventListener("submit", createNewWorkspaceFile);
  elements.newFileKind.addEventListener("change", updateNewFileSuggestion);
  elements.newFilePath.addEventListener("input", updateNewFileKindFromPath);
  elements.newProjectForm.addEventListener("submit", createNewProject);
  elements.confirmDialog.addEventListener("close", finishConfirmDialog);
  document.querySelector("#snapshot-dialog-close").addEventListener("click", () => elements.snapshotDialog.close());
  document.querySelector("#snapshot-create-form").addEventListener("submit", saveSnapshotFromDialog);
  elements.snapshotList.addEventListener("click", handleSnapshotAction);
  document.querySelector("#example-dialog-close").addEventListener("click", () => elements.exampleDialog.close());
  for (const control of [elements.exampleSearch, elements.exampleBoardFilter, elements.exampleTargetFilter]) {
    control.addEventListener(control === elements.exampleSearch ? "input" : "change", renderExampleBrowser);
  }
  elements.exampleBoardList.addEventListener("click", selectExampleBoard);
  elements.exampleResults.addEventListener("click", handleExampleAction);
  document.querySelector("#device-permission-close").addEventListener("click", () => elements.devicePermissionDialog.close("cancel"));
  document.querySelector("#device-permission-cancel").addEventListener("click", () => elements.devicePermissionDialog.close("cancel"));
  elements.devicePermissionAccept.addEventListener("click", acceptDevicePermission);
  elements.picoMonitorBuild.addEventListener("click", compilePicoMonitor);
  elements.devicePermissionDialog.addEventListener("close", finishDevicePermission);
  document.querySelectorAll(".panel-tab").forEach((button) => button.addEventListener("click", () => showPanel(button.dataset.panel)));
  elements.togglePanel.addEventListener("click", togglePanel);
  elements.togglePlotterSize.addEventListener("click", togglePlotterSize);
  document.querySelector("#close-panel").addEventListener("click", () => {
    setPlotterExpanded(false);
    elements.workbench.classList.add("panel-hidden");
    elements.togglePanel.setAttribute("aria-pressed", "false");
  });
  elements.clearOutput.addEventListener("click", clearActivePanel);
  elements.problemStatus.addEventListener("click", () => showPanel("problems"));
  window.addEventListener("keydown", (event) => {
    if ((event.ctrlKey || event.metaKey) && !event.shiftKey && !event.altKey && event.key.toLowerCase() === "s") {
      event.preventDefault();
      if (!event.repeat) saveAndDeploy();
      return;
    }
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "j") {
      event.preventDefault();
      togglePanel();
    }
    if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key.toLowerCase() === "f") {
      event.preventDefault(); searchProject();
    }
  });
  window.addEventListener("pagehide", () => {
    if (espSession) espSession.close().catch(() => {});
    else if (espPort) espPort.close().catch(() => {});
  });
  syncBuildScope();
}

function selectTarget(name, updateCommand) {
  const previousTarget = selectedTarget;
  const changed = selectedTarget?.name !== name;
  selectedTarget = targetCatalog?.targets.find((target) => target.name === name);
  if (!selectedTarget) return;
  if (updateCommand) restoredTargetName = selectedTarget.name;
  if (changed && isBoardTarget(previousTarget) && espPort) {
    if (espSession) espSession.close().catch(() => {});
    else espPort.close().catch(() => {});
    espSession = undefined;
    espPort = undefined;
    espPortTransport = undefined;
  }
  elements.targetLabel.textContent = targetDisplayName(selectedTarget);
  elements.targetButton.title = `${targetDisplayName(selectedTarget)} · ${selectedTarget.name}`;
  const board = isBoardTarget(selectedTarget);
  if (selectedTarget.device === "rp2") elements.flashTransport.value = "webusb";
  updateFlashTransportChoices();
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
  if (changed) prefetchTargetBackend(selectedTarget);
  if (changed && standardCatalog) renderSidebarExamples(standardCatalog);
  if (changed && monaco) saveFiles();
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
  const index = Array.from(elements.targetMenu.querySelectorAll(".target-option"))
    .findIndex((option) => option.dataset.target === selectedTarget?.name);
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
  const count = elements.targetMenu.querySelectorAll(".target-option").length;
  if (event.key === "Home") setFocusedTarget(0, true);
  else if (event.key === "End") setFocusedTarget(count - 1, true);
  else setFocusedTarget(focusedTargetIndex + (event.key === "ArrowDown" ? 1 : -1), true);
}

function primaryTargetAction() {
  if (isBoardTarget(selectedTarget)) return runArtifact();
  return downloadValidatedArtifact();
}

function secondaryTargetAction() {
  if (isBoardTarget(selectedTarget)) return downloadValidatedArtifact();
  return runArtifact();
}

function targetDisplayName(target) {
  if (target?.label) return target.label;
  const [platform = "Target", architecture = ""] = (target?.name || "").split("/");
  const platforms = {
    browser: "Web application", darwin: "macOS", freebsd: "FreeBSD", linux: "Linux",
    netbsd: "NetBSD", openbsd: "OpenBSD", vm: "Renvo VM", wasi: "WebAssembly (WASI)", windows: "Windows",
  };
  const architectures = {
    "386": "x86 (32-bit)", amd64: "x86-64", arm: "ARM (32-bit)", aarch64: "ARM64", arm64: "ARM64",
    riscv32: "RISC-V 32-bit", vm32: "32-bit bytecode", wasm32: "WebAssembly",
  };
  const system = platforms[platform] || platform.replaceAll(/[-_]/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
  const machine = architectures[architecture] || architecture.replaceAll(/[-_]/g, " ").toUpperCase();
  return machine ? `${system} · ${machine}` : system;
}

function downloadValidatedArtifact() {
  if (!hasDownloadableOutput(selectedTarget)) return;
  const cached = validatedBuild;
  if (!cached || cached.revision !== buildRevision || cached.target.name !== selectedTarget?.name || cached.result.exitCode !== 0) return;
  const artifact = cached.result.files.find((file) => file.name === cached.target.output) || cached.result.files[0];
  if (!artifact) {
    setCompilerStatus("error", "Build produced no downloadable output");
    return;
  }
  downloadArtifact(artifact);
}

function downloadArtifact(artifact, filename = artifact.name.split("/").pop()) {
  const data = artifact.data;
  const type = filename.endsWith(".wasm") ? "application/wasm" : "application/octet-stream";
  const url = createDownloadURL(data, type);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.hidden = true;
  document.body.append(link);
  link.click();
  link.remove();
  if (url.startsWith("blob:")) setTimeout(() => releaseDownloadURL(url), 60000);
  setCompilerStatus("ready", `Downloading ${filename}`);
}

async function compilePicoMonitor() {
  if (!compilerReady || building || selectedTarget?.device !== "rp2" || !standardCatalogPromise) return;
  const buildTarget = selectedTarget;
  building = true;
  elements.picoMonitorBuild.disabled = true;
  elements.picoMonitorBuildStatus.textContent = "Loading the Renvo monitor sources…";
  updateReadyState();
  try {
    const catalog = await standardCatalogPromise;
    await loadStandardPackage("renvo.dev/cmd/renvopico-monitor", catalog);
    const args = ["-arena-size", "8192", "-o", "renvo-rp2-monitor.uf2", "./cmd/renvopico-monitor"];
    args.unshift("-t", buildTarget.frontendTarget || buildTarget.name);
    for (const tag of buildTarget.tags || []) args.unshift("-tags", tag);
    if (buildTarget.definition) {
      args.unshift("-target-version", String(buildTarget.descriptorVersion), "-target-definition", buildTarget.definition);
    }
    const backend = new URL(buildTarget.backend, catalogUrl).href;
    const payload = workspacePayload();
    const id = ++requestID;
    pendingBuild = { id, revision: buildRevision, target: buildTarget, backend, args, action: "pico-monitor" };
    elements.picoMonitorBuildStatus.textContent = "Compiling the monitor with Renvo…";
    setCompilerStatus("busy", "Building Pico monitor…");
    worker.postMessage({
      type: "compile", id, args, files: payload.files, backend,
      backendTarget: buildTarget.backendTarget, backendFormat: buildTarget.backendFormat || "wasm",
    }, payload.transfers);
  } catch (error) {
    building = false;
    pendingBuild = undefined;
    elements.picoMonitorBuild.disabled = false;
    elements.picoMonitorBuildStatus.textContent = `Monitor build failed: ${error.message || error}`;
    setCompilerStatus("error", "Monitor build failed");
    updateReadyState();
  }
}

function publishValidatedBuild() {
  const cached = validatedBuild;
  if (!cached || cached.revision !== buildRevision || cached.target.name !== selectedTarget?.name || cached.result.exitCode !== 0) return false;
  saveFiles();
  clearMarkers();
  lastRunnableArtifact = undefined;
  elements.output.textContent = `$ renvo ${cached.args.join(" ")}\n`;
  showPanel("output");
  pendingBuild = {
    id: cached.result.id,
    revision: cached.revision,
    target: cached.target,
    backend: cached.backend,
    args: cached.args,
    action: "build",
  };
  renderResult({ ...cached.result, type: "result" });
  return true;
}

async function compileTarget(buildTarget) {
  if (!compilerReady || building || !monaco || !selectedTarget) return;
  if (!buildTarget) return;
  if (buildTarget.name === selectedTarget.name) {
    if (publishValidatedBuild()) return;
    if (buildValidationState !== "success" || buildValidationRevision !== buildRevision) return;
  }
  if (buildTarget.projectBackend && (buildTarget.backendStale || !buildTarget.backend)) {
    building = true;
    updateReadyState();
    closeTargetMenu();
    setCompilerStatus("busy", `Preparing ${buildTarget.name} backend…`);
    try {
      buildTarget = await prepareProjectBackend(buildTarget);
      selectedTarget = buildTarget;
      setCompilerStatus("ready", "Compiler ready");
    } catch (error) {
      renderProblems(parseDiagnostics(error.message || String(error)));
      showPanel("problems");
      setCompilerStatus("error", "Backend preparation failed");
      return;
    } finally {
      building = false;
      updateReadyState();
    }
  }
  const revision = buildRevision;
  let args;
  try {
    args = splitArguments(elements.command.value);
    args = controlledArguments(args, buildTarget);
  } catch (error) {
    runAfterBuild = false;
    updateReadyState();
    renderProblems([{ message: error.message, file: "", line: 0, column: 0 }]);
    showPanel("problems");
    return;
  }
  saveFiles();
  clearMarkers();
  lastRunnableArtifact = undefined;
  building = true;
  updateReadyState();
  closeTargetMenu();
  const backendPath = activeBuildLanguage() === "c" && buildTarget.cBackend
    ? buildTarget.cBackend : buildTarget.backend;
  const backend = new URL(backendPath, catalogUrl).href;
  const loading = !backendReady.has(backend);
  setCompilerStatus("busy", loading ? `Loading ${buildTarget.name} backend…` : "Building…");
  elements.output.textContent = `$ renvo ${args.join(" ")}\n`;
  showPanel("output");
  try {
    if (mobileDeploymentActive && runAfterBuild) {
      setMobileDeployStep("check", "active", "Loading the libraries used by this example…");
    }
    await ensureWorkspaceDependencies();
    const payload = workspacePayload();
    const id = ++requestID;
    pendingBuild = { id, revision, target: buildTarget, backend, args, action: "build" };
    worker.postMessage({
      type: "compile", id, args, files: payload.files,
      backend, backendTarget: buildTarget.backendTarget, backendFormat: buildTarget.backendFormat || "wasm",
      rtgDefinition: buildTarget.rtgDefinition ? new URL(buildTarget.rtgDefinition, catalogUrl).href : "",
      rtgDefinitionName: buildTarget.rtgDefinitionName || buildTarget.projectDefinition || "",
      rtgImports: (buildTarget.rtgImports || []).map((item) => ({ ...item, source: new URL(item.source, catalogUrl).href })),
    }, payload.transfers);
  } catch (error) {
    building = false;
    pendingBuild = undefined;
    runAfterBuild = false;
    updateReadyState();
    setCompilerStatus("error", "Build failed");
    elements.output.textContent += `${error.message}\n`;
    if (mobileDeploymentActive) failMobileDeployment(mobileDeploymentStep || "check", error.message || String(error));
  }
}

function receiveCompileProgress(message) {
  if (pendingBuild?.id !== message.id || !runAfterBuild || !mobileDeploymentActive) return;
  if (message.phase === "check") {
    setMobileDeployStep("check", "active", message.message);
  } else if (message.phase === "firmware") {
    setMobileDeployStep("check", "done", "Project code is ready.");
    setMobileDeployStep("firmware", "active", message.message);
  }
}

function controlledArguments(args, target) {
  const result = [];
  for (let i = 0; i < args.length; i++) {
    if ((args[i] === "-t" || args[i] === "-target-definition" || args[i] === "-target-version" || args[i] === "-arena-size") && i + 1 < args.length) {
      i++;
      continue;
    }
    if (args[i] === "-emit-unit" || args[i] === "-emit-image" || args[i] === "-windows-gui" || args[i] === "-mode" && i + 1 < args.length) {
      if (args[i] === "-mode") i++;
      continue;
    }
    if (args[i].startsWith("-mode=")) continue;
    result.push(args[i]);
  }
  validateBrowserArguments(result);
  if (result[0] === "make") return result;
  result.unshift("-t", target.frontendTarget || target.name);
  for (const tag of target.tags || []) result.unshift("-tags", tag);
  if (target.definition) result.unshift("-target-version", String(target.descriptorVersion), "-target-definition", target.definition);
  if (elements.arenaSize.value.trim()) result.unshift("-arena-size", elements.arenaSize.value.trim());
  if (elements.buildMode.value !== "executable") result.unshift(`-mode=${elements.buildMode.value}`);
  if (elements.emitUnit.checked) result.unshift("-emit-unit");
  if (elements.emitImage.checked) result.unshift("-emit-image");
  if (elements.windowsGUI.checked) result.unshift("-windows-gui");
  if (activeBuildLanguage() === "c") result.unshift("cc");
  return result;
}

function deviceMachineTarget(target = selectedTarget) {
  return target?.frontendTarget || target?.backendTarget || target?.name || "";
}

function isBoardTarget(target = selectedTarget) {
  return target?.device === "esp32" || target?.device === "rp2";
}

function isHotReloadDeployment(target = selectedTarget) {
  return target?.device === "rp2" ||
    (target?.device === "esp32" && elements.flashTransport.value === "webusb" && supportsESPWebUSBJTAG(deviceMachineTarget(target)));
}

function validateBrowserArguments(args) {
  const values = new Set(["-o", "-tags", "-arena-size", "-system", "-I", "-isystem", "-module-license"]);
  const flags = new Set(["-s", "-emit-unit", "-emit-image", "-windows-gui"]);
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (!arg.startsWith("-")) continue;
    if (values.has(arg)) {
      if (i + 1 >= args.length) throw new Error(`${arg} requires a value.`);
      i++;
    } else if (!flags.has(arg) && !arg.startsWith("-mode=")) {
      throw new Error(`The browser compiler does not support ${arg}.`);
    }
  }
}

async function runTests() {
  if (!compilerReady || building || running || !monaco || !targetCatalog) return;
  const target = targetCatalog.targets.find((item) => item.name === "wasi/wasm32");
  if (!target) { showProjectError(new Error("The WASI target required by the browser test runner is unavailable.")); return; }
  let generated;
  try { generated = generateBrowserTestProject(projectFiles()); }
  catch (error) { elements.testsOutput.textContent = `${error.message || error}\n`; showPanel("tests"); return; }
  building = true; testBuild = true; updateReadyState(); clearMarkers();
  elements.testsOutput.textContent = `$ renvo test .\nDiscovered ${generated.tests.length} test${generated.tests.length === 1 ? "" : "s"}; building…\n`;
  showPanel("tests");
  setCompilerStatus("busy", "Building tests…");
  try {
    await ensureWorkspaceDependencies();
    const payload = workspacePayload();
    for (const [name, source] of Object.entries(generated.files)) {
      const data = encoder.encode(source); payload.files.push({ name, data }); payload.transfers.push(data.buffer);
    }
    const id = ++requestID;
    const backend = new URL(target.backend, catalogUrl).href;
    pendingBuild = { id, revision: buildRevision, target, backend, action: "test" };
    worker.postMessage({
      type: "compile", id, args: ["-t", target.name, "-s", "-o", "renvo_tests/tests.wasm", "./renvo_tests"],
      files: payload.files, backend, backendTarget: target.backendTarget,
    }, payload.transfers);
  } catch (error) {
    building = false; testBuild = false; pendingBuild = undefined; updateReadyState();
    elements.testsOutput.textContent += `${error.message || error}\n`;
  }
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
  if (build.action === "pico-monitor") {
    const artifact = result.files.find((file) => file.name.endsWith("renvo-rp2-monitor.uf2")) || result.files[0];
    elements.picoMonitorBuild.disabled = false;
    if (result.exitCode === 0 && artifact) {
      backendReady.add(build.backend);
      downloadArtifact(artifact, "renvo-rp2-monitor.uf2");
      elements.picoMonitorBuildStatus.textContent = "Monitor downloaded. Copy it to the BOOTSEL USB drive, then reconnect the board normally.";
    } else {
      elements.picoMonitorBuildStatus.textContent = `Monitor build failed. ${text || "Open the build output for details."}`;
      setCompilerStatus("error", "Monitor build failed");
    }
    updateReadyState();
    return;
  }
  if (build.action === "test") {
    renderTestBuildResult(result, build, summary, text);
    return;
  }
  const buildOutput = build.action === "terminal" ? elements.terminalOutput : elements.output;
  buildOutput.textContent += `${text}${text && !text.endsWith("\n") ? "\n" : ""}${summary}\n`;
  elements.memoryStatus.textContent = `${(result.linearMemoryBytes / 1048576).toFixed(1)} MiB`;
  setCompilerStatus(result.exitCode === 0 ? "ready" : "error", result.exitCode === 0 ? "Build succeeded" : "Build failed");
  const diagnosticText = result.exitCode === 0 ? "" : [result.stderr, result.stdout].filter(Boolean).join("\n");
  const problems = parseDiagnostics(diagnosticText);
  renderProblems(problems);
  if (build.revision === buildRevision && build.target?.name === selectedTarget?.name) {
    buildValidationRevision = build.revision;
    buildValidationState = result.exitCode === 0 ? "success" : "failure";
    validatedBuild = result.exitCode === 0 ? { ...build, result } : undefined;
  }
  if (result.exitCode === 0) {
    backendReady.add(build.backend);
    const artifact = result.files.find((file) => file.name === build.target.output) || result.files[0];
    if ((build.target.runnable || isBoardTarget(build.target)) && artifact) {
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
  if (result.exitCode !== 0) showPanel(build.action === "terminal" ? "terminal" : problems.length ? "problems" : "output");
  if (result.exitCode !== 0 && shouldRun && isPhoneWorkspace()) {
    elements.terminalOutput.textContent = `${text}${text && !text.endsWith("\n") ? "\n" : ""}${summary}\n`;
    failMobileDeployment(mobileDeploymentStep || "firmware", "The firmware build failed. Open the activity log for the compiler message.");
  }
  if (result.exitCode === 0 && shouldRun) {
    if (mobileDeploymentActive) {
      setMobileDeployStep("check", "done", "Project code is ready.");
      setMobileDeployStep("firmware", "done", "Firmware built for the selected board.");
    }
    queueMicrotask(resumeArtifactAfterBuild);
  }
}

async function runTerminalCommand() {
  if (!compilerReady || building || !selectedTarget) return;
  let args;
  try {
    args = splitArguments(elements.terminalCommand.value.trim());
    if (args[0] === "renvo") args.shift();
    if (args[0] !== "make") throw new Error("The Web IDE terminal currently runs renvo make commands.");
  } catch (error) {
    elements.terminalOutput.textContent = `${error.message || error}\n`; showPanel("terminal"); return;
  }
  saveFiles(); building = true; updateReadyState(); clearMarkers(); showPanel("terminal");
  elements.terminalOutput.textContent = `$ renvo ${args.join(" ")}\n`;
  try {
    await ensureWorkspaceDependencies();
    const payload = workspacePayload(), id = ++requestID;
    const backendPath = selectedTarget.cBackend || selectedTarget.backend;
    const backend = new URL(backendPath, catalogUrl).href;
    pendingBuild = { id, revision: buildRevision, target: selectedTarget, backend, args, action: "terminal" };
    worker.postMessage({ type: "compile", id, args, files: payload.files, backend,
      backendTarget: selectedTarget.backendTarget, backendFormat: selectedTarget.backendFormat || "wasm" }, payload.transfers);
  } catch (error) {
    building = false; pendingBuild = undefined; updateReadyState();
    elements.terminalOutput.textContent += `${error.message || error}\n`;
  }
}

function renderTestBuildResult(result, build, summary, text) {
  building = false; testBuild = false; pendingBuild = undefined;
  const diagnosticText = result.exitCode === 0 ? "" : [result.stderr, result.stdout].filter(Boolean).join("\n");
  const problems = parseDiagnostics(diagnosticText); renderProblems(problems);
  elements.testsOutput.textContent += `${text}${text && !text.endsWith("\n") ? "\n" : ""}${summary}\n`;
  elements.memoryStatus.textContent = `${(result.linearMemoryBytes / 1048576).toFixed(1)} MiB`;
  if (result.exitCode !== 0) {
    setCompilerStatus("error", "Tests failed to build"); updateReadyState(); showPanel("tests"); return;
  }
  const artifact = result.files.find((file) => file.name.endsWith("tests.wasm")) || result.files[0];
  if (!artifact) {
    setCompilerStatus("error", "Test artifact missing"); updateReadyState(); return;
  }
  running = true; testRunning = true; updateReadyState(); setCompilerStatus("busy", "Running tests…");
  const data = artifact.data.slice(0);
  worker.postMessage({ type: "run", purpose: "test", id: ++requestID, name: "tests.wasm", data, args: [], stdin: "" }, [data]);
}

function runArtifact() {
  return runArtifactWithMode(false);
}

function resumeArtifactAfterBuild() {
  return runArtifactWithMode(true);
}

function deploymentBuildTarget() {
  if (selectedTarget?.device === "rp2") {
    const target = targetCatalog?.targets.find((candidate) => candidate.name === "rp2-debug/thumb");
    if (!target) throw new Error("The RP2 monitor debug backend is unavailable in this browser bundle.");
    return target;
  }
  if (elements.flashTransport.value === "webusb" && supportsESPWebUSBJTAG(deviceMachineTarget())) {
    const target = targetCatalog?.targets.find((candidate) => candidate.name === "esp32c6-jtag/riscv32");
    if (!target) throw new Error("The ESP32-C6 JTAG backend is unavailable in this browser bundle.");
    return target;
  }
  return selectedTarget;
}

async function runArtifactWithMode(resumeAfterBuild) {
  if ((!selectedTarget?.runnable && !isBoardTarget(selectedTarget)) || running) return;
  const board = isBoardTarget(selectedTarget);
  const activeESPPort = espPort && (espPort.readable || espPort.writable);
  const reusableESPPort = espPortTransport === "webusb" && espPort?.canReopen?.();
  if (board && !resumeAfterBuild && !activeESPPort && !reusableESPPort) {
    if (!await requestDevicePermission()) {
      if (isPhoneWorkspace()) showMobileView("device");
      return;
    }
    const plannedJTAG = isHotReloadDeployment();
    startMobileDeployment(plannedJTAG);
    setMobileDeployStep("usb", "active", "Choose your board in the browser's USB picker.");
    try {
      const previousSession = espSession;
      const previousPort = espPort;
      // Start the permission prompt while the click/key activation is still
      // live; cleanup can safely happen after the user selects the device.
      const transport = elements.flashTransport.value;
      const machineTarget = deviceMachineTarget();
      const jtag = isHotReloadDeployment();
	  const nextPort = selectedTarget.device === "rp2"
		? await requestPicoMonitor()
		: jtag ? await requestESPUSBJTAG(machineTarget)
		: transport === "webusb" ? await requestESPUSBPort(machineTarget)
		: await requestESPPort(machineTarget);
      espSession = undefined;
      espPort = undefined;
      if (previousSession) await previousSession.close();
      else if (previousPort) {
        try { await previousPort.close(); } catch {}
      }
      espPort = nextPort;
      espPortTransport = transport;
      setMobileDeployStep("usb", "done", "USB device selected.");
    } catch (error) {
      elements.terminalOutput.textContent = `${error.message || error}\n`;
      failMobileDeployment("usb", "USB device selection failed.");
      showPanel("terminal");
      return;
    }
  } else if (board && !resumeAfterBuild && !mobileDeploymentActive) {
    const plannedJTAG = isHotReloadDeployment();
    startMobileDeployment(plannedJTAG);
    setMobileDeployStep("usb", "done", "Using the connected USB device.");
  }
  if (board && !espPort) {
    elements.terminalOutput.textContent = "The selected board or debug probe disconnected before loading. Click Flash & Run again.\n";
    failMobileDeployment("usb", "The USB device disconnected before the load started.");
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
  let deploymentTarget;
  try {
    deploymentTarget = deploymentBuildTarget();
  } catch (error) {
    elements.terminalOutput.textContent = `${error.message || error}\n`;
    failMobileDeployment("firmware", error.message || String(error));
    showPanel("terminal");
    return;
  }
  const stale = !lastRunnableArtifact || lastRunnableArtifact.revision !== buildRevision ||
    lastRunnableArtifact.target !== deploymentTarget?.name;
  if (building || stale) {
    runAfterBuild = true;
    if (board) setMobileDeployStep("check", "active", "Loading project libraries…");
    updateReadyState();
    if (!building) compileTarget(deploymentTarget);
    return;
  }
  if (selectedTarget.name === "browser/wasm32") {
    running = true; updateReadyState();
    try {
      await showBrowserPreview(lastRunnableArtifact.data);
      elements.terminalOutput.textContent = "$ app.html\nBrowser application launched in Preview.\n";
      showPanel("preview");
    } catch (error) {
      elements.terminalOutput.textContent = `${error.message || error}\n`; showPanel("terminal");
    } finally { running = false; updateReadyState(); }
    return;
  }
  running = true;
  updateReadyState();
  showPanel("terminal");
  if (board) {
    setMobileDeployStep("load", "active", selectedTarget.device === "rp2" ? "Connecting to the resident USB monitor…" : "Connecting to the board over JTAG…");
    const portInfo = espPort.getInfo?.() || {};
    const identity = portInfo.usbVendorId === undefined ? "" :
      ` (USB ${portInfo.usbVendorId.toString(16).padStart(4, "0")}:${(portInfo.usbProductId || 0).toString(16).padStart(4, "0")})`;
    const picoMonitor = espPort?.transport === "webusb-pico-monitor";
    const jtag = espPort?.transport === "webusb-jtag" || espPort?.transport === "webusb-cmsis-dap";
    const hotReload = jtag || picoMonitor;
    const transportName = espPortTransport === "webusb" ? "WebUSB" : "WebSerial";
    serialPlotter.clear();
    plotterAutoShown = false;
    elements.terminalOutput.textContent = `$ ${picoMonitor ? "monitor-load" : jtag ? "jtag-load" : "flash"} --transport ${transportName} ${lastRunnableArtifact.target}${identity}\n`;
    elements.terminalOutput.textContent += `Build: ${formatElapsed(lastRunnableArtifact.buildMilliseconds)}\n`;
    const flashStarted = performance.now();
    try {
      const progress = (value) => {
        const verb = hotReload ? "Loading" : "Flashing";
        elements.compile.querySelector("span").textContent = `${verb} ${Math.round(value * 100)}%`;
        setMobileDeployStep("load", "active", `${verb} firmware, ${Math.round(value * 100)}%.`, value);
      };
      let report;
      if (hotReload) {
        if (!espSession) espSession = picoMonitor
          ? new PicoMonitorHotReloadSession(espPort, { progress })
          : new ESPJTAGHotReloadSession(espPort, { progress });
        report = await espSession.update(lastRunnableArtifact.data);
      } else {
        if (!espSession) espSession = new ESPWebSerial(espPort, {
          log: (message) => { elements.terminalOutput.textContent += `${message}\n`; },
          serial: appendSerialText,
          progress,
        });
		await espSession.flash(lastRunnableArtifact.data, deviceMachineTarget());
      }
      const flashMilliseconds = performance.now() - flashStarted;
      if (hotReload) {
        const change = report.unchanged ? "no changed words" : `${report.bytesWritten} bytes in ${report.patchCount} patches`;
        elements.terminalOutput.textContent += `${picoMonitor ? "Monitor" : "JTAG"} load: ${change} · ${formatElapsed(flashMilliseconds)} · Build + load: ${formatElapsed(lastRunnableArtifact.buildMilliseconds + flashMilliseconds)}\n`;
        elements.terminalOutput.textContent += "Running from SRAM. Press Flash after an edit to load the changes.\n";
        setMobileDeployStep("load", "done", `Firmware loaded over ${picoMonitor ? "the Pico monitor" : "JTAG"}.`);
        setMobileDeployStep("run", "done", "Running from SRAM. Hot reload is ready.");
        finishMobileDeployment(`${picoMonitor ? "Monitor" : "JTAG"} load complete`);
      } else {
        elements.terminalOutput.textContent += `Flash: ${formatElapsed(flashMilliseconds)} · Build + flash: ${formatElapsed(lastRunnableArtifact.buildMilliseconds + flashMilliseconds)}\n`;
        setMobileDeployStep("load", "done", "Firmware flashed.");
        setMobileDeployStep("run", "done", "Running. Serial is connected.");
        finishMobileDeployment("Flash complete");
      }
    } catch (error) {
      const flashMilliseconds = performance.now() - flashStarted;
      const failedAction = picoMonitor ? "Monitor load" : jtag ? "JTAG load" : "Flash";
      elements.terminalOutput.textContent += `${failedAction} failed after ${formatElapsed(flashMilliseconds)}: ${error.message || error}\n`;
      failMobileDeployment("load", `${failedAction} failed: ${error.message || error}`);
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
    type: "run", purpose: "app", id: ++requestID, name: lastRunnableArtifact.name,
    data, args, stdin: elements.runStdin.value,
  }, [data]);
}

async function showBrowserPreview(data) {
  const html = await packageBrowserArtifact(data);
  if (previewURL) URL.revokeObjectURL(previewURL);
  previewURL = URL.createObjectURL(new Blob([html], { type: "text/html" }));
  const iframe = document.createElement("iframe");
  iframe.title = "Renvo browser application preview";
  iframe.sandbox = "allow-scripts allow-downloads";
  iframe.src = previewURL;
  elements.preview.replaceChildren(iframe);
}

async function packageBrowserArtifact(data) {
  if (!targetCatalog?.browserPrefix || !targetCatalog?.browserSuffix) throw new Error("The browser preview host is unavailable.");
  if (!browserHostParts) browserHostParts = Promise.all([
    fetchAsset(new URL(targetCatalog.browserPrefix, catalogUrl)).then(checkTextResponse),
    fetchAsset(new URL(targetCatalog.browserSuffix, catalogUrl)).then(checkTextResponse),
  ]);
  const [prefix, suffix] = await browserHostParts;
  const bytes = new Uint8Array(data);
  let binary = "";
  for (let at = 0; at < bytes.length; at += 0x8000) binary += String.fromCharCode(...bytes.subarray(at, at + 0x8000));
  return prefix + btoa(binary) + suffix;
}

async function checkTextResponse(response) {
  if (!response.ok) throw new Error(`could not load browser preview host: HTTP ${response.status}`);
  return response.text();
}

function renderRunResult(result) {
  running = false;
  const output = `${result.stdout || ""}${result.stderr || ""}`;
  if (result.purpose === "test" || testRunning) {
    testRunning = false;
    elements.testsOutput.textContent += `${output}${output && !output.endsWith("\n") ? "\n" : ""}[tests exited ${result.exitCode} · ${result.elapsedMilliseconds.toFixed(1)} ms]\n`;
    elements.memoryStatus.textContent = `${(result.linearMemoryBytes / 1048576).toFixed(1)} MiB`;
    setCompilerStatus(result.exitCode === 0 ? "ready" : "error", result.exitCode === 0 ? "Tests passed" : "Tests failed");
    updateReadyState(); showPanel("tests"); return;
  }
  elements.terminalOutput.textContent += `${output}${output && !output.endsWith("\n") ? "\n" : ""}[process exited ${result.exitCode} · ${result.elapsedMilliseconds.toFixed(1)} ms]\n`;
  elements.memoryStatus.textContent = `${(result.linearMemoryBytes / 1048576).toFixed(1)} MiB`;
  updateReadyState();
}

async function ensureWorkspaceDependencies() {
  if (!standardCatalogPromise) return;
  const catalog = await standardCatalogPromise;
  if (activeBuildLanguage() === "c") {
    await Promise.all([loadCLibrary(catalog), loadStandardPackage("unsafe", catalog)]);
  }
  const imports = new Set();
  for (const model of models.values()) {
    if (model.uri.path.endsWith(".go")) for (const name of scanImports(model.getValue())) imports.add(name);
  }
  await Promise.all(Array.from(imports, (name) => loadStandardPackage(name, catalog)));
}

async function loadCLibrary(catalog) {
  if (!catalog.libc?.length) throw new Error("The browser bundle does not contain the C standard library.");
  if (cLibraryPromise) return cLibraryPromise;
  cLibraryPromise = (async () => {
    if (catalog.module && !models.has("go.mod")) stdlibFiles.set("go.mod", encoder.encode(catalog.module));
    const values = await Promise.all(catalog.libc.map(async (file) => {
      const path = file.split("/").map(encodeURIComponent).join("/");
      const response = await fetchAsset(new URL(`libc/${path}`, catalog.url));
      if (!response.ok) throw new Error(`could not load C library file ${file}: HTTP ${response.status}`);
      return [file, new Uint8Array(await response.arrayBuffer())];
    }));
    for (const [file, data] of values) stdlibFiles.set(`libc/${file}`, data);
    languageWorkspaceRevision++;
  })();
  try {
    await cLibraryPromise;
  } catch (error) {
    cLibraryPromise = undefined;
    throw error;
  }
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
      const response = await fetchAsset(new URL(file.split("/").map(encodeURIComponent).join("/"), base));
      if (!response.ok) throw new Error(`could not load library file ${key}/${file}: HTTP ${response.status}`);
      return [file, new Uint8Array(await response.arrayBuffer())];
    }));
    const prefix = platform ? item.root : `std/${name}`;
    for (const [file, data] of values) stdlibFiles.set(`${prefix}/${file}`, data);
    loadedStandardPackages.add(key);
    languageWorkspaceRevision++;
    await Promise.all((item.imports || []).map((dependency) => loadStandardPackage(dependency, catalog)));
  })();
  loadingStandardPackages.set(key, loading);
  try {
    await loading;
  } catch (error) {
    elements.languageStatus.textContent = `Could not open link: ${error.message || error}`;
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

function languageWorkspacePayload() {
  if (sentLanguageWorkspaceRevision === languageWorkspaceRevision) {
    return { revision: languageWorkspaceRevision, transfers: [] };
  }
  const payload = workspacePayload();
  sentLanguageWorkspaceRevision = languageWorkspaceRevision;
  return { ...payload, revision: languageWorkspaceRevision };
}

function installLanguageProviders() {
	const semanticLanguages = ["go", C_LANGUAGE_ID];
	monaco.languages.registerCompletionItemProvider(semanticLanguages, {
    triggerCharacters: [".", '"', "`", "/", "<"],
    provideCompletionItems: async (model, position) => {
      const includeContext = cIncludeContextAt(model, position);
      if (includeContext) {
        await ensureWorkspaceDependencies();
        const headers = new Set([...editableFiles].filter((name) => name.endsWith(".h")));
        for (const name of stdlibFiles.keys()) {
          if (name.startsWith("libc/include/") && name.endsWith(".h")) headers.add(name.slice("libc/include/".length));
        }
        return { suggestions: [...headers].filter((name) => name.startsWith(includeContext.prefix)).sort().map((name) => ({
          label: name, kind: monaco.languages.CompletionItemKind.File,
          detail: "C header", insertText: name, range: includeContext.range,
        })) };
      }
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
	monaco.languages.registerSignatureHelpProvider(semanticLanguages, {
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
	monaco.languages.registerDefinitionProvider(semanticLanguages, {
    provideDefinition: async (model, position) => {
      const records = await requestLanguage("definition", model, byteOffset(model, position));
      const record = records.find((item) => item[0] === "L");
      return record ? languageLocation(record) : undefined;
    },
  });
	monaco.languages.registerHoverProvider(semanticLanguages, {
    provideHover: async (model, position) => {
      const records = await requestLanguage("hover", model, byteOffset(model, position));
      const record = records.find((item) => item[0] === "H");
      if (!record) return undefined;
      const start = positionAtByteOffset(model, Number(record[3]));
      const end = positionAtByteOffset(model, Number(record[4]));
		const language = model.getLanguageId() === C_LANGUAGE_ID ? "c" : "go";
		const contents = [{ value: `\`\`\`${language}\n${record[1]}\n\`\`\`` }];
      if (record[2]) contents.push({ value: record[2] });
      return {
        contents,
        range: new monaco.Range(start.lineNumber, start.column, end.lineNumber, end.column),
      };
    },
  });
	monaco.languages.registerReferenceProvider(semanticLanguages, {
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
	monaco.languages.registerRenameProvider(semanticLanguages, {
    resolveRenameLocation(model, position) {
      const word = model.getWordAtPosition(position);
      return word ? { range: new monaco.Range(position.lineNumber, word.startColumn, position.lineNumber, word.endColumn), text: word.word } : { range: new monaco.Range(position.lineNumber, position.column, position.lineNumber, position.column), text: "", rejectReason: "Place the cursor on a symbol." };
    },
    provideRenameEdits: async (model, position, newName) => {
		if (!/^[A-Za-z_]\w*$/.test(newName)) return { edits: [], rejectReason: "Enter a valid identifier." };
      const offset = byteOffset(model, position);
      const [references, definitions] = await Promise.all([requestLanguage("references", model, offset), requestLanguage("definition", model, offset)]);
      const seen = new Set(); const edits = [];
      for (const record of [...references, ...definitions].filter((item) => item[0] === "L")) {
        const key = `${record[1]}:${record[2]}:${record[3]}`; if (seen.has(key)) continue; seen.add(key);
        const location = await languageLocation(record); if (!location) continue;
        const name = location.uri.path.replace(/^\//, ""); if (!isEditableFile(name)) continue;
        edits.push({ resource: location.uri, textEdit: { range: location.range, text: newName }, versionId: models.get(name)?.getVersionId() });
      }
      return edits.length ? { edits } : { edits: [], rejectReason: "No editable references were found." };
    },
  });
	monaco.languages.registerDocumentSymbolProvider(semanticLanguages, {
    provideDocumentSymbols(model) {
      return outlineItems(model).map((item) => ({
        name: item.name, detail: item.kind,
        kind: item.kind === "func" ? monaco.languages.SymbolKind.Function : item.kind === "type" ? monaco.languages.SymbolKind.Class : monaco.languages.SymbolKind.Variable,
        range: new monaco.Range(item.line, 1, item.line, model.getLineMaxColumn(item.line)),
        selectionRange: new monaco.Range(item.line, item.column, item.line, item.column + item.name.length),
      }));
    },
  });
  monaco.languages.registerDocumentFormattingEditProvider("go", {
    provideDocumentFormattingEdits: async (model) => [{ range: model.getFullModelRange(), text: await requestFormat(fileName(model), model.getValue()) }],
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
  const payload = languageWorkspacePayload();
  const result = new Promise((resolve) => languageRequests.set(id, resolve));
  worker.postMessage({
		type: mode, id, files: payload.files, workspaceRevision: payload.revision, target: selectedTarget.frontendTarget || selectedTarget.name,
		tags: selectedTarget.tags || [], file: fileName(model), offset,
		language: activeBuildLanguage(),
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

function scheduleBuildValidation(delay = 650) {
  clearTimeout(validationTimer);
  const generation = ++validationGeneration;
  validatedBuild = undefined;
  buildValidationState = "checking";
  buildValidationRevision = 0;
  if (compilerReady && monaco && selectedTarget && !building) setCompilerStatus("busy", "Checking build…");
  updateReadyState();
  validationTimer = setTimeout(() => runBuildValidation(generation), delay);
}

async function runBuildValidation(generation) {
  if (!compilerReady || !monaco || !selectedTarget || building || generation !== validationGeneration) return;
  if (pendingValidation) {
    validationTimer = setTimeout(() => runBuildValidation(generation), 100);
    return;
  }
  const revision = buildRevision;
  let target = selectedTarget;
  try {
    if (target.projectBackend && (target.backendStale || !target.backend)) {
      target = await prepareProjectBackend(target);
      if (generation !== validationGeneration) return;
    }
    let args = splitArguments(elements.command.value);
    args = controlledArguments(args, target);
    await ensureWorkspaceDependencies();
    if (generation !== validationGeneration || revision !== buildRevision || target.name !== selectedTarget?.name) return;
    const payload = workspacePayload();
    const backendPath = activeBuildLanguage() === "c" && target.cBackend ? target.cBackend : target.backend;
    const backend = backendPath ? new URL(backendPath, catalogUrl).href : "";
    const id = ++requestID;
    pendingValidation = { id, generation, revision, target, backend, args };
    worker.postMessage({
      type: "validate", id, args, files: payload.files,
      backend, backendTarget: target.backendTarget, backendFormat: target.backendFormat || "wasm",
      rtgDefinition: target.rtgDefinition ? new URL(target.rtgDefinition, catalogUrl).href : "",
      rtgDefinitionName: target.rtgDefinitionName || target.projectDefinition || "",
      rtgImports: (target.rtgImports || []).map((item) => ({ ...item, source: new URL(item.source, catalogUrl).href })),
    }, payload.transfers);
  } catch (error) {
    if (generation !== validationGeneration || revision !== buildRevision) return;
    applyBuildValidationFailure(revision, error.message || String(error));
  }
}

function receiveBuildValidation(result) {
  const pending = pendingValidation;
  if (!pending || pending.id !== result.id) return;
  pendingValidation = undefined;
  if (pending.generation !== validationGeneration || pending.revision !== buildRevision || pending.target.name !== selectedTarget?.name) {
    validationTimer = setTimeout(() => runBuildValidation(validationGeneration), 20);
    return;
  }
  const diagnosticText = result.exitCode === 0 ? "" : [result.stderr, result.stdout].filter(Boolean).join("\n");
  clearMarkers(false);
  renderProblems(parseDiagnostics(diagnosticText));
  buildValidationRevision = pending.revision;
  elements.memoryStatus.textContent = `${(result.linearMemoryBytes / 1048576).toFixed(1)} MiB`;
  if (result.exitCode === 0) {
    buildValidationState = "success";
    validatedBuild = { ...pending, result };
    if (pending.backend) backendReady.add(pending.backend);
    setCompilerStatus("ready", "Build ready");
  } else {
    buildValidationState = "failure";
    validatedBuild = undefined;
    setCompilerStatus("error", "Build has errors");
  }
  updateReadyState();
}

function applyBuildValidationFailure(revision, message) {
  buildValidationState = "failure";
  buildValidationRevision = revision;
  validatedBuild = undefined;
  clearMarkers(false);
  renderProblems(parseDiagnostics(message));
  setCompilerStatus("error", "Build has errors");
  updateReadyState();
}

async function runAnalysis(generation) {
	if (!compilerReady || !targetCatalog?.languageService || !selectedTarget || generation !== languageGeneration) return;
	elements.languageStatus.textContent = activeBuildLanguage() === "c" ? "Checking C…" : "Checking…";
  try {
    await ensureWorkspaceDependencies();
    if (generation !== languageGeneration) return;
    const payload = languageWorkspacePayload();
    const id = ++requestID;
    latestAnalysisRequestID = id;
    worker.postMessage({
		type: "analyze", id, files: payload.files, workspaceRevision: payload.revision, target: selectedTarget.frontendTarget || selectedTarget.name,
		tags: selectedTarget.tags || [], file: activeFile, offset: 0,
		language: activeBuildLanguage(),
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
	if (buildValidationRevision === buildRevision) return;
  const problems = records.filter((record) => record[0] === "D").map((record) => ({
    file: cleanPath(record[1]), start: Number(record[2]), end: Number(record[3]),
    line: Number(record[4]), column: Number(record[5]), code: record[6], message: record[7],
  }));
  if (error) problems.push({ file: "", line: 0, column: 0, message: error.trim() });
  clearMarkers(false);
  renderProblems(problems);
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

function cIncludeContextAt(model, position) {
  if (model.getLanguageId() !== C_LANGUAGE_ID) return null;
  const offset = model.getOffsetAt(position);
  const line = model.getValue().slice(0, offset).split("\n").pop();
  const match = /^\s*#\s*include\s*[<"]([^>"]*)$/.exec(line);
  if (!match) return null;
  return {
    prefix: match[1],
    range: new monaco.Range(position.lineNumber, position.column - match[1].length, position.lineNumber, position.column),
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
  activeHelp = "";
  activeFile = name;
  if (!openFiles.includes(name)) openFiles.push(name);
  elements.helpView.hidden = true;
  elements.editorHost.hidden = false;
  elements.copyHelpPage.hidden = true;
  document.querySelector("#search-project").hidden = false;
  elements.formatFile.hidden = false;
  renderEditorTabs();
  editor.setModel(model);
  const editable = isEditableFile(name);
  if (editable) lastWorkspaceFile = name;
  editor.updateOptions({ readOnly: !editable, readOnlyMessage: { value: "Copy this library source into the project to edit it." } });
  elements.copyToPlayground.hidden = editable;
  elements.copyToPlayground.textContent = `Copy ${name.split("/").pop()} to project`;
  elements.copyToPlayground.title = `Copy ${name} into the editable project`;
  elements.formatFile.disabled = !editable || !name.endsWith(".go");
  elements.formatFile.title = elements.formatFile.disabled ? "Formatting is currently available for editable Go files" : "Format document with gofmt (Shift+Alt+F)";
  elements.languageMode.textContent = name.endsWith(".go") ? "Go" : /\.[ch]$/.test(name) ? "C" : name.endsWith(".rtg") ? "RTG" : "Plain Text";
  elements.useBackend.hidden = !editable || !name.endsWith(".rtg");
  document.querySelectorAll(".file").forEach((item) => item.classList.toggle("active", item.dataset.file === name));
  document.querySelectorAll(".stdlib-file").forEach((item) => item.classList.toggle("active", item.dataset.file === name));
  renderOutline(model);
  updateMobileHeader();
  syncCodeDeepLink(name, editable);
  requestAnimationFrame(() => editor?.layout());
  if (!isPhoneWorkspace()) editor.focus();
}

function isEditableFile(name) {
  return editableFiles.has(name);
}

function installWorkspaceFileButton(button) {
  if (button.dataset.installed) return;
  button.dataset.installed = "true";
  button.addEventListener("click", () => {
    openFile(button.dataset.file);
    setBuildPackage(".");
    if (!isPhoneWorkspace() && matchMedia("(max-width: 820px)").matches) {
      elements.ide.classList.remove("sidebar-open");
      elements.toggleSidebar.setAttribute("aria-expanded", "false");
    }
    if (isPhoneWorkspace()) showMobileView("editor");
  });
}

async function copyActiveFileToPlayground() {
  const source = models.get(activeFile);
  if (!source || isEditableFile(activeFile)) return;
  const destination = activeFile.split("/").pop();
  let target = models.get(destination);
  if (target && !await requestConfirmation({ title: "Replace project file?", message: `${destination} already exists in the project.`, accept: "Replace" })) return;
  if (!target) target = createProjectModel(destination, "");
  target.pushEditOperations([], [{ range: target.getFullModelRange(), text: source.getValue() }], () => null);
  editableBaselines.set(destination, "");
  renderWorkspaceFiles();
  openFile(destination);
  setBuildPackage(".");
  saveFiles();
  scheduleAnalysis(20);
  if (isPhoneWorkspace()) showMobileView("editor");
}

function handleModelChange(name, model) {
  document.querySelector(`.file[data-file="${name}"]`)?.classList.toggle("modified", model.getValue() !== editableBaselines.get(name));
  saveFiles();
  markBuildStale();
  languageWorkspaceRevision++;
  if (name.endsWith(".rtg")) {
    for (const target of targetCatalog?.targets || []) {
      if (target.projectBackend) target.backendStale = true;
    }
  }
	if (name.endsWith(".go") || name.endsWith(".c") || name.endsWith(".h") || name === "go.mod") scheduleAnalysis();
  if (name === activeFile) renderOutline(model);
}

function createProjectModel(name, value = "") {
  name = normalizeProjectPath(name);
  if (models.has(name)) return models.get(name);
  editableFiles.add(name);
  if (!editableBaselines.has(name)) editableBaselines.set(name, value);
  const model = monaco.editor.createModel(value, languageForFile(name), monaco.Uri.parse(`file:///${name}`));
  model.onDidChangeContent(() => handleModelChange(name, model));
  models.set(name, model);
  languageWorkspaceRevision++;
  return model;
}

function languageForFile(name) {
	const base = name.split("/").pop();
	if (base === "Makefile" || base === "makefile" || name.endsWith(".mk")) return MAKEFILE_LANGUAGE_ID;
  if (name.endsWith(".go")) return "go";
  if (name.endsWith(".c") || name.endsWith(".h")) return C_LANGUAGE_ID;
  if (name.endsWith(".rtg") || name.endsWith(".rtgasm")) return RTG_LANGUAGE_ID;
  if (name.endsWith(".md")) return "markdown";
  if (name.endsWith(".json")) return "json";
  return "plaintext";
}

function fileIcon(name) {
	const base = name.split("/").pop();
	if (base === "Makefile" || base === "makefile" || name.endsWith(".mk")) return ["MK", "make-icon"];
  if (name.endsWith(".go")) return ["Go", "go-icon"];
  if (name.endsWith(".c") || name.endsWith(".h")) return ["C", "c-icon"];
  if (name.endsWith(".rtg") || name.endsWith(".rtgasm")) return ["RTG", "rtg-icon"];
  if (name.endsWith(".md")) return ["MD", "mod-icon"];
  return [name === "go.mod" ? "M" : "·", "mod-icon"];
}

function renderWorkspaceFiles() {
  const entries = [...editableFiles].sort((left, right) => left.localeCompare(right));
  elements.projectFileCount.textContent = String(entries.length);
  elements.fileTree.replaceChildren(...entries.map((name) => {
    const row = document.createElement("div");
    row.className = "file-row"; row.dataset.file = name; row.setAttribute("role", "none");
    const button = document.createElement("button");
    button.type = "button"; button.className = "file"; button.dataset.file = name; button.setAttribute("role", "treeitem");
    const [text, className] = fileIcon(name);
    const icon = document.createElement("span"); icon.className = className; icon.textContent = text;
    const label = document.createElement("span"); label.textContent = name;
    const dirty = document.createElement("span"); dirty.className = "dirty"; dirty.setAttribute("aria-label", "Modified");
    button.append(icon, label, dirty);
    button.classList.toggle("active", name === activeFile);
    button.classList.toggle("modified", models.get(name)?.getValue() !== editableBaselines.get(name));
    installWorkspaceFileButton(button);
    const actions = document.createElement("button");
    actions.type = "button"; actions.className = "file-more"; actions.textContent = "•••";
    actions.setAttribute("aria-label", `Actions for ${name}`); actions.title = `Actions for ${name}`;
    actions.addEventListener("click", (event) => {
      event.stopPropagation();
      const bounds = actions.getBoundingClientRect();
      openFileActionMenu(name, bounds.right - 190, bounds.bottom + 2);
    });
    row.append(button, actions);
    return row;
  }));
}

function renderEditorTabs() {
  elements.openEditorTabs.replaceChildren(...openFiles.filter((name) => models.has(name) || isHelpTab(name)).map((name) => {
    const tab = document.createElement("button");
    tab.type = "button"; tab.className = "editor-tab"; tab.dataset.file = name; tab.setAttribute("role", "tab");
    const active = isHelpTab(name) ? name === activeHelp : !activeHelp && name === activeFile;
    tab.setAttribute("aria-selected", String(active)); tab.classList.toggle("active", active); tab.title = isHelpTab(name) ? helpImportPath(name) : name;
    const [text, className] = isHelpTab(name) ? ["?", "help-icon"] : fileIcon(name);
    const icon = document.createElement("span"); icon.className = className; icon.textContent = text;
    const label = document.createElement("span"); label.textContent = isHelpTab(name) ? helpImportPath(name).split("/").pop() : name.split("/").pop();
    const close = document.createElement("span"); close.className = "tab-close"; close.textContent = "×"; close.title = `Close ${name}`;
    close.addEventListener("click", (event) => { event.stopPropagation(); closeEditorTab(name); });
    tab.append(icon, label, close);
    tab.addEventListener("click", () => isHelpTab(name) ? openHelpPage(helpImportPath(name)) : openFile(name));
    tab.addEventListener("auxclick", (event) => { if (event.button === 1) closeEditorTab(name); });
    return tab;
  }));
}

function closeEditorTab(name) {
  const at = openFiles.indexOf(name);
  if (at < 0) return;
  openFiles.splice(at, 1);
  if (activeHelp === name || !activeHelp && activeFile === name) {
    const next = openFiles[Math.min(at, openFiles.length - 1)] || [...editableFiles][0];
    if (isHelpTab(next)) openHelpPage(helpImportPath(next));
    else if (next) openFile(next);
  }
  renderEditorTabs();
}

function createWorkspaceFile() {
  elements.newFileKind.value = "go";
  elements.newFilePath.value = "new.go";
  elements.newFileError.textContent = "";
  updateNewFileHelp();
  elements.newFileDialog.showModal();
  queueMicrotask(() => { elements.newFilePath.focus(); elements.newFilePath.select(); });
}

function createNewWorkspaceFile(event) {
  if (event.submitter?.value !== "accept") return;
  event.preventDefault();
  try {
    const name = normalizeProjectPath(elements.newFilePath.value);
    if (models.has(name)) throw new Error(`${name} already exists.`);
    const source = starterSourceForFile(name, elements.newFileKind.value);
    createProjectModel(name, source);
    editableBaselines.set(name, "");
    setBuildPackage(".");
    renderWorkspaceFiles(); openFile(name); saveFiles(); markBuildStale();
    elements.languageStatus.textContent = `Created ${name} · building project`;
    elements.newFileDialog.close("accept");
    if (isPhoneWorkspace()) showMobileView("editor");
  } catch (error) {
    elements.newFileError.textContent = error.message || String(error);
    elements.newFilePath.focus();
  }
}

function updateNewFileSuggestion() {
  const extension = ({ go: ".go", c: ".c", h: ".h", rtg: ".rtg" })[elements.newFileKind.value];
  if (extension) elements.newFilePath.value = elements.newFilePath.value.replace(/(?:\.[^./]+)?$/, extension);
  updateNewFileHelp();
}

function updateNewFileKindFromPath() {
  const name = elements.newFilePath.value.toLowerCase();
  if (name.endsWith(".go")) elements.newFileKind.value = "go";
  else if (name.endsWith(".c")) elements.newFileKind.value = "c";
  else if (name.endsWith(".h")) elements.newFileKind.value = "h";
  else if (name.endsWith(".rtg")) elements.newFileKind.value = "rtg";
  else elements.newFileKind.value = "empty";
  updateNewFileHelp();
}

function updateNewFileHelp() {
  elements.newFileHelp.textContent = ({
    go: "Starts with a package declaration and builds with this project.",
    c: "Creates a C source file with a short Go interop note and builds it with this project.",
    h: "Creates a guarded C header for declarations shared by C files.",
    rtg: "Creates a target definition that can be selected as a project backend.",
    empty: "Creates an empty file and builds supported source extensions with this project.",
  })[elements.newFileKind.value];
}

function starterSourceForFile(name, kind) {
  if (name.endsWith(".go") || kind === "go") return "package main\n";
  if (name.endsWith(".c") || kind === "c") return "/* Package-level Go functions can be declared here with extern. */\n";
  if (name.endsWith(".h") || kind === "h") return "#pragma once\n";
  if (name.endsWith(".rtg") || kind === "rtg") return "definition 1\nunit custom\nimplements direct_emitter_v1\n\n# Define or import an architecture, ABI, runtime, format, and target.\n";
  return "";
}

function handleWorkspaceFileMenu(event) {
  const button = event.target.closest(".file-row") || event.target.closest(".file");
  if (!button) return;
  event.preventDefault();
  openFileActionMenu(button.dataset.file, event.clientX, event.clientY);
}

function openFileActionMenu(name, left, top) {
  fileMenuTarget = name;
  elements.fileActionMenu.setAttribute("aria-label", `Actions for ${fileMenuTarget}`);
  elements.fileActionMenu.style.left = `${Math.max(4, Math.min(left, innerWidth - 200))}px`;
  elements.fileActionMenu.style.top = `${Math.max(4, Math.min(top, innerHeight - 90))}px`;
  elements.fileActionMenu.hidden = false;
  const backend = elements.fileActionMenu.querySelector('[data-file-action="backend"]');
  if (backend) backend.hidden = !name.endsWith(".rtg");
  elements.fileActionMenu.querySelector("button")?.focus();
}

async function handleFileAction(event) {
  const button = event.target.closest("[data-file-action]");
  if (!button || !fileMenuTarget) return;
  const name = fileMenuTarget;
  closeFileActionMenu();
  if (button.dataset.fileAction === "backend") await useProjectBackend(name);
  else if (button.dataset.fileAction === "rename") await renameWorkspaceFile(name);
  else if (button.dataset.fileAction === "delete") await deleteWorkspaceFile(name);
}

function closeFileActionMenu() {
  elements.fileActionMenu.hidden = true;
  fileMenuTarget = "";
}

async function renameWorkspaceFile(oldName) {
  const name = await requestText({
    title: "Rename project file", label: "Path", value: oldName, accept: "Rename",
    validate: (value) => {
      const normalized = normalizeProjectPath(value);
      if (normalized !== oldName && models.has(normalized)) throw new Error(`${normalized} already exists.`);
      return normalized;
    },
  });
  if (!name || name === oldName) return;
  const old = models.get(oldName);
  const value = old.getValue();
  const wasActive = activeFile === oldName;
  editableFiles.delete(oldName); models.delete(oldName); old.dispose();
  editableBaselines.delete(oldName);
  createProjectModel(name, value); editableBaselines.set(name, value);
  const tabAt = openFiles.indexOf(oldName); if (tabAt >= 0) openFiles[tabAt] = name;
  if (wasActive) activeFile = name;
  if (projectBackendRoots.delete(oldName)) projectBackendRoots.add(name);
  for (const target of targetCatalog?.targets || []) {
    if (target.projectDefinition === oldName) target.projectDefinition = name;
  }
  renderWorkspaceFiles(); renderEditorTabs(); if (wasActive) openFile(name);
  saveFiles(); markBuildStale(); scheduleAnalysis(20);
}

async function deleteWorkspaceFile(name) {
  if (!await requestConfirmation({ title: "Delete file?", message: `${name} will be removed from this browser project.`, accept: "Delete" })) return;
  const model = models.get(name);
  projectBackendRoots.delete(name);
  if (targetCatalog) {
    targetCatalog.targets = targetCatalog.targets.filter((target) => target.projectDefinition !== name);
    if (selectedTarget?.projectDefinition === name) {
      selectedTarget = undefined;
      restoredTargetName = "";
    }
    configureTargets(targetCatalog.targets);
  }
  editableFiles.delete(name); models.delete(name); editableBaselines.delete(name); model?.dispose();
  const at = openFiles.indexOf(name); if (at >= 0) openFiles.splice(at, 1);
  if (!editableFiles.size) createProjectModel("main.go", initialFiles["main.go"]);
  if (activeFile === name) activeFile = [...editableFiles][0];
  renderWorkspaceFiles(); renderEditorTabs(); openFile(activeFile); saveFiles(); markBuildStale(); scheduleAnalysis(20);
}

async function importProjectFiles(list, directory = false) {
  if (!list?.length) return;
  try {
    const imported = {};
    for (const file of list) {
      if (file.name.toLowerCase().endsWith(".zip")) Object.assign(imported, decodeProjectZip(await file.arrayBuffer()));
      else imported[normalizeProjectPath(directory && file.webkitRelativePath ? file.webkitRelativePath.split("/").slice(1).join("/") : file.name)] = await file.text();
    }
    for (const [name, source] of Object.entries(imported)) {
      if (models.has(name) && !await requestConfirmation({ title: "Replace file?", message: `${name} already exists in this project.`, accept: "Replace" })) continue;
      if (models.has(name)) models.get(name).setValue(source);
      else createProjectModel(name, source);
      editableBaselines.set(name, source);
    }
    setBuildPackage(".");
    renderWorkspaceFiles(); openFile(Object.keys(imported)[0]); saveFiles(); markBuildStale(); scheduleAnalysis(20);
  } catch (error) { showProjectError(error); }
  elements.projectFileInput.value = ""; elements.projectDirectoryInput.value = "";
}

function projectFiles() {
  return Object.fromEntries([...editableFiles].sort().map((name) => [name, models.get(name)?.getValue() ?? fileValues[name] ?? ""]));
}

function exportProject() {
  const data = encodeProjectZip(projectFiles());
  const url = createDownloadURL(data, "application/zip");
  const link = document.createElement("a"); link.href = url; link.download = `${safeProjectName(projectName)}.zip`; link.click();
  setTimeout(() => releaseDownloadURL(url), 0);
}

function toggleProjectActionMenu() {
  const open = elements.projectActionMenu.hidden;
  elements.projectActionMenu.hidden = !open;
  elements.projectMenuButton.setAttribute("aria-expanded", String(open));
  if (open) elements.projectActionMenu.querySelector("button")?.focus();
}

function closeProjectActionMenu() {
  elements.projectActionMenu.hidden = true;
  elements.projectMenuButton.setAttribute("aria-expanded", "false");
}

async function handleProjectAction(event) {
  const button = event.target.closest("[data-project-action]");
  if (!button) return;
  const action = button.dataset.projectAction;
  closeProjectActionMenu();
  try {
    if (action === "new") openNewProjectDialog();
    else if (action === "import") elements.projectFileInput.click();
    else if (action === "rename") {
      const name = await requestText({ title: "Rename project", label: "Project name", value: projectName, accept: "Rename", validate: (value) => value.trim() || "playground" });
      if (name) { projectName = name; syncProjectName(); saveFiles(); }
    } else if (action === "export") exportProject();
    else if (action === "directory") elements.projectDirectoryInput.click();
    else if (action === "share") await shareProject();
    else if (action === "snapshots") await openSnapshotDialog();
    else if (action === "reset" && await requestConfirmation({ title: "Reset project?", message: "All current project files will be replaced with the starter project. Save a snapshot first if you may need them again.", accept: "Reset project" })) resetProject();
  } catch (error) { showProjectError(error); }
}

function openNewProjectDialog() {
  const selected = elements.newProjectForm.querySelector(`input[name="project-kind"][value="${projectLanguage}"]`);
  if (selected) selected.checked = true;
  elements.newProjectDialog.showModal();
  queueMicrotask(() => elements.newProjectForm.querySelector("input[name=\"project-kind\"]:checked")?.focus());
}

function starterCommand() {
  return `-s -o ${selectedTarget?.output || "app.wasm"} .`;
}

function createNewProject(event) {
  if (event.submitter?.value !== "accept") return;
  event.preventDefault();
  const language = new FormData(elements.newProjectForm).get("project-kind") === "c" ? "c" : "go";
  const c = language === "c";
  elements.newProjectDialog.close("accept");
  replaceProject({
    name: c ? "c-playground" : "playground",
    language,
    files: c ? initialCFiles : initialFiles,
    activeFile: c ? "main.c" : "main.go",
    openFiles: [c ? "main.c" : "main.go"],
    command: starterCommand(),
  });
  elements.languageStatus.textContent = c ? "C project created" : "Go project created";
  if (isPhoneWorkspace()) showMobileView("editor");
}

async function openSnapshotDialog() {
  await renderSnapshots();
  elements.snapshotName.value = "Snapshot";
  elements.snapshotDialog.showModal();
  queueMicrotask(() => { elements.snapshotName.focus(); elements.snapshotName.select(); });
}

async function renderSnapshots() {
  const snapshots = await loadProjectSnapshots();
  if (!snapshots.length) {
    const empty = document.createElement("div"); empty.className = "snapshot-empty"; empty.textContent = "No snapshots yet.";
    elements.snapshotList.replaceChildren(empty);
    return;
  }
  elements.snapshotList.replaceChildren(...snapshots.map((snapshot) => {
    const row = document.createElement("div"); row.className = "snapshot-item"; row.dataset.snapshot = snapshot.id;
    const details = document.createElement("div");
    const name = document.createElement("strong"); name.textContent = snapshot.label || "Snapshot";
    const date = document.createElement("small"); date.textContent = `${new Date(snapshot.savedAt).toLocaleString()} · ${Object.keys(snapshot.files || {}).length} files`;
    details.append(name, date);
    const restore = document.createElement("button"); restore.type = "button"; restore.dataset.snapshotAction = "restore"; restore.textContent = "Restore";
    const remove = document.createElement("button"); remove.type = "button"; remove.dataset.snapshotAction = "delete"; remove.className = "danger"; remove.textContent = "Delete";
    row.append(details, restore, remove); return row;
  }));
}

async function saveSnapshotFromDialog(event) {
  event.preventDefault();
  const label = elements.snapshotName.value.trim() || "Snapshot";
  await saveProjectSnapshot(currentProject(), label);
  elements.snapshotName.value = "Snapshot";
  elements.languageStatus.textContent = "Project snapshot saved";
  await renderSnapshots();
}

async function handleSnapshotAction(event) {
  const button = event.target.closest("[data-snapshot-action]");
  const row = button?.closest("[data-snapshot]");
  if (!button || !row) return;
  const snapshots = await loadProjectSnapshots();
  const snapshot = snapshots.find((item) => item.id === row.dataset.snapshot);
  if (!snapshot) return;
  if (button.dataset.snapshotAction === "restore") {
    const accepted = await requestConfirmation({ title: "Restore snapshot?", message: `Replace the current project with "${snapshot.label || "Snapshot"}"?`, accept: "Restore" });
    if (!accepted) return;
    elements.snapshotDialog.close(); replaceProject(snapshot); elements.languageStatus.textContent = "Project snapshot restored";
  } else if (button.dataset.snapshotAction === "delete") {
    const accepted = await requestConfirmation({ title: "Delete snapshot?", message: `Delete "${snapshot.label || "Snapshot"}" from this browser?`, accept: "Delete" });
    if (!accepted) return;
    await deleteProjectSnapshot(snapshot.id); await renderSnapshots();
  }
}

async function shareProject() {
  const encoded = encodeSharedProject(projectFiles());
  const url = new URL(location.href); url.hash = new URLSearchParams({ project: encoded }).toString();
  if (url.href.length > 100000) throw new Error("This project is too large for a share link; export a ZIP instead.");
  await navigator.clipboard.writeText(url.href);
  elements.languageStatus.textContent = "Share link copied";
}

function resetProject() {
  const c = projectLanguage === "c";
  replaceProject({ name: c ? "c-playground" : "playground", language: projectLanguage, files: c ? initialCFiles : initialFiles, activeFile: c ? "main.c" : "main.go", openFiles: [c ? "main.c" : "main.go"], command: starterCommand() });
}

function replaceProject(project) {
  for (const model of models.values()) model.dispose();
  editableFiles.clear(); editableBaselines.clear(); models.clear(); openFiles.length = 0;
  projectBackendRoots.clear();
  for (const name of project.backendRoots || []) if (typeof name === "string") projectBackendRoots.add(name);
  projectLanguage = project.language === "c" || !project.language && inferProjectLanguage(project.files) === "c" ? "c" : "go";
  externalBuildLanguage = "go";
  const fallbackFiles = projectLanguage === "c" ? initialCFiles : initialFiles;
  const files = project.files && Object.keys(project.files).length ? project.files : fallbackFiles;
  for (const [name, source] of Object.entries(files)) { createProjectModel(name, source); editableBaselines.set(name, source); }
  activeFile = Object.hasOwn(files, project.activeFile) ? project.activeFile : Object.keys(files)[0];
  lastWorkspaceFile = activeFile;
  for (const name of project.openFiles || []) if (Object.hasOwn(files, name) && !openFiles.includes(name)) openFiles.push(name);
  if (!openFiles.includes(activeFile)) openFiles.push(activeFile);
  projectName = project.name || "playground"; syncProjectName();
  if (project.command) elements.command.value = project.command;
  elements.arenaSize.value = project.arenaSize ? String(project.arenaSize) : "";
  restoredTargetName = project.target || selectedTarget?.name || "";
  if (targetCatalog?.targets.some((target) => target.name === restoredTargetName)) selectTarget(restoredTargetName, false);
  syncBuildRootFromCommand();
  renderWorkspaceFiles(); renderEditorTabs(); openFile(activeFile); saveFiles(); markBuildStale(); scheduleAnalysis(20);
}

function syncProjectName() {
  elements.projectName.textContent = projectName;
  elements.sidebarProjectName.textContent = projectName;
  document.title = `${projectName} | Renvo`;
  updateMobileHeader();
}

function safeProjectName(name) { return (name || "renvo-project").replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-|-$/g, "") || "renvo-project"; }

function inferProjectLanguage(files) {
  const names = Object.keys(files || {});
  return names.some((name) => name.endsWith(".c")) && !names.some((name) => name.endsWith(".go")) ? "c" : "go";
}

function showProjectError(error) {
  elements.output.textContent = `${error.message || error}\n`; showPanel("output");
}

function requestText({ title, label, value = "", accept = "Save", validate }) {
  if (textDialogResolve) throw new Error("Another editor dialog is already open.");
  elements.textDialogTitle.textContent = title;
  elements.textDialogLabel.textContent = label;
  elements.textDialogInput.value = value;
  elements.textDialogAccept.textContent = accept;
  elements.textDialogError.textContent = "";
  elements.textDialog.returnValue = "cancel";
  const result = new Promise((resolve) => { textDialogResolve = { resolve, validate, value: undefined }; });
  elements.textDialog.showModal();
  queueMicrotask(() => { elements.textDialogInput.focus(); elements.textDialogInput.select(); });
  return result;
}

function validateTextDialog(event) {
  if (event.submitter?.value !== "accept") return;
  event.preventDefault();
  try {
    const value = textDialogResolve?.validate ? textDialogResolve.validate(elements.textDialogInput.value) : elements.textDialogInput.value;
    textDialogResolve.value = value;
    elements.textDialog.close("accept");
  } catch (error) {
    elements.textDialogError.textContent = error.message || String(error);
    elements.textDialogInput.focus();
  }
}

function finishTextDialog() {
  const pending = textDialogResolve;
  textDialogResolve = undefined;
  pending?.resolve(elements.textDialog.returnValue === "accept" ? pending.value : undefined);
}

function requestConfirmation({ title, message, accept = "Continue", danger = true }) {
  if (confirmDialogResolve) throw new Error("Another confirmation is already open.");
  elements.confirmDialogTitle.textContent = title;
  elements.confirmDialogMessage.textContent = message;
  elements.confirmDialogAccept.textContent = accept;
  elements.confirmDialogAccept.className = danger ? "dialog-danger" : "dialog-primary";
  elements.confirmDialog.returnValue = "cancel";
  const result = new Promise((resolve) => { confirmDialogResolve = resolve; });
  elements.confirmDialog.showModal();
  queueMicrotask(() => elements.confirmDialogAccept.focus());
  return result;
}

function finishConfirmDialog() {
  const resolve = confirmDialogResolve;
  confirmDialogResolve = undefined;
  resolve?.(elements.confirmDialog.returnValue === "accept");
}

function deviceAccessSupport() {
  const profile = currentDeviceProfile();
  const secure = globalThis.isSecureContext !== false;
  const choices = chooseESPTransportAvailability({
    profile,
    webSerial: secure && Boolean(navigator.serial),
    webUSB: secure && Boolean(navigator.usb),
  });
  return { profile, choices, secure };
}

function renderDevicePermission() {
  const { profile, choices, secure } = deviceAccessSupport();
  const pico = selectedTarget?.device === "rp2";
  elements.picoMonitorInstaller.hidden = !pico;
  const availableChoices = pico ? { ...choices, webserial: false } : choices;
  const statuses = [
    [elements.deviceWebUSBStatus, "webusb", "WebUSB"],
    [elements.deviceWebSerialStatus, "webserial", "WebSerial"],
  ];
  for (const [element, key] of statuses) {
    const available = availableChoices[key];
    element.classList.toggle("available", available);
    element.classList.toggle("unavailable", !available);
    element.querySelector("small").textContent = available ? "Supported" : "Not supported";
  }
  const available = availableChoices.webusb || availableChoices.webserial;
  if (profile.ios && !available) {
    elements.devicePermissionIntro.textContent = "Browsers on iPhone and iPad cannot flash USB boards.";
    elements.devicePermissionNote.textContent = "You can edit and build here. To flash, open the project in Chrome or Edge on a computer, or use a supported Android browser.";
  } else if (!secure) {
    elements.devicePermissionIntro.textContent = "This page cannot ask for USB access.";
    elements.devicePermissionNote.textContent = "Open Renvo over HTTPS or localhost, then try again.";
  } else if (!available) {
    elements.devicePermissionIntro.textContent = "This browser cannot flash USB boards.";
    elements.devicePermissionNote.textContent = "Use Chrome or Edge on a computer, or a supported Android browser. You can still edit and build here.";
  } else {
    elements.devicePermissionIntro.textContent = "Your browser will ask which device Renvo may use.";
    elements.devicePermissionNote.textContent = pico
      ? "Choose the Pico running the Renvo monitor. No debug probe is used; disconnect it at any time to end access."
      : "Disconnect the board at any time to end access.";
  }
  elements.devicePermissionAccept.disabled = !available;
  elements.devicePermissionAccept.textContent = available ? pico ? "Monitor installed — select board" : "Select device" : "USB not supported";
  return available;
}

function requestDevicePermission() {
  const available = renderDevicePermission();
  const storageKey = selectedTarget?.device === "rp2" ? "renvo.devicePermissionExplained.rp2.v1" : "renvo.devicePermissionExplained.v1";
  if (available && localStorage.getItem(storageKey) === "yes") return Promise.resolve(true);
  if (devicePermissionResolve) return Promise.resolve(false);
  elements.devicePermissionDialog.returnValue = "cancel";
  const result = new Promise((resolve) => { devicePermissionResolve = resolve; });
  elements.devicePermissionDialog.showModal();
  queueMicrotask(() => (available ? elements.devicePermissionAccept : document.querySelector("#device-permission-cancel")).focus());
  return result;
}

function acceptDevicePermission() {
  if (elements.devicePermissionAccept.disabled) return;
  const storageKey = selectedTarget?.device === "rp2" ? "renvo.devicePermissionExplained.rp2.v1" : "renvo.devicePermissionExplained.v1";
  localStorage.setItem(storageKey, "yes");
  elements.devicePermissionDialog.close("accept");
}

function finishDevicePermission() {
  const resolve = devicePermissionResolve;
  devicePermissionResolve = undefined;
  resolve?.(elements.devicePermissionDialog.returnValue === "accept");
}

function toggleAdvancedBuild() {
  const panel = document.querySelector("#advanced-build");
  panel.hidden = !panel.hidden;
  document.querySelector("#advanced-heading").classList.toggle("collapsed", panel.hidden);
  document.querySelector("#advanced-heading .chevron").textContent = panel.hidden ? "›" : "⌄";
}

function toggleOutline() {
  elements.outlineTree.hidden = !elements.outlineTree.hidden;
  document.querySelector("#outline-heading").classList.toggle("collapsed", elements.outlineTree.hidden);
  document.querySelector("#outline-heading .chevron").textContent = elements.outlineTree.hidden ? "›" : "⌄";
}

function toggleSidebarExamples() {
  elements.sidebarExamples.hidden = !elements.sidebarExamples.hidden;
  document.querySelector("#examples-heading").classList.toggle("collapsed", elements.sidebarExamples.hidden);
  document.querySelector("#examples-heading .chevron").textContent = elements.sidebarExamples.hidden ? "›" : "⌄";
  if (!elements.sidebarExamples.hidden && standardCatalog) renderSidebarExamples();
}

function toggleHelp() {
  elements.helpTree.hidden = !elements.helpTree.hidden;
  document.querySelector("#help-heading").classList.toggle("collapsed", elements.helpTree.hidden);
  document.querySelector("#help-heading .chevron").textContent = elements.helpTree.hidden ? "›" : "⌄";
  if (!elements.helpTree.hidden && standardCatalog) {
    renderHelpCatalog();
    queueMicrotask(() => elements.helpTree.querySelector("input")?.focus());
  }
}

function toggleLibrary() {
  elements.stdlibTree.hidden = !elements.stdlibTree.hidden;
  document.querySelector("#library-heading").classList.toggle("collapsed", elements.stdlibTree.hidden);
  document.querySelector("#library-heading .chevron").textContent = elements.stdlibTree.hidden ? "›" : "⌄";
}

function outlineItems(model) {
  if (!model) return [];
  const items = [];
  const lines = model.getLinesContent();
  for (let index = 0; index < lines.length; index++) {
    const line = lines[index];
    let match;
    if (fileName(model).endsWith(".go")) {
      match = /^\s*func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)/.exec(line) ||
        /^\s*(type|var|const)\s+([A-Za-z_]\w*)/.exec(line);
      if (match) items.push({ name: match[2] || match[1], kind: match[2] ? match[1] : "func", line: index + 1, column: line.indexOf(match[2] || match[1]) + 1 });
    } else if (/\.[ch]$/.test(fileName(model))) {
      match = /^\s*(?:[A-Za-z_]\w*[\s*]+)+([A-Za-z_]\w*)\s*\([^;]*\)\s*\{?/.exec(line) || /^\s*(struct|enum|union)\s+([A-Za-z_]\w*)/.exec(line);
      if (match) items.push({ name: match[2] || match[1], kind: match[2] ? match[1] : "func", line: index + 1, column: line.indexOf(match[2] || match[1]) + 1 });
    }
  }
  return items;
}

function renderOutline(model) {
  const items = outlineItems(model);
  elements.outlineCount.textContent = items.length ? String(items.length) : "";
  if (!items.length) {
    elements.outlineTree.innerHTML = '<div class="tree-loading">No symbols</div>';
    return;
  }
  elements.outlineTree.replaceChildren(...items.map((item) => {
    const button = document.createElement("button");
    button.type = "button"; button.className = "outline-item"; button.setAttribute("role", "treeitem");
    const kind = document.createElement("span"); kind.className = "outline-kind"; kind.textContent = item.kind;
    const name = document.createElement("span"); name.className = "outline-name"; name.textContent = item.name;
    button.append(kind, name);
    button.addEventListener("click", () => {
      const position = { lineNumber: item.line, column: item.column }; editor.setPosition(position); editor.revealPositionInCenter(position); editor.focus();
    });
    return button;
  }));
}

async function formatActiveFile() {
  const model = models.get(activeFile);
  if (!model || !isEditableFile(activeFile) || !activeFile.endsWith(".go")) return;
  try {
    elements.languageStatus.textContent = "Formatting…";
    const formatted = await requestFormat(activeFile, model.getValue());
    if (formatted === model.getValue()) {
      elements.languageStatus.textContent = "Already formatted";
      return;
    }
    editor.executeEdits("gofmt", [{ range: model.getFullModelRange(), text: formatted, forceMoveMarkers: true }]);
    elements.languageStatus.textContent = "Formatted with gofmt";
  } catch (error) {
    renderProblems([{ file: activeFile, line: 1, column: 1, message: error.message || String(error) }]); showPanel("problems");
  }
}

function requestFormat(name, source) {
  const id = ++requestID;
  const data = encoder.encode(source);
  const result = new Promise((resolve, reject) => formatRequests.set(id, { resolve, reject }));
  worker.postMessage({ type: "format", id, name, data }, [data.buffer]);
  return result;
}

function receiveFormatResult(result) {
  const pending = formatRequests.get(result.id); if (!pending) return;
  formatRequests.delete(result.id);
  if (result.exitCode !== 0 || result.error) pending.reject(new Error((result.error || "gofmt failed").trim()));
  else pending.resolve(result.output);
}

function searchProject() {
  showPanel("search");
  if (!elements.searchQuery.value) elements.searchQuery.value = editor?.getModel()?.getWordAtPosition(editor.getPosition())?.word || "";
  elements.searchQuery.focus();
  elements.searchQuery.select();
}

function submitProjectSearch(event) {
  event.preventDefault();
  renderProjectSearch(elements.searchQuery.value.trim());
}

function renderProjectSearch(query) {
  if (!query) return;
  const matches = [];
  for (const name of [...editableFiles].sort()) {
    const model = models.get(name); if (!model) continue;
    const lines = model.getLinesContent();
    for (let index = 0; index < lines.length; index++) {
      let at = lines[index].toLocaleLowerCase().indexOf(query.toLocaleLowerCase());
      while (at >= 0) {
        matches.push({ name, line: index + 1, column: at + 1, text: lines[index].trim() });
        at = lines[index].toLocaleLowerCase().indexOf(query.toLocaleLowerCase(), at + Math.max(1, query.length));
      }
    }
  }
  if (!matches.length) {
    const empty = document.createElement("div"); empty.className = "empty-state"; empty.textContent = `No results for ${query}.`;
    elements.searchMatches.replaceChildren(empty);
  } else elements.searchMatches.replaceChildren(...matches.map((match) => {
    const row = document.createElement("button"); row.type = "button"; row.className = "search-row";
    const location = document.createElement("span"); location.textContent = `${match.name}:${match.line}:${match.column}`;
    const text = document.createElement("span"); text.textContent = match.text;
    row.append(location, text); row.addEventListener("click", () => { openFile(match.name); editor.setPosition({ lineNumber: match.line, column: match.column }); editor.revealPositionInCenter({ lineNumber: match.line, column: match.column }); });
    return row;
  }));
  showPanel("search");
}

async function installCachedBackends() {
  let cached = [];
  try { cached = await loadPreparedBackends(); } catch {}
  for (const record of cached) {
    const compiler = record.compiler || record.wasm;
    if (!compiler || !record.manifest) continue;
    cachedBackendRecords.set(backendCacheID(record.manifest), { ...record, compiler });
    if (!record.manifest.projectDefinition) installCustomTarget(record.manifest, compiler, false);
  }
}

async function restoreProjectBackends() {
  if (!projectBackendRoots.size) return;
  let changed = false;
  for (const definition of [...projectBackendRoots]) {
    await ensureBundledBackendModel(definition);
    if (!models.has(definition)) continue;
    try {
      await inspectProjectBackend(definition, false);
      changed = true;
    } catch (error) {
      renderProblems(parseDiagnostics(error.message || String(error)));
    }
  }
  if (changed) configureTargets(targetCatalog.targets);
}

async function ensureBundledBackendModel(definition) {
  if (models.has(definition) || !standardCatalogPromise) return;
  const catalog = await standardCatalogPromise;
  for (const [importPath, item] of Object.entries(catalog.platforms || {})) {
    if (!(item.files || []).some((file) => `${item.root}/${file}` === definition)) continue;
    await loadStandardPackage(importPath, catalog);
    await ensureSourceModel(definition);
    return;
  }
}

async function useProjectBackend(definition) {
  if (!definition?.endsWith(".rtg") || !models.has(definition)) return;
  projectBackendRoots.add(definition);
  saveFiles();
  setCompilerStatus("busy", `Reading ${definition}…`);
  try {
    const targets = await inspectProjectBackend(definition, false);
    if (!targets.length) throw new Error(`${definition} does not export a target.`);
    configureTargets(targetCatalog.targets);
    selectTarget(targets[0].name, true);
    setCompilerStatus("ready", "Compiler ready");
    elements.languageStatus.textContent = targets.length === 1
      ? `Project backend ${targets[0].name} selected`
      : `${targets.length} project backend targets added`;
  } catch (error) {
    renderProblems(parseDiagnostics(error.message || String(error)));
    showPanel("problems");
    setCompilerStatus("error", "Backend definition failed");
  }
}

async function inspectProjectBackend(definition, configure = true) {
  const result = await requestProjectBackend("backend-inspect", definition);
  if (result.exitCode !== 0) throw new Error(result.stderr.trim() || "backend definition inspection failed");
  let manifests;
  try { manifests = JSON.parse(result.stdout); } catch { throw new Error("backend JIT returned an invalid target manifest"); }
  if (!Array.isArray(manifests)) throw new Error("backend JIT did not return a target list");
  const prior = new Map(targetCatalog.targets.map((target) => [target.name, target]));
  targetCatalog.targets = targetCatalog.targets.filter((target) => target.projectDefinition !== definition);
  const installed = [];
  for (const manifest of manifests) {
    validateBackendManifest(manifest);
    const previous = prior.get(manifest.name);
    const record = cachedBackendRecords.get(backendCacheID(manifest));
    const cached = previous?.definition === manifest.definition.toLowerCase() && previous.backend;
    const backendFormat = manifest.backendFormat || record?.manifest?.backendFormat || "vm32";
    const backend = cached ? previous.backend : record?.compiler
      ? customBackendURL(manifest.name, backendFormat, record.compiler) : "";
    const target = {
      name: manifest.name, backendTarget: manifest.backendTarget || manifest.name,
      backend, backendFormat: cached ? previous.backendFormat || backendFormat : backendFormat,
      output: manifest.output || "app", runnable: Boolean(manifest.runnable), tags: manifest.tags || [],
      definition: manifest.definition.toLowerCase(), descriptorVersion: manifest.descriptorVersion,
      device: manifest.name.startsWith("esp32") ? "esp32" : "",
      projectBackend: true, projectDefinition: definition, backendStale: false,
    };
    targetCatalog.targets = targetCatalog.targets.filter((item) => item.name !== target.name);
    targetCatalog.targets.push(target);
    installed.push(target);
  }
  if (configure) configureTargets(targetCatalog.targets);
  return installed;
}

async function prepareProjectBackend(target) {
  const targets = await inspectProjectBackend(target.projectDefinition, false);
  const current = targets.find((item) => item.name === target.name);
  if (!current) throw new Error(`${target.projectDefinition} no longer exports ${target.name}.`);
  if (current.backend && !current.backendStale) {
    configureTargets(targetCatalog.targets);
    selectTarget(current.name, false);
    return targetCatalog.targets.find((item) => item.name === current.name);
  }
  const result = await requestProjectBackend("backend-prepare", target.projectDefinition, target.name);
  if (result.exitCode !== 0 || !result.compiler) throw new Error(result.stderr.trim() || "backend compiler preparation failed");
  let manifest;
  try { manifest = JSON.parse(result.stdout); } catch { throw new Error("backend JIT returned an invalid prepared manifest"); }
  manifest = { ...manifest, backendFormat: "vm32", projectDefinition: target.projectDefinition };
  installCustomTarget(manifest, result.compiler, true);
  configureTargets(targetCatalog.targets);
  selectTarget(manifest.name, false);
  return targetCatalog.targets.find((item) => item.name === manifest.name);
}

function requestProjectBackend(type, definition, target = "") {
  const payload = workspacePayload();
  const id = ++requestID;
  const pending = new Promise((resolve) => backendRequests.set(id, resolve));
  worker.postMessage({ type, id, definition, target, files: payload.files }, payload.transfers);
  return pending;
}

function receiveBackendResult(result) {
  const pending = backendRequests.get(result.id);
  if (!pending) return;
  backendRequests.delete(result.id);
  pending(result);
}

async function importPreparedBackend(list) {
  if (!list?.length) return;
  try {
    const manifestFile = [...list].find((file) => file.name.endsWith(".json"));
    const wasmFile = [...list].find((file) => file.name.endsWith(".wasm"));
    const rtgFile = [...list].find((file) => file.name.endsWith(".rtg"));
    if (rtgFile && !models.has(rtgFile.name)) {
      createProjectModel(rtgFile.name, await rtgFile.text()); editableBaselines.set(rtgFile.name, models.get(rtgFile.name).getValue()); renderWorkspaceFiles();
    }
    if (rtgFile && !manifestFile && !wasmFile) {
      await useProjectBackend(rtgFile.name);
    } else {
      if (!manifestFile || !wasmFile) throw new Error("Choose a prepared backend manifest (.json) and its WebAssembly compiler (.wasm), or import an RTG definition.");
      const manifest = JSON.parse(await manifestFile.text());
      const wasm = await wasmFile.arrayBuffer();
      validateBackendManifest(manifest);
      installCustomTarget(manifest, wasm, true);
      configureTargets(targetCatalog.targets);
      selectTarget(manifest.name, true);
      elements.languageStatus.textContent = `Imported backend ${manifest.name}`;
    }
  } catch (error) { showProjectError(error); }
  elements.backendFileInput.value = "";
}

function validateBackendManifest(manifest) {
  if (!manifest || typeof manifest.name !== "string" || !manifest.name.includes("/")) throw new Error("The backend manifest needs a target name such as acme/amd64.");
  if (!/^[0-9a-fA-F]{64}$/.test(manifest.definition || "")) throw new Error("The backend manifest needs the 64-digit target definition signature.");
  if (!Number.isInteger(manifest.descriptorVersion) || manifest.descriptorVersion <= 0) throw new Error("The backend manifest needs a positive descriptorVersion.");
  if (manifest.backendFormat && manifest.backendFormat !== "wasm" && manifest.backendFormat !== "vm32") throw new Error("The backend manifest has an unsupported compiler format.");
}

function installCustomTarget(manifest, compiler, persist) {
  validateBackendManifest(manifest);
  const existing = targetCatalog.targets.findIndex((target) => target.name === manifest.name);
  if (existing >= 0) targetCatalog.targets.splice(existing, 1);
  const backendFormat = manifest.backendFormat || "wasm";
  const url = customBackendURL(manifest.name, backendFormat, compiler);
  targetCatalog.targets.push({
    name: manifest.name, backendTarget: manifest.backendTarget || manifest.name, backend: url,
    output: manifest.output || "app", runnable: Boolean(manifest.runnable), tags: manifest.tags || [],
    definition: manifest.definition.toLowerCase(), descriptorVersion: manifest.descriptorVersion, custom: true,
    backendFormat, projectBackend: Boolean(manifest.projectDefinition), projectDefinition: manifest.projectDefinition || "",
    device: manifest.name.startsWith("esp32") ? "esp32" : "", backendStale: false,
  });
  if (persist) {
    const record = { id: backendCacheID(manifest), manifest, compiler };
    cachedBackendRecords.set(record.id, record);
    savePreparedBackend(record).catch((error) => showProjectError(error));
  }
}

function customBackendURL(name, backendFormat, compiler) {
  const old = customBackendURLs.get(name); if (old) URL.revokeObjectURL(old);
  const type = backendFormat === "wasm" ? "application/wasm" : "application/octet-stream";
  const url = URL.createObjectURL(new Blob([compiler], { type }));
  customBackendURLs.set(name, url);
  return url;
}

function backendCacheID(manifest) {
  return `${manifest.name}:${String(manifest.definition || "").toLowerCase()}:v${manifest.descriptorVersion}:${manifest.backendFormat || "wasm"}`;
}

function markBuildStale() {
  buildRevision++;
  scheduleBuildValidation();
}

function renderProblems(problems) {
  elements.problemCount.textContent = String(problems.length);
  elements.problemStatus.querySelector("span").textContent = String(problems.length);
  elements.languageStatus.textContent = problems.length ? `${problems.length} problem${problems.length === 1 ? "" : "s"}` : "No problems";
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

function parseDiagnostics(stderr) {
  const problems = [];
  for (const rawLine of stderr.split("\n")) {
    const line = rawLine.trim();
    if (!line) continue;
    const match = /^(?:renvo:\s*)?(.+?\.(?:go|c|h|rtg|rtgasm)):(\d+)(?::(\d+))?:\s*(.*)$/.exec(line);
    if (match) {
      const structured = /^error\s+([A-Z0-9-]+)(?:\s+\([^)]+\))?:\s*(.*)$/.exec(match[4]);
      problems.push({
        file: cleanPath(match[1]), line: Number(match[2]), column: Number(match[3] || 1),
        code: structured?.[1] || "", message: structured?.[2] || match[4],
      });
    } else {
      const text = line.replace(/^renvo:\s*/, "");
      const structured = /^error\s+([A-Z0-9-]+)(?:\s+\([^)]+\))?:\s*(.*)$/.exec(text);
      problems.push({ file: "", line: 0, column: 0, code: structured?.[1] || "", message: structured?.[2] || text });
    }
  }
  if (!problems.some((problem) => problem.code)) return problems;
  return problems.filter((problem) => problem.code || !/^(?:compilation|frontend compilation|build) failed$/i.test(problem.message));
}

function revealProblem(problem) {
  openFile(problem.file);
  const position = Number.isFinite(problem.start) ? positionAtByteOffset(models.get(problem.file), problem.start) : { lineNumber: problem.line || 1, column: problem.column || 1 };
  editor.setPosition(position);
  editor.revealPositionInCenter(position);
}

function showPanel(name) {
  if (name !== "plotter") setPlotterExpanded(false);
  elements.workbench.classList.remove("panel-hidden");
  document.querySelectorAll(".panel-tab").forEach((tab) => {
    const active = tab.dataset.panel === name;
    tab.classList.toggle("active", active);
    tab.setAttribute("aria-selected", String(active));
  });
  document.querySelectorAll(".panel-view").forEach((view) => view.classList.toggle("active", view.dataset.panelView === name));
  elements.togglePlotterSize.hidden = name !== "plotter" || isPhoneWorkspace();
  elements.clearOutput.hidden = !["output", "terminal", "plotter", "tests", "search"].includes(name);
}

function togglePlotterSize() {
  setPlotterExpanded(!elements.workbench.classList.contains("plotter-expanded"));
}

function setPlotterExpanded(expanded) {
  elements.workbench.classList.toggle("plotter-expanded", expanded);
  elements.editorHost.inert = expanded;
  elements.editorHost.setAttribute("aria-hidden", String(expanded));
  elements.togglePlotterSize.setAttribute("aria-pressed", String(expanded));
  elements.togglePlotterSize.textContent = expanded ? "Restore" : "Expand";
  elements.togglePlotterSize.title = expanded ? "Restore editor and plotter" : "Expand plotter over editor";
}

function appendSerialText(text) {
  elements.terminalOutput.textContent += text;
  if (serialPlotter.push(text) && !plotterAutoShown) {
    plotterAutoShown = true;
    showPanel("plotter");
  }
}

function clearActivePanel() {
  const active = document.querySelector(".panel-tab.active")?.dataset.panel;
  if (active === "terminal") elements.terminalOutput.textContent = "";
  else if (active === "plotter") serialPlotter.clear();
  else if (active === "tests") elements.testsOutput.textContent = "Add a _test.go file, then press Test.";
  else if (active === "search") {
    elements.searchQuery.value = "";
    elements.searchMatches.innerHTML = '<div class="empty-state">Search the project with Ctrl+Shift+F.</div>';
  } else if (active === "output") elements.output.textContent = "";
}

function togglePanel() {
  if (!elements.workbench.classList.contains("panel-hidden")) setPlotterExpanded(false);
  elements.workbench.classList.toggle("panel-hidden");
  elements.togglePanel.setAttribute("aria-pressed", String(!elements.workbench.classList.contains("panel-hidden")));
}

function toggleSidebar() {
  const compact = matchMedia("(max-width: 820px)").matches;
  if (compact) elements.ide.classList.toggle("sidebar-open");
  else elements.ide.classList.toggle("sidebar-hidden");
  const visible = compact ? elements.ide.classList.contains("sidebar-open") : !elements.ide.classList.contains("sidebar-hidden");
  elements.toggleSidebar.setAttribute("aria-expanded", String(visible));
  requestAnimationFrame(() => editor?.layout());
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

function setSetupStep(step, state, detail) {
  const order = ["workspace", "catalog", "editor", "compiler"];
  const item = elements.mobileSetup?.querySelector(`[data-setup-step="${step}"]`);
  if (!item) return;
  item.dataset.state = state;
  item.dataset.detail = detail || "";
  const items = order.map((name) => elements.mobileSetup.querySelector(`[data-setup-step="${name}"]`));
  const done = items.filter((entry) => entry.dataset.state === "done").length;
  elements.mobileSetupProgress.value = done;
  elements.mobileSetupProgress.textContent = `${done} of ${order.length}`;
  const error = items.find((entry) => entry.dataset.state === "error");
  const active = [...items].reverse().find((entry) => entry.dataset.state === "active");
  if (error) {
    elements.mobileSetup.dataset.state = "error";
    elements.mobileSetupTitle.textContent = "Renvo could not start";
    elements.mobileSetupDetail.textContent = error.dataset.detail;
  } else if (done === order.length) {
    elements.mobileSetup.dataset.state = "done";
    elements.mobileSetupTitle.textContent = "Renvo is ready";
    elements.mobileSetupDetail.textContent = "Choose an example, then load it onto the board in the editor.";
  } else {
    elements.mobileSetup.dataset.state = "loading";
    elements.mobileSetupTitle.textContent = "Getting Renvo ready";
    elements.mobileSetupDetail.textContent = active?.dataset.detail || detail || "Starting…";
  }
  if (isPhoneWorkspace()) updateMobileHeader();
}

function updateReadyState() {
  const board = isBoardTarget(selectedTarget);
  const capabilities = targetCapabilities(selectedTarget);
  const downloadable = hasDownloadableOutput(selectedTarget);
  const jtag = selectedTarget?.device === "esp32" && elements.flashTransport.value === "webusb" && supportsESPWebUSBJTAG(deviceMachineTarget());
  const deviceAction = selectedTarget?.device === "rp2" ? "Monitor load" : jtag ? "JTAG load" : "Flash";
  const primaryLabel = board ? deviceAction : "Download";
  const readiness = buildReadiness({
    compilerReady, editorReady: Boolean(monaco), building, state: buildValidationState,
    currentRevision: buildRevision, validatedRevision: buildValidationRevision, readyLabel: primaryLabel,
  });
  const downloadReadiness = buildReadiness({
    compilerReady, editorReady: Boolean(monaco), building, state: buildValidationState,
    currentRevision: buildRevision, validatedRevision: buildValidationRevision, readyLabel: "Download",
  });
  elements.compile.disabled = !readiness.ready || (!board && !downloadable) || running || runAfterBuild;
  elements.compile.title = readiness.title;
  elements.compile.querySelector("span").textContent = running && board ? `${deviceAction}…` : runAfterBuild && board ? "Preparing…" : readiness.label;
  elements.compile.querySelector("path").setAttribute("d", board ? "M8 14V6m-3 3 3-3 3 3M3 3h10" : "M8 2v8m-3-3 3 3 3-3M3 13h10");
  elements.mobileDeviceBuild.disabled = elements.compile.disabled;
  elements.mobileDeviceBuild.textContent = elements.compile.querySelector("span").textContent;
  elements.test.disabled = !compilerReady || !monaco || building || running;
  elements.test.textContent = testBuild ? "Testing…" : "Test";
  elements.targetButton.disabled = building || running;
  elements.run.disabled = !readiness.ready || (board ? !downloadable : !capabilities.runsInBrowser) || running || runAfterBuild;
  elements.flashTransport.disabled = building || running || runAfterBuild;
  elements.run.title = board ? "Download firmware" : "Run (F5)";
  elements.run.querySelector("span").textContent = board ? "Download" : running ? "Running…" : runAfterBuild ? "Run pending…" : "Run";
  elements.run.querySelector("path").setAttribute("d", board ? "M8 2v8m-3-3 3 3 3-3M3 13h10" : "m5 3 8 5-8 5V3Z");
  elements.mobileDownload.disabled = !downloadReadiness.ready || !downloadable || running || runAfterBuild;
  elements.mobileDownload.textContent = !compilerReady || !monaco ? "Loading…" : downloadReadiness.label;
  elements.mobileDownload.title = downloadable ? downloadReadiness.title : "This target has no downloadable output";
  elements.mobileDownload.classList.toggle("primary", !board && !capabilities.runsInBrowser && downloadable);
  elements.mobileRun.disabled = board ? elements.compile.disabled : elements.run.disabled;
  elements.mobileRun.textContent = !compilerReady || !monaco ? "Loading…" : running ? (board ? `${deviceAction}…` : "Running…") :
    runAfterBuild ? "Pending…" : (board ? deviceAction : "Run");
  elements.mobileRun.classList.toggle("deploying", mobileDeploymentActive);
  elements.mobileRun.classList.toggle("primary", board || capabilities.runsInBrowser);
  if (mobileDeploymentActive) {
    elements.mobileRun.disabled = false;
    elements.mobileRun.textContent = mobileDeploymentLabel || "Working…";
    elements.mobileRun.title = "Tap to see device load details";
  }
  elements.mobileDeviceRun.disabled = elements.run.disabled;
  elements.mobileDeviceRun.textContent = board ? "Download" : running ? "Running…" : runAfterBuild ? "Waiting…" : "Run";
  elements.mobileDeviceBuild.classList.toggle("primary", board || !capabilities.runsInBrowser);
  elements.mobileDeviceRun.classList.toggle("primary", !board && capabilities.runsInBrowser);
  if (autoBuildPending && readiness.ready && selectedTarget && !building) {
    autoBuildPending = false;
    queueMicrotask(capabilities.runsInBrowser ? runArtifact : primaryTargetAction);
  }
}

function isPhoneWorkspace() {
  return phoneWorkspace.matches || currentDeviceProfile().phone;
}

function configureMobileWorkspace() {
  const profile = currentDeviceProfile();
  elements.ide.dataset.deviceClass = profile.deviceClass;
  if (isPhoneWorkspace()) {
    setPlotterExpanded(false);
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
  updateMobileHeader();
  if (view === "editor") requestAnimationFrame(() => editor?.layout());
}

function updateMobileHeader() {
  const view = elements.ide.dataset.mobileView || "files";
  const file = activeFile.split("/").pop();
  elements.mobileTargetButton.textContent = view === "device" ? "Code" : "Device";
  elements.mobileTargetButton.title = selectedTarget ? `Change target. Using ${selectedTarget.label || selectedTarget.name}.` : "Choose a target";
  if (view === "files") {
    elements.mobileStep.textContent = "Project";
    elements.mobileContext.textContent = projectName;
  } else if (view === "device") {
    elements.mobileStep.textContent = "Device";
    elements.mobileContext.textContent = selectedTarget?.label || selectedTarget?.name || "No target selected";
  } else if (view === "editor") {
    elements.mobileStep.textContent = file;
    const target = selectedTarget?.label || selectedTarget?.name || "No target selected";
    elements.mobileContext.textContent = compilerReady ? target : `Loading compiler · ${target}`;
  } else {
    elements.mobileStep.textContent = "Project";
    elements.mobileContext.textContent = projectName;
  }
  if (elements.mobileDeviceTarget) elements.mobileDeviceTarget.textContent = selectedTarget?.label || selectedTarget?.name || "Choose a target";
  if (elements.mobileDeviceHint) elements.mobileDeviceHint.textContent = targetCapabilityHint(selectedTarget);
  document.querySelector(".mobile-transport-picker").hidden = !isBoardTarget(selectedTarget);
}

function openMobileFlashView(state = "Preparing…") {
  if (!isPhoneWorkspace()) return;
  elements.mobileFlashView.hidden = false;
  elements.mobileFlashState.textContent = state;
  document.querySelectorAll(".mobile-nav button").forEach((button) => {
    button.classList.toggle("active", button.dataset.mobileView === "device");
  });
  syncMobileFlashOutput();
}

function startMobileDeployment(jtag) {
  mobileDeploymentActive = true;
  mobileDeploymentLabel = "Starting…";
  mobileDeploymentStep = "";
  elements.mobileFlashView.hidden = true;
  elements.mobileFlashProgress.value = 0;
  const monitor = selectedTarget?.device === "rp2";
  elements.mobileFlashState.textContent = monitor ? "Monitor load" : jtag ? "JTAG load" : "Flash board";
  elements.mobileFlashDetail.textContent = "Preparing the board…";
  for (const item of document.querySelectorAll("[data-deploy-step]")) {
    item.dataset.state = "pending";
    item.querySelector("small").textContent = "Waiting";
  }
  const loadName = document.querySelector('[data-deploy-step="load"] strong');
  if (loadName) loadName.textContent = monitor ? "Monitor load" : jtag ? "JTAG load" : "Flash board";
  elements.terminalOutput.textContent = `$ ${monitor ? "monitor-load" : jtag ? "jtag-load" : "flash"} ${selectedTarget?.label || selectedTarget?.name || "board"}\n`;
  document.querySelector(".mobile-flash-log").open = false;
  updateReadyState();
}

function setMobileDeployStep(step, state, detail, partial = 0) {
  const order = ["usb", "check", "firmware", "load", "run"];
  const item = document.querySelector(`[data-deploy-step="${step}"]`);
  if (!item) return;
  mobileDeploymentStep = step;
  item.dataset.state = state;
  item.querySelector("small").textContent = detail;
  const labels = {
    usb: "USB…", check: "Checking…", firmware: "Building…",
    load: partial > 0 ? `Loading ${Math.round(partial * 100)}%` : "Connecting…", run: "Starting…",
  };
  if (state === "active") mobileDeploymentLabel = labels[step];
  const items = order.map((name) => document.querySelector(`[data-deploy-step="${name}"]`));
  const done = items.filter((entry) => entry.dataset.state === "done").length;
  const position = state === "active" ? order.indexOf(step) + Math.max(0, Math.min(1, partial)) : done;
  elements.mobileFlashProgress.value = Math.max(done, position) / order.length;
  elements.mobileFlashState.textContent = state === "error" ? "Load failed" : item.querySelector("strong").textContent;
  elements.mobileFlashDetail.textContent = detail;
  updateReadyState();
}

function finishMobileDeployment(state) {
  mobileDeploymentActive = false;
  mobileDeploymentLabel = "";
  mobileDeploymentStep = "run";
  elements.mobileFlashProgress.value = 1;
  elements.mobileFlashState.textContent = state;
  updateReadyState();
}

function failMobileDeployment(step, detail) {
  if (!mobileDeploymentActive) startMobileDeployment(elements.flashTransport.value === "webusb");
  setMobileDeployStep(step, "error", detail);
  mobileDeploymentActive = false;
  mobileDeploymentLabel = "";
  document.querySelector(".mobile-flash-log").open = true;
  openMobileFlashView("Load failed");
  updateReadyState();
}

function closeMobileFlashView() {
  elements.mobileFlashView.hidden = true;
  showMobileView("editor");
}

function syncMobileFlashOutput() {
  elements.mobileFlashOutput.textContent = elements.terminalOutput.textContent;
  elements.mobileFlashOutput.scrollTop = elements.mobileFlashOutput.scrollHeight;
  elements.mobileDeviceOutput.textContent = elements.terminalOutput.textContent || "Nothing to show yet.";
}

function configureFlashTransports() {
  const { profile, choices } = deviceAccessSupport();
  for (const option of elements.flashTransport.options) {
    option.disabled = !choices[option.value];
    const name = option.value === "webusb" ? "WebUSB (JTAG on ESP32-C6)" : "WebSerial";
    option.textContent = `${name}${option.disabled ? " unavailable" : ""}`;
  }
  const saved = localStorage.getItem("renvo.espFlashTransport");
  elements.flashTransport.value = preferredESPTransport({
    saved, android: profile.mobileCapable, webSerial: choices.webserial, webUSB: choices.webusb,
  });
  if (elements.mobileTransportStatus) {
    const available = [choices.webusb && "WebUSB", choices.webserial && "WebSerial"].filter(Boolean);
    elements.mobileTransportStatus.textContent = available.length ? `Use ${available.join(" or ")}` :
      profile.ios ? "iPhone and iPad browsers cannot use USB" : "This browser cannot use USB";
  }
  syncMobileTransportPicker();
}

function updateFlashTransportChoices() {
  const { choices } = deviceAccessSupport();
  const pico = selectedTarget?.device === "rp2";
  for (const option of elements.flashTransport.options) {
    option.disabled = pico ? option.value !== "webusb" || !choices.webusb : !choices[option.value];
    const name = option.value === "webusb"
      ? pico ? "WebUSB (Pico monitor)" : "WebUSB (JTAG on ESP32-C6)"
      : "WebSerial";
    option.textContent = `${name}${option.disabled ? " unavailable" : ""}`;
  }
  syncMobileTransportPicker();
}

function currentDeviceProfile() {
  const userAgentData = navigator.userAgentData || {};
  const screenWidth = Math.min(screen?.width || innerWidth, screen?.height || innerHeight);
  const profile = detectDeviceProfile({
    platform: userAgentData.platform || navigator.platform || "",
    userAgent: navigator.userAgent || "",
    mobile: userAgentData.mobile,
    maxTouchPoints: navigator.maxTouchPoints,
    coarsePointer: matchMedia("(pointer: coarse)").matches,
    width: innerWidth,
    shortSide: screenWidth,
  });
  const forced = parameters.get("device");
  if (forced === "phone") return { ...profile, mobileCapable: true, phone: true, tablet: false, deviceClass: "phone" };
  if (forced === "tablet") return { ...profile, mobileCapable: true, phone: false, tablet: true, deviceClass: "tablet" };
  return profile;
}

async function changeFlashTransport() {
  localStorage.setItem("renvo.espFlashTransport", elements.flashTransport.value);
  syncMobileTransportPicker();
  updateReadyState();
  prefetchTargetBackend(selectedTarget);
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
    button.hidden = selectedTarget?.device === "rp2" && button.dataset.mobileTransport !== "webusb";
    button.setAttribute("aria-checked", String(button.dataset.mobileTransport === elements.flashTransport.value));
  }
}

function showFatalError(error) {
  building = false; running = false; runAfterBuild = false; compilerReady = false;
  setSetupStep("compiler", "error", error.message || String(error));
  updateReadyState(); setCompilerStatus("error", "Unavailable");
  elements.output.textContent = `${error.message || error}\n`; showPanel("output");
}

function defineTheme() {
  monaco.editor.defineTheme("renvo-dark", {
    base: "vs-dark", inherit: true,
    rules: [
      { token: "keyword", foreground: "E36388" }, { token: "type", foreground: "A6BE82" },
      { token: "keyword.directive", foreground: "D79BC8" }, { token: "identifier.function", foreground: "DCCDE8" },
      { token: "string", foreground: "AFC29D" }, { token: "number", foreground: "D0B06E" },
      { token: "comment", foreground: "8D9E7E", fontStyle: "italic" },
      { token: "operator", foreground: "CDA8D5" }, { token: "delimiter", foreground: "B6A4BA" },
      { token: "identifier", foreground: "E8DDEA" },
    ],
    colors: {
      "editor.background": "#1b0718", "editor.foreground": "#d8cae2",
      "editorLineNumber.foreground": "#846f88", "editorLineNumber.activeForeground": "#dccde8",
      "editor.lineHighlightBackground": "#35142e99", "editorCursor.foreground": "#f4ebf7",
      "editor.selectionBackground": "#596f3f99", "editor.inactiveSelectionBackground": "#45503b80",
      "editor.selectionHighlightBackground": "#596f3f4d", "editor.wordHighlightBackground": "#45503b66",
      "editorIndentGuide.background1": "#3c2235", "editorIndentGuide.activeBackground1": "#6c4d64",
      "editorWhitespace.foreground": "#5a3a51", "editorGutter.background": "#1b0718",
      "editorError.foreground": "#e36388", "editorWarning.foreground": "#d0b06e",
      "editorInfo.foreground": "#a6be82", "editorHint.foreground": "#9aad87",
      "editorWidget.background": "#301129", "editorWidget.border": "#5a3a51",
      "editorHoverWidget.background": "#301129", "editorHoverWidget.border": "#5a3a51",
      "editorSuggestWidget.background": "#301129", "editorSuggestWidget.border": "#5a3a51",
      "editorSuggestWidget.selectedBackground": "#29321f", "editorSuggestWidget.highlightForeground": "#a6be82",
      "input.background": "#130410", "input.border": "#5a3a51", "input.foreground": "#d8cae2",
      "focusBorder": "#b2c98d", "list.hoverBackground": "#35142e", "list.activeSelectionBackground": "#29321f",
      "scrollbarSlider.background": "#5a3a5166", "scrollbarSlider.hoverBackground": "#b6244f80",
      "scrollbarSlider.activeBackground": "#e3638899", "minimap.background": "#180414",
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
  if (!editableFiles.size) return;
  const project = currentProject();
  clearTimeout(saveTimer);
  saveTimer = setTimeout(() => saveCurrentProject(project).catch(() => {
    try { localStorage.setItem("renvo.playground.files.v1", JSON.stringify(project.files)); } catch {}
  }), 120);
}

function saveAndDeploy() {
  saveFiles();
  if (isBoardTarget(selectedTarget)) runArtifact();
}

function currentProject() {
  return { name: projectName, language: projectLanguage, buildLanguage: activeBuildRoot === "." ? "" : externalBuildLanguage, files: projectFiles(), activeFile, openFiles: [...openFiles], command: elements.command?.value || "-s -o app.wasm .", arenaSize: elements.arenaSize?.value || "", target: selectedTarget?.name || restoredTargetName, backendRoots: [...projectBackendRoots] };
}

async function restoreProject() {
  let project;
  try {
    const shared = new URLSearchParams(location.hash.replace(/^#/, "")).get("project");
    if (shared) project = { name: "shared-project", files: decodeSharedProject(shared), activeFile: "main.go" };
    else project = await loadCurrentProject();
  } catch (error) { console.warn("Could not restore IndexedDB project", error); }
  if (!project) {
    try {
      const legacy = JSON.parse(localStorage.getItem("renvo.playground.files.v1") || "{}");
      if (legacy && typeof legacy === "object" && !Array.isArray(legacy) && Object.keys(legacy).length) project = { name: "playground", files: legacy };
    } catch {}
  }
  projectLanguage = project?.language === "c" || !project?.language && inferProjectLanguage(project?.files) === "c" ? "c" : "go";
  const fallbackFiles = projectLanguage === "c" ? initialCFiles : initialFiles;
  fileValues = project?.files && Object.keys(project.files).length ? { ...project.files } : { ...fallbackFiles };
  editableFiles.clear(); editableBaselines.clear();
  for (const [name, source] of Object.entries(fileValues)) { editableFiles.add(name); editableBaselines.set(name, source); }
  projectName = project?.name || "playground";
  restoredTargetName = project?.target || "";
  projectBackendRoots.clear();
  for (const name of project?.backendRoots || []) if (typeof name === "string") projectBackendRoots.add(name);
  activeFile = project?.activeFile && Object.hasOwn(fileValues, project.activeFile) ? project.activeFile : Object.keys(fileValues)[0];
  lastWorkspaceFile = activeFile;
  openFiles.length = 0;
  for (const name of project?.openFiles || []) if (Object.hasOwn(fileValues, name) && !openFiles.includes(name)) openFiles.push(name);
  if (!openFiles.includes(activeFile)) openFiles.push(activeFile);
  if (project?.command) elements.command.value = project.command;
  elements.arenaSize.value = project?.arenaSize ? String(project.arenaSize) : "";
  externalBuildLanguage = project?.buildLanguage === "c" ? "c" : "go";
  syncProjectName();
  syncBuildRootFromCommand();
}

function createDownloadURL(data, type) {
  if (data.byteLength <= inlineDownloadLimit) {
    const bytes = new Uint8Array(data);
    const chunks = [];
    for (let offset = 0; offset < bytes.length; offset += 0x8000) {
      chunks.push(String.fromCharCode(...bytes.subarray(offset, offset + 0x8000)));
    }
    return `data:${type};base64,${btoa(chunks.join(""))}`;
  }
  return URL.createObjectURL(new Blob([data], { type }));
}

function releaseDownloadURL(url) { if (url.startsWith("blob:")) URL.revokeObjectURL(url); }

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

function syncBuildRootFromCommand() {
  try {
    const args = splitArguments(elements.command.value);
    const candidate = args[args.length - 1];
    activeBuildRoot = candidate === "." || candidate?.startsWith("./") ? candidate : ".";
  } catch {
    activeBuildRoot = ".";
  }
  syncBuildScope();
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

function exampleEntries(catalog = standardCatalog) {
  return Object.entries(catalog?.platforms || {})
    .filter(([, item]) => item.main && !item.hidden)
    .map(([importPath, item]) => {
      const boards = item.boards || (item.board ? [{ name: item.board, target: item.target }] : []);
      const computers = (item.computers || []).map((computer) => ({ ...computer, device: "computer" }));
      const machines = [...boards, ...computers];
      const targets = [...new Set([...machines.map((choice) => choice.target), item.target].filter(Boolean))];
      const slug = item.root.split("/").pop();
      return { importPath, item, boards, computers, machines, targets, slug, title: exampleTitle(slug) };
    })
    .sort((left, right) => left.title.localeCompare(right.title));
}

function exampleTitle(slug) {
  if (slug === "pdp11v7") return "PDP-11 V7";
  if (slug === "msdos") return "MS-DOS 8086";
  return slug.split(/[_-]+/).map((word) => {
    if (/^(c|i2c|http|rgb|usb|ws2812|adxl345|sgp30)$/i.test(word)) return word.toUpperCase();
    return word.charAt(0).toUpperCase() + word.slice(1);
  }).join(" ");
}

function examplePresentation(entry) {
  const slug = entry.slug;
  if (entry.computers.length) return { icon: "R", category: entry.computers[0].family || "Computer", tone: "system" };
  if (/adxl|quality|env|sgp|sensor/.test(slug)) return { icon: "⌁", category: "Sensors", tone: "sensor" };
  if (/forms|touch|keyboard/.test(slug)) return { icon: "▦", category: "Input", tone: "interface" };
  if (/terminal/.test(slug)) return { icon: ">_", category: "Terminal", tone: "console" };
  if (/blink|rgb|ws2812/.test(slug)) return { icon: "✦", category: "LEDs", tone: "light" };
  return { icon: "R", category: "General", tone: "system" };
}

const boardArtworkPaths = {
  nanoc6: `
    <rect x="18" y="31" width="124" height="38" rx="9" class="board-shell"/>
    <rect x="142" y="39" width="15" height="22" rx="3" class="board-metal"/>
    <path d="M28 38h30v24H28zM33 43h20v14H33z" class="board-chip"/>
    <circle cx="77" cy="50" r="6" class="board-led"/><circle cx="120" cy="50" r="5" class="board-button"/>
    <path d="M22 26v-8m12 8v-8m12 8v-8M22 82v-8m12 8v-8m12 8v-8" class="board-pin"/>`,
  atoms3lite: `
    <rect x="43" y="15" width="74" height="70" rx="16" class="board-shell"/>
    <rect x="54" y="26" width="52" height="48" rx="10" class="board-face"/>
    <circle cx="80" cy="50" r="10" class="board-led"/><rect x="117" y="39" width="17" height="22" rx="3" class="board-metal"/>
    <circle cx="55" cy="76" r="3" class="board-port"/><circle cx="65" cy="76" r="3" class="board-port"/>`,
  sticks3: `
    <rect x="54" y="7" width="52" height="86" rx="16" class="board-shell"/>
    <rect x="61" y="17" width="38" height="55" rx="7" class="board-screen"/>
    <path d="M68 57l9-13 8 8 8-14" class="board-graph"/><circle cx="80" cy="82" r="5" class="board-button"/>
    <rect x="106" y="69" width="12" height="15" rx="3" class="board-metal"/>`,
  cardputeradv: `
    <rect x="14" y="18" width="132" height="66" rx="10" class="board-shell"/>
    <rect x="22" y="25" width="46" height="23" rx="3" class="board-screen"/>
    <g class="board-keys"><path d="M77 26h58M77 34h58M22 57h112M22 65h112M22 73h112"/><path d="M84 23v55M94 23v55M104 23v55M114 23v55M124 23v55M32 53v25M42 53v25M52 53v25M62 53v25M72 53v25"/></g>
    <rect x="146" y="38" width="11" height="21" rx="3" class="board-metal"/>`,
  tab5: `
    <rect x="17" y="10" width="126" height="80" rx="10" class="board-shell"/>
    <rect x="25" y="18" width="110" height="64" rx="4" class="board-screen"/>
    <path d="M34 68l22-22 15 12 18-24 36 34" class="board-graph"/>
    <circle cx="80" cy="14" r="2" class="board-camera"/><rect x="143" y="40" width="13" height="20" rx="3" class="board-metal"/>`,
  pdp11: `
    <rect x="18" y="18" width="124" height="64" rx="5" class="board-shell"/>
    <rect x="27" y="27" width="50" height="30" rx="2" class="board-face"/>
    <path d="M34 35h35M34 42h27M34 49h31" class="board-graph"/>
    <g class="board-keys"><path d="M88 30h43M88 39h43M88 48h43M88 57h43M27 67h104"/><path d="M98 25v38M109 25v38M120 25v38"/></g>`,
  ibmpc: `
    <rect x="21" y="14" width="80" height="58" rx="5" class="board-shell"/>
    <rect x="30" y="23" width="61" height="36" rx="2" class="board-screen"/>
    <path d="M38 49h28M38 42h39" class="board-graph"/><circle cx="91" cy="65" r="2" class="board-led"/>
    <rect x="16" y="72" width="90" height="8" rx="3" class="board-face"/>
    <path d="M115 26h25v47h-25zM120 35h15M120 43h15M120 64h15" class="board-keys"/>`,
};

function createBoardArtwork(kind) {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.classList.add("board-artwork");
  svg.setAttribute("viewBox", "0 0 160 100");
  svg.setAttribute("aria-hidden", "true");
  svg.innerHTML = boardArtworkPaths[kind] || '<rect x="28" y="22" width="104" height="56" rx="12" class="board-shell"/><path d="M45 40h70M45 50h70M45 60h45" class="board-keys"/>';
  return svg;
}

function catalogBoards(entries = exampleEntries()) {
  const boards = new Map();
  for (const entry of entries) for (const board of entry.boards) {
    const current = boards.get(board.name) || { ...board, count: 0 };
    current.count++;
    boards.set(board.name, current);
  }
  return [...boards.values()].sort((left, right) => left.name.localeCompare(right.name));
}

function catalogComputers(entries = exampleEntries()) {
  const computers = new Map();
  for (const entry of entries) for (const computer of entry.computers) {
    const current = computers.get(computer.name) || { ...computer, count: 0 };
    current.count++;
    computers.set(computer.name, current);
  }
  return [...computers.values()].sort((left, right) => left.name.localeCompare(right.name));
}

function catalogMachines(entries = exampleEntries()) {
  return [...catalogBoards(entries), ...catalogComputers(entries)];
}

function renderExampleBoards(entries = exampleEntries()) {
  const boards = catalogBoards(entries);
  const computers = catalogComputers(entries);
  const selected = elements.exampleBoardFilter.value;
  const choices = [];
  const all = document.createElement("button");
  all.type = "button"; all.className = "example-board-choice example-board-all"; all.dataset.exampleBoard = "";
  all.classList.toggle("selected", !selected); all.setAttribute("aria-pressed", String(!selected));
  const allIcon = document.createElement("span"); allIcon.className = "example-board-all-icon"; allIcon.textContent = "All";
  const allText = document.createElement("span"); allText.innerHTML = `<strong>All hardware</strong><small>${entries.length} examples</small>`;
  all.append(allIcon, allText);
  if (!isPhoneWorkspace()) choices.push(all);
  if (boards.length) {
    const heading = document.createElement("div"); heading.className = "example-machine-group"; heading.textContent = "Boards";
    choices.push(heading);
  }
  for (const board of boards) {
    const row = document.createElement("div"); row.className = "example-board-row";
    const button = document.createElement("button"); button.type = "button"; button.className = "example-board-choice";
    button.dataset.exampleBoard = board.name; button.classList.toggle("selected", selected === board.name);
    button.setAttribute("aria-pressed", String(selected === board.name));
    button.append(createBoardArtwork(board.artwork));
    const text = document.createElement("span");
    const name = document.createElement("strong"); name.textContent = board.name;
    const count = document.createElement("small"); count.textContent = `${board.count} example${board.count === 1 ? "" : "s"}`;
    text.append(name, count); button.append(text); row.append(button);
    if (board.docs) {
      const docs = document.createElement("a"); docs.href = board.docs; docs.target = "_blank"; docs.rel = "noreferrer";
      docs.className = "example-board-docs"; docs.textContent = "Hardware guide"; docs.setAttribute("aria-label", `${board.name} hardware guide`);
      row.append(docs);
    }
    choices.push(row);
  }
  if (computers.length) {
    const heading = document.createElement("div"); heading.className = "example-machine-group"; heading.textContent = "Computers";
    choices.push(heading);
  }
  for (const computer of computers) {
    const button = document.createElement("button"); button.type = "button"; button.className = "example-board-choice";
    button.dataset.exampleBoard = computer.name; button.classList.toggle("selected", selected === computer.name);
    button.setAttribute("aria-pressed", String(selected === computer.name));
    button.append(createBoardArtwork(computer.artwork));
    const text = document.createElement("span");
    const name = document.createElement("strong"); name.textContent = computer.name;
    const detail = document.createElement("small");
    const count = `${computer.count} example${computer.count === 1 ? "" : "s"}`;
    detail.textContent = computer.family ? `${computer.family} · ${count}` : count;
    text.append(name, detail); button.append(text); choices.push(button);
  }
  if (isPhoneWorkspace()) choices.push(all);
  elements.exampleBoardList.replaceChildren(...choices);
  if (isPhoneWorkspace() && selected) queueMicrotask(() => {
    elements.exampleBoardList.querySelector(".example-board-choice.selected")?.scrollIntoView({ block: "nearest", inline: "start" });
  });
}

function selectExampleBoard(event) {
  const button = event.target.closest("[data-example-board]");
  if (!button) return;
  exampleBoardSelectionTouched = true;
  elements.exampleBoardFilter.value = button.dataset.exampleBoard;
  const machine = catalogMachines().find((choice) => choice.name === button.dataset.exampleBoard);
  if (machine && elements.exampleTargetFilter.value && elements.exampleTargetFilter.value !== machine.target) {
    elements.exampleTargetFilter.value = "";
  }
  if (machine && machine.device !== "computer") prefetchExampleBoard();
  renderExampleBrowser();
}

function selectInitialExampleBoard(entries = exampleEntries()) {
  if (!isPhoneWorkspace() || exampleBoardSelectionTouched || elements.exampleBoardFilter.value) return;
  const boards = catalogBoards(entries);
  const current = boards.find((board) => board.target === selectedTarget?.name);
  const choice = current || boards[0];
  if (choice) elements.exampleBoardFilter.value = choice.name;
}

function maybeOpenMobileExamples() {
  if (parameters.has("help") || !isPhoneWorkspace() || elements.exampleDialog.open || document.querySelector("dialog[open]")) return;
  openExampleBrowser();
}

async function openExampleBrowser() {
  if (!elements.exampleDialog.open) elements.exampleDialog.showModal();
  elements.exampleResults.innerHTML = '<div class="example-empty">Loading examples…</div>';
  try {
    const catalog = standardCatalog || await (exampleCatalogPromise || standardCatalogPromise);
    if (!catalog) throw new Error("The example catalog is unavailable.");
    const entries = exampleEntries(catalog);
    fillExampleFilter(elements.exampleBoardFilter, [...new Set(entries.flatMap((entry) => entry.machines.map((machine) => machine.name)))].sort());
    fillExampleFilter(elements.exampleTargetFilter, [...new Set(entries.flatMap((entry) => entry.targets))].sort());
    if (selectedTarget) selectInitialExampleBoard(entries);
    renderExampleBrowser();
    queueMicrotask(() => elements.exampleSearch.focus());
  } catch (error) {
    elements.exampleResultCount.textContent = "Examples unavailable";
    const message = document.createElement("div"); message.className = "example-empty"; message.textContent = error.message || String(error);
    elements.exampleResults.replaceChildren(message);
  }
}

function fillExampleFilter(select, values) {
  const current = select.value;
  const first = select.options[0];
  select.replaceChildren(first, ...values.map((value) => Object.assign(document.createElement("option"), { value, textContent: value })));
  if (values.includes(current)) select.value = current;
}

function renderExampleBrowser() {
  if (!standardCatalog) return;
  const query = elements.exampleSearch.value.trim().toLowerCase();
  const board = elements.exampleBoardFilter.value;
  const target = elements.exampleTargetFilter.value;
  const entries = exampleEntries().filter((entry) => {
    const searchable = [entry.title, entry.slug, entry.importPath, entry.item.language || "go",
      ...entry.machines.flatMap((choice) => [choice.name, choice.family, choice.description]), ...entry.targets].join(" ").toLowerCase();
    return (!query || searchable.includes(query)) &&
      (!board || entry.machines.some((choice) => choice.name === board)) &&
      (!target || entry.targets.includes(target));
  });
  elements.exampleResults.scrollTop = 0;
  renderExampleBoards();
  elements.exampleResultCount.textContent = `${entries.length} example${entries.length === 1 ? "" : "s"}`;
  if (!entries.length) {
    elements.exampleResults.innerHTML = '<div class="example-empty"><strong>No examples found</strong><span>Clear a filter or try another search.</span></div>';
    return;
  }
  elements.exampleResults.replaceChildren(...entries.map((entry) => {
    const card = document.createElement("article"); card.className = "example-card";
    const presentation = examplePresentation(entry);
    const heading = document.createElement("div"); heading.className = "example-card-heading";
    const title = document.createElement("div"); title.className = "example-card-title";
    const category = document.createElement("span"); category.className = `example-category ${presentation.tone}`; category.textContent = presentation.category;
    const name = document.createElement("strong"); name.textContent = entry.title;
    title.append(category, name);
    const actions = document.createElement("div"); actions.className = "example-card-actions";
    const view = document.createElement("button"); view.type = "button"; view.className = "dialog-secondary"; view.dataset.exampleView = entry.importPath;
    view.disabled = !monaco; view.textContent = monaco ? "View code" : "Editor loading";
    const load = document.createElement("button"); load.type = "button"; load.className = "dialog-primary"; load.dataset.example = entry.importPath;
    load.disabled = !monaco; load.textContent = monaco ? "Use" : "Editor loading";
    actions.append(view, load); heading.append(title, actions);
    const metadata = document.createElement("div"); metadata.className = "example-metadata";
    const language = document.createElement("span"); language.className = "example-language"; language.textContent = entry.item.language === "c" ? "C" : "Go";
    const files = document.createElement("span"); files.textContent = `${entry.item.files.length} file${entry.item.files.length === 1 ? "" : "s"}`;
    metadata.append(language, files);
    const compatible = document.createElement("span"); compatible.className = "example-compatible";
    compatible.textContent = entry.machines.length ? entry.machines.map((choice) => choice.name).join(", ") : entry.targets.join(", ");
    metadata.append(compatible);
    const hardware = document.createElement("div"); hardware.className = "example-hardware";
    const hardwareLabel = document.createElement("span"); hardwareLabel.textContent = entry.computers.length ? "Runs on" : "Hardware"; hardware.append(hardwareLabel);
    const relevantBoards = board ? entry.boards.filter((choice) => choice.name === board) : entry.boards;
    const requirements = new Map();
    for (const choice of relevantBoards) for (const item of choice.hardware || []) {
      requirements.set(`${item.name}|${item.docs}|${item.optional}`, item);
    }
    if (entry.computers.length) {
      const system = document.createElement("strong");
      system.textContent = entry.computers.map((computer) => computer.description || computer.name).join(", ");
      hardware.append(system);
    } else if (!requirements.size) {
      const included = document.createElement("strong"); included.textContent = "Board only"; hardware.append(included);
    } else for (const item of requirements.values()) {
      const link = document.createElement("a"); link.href = item.docs; link.target = "_blank"; link.rel = "noreferrer";
      link.textContent = `${item.optional ? "Optional: " : ""}${item.name}`; hardware.append(link);
    }
    card.append(heading, hardware, metadata);
    return card;
  }));
}

const viewParameterNames = ["help", "source", "example", "file"];

function setViewDeepLink(kind = "", value = "", file = "") {
  if (!deepLinksReady || applyingDeepLink) return;
  const url = new URL(location.href);
  for (const name of viewParameterNames) url.searchParams.delete(name);
  if (kind && value) url.searchParams.set(kind, value);
  if (kind === "example" && file) url.searchParams.set("file", file);
  if (url.href !== location.href) history.pushState({ renvoView: true }, "", url);
}

function syncCodeDeepLink(name, editable) {
  if (editable) {
    setViewDeepLink();
    return;
  }
  const example = examplePreviewFiles.get(name);
  if (example) setViewDeepLink("example", example.importPath, example.file);
  else setViewDeepLink("source", name);
}

async function restoreDeepLink() {
  if (!monaco) return;
  applyingDeepLink = true;
  try {
    const current = new URLSearchParams(location.search);
    const help = current.get("help");
    const example = current.get("example");
    const source = current.get("source");
    if (help) {
      if (helpDocument(help)) openHelpPage(help);
      else elements.languageStatus.textContent = `Documentation not found: ${help}`;
      return;
    }
    if (example) {
      const entry = exampleEntries().find((candidate) => candidate.importPath === example);
      if (entry) await viewExample(entry, current.get("file") || "");
      else elements.languageStatus.textContent = `Example not found: ${example}`;
      return;
    }
    if (source) {
      const model = await ensureSourceModel(source);
      if (model) openFile(cleanPath(source));
      else elements.languageStatus.textContent = `Library source not found: ${source}`;
      return;
    }
    if (activeHelp || !isEditableFile(activeFile)) {
      const fallback = models.has(lastWorkspaceFile) ? lastWorkspaceFile : [...editableFiles][0];
      if (fallback) openFile(fallback);
    }
  } finally {
    applyingDeepLink = false;
  }
}

function isHelpTab(name) { return typeof name === "string" && name.startsWith("help:"); }
function helpTab(importPath) { return `help:${importPath}`; }
function helpImportPath(name) { return name.slice("help:".length); }

function helpDocuments(catalog = standardCatalog) {
  const documents = [];
  if (catalog?.builtins) documents.push(catalog.builtins);
  for (const [importPath, item] of Object.entries(catalog?.packages || {})) {
    if (item.docs) documents.push({ ...item.docs, importPath });
  }
  for (const [importPath, item] of Object.entries(catalog?.platforms || {})) {
    if (item.docs && !item.main) documents.push({ ...item.docs, importPath });
  }
  return documents.sort((left, right) => left.importPath.localeCompare(right.importPath));
}

function helpDocument(importPath) {
  return helpDocuments().find((item) => item.importPath === importPath);
}

function renderHelpCatalog(catalog = standardCatalog, query = "") {
  if (!catalog) return;
  const normalized = query.trim().toLowerCase();
  const documents = helpDocuments(catalog).filter((item) => {
    const declarations = [...(item.constants || []), ...(item.variables || []), ...(item.functions || []), ...(item.types || [])];
    return !normalized || [item.importPath, item.name, item.doc, ...declarations.flatMap((entry) => [entry.name, entry.doc])]
      .join(" ").toLowerCase().includes(normalized);
  });
  const search = document.createElement("label"); search.className = "help-search";
  const input = document.createElement("input"); input.type = "search"; input.placeholder = "Package or symbol";
  input.value = query; input.setAttribute("aria-label", "Search documentation");
  input.addEventListener("input", () => renderHelpCatalog(catalog, input.value));
  search.append(input);
  const children = [search];
  const appendGroup = (title, choices) => {
    if (!choices.length) return;
    const heading = document.createElement("div"); heading.className = "library-group"; heading.textContent = title; children.push(heading);
    for (const item of choices) {
      const button = document.createElement("button"); button.type = "button"; button.className = "help-package";
      button.textContent = item.importPath; button.title = item.doc || `Open ${item.importPath} documentation`;
      button.addEventListener("click", () => openHelpPage(item.importPath)); children.push(button);
    }
  };
  appendGroup("Language", documents.filter((item) => item.importPath === "builtin"));
  appendGroup("Standard library", documents.filter((item) => !item.importPath.startsWith("renvo.dev/") && item.importPath !== "builtin"));
  appendGroup("Device APIs", documents.filter((item) => item.importPath.startsWith("renvo.dev/device/")));
  appendGroup("Frameworks", documents.filter((item) => item.importPath.startsWith("renvo.dev/") && !item.importPath.startsWith("renvo.dev/device/")));
  if (children.length === 1) {
    const empty = document.createElement("span"); empty.className = "tree-loading"; empty.textContent = "No documentation found."; children.push(empty);
  }
  elements.helpTree.replaceChildren(...children);
  if (query) queueMicrotask(() => {
    const next = elements.helpTree.querySelector(".help-search input");
    next?.focus(); next?.setSelectionRange(query.length, query.length);
  });
}

function openHelpPage(importPath) {
  const page = helpDocument(importPath);
  if (!page) return;
  activeHelp = helpTab(importPath);
  if (!openFiles.includes(activeHelp)) openFiles.push(activeHelp);
  elements.editorHost.hidden = true;
  elements.helpView.hidden = false;
  elements.copyHelpPage.hidden = false;
  elements.copyToPlayground.hidden = true;
  elements.useBackend.hidden = true;
  elements.formatFile.disabled = true;
  elements.formatFile.hidden = true;
  document.querySelector("#search-project").hidden = true;
  elements.languageMode.textContent = "Docs";
  elements.cursorStatus.textContent = importPath;
  renderHelpPage(page);
  renderEditorTabs();
  setViewDeepLink("help", importPath);
  if (isPhoneWorkspace()) showMobileView("editor");
}

function helpAnchor(section, name) {
  return `doc-${section}-${name}`.toLowerCase().replace(/[^a-z0-9_-]+/g, "-");
}

function renderHelpPage(page) {
  const breadcrumb = document.createElement("div"); breadcrumb.className = "help-breadcrumb";
  const home = document.createElement("button"); home.type = "button"; home.textContent = "Renvo documentation";
  home.addEventListener("click", () => {
    elements.helpTree.hidden = false;
    document.querySelector("#help-heading").classList.remove("collapsed");
    document.querySelector("#help-heading .chevron").textContent = "⌄";
  });
  breadcrumb.append(home, document.createTextNode(` / ${page.importPath}`));
  const title = document.createElement("h1"); title.className = "help-package-title"; title.textContent = `package ${page.name}`;
  const path = document.createElement("div"); path.className = "help-import-path"; path.textContent = page.importPath;
  const overview = document.createElement("p"); overview.className = "help-overview";
  overview.textContent = page.doc || "This package has no overview documentation yet.";
  const nodes = [breadcrumb, title, path, overview];
  const sections = [
    ["Constants", page.constants || []], ["Variables", page.variables || []],
    ["Functions", page.functions || []], ["Types", page.types || []],
  ].filter(([, entries]) => entries.length);
  if (sections.length) {
    const indexTitle = document.createElement("h2"); indexTitle.textContent = "Index"; nodes.push(indexTitle);
    const index = document.createElement("ul"); index.className = "help-index";
    for (const [section, entries] of sections) for (const entry of entries) {
      const item = document.createElement("li"); const link = document.createElement("a");
      link.href = `#${helpAnchor(section, entry.name)}`; link.textContent = helpIndexLabel(section, entry); item.append(link); index.append(item);
      link.addEventListener("click", (event) => { event.preventDefault(); elements.helpView.querySelector(link.getAttribute("href"))?.scrollIntoView(); });
      for (const method of entry.methods || []) {
        const methodItem = document.createElement("li"); const methodLink = document.createElement("a");
        methodLink.href = `#${helpAnchor("method", `${entry.name}-${method.name}`)}`; methodLink.textContent = method.signature.split("\n")[0];
        methodLink.addEventListener("click", (event) => { event.preventDefault(); elements.helpView.querySelector(methodLink.getAttribute("href"))?.scrollIntoView(); });
        methodItem.append(methodLink); index.append(methodItem);
      }
    }
    nodes.push(index);
  }
  for (const [section, entries] of sections) {
    const heading = document.createElement("h2"); heading.textContent = section; nodes.push(heading);
    for (const entry of entries) {
      const declaration = renderHelpDeclaration(page, entry, helpAnchor(section, entry.name)); nodes.push(declaration);
      if (entry.methods?.length) {
        const methods = document.createElement("div"); methods.className = "help-methods";
        for (const method of entry.methods) methods.append(renderHelpDeclaration(page, method, helpAnchor("method", `${entry.name}-${method.name}`)));
        declaration.append(methods);
      }
    }
  }
  elements.helpView.replaceChildren(...nodes);
  elements.helpView.scrollTop = 0;
}

function helpIndexLabel(section, entry) {
  if (section === "Constants") return `const ${entry.name}`;
  if (section === "Variables") return `var ${entry.name}`;
  if (section === "Types") return `type ${entry.name}`;
  return entry.signature.split("\n")[0];
}

function renderHelpDeclaration(page, entry, id) {
  const declaration = document.createElement("section"); declaration.className = "help-declaration"; declaration.id = id;
  const headingRow = document.createElement("div"); headingRow.className = "help-declaration-heading";
  const heading = document.createElement("h3"); heading.textContent = entry.name;
  headingRow.append(heading);
  if (entry.file && entry.line) {
    const source = document.createElement("button"); source.type = "button"; source.className = "help-source-link";
    source.textContent = `${entry.file}:${entry.line}`; source.title = `Open ${entry.name} in ${entry.file}`;
    source.addEventListener("click", () => openHelpSource(page, entry)); headingRow.append(source);
  }
  const signature = document.createElement("pre"); signature.className = "help-signature";
  colorizeHelpSignature(signature, entry.signature);
  const docs = document.createElement("p"); docs.className = `help-doc${entry.doc ? "" : " help-empty-doc"}`;
  docs.textContent = entry.doc || "No documentation is available for this declaration.";
  declaration.append(headingRow, signature, docs); return declaration;
}

async function openHelpSource(page, entry) {
  const platform = standardCatalog?.platforms?.[page.importPath];
  let path = platform ? `${platform.root}/${entry.file}` : "";
  if (!path) {
    const name = page.importPath.startsWith("renvo.dev/std/") ? page.importPath.slice("renvo.dev/std/".length) : page.importPath;
    if (standardCatalog?.packages?.[name]) path = `std/${name}/${entry.file}`;
  }
  if (!path) return;
  try {
    const model = await ensureSourceModel(path);
    if (!model) throw new Error(`${path} is not in the browser bundle`);
    openFile(path);
    const position = { lineNumber: entry.line, column: 1 };
    editor.setPosition(position); editor.revealPositionInCenter(position); editor.focus();
  } catch (error) {
    elements.languageStatus.textContent = `Could not open source: ${error.message || error}`;
  }
}

function colorizeHelpSignature(element, source) {
  element.textContent = source;
  if (!monaco?.editor?.colorize) return;
  monaco.editor.colorize(source, "go", { tabSize: 4 }).then((html) => {
    if (element.isConnected && element.textContent === source) element.innerHTML = html;
  }).catch(() => {});
}

function helpPageText(page) {
  const output = [`# package ${page.name}`, "", `Import path: ${page.importPath}`, "", page.doc || "No package overview."];
  const sections = [["Constants", page.constants], ["Variables", page.variables], ["Functions", page.functions], ["Types", page.types]];
  for (const [title, entries] of sections) {
    if (!entries?.length) continue;
    output.push("", `## ${title}`);
    for (const entry of entries) {
      output.push("", `### ${entry.name}`, "", "```go", entry.signature, "```", "", entry.doc || "No documentation is available for this declaration.");
      for (const method of entry.methods || []) {
        output.push("", `#### ${method.name}`, "", "```go", method.signature, "```", "", method.doc || "No documentation is available for this declaration.");
      }
    }
  }
  return `${output.join("\n").trim()}\n`;
}

async function copyActiveHelpPage() {
  const page = activeHelp && helpDocument(helpImportPath(activeHelp));
  if (!page) return;
  try {
    await navigator.clipboard.writeText(helpPageText(page));
    elements.copyHelpPage.textContent = "Copied";
    elements.languageStatus.textContent = `Copied all ${page.importPath} documentation`;
    setTimeout(() => { elements.copyHelpPage.textContent = "Copy docs"; }, 1400);
  } catch (error) {
    elements.languageStatus.textContent = `Could not copy documentation: ${error.message || error}`;
  }
}

async function handleExampleAction(event) {
  const button = event.target.closest("[data-example], [data-example-view]");
  if (!button || !standardCatalog) return;
  const importPath = button.dataset.exampleView || button.dataset.example;
  const entry = exampleEntries().find((candidate) => candidate.importPath === importPath);
  if (!entry) return;
  if (button.dataset.exampleView) {
    elements.exampleDialog.close();
    await viewExample(entry);
  } else await useExample(entry, true);
}

function renderSidebarExamples(catalog = standardCatalog) {
  if (!catalog) return;
  const entries = exampleEntries(catalog).sort((left, right) => {
    const leftMatches = left.targets.includes(selectedTarget?.name) ? 0 : 1;
    const rightMatches = right.targets.includes(selectedTarget?.name) ? 0 : 1;
    return leftMatches - rightMatches || left.title.localeCompare(right.title);
  });
  elements.sidebarExampleCount.textContent = String(entries.length);
  const rows = entries.map((entry) => {
    const row = document.createElement("div"); row.className = "sidebar-example"; row.dataset.exampleTree = entry.importPath; row.setAttribute("role", "none");
    const folder = document.createElement("div"); folder.className = "sidebar-example-folder";
    const toggle = document.createElement("button"); toggle.type = "button"; toggle.className = "sidebar-example-toggle";
    toggle.setAttribute("role", "treeitem");
    const open = expandedExamples.has(entry.importPath);
    toggle.setAttribute("aria-expanded", String(open));
    const chevron = document.createElement("span"); chevron.className = "sidebar-example-chevron"; chevron.textContent = open ? "⌄" : "›";
    const details = document.createElement("div");
    const title = document.createElement("strong"); title.textContent = entry.title;
    const target = document.createElement("small");
    target.textContent = entry.targets.includes(selectedTarget?.name) ? targetDisplayName(selectedTarget) : entry.targets.map((name) => targetDisplayName(targetCatalog?.targets.find((item) => item.name === name) || { name })).join(", ");
    details.append(title, target);
    toggle.append(chevron, details);
    const use = document.createElement("button"); use.type = "button"; use.textContent = "Use";
    use.className = "sidebar-example-use";
    use.title = `Replace the current project with ${entry.title}`;
    use.addEventListener("click", () => useExample(entry, false));
    const files = document.createElement("div"); files.className = "sidebar-example-files"; files.hidden = !open;
    files.setAttribute("role", "group");
    files.replaceChildren(...entry.item.files.map((file) => {
      const button = document.createElement("button"); button.type = "button"; button.className = "sidebar-example-file";
      button.setAttribute("role", "treeitem"); button.title = `View ${entry.title} · ${file}`;
      button.classList.toggle("active", activeFile === `${entry.item.root}/${file}` && !activeHelp);
      const [iconText, iconClass] = fileIcon(file);
      const icon = document.createElement("span"); icon.className = iconClass; icon.textContent = iconText;
      const label = document.createElement("span"); label.textContent = file;
      button.append(icon, label);
      button.addEventListener("click", () => viewExample(entry, file));
      return button;
    }));
    toggle.addEventListener("click", () => {
      const opening = files.hidden;
      files.hidden = !opening;
      toggle.setAttribute("aria-expanded", String(opening));
      chevron.textContent = opening ? "⌄" : "›";
      if (opening) expandedExamples.add(entry.importPath); else expandedExamples.delete(entry.importPath);
    });
    folder.append(toggle, use); row.append(folder, files); return row;
  });
  const browse = document.createElement("button"); browse.type = "button"; browse.className = "sidebar-browser-action";
  browse.textContent = "Browse by hardware…"; browse.addEventListener("click", openExampleBrowser); rows.push(browse);
  elements.sidebarExamples.replaceChildren(...rows);
}

async function viewExample(entry, requestedFile = "") {
  try {
    elements.languageStatus.textContent = `Loading ${entry.title}…`;
    await loadStandardPackage(entry.importPath, standardCatalog);
    const sourceFiles = entry.item.files;
    for (const file of sourceFiles) examplePreviewFiles.set(`${entry.item.root}/${file}`, { importPath: entry.importPath, file });
    const file = sourceFiles.includes(requestedFile) ? requestedFile :
      sourceFiles.find((name) => name === "main.go" || name === "main.c") || sourceFiles[0];
    if (!file) throw new Error(`${entry.title} has no source files`);
    const path = `${entry.item.root}/${file}`;
    const model = await ensureSourceModel(path);
    if (!model) throw new Error(`${path} is not in the browser bundle`);
    openFile(path);
    expandedExamples.add(entry.importPath);
    openSidebarExamples();
    queueMicrotask(() => [...elements.sidebarExamples.querySelectorAll("[data-example-tree]")]
      .find((row) => row.dataset.exampleTree === entry.importPath)?.scrollIntoView({ block: "nearest" }));
    elements.languageStatus.textContent = `Viewing ${entry.title} · project unchanged`;
    if (isPhoneWorkspace()) showMobileView("editor");
  } catch (error) {
    showProjectError(error);
    elements.languageStatus.textContent = `Could not open ${entry.title}`;
  }
}

async function useExample(entry, fromDialog) {
  if (fromDialog) elements.exampleDialog.close();
  const accepted = await requestConfirmation({
    title: `Use ${entry.title}?`,
    message: "This replaces the current files. Saved snapshots and exported ZIP files will stay where they are.",
    accept: "Replace project",
    danger: false,
  });
  if (!accepted) { if (fromDialog) elements.exampleDialog.showModal(); return; }
  try {
    elements.languageStatus.textContent = `Loading ${entry.title}…`;
    const files = {};
    await Promise.all(entry.item.files.map(async (name) => {
      const asset = new URL(`module/${entry.item.root}/${name.split("/").map(encodeURIComponent).join("/")}`, standardCatalog.url);
      const response = await fetchAsset(asset);
      if (!response.ok) throw new Error(`Could not load ${name}: HTTP ${response.status}`);
      files[name] = await response.text();
    }));
    if (Object.keys(files).some((name) => name.endsWith(".go")) && !Object.hasOwn(files, "go.mod")) files["go.mod"] = initialFiles["go.mod"];
    const filteredTarget = elements.exampleTargetFilter.value;
    const target = filteredTarget && entry.targets.includes(filteredTarget) ? filteredTarget :
      entry.targets.includes(selectedTarget?.name) ? selectedTarget.name : entry.targets[0] || selectedTarget?.name;
    const targetDefinition = targetCatalog?.targets.find((candidate) => candidate.name === target);
    const active = entry.item.language === "c" && Object.hasOwn(files, "main.c") ? "main.c" :
      Object.hasOwn(files, "main.go") ? "main.go" : Object.keys(files).find((name) => /\.(?:go|c)$/.test(name)) || Object.keys(files)[0];
    const backendDefinition = Object.keys(files).find((name) => name.endsWith(".rtg"));
    replaceProject({
      name: entry.slug,
      language: entry.item.language || "go",
      files,
      activeFile: active,
      openFiles: [active],
      target,
      arenaSize: entry.item.arenaSize || "",
      backendRoots: backendDefinition ? [backendDefinition] : [],
      command: `-s -o ${targetDefinition?.output || "app.wasm"} .`,
    });
    if (backendDefinition) await useProjectBackend(backendDefinition);
    else elements.languageStatus.textContent = `${entry.title} is open`;
    renderSidebarExamples();
    if (isPhoneWorkspace()) showMobileView("editor");
  } catch (error) {
    showProjectError(error);
    elements.languageStatus.textContent = `Could not load ${entry.title}`;
  }
}

function renderLibraryCatalog(catalog) {
  const children = [];
  const appendGroup = (title, entries) => {
    const heading = document.createElement("div");
    heading.className = "library-group"; heading.textContent = title; children.push(heading);
    for (const [name, item] of entries) children.push(libraryPackage(catalog, name, item));
  };
  appendGroup("Standard library", Object.entries(catalog.packages || {}).sort(([left], [right]) => left.localeCompare(right)));
  if (catalog.libc?.length) {
    const heading = document.createElement("div");
    heading.className = "library-group"; heading.textContent = "C standard library"; children.push(heading);
    children.push(cLibraryDirectory(catalog, "include", "Headers"));
    children.push(cLibraryDirectory(catalog, "src", "Implementation"));
  }
  const platforms = Object.entries(catalog.platforms || {}).filter(([, item]) => !item.hidden);
  const frameworks = platforms.filter(([name, item]) => !name.includes("/examples/m5") && !item.main);
  if (frameworks.length) appendGroup("Frameworks", frameworks);
  elements.stdlibTree.replaceChildren(...children);
}

function libraryPackage(catalog, importPath, item, label = importPath.replace(/^renvo\.dev\//, "")) {
  const wrapper = document.createElement("div");
  if (item.board) wrapper.className = "library-board-package";
  const button = document.createElement("button");
  button.type = "button"; button.className = "stdlib-package";
  button.textContent = label;
  button.title = item.main ? `${importPath}. Click to use it as the app.` : importPath;
  const files = document.createElement("div");
  files.className = "stdlib-files"; files.hidden = true;
  button.addEventListener("click", async () => {
    const opening = files.hidden;
    files.hidden = !opening; button.classList.toggle("open", opening);
    if (item.main) setBuildPackage(item.root, item.target, button, item.language);
    if (!opening) return;
    if (files.childElementCount) {
      if (item.main) await openPackageEntry(item);
      const backendDefinition = item.files.find((file) => file.endsWith(".rtg"));
      if (item.main && backendDefinition) {
        const definition = `${item.root}/${backendDefinition}`;
        await ensureSourceModel(definition);
        await useProjectBackend(definition);
      }
      return;
    }
    button.disabled = true;
    try {
      await loadStandardPackage(importPath, catalog);
      const prefix = item.root || `std/${importPath}`;
      files.replaceChildren(...item.files.filter((file) => /\.(?:go|c|h|rtg)$/.test(file))
        .map((file) => librarySourceFile(`${prefix}/${file}`, file)));
      if (item.main) await openPackageEntry(item);
      const backendDefinition = item.files.find((file) => file.endsWith(".rtg"));
      if (item.main && backendDefinition) {
        const definition = `${item.root}/${backendDefinition}`;
        await ensureSourceModel(definition);
        await useProjectBackend(definition);
      }
    } catch (error) {
      files.replaceChildren(Object.assign(document.createElement("span"), { className: "tree-loading", textContent: error.message }));
    } finally {
      button.disabled = false;
    }
  });
  wrapper.append(button, files);
  return wrapper;
}

function cLibraryDirectory(catalog, directory, label) {
  const wrapper = document.createElement("div");
  wrapper.className = "library-board-package";
  const button = document.createElement("button");
  button.type = "button"; button.className = "stdlib-package"; button.textContent = label;
  button.title = `libc/${directory}`;
  const files = document.createElement("div");
  files.className = "stdlib-files"; files.hidden = true;
  button.addEventListener("click", async () => {
    const opening = files.hidden;
    files.hidden = !opening; button.classList.toggle("open", opening);
    if (!opening || files.childElementCount) return;
    button.disabled = true;
    try {
      await loadCLibrary(catalog);
      const prefix = `${directory}/`;
      files.replaceChildren(...catalog.libc.filter((file) => file.startsWith(prefix))
        .map((file) => librarySourceFile(`libc/${file}`, file.slice(prefix.length))));
    } catch (error) {
      files.replaceChildren(Object.assign(document.createElement("span"), { className: "tree-loading", textContent: error.message }));
    } finally {
      button.disabled = false;
    }
  });
  wrapper.append(button, files);
  return wrapper;
}

function librarySourceFile(path, name) {
  const entry = document.createElement("button");
  entry.type = "button"; entry.className = "stdlib-file"; entry.dataset.file = path;
  const [iconText, iconClass] = fileIcon(name);
  const icon = document.createElement("span"); icon.className = iconClass; icon.textContent = iconText;
  const label = document.createElement("span"); label.textContent = name;
  entry.append(icon, label);
  entry.addEventListener("click", async () => {
    await ensureSourceModel(path);
    openFile(path);
    if (isPhoneWorkspace()) showMobileView("editor");
  });
  return entry;
}

async function openPackageEntry(item) {
  const entryFile = item.language === "c" && item.files.find((file) => file === "main.c") ||
    item.files.find((file) => file === "main.go") ||
    item.files.find((file) => file.endsWith(".go"));
  if (!entryFile) return;
  const path = `${item.root}/${entryFile}`;
  await ensureSourceModel(path);
  openFile(path);
}

function setBuildPackage(root, target, button, language = "go") {
  activeBuildRoot = root === "." ? "." : `./${root}`;
  externalBuildLanguage = activeBuildRoot === "." ? "go" : language === "c" ? "c" : "go";
  if (target && target !== selectedTarget?.name) selectTarget(target, true);
  let args;
  try { args = splitArguments(elements.command.value); } catch { return; }
  if (args.length && (args[args.length - 1] === "." || args[args.length - 1].startsWith("./"))) args[args.length - 1] = activeBuildRoot;
  else args.push(activeBuildRoot);
  elements.command.value = args.join(" ");
  document.querySelectorAll(".stdlib-package").forEach((item) => item.classList.toggle("build-root", Boolean(button) && item === button));
  syncBuildScope();
  markBuildStale();
  saveFiles();
}

function activeBuildLanguage() {
  return activeBuildRoot === "." ? projectLanguage : externalBuildLanguage;
}

function syncBuildScope() {
  document.querySelectorAll(".stdlib-package").forEach((item) => {
    item.classList.toggle("build-root", item.dataset.root && activeBuildRoot === `./${item.dataset.root}`);
  });
}

async function ensureSourceModel(path) {
  const name = cleanPath(path);
  if (models.has(name)) return models.get(name);
  if (!standardCatalogPromise) return undefined;
  const catalog = await standardCatalogPromise;
  if (isCLibrarySourcePath(name, catalog)) {
    await loadCLibrary(catalog);
  } else {
    const importPath = sourceImportPath(name, catalog);
    if (!importPath) return undefined;
    await loadStandardPackage(importPath, catalog);
  }
  const source = stdlibFiles.get(name);
  if (!source) return undefined;
  const model = monaco.editor.createModel(decoder.decode(source), languageForFile(name), monaco.Uri.parse(`file:///${name}`));
  models.set(name, model);
  return model;
}


function formatElapsed(milliseconds) {
  return milliseconds < 1000 ? `${milliseconds.toFixed(1)} ms` : `${(milliseconds / 1000).toFixed(2)} s`;
}
