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
