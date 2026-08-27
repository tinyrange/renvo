export const MAKEFILE_LANGUAGE_ID = "renvo-makefile";

const builtinVariables = ["CC", "CFLAGS", "RENVO"];
const automaticVariables = ["$@", "$<", "$^"];
const commandCompletions = [
  ["renvo cc", "Compile or link C files"],
  ["renvo run", "Compile and run a Renvo script"],
  ["renvo test", "Compile and run package tests"],
  ["renvo make", "Build another Makefile target"],
];
const optionCompletions = [
  ["-t linux/amd64", "Select Linux x86-64"], ["-c", "Emit an object file"],
  ["-o $@", "Write the current target"], ["-nostdinc", "Disable standard C includes"],
  ["-I.", "Search the project directory for headers"], ["-s", "Strip the output"],
];

export function registerMakefileLanguage(monaco, projectFiles = () => []) {
  if (monaco.languages.getLanguages().some((language) => language.id === MAKEFILE_LANGUAGE_ID)) return;
  monaco.languages.register({ id: MAKEFILE_LANGUAGE_ID, aliases: ["Renvo Makefile", "Makefile"], filenames: ["Makefile", "makefile"], extensions: [".mk"] });
  monaco.languages.setLanguageConfiguration(MAKEFILE_LANGUAGE_ID, {
    comments: { lineComment: "#" },
    autoClosingPairs: [{ open: "$(", close: ")" }, { open: "${", close: "}" }, { open: '"', close: '"' }, { open: "'", close: "'" }],
    surroundingPairs: [["$(", ")"], ["${", "}"], ['"', '"'], ["'", "'"]],
  });
  monaco.languages.setMonarchTokensProvider(MAKEFILE_LANGUAGE_ID, {
    defaultToken: "",
    tokenPostfix: ".makefile",
    tokenizer: {
      root: [
        [/^\s*#.*$/, "comment"],
        [/^(\.PHONY)(\s*)(:)/, ["keyword", "white", "delimiter"]],
        [/^([A-Za-z_][\w.-]*)(\s*)(\?=|\+=|:=|=)/, ["variable", "white", "operator"]],
        [/^([^\s:#=][^:#=]*?)(\s*)(:)/, ["type.identifier", "white", "delimiter"]],
        [/^(\t@?)(renvo)(\s+)(cc|run|test|make)\b/, ["white", "keyword", "white", "keyword"]],
        [/\$[<@^]/, "variable.predefined"],
        [/\$\([A-Za-z_][\w.-]*\)|\$\{[A-Za-z_][\w.-]*\}/, "variable"],
        [/(?:^|\s)-{1,2}[A-Za-z][\w-]*(?:=[^\s#]+)?/, "attribute.name"],
        [/'(?:\\.|[^\\'])+'|"(?:\\.|[^\\"])*"/, "string"],
        [/#.*$/, "comment"],
        [/[A-Za-z_][\w./-]*/, "identifier"],
        [/[0-9]+/, "number"],
        [/[?+:=]/, "operator"],
        [/[ \t\r\n]+/, "white"],
      ],
    },
  });
  monaco.languages.registerCompletionItemProvider(MAKEFILE_LANGUAGE_ID, {
    triggerCharacters: ["$", "(", "{", "-", " ", "\t"],
    provideCompletionItems(model, position) {
      return { suggestions: makefileCompletions(monaco, model, position, projectFiles()) };
    },
  });
}

export function makefileCompletions(monaco, model, position, projectFiles = []) {
  const source = model.getValue(), line = model.getLineContent(position.lineNumber), before = line.slice(0, position.column - 1);
  const variables = new Set(builtinVariables), targets = new Set();
  for (const sourceLine of source.split(/\r?\n/)) {
    const assignment = sourceLine.match(/^([A-Za-z_][\w.-]*)\s*(?:\?=|\+=|:=|=)/);
    if (assignment) variables.add(assignment[1]);
    const rule = sourceLine.match(/^([^\s:#=][^:#=]*?)\s*:/);
    if (rule && rule[1] !== ".PHONY") for (const target of rule[1].trim().split(/\s+/)) targets.add(target);
  }
  const variable = before.match(/\$(?:\(|\{)([A-Za-z_][\w.-]*)?$/);
  const word = model.getWordUntilPosition(position);
  const range = { startLineNumber: position.lineNumber, endLineNumber: position.lineNumber,
    startColumn: variable ? position.column - (variable[1] || "").length : word.startColumn, endColumn: position.column };
  if (variable) return [...variables].sort().map((name) => suggestion(monaco, name, "Makefile variable", name, range, "Variable"));
  if (/^\t@?[^#]*$/.test(before)) {
    const values = before.trimStart().replace(/^@/, "").startsWith("renvo ") ? optionCompletions : commandCompletions;
    return values.map(([label, detail]) => suggestion(monaco, label, detail, label, range, "Function"));
  }
  const suggestions = [
    suggestion(monaco, ".PHONY", "Always evaluate named targets", ".PHONY: ", range, "Keyword"),
    ...automaticVariables.map((name) => suggestion(monaco, name, "Automatic Makefile variable", name, range, "Variable")),
  ];
  for (const name of [...targets].sort()) suggestions.push(suggestion(monaco, name, "Makefile target", name, range, "Reference"));
  for (const name of [...new Set(projectFiles)].sort()) suggestions.push(suggestion(monaco, name, "Project file", name, range, "File"));
  return suggestions;
}

function suggestion(monaco, label, detail, insertText, range, kind) {
  return { label, detail, insertText, range, kind: monaco.languages.CompletionItemKind[kind] };
}
