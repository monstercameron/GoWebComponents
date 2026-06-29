package atlas

import "strings"

type atlasShellToast struct {
	Title  string
	Detail string
	Tone   string
}

var atlasShellToastBus = make(chan atlasShellToast, 16)

func dispatchAtlasShellToast(parseToast atlasShellToast) {
	if strings.TrimSpace(parseToast.Title) == "" {
		return
	}
	select {
	case atlasShellToastBus <- parseToast:
	default:
	}
}
