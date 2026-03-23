package head

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// SocialMetadata describes common Open Graph and Twitter card metadata.
type SocialMetadata struct {
	Type        string
	Title       string
	Description string
	ImageURL    string
	URL         string
	TwitterCard string
}

// Compose combines router-managed metadata with optional explicit head tags.
func Compose(metadata router.Metadata, extras ...ui.Node) ui.Node {
	children := make([]ui.Node, 0, 1+len(extras))
	children = append(children, router.MetadataNode(metadata))
	for _, extra := range extras {
		if extra != nil {
			children = append(children, extra)
		}
	}
	return ui.Fragment(children...)
}

// MetaName renders a meta tag with a name-based attribute.
func MetaName(name, content string) ui.Node {
	name = strings.TrimSpace(name)
	content = strings.TrimSpace(content)
	if name == "" || content == "" {
		return nil
	}
	return html.Meta(html.Props{Raw: map[string]interface{}{
		"name":    name,
		"content": content,
	}})
}

// MetaProperty renders a meta tag with a property-based attribute.
func MetaProperty(property, content string) ui.Node {
	property = strings.TrimSpace(property)
	content = strings.TrimSpace(content)
	if property == "" || content == "" {
		return nil
	}
	return html.Meta(html.Props{Raw: map[string]interface{}{
		"property": property,
		"content":  content,
	}})
}

// LinkRel renders a link tag for a relationship and target URL.
func LinkRel(rel, href string) ui.Node {
	rel = strings.TrimSpace(rel)
	href = strings.TrimSpace(href)
	if rel == "" || href == "" {
		return nil
	}
	return html.Link(html.Props{Raw: map[string]interface{}{
		"rel":  rel,
		"href": href,
	}})
}

// Robots renders a robots meta tag.
func Robots(content string) ui.Node {
	return MetaName("robots", content)
}

// OpenGraph renders an Open Graph meta tag for the given field.
func OpenGraph(field, content string) ui.Node {
	field = strings.TrimSpace(field)
	if field == "" {
		return nil
	}
	return MetaProperty("og:"+field, content)
}

// Twitter renders a Twitter/X card meta tag for the given field.
func Twitter(field, content string) ui.Node {
	field = strings.TrimSpace(field)
	if field == "" {
		return nil
	}
	return MetaName("twitter:"+field, content)
}

// SocialTags renders a small common set of Open Graph and Twitter card tags.
func SocialTags(metadata SocialMetadata) ui.Node {
	card := strings.TrimSpace(metadata.TwitterCard)
	if card == "" && strings.TrimSpace(metadata.ImageURL) != "" {
		card = "summary_large_image"
	}
	return fragmentNonNil(
		OpenGraph("type", metadata.Type),
		OpenGraph("title", metadata.Title),
		OpenGraph("description", metadata.Description),
		OpenGraph("image", metadata.ImageURL),
		OpenGraph("url", metadata.URL),
		Twitter("card", card),
		Twitter("title", metadata.Title),
		Twitter("description", metadata.Description),
		Twitter("image", metadata.ImageURL),
	)
}

func fragmentNonNil(children ...ui.Node) ui.Node {
	filtered := make([]ui.Node, 0, len(children))
	for _, child := range children {
		if child != nil {
			filtered = append(filtered, child)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return ui.Fragment(filtered...)
}
