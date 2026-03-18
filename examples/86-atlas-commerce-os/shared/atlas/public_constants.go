package atlas

import "strings"

const (
	publicBrandLabel                     = "Atlas Commerce OS"
	publicIdentityLabel                  = "Workspace buying with regional delivery clarity"
	publicBrowseShopLabel                = "Browse shop"
	publicNavStorefrontLabel             = "Storefront"
	publicNavShopLabel                   = "Shop"
	publicNavWarehousesLabel             = "Warehouses"
	publicRequestTimeSSRLabel            = "Delivery-aware shopping"
	publicAtlasSystemLabel               = "Atlas system"
	publicRegionalHubLabel               = "Delivery region"
	publicRegionalPromiseLabel           = "Regional promise"
	publicOperationalContinuityLabel     = "Buying confidence"
	publicStartingAtLabel                = "Starting at"
	publicBuyerNextStepLabel             = "Buyer next step"
	publicRegionalNextStepLabel          = "Regional next step"
	publicCatalogOverviewLabel           = "Catalog overview"
	publicRegionalNetworkLabel           = "Regional network"
	publicPromiseFirstUXLabel            = "Regional delivery clarity"
	publicWhyAtlasFeelsReady             = "Why Atlas feels ready"
	publicAvailabilityStoryLabel         = "Availability story"
	publicWhyThisMattersLabel            = "Why this matters"
	publicDesignedForLabel               = "Designed for"
	publicFulfillmentLabel               = "Delivery outlook"
	publicBuyingMotionLabel              = "Buying motion"
	publicEditorialProductStoryLabel     = "Editorial product story"
	publicRouteStableTrustLabel          = "Buying confidence"
	publicCommercialSupportLabel         = "Commercial support"
	publicQuietHierarchyLabel            = "Quiet hierarchy"
	publicProgressiveFormsLabel          = "Easy follow-up"
	publicPremiumUtilityLabel            = "Premium utility"
	publicRegionalReadLabel              = "Best for"
	publicServicePostureLabel            = "Delivery pace"
	publicRegionalAvailabilityPicksLabel = "Regional availability picks"
	publicBrowseRegionalProductsLabel    = "Browse products in this region"
	publicCompareAllSystemsLabel         = "Compare all Atlas systems"
	publicRegionalFocusLabel             = "Regional focus"
	publicServiceLevelLabel              = "Service level"
	publicPromiseLensLabel               = "Why choose this region"
	publicOpenRegionalAvailabilityLabel  = "View delivery options"
)

func normalizedPublicStatus(status string) string {
	return strings.TrimSpace(strings.ToLower(status))
}

func isPublicReadyStatus(status string) bool {
	switch normalizedPublicStatus(status) {
	case "in_stock", "healthy", "approved", "available":
		return true
	default:
		return false
	}
}

func isPublicPlanningStatus(status string) bool {
	switch normalizedPublicStatus(status) {
	case "low_stock", "pending", "submitted", "in_review":
		return true
	default:
		return false
	}
}
