//go:build renvo && darwin && !ios && arm64

package graphics

func configureCocoaPanel(panel int, title, initialDirectory string) {
	if panel == 0 {
		return
	}
	if title != "" {
		objcMsg1(panel, selector("setTitle:"), cocoaString(title))
	}
	if initialDirectory != "" {
		url := objcMsg1(objcGetClass("NSURL"), selector("fileURLWithPath:"), cocoaString(initialDirectory))
		if url != 0 {
			objcMsg1(panel, selector("setDirectoryURL:"), url)
		}
	}
}

func cocoaPanelPath(panel int) (string, bool) {
	if panel == 0 || objcMsg0(panel, selector("runModal")) != 1 {
		return "", false
	}
	url := objcMsg0(panel, selector("URL"))
	if url == 0 {
		return "", false
	}
	path := cocoaUTF8(objcMsg0(url, selector("path")))
	return path, path != ""
}

func (w *Window) OpenFileDialog(options FileDialogOptions) (string, bool) {
	if w == nil || w.closed {
		return "", false
	}
	panel := objcMsg0(objcGetClass("NSOpenPanel"), selector("openPanel"))
	configureCocoaPanel(panel, options.Title, options.InitialDirectory)
	objcMsg1(panel, selector("setCanChooseFiles:"), 1)
	objcMsg1(panel, selector("setCanChooseDirectories:"), 0)
	objcMsg1(panel, selector("setAllowsMultipleSelection:"), 0)
	objcMsg1(panel, selector("setResolvesAliases:"), 1)
	return cocoaPanelPath(panel)
}

func (w *Window) SaveFileDialog(options FileDialogOptions) (string, bool) {
	if w == nil || w.closed {
		return "", false
	}
	panel := objcMsg0(objcGetClass("NSSavePanel"), selector("savePanel"))
	configureCocoaPanel(panel, options.Title, options.InitialDirectory)
	objcMsg1(panel, selector("setCanCreateDirectories:"), 1)
	if options.DefaultName != "" {
		objcMsg1(panel, selector("setNameFieldStringValue:"), cocoaString(options.DefaultName))
	}
	return cocoaPanelPath(panel)
}

func (w *Window) SelectFolderDialog(options FolderDialogOptions) (string, bool) {
	if w == nil || w.closed {
		return "", false
	}
	panel := objcMsg0(objcGetClass("NSOpenPanel"), selector("openPanel"))
	configureCocoaPanel(panel, options.Title, options.InitialDirectory)
	objcMsg1(panel, selector("setCanChooseFiles:"), 0)
	objcMsg1(panel, selector("setCanChooseDirectories:"), 1)
	objcMsg1(panel, selector("setAllowsMultipleSelection:"), 0)
	objcMsg1(panel, selector("setCanCreateDirectories:"), 1)
	objcMsg1(panel, selector("setResolvesAliases:"), 1)
	return cocoaPanelPath(panel)
}
