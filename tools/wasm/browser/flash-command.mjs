export function flashCommand(transport) {
  return `flash --transport ${transport}`;
}

export function parseFlashArguments(args) {
  let transport = "";
  for (let index = 0; index < args.length; index++) {
    const argument = args[index];
    if (argument === "--help" || argument === "-h") return { help: true, transport: "" };
    if (argument === "--transport") {
      if (index + 1 >= args.length) throw new Error("flash: --transport requires webusb or webserial");
      transport = args[++index];
    } else if (argument.startsWith("--transport=")) {
      transport = argument.slice("--transport=".length);
    } else {
      throw new Error(`flash: unsupported argument ${argument}`);
    }
  }
  if (transport && transport !== "webusb" && transport !== "webserial") {
    throw new Error(`flash: unsupported transport ${transport}`);
  }
  return { help: false, transport };
}
