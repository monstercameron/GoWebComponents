// Package head provides optional SSR head-management helpers built on the
// public html, router, and ui packages.
//
// Core router metadata remains the source of truth for title, description, and
// canonical URL. This companion package layers a higher-level SSR composition
// surface on top of that managed slice for robots tags, social tags, alternate
// locale links, resource hints, JSON-LD, and other explicit head markup.
package head
