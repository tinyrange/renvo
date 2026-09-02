export function splitArguments(text) {
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

export function outputArgument(command) {
  const args = Array.isArray(command) ? command : splitArguments(command);
  let output = "";
  for (let index = 0; index < args.length; index++) {
    if (args[index] === "-o" && index + 1 < args.length) output = args[++index];
    else if (args[index].startsWith("-o=")) output = args[index].slice(3);
  }
  return output;
}

export function replaceOutput(command, output) {
  const args = splitArguments(command);
  const replaced = [];
  let found = false;
  for (let index = 0; index < args.length; index++) {
    if (args[index] === "-o") {
      if (index + 1 < args.length) index++;
      replaced.push("-o", output);
      found = true;
    } else if (args[index].startsWith("-o=")) {
      replaced.push("-o", output);
      found = true;
    } else {
      replaced.push(args[index]);
    }
  }
  if (!found) replaced.unshift("-o", output);
  return replaced.join(" ");
}

export function terminalRenvoArguments(input) {
  const args = [...input];
  const hasTarget = args.some((value, index) => value === "-t" || value === "--target" || value.startsWith("-t=") || value.startsWith("--target=") || index > 0 && ["-system", "--system"].includes(args[index - 1]));
  if (hasTarget || !args.length) return args;
  const command = args[0];
  if (["-h", "--help", "help", "-version", "--version", "version", "cc", "make", "backend"].includes(command)) return args;
  if (command === "run" || command === "test") return [command, "-t", "wasi/wasm32", ...args.slice(1)];
  return ["-t", "wasi/wasm32", ...args];
}
