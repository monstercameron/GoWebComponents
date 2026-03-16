package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func renderPortfolioSnapshot() ui.Node {
	stats := portfolioHeroStats()
	projects := featuredPortfolioProjects()

	statNodes := make([]ui.Node, 0, len(stats))
	for _, stat := range stats {
		stat := stat
		statNodes = append(statNodes, html.Div(html.Props{Class: "stat-card"},
			html.Strong(html.Props{}, html.Text(stat.Number)),
			html.Span(html.Props{}, html.Text(stat.Label)),
		))
	}

	projectNodes := make([]ui.Node, 0, len(projects))
	for _, project := range projects {
		project := project
		projectNodes = append(projectNodes, html.Article(html.Props{Class: "project-card"},
			html.H2(html.Props{}, html.Text(project.Title)),
			html.P(html.Props{}, html.Text(project.Subtitle)),
			html.P(html.Props{}, html.Text(project.Description)),
		))
	}

	return html.Main(html.Props{ID: "portfolio-snapshot"},
		html.H1(html.Props{}, html.Text("Earl Cameron")),
		html.Section(html.Props{ID: "hero-stats"}, statNodes...),
		html.Section(html.Props{ID: "projects"}, projectNodes...),
	)
}

func TestPortfolioHeroStatsStable(t *testing.T) {
	stats := portfolioHeroStats()
	if len(stats) != 4 {
		t.Fatalf("expected 4 portfolio hero stats, got %d", len(stats))
	}
	if stats[0].Number != "100%" || stats[0].Label != "Go Powered" {
		t.Fatalf("unexpected first stat: %+v", stats[0])
	}
	if stats[1].Number != "0" || stats[1].Label != "JavaScript Required" {
		t.Fatalf("unexpected second stat: %+v", stats[1])
	}
}

func TestPortfolioProjectsLookupAndTechnologyTotals(t *testing.T) {
	projects := portfolioProjects()
	if len(projects) != 2 {
		t.Fatalf("expected 2 portfolio projects, got %d", len(projects))
	}
	if total := totalPortfolioTechnologies(projects); total != 10 {
		t.Fatalf("expected 10 total technology tags, got %d", total)
	}

	project, ok := findPortfolioProject("GoWebComponents")
	if !ok {
		t.Fatal("expected GoWebComponents project lookup to succeed")
	}
	if !project.Featured {
		t.Fatalf("expected GoWebComponents to be featured, got %+v", project)
	}

	project, ok = findPortfolioProject("grpc tunnel")
	if !ok {
		t.Fatal("expected case-insensitive gRPC Tunnel lookup to succeed")
	}
	if !strings.Contains(project.Description, "WebSocket") {
		t.Fatalf("expected gRPC Tunnel description to mention WebSocket, got %q", project.Description)
	}
	if _, ok := findPortfolioProject("missing"); ok {
		t.Fatal("expected missing project lookup to fail")
	}
}

func TestPortfolioSnapshotRenderToString(t *testing.T) {
	markup, err := ui.RenderToString(renderPortfolioSnapshot())
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	checks := []string{
		"Earl Cameron",
		"Go Powered",
		"GoWebComponents",
		"gRPC Tunnel",
	}
	for _, check := range checks {
		if !strings.Contains(markup, check) {
			t.Fatalf("expected snapshot markup to contain %q, got %q", check, markup)
		}
	}
}
