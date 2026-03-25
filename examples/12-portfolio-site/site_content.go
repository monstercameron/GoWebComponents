package main

import "strings"

const (
	portfolioHomeRoute     = "/"
	portfolioDocsRoute     = "/docs"
	portfolioCatchAllRoute = "*"
)

type portfolioStat struct {
	Number string
	Label  string
}

type portfolioProject struct {
	Title        string
	Subtitle     string
	Description  string
	Technologies []string
	Link         string
	Featured     bool
}

func portfolioHeroStats() []portfolioStat {
	return []portfolioStat{
		{Number: "100%", Label: "Go Powered"},
		{Number: "0", Label: "JavaScript Required"},
		{Number: "∞", Label: "Possibilities"},
		{Number: "1", Label: "Revolutionary Framework"},
	}
}

func portfolioProjects() []portfolioProject {
	return []portfolioProject{
		{
			Title:       "GoWebComponents",
			Subtitle:    "Revolutionary Frontend Framework",
			Description: "A React-like framework for building web applications entirely in Go using WebAssembly. Features component-based architecture, state management, and zero JavaScript required.",
			Technologies: []string{
				"Go",
				"WebAssembly",
				"React-like",
				"State Management",
				"Frontend Framework",
			},
			Link:     "https://github.com/monstercameron/GoWebComponents",
			Featured: true,
		},
		{
			Title:       "gRPC Tunnel",
			Subtitle:    "Native gRPC-over-WebSocket Solution",
			Description: "Innovative project that tunnels native gRPC calls over WebSocket connections, enabling full gRPC communication from browsers using WebAssembly without gRPC-Web limitations.",
			Technologies: []string{
				"Go",
				"gRPC",
				"WebSocket",
				"WebAssembly",
				"Protobuf",
			},
			Link:     "https://github.com/monstercameron/grpc-tunnel",
			Featured: true,
		},
	}
}

func featuredPortfolioProjects() []portfolioProject {
	parseProjects := portfolioProjects()
	parseFeatured := make([]portfolioProject, 0, len(parseProjects))
	for _, parseProject := range parseProjects {
		if parseProject.Featured {
			parseFeatured = append(parseFeatured, parseProject)
		}
	}
	return parseFeatured
}

func findPortfolioProject(parseTitle string) (portfolioProject, bool) {
	parseTrimmedTitle := strings.TrimSpace(strings.ToLower(parseTitle))
	for _, parseProject := range portfolioProjects() {
		if strings.ToLower(parseProject.Title) == parseTrimmedTitle {
			return parseProject, true
		}
	}
	return portfolioProject{}, false
}

func totalPortfolioTechnologies(parseProjects []portfolioProject) int {
	parseTotal := 0
	for _, parseProject := range parseProjects {
		parseTotal += len(parseProject.Technologies)
	}
	return parseTotal
}
