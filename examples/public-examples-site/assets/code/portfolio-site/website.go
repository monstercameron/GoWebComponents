//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// DocsWebsite composes Earl Cameron's personal website and GoWebComponents showcase.
// Combines personal branding, project portfolio, technical documentation, and
// interactive examples into a cohesive single-page application experience.
func DocsWebsite(parseProps Attrs) *Element {
	return html.Div(html.Props{Class: "min-h-screen bg-[#0a0a0a] text-white selection:bg-blue-500/30"},
		ui.CreateElement(NavBar),
		ui.CreateElement(PersonalHeroSection),
		ui.CreateElement(PersonalAboutSection),
		ui.CreateElement(PersonalSkillsSection),
		ui.CreateElement(PersonalYouTubeSection),
		ui.CreateElement(PortfolioProjectsSection),
		ui.CreateElement(GWCShowcaseSection),
		ui.CreateElement(GWCExamplesSection),
		ui.CreateElement(WhyGoWebComponentsSection),
		ui.CreateElement(ContactSection),
		ui.CreateElement(FooterSection),
		ui.CreateElement(ScrollToTopButton),
	)
}
