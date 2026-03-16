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

func renderPortfolioProjectsGridSnapshot() ui.Node {
	projects := portfolioProjects()
	projectNodes := make([]ui.Node, 0, len(projects))
	for _, project := range projects {
		technologyNodes := make([]ui.Node, 0, len(project.Technologies))
		for _, tech := range project.Technologies {
			technologyNodes = append(technologyNodes, html.Span(html.Props{Class: "tech-chip"}, html.Text(tech)))
		}

		projectNodes = append(projectNodes, html.Article(html.Props{Class: "project-card"},
			html.H2(html.Props{}, html.Text(project.Title)),
			html.P(html.Props{}, html.Text(project.Subtitle)),
			html.P(html.Props{}, html.Text(project.Description)),
			html.Div(html.Props{Class: "technology-stack"}, technologyNodes...),
			html.A(html.Props{Href: project.Link}, html.Text("View Project")),
		))
	}

	return html.Section(html.Props{ID: "portfolio-project-grid"}, projectNodes...)
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

func TestFeaturedPortfolioProjectsAreStableAndOrdered(t *testing.T) {
	featured := featuredPortfolioProjects()
	if len(featured) != 2 {
		t.Fatalf("expected 2 featured portfolio projects, got %d", len(featured))
	}
	if featured[0].Title != "GoWebComponents" || featured[1].Title != "gRPC Tunnel" {
		t.Fatalf("unexpected featured project ordering: %+v", featured)
	}
	for _, project := range featured {
		if !project.Featured {
			t.Fatalf("expected featured project list to contain only featured projects, got %+v", project)
		}
	}
}

func TestPortfolioProjectMetadataValidity(t *testing.T) {
	projects := portfolioProjects()
	seenTitles := make(map[string]bool, len(projects))
	for _, project := range projects {
		if project.Title == "" || project.Subtitle == "" || project.Description == "" {
			t.Fatalf("expected complete project metadata, got %+v", project)
		}
		if !strings.HasPrefix(project.Link, "https://") {
			t.Fatalf("expected secure project link, got %q", project.Link)
		}
		if len(project.Technologies) == 0 {
			t.Fatalf("expected technologies for project %+v", project)
		}
		key := strings.ToLower(project.Title)
		if seenTitles[key] {
			t.Fatalf("expected unique project title, duplicate %q", project.Title)
		}
		seenTitles[key] = true
	}

	if portfolioHomeRoute != "/" || portfolioDocsRoute != "/docs" || portfolioCatchAllRoute != "*" {
		t.Fatalf("unexpected route constants: home=%q docs=%q catchAll=%q", portfolioHomeRoute, portfolioDocsRoute, portfolioCatchAllRoute)
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

func TestPortfolioProjectsGridSnapshotRenderToString(t *testing.T) {
	markup, err := ui.RenderToString(renderPortfolioProjectsGridSnapshot())
	if err != nil {
		t.Fatalf("unexpected grid render error: %v", err)
	}

	checks := []string{
		"Revolutionary Frontend Framework",
		"Native gRPC-over-WebSocket Solution",
		"Frontend Framework",
		"https://github.com/monstercameron/grpc-tunnel",
	}
	for _, check := range checks {
		if !strings.Contains(markup, check) {
			t.Fatalf("expected project-grid markup to contain %q, got %q", check, markup)
		}
	}
}
