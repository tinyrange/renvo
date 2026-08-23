export const C_LANGUAGE_ID = "renvo-c";

const keywords = [
  "_Alignas", "_Alignof", "_Atomic", "_Bool", "_Complex", "_Generic",
  "_Imaginary", "_Noreturn", "_Static_assert", "_Thread_local", "alignas",
  "alignof", "asm", "auto", "break", "case", "const", "continue",
  "default", "do", "else", "enum", "extern", "for", "goto", "if",
  "inline", "register", "restrict", "return", "sizeof", "static",
  "static_assert", "struct", "switch", "thread_local", "typedef", "typeof",
  "typeof_unqual", "union", "volatile", "while",
];

const typeKeywords = [
  "char", "double", "float", "int", "long", "short", "signed", "unsigned",
  "void", "bool", "true", "false", "nullptr",
];

const operators = [
  "=", ">", "<", "!", "~", "?", ":", "==", "<=", ">=", "!=", "&&",
  "||", "++", "--", "+", "-", "*", "/", "&", "|", "^", "%", "<<",
  ">>", "+=", "-=", "*=", "/=", "&=", "|=", "^=", "%=", "<<=", ">>=",
  "->", ".", "...",
];

export function registerCLanguage(monaco) {
  if (monaco.languages.getLanguages().some((language) => language.id === C_LANGUAGE_ID)) return;
  monaco.languages.register({ id: C_LANGUAGE_ID, aliases: ["C", "c"], extensions: [".c", ".h"] });
  monaco.languages.setLanguageConfiguration(C_LANGUAGE_ID, {
    comments: { lineComment: "//", blockComment: ["/*", "*/"] },
    brackets: [["{", "}"], ["[", "]"], ["(", ")"]],
    autoClosingPairs: [
      { open: "{", close: "}" }, { open: "[", close: "]" }, { open: "(", close: ")" },
      { open: "\"", close: "\"", notIn: ["string", "comment"] },
      { open: "'", close: "'", notIn: ["string", "comment"] },
    ],
    surroundingPairs: [["{", "}"], ["[", "]"], ["(", ")"], ["\"", "\""], ["'", "'"]],
    folding: { markers: { start: /^\s*#\s*(?:pragma\s+)?region\b/, end: /^\s*#\s*(?:pragma\s+)?endregion\b/ } },
  });
  monaco.languages.setMonarchTokensProvider(C_LANGUAGE_ID, {
    defaultToken: "",
    tokenPostfix: ".c",
    keywords,
    typeKeywords,
    operators,
    symbols: /[=><!~?:&|+\-*\/%^\.]+/,
    escapes: /\\(?:[abfnrtv\\?"']|x[0-9A-Fa-f]+|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8}|[0-7]{1,3})/,
    tokenizer: {
      root: [
        [/^(\s*#\s*include\s*)(<[^>\n]+>)/, ["keyword.directive", "string.include"]],
        [/^(\s*#\s*include\s*)("(?:[^"\\]|\\.)*")/, ["keyword.directive", "string.include"]],
        [/^\s*#\s*[A-Za-z_]\w*/, "keyword.directive"],
        [/\/\*/, "comment", "@comment"],
        [/\/\/.*$/, "comment"],
        [/"/, "string.quote", "@string"],
        [/'(?:\\.|[^\\'])+'/, "string"],
        [/'/, "string.invalid"],
        [/[ \t\r\n]+/, "white"],
        [/[A-Za-z_]\w*(?=\s*\()/, { cases: { "@keywords": "keyword", "@typeKeywords": "type", "@default": "identifier.function" } }],
        [/[A-Za-z_]\w*/, { cases: { "@keywords": "keyword", "@typeKeywords": "type", "@default": "identifier" } }],
        [/0[xX][0-9A-Fa-f](?:'?[0-9A-Fa-f])*[uUlL]*/, "number.hex"],
        [/0[bB][01](?:'?[01])*[uUlL]*/, "number.binary"],
        [/(?:\d(?:'?\d)*)?\.\d(?:'?\d)*(?:[eE][+-]?\d+)?[fFlL]?/, "number.float"],
        [/\d(?:'?\d)*(?:[eE][+-]?\d+)[fFlL]?/, "number.float"],
        [/\d(?:'?\d)*[uUlL]*/, "number"],
        [/[{}\[\]()]/, "@brackets"],
        [/[;,]/, "delimiter"],
        [/@symbols/, { cases: { "@operators": "operator", "@default": "delimiter" } }],
      ],
      comment: [
        [/[^/*]+/, "comment"], [/\/\*/, "comment"], [/\*\//, "comment", "@pop"], [/[/*]/, "comment"],
      ],
      string: [
        [/[^\\"]+/, "string"], [/@escapes/, "string.escape"], [/\\./, "string.escape.invalid"], [/"/, "string.quote", "@pop"],
      ],
    },
  });
}
