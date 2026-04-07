// Package plugin provides an explicit, application-owned plugin host for
// companion packages and example integrations.
//
// The package is a supported companion API. It does not grant privileged
// runtime access, and it does not replace the internal framework-owned plugin
// kernel used for deep devtools and service interposition. Companion plugins
// remain app-owned integrations over a Host instance.
package plugin
