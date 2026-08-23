export const RTG_LANGUAGE_ID = "renvo-rtg";

const declarationKeywords = [
  "definition", "unit", "implements", "system", "arch", "extend", "abi",
  "runtime", "format", "target", "ir", "go", "backend",
];

const blockKeywords = [
  "registers", "register_class", "register_group", "locations", "conditions",
  "address_space", "forms", "instructions", "bind", "reject", "relocation",
  "segments", "operations", "syscall", "code", "data",
];

const valueKeywords = [
  "true", "false", "little", "big", "native_v1", "structured32",
];

const goKeywords = [
  "break", "case", "chan", "const", "continue", "default", "defer", "else",
  "fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
  "map", "package", "range", "return", "select", "struct", "switch", "type",
  "var", "any", "bool", "byte", "complex64", "complex128", "error", "float32",
  "float64", "int", "int8", "int16", "int32", "int64", "rune", "string",
  "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "nil", "iota",
];

export function registerRTGLanguage(monaco) {
  if (monaco.languages.getLanguages().some((language) => language.id === RTG_LANGUAGE_ID)) return;
  monaco.languages.register({ id: RTG_LANGUAGE_ID, aliases: ["Renvo Target Generation", "RTG"], extensions: [".rtg"] });
  monaco.languages.setLanguageConfiguration(RTG_LANGUAGE_ID, {
    comments: { lineComment: "#", blockComment: ["/*", "*/"] },
    brackets: [["{", "}"], ["[", "]"], ["(", ")"]],
    autoClosingPairs: [
      { open: "{", close: "}" }, { open: "[", close: "]" }, { open: "(", close: ")" },
      { open: "\"", close: "\"", notIn: ["string", "comment"] },
    ],
    surroundingPairs: [["{", "}"], ["[", "]"], ["(", ")"], ["\"", "\""]],
  });
  monaco.languages.setMonarchTokensProvider(RTG_LANGUAGE_ID, {
    defaultToken: "",
    tokenPostfix: ".rtg",
    declarationKeywords,
    blockKeywords,
    valueKeywords,
    goKeywords,
    tokenizer: {
      root: [
        [/^(\s*)(@)(import)(\s+)("(?:[^"\\]|\\.)*")/, ["white", "keyword.directive", "keyword.directive", "white", "string"]],
        [/^\s*#.*$/, "comment"],
        [/\/\*/, "comment", "@comment"],
        [/\/\/.*$/, "comment"],
        [/"/, "string.quote", "@string"],
        [/'(?:\\.|[^\\'])+'/, "string"],
        [/0[xX][0-9A-Fa-f]+/, "number.hex"],
        [/0[bB][01]+/, "number.binary"],
        [/(?:\d+\.\d*|\.\d+)(?:[eE][+-]?\d+)?/, "number.float"],
        [/\d+/, "number"],
        [/[A-Za-z_][\w]*(?=\s*\()/, { cases: { "@goKeywords": "keyword", "@default": "identifier.function" } }],
        [/[A-Za-z_][\w]*/, { cases: {
          "@declarationKeywords": "keyword",
          "@blockKeywords": "type",
          "@valueKeywords": "constant.language",
          "@goKeywords": "keyword",
          "@default": "identifier",
        } }],
        [/[{}\[\]()]/, "@brackets"],
        [/[=,:;.@]/, "delimiter"],
        [/[+\-*\/%&|^!<>]+/, "operator"],
        [/[ \t\r\n]+/, "white"],
      ],
      comment: [
        [/[^/*]+/, "comment"], [/\/\*/, "comment"], [/\*\//, "comment", "@pop"], [/[/*]/, "comment"],
      ],
      string: [
        [/[^\\"]+/, "string"], [/\\./, "string.escape"], [/"/, "string.quote", "@pop"],
      ],
    },
  });
}
