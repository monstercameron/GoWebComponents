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
	projects := portfolioProjects()
	featured := make([]portfolioProject, 0, len(projects))
	for _, project := range projects {
		if project.Featured {
			featured = append(featured, project)
		}
	}
	return featured
}

func findPortfolioProject(title string) (portfolioProject, bool) {
	trimmedTitle := strings.TrimSpace(strings.ToLower(title))
	for _, project := range portfolioProjects() {
		if strings.ToLower(project.Title) == trimmedTitle {
			return project, true
		}
	}
	return portfolioProject{}, false
}

func totalPortfolioTechnologies(projects []portfolioProject) int {
	total := 0
	for _, project := range projects {
		total += len(project.Technologies)
	}
	return total
}
