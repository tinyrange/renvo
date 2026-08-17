export function installEditorOpener(monaco, host) {
  return monaco.editor.registerEditorOpener({
    async openCodeEditor(_source, resource, selectionOrPosition) {
      const name = host.cleanPath(resource.path);
      const model = await host.ensureSourceModel(name);
      if (!model) return false;

      host.openFile(name);
      if (selectionOrPosition) {
        if (typeof selectionOrPosition.startLineNumber === "number") {
          host.editor.setSelection(selectionOrPosition);
          host.editor.revealRangeInCenter(selectionOrPosition);
        } else {
          host.editor.setPosition(selectionOrPosition);
          host.editor.revealPositionInCenter(selectionOrPosition);
        }
      }
      host.editor.focus();
      return true;
    },
  });
}
