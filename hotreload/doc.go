// Package hotreload provides the public state-preserving hot reload API for
// GoWebComponents development builds.
//
// Typical app usage is:
//
//	hotreload.Enable()
//
// Or, when only selected atom ids should be exported:
//
//	hotreload.Configure(hotreload.Config{AtomIDs: []string{"session", "draft"}})
//
// To force an intentional state reset after an edit, change ResetKey:
//
//	hotreload.Configure(hotreload.Config{ResetKey: "layout-v2"})
//
// The standard tools/dev.ps1 and tools/dev.sh dev server wrappers use this
// bridge automatically when the application enables it.
package hotreload
