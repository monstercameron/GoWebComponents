//go:build js && wasm
// +build js,wasm

package website
import (
	"github.com/monstercameron/GoWebComponents/dom"
)

// DocsWebsite composes Earl Cameron's personal website and GoWebComponents showcase.
// Combines personal branding, project portfolio, technical documentation, and
// interactive examples into a cohesive single-page application experience.
func DocsWebsite(props Attrs) *Element {
	return dom.Div(
		Attrs{"class": "min-h-screen bg-gradient-to-br from-gray-50 to-blue-50"},

		// Navigation with dark mode and smooth scrolling
		NavBar,

		// Personal branding and introduction
		PersonalHeroSection,
		PersonalAboutSection,
		PersonalSkillsSection,
		PersonalYouTubeSection,

		// Project showcase and portfolio
		PortfolioProjectsSection,

		// GoWebComponents feature highlights
		GWCShowcaseSection,
		GWCExamplesSection,
		WhyGoWebComponentsSection,

		// Contact and site utilities
		ContactSection,
		FooterSection,
		ScrollToTopButton,
	)
}




