//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/dom"
)

// DocsWebsite composes Earl Cameron's personal website and GoWebComponents showcase.
// Combines personal branding, project portfolio, technical documentation, and
// interactive examples into a cohesive single-page application experience.
func DocsWebsite(props Attrs) *Element {
	return dom.Div(
		Attrs{"class": "min-h-screen bg-[#0a0a0a] text-white selection:bg-blue-500/30"},

		// Navigation with dark mode and smooth scrolling
		dom.CreateElement(NavBar, nil),

		// Personal branding and introduction
		dom.CreateElement(PersonalHeroSection, nil),
		dom.CreateElement(PersonalAboutSection, nil),
		dom.CreateElement(PersonalSkillsSection, nil),
		dom.CreateElement(PersonalYouTubeSection, nil),

		// Project showcase and portfolio
		dom.CreateElement(PortfolioProjectsSection, nil),

		// GoWebComponents feature highlights
		dom.CreateElement(GWCShowcaseSection, nil),
		dom.CreateElement(GWCExamplesSection, nil),
		dom.CreateElement(WhyGoWebComponentsSection, nil),

		// Contact and site utilities
		dom.CreateElement(ContactSection, nil),
		dom.CreateElement(FooterSection, nil),
		dom.CreateElement(ScrollToTopButton, nil),
	)
}
