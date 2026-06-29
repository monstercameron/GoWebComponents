//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/flags"
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// FeatureFlagsExample shows the flags package: seed a flag Set with UseRegistry, then read
// individual flags with UseFlag and branch the UI on them. Flipping a flag's Enabled value (or
// loading a remote Set) changes what renders, with no code change at the callsite.
func FeatureFlagsExample() ui.Node {
	flags.UseRegistry(flags.BuildSet(map[string]flags.Flag{
		"new-dashboard": {Enabled: true, Value: "v2", Reason: "staged rollout"},
		"beta-banner":   {Enabled: false},
	}, nil))

	dashboard := flags.UseFlag("new-dashboard", false)
	banner := flags.UseFlag("beta-banner", false)

	return Main(ClassStr("mx-auto max-w-xl space-y-4 p-6"),
		H1("Feature flags"),
		P(Textf("new-dashboard enabled: %v — variant %q", dashboard.Enabled(), dashboard.Value("v1"))),
		If(banner.Enabled(), P(ClassStr("rounded bg-amber-400/10 p-2"), Text("Beta banner is ON"))),
		If(!banner.Enabled(), P(ClassStr("text-slate-500"), Text("Beta banner is OFF (flip beta-banner to show it)"))),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(FeatureFlagsExample))
	exampleboot.WaitExampleRuntime()
}
