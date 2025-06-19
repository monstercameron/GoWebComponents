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
		NavBar,

		// Personal Hero Section
		PersonalHeroSection,

		// About Me Section
		PersonalAboutSection,

		// Skills & Technologies Section
		PersonalSkillsSection,

		// YouTube Channel Section
		PersonalYouTubeSection,

		// Featured Projects Section
		PortfolioProjectsSection,

		// GoWebComponents Showcase
		GWCShowcaseSection,

		// Interactive Examples Section
		GWCExamplesSection,

		// Why GoWebComponents Section
		WhyGoWebComponentsSection,

		// Contact Section
		ContactSection,

		// Footer
		FooterSection,

		// Scroll to Top Button
		ScrollToTopButton,
	)
}
