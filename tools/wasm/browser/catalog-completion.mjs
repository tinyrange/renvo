export function catalogSelectorCompletions(catalog, selector, imports) {
  if (!selector) return undefined;
  const imported = imports.find((item) => item.name === selector.base);
  if (!imported) return undefined;
  const standardName = imported.importPath.replace(/^renvo\.dev\/std\//, "");
  const item = catalog?.packages?.[standardName] || catalog?.platforms?.[imported.importPath];
  if (!item?.docs) return undefined;
  const prefix = selector.prefix;
  const records = [
    ...(item.docs.constants || []).map((entry) => ({ ...entry, kind: "constant" })),
    ...(item.docs.variables || []).map((entry) => ({ ...entry, kind: "variable" })),
    ...(item.docs.functions || []).map((entry) => ({ ...entry, kind: "function" })),
    ...(item.docs.types || []).map((entry) => ({ ...entry, kind: "type" })),
  ];
  const found = new Map();
  for (const record of records) {
    if (!record.name || record.name.includes(",") || !record.name.startsWith(prefix) || found.has(record.name)) continue;
    found.set(record.name, record);
  }
  return { packageName: selector.base, importPath: imported.importPath, prefix, items: [...found.values()] };
}
