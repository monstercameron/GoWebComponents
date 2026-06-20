//go:build !(js && wasm)

package ui

func defaultDownload(parseData []byte, parseName, parseMime string)    {}
func defaultPickFile(parseAccept string, parseOnPick func(PickedFile)) {}
