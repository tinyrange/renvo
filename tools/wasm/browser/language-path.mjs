const sourceRoots = ["std", "device", "examples", "forms"];

// cleanLanguagePath maps language-service filesystem paths back to the model
// names used by Monaco. The service may return absolute paths while catalog
// models always use repository-relative names.
export function cleanLanguagePath(name, models = new Map(), initialFiles = {}) {
  let value = name.replaceAll("\\", "/").replace(/^\.\//, "");
  for (const root of sourceRoots) {
    if (value.startsWith(`${root}/`)) return value;
    const at = value.indexOf(`/${root}/`);
    if (at >= 0) return value.slice(at + 1);
  }
  while (value.startsWith("../")) value = value.slice(3);
  value = value.replace(/^\/+/, "");
  if (models.has(value)) return value;
  const base = value.split("/").pop();
  return models.has(base) || Object.hasOwn(initialFiles, base) ? base : value;
}

// sourceImportPath maps a definition source back to the catalog package that
// can materialize it in Monaco. Longest-root matching avoids a parent package
// claiming a nested package source.
export function sourceImportPath(name, catalog) {
  const value = cleanLanguagePath(name);
  if (value.startsWith("std/")) {
    const slash = value.lastIndexOf("/");
    if (slash > 4) return value.slice(4, slash);
  }
  let importPath = "";
  let rootLength = -1;
  for (const [candidate, item] of Object.entries(catalog.platforms || {})) {
    const root = item.root || "";
    if (root.length > rootLength && (value === root || value.startsWith(`${root}/`))) {
      importPath = candidate;
      rootLength = root.length;
    }
  }
  return importPath;
}
