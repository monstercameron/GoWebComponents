package ui

// File I/O helpers (G14, G23): download bytes and pick a file without hand-rolled
// Blob/anchor/off-DOM-input/FileReader interop. The seams are swappable for
// tests; on the native/SSR build they are safe no-ops.

// PickedFile is one file chosen by the user.
type PickedFile struct {
	Name string
	Data []byte
}

var (
	downloadFn = defaultDownload
	pickFileFn = defaultPickFile
)

// Download saves bytes as a file with the given name and MIME type. No-op on the
// native/SSR build.
func Download(parseData []byte, parseName, parseMime string) {
	downloadFn(parseData, parseName, parseMime)
}

// PickFile opens the OS file picker (filtered by accept, e.g. "image/*" or
// ".csv") and calls onPick with the chosen file. No-op on native/SSR. onPick runs
// asynchronously after the user selects a file.
func PickFile(parseAccept string, parseOnPick func(PickedFile)) {
	if parseOnPick == nil {
		return
	}
	pickFileFn(parseAccept, parseOnPick)
}
