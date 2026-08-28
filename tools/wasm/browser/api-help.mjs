function clean(value) {
  return String(value || "").replaceAll("\\", "/").replace(/^\/+/, "").replace(/\/+/g, "/");
}

export function helpAnchor(section, name) {
  return `doc-${section}-${name}`.toLowerCase().replace(/[^a-z0-9_-]+/g, "-");
}

export function catalogHelpPages(catalog) {
  const pages = [];
  if (catalog?.builtins) pages.push({ ...catalog.builtins, sourceRoot: "" });
  for (const [importPath, item] of Object.entries(catalog?.packages || {})) {
    if (item.docs) pages.push({ ...item.docs, importPath, sourceRoot: `std/${importPath}` });
  }
  for (const [importPath, item] of Object.entries(catalog?.platforms || {})) {
    if (item.docs && !item.main) pages.push({ ...item.docs, importPath, sourceRoot: clean(item.root) });
  }
  return pages.sort((left, right) => left.importPath.localeCompare(right.importPath));
}

function entryNames(entry) {
  return String(entry?.name || "").split(",").map((name) => name.trim()).filter(Boolean);
}

export function findHelpSymbol(page, symbol) {
  if (!page || !symbol) return undefined;
  const sections = [["constants", page.constants], ["variables", page.variables], ["functions", page.functions], ["types", page.types]];
  for (const [section, entries] of sections) {
    for (const entry of entries || []) {
      if (entryNames(entry).includes(symbol)) return { entry, anchor: helpAnchor(section, entry.name) };
      for (const method of entry.methods || []) {
        if (method.name === symbol) return { entry: method, anchor: helpAnchor("method", `${entry.name}-${method.name}`) };
      }
    }
  }
  return undefined;
}

export function findAPIHelpReference(catalog, definitionPath, line, symbol) {
  const path = clean(definitionPath);
  const pages = catalogHelpPages(catalog).filter((page) => page.sourceRoot &&
    (path === page.sourceRoot || path.startsWith(`${page.sourceRoot}/`)));
  for (const page of pages) {
    const relative = path.slice(page.sourceRoot.length).replace(/^\//, "");
    const declarations = [
      ...(page.constants || []).map((entry) => ["constants", entry]),
      ...(page.variables || []).map((entry) => ["variables", entry]),
      ...(page.functions || []).map((entry) => ["functions", entry]),
      ...(page.types || []).map((entry) => ["types", entry]),
    ];
    const named = [];
    for (const [section, entry] of declarations) {
      if (clean(entry.file) === relative && entryNames(entry).includes(symbol)) {
        named.push({ importPath: page.importPath, anchor: helpAnchor(section, entry.name), line: Number(entry.line) });
      }
      for (const method of entry.methods || []) {
        if (clean(method.file) === relative && method.name === symbol) {
          named.push({ importPath: page.importPath, anchor: helpAnchor("method", `${entry.name}-${method.name}`), line: Number(method.line) });
        }
      }
    }
    const exact = named.find((reference) => reference.line === Number(line));
    if (exact) return { importPath: exact.importPath, anchor: exact.anchor };
    if (named.length === 1) return { importPath: named[0].importPath, anchor: named[0].anchor };
    if (named.length > 1 && Number(line) > 0) {
      named.sort((left, right) => Math.abs(left.line - Number(line)) - Math.abs(right.line - Number(line)));
      return { importPath: named[0].importPath, anchor: named[0].anchor };
    }
    for (const [section, entry] of declarations) {
      if (clean(entry.file) === relative && Number(entry.line) === Number(line)) {
        return { importPath: page.importPath, anchor: helpAnchor(section, entry.name) };
      }
      for (const method of entry.methods || []) {
        if (clean(method.file) === relative && Number(method.line) === Number(line)) {
          return { importPath: page.importPath, anchor: helpAnchor("method", `${entry.name}-${method.name}`) };
        }
      }
    }
    return { importPath: page.importPath, anchor: "" };
  }
  const builtin = catalog?.builtins && findHelpSymbol(catalog.builtins, symbol);
  return builtin ? { importPath: "builtin", anchor: builtin.anchor } : undefined;
}

export function installAPIHelpAction(monaco, editor, openAtCursor) {
  return editor.addAction({
    id: "renvo.openApiDocumentation",
    label: "Open API Documentation",
    contextMenuGroupId: "navigation",
    contextMenuOrder: 1.6,
    precondition: "editorLangId == go",
    async run(activeEditor) {
      const model = activeEditor.getModel();
      const position = activeEditor.getPosition();
      if (model && position) await openAtCursor(model, position);
    },
  });
}
