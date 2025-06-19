//go:build js && wasm
// +build js,wasm

package website

import (
	. "github.com/monstercameron/GoWebComponents/fiber"
)

// Earl Cameron's Personal Website & GoWebComponents Showcase

// DocsWebsite is the main entrypoint for Earl Cameron's personal website
func DocsWebsite(props Attrs) *Element {
	return Div(
		Attrs{"class": "min-h-screen bg-gradient-to-br from-gray-50 to-blue-50"},
		NavBar(nil),

		// Personal Hero Section
		PersonalHeroSection(nil),

		// About Me Section
		PersonalAboutSection(nil),

		// Skills & Technologies Section
		PersonalSkillsSection(nil),

		// YouTube Channel Section
		PersonalYouTubeSection(nil),

		// Featured Projects Section
		PortfolioProjectsSection(nil),

		// GoWebComponents Showcase
		GWCShowcaseSection(nil),

		// Interactive Examples Section
		GWCExamplesSection(nil),

		// Why GoWebComponents Section (moved after Mini Apps Gallery)
		WhyGoWebComponentsSection(nil),

		// Contact Section
		ContactSection(nil),

		// Footer
		FooterSection(nil),

		// Scroll to Top Button
		ScrollToTopButton(nil),
	)
}

// Components are now imported from separate files
