function words(text) {
  const result = []; let word = "", quote = "", escaped = false;
  const flush = () => { if (word) result.push(word); word = ""; };
  for (const ch of text) {
    if (escaped) { word += ch; escaped = false; continue; }
    if (ch === "\\" && quote !== "'") { escaped = true; continue; }
    if (quote) { if (ch === quote) quote = ""; else word += ch; continue; }
    if (ch === "'" || ch === '"') quote = ch;
    else if (ch === " " || ch === "\t") flush();
    else word += ch;
  }
  if (escaped || quote) throw new Error("unterminated quote or escape in recipe");
  flush(); return result;
}

function expand(text, variables, automatic = {}) {
  return text.replace(/\$\$|\$[<@^]|\$\([^)]+\)|\$\{[^}]+\}/g, (match) => {
    if (match === "$$") return "$";
    const name = match.length === 2 ? match[1] : match.slice(2, -1);
    return automatic[name] ?? variables.get(name) ?? "";
  });
}

function uncomment(text) {
  let quote = "";
  for (let i = 0; i < text.length; i++) {
    if (quote) { if (text[i] === quote) quote = ""; }
    else if (text[i] === "'" || text[i] === '"') quote = text[i];
    else if (text[i] === "#") return text.slice(0, i).trim();
  }
  return text.trim();
}

export function parseMakefile(source) {
  const variables = new Map(), rules = [], phony = new Set(); let current = null, defaultTarget = "";
  const lines = source.replace(/\r\n?/g, "\n").split("\n");
  for (let index = 0; index < lines.length; index++) {
    const raw = lines[index].trimEnd(), line = index + 1;
    if (!raw || raw.trimStart().startsWith("#")) continue;
    if (raw.startsWith("\t")) {
      if (!current) throw makeError(line, "recipe has no preceding rule");
      const recipe = raw.slice(1).trim(); if (recipe) current.recipes.push({ text: recipe, line });
      continue;
    }
    current = null;
    const assignment = raw.match(/^([^\s:]+)\s*(:=|\?=|\+=|=)\s*(.*)$/);
    if (assignment) {
      const [, name, operator, input] = assignment, value = expand(uncomment(input), variables);
      if (operator === "?=" && variables.has(name)) continue;
      variables.set(name, operator === "+=" && variables.get(name) ? `${variables.get(name)} ${value}` : value);
      continue;
    }
    const colon = raw.indexOf(":");
    if (colon < 0) throw makeError(line, "expected a variable assignment or target rule");
    const targets = words(expand(raw.slice(0, colon).trim(), variables));
    const prerequisites = words(expand(uncomment(raw.slice(colon + 1)), variables));
    if (!targets.length) throw makeError(line, "target list is malformed");
    if (targets.length === 1 && targets[0] === ".PHONY") { prerequisites.forEach((name) => phony.add(name)); continue; }
    current = { targets, prerequisites, recipes: [], line, phony: false }; rules.push(current);
    if (!defaultTarget && !targets[0].startsWith(".")) defaultTarget = targets[0];
  }
  for (const rule of rules) rule.phony = rule.targets.some((target) => phony.has(target));
  return { variables, rules, defaultTarget: defaultTarget || rules[0]?.targets[0] || "" };
}

export function planMakefile(file, requested = [], exists = null) {
  const commands = [], visiting = new Set(), done = new Set();
  const targets = requested.length ? requested : [file.defaultTarget];
  if (!targets[0]) throw makeError(0, "Makefile contains no targets");
  const findRule = (target) => file.rules.find((rule) => rule.targets.includes(target));
  const visit = (target) => {
    if (done.has(target)) return;
    if (visiting.has(target)) throw makeError(0, `dependency cycle contains ${target}`);
    const rule = findRule(target);
    if (!rule) { if (exists?.(target)) return; throw makeError(0, `no rule to make target ${target}`); }
    visiting.add(target);
    for (const dependency of rule.prerequisites) {
      if (findRule(dependency)) visit(dependency);
      else if (exists && !exists(dependency)) throw makeError(rule.line, `no rule to make prerequisite ${dependency} for ${target}`);
    }
    visiting.delete(target);
    const automatic = { "@": target, "<": rule.prerequisites[0] || "", "^": rule.prerequisites.join(" ") };
    for (const recipe of rule.recipes) {
      let text = expand(recipe.text, file.variables, automatic), quiet = false;
      if (text.startsWith("@")) { quiet = true; text = text.slice(1).trim(); }
      const args = words(text); if (!args.length) continue;
      if (args[0] !== "renvo") throw makeError(recipe.line, "recipes must invoke renvo directly");
      commands.push({ args, text, quiet, target, line: recipe.line });
    }
    done.add(target);
  };
  targets.forEach(visit); return commands;
}

function makeError(line, message) { return Object.assign(new Error(message), { line }); }
