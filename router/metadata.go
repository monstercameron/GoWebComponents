package router

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

const managedMetadataAttr = "data-gwc-router-managed"
const managedMetadataValue = "true"

// Metadata describes the route-owned metadata currently supported by the router.
//
// These fields are intentionally narrow for the first SSR-aware metadata slice:
// title, description, and canonical URL. Broader head management remains a
// separate future concern.
type Metadata struct {
	Title        string
	Description  string
	CanonicalURL string
}

// BuildMetadataNode renders route-managed metadata as SSR-safe head children.
//
// The generated tags are marked so the client router can reconcile and clean up
// only framework-owned metadata during hydration and later navigations.
func BuildMetadataNode(parseRouterMetadata Metadata) *runtime.Element {
	parseMetadataChildren := make([]any, 0, 3)

	if parseMetadataTitle := strings.TrimSpace(parseRouterMetadata.Title); parseMetadataTitle != "" {
		parseMetadataChildren = append(parseMetadataChildren, runtime.CreateElement("title", map[string]any{
			managedMetadataAttr: managedMetadataValue,
		}, parseMetadataTitle))
	}

	if parseMetadataDescription := strings.TrimSpace(parseRouterMetadata.Description); parseMetadataDescription != "" {
		parseMetadataChildren = append(parseMetadataChildren, runtime.CreateElement("meta", map[string]any{
			managedMetadataAttr: managedMetadataValue,
			"name":              "description",
			"content":           parseMetadataDescription,
		}))
	}

	if parseMetadataCanonicalURL := strings.TrimSpace(parseRouterMetadata.CanonicalURL); parseMetadataCanonicalURL != "" {
		parseMetadataChildren = append(parseMetadataChildren, runtime.CreateElement("link", map[string]any{
			managedMetadataAttr: managedMetadataValue,
			"rel":               "canonical",
			"href":              parseMetadataCanonicalURL,
		}))
	}

	return runtime.CreateElement("FRAGMENT", nil, parseMetadataChildren...)
}
