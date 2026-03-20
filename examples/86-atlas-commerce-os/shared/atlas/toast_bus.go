package atlas

import "strings"

type atlasShellToast struct {
	Title  string
	Detail string
	Tone   string
}

var atlasShellToastBus = make(chan atlasShellToast, 16)

func dispatchAtlasShellToast(toast atlasShellToast) {
	if strings.TrimSpace(toast.Title) == "" {
		return
	}
	select {
	case atlasShellToastBus <- toast:
	default:
	}
}
