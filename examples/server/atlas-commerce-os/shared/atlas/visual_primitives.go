package atlas

import "strings"

type atlasVisualSurface string

const (
	atlasVisualSurfacePublic   atlasVisualSurface = "public"
	atlasVisualSurfaceInternal atlasVisualSurface = "internal"
)

func atlasVisualSurfaceName(parseSurface atlasVisualSurface) string {
	parseName := strings.TrimSpace(string(parseSurface))
	if parseName == "" {
		return string(atlasVisualSurfacePublic)
	}
	return parseName
}

func atlasRootSurfaceClass(parseSurface atlasVisualSurface) string {
	switch atlasVisualSurfaceName(parseSurface) {
	case string(atlasVisualSurfaceInternal):
		return "min-h-screen bg-[radial-gradient(circle_at_top,rgba(34,211,238,0.12),transparent_28%),linear-gradient(180deg,rgba(2,6,23,1),rgba(15,23,42,1))] text-white"
	default:
		return "min-h-screen bg-[radial-gradient(circle_at_top,rgba(245,158,11,0.16),transparent_22%),linear-gradient(180deg,rgba(7,10,18,1),rgba(13,18,30,1)_42%,rgba(7,10,18,1))] text-stone-100"
	}
}

func atlasMainShellClass(parseSurface atlasVisualSurface) string {
	switch atlasVisualSurfaceName(parseSurface) {
	case string(atlasVisualSurfaceInternal):
		return "mx-auto grid w-full max-w-6xl gap-8 px-6 pb-12 pt-6 lg:px-10"
	default:
		return "mx-auto grid w-full max-w-7xl gap-8 px-5 pb-12 pt-6 sm:px-6 lg:px-10"
	}
}

func publicHeaderShellClass() string {
	return "sticky top-0 z-20 border-b border-white/10 bg-[rgba(7,10,18,0.86)] backdrop-blur-xl"
}

func publicHeaderInnerClass() string {
	return "mx-auto flex w-full max-w-7xl flex-col gap-4 px-5 py-4 sm:px-6 lg:flex-row lg:items-center lg:justify-between lg:px-10"
}

func publicNavLinkClass(isActive bool) string {
	if isActive {
		return "rounded-full border border-amber-300/70 bg-amber-300/12 px-4 py-2.5 text-sm font-semibold text-amber-100 shadow-[0_12px_28px_rgba(245,158,11,0.14)]"
	}
	return "rounded-full border border-white/12 bg-white/6 px-4 py-2.5 text-sm font-medium text-stone-300 shadow-[0_10px_25px_rgba(0,0,0,0.16)] transition hover:border-amber-300/60 hover:bg-white/10 hover:text-white"
}

func publicMobileNavLinkClass(isActive bool) string {
	if isActive {
		return "flex items-center justify-between rounded-[1.35rem] border border-amber-300/60 bg-amber-300/12 px-4 py-4 text-left text-sm font-semibold text-amber-100 shadow-[0_12px_28px_rgba(245,158,11,0.14)]"
	}
	return "flex items-center justify-between rounded-[1.35rem] border border-white/10 bg-white/5 px-4 py-4 text-left text-sm font-medium text-stone-200 transition hover:border-amber-300/45 hover:bg-white/10 hover:text-white"
}

func publicHeroSurfaceClass() string {
	return "rounded-[1.9rem] border border-white/10 bg-[linear-gradient(145deg,rgba(13,18,30,0.96),rgba(18,25,40,0.9))] p-6 shadow-[0_28px_65px_rgba(0,0,0,0.22)]"
}

func publicGlassCardClass() string {
	return "rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"
}

func publicMetricSurfaceClass() string {
	return "rounded-[1.5rem] border border-white/10 bg-[linear-gradient(180deg,rgba(255,255,255,0.06),rgba(255,255,255,0.03))] p-5 shadow-[0_14px_32px_rgba(0,0,0,0.18)]"
}

func publicCatalogCardClass() string {
	return "group grid gap-5 rounded-[1.9rem] border border-white/10 bg-[linear-gradient(180deg,rgba(15,20,33,0.98),rgba(10,15,24,0.95))] p-5 shadow-[0_20px_48px_rgba(0,0,0,0.22)] transition duration-200 hover:-translate-y-1 hover:border-amber-300/35 hover:bg-[linear-gradient(180deg,rgba(18,24,38,1),rgba(12,17,28,0.98))] hover:shadow-[0_28px_65px_rgba(0,0,0,0.28)]"
}

func publicCatalogControlShellClass() string {
	return "grid gap-4 " + publicGlassCardClass() + " lg:grid-cols-[minmax(0,1.2fr)_repeat(3,minmax(0,0.75fr))_auto] lg:items-end"
}

func publicFormControlClass() string {
	return "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"
}

func publicSignalPillClass() string {
	return "rounded-full border border-white/12 bg-white/8 px-4 py-2 text-sm font-semibold text-stone-200"
}

func atlasVisualEfficiencyRules() []string {
	return []string{
		"public shell backgrounds stay on the route root instead of repeating heavy gradients in nested panels",
		"hero, metric, feature, and catalog card surfaces use shared class helpers before route-specific additions",
		"internal list routes keep one table shell with semantic table children and no wrapper rows inside tbody",
		"mobile and reduced-motion reviews may simplify glow, hover lift, blur, and drawer travel without changing DOM shape",
	}
}
