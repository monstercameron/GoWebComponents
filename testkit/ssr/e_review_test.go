package ssr

import "testing"

// TestStaticExportRouteBootstrapIsDeterministic verifies a route resolves to its
// plain bootstrap sidecar every time, not randomly to a compressed sibling.
func TestStaticExportRouteBootstrapIsDeterministic(parseT *testing.T) {
	parseExport := StaticExport{
		HTMLFiles: map[string]Snapshot{"about/index.html": {HTML: "<p>about</p>"}},
		Bootstrap: map[string][]byte{
			"bootstrap/about.json":    []byte(`{"plain":true}`),
			"bootstrap/about.json.br": []byte("brotli"),
			"bootstrap/about.json.gz": []byte("gzip"),
		},
	}
	for parseI := 0; parseI < 100; parseI++ {
		parseRoute, parseErr := parseExport.Route("/about")
		if parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if parseRoute.BootstrapFile != "bootstrap/about.json" {
			parseT.Fatalf("iteration %d resolved %q, want the plain sidecar", parseI, parseRoute.BootstrapFile)
		}
	}
}
