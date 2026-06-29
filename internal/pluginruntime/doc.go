// Package pluginruntime implements the internal framework-owned plugin kernel.
//
// This package is distinct from the public companion-host package at
// github.com/monstercameron/GoWebComponents/plugin. The public companion host
// remains the app-owned extension surface. pluginruntime owns framework-managed
// lifecycle, guarded execution, service resolution, and contribution
// registration for trusted internal plugins.
package pluginruntime
