export function targetCapabilities(target) {
  return {
    runsInBrowser: Boolean(target?.runnable),
    flashable: target?.device === "esp32" || target?.device === "rp2",
  };
}

export function hasDownloadableOutput(target) {
  return Boolean(target?.output);
}

export function targetCapabilityTags(target) {
  const capabilities = targetCapabilities(target);
  const tags = [];
  if (capabilities.runsInBrowser) tags.push({ name: "browser", label: "Runs in browser" });
  if (capabilities.flashable) tags.push({ name: "flash", label: "Flash over USB" });
  return tags;
}

export function targetCapabilityHint(target) {
  if (!target) return "Choose where the project will run.";
  const capabilities = targetCapabilities(target);
  const output = target.output || "the compiled file";
  if (target.device === "rp2") return `Load through the resident USB monitor or download ${output}.`;
  if (capabilities.flashable) return `Flash this board over USB or download ${output}.`;
  if (capabilities.runsInBrowser) {
    return `Run this target in the browser or download ${output}.`;
  }
  return `Build and download ${output} for this target.`;
}
