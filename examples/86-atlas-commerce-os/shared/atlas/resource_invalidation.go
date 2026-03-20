package atlas

import "strings"

var mutationInvalidationRules = map[string][]string{
	"comment-submitted": {
		"/app/comments",
		"/app/dashboard",
		"/shop",
	},
	"comment-moderated": {
		"/app/comments",
		"/app/dashboard",
		"/shop",
	},
	"comments-bulk-moderated": {
		"/app/comments",
		"/app/dashboard",
		"/shop",
	},
	"inventory-updated": {
		"/app/inventory",
		"/app/warehouses",
		"/app/dashboard",
		"/shop",
		"/warehouses",
	},
	"preferences-saved": {
		"/app/dashboard",
		"/app/products",
		"/app/inventory",
		"/app/warehouses",
		"/app/transfers",
		"/app/purchase-orders",
		"/app/receiving",
		"/app/comments",
		"/app/settings",
	},
	"product-created": {
		"/app/products",
		"/app/inventory",
		"/app/warehouses",
		"/shop",
		"/warehouses",
	},
	"product-deleted": {
		"/app/products",
		"/app/inventory",
		"/app/warehouses",
		"/shop",
		"/warehouses",
	},
	"product-updated": {
		"/app/products",
		"/app/inventory",
		"/app/warehouses",
		"/shop",
		"/warehouses",
	},
	"purchase-order-created": {
		"/app/purchase-orders",
		"/app/receiving",
		"/app/dashboard",
		"/app/inventory",
		"/app/warehouses",
		"/shop",
		"/warehouses",
	},
	"purchase-order-updated": {
		"/app/purchase-orders",
		"/app/receiving",
		"/app/dashboard",
	},
	"quote-request-submitted": {
		"/shop",
		"/warehouses",
	},
	"receiving-reconciled": {
		"/app/receiving",
		"/app/purchase-orders",
		"/app/dashboard",
	},
	"restock-request-submitted": {
		"/shop",
		"/warehouses",
	},
	"saved-view-created": {
		"/app/dashboard",
		"/app/products",
		"/app/inventory",
		"/app/warehouses",
		"/app/transfers",
		"/app/purchase-orders",
		"/app/receiving",
		"/app/comments",
		"/app/settings",
	},
	"saved-views-imported": {
		"/app/dashboard",
		"/app/products",
		"/app/inventory",
		"/app/warehouses",
		"/app/transfers",
		"/app/purchase-orders",
		"/app/receiving",
		"/app/comments",
		"/app/settings",
	},
	"threshold-updated": {
		"/app/inventory",
		"/app/warehouses",
		"/app/dashboard",
	},
	"transfer-created": {
		"/app/transfers",
		"/app/inventory",
		"/app/dashboard",
	},
}

func MutationRoutePrefixes(path string, notice string) []string {
	trimmed := strings.TrimSpace(strings.ToLower(notice))
	if prefixes, ok := mutationInvalidationRules[trimmed]; ok {
		return append([]string(nil), prefixes...)
	}
	return []string{path}
}

func MutationRequestPrefixes(routePrefixes []string) []string {
	requestPrefixes := make([]string, 0, len(routePrefixes))
	for _, prefix := range routePrefixes {
		requestPrefixes = append(requestPrefixes, mutationRequestPrefixesForRoutePrefix(prefix)...)
	}
	return requestPrefixes
}

func mutationRequestPrefixesForRoutePrefix(prefix string) []string {
	switch prefix {
	case "/shop":
		return []string{"/api/public/catalog", "/api/public/products"}
	case "/warehouses":
		return []string{"/api/public/warehouses"}
	case "/app/dashboard":
		return []string{"/api/app/dashboard"}
	case "/app/products":
		return []string{"/api/app/products"}
	case "/app/inventory":
		return []string{"/api/app/inventory"}
	case "/app/warehouses":
		return []string{"/api/app/warehouses"}
	case "/app/transfers":
		return []string{"/api/app/transfers"}
	case "/app/purchase-orders":
		return []string{"/api/app/purchase-orders"}
	case "/app/receiving":
		return []string{"/api/app/receiving"}
	case "/app/comments":
		return []string{"/api/app/comments"}
	case "/app/settings":
		return []string{"/api/app/settings", "/api/app/saved-views"}
	default:
		return nil
	}
}
