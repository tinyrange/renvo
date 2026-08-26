export function buildReadiness({ compilerReady, editorReady, building, state, currentRevision, validatedRevision, readyLabel = "Build" }) {
  if (building) return { ready: false, label: "Building…", title: "Build in progress" };
  if (!compilerReady || !editorReady) return { ready: false, label: "Build", title: "Compiler is still loading" };
  if (state === "checking") return { ready: false, label: "Checking…", title: "Checking whether the current project will build" };
  if (state === "success" && currentRevision === validatedRevision) {
    return { ready: true, label: readyLabel, title: `${readyLabel} (Ctrl+Enter)` };
  }
  if (state === "failure" && currentRevision === validatedRevision) {
    return { ready: false, label: "Fix errors", title: "Fix the reported errors before building" };
  }
  return { ready: false, label: "Checking…", title: "The project has changed and must be checked again" };
}
