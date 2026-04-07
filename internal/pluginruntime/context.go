package pluginruntime

import "fmt"

// Context exposes the kernel-facing API available to one running plugin.
type Context interface {
	ResolveService(ServiceKey) (any, bool)
	RegisterContribution(ContributionRegistration) error
	RegisterCleanup(CleanupFunc)
	KernelInfo() KernelInfo
}

type pluginContext struct {
	getKernel   *Kernel
	getPluginID string
}

// ResolveService resolves one typed service from the running kernel.
func (parseContext pluginContext) ResolveService(parseKey ServiceKey) (any, bool) {
	if parseContext.getKernel == nil {
		return nil, false
	}
	return parseContext.getKernel.ResolveService(parseKey)
}

// RegisterContribution registers one immutable contribution for the active plugin.
func (parseContext pluginContext) RegisterContribution(parseRegistration ContributionRegistration) error {
	if parseContext.getKernel == nil {
		return fmt.Errorf("pluginruntime: kernel is unavailable")
	}
	return parseContext.getKernel.registerContribution(parseContext.getPluginID, parseRegistration)
}

// RegisterCleanup registers one plugin-owned cleanup callback.
func (parseContext pluginContext) RegisterCleanup(parseCleanup CleanupFunc) {
	if parseContext.getKernel == nil || parseCleanup == nil {
		return
	}
	parseContext.getKernel.registerCleanup(parseContext.getPluginID, parseCleanup)
}

// KernelInfo reports stable kernel metadata to the active plugin.
func (parseContext pluginContext) KernelInfo() KernelInfo {
	if parseContext.getKernel == nil {
		return KernelInfo{}
	}
	return parseContext.getKernel.Info()
}
