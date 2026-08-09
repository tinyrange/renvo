//go:build renvo && android

package graphics

func (w *Window) OpenFileDialog(options FileDialogOptions) (string, bool) {
	return "", false
}
func (w *Window) SaveFileDialog(options FileDialogOptions) (string, bool) {
	return "", false
}
func (w *Window) SelectFolderDialog(options FolderDialogOptions) (string, bool) {
	return "", false
}
