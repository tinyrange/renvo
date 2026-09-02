export const terminalCommands = ["help", "clear", "pwd", "ls", "cat", "file", "stat", "touch", "mkdir", "cp", "mv", "rm", "run", "flash", "renvo"];

const renvoChoices = ["cc", "make", "run", "test", "backend", "--help", "-t", "-o", "-s", "-mode", "-tags", "-arena-size", "-backend", "-system", "-emit-unit", "-emit-image"];

function cleanRelativePath(name) {
  return String(name || "").replace(/^\.\//, "").replace(/^\/+|\/+$/g, "");
}

export function terminalPathCompletions(fileNames, folderNames, prefix, fileFilter) {
  if (prefix.startsWith("/") || prefix.startsWith("../") || prefix.includes("/../")) return [];
  const displayRoot = prefix === "." || prefix.startsWith("./") ? "./" : "";
  const query = prefix === "." ? "" : displayRoot ? prefix.slice(2) : prefix;
  const slash = query.lastIndexOf("/");
  const directory = slash < 0 ? "" : query.slice(0, slash + 1);
  const fragment = slash < 0 ? query : query.slice(slash + 1);
  const files = [...new Set(fileNames.map(cleanRelativePath).filter(Boolean))];
  const directories = new Set();
  for (const original of [...folderNames, ...files.map((name) => name.includes("/") ? name.slice(0, name.lastIndexOf("/")) : "")]) {
    const path = cleanRelativePath(original);
    if (!path) continue;
    const parts = path.split("/");
    for (let count = 1; count <= parts.length; count++) directories.add(parts.slice(0, count).join("/"));
  }
  const candidates = new Set();
  for (const path of directories) {
    if (!path.startsWith(directory)) continue;
    const rest = path.slice(directory.length);
    if (!rest || rest.includes("/") || !rest.startsWith(fragment)) continue;
    if (fileFilter && !files.some((name) => name.startsWith(`${path}/`) && fileFilter(name))) continue;
    candidates.add(`${displayRoot}${directory}${rest}/`);
  }
  for (const path of files) {
    if (!path.startsWith(directory)) continue;
    const rest = path.slice(directory.length);
    if (!rest || rest.includes("/") || !rest.startsWith(fragment) || fileFilter && !fileFilter(path)) continue;
    candidates.add(`${displayRoot}${directory}${rest}`);
  }
  return [...candidates].sort();
}

export function terminalCompletionChoices(before, fileNames, folderNames, targets = []) {
  const match = /(?:^|\s)([^\s]*)$/.exec(before);
  const prefix = match?.[1] || "";
  const words = before.trim().split(/\s+/).filter(Boolean);
  const trailingSpace = /\s$/.test(before);
  const command = words[0] || "";
  const commandPosition = words.length === 0 || words.length === 1 && !trailingSpace;
  const paths = terminalPathCompletions(fileNames, folderNames, prefix);
  let choices;
  if (commandPosition) {
    choices = prefix.startsWith(".") || prefix.includes("/") ? paths : [...terminalCommands, ...paths];
  } else if (command === "renvo") {
    const previous = trailingSpace ? words.at(-1) || "" : words.at(-2) || "";
    choices = previous === "-t" || previous === "--target" ? targets : [...renvoChoices, ...paths];
  } else if (command === "run") {
    choices = terminalPathCompletions(fileNames, folderNames, prefix, (name) => name.endsWith(".wasm"));
  } else if (command === "flash") {
    const previous = trailingSpace ? words.at(-1) || "" : words.at(-2) || "";
    choices = previous === "--transport" ? ["webusb", "webserial"] : ["--help", "--transport"];
  } else if (["ls", "cat", "file", "stat", "touch", "mkdir", "cp", "mv", "rm"].includes(command)) {
    choices = paths;
  } else {
    choices = paths;
  }
  return { prefix, candidates: [...new Set(choices.filter((value) => value.startsWith(prefix)))].sort() };
}
