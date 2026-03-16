package router

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
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

// MetadataNode renders route-managed metadata as SSR-safe head children.
//
// The generated tags are marked so the client router can reconcile and clean up
// only framework-owned metadata during hydration and later navigations.
func MetadataNode(metadata Metadata) *runtime.Element {
	children := make([]interface{}, 0, 3)

	if title := strings.TrimSpace(metadata.Title); title != "" {
		children = append(children, runtime.CreateElement("title", map[string]interface{}{
			managedMetadataAttr: managedMetadataValue,
		}, title))
	}

	if description := strings.TrimSpace(metadata.Description); description != "" {
		children = append(children, runtime.CreateElement("meta", map[string]interface{}{
			managedMetadataAttr: managedMetadataValue,
			"name":              "description",
			"content":           description,
		}))
	}

	if canonical := strings.TrimSpace(metadata.CanonicalURL); canonical != "" {
		children = append(children, runtime.CreateElement("link", map[string]interface{}{
			managedMetadataAttr: managedMetadataValue,
			"rel":               "canonical",
			"href":              canonical,
		}))
	}

	return runtime.CreateElement("FRAGMENT", nil, children...)
}
