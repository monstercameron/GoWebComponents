package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func renderPortfolioSnapshot() ui.Node {
	parseStats := portfolioHeroStats()
	parseProjects := featuredPortfolioProjects()

	parseStatNodes := make([]ui.Node, 0, len(parseStats))
	for _, parseStat := range parseStats {
		parseStat2 := parseStat
		parseStatNodes = append(parseStatNodes, html.Div(html.Props{Class: "stat-card"},
			html.Strong(html.Props{}, html.Text(parseStat2.Number)),
			html.Span(html.Props{}, html.Text(parseStat2.Label)),
		))
	}

	parseProjectNodes := make([]ui.Node, 0, len(parseProjects))
	for _, parseProject := range parseProjects {
		parseProject2 := parseProject
		parseProjectNodes = append(parseProjectNodes, html.Article(html.Props{Class: "project-card"},
			html.H2(html.Props{}, html.Text(parseProject2.Title)),
			html.P(html.Props{}, html.Text(parseProject2.Subtitle)),
			html.P(html.Props{}, html.Text(parseProject2.Description)),
		))
	}

	return html.Main(html.Props{ID: "portfolio-snapshot"},
		html.H1(html.Props{}, html.Text("Earl Cameron")),
		html.Section(html.Props{ID: "hero-stats"}, parseStatNodes...),
		html.Section(html.Props{ID: "projects"}, parseProjectNodes...),
	)
}

func renderPortfolioProjectsGridSnapshot() ui.Node {
	parseProjects := portfolioProjects()
	parseProjectNodes := make([]ui.Node, 0, len(parseProjects))
	for _, parseProject := range parseProjects {
		parseTechnologyNodes := make([]ui.Node, 0, len(parseProject.Technologies))
		for _, parseTech := range parseProject.Technologies {
			parseTechnologyNodes = append(parseTechnologyNodes, html.Span(html.Props{Class: "tech-chip"}, html.Text(parseTech)))
		}

		parseProjectNodes = append(parseProjectNodes, html.Article(html.Props{Class: "project-card"},
			html.H2(html.Props{}, html.Text(parseProject.Title)),
			html.P(html.Props{}, html.Text(parseProject.Subtitle)),
			html.P(html.Props{}, html.Text(parseProject.Description)),
			html.Div(html.Props{Class: "technology-stack"}, parseTechnologyNodes...),
			html.A(html.Props{Href: parseProject.Link}, html.Text("View Project")),
		))
	}

	return html.Section(html.Props{ID: "portfolio-project-grid"}, parseProjectNodes...)
}

func TestPortfolioHeroStatsStable(parseT *testing.T) {
	parseStats := portfolioHeroStats()
	if len(parseStats) != 4 {
		parseT.Fatalf("expected 4 portfolio hero stats, got %d", len(parseStats))
	}
	if parseStats[0].Number != "100%" || parseStats[0].Label != "Go Powered" {
		parseT.Fatalf("unexpected first stat: %+v", parseStats[0])
	}
	if parseStats[1].Number != "0" || parseStats[1].Label != "JavaScript Required" {
		parseT.Fatalf("unexpected second stat: %+v", parseStats[1])
	}
}

func TestPortfolioProjectsLookupAndTechnologyTotals(parseT *testing.T) {
	parseProjects := portfolioProjects()
	if len(parseProjects) != 2 {
		parseT.Fatalf("expected 2 portfolio projects, got %d", len(parseProjects))
	}
	if parseTotal := totalPortfolioTechnologies(parseProjects); parseTotal != 10 {
		parseT.Fatalf("expected 10 total technology tags, got %d", parseTotal)
	}

	parseProject, parseOk := findPortfolioProject("GoWebComponents")
	if !parseOk {
		parseT.Fatal("expected GoWebComponents project lookup to succeed")
	}
	if !parseProject.Featured {
		parseT.Fatalf("expected GoWebComponents to be featured, got %+v", parseProject)
	}

	parseProject, parseOk = findPortfolioProject("grpc tunnel")
	if !parseOk {
		parseT.Fatal("expected case-insensitive gRPC Tunnel lookup to succeed")
	}
	if !strings.Contains(parseProject.Description, "WebSocket") {
		parseT.Fatalf("expected gRPC Tunnel description to mention WebSocket, got %q", parseProject.Description)
	}
	if _, parseOk2 := findPortfolioProject("missing"); parseOk2 {
		parseT.Fatal("expected missing project lookup to fail")
	}
}

func TestFeaturedPortfolioProjectsAreStableAndOrdered(parseT *testing.T) {
	parseFeatured := featuredPortfolioProjects()
	if len(parseFeatured) != 2 {
		parseT.Fatalf("expected 2 featured portfolio projects, got %d", len(parseFeatured))
	}
	if parseFeatured[0].Title != "GoWebComponents" || parseFeatured[1].Title != "gRPC Tunnel" {
		parseT.Fatalf("unexpected featured project ordering: %+v", parseFeatured)
	}
	for _, parseProject := range parseFeatured {
		if !parseProject.Featured {
			parseT.Fatalf("expected featured project list to contain only featured projects, got %+v", parseProject)
		}
	}
}

func TestPortfolioProjectMetadataValidity(parseT *testing.T) {
	parseProjects := portfolioProjects()
	parseSeenTitles := make(map[string]bool, len(parseProjects))
	for _, parseProject := range parseProjects {
		if parseProject.Title == "" || parseProject.Subtitle == "" || parseProject.Description == "" {
			parseT.Fatalf("expected complete project metadata, got %+v", parseProject)
		}
		if !strings.HasPrefix(parseProject.Link, "https://") {
			parseT.Fatalf("expected secure project link, got %q", parseProject.Link)
		}
		if len(parseProject.Technologies) == 0 {
			parseT.Fatalf("expected technologies for project %+v", parseProject)
		}
		parseKey := strings.ToLower(parseProject.Title)
		if parseSeenTitles[parseKey] {
			parseT.Fatalf("expected unique project title, duplicate %q", parseProject.Title)
		}
		parseSeenTitles[parseKey] = true
	}

	if portfolioHomeRoute != "/" || portfolioDocsRoute != "/docs" || portfolioCatchAllRoute != "*" {
		parseT.Fatalf("unexpected route constants: home=%q docs=%q catchAll=%q", portfolioHomeRoute, portfolioDocsRoute, portfolioCatchAllRoute)
	}
}

func TestPortfolioSnapshotRenderToString(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(renderPortfolioSnapshot())
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	parseChecks := []string{
		"Earl Cameron",
		"Go Powered",
		"GoWebComponents",
		"gRPC Tunnel",
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseMarkup, parseCheck) {
			parseT.Fatalf("expected snapshot markup to contain %q, got %q", parseCheck, parseMarkup)
		}
	}
}

func TestPortfolioProjectsGridSnapshotRenderToString(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(renderPortfolioProjectsGridSnapshot())
	if parseErr != nil {
		parseT.Fatalf("unexpected grid render error: %v", parseErr)
	}

	parseChecks := []string{
		"Revolutionary Frontend Framework",
		"Native gRPC-over-WebSocket Solution",
		"Frontend Framework",
		"https://github.com/monstercameron/GoGRPCBridge",
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseMarkup, parseCheck) {
			parseT.Fatalf("expected project-grid markup to contain %q, got %q", parseCheck, parseMarkup)
		}
	}
}
