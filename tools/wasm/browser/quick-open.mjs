const modePrefixes = new Map([
  [">", "commands"],
  ["@", "symbols"],
  ["#", "definitions"],
  [":", "line"],
  ["?", "help"],
]);

export function quickOpenQuery(value = "") {
  const prefix = value.charAt(0);
  const mode = modePrefixes.get(prefix) || "files";
  return { mode, query: (mode === "files" ? value : value.slice(1)).trim(), prefix: mode === "files" ? "" : prefix };
}

export function fuzzyScore(value, query) {
  const text = value.toLowerCase();
  const search = query.trim().toLowerCase();
  if (!search) return 0;
  const direct = text.indexOf(search);
  if (direct >= 0) return direct * 4 + Math.max(0, text.length - search.length) / 100;
  let position = -1;
  let gap = 0;
  for (const character of search) {
    const next = text.indexOf(character, position + 1);
    if (next < 0) return Infinity;
    if (position >= 0) gap += next - position - 1;
    position = next;
  }
  return 100 + gap * 3 + position / 100;
}

export function filterQuickOpenItems(items, query, limit = 80) {
  const terms = query.trim().split(/\s+/).filter(Boolean);
  return items
    .map((item, index) => ({
      item,
      index,
      score: terms.reduce((total, term) => total + fuzzyScore(`${item.label} ${item.detail || ""}`, term), 0),
    }))
    .filter(({ score }) => Number.isFinite(score))
    .sort((left, right) => left.score - right.score || left.index - right.index)
    .slice(0, limit)
    .map(({ item }) => item);
}

export function catalogFileItems(catalog = {}) {
  const files = [];
  for (const [importPath, item] of Object.entries(catalog.packages || {})) {
    for (const file of item.files || []) files.push({ path: `std/${importPath}/${file}`, source: "Standard library" });
  }
  for (const [importPath, item] of Object.entries(catalog.platforms || {})) {
    if (!item.root) continue;
    const source = item.main && !item.hidden ? "Sample" : "Platform library";
    for (const file of item.files || []) files.push({ path: `${item.root}/${file}`, source, importPath });
  }
  for (const file of catalog.libc || []) files.push({ path: `libc/${file}`, source: "C standard library" });
  return files;
}
