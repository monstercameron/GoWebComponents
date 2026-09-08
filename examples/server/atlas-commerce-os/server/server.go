package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/css"
	serverauth "github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/server/auth"
	serverdb "github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/server/db"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/api"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/atlas"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/bootfallback"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/repository"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type atlasServer struct {
	cfg      config
	store    *serverdb.Store
	sessions *serverauth.MockSessionManager
}

type routeMeta struct {
	Path        string
	Surface     string
	Screen      string
	Title       string
	Description string
	Canonical   string
}

const (
	atlasNoticeQueryKey             = "atlas_notice"
	atlasBootstrapModeQueryKey      = "atlas_bootstrap"
	atlasBootstrapModeExternal      = "external"
	atlasBootstrapScriptID          = "__ATLAS_BOOTSTRAP__"
	atlasBootstrapReferenceScriptID = "__ATLAS_BOOTSTRAP_REF__"
)

func newAtlasServer(parseCfg config, store *serverdb.Store, parseSessions *serverauth.MockSessionManager) *atlasServer {
	return &atlasServer{cfg: parseCfg, store: store, sessions: parseSessions}
}

func (parseS *atlasServer) routes() http.Handler {
	parseMux := http.NewServeMux()

	// Typed server functions (shared/api). This is the v5 way to cross the
	// client/server boundary: one Go signature per operation, from which
	// `gwc server gen` writes both the registration below and the browser stub
	// that calls it, so neither side can drift from the other.
	//
	// Contrast the hand-rolled routes further down this function — each is a URL
	// string here, a matching URL string built by concatenation in
	// shared/atlas/legacy_shared.go, a handler that encodes JSON, and a caller that
	// decodes it into an any-shaped payload. Four places to agree, checked nowhere.
	// A server function is one place, checked by the compiler.
	//
	// The dependency is registered before the mux is served: ListWarehouses is a
	// plain top-level func and cannot take the store as an argument, so it reads it
	// through a seam that api owns. Wiring it here rather than in main keeps the
	// registration next to the routing it belongs to, and means a test that builds
	// a server gets a working API without a separate setup call to forget.
	parseStore := parseS.store
	api.SetWarehouseSource(func(parseCtx context.Context) ([]api.Warehouse, error) {
		parseWarehouses, parseErr := parseStore.Warehouses(parseCtx)
		if parseErr != nil {
			return nil, parseErr
		}
		// Mapped field by field on purpose. api.Warehouse is the PUBLIC shape and
		// serverdb.Warehouse is the storage shape; a struct conversion or an alias
		// would mean any column added to storage silently reaches the browser.
		parsePublic := make([]api.Warehouse, 0, len(parseWarehouses))
		for _, parseWarehouse := range parseWarehouses {
			parsePublic = append(parsePublic, api.Warehouse{
				ID:            parseWarehouse.ID,
				Slug:          parseWarehouse.Slug,
				Name:          parseWarehouse.Name,
				Region:        parseWarehouse.Region,
				ServiceLevel:  parseWarehouse.ServiceLevel,
				PublicSummary: parseWarehouse.PublicSummary,
			})
		}
		return parsePublic, nil
	})
	api.RegisterServerFunctions(parseMux)

	parseStaticFS := http.FileServer(http.Dir(parseS.cfg.StaticDir))
	parseMux.Handle("/assets/", http.StripPrefix("/assets/", parseStaticFS))
	parseMux.HandleFunc("GET /healthz", parseS.handleHealth)
	parseMux.HandleFunc("GET /auth/mock-sign-in", parseS.handleMockSignInPage)
	parseMux.HandleFunc("POST /auth/mock-sign-in", parseS.handleMockSignIn)
	parseMux.HandleFunc("POST /auth/mock-sign-out", parseS.handleMockSignOut)
	parseMux.HandleFunc("GET /api/public/catalog", parseS.handlePublicCatalog)
	parseMux.HandleFunc("GET /api/public/products/{slug}", parseS.handlePublicProduct)
	parseMux.HandleFunc("GET /api/public/products/{slug}/comments", parseS.handlePublicProductComments)
	parseMux.HandleFunc("GET /api/public/products/{slug}/related-products", parseS.handlePublicRelatedProducts)
	parseMux.HandleFunc("GET /api/public/warehouses", parseS.handlePublicWarehouses)
	parseMux.HandleFunc("GET /api/public/warehouses/{slug}", parseS.handlePublicWarehouse)
	parseMux.HandleFunc("GET /api/public/warehouses/{slug}/availability/{productSlug}", parseS.handlePublicAvailability)
	parseMux.HandleFunc("GET /api/app/bootstrap", parseS.handleInternalBootstrap)
	parseMux.HandleFunc("GET /api/app/dashboard", parseS.handleInternalDashboard)
	parseMux.HandleFunc("GET /api/app/preferences", parseS.handleInternalPreferences)
	parseMux.HandleFunc("GET /api/app/settings", parseS.handleInternalSettings)
	parseMux.HandleFunc("GET /api/app/saved-views", parseS.handleInternalSavedViews)
	parseMux.HandleFunc("GET /api/app/saved-views/export", parseS.handleInternalSavedViewsExport)
	parseMux.HandleFunc("GET /api/app/comments", parseS.handleInternalComments)
	parseMux.HandleFunc("GET /api/app/products", parseS.handleInternalProducts)
	parseMux.HandleFunc("GET /api/app/products/{slug}", parseS.handleInternalProductDetail)
	parseMux.HandleFunc("GET /api/app/inventory", parseS.handleInternalInventory)
	parseMux.HandleFunc("GET /api/app/inventory/{sku}", parseS.handleInternalInventoryDetail)
	parseMux.HandleFunc("GET /api/app/inventory/{sku}/threshold-panel", parseS.handleInternalThresholdPanel)
	parseMux.HandleFunc("GET /api/app/inventory/{sku}/threshold-history", parseS.handleInternalThresholdHistory)
	parseMux.HandleFunc("GET /api/app/inventory/{sku}/transfer-recommendations", parseS.handleInternalTransferRecommendations)
	parseMux.HandleFunc("GET /api/app/warehouses", parseS.handleInternalWarehouses)
	parseMux.HandleFunc("GET /api/app/warehouses/{warehouseId}", parseS.handleInternalWarehouseDetail)
	parseMux.HandleFunc("GET /api/app/warehouses/{warehouseId}/items/{sku}", parseS.handleInternalWarehouseItemDetail)
	parseMux.HandleFunc("GET /api/app/transfers", parseS.handleInternalTransfers)
	parseMux.HandleFunc("GET /api/app/transfers/{id}", parseS.handleInternalTransferDetail)
	parseMux.HandleFunc("GET /api/app/receiving", parseS.handleInternalReceiving)
	parseMux.HandleFunc("GET /api/app/receiving/{id}", parseS.handleInternalReceivingDetail)
	parseMux.HandleFunc("GET /api/app/purchase-orders", parseS.handleInternalPurchaseOrders)
	parseMux.HandleFunc("GET /api/app/purchase-orders/{id}", parseS.handleInternalPurchaseOrderDetail)
	parseMux.HandleFunc("POST /api/public/products/{slug}/comments", parseS.handlePublicCommentCreate)
	parseMux.HandleFunc("POST /api/public/products/{slug}/quote-requests", parseS.handlePublicQuoteRequestCreate)
	parseMux.HandleFunc("POST /api/public/products/{slug}/restock-requests", parseS.handlePublicRestockRequestCreate)
	parseMux.HandleFunc("POST /api/app/comments/{id}/moderate", parseS.handleInternalCommentModeration)
	parseMux.HandleFunc("POST /api/app/comments/bulk-moderate", parseS.handleInternalBulkCommentModeration)
	parseMux.HandleFunc("POST /api/app/products", parseS.handleInternalProductCreate)
	parseMux.HandleFunc("POST /api/app/products/{slug}/update", parseS.handleInternalProductUpdate)
	parseMux.HandleFunc("POST /api/app/products/{slug}/delete", parseS.handleInternalProductDelete)
	parseMux.HandleFunc("POST /api/app/inventory/{sku}/update", parseS.handleInternalInventoryUpdate)
	parseMux.HandleFunc("POST /api/app/inventory/{sku}/threshold", parseS.handleInternalThresholdUpdate)
	parseMux.HandleFunc("POST /api/app/preferences", parseS.handleInternalPreferencesSave)
	parseMux.HandleFunc("PUT /api/app/preferences", parseS.handleInternalPreferencesSave)
	parseMux.HandleFunc("POST /api/app/saved-views", parseS.handleInternalSavedViewCreate)
	parseMux.HandleFunc("POST /api/app/saved-views/import", parseS.handleInternalSavedViewImport)
	parseMux.HandleFunc("POST /api/app/transfers", parseS.handleInternalTransferCreate)
	parseMux.HandleFunc("POST /api/app/purchase-orders", parseS.handleInternalPurchaseOrderCreate)
	parseMux.HandleFunc("POST /api/app/receiving/{id}/reconcile", parseS.handleInternalReceivingReconcile)
	parseMux.HandleFunc("POST /api/app/receiving/{id}/attachments", parseS.handleInternalReceivingAttachmentCreate)
	parseMux.HandleFunc("POST /api/app/purchase-orders/{id}/status", parseS.handleInternalPurchaseOrderStatus)
	parseMux.HandleFunc("GET /{$}", parseS.handleLandingPage)
	parseMux.HandleFunc("GET /shop", parseS.handleCatalogPage)
	parseMux.HandleFunc("GET /shop/{slug}", parseS.handleProductPage)
	parseMux.HandleFunc("GET /warehouses", parseS.handleWarehousesPage)
	parseMux.HandleFunc("GET /warehouses/{slug}", parseS.handleWarehousePage)
	parseMux.HandleFunc("GET /warehouses/{slug}/availability/{productSlug}", parseS.handleAvailabilityPage)
	parseMux.HandleFunc("GET /app", parseS.handleAppRoot)
	parseMux.HandleFunc("GET /app/dashboard", parseS.handleDashboardPage)
	parseMux.HandleFunc("GET /app/products", parseS.handleInternalProductsPage)
	parseMux.HandleFunc("GET /app/products/{slug}", parseS.handleInternalProductEditorPage)
	parseMux.HandleFunc("GET /app/inventory", parseS.handleInventoryPage)
	parseMux.HandleFunc("GET /app/inventory/{sku}", parseS.handleInventoryDetailPage)
	parseMux.HandleFunc("GET /app/inventory/{sku}/threshold-history", parseS.handleInventoryThresholdHistoryPage)
	parseMux.HandleFunc("GET /app/warehouses", parseS.handleWarehouseOpsPage)
	parseMux.HandleFunc("GET /app/warehouses/{warehouseId}", parseS.handleWarehouseOpsDetailPage)
	parseMux.HandleFunc("GET /app/warehouses/{warehouseId}/items/{sku}", parseS.handleWarehouseOpsItemPage)
	parseMux.HandleFunc("GET /app/transfers", parseS.handleTransfersPage)
	parseMux.HandleFunc("GET /app/transfers/{id}", parseS.handleTransferDetailPage)
	parseMux.HandleFunc("GET /app/purchase-orders", parseS.handlePurchaseOrdersPage)
	parseMux.HandleFunc("GET /app/purchase-orders/{id}", parseS.handlePurchaseOrderDetailPage)
	parseMux.HandleFunc("GET /app/receiving", parseS.handleReceivingPage)
	parseMux.HandleFunc("GET /app/receiving/{id}", parseS.handleReceivingDetailPage)
	parseMux.HandleFunc("GET /app/comments", parseS.handleCommentsPage)
	parseMux.HandleFunc("GET /app/comments/moderation/{status}", parseS.handleCommentsModerationPage)
	parseMux.HandleFunc("GET /app/comments/{id}", parseS.handleCommentDetailPage)
	parseMux.HandleFunc("GET /app/settings", parseS.handleSettingsPage)
	parseMux.HandleFunc("GET /app/settings/appearance", parseS.handleSettingsAppearancePage)
	parseMux.HandleFunc("GET /app/settings/locale", parseS.handleSettingsLocalePage)
	parseMux.HandleFunc("GET /app/settings/workspace-defaults", parseS.handleSettingsWorkspaceDefaultsPage)
	parseMux.HandleFunc("GET /__atlas/bootstrap.json", parseS.handleBootstrapJSON)
	parseBaseRoutes := http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.Method != http.MethodGet || strings.HasPrefix(parseR.URL.Path, "/assets/") || strings.HasPrefix(parseR.URL.Path, "/api/") {
			parseMux.ServeHTTP(parseW, parseR)
			return
		}
		parseProbe := newBufferedResponseWriter()
		parseMux.ServeHTTP(parseProbe, parseR)
		if parseProbe.status == http.StatusNotFound {
			parseS.handleRouteRecoveryPage(parseW, parseR)
			return
		}
		parseProbe.FlushTo(parseW)
	})
	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseStarted := time.Now()
		parseCaptureW := newServerStatusCaptureWriter(parseW)
		parseBaseRoutes.ServeHTTP(parseCaptureW, parseR)
		parseS.logServerRequestEvent(parseR, parseCaptureW.status, time.Since(parseStarted), parseCaptureW.Header())
	})
}

func (parseS *atlasServer) handleHealth(parseW http.ResponseWriter, parseR *http.Request) {
	parseStatus := map[string]any{
		"ok":           true,
		"service":      "atlas-commerce-os",
		"databasePath": parseS.cfg.SQLitePath,
		"wasmPresent":  fileExists(parseS.cfg.AtlasWASM),
	}
	parseS.writeJSON(parseW, http.StatusOK, parseStatus)
}

func (parseS *atlasServer) handleMockSignInPage(parseW http.ResponseWriter, parseR *http.Request) {
	if parseSession := parseS.sessions.Resolve(parseR); parseSession != nil {
		http.Redirect(parseW, parseR, sanitizeNextPath(parseR.URL.Query().Get("next")), http.StatusSeeOther)
		return
	}
	parseRoles := make([]map[string]string, 0, len(serverauth.AllowedRoles()))
	for _, parseRole := range serverauth.AllowedRoles() {
		parseRoles = append(parseRoles, map[string]string{
			"value":       parseRole,
			"label":       roleLabel(parseRole),
			"description": roleDescription(parseRole),
		})
	}
	parseS.renderPageStatus(parseW, parseR, http.StatusOK, routeMeta{Path: serverauth.MockSignInPath, Surface: "public", Screen: "mock-sign-in", Title: "Atlas Mock Sign In", Description: "Start a mock internal session for the Atlas operator console.", Canonical: serverauth.MockSignInPath}, map[string]any{
		"next":    sanitizeNextPath(parseR.URL.Query().Get("next")),
		"roles":   parseRoles,
		"message": "Start a mock Atlas internal session to access the operator console.",
	}, nil)
}

func (parseS *atlasServer) handleMockSignIn(parseW http.ResponseWriter, parseR *http.Request) {
	if parseErr := parseR.ParseForm(); parseErr != nil {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_mock_sign_in", "Mock sign-in could not be parsed.", map[string]string{"role": "Choose one of the supported mock roles."})
		return
	}
	parseRole := strings.TrimSpace(parseR.Form.Get("role"))
	parseNext := sanitizeNextPath(parseR.Form.Get("next"))
	parseCookie := parseS.sessions.StartCookie(parseRole)
	if parseCookie.Value == "" {
		if wantsHTMLResponse(parseR) {
			http.Redirect(parseW, parseR, withNotice(serverauth.MockSignInPath+"?next="+url.QueryEscape(parseNext), "invalid-mock-role"), http.StatusSeeOther)
			return
		}
		parseS.writeValidationError(parseW, http.StatusBadRequest, "invalid_mock_sign_in", "Choose one of the supported mock roles.", map[string]string{"role": "Unsupported mock role."})
		return
	}
	http.SetCookie(parseW, parseCookie)
	if wantsHTMLResponse(parseR) {
		http.Redirect(parseW, parseR, withNotice(parseNext, "mock-session-started"), http.StatusSeeOther)
		return
	}
	parseS.writeJSON(parseW, http.StatusCreated, map[string]any{"ok": true, "next": parseNext, "role": parseCookie.Value})
}

func (parseS *atlasServer) handleMockSignOut(parseW http.ResponseWriter, parseR *http.Request) {
	http.SetCookie(parseW, parseS.sessions.ClearCookie())
	parseNext := sanitizeNextPath(parseR.FormValue("next"))
	if wantsHTMLResponse(parseR) {
		http.Redirect(parseW, parseR, withNotice(parseNext, "mock-session-cleared"), http.StatusSeeOther)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"ok": true, "next": parseNext})
}

func (parseS *atlasServer) handlePublicCatalog(parseW http.ResponseWriter, parseR *http.Request) {
	parsePage := parsePositiveInt(parseR.URL.Query().Get("page"), 1)
	parseResult, parseErr := parseS.store.Catalog(parseR.Context(), serverdb.CatalogQuery{
		Search:    strings.TrimSpace(parseR.URL.Query().Get("q")),
		Category:  strings.TrimSpace(parseR.URL.Query().Get("category")),
		Warehouse: strings.TrimSpace(parseR.URL.Query().Get("warehouse")),
		Sort:      strings.TrimSpace(parseR.URL.Query().Get("sort")),
		Page:      parsePage,
		PageSize:  12,
	})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "catalog_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseResult)
}

func (parseS *atlasServer) handlePublicProduct(parseW http.ResponseWriter, parseR *http.Request) {
	_, _, parsePageData, parseErr := parseS.publicProductPage(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "product_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parsePageData)
}

func (parseS *atlasServer) handlePublicProductComments(parseW http.ResponseWriter, parseR *http.Request) {
	parseStatus := strings.TrimSpace(parseR.URL.Query().Get("status"))
	if parseStatus == "" {
		parseStatus = "approved"
	}
	parseItems, parseErr := parseS.store.ProductComments(parseR.Context(), parseR.PathValue("slug"), parseStatus)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "product_comments_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"items": parseItems})
}

func (parseS *atlasServer) handlePublicRelatedProducts(parseW http.ResponseWriter, parseR *http.Request) {
	parseItems, parseErr := parseS.store.RelatedProducts(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "related_products_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"items": parseItems})
}

func (parseS *atlasServer) handlePublicWarehouses(parseW http.ResponseWriter, parseR *http.Request) {
	parseItems, parseErr := parseS.store.Warehouses(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "warehouse_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"items": parseItems})
}

func (parseS *atlasServer) handlePublicWarehouse(parseW http.ResponseWriter, parseR *http.Request) {
	parseWarehouse, parseErr := parseS.store.WarehouseBySlug(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "warehouse_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseWarehouse)
}

func (parseS *atlasServer) handlePublicAvailability(parseW http.ResponseWriter, parseR *http.Request) {
	parseAvailability, parseErr := parseS.store.Availability(parseR.Context(), parseR.PathValue("slug"), parseR.PathValue("productSlug"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "availability_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseAvailability)
}

type publicWarehouseProductQuery struct {
	Search   string
	Category string
	Status   string
	Sort     string
}

func parsePublicWarehouseProductQuery(parseValues url.Values) publicWarehouseProductQuery {
	parseQuery := publicWarehouseProductQuery{
		Search:   strings.TrimSpace(parseValues.Get("q")),
		Category: strings.TrimSpace(parseValues.Get("category")),
		Status:   strings.TrimSpace(parseValues.Get("status")),
		Sort:     strings.TrimSpace(parseValues.Get("sort")),
	}
	if parseQuery.Sort == "" {
		parseQuery.Sort = "volume"
	}
	return parseQuery
}

func (parseS *atlasServer) handleInternalBootstrap(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseTargetPath := strings.TrimSpace(parseR.URL.Query().Get("path"))
	if parseTargetPath == "" {
		parseTargetPath = "/app/dashboard"
	}
	parsePayload, parseMeta, parseErr := parseS.bootstrapForPath(parseR, parseTargetPath, parseSession)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "bootstrap_failed", parseErr)
		return
	}
	parsePayload.CSRF = ensureCSRFCookie(parseW, parseR)
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"meta": parseMeta, "bootstrap": parsePayload})
}

func (parseS *atlasServer) handleInternalDashboard(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseData, parseErr := parseS.internalDashboardPageData(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "dashboard_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseData)
}

func (parseS *atlasServer) handleInternalPreferences(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseItem, parseErr := parseS.store.PreferencesByOwner(parseR.Context(), parseSession.UserID)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "preferences_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseItem)
}

func (parseS *atlasServer) handleInternalSettings(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseData, parseErr := parseS.internalSettingsPageData(parseR.Context(), parseSession.UserID)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "settings_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseData)
}

func (parseS *atlasServer) handleInternalSavedViews(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseItems, parseErr := parseS.store.SavedViewsByOwner(parseR.Context(), parseSession.UserID)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "saved_views_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"items": parseItems})
}

func (parseS *atlasServer) handleInternalComments(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseData, parseErr := parseS.internalCommentsPageData(parseR.Context(), strings.TrimSpace(parseR.URL.Query().Get("status")))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "comment_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseData)
}

func (parseS *atlasServer) handleInternalInventory(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseData, parseErr := parseS.internalInventoryPageData(parseR.Context(), parseR.URL.Query())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "inventory_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseData)
}

func (parseS *atlasServer) handleInternalInventoryDetail(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseData, parseErr := parseS.internalInventoryDetailPageData(parseR.Context(), parseR.PathValue("sku"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "inventory_item_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseData)
}

func (parseS *atlasServer) handleInternalThresholdPanel(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseData, parseErr := parseS.internalInventoryThresholdPanelPageData(parseR.Context(), parseR.PathValue("sku"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "threshold_panel_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseData)
}

func (parseS *atlasServer) handleInternalThresholdHistory(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItems, parseErr := parseS.store.ThresholdHistory(parseR.Context(), parseR.PathValue("sku"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "threshold_history_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"items": parseItems})
}

func (parseS *atlasServer) handleInternalTransferRecommendations(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItems, parseErr := parseS.store.TransferRecommendations(parseR.Context(), parseR.PathValue("sku"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "transfer_recommendations_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"items": parseItems})
}

func (parseS *atlasServer) handleInternalWarehouses(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseData, parseErr := parseS.internalWarehouseOpsPageData(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "warehouse_pressure_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseData)
}

func (parseS *atlasServer) handleInternalWarehouseDetail(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItem, parseErr := parseS.internalWarehouseDetailPageDataWithFilters(parseR.Context(), parseR.PathValue("warehouseId"), parseR.URL.Query())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "warehouse_detail_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseItem)
}

func (parseS *atlasServer) handleInternalWarehouseItemDetail(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItem, parseErr := parseS.internalWarehouseItemPageData(parseR.Context(), parseR.PathValue("warehouseId"), parseR.PathValue("sku"), parseR.URL.Query())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "warehouse_item_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseItem)
}

func (parseS *atlasServer) handleInternalTransfers(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItems, parseErr := parseS.store.Transfers(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "transfer_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"items": parseItems})
}

func (parseS *atlasServer) handleInternalTransferDetail(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItem, parseErr := parseS.store.TransferDetail(parseR.Context(), parseR.PathValue("id"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "transfer_detail_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseItem)
}

func (parseS *atlasServer) handleInternalReceiving(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItems, parseErr := parseS.store.ReceivingSessions(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "receiving_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, map[string]any{"items": parseItems})
}

func (parseS *atlasServer) handleInternalReceivingDetail(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItem, parseErr := parseS.store.ReceivingDetail(parseR.Context(), parseR.PathValue("id"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "receiving_detail_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseItem)
}

func (parseS *atlasServer) handleInternalPurchaseOrders(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseData, parseErr := parseS.internalPurchaseOrdersPageData(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "purchase_orders_query_failed", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseData)
}

func (parseS *atlasServer) handleInternalPurchaseOrderDetail(parseW http.ResponseWriter, parseR *http.Request) {
	if parseS.sessions.RequireInternalSession(parseW, parseR) == nil {
		return
	}
	parseItem, parseErr := parseS.store.PurchaseOrderDetail(parseR.Context(), parseR.PathValue("id"))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "purchase_order_detail_not_found", parseErr)
		return
	}
	parseS.writeJSON(parseW, http.StatusOK, parseItem)
}

func (parseS *atlasServer) handleLandingPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/", Surface: "public", Screen: "landing", Title: "Atlas Commerce OS", Description: "Premium modular workspace systems with warehouse-aware availability.", Canonical: "/"}, map[string]any{"message": "Atlas storefront landing"}, nil)
}

func (parseS *atlasServer) handleCatalogPage(parseW http.ResponseWriter, parseR *http.Request) {
	parsePage := parsePositiveInt(parseR.URL.Query().Get("page"), 1)
	parseResult, parseErr := parseS.store.Catalog(parseR.Context(), serverdb.CatalogQuery{Search: parseR.URL.Query().Get("q"), Category: parseR.URL.Query().Get("category"), Warehouse: parseR.URL.Query().Get("warehouse"), Sort: parseR.URL.Query().Get("sort"), Page: parsePage, PageSize: 12})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "catalog_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/shop", Surface: "public", Screen: "catalog", Title: "Atlas Shop", Description: "Browse Atlas modular workspace systems with filterable discovery and regional fulfillment context.", Canonical: "/shop"}, parseResult, nil)
}

func (parseS *atlasServer) handleProductPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseProduct, _, parsePageData, parseErr := parseS.publicProductPage(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: "/shop/" + parseR.PathValue("slug"), Surface: "public", Screen: "recovery", Title: "Atlas Product Not Found", Description: "The requested Atlas product could not be loaded.", Canonical: "/shop/" + parseR.PathValue("slug")}, nil, "Product not found", "The requested Atlas product is unavailable or no longer part of the demo seed.", "/shop", "Back to shop", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/shop/" + parseR.PathValue("slug"), Surface: "public", Screen: "product", Title: "Atlas " + parseProduct.Title, Description: parseProduct.SEODescription, Canonical: "/shop/" + parseProduct.Slug}, parsePageData, nil)
}

func (parseS *atlasServer) publicProductPage(parseCtx context.Context, parseSlug string) (repository.Product, []serverdb.CommentRecord, map[string]any, error) {
	parseProduct, parseErr := parseS.store.ProductBySlug(parseCtx, parseSlug)
	if parseErr != nil {
		return repository.Product{}, nil, nil, parseErr
	}
	parseComments, parseErr := parseS.store.ProductComments(parseCtx, parseSlug, "approved")
	if parseErr != nil {
		return repository.Product{}, nil, nil, parseErr
	}
	return parseProduct, parseComments, map[string]any{"product": parseProduct, "comments": parseComments}, nil
}

func (parseS *atlasServer) handleWarehousesPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseItems, parseErr := parseS.store.Warehouses(parseR.Context())
	if parseErr != nil {
		parseS.renderRecoveryPage(parseW, parseR, http.StatusInternalServerError, routeMeta{Path: "/warehouses", Surface: "public", Screen: "recovery", Title: "Atlas Delivery Regions Unavailable", Description: "The Atlas delivery-region directory could not be loaded.", Canonical: "/warehouses"}, nil, "Delivery regions unavailable", "The Atlas delivery-region directory is temporarily unavailable. Retry the page or return to the storefront.", "/", "Back to storefront", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/warehouses", Surface: "public", Screen: "warehouses", Title: "Atlas Delivery Regions", Description: "Compare Atlas delivery regions, service levels, and stocked highlights before opening a warehouse route.", Canonical: "/warehouses"}, map[string]any{"items": parseItems}, nil)
}

func (parseS *atlasServer) handleWarehousePage(parseW http.ResponseWriter, parseR *http.Request) {
	parseWarehouse, parseErr := parseS.store.WarehouseBySlug(parseR.Context(), parseR.PathValue("slug"))
	if parseErr != nil {
		parsePath := "/warehouses/" + parseR.PathValue("slug")
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "public", Screen: "recovery", Title: "Atlas Warehouse Not Found", Description: "The requested warehouse could not be loaded.", Canonical: parsePath}, nil, "Warehouse not found", "That warehouse route is not available in the current Atlas seed set.", "/warehouses", "Back to warehouses", parseErr)
		return
	}
	parseProducts, parseErr := parseS.store.Catalog(parseR.Context(), serverdb.CatalogQuery{Warehouse: parseWarehouse.ID, Page: 1, PageSize: 3})
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "warehouse_catalog_query_failed", parseErr)
		return
	}
	parsePath2 := "/warehouses/" + parseWarehouse.Slug
	parseS.renderPage(parseW, parseR, routeMeta{Path: parsePath2, Surface: "public", Screen: "warehouse-detail", Title: "Atlas Warehouse Detail", Description: parseWarehouse.PublicSummary, Canonical: parsePath2}, map[string]any{"warehouse": parseWarehouse, "products": parseProducts.Items}, nil)
}

func (parseS *atlasServer) handleAvailabilityPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseAvailability, parseErr := parseS.store.Availability(parseR.Context(), parseR.PathValue("slug"), parseR.PathValue("productSlug"))
	if parseErr != nil {
		parsePath := fmt.Sprintf("/warehouses/%s/availability/%s", parseR.PathValue("slug"), parseR.PathValue("productSlug"))
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "public", Screen: "recovery", Title: "Atlas Availability Not Found", Description: "The requested warehouse availability view could not be loaded.", Canonical: parsePath}, nil, "Availability view not found", "That warehouse-specific availability route is not available in the current Atlas demo data.", "/warehouses", "Back to warehouses", parseErr)
		return
	}
	parsePath2 := fmt.Sprintf("/warehouses/%s/availability/%s", parseR.PathValue("slug"), parseR.PathValue("productSlug"))
	parseS.renderPage(parseW, parseR, routeMeta{Path: parsePath2, Surface: "public", Screen: "warehouse-availability", Title: "Atlas Warehouse Availability", Description: "Inspect a warehouse-specific product promise for one Atlas item and one regional fulfillment hub.", Canonical: parsePath2}, parseAvailability, nil)
}

func (parseS *atlasServer) handleAppRoot(parseW http.ResponseWriter, parseR *http.Request) {
	http.Redirect(parseW, parseR, "/app/dashboard", http.StatusFound)
}

func (parseS *atlasServer) handleDashboardPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseData, parseErr := parseS.internalDashboardPageData(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "dashboard_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/app/dashboard", Surface: "internal", Screen: "dashboard", Title: "Atlas Ops Dashboard", Description: "Operational overview of buyer questions, stock pressure, receiving exceptions, and warehouse health.", Canonical: "/app/dashboard"}, parseData, parseSession)
}

func (parseS *atlasServer) handleInventoryPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseData, parseErr := parseS.internalInventoryPageData(parseR.Context(), parseR.URL.Query())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "inventory_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/app/inventory", Surface: "internal", Screen: "inventory", Title: "Atlas Inventory", Description: "Review inventory health, saved views, and warehouse-aware stock pressure.", Canonical: "/app/inventory"}, parseData, parseSession)
}

func (parseS *atlasServer) handleInventoryDetailPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseData, parseErr := parseS.internalInventoryDetailPageData(parseR.Context(), parseR.PathValue("sku"))
	if parseErr != nil {
		parsePath := "/app/inventory/" + parseR.PathValue("sku")
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas SKU Not Found", Description: "The requested inventory detail route could not be loaded.", Canonical: parsePath}, parseSession, "SKU not found", "That inventory item is not available in the current Atlas seed set.", "/app/inventory", "Back to inventory", parseErr)
		return
	}
	parsePath2 := "/app/inventory/" + parseR.PathValue("sku")
	parseS.renderPage(parseW, parseR, routeMeta{Path: parsePath2, Surface: "internal", Screen: "sku-detail", Title: "Atlas SKU Detail", Description: "Inspect warehouse breakdown, thresholds, and activity for a single Atlas SKU.", Canonical: parsePath2}, parseData, parseSession)
}

func (parseS *atlasServer) handleInventoryThresholdHistoryPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseSku := parseR.PathValue("sku")
	parsePageData, parseErr := parseS.internalInventoryDetailPageData(parseR.Context(), parseSku)
	if parseErr != nil {
		parsePath := "/app/inventory/" + parseSku + "/threshold-history"
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas Threshold History Unavailable", Description: "The threshold-history route could not be loaded for this SKU.", Canonical: parsePath}, parseSession, "Threshold history unavailable", "That inventory item is not available in the current Atlas seed set.", "/app/inventory", "Back to inventory", parseErr)
		return
	}
	parseOverlayData, parseErr := parseS.internalInventoryThresholdPanelPageData(parseR.Context(), parseSku)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "threshold_panel_query_failed", parseErr)
		return
	}
	parsePath2 := "/app/inventory/" + parseSku + "/threshold-history"
	parseRequests := startupRequestsForPage("/app/inventory/"+parseSku, parseR.URL.Query(), parsePageData)
	parseRequests["overlay"] = atlas.Request{
		Method: http.MethodGet,
		URL:    "/api/app/inventory/" + parseSku + "/threshold-panel",
		Status: http.StatusOK,
		Data:   map[string]any{"overlay": parseOverlayData},
	}
	parseS.renderPageStatusWithPayload(parseW, parseR, http.StatusOK, routeMeta{
		Path:        parsePath2,
		Surface:     "internal",
		Screen:      "sku-threshold-history",
		Title:       "Atlas Threshold History",
		Description: "Review threshold edits and transfer cues for one Atlas SKU without leaving the inventory route context.",
		Canonical:   parsePath2,
	}, map[string]any{
		"page":    parsePageData,
		"overlay": parseOverlayData,
	}, parseRequests, parseSession)
}

func (parseS *atlasServer) handleWarehouseOpsPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseData, parseErr := parseS.internalWarehouseOpsPageData(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "warehouse_pressure_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/app/warehouses", Surface: "internal", Screen: "warehouse-ops", Title: "Atlas Warehouse Operations", Description: "Compare staffing, backlog, service posture, and warehouse pressure across the Atlas network.", Canonical: "/app/warehouses"}, parseData, parseSession)
}

func (parseS *atlasServer) handleWarehouseOpsDetailPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parsePageData, parseErr := parseS.internalWarehouseOpsPageData(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "warehouse_pressure_query_failed", parseErr)
		return
	}
	parseDetailData, parseErr := parseS.internalWarehouseDetailPageDataWithFilters(parseR.Context(), parseR.PathValue("warehouseId"), parseR.URL.Query())
	if parseErr != nil {
		parsePath := "/app/warehouses/" + parseR.PathValue("warehouseId")
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas Warehouse Not Found", Description: "The requested internal warehouse route could not be loaded.", Canonical: parsePath}, parseSession, "Warehouse not found", "That internal warehouse route is not available in the current Atlas seed set.", "/app/warehouses", "Back to internal warehouses", parseErr)
		return
	}
	parsePath := "/app/warehouses/" + parseR.PathValue("warehouseId")
	parseRequests := startupRequestsForPage("/app/warehouses", parseR.URL.Query(), parsePageData)
	parseDetailURL := startupRequestURL(parsePath, atlasDataQuery(parseR.URL.Query()))
	if strings.TrimSpace(parseDetailURL) != "" {
		parseRequests["detail"] = atlas.Request{
			Method: http.MethodGet,
			URL:    parseDetailURL,
			Status: http.StatusOK,
			Data:   map[string]any{"detail": parseDetailData},
		}
	}
	parseS.renderPageStatusWithPayload(parseW, parseR, http.StatusOK, routeMeta{Path: parsePath, Surface: "internal", Screen: "warehouse-detail", Title: "Atlas Warehouse Detail", Description: "Inspect staffing, backlog, and next action for a single Atlas warehouse.", Canonical: parsePath}, map[string]any{
		"page":   parsePageData,
		"detail": parseDetailData,
	}, parseRequests, parseSession)
}

func (parseS *atlasServer) handleWarehouseOpsItemPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseWarehouseID := parseR.PathValue("warehouseId")
	parsePath := "/app/warehouses/" + parseWarehouseID + "/items/" + parseR.PathValue("sku")
	parsePageData, parseErr := parseS.internalWarehouseOpsPageData(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "warehouse_pressure_query_failed", parseErr)
		return
	}
	parseDetailData, parseErr := parseS.internalWarehouseDetailPageDataWithFilters(parseR.Context(), parseWarehouseID, parseR.URL.Query())
	if parseErr != nil {
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas Warehouse Item Not Found", Description: "The requested warehouse item route could not be loaded.", Canonical: parsePath}, parseSession, "Warehouse item not found", "That warehouse item is not available in the current Atlas seed set for this facility.", "/app/warehouses/"+parseWarehouseID, "Back to warehouse items", parseErr)
		return
	}
	parseItem, parseErr := parseS.internalWarehouseItemPageData(parseR.Context(), parseWarehouseID, parseR.PathValue("sku"), parseR.URL.Query())
	if parseErr != nil {
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas Warehouse Item Not Found", Description: "The requested warehouse item route could not be loaded.", Canonical: parsePath}, parseSession, "Warehouse item not found", "That warehouse item is not available in the current Atlas seed set for this facility.", "/app/warehouses/"+parseR.PathValue("warehouseId"), "Back to warehouse items", parseErr)
		return
	}
	parseRequests := startupRequestsForPage("/app/warehouses", parseR.URL.Query(), parsePageData)
	parseDetailPath := "/app/warehouses/" + parseWarehouseID
	parseDetailURL := startupRequestURL(parseDetailPath, atlasDataQuery(parseR.URL.Query()))
	if strings.TrimSpace(parseDetailURL) != "" {
		parseRequests["detail"] = atlas.Request{
			Method: http.MethodGet,
			URL:    parseDetailURL,
			Status: http.StatusOK,
			Data:   map[string]any{"detail": parseDetailData},
		}
	}
	parseRequests["item"] = atlas.Request{
		Method: http.MethodGet,
		URL:    startupRequestURL(parsePath, atlasDataQuery(parseR.URL.Query())),
		Status: http.StatusOK,
		Data:   map[string]any{"item": parseItem},
	}
	parseS.renderPageStatusWithPayload(parseW, parseR, http.StatusOK, routeMeta{Path: parsePath, Surface: "internal", Screen: "warehouse-item-detail", Title: "Atlas Warehouse Item", Description: "Manage one warehouse item with inventory edits, replenishment, and demand context.", Canonical: parsePath}, map[string]any{
		"page":   parsePageData,
		"detail": parseDetailData,
		"item":   parseItem,
	}, parseRequests, parseSession)
}

func (parseS *atlasServer) handleTransfersPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseItems, parseErr := parseS.store.Transfers(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "transfer_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/app/transfers", Surface: "internal", Screen: "transfers", Title: "Atlas Transfers", Description: "Plan, review, and approve cross-warehouse transfer recommendations.", Canonical: "/app/transfers"}, map[string]any{"items": parseItems}, parseSession)
}

func (parseS *atlasServer) handleTransferDetailPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseItem, parseErr := parseS.store.TransferDetail(parseR.Context(), parseR.PathValue("id"))
	if parseErr != nil {
		parsePath := "/app/transfers/" + parseR.PathValue("id")
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas Transfer Not Found", Description: "The requested transfer detail route could not be loaded.", Canonical: parsePath}, parseSession, "Transfer not found", "That transfer detail route is not available in the current Atlas seed set.", "/app/transfers", "Back to transfers", parseErr)
		return
	}
	parsePath2 := "/app/transfers/" + parseR.PathValue("id")
	parseS.renderPage(parseW, parseR, routeMeta{Path: parsePath2, Surface: "internal", Screen: "transfer-detail", Title: "Atlas Transfer Detail", Description: "Inspect one Atlas transfer, including approval state, lane context, and audit activity.", Canonical: parsePath2}, parseItem, parseSession)
}

func (parseS *atlasServer) handlePurchaseOrdersPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseData, parseErr := parseS.internalPurchaseOrdersPageData(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "purchase_orders_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/app/purchase-orders", Surface: "internal", Screen: "purchase-orders", Title: "Atlas Purchase Orders", Description: "Review vendor approvals, inbound shipment rows, and purchase-order planning context.", Canonical: "/app/purchase-orders"}, parseData, parseSession)
}

func (parseS *atlasServer) handlePurchaseOrderDetailPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parsePageData, parseErr := parseS.internalPurchaseOrdersPageData(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "purchase_orders_query_failed", parseErr)
		return
	}
	parseItem, parseErr := parseS.store.PurchaseOrderDetail(parseR.Context(), parseR.PathValue("id"))
	if parseErr != nil {
		parsePath := "/app/purchase-orders/" + parseR.PathValue("id")
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas Purchase Order Not Found", Description: "The requested purchase-order detail route could not be loaded.", Canonical: parsePath}, parseSession, "Purchase order not found", "That purchase-order route is not available in the current Atlas seed set.", "/app/purchase-orders", "Back to purchase orders", parseErr)
		return
	}
	parsePath := "/app/purchase-orders/" + parseR.PathValue("id")
	parseRequests := startupRequestsForPage("/app/purchase-orders", parseR.URL.Query(), parsePageData)
	parseDetailURL := startupRequestURL(parsePath, atlasDataQuery(parseR.URL.Query()))
	if strings.TrimSpace(parseDetailURL) != "" {
		parseRequests["detail"] = atlas.Request{
			Method: http.MethodGet,
			URL:    parseDetailURL,
			Status: http.StatusOK,
			Data:   map[string]any{"detail": parseItem},
		}
	}
	parseS.renderPageStatusWithPayload(parseW, parseR, http.StatusOK, routeMeta{Path: parsePath, Surface: "internal", Screen: "purchase-order-detail", Title: "Atlas Purchase Order Detail", Description: "Inspect one Atlas purchase order, including vendor state, ETA, and inbound shipment rows.", Canonical: parsePath}, map[string]any{
		"page":   parsePageData,
		"detail": parseItem,
	}, parseRequests, parseSession)
}

func (parseS *atlasServer) handleReceivingPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseItems, parseErr := parseS.store.ReceivingSessions(parseR.Context())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "receiving_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/app/receiving", Surface: "internal", Screen: "receiving", Title: "Atlas Receiving", Description: "Track inbound sessions, discrepancies, and receiving closeout state.", Canonical: "/app/receiving"}, map[string]any{"items": parseItems}, parseSession)
}

func (parseS *atlasServer) handleReceivingDetailPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseItem, parseErr := parseS.store.ReceivingDetail(parseR.Context(), parseR.PathValue("id"))
	if parseErr != nil {
		parsePath := "/app/receiving/" + parseR.PathValue("id")
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas Receiving Session Not Found", Description: "The requested receiving-session route could not be loaded.", Canonical: parsePath}, parseSession, "Receiving session not found", "That receiving-session route is not available in the current Atlas seed set.", "/app/receiving", "Back to receiving", parseErr)
		return
	}
	parsePath2 := "/app/receiving/" + parseR.PathValue("id")
	parseS.renderPage(parseW, parseR, routeMeta{Path: parsePath2, Surface: "internal", Screen: "receiving-session-detail", Title: "Atlas Receiving Session", Description: "Inspect one receiving session, including discrepancy classification and closeout readiness.", Canonical: parsePath2}, parseItem, parseSession)
}

func (parseS *atlasServer) handleCommentsPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseData, parseErr := parseS.internalCommentsPageData(parseR.Context(), strings.TrimSpace(parseR.URL.Query().Get("status")))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "comment_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: "/app/comments", Surface: "internal", Screen: "comments", Title: "Atlas Buyer Inbox", Description: "Review buyer questions, moderation decisions, and follow-up paths into product, inventory, or warehouse work.", Canonical: "/app/comments"}, parseData, parseSession)
}

func (parseS *atlasServer) handleCommentsModerationPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseStatus := strings.TrimSpace(parseR.PathValue("status"))
	if parseStatus == "" {
		parseStatus = "pending"
	}
	parseData, parseErr := parseS.internalCommentsPageData(parseR.Context(), parseStatus)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "comment_query_failed", parseErr)
		return
	}
	parsePath := "/app/comments/moderation/" + parseStatus
	parseS.renderPage(parseW, parseR, routeMeta{Path: parsePath, Surface: "internal", Screen: "comments-moderation", Title: "Atlas Buyer Inbox Moderation", Description: "Filter Atlas buyer inbox records by moderation posture and review queue-level decisions in one route.", Canonical: parsePath}, parseData, parseSession)
}

func (parseS *atlasServer) handleCommentDetailPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseCommentID := strings.TrimSpace(parseR.PathValue("id"))
	parseData, parseErr := parseS.internalCommentsPageData(parseR.Context(), "")
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "comment_query_failed", parseErr)
		return
	}
	isParseFound := false
	for _, parseItem := range parseData.Items {
		if strings.EqualFold(strings.TrimSpace(parseItem.ID), parseCommentID) {
			isParseFound = true
			break
		}
	}
	parsePath := "/app/comments/" + parseCommentID
	if !isParseFound {
		parseErr2 := fmt.Errorf("comment %s not found", parseCommentID)
		parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, routeMeta{Path: parsePath, Surface: "internal", Screen: "recovery", Title: "Atlas Comment Not Found", Description: "The requested comment detail route could not be loaded.", Canonical: parsePath}, parseSession, "Comment not found", "That buyer inbox record is not available in the current Atlas seed set.", "/app/comments", "Back to comments", parseErr2)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: parsePath, Surface: "internal", Screen: "comment-detail", Title: "Atlas Buyer Comment Detail", Description: "Inspect one buyer inbox record while keeping moderation actions and route context in view.", Canonical: parsePath}, parseData, parseSession)
}

func (parseS *atlasServer) handleSettingsPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseS.handleSettingsSubroutePage(parseW, parseR, "/app/settings", "settings", "Atlas Settings", "Manage theme, locale, density, default warehouse, and saved-view preferences.")
}

func (parseS *atlasServer) handleSettingsAppearancePage(parseW http.ResponseWriter, parseR *http.Request) {
	parseS.handleSettingsSubroutePage(parseW, parseR, "/app/settings/appearance", "settings-appearance", "Atlas Settings Appearance", "Tune Atlas theme and density preferences for the internal shell workspace.")
}

func (parseS *atlasServer) handleSettingsLocalePage(parseW http.ResponseWriter, parseR *http.Request) {
	parseS.handleSettingsSubroutePage(parseW, parseR, "/app/settings/locale", "settings-locale", "Atlas Settings Locale", "Review locale behavior and language direction settings for Atlas operator routes.")
}

func (parseS *atlasServer) handleSettingsWorkspaceDefaultsPage(parseW http.ResponseWriter, parseR *http.Request) {
	parseS.handleSettingsSubroutePage(parseW, parseR, "/app/settings/workspace-defaults", "settings-workspace-defaults", "Atlas Settings Workspace Defaults", "Manage default warehouse routing and saved-view workspace defaults for Atlas operators.")
}

func (parseS *atlasServer) handleSettingsSubroutePage(parseW http.ResponseWriter, parseR *http.Request, parsePath string, parseScreen string, parseTitle string, parseDescription string) {
	parseSession := parseS.sessions.RequireInternalSession(parseW, parseR)
	if parseSession == nil {
		return
	}
	parseData, parseErr := parseS.internalSettingsPageData(parseR.Context(), parseSession.UserID)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "settings_query_failed", parseErr)
		return
	}
	parseS.renderPage(parseW, parseR, routeMeta{Path: parsePath, Surface: "internal", Screen: parseScreen, Title: parseTitle, Description: parseDescription, Canonical: parsePath}, parseData, parseSession)
}

func (parseS *atlasServer) handleRouteRecoveryPage(parseW http.ResponseWriter, parseR *http.Request) {
	var parseSession *serverauth.Session
	if strings.HasPrefix(parseR.URL.Path, "/app/") {
		parseSession = parseS.sessions.RequireInternalSession(parseW, parseR)
		if parseSession == nil {
			return
		}
	}
	parseMeta := routeMeta{Path: parseR.URL.Path, Surface: "public", Screen: "recovery", Title: "Atlas Route Not Found", Description: "The requested Atlas route could not be resolved.", Canonical: parseR.URL.Path}
	parseRecoveryPath := "/shop"
	parseRecoveryLabel := "Back to shop"
	parseTitle := "Route not found"
	parseMessage := "The requested Atlas route is not part of the current demo route set."
	if strings.HasPrefix(parseR.URL.Path, "/app/") {
		parseMeta.Surface = "internal"
		parseMeta.Title = "Atlas Internal Route Not Found"
		parseRecoveryPath = "/app/dashboard"
		parseRecoveryLabel = "Back to dashboard"
		parseMessage = "The requested Atlas internal route is not part of the current demo route set."
	}
	parseS.renderRecoveryPage(parseW, parseR, http.StatusNotFound, parseMeta, parseSession, parseTitle, parseMessage, parseRecoveryPath, parseRecoveryLabel, nil)
}

func (parseS *atlasServer) renderPage(parseW http.ResponseWriter, parseR *http.Request, parseMeta routeMeta, parseData any, parseSession *serverauth.Session) {
	parseS.renderPageStatus(parseW, parseR, http.StatusOK, parseMeta, parseData, parseSession)
}

func (parseS *atlasServer) renderPageStatus(parseW http.ResponseWriter, parseR *http.Request, parseStatus int, parseMeta routeMeta, parseData any, parseSession *serverauth.Session) {
	parseS.renderPageStatusWithPayload(parseW, parseR, parseStatus, parseMeta, map[string]any{"page": parseData}, startupRequestsForPage(parseMeta.Path, parseR.URL.Query(), parseData), parseSession)
}

func (parseS *atlasServer) renderPageStatusWithPayload(parseW http.ResponseWriter, parseR *http.Request, parseStatus int, parseMeta routeMeta, parsePayloadData map[string]any, parseRequests map[string]atlas.Request, parseSession *serverauth.Session) {
	// SSR boundary: all page handlers converge into one bootstrap contract so server HTML, route metadata,
	// and hydrated client state stay sourced from the same payload shape.
	parsePayload, _, parseErr := parseS.bootstrapForPath(parseR, parseMeta.Path, parseSession)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "bootstrap_failed", parseErr)
		return
	}
	parsePayload.Route.Surface = parseMeta.Surface
	parsePayload.Route.Screen = parseMeta.Screen
	parsePayload.Route.Title = parseMeta.Title
	parsePayload.Route.Description = parseMeta.Description
	parsePayload.Route.Canonical = parseMeta.Canonical
	parsePayload.CSRF = ensureCSRFCookie(parseW, parseR)
	// Level (1) of the locale contract just resolved inside bootstrapForPath; record
	// it so the next link without a ?locale= parameter keeps the language.
	persistLocaleCookie(parseW, parseR.URL.Query())
	parsePayload.Data = parsePayloadData
	parsePayload.Requests = parseRequests
	parseBootstrapScript, parseBootstrapBytes, parseBootstrapMode, parseErr := parseS.renderBootstrapScript(parseMeta.Path, parseR.URL.Query(), parsePayload)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "bootstrap_encode_failed", parseErr)
		return
	}
	parseW.Header().Set("X-Atlas-Bootstrap-Bytes", fmt.Sprintf("%d", parseBootstrapBytes))
	parseW.Header().Set("X-Atlas-Bootstrap-Mode", parseBootstrapMode)
	parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
	parseW.WriteHeader(parseStatus)
	// Rendering boundary: metadata tags and bootstrap script are emitted together so direct-entry SSR and
	// post-hydration client navigation both start from equivalent route semantics.
	//
	// SCRIPT INVENTORY — this document ships exactly two scripts and one inline snippet, and that is the
	// whole JavaScript surface of Atlas:
	//
	//   1. <script src="/assets/script/wasm_exec.js">  the Go toolchain's own glue from $GOROOT/lib/wasm.
	//      It implements the host half of the Go wasm ABI (syscall/js value table, runtime.wasmWrite,
	//      timers, memory growth). It is the one irreducible piece: Go code cannot bootstrap the runtime
	//      that runs it, and this file is version-locked to the compiler, so it is copied, never authored.
	//   2. <script id="__ATLAS_BOOTSTRAP__" type="application/json">  data, not code — the SSR payload the
	//      client hydrates from. It is emitted by Go and parsed by Go.
	//   3. the inline snippet from wasmRuntimeSnippet  ~20 lines whose only jobs are instantiating the
	//      module and revealing a server-rendered failure message if that does not work.
	//
	// example-logger.js used to be linked here as a fourth script. It set window.__gwcExampleLogger to
	// four console wrappers, nothing in Atlas ever called it, and Go reaches console directly through
	// syscall/js when it wants to (see debugLog in client/main.go). It is gone from this document. The
	// file itself stays in examples/static/script because ~290 other example pages link it.
	//
	// STYLESHEET INVENTORY — there are none. The document links no CSS at all and carries one
	// inline <style data-gwc-css> instead. See atlasDesignStyleBlock for what came out and why.
	parseIsWasmPresent := fileExists(parseS.cfg.AtlasWASM)
	_, _ = fmt.Fprintf(parseW, "<!DOCTYPE html><html lang=%q dir=%q class=%q data-atlas-surface=%q data-atlas-debug-logs=%q><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title data-gwc-router-managed=\"true\">%s</title><meta name=\"description\" content=%q data-gwc-router-managed=\"true\"><link rel=\"canonical\" href=%q data-gwc-router-managed=\"true\">%s<script src=%q></script></head><body><div id=\"app\"></div>%s%s%s</body></html>", parsePayload.I18n.Locale, parsePayload.I18n.Direction, atlasDocumentClass(parsePayload), parsePayload.Route.Surface, formatAtlasDebugLogsFlag(parseS.cfg.LogsEnabled), parseMeta.Title, parseMeta.Description, parseMeta.Canonical, atlasDesignStyleBlock(), atlasWASMExecURL, atlasBootFallbackMarkup(parseIsWasmPresent), parseBootstrapScript, wasmRuntimeSnippet(parseIsWasmPresent))
}

// atlasDesignStyleBlock returns the <style> element that carries Atlas's entire
// base layer, and it is the reason this document links no stylesheet at all.
//
// # WHY A GO PACKAGE REPLACED A 10,646-LINE STYLESHEET
//
// The document used to link two files:
//
//   - /assets/css/tailwind.css — 10,646 lines of Tailwind v4 build output, shared
//     with ~145 other examples in this repo. Atlas used a few hundred of those
//     utilities, shipped the rest, and paid for a build step (and a scan-bait file,
//     examples/static/generated/tailwind-manifest.html, whose only job was to stop
//     the compiler tree-shaking classes it could not see) to get them. The example's
//     whole claim is that a browser app can be written in Go; an external CSS build
//     is the largest remaining hole in that claim.
//   - /assets/css/example-shell.css — 51 lines, and the more instructive of the two.
//     It hard-set `color-scheme: dark` and painted three dark gradients with
//     `!important`. That is why Atlas reported THEME: light while rendering
//     near-black: `!important` on a global element selector cannot be beaten by
//     anything a component emits, at any specificity, so the app's own theme state
//     was decorative. An `!important` theme lock is not a strong default, it is a
//     lock — the app can no longer be wrong about its own appearance in a way a
//     developer can fix from the app. shared/design emits no `!important` anywhere
//     and declares `color-scheme: light dark`, so the theme follows the user.
//
// Neither FILE was deleted: both are linked by ~145 and ~164 other files
// respectively (examples/public-examples-site/**, examples/static/index.html, and
// the generator in tools/gwc/examples.go). Atlas simply stopped linking them.
//
// # WHAT IS IN THE BLOCK, AND WHY IT IS SAFE TO CACHE
//
// design.Install() emits the global layer — the framework reset, the :root light
// token block, the prefers-color-scheme dark override of the same names, the
// element baseline, the accessibility floor and the print token block — through the
// css package's Sink. On native that Sink is a process-wide buffer, so
// css.StyleBlock() serializes it into <style data-gwc-css="…"> with the emitted
// class names in the attribute; the client calls css.SeedFromDocument() and
// therefore recognizes every one of them as already present instead of re-injecting
// it after hydration.
//
// The result is computed once because the buffer sink only grows: harvesting it per
// request would make each document's <style> depend on how many requests had been
// served before it, which is the kind of non-determinism that makes a golden-file
// test flap on the CI machine and pass locally. Atlas's SSR emits no design-classed
// markup (the shell is `<div id="app"></div>` and every pixel is client-rendered),
// so the global layer IS the server's whole styling contribution and nothing
// request-scoped belongs in it.
//
// The trade this leaves open, stated plainly rather than hidden: the theme now
// follows prefers-color-scheme, so the `atlas-theme-light`/`atlas-theme-dark`
// classes on <html> are a data channel for the client's hydration audit and no
// longer drive any pixels. Making the stored preference authoritative again means
// adding a class-scoped token block to shared/design (its tokens.go documents the
// pattern: css.Global(`[data-theme="dusk"]`, css.Raw(design.TokenPaper, …))) plus a
// contrast test for it, which belongs in that package and not in this one.
var atlasDesignStyleBlock = sync.OnceValue(func() string {
	// Install BEFORE the first render, per design.Install's contract: its emission
	// order (reset, then tokens, then baseline, then a11y floor, then print) is the
	// cascade order, and anything folded ahead of it would land above the reset.
	design.Install()
	return css.StyleBlock()
})

// atlasDocumentClass builds the theme+density class pair on <html>.
//
// HONEST STATUS, because the name promises more than it currently delivers: with the
// external stylesheets retired these two classes no longer paint anything. Nothing in
// shared/design selects on them — its dark theme is a prefers-color-scheme override
// of the same :root token names, so the visual theme follows the OS and the stored
// `theme` preference does not move a pixel.
//
// They are kept, and kept accurate, because they are still a live contract: the
// client's logHydrationDocumentMismatch asserts the document carries exactly the
// classes the bootstrap payload implies, which is how SSR/hydration drift gets
// caught. Making the preference authoritative again is a shared/design change — a
// class-scoped token block, per the pattern in its tokens.go, plus a contrast test
// for the new pairing — not a change here.
func atlasDocumentClass(parsePayload atlas.Payload) string {
	// SSR applies theme+density classes from the same preference payload the client hydrates from so
	// direct-entry pages do not flash between default styling and resumed user settings.
	parseThemeClass := "atlas-theme-dark"
	if strings.EqualFold(strings.TrimSpace(parsePayload.Theme.Mode), "light") {
		parseThemeClass = "atlas-theme-light"
	}
	parseDensityClass := "atlas-density-compact"
	if strings.EqualFold(strings.TrimSpace(parsePayload.Preferences.Density), "comfortable") {
		parseDensityClass = "atlas-density-comfortable"
	}
	return parseThemeClass + " " + parseDensityClass
}

// formatAtlasDebugLogsFlag serializes the debug-log document attribute for client bootstrap diagnostics.
func formatAtlasDebugLogsFlag(isEnabled bool) string {
	if isEnabled {
		return "1"
	}
	return "0"
}

func cloneURLValues(parseValues url.Values) url.Values {
	parseCloned := url.Values{}
	for parseKey, parseItems := range parseValues {
		parseCloned[parseKey] = append([]string(nil), parseItems...)
	}
	return parseCloned
}

func atlasDataQuery(parseValues url.Values) url.Values {
	parseFiltered := url.Values{}
	for parseKey, parseItems := range parseValues {
		parseTrimmedKey := strings.TrimSpace(strings.ToLower(parseKey))
		if parseTrimmedKey == atlasNoticeQueryKey || parseTrimmedKey == atlasBootstrapModeQueryKey {
			continue
		}
		for _, parseItem := range parseItems {
			parseFiltered.Add(parseKey, parseItem)
		}
	}
	return parseFiltered
}

func atlasBootstrapReferenceURL(parsePath string, parseQuery url.Values) string {
	parseValues := url.Values{}
	parseValues.Set("path", parsePath)
	if parseEncoded := cloneURLValues(parseQuery).Encode(); parseEncoded != "" {
		parseValues.Set("route_query", parseEncoded)
	}
	return "/__atlas/bootstrap.json?" + parseValues.Encode()
}

func atlasBootstrapMode(parseValues url.Values) string {
	if strings.EqualFold(strings.TrimSpace(parseValues.Get(atlasBootstrapModeQueryKey)), atlasBootstrapModeExternal) {
		return atlasBootstrapModeExternal
	}
	return "inline"
}

func supportsExternalBootstrap(parsePath string) bool {
	return strings.HasPrefix(parsePath, atlas.RouteInventory+"/") && strings.HasSuffix(parsePath, "/threshold-history")
}

func (parseS *atlasServer) renderBootstrapScript(parsePath string, parseQuery url.Values, parsePayload atlas.Payload) (string, int, string, error) {
	parseBootstrap := parsePayload.ToSSRBootstrap()
	parseEncoded, parseErr := ui.MarshalSSRBootstrap(parseBootstrap)
	if parseErr != nil {
		return "", 0, "", parseErr
	}
	parseMode := atlasBootstrapMode(parseQuery)
	if parseMode == atlasBootstrapModeExternal && supportsExternalBootstrap(parsePath) {
		parseScript, parseErr2 := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{
			URL:    atlasBootstrapReferenceURL(parsePath, parseQuery),
			Format: ui.SSRBootstrapFormatJSON,
		}, atlasBootstrapReferenceScriptID)
		return parseScript, len(parseEncoded), parseMode, parseErr2
	}
	parseMode = "inline"
	parseScript2, parseErr := ui.RenderBootstrapScript(parseBootstrap, atlasBootstrapScriptID)
	return parseScript2, len(parseEncoded), parseMode, parseErr
}

func startupRequestsForPage(parsePath string, parseQuery url.Values, parsePageData any) map[string]atlas.Request {
	parseRequestURL := startupRequestURL(parsePath, atlasDataQuery(parseQuery))
	if parseRequestURL == "" {
		return map[string]atlas.Request{}
	}
	return map[string]atlas.Request{
		"page": {
			Method: http.MethodGet,
			URL:    parseRequestURL,
			Status: http.StatusOK,
			Data:   map[string]any{"page": parsePageData},
		},
	}
}

func startupRequestURL(parsePath string, parseQuery url.Values) string {
	return atlas.StartupRequestURL(parsePath, parseQuery)
}

func (parseS *atlasServer) renderRecoveryPage(parseW http.ResponseWriter, parseR *http.Request, parseStatus int, parseMeta routeMeta, parseSession *serverauth.Session, parseTitle, parseMessage, parseRecoveryPath, parseRecoveryLabel string, parseErr error) {
	parsePage := map[string]any{
		"title":         parseTitle,
		"message":       parseMessage,
		"recoveryHref":  parseRecoveryPath,
		"recoveryLabel": parseRecoveryLabel,
	}
	if parseErr != nil {
		parsePage["detail"] = parseErr.Error()
	}
	parseS.renderPageStatus(parseW, parseR, parseStatus, parseMeta, parsePage, parseSession)
}

func (parseS *atlasServer) handleBootstrapJSON(parseW http.ResponseWriter, parseR *http.Request) {
	parseTargetPath := strings.TrimSpace(parseR.URL.Query().Get("path"))
	if parseTargetPath == "" {
		parseTargetPath = "/"
	}
	if !supportsExternalBootstrap(parseTargetPath) {
		parseS.writeValidationError(parseW, http.StatusBadRequest, "bootstrap_route_unsupported", "External bootstrap mode is only wired for the inventory threshold-history route today.", map[string]string{"path": "Use /app/inventory/{sku}/threshold-history for the proof-of-concept external bootstrap flow."})
		return
	}
	parseRouteQuery, parseErr := url.ParseQuery(strings.TrimSpace(parseR.URL.Query().Get("route_query")))
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusBadRequest, "bootstrap_query_invalid", parseErr)
		return
	}
	var parseSession *serverauth.Session
	if strings.HasPrefix(parseTargetPath, "/app/") {
		parseSession = parseS.sessions.RequireInternalSession(parseW, parseR)
		if parseSession == nil {
			return
		}
	}
	parseSku := strings.TrimSuffix(strings.TrimPrefix(parseTargetPath, atlas.RouteInventory+"/"), "/threshold-history")
	parsePageData, parseErr := parseS.internalInventoryDetailPageData(parseR.Context(), parseSku)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusNotFound, "inventory_item_not_found", parseErr)
		return
	}
	parseOverlayData, parseErr := parseS.internalInventoryThresholdPanelPageData(parseR.Context(), parseSku)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "threshold_panel_query_failed", parseErr)
		return
	}
	parsePayload, _, parseErr := parseS.bootstrapForRouteQuery(parseR, parseTargetPath, parseRouteQuery, parseSession)
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "bootstrap_failed", parseErr)
		return
	}
	parsePayload.Route.Screen = "sku-threshold-history"
	parsePayload.Route.Title = "Atlas Threshold History"
	parsePayload.Route.Description = "Review threshold edits and transfer cues for one Atlas SKU without leaving the inventory route context."
	parsePayload.Route.Canonical = parseTargetPath
	parsePayload.Data = map[string]any{
		"page":    parsePageData,
		"overlay": parseOverlayData,
	}
	parsePayload.Requests = startupRequestsForPage("/app/inventory/"+parseSku, parseRouteQuery, parsePageData)
	parsePayload.Requests["overlay"] = atlas.Request{
		Method: http.MethodGet,
		URL:    "/api/app/inventory/" + parseSku + "/threshold-panel",
		Status: http.StatusOK,
		Data:   map[string]any{"overlay": parseOverlayData},
	}
	parseEncoded, parseErr := ui.MarshalSSRBootstrap(parsePayload.ToSSRBootstrap())
	if parseErr != nil {
		parseS.writeError(parseW, http.StatusInternalServerError, "bootstrap_encode_failed", parseErr)
		return
	}
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	parseW.Header().Set("X-Atlas-Bootstrap-Bytes", fmt.Sprintf("%d", len(parseEncoded)))
	parseW.WriteHeader(http.StatusOK)
	_, _ = parseW.Write(parseEncoded)
}

func (parseS *atlasServer) bootstrapForPath(parseR *http.Request, parsePath string, parseSession *serverauth.Session) (atlas.Payload, routeMeta, error) {
	return parseS.bootstrapForRouteQuery(parseR, parsePath, parseR.URL.Query(), parseSession)
}

func (parseS *atlasServer) bootstrapForRouteQuery(parseR *http.Request, parsePath string, parseRouteQuery url.Values, parseSession *serverauth.Session) (atlas.Payload, routeMeta, error) {
	parseMeta := routeMetaForPath(parsePath)
	parsePreferences, _ := parseS.store.PreferencesByOwner(parseR.Context(), sessionOwnerID(parseSession))
	// One locale for every surface. This used to be gated on
	// `parseSession == nil && parseMeta.Surface == "public"`, which meant an operator
	// who pasted ?locale=ar into an /app URL silently got English — two different
	// answers to "what language is this page" depending on who was signed in.
	// resolveRequestLocale is now the single answer; see its doc for the precedence.
	parsePreferences.Locale = resolveRequestLocale(parseR, parseRouteQuery, parsePreferences.Locale)
	parseSavedViews, _ := parseS.store.SavedViewsByOwner(parseR.Context(), sessionOwnerID(parseSession))
	parseQuery := cloneQuery(parseRouteQuery)
	parsePayload := atlas.Payload{
		Route: atlas.RouteBootstrap{
			Path:        parsePath,
			Query:       parseQuery,
			Params:      map[string]string{},
			Surface:     parseMeta.Surface,
			Screen:      parseMeta.Screen,
			Title:       parseMeta.Title,
			Description: parseMeta.Description,
			Canonical:   parseMeta.Canonical,
		},
		Preferences: atlas.PreferencesState{
			Theme:            parsePreferences.Theme,
			Locale:           parsePreferences.Locale,
			Density:          parsePreferences.Density,
			DefaultWarehouse: parsePreferences.DefaultWarehouseID,
		},
		I18n: atlas.I18nState{
			// Locale and direction are part of SSR bootstrap so document `lang` and route copy direction
			// stay aligned before any client-side preference restoration runs.
			Locale:           nonEmpty(parsePreferences.Locale, "en"),
			SupportedLocales: atlas.SupportedLocales(),
			Direction:        localeDirection(parsePreferences.Locale),
		},
		Theme: atlas.ThemeState{
			Mode:                 nonEmpty(parsePreferences.Theme, "dark"),
			PrefersReducedMotion: false,
		},
		SavedViews: toSavedViewPayloads(parseSavedViews),
		Data: map[string]any{
			"canonical":   parseMeta.Canonical,
			"description": parseMeta.Description,
		},
	}
	if parseSession != nil {
		parsePayload.User = &atlas.UserSession{
			ID:               parseSession.UserID,
			DisplayName:      parseSession.DisplayName,
			Role:             parseSession.Role,
			DefaultWarehouse: parseSession.DefaultWarehouse,
		}
	}
	return parsePayload, parseMeta, nil
}

func routeMetaForPath(parsePath string) routeMeta {
	switch {
	case parsePath == "/":
		return routeMeta{Path: parsePath, Surface: "public", Screen: "landing", Title: "Atlas Commerce OS", Description: "Premium modular workspace systems with warehouse-aware availability.", Canonical: parsePath}
	case parsePath == "/shop":
		return routeMeta{Path: parsePath, Surface: "public", Screen: "catalog", Title: "Atlas Shop", Description: "Browse Atlas modular workspace systems with filterable discovery and regional fulfillment context.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/shop/"):
		return routeMeta{Path: parsePath, Surface: "public", Screen: "product", Title: "Atlas Product", Description: "Warehouse-aware availability and premium workspace design for Atlas products.", Canonical: parsePath}
	case parsePath == "/warehouses":
		return routeMeta{Path: parsePath, Surface: "public", Screen: "warehouses", Title: "Atlas Delivery Regions", Description: "Compare Atlas delivery regions, service levels, and stocked highlights before opening a warehouse route.", Canonical: parsePath}
	case parsePath == serverauth.MockSignInPath:
		return routeMeta{Path: parsePath, Surface: "public", Screen: "mock-sign-in", Title: "Atlas Mock Sign In", Description: "Start a mock internal session for the Atlas operator console.", Canonical: parsePath}
	case strings.Contains(parsePath, "/availability/"):
		return routeMeta{Path: parsePath, Surface: "public", Screen: "warehouse-availability", Title: "Atlas Warehouse Availability", Description: "Inspect a warehouse-specific product promise for one Atlas item and one regional fulfillment hub.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/warehouses/"):
		return routeMeta{Path: parsePath, Surface: "public", Screen: "warehouse-detail", Title: "Atlas Warehouse Region", Description: "Inspect one Atlas delivery region, including service posture, stocked highlights, and product-specific availability links.", Canonical: parsePath}
	case parsePath == "/app/dashboard":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "dashboard", Title: "Atlas Ops Dashboard", Description: "Operational overview of buyer questions, stock pressure, receiving exceptions, and warehouse health.", Canonical: parsePath}
	case parsePath == "/app/products":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "products", Title: "Atlas Product Merchandising", Description: "Manage product copy, pricing, launch posture, and merchandising details for Atlas workspace systems.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/app/products/"):
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "product-editor", Title: "Atlas Product Editor", Description: "Edit product copy, pricing, and volume for one Atlas storefront item.", Canonical: parsePath}
	case parsePath == "/app/inventory":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "inventory", Title: "Atlas Inventory", Description: "Review inventory health, saved views, and warehouse-aware stock pressure.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/app/inventory/") && strings.HasSuffix(parsePath, "/threshold-history"):
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "sku-threshold-history", Title: "Atlas Threshold History", Description: "Review threshold edits and transfer cues for one Atlas SKU without leaving the inventory route context.", Canonical: parsePath}
	case parsePath == "/app/warehouses":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "warehouse-ops", Title: "Atlas Warehouse Operations", Description: "Compare staffing, backlog, service posture, and warehouse pressure across the Atlas network.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/app/warehouses/"):
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "warehouse-detail", Title: "Atlas Warehouse Detail", Description: "Inspect staffing, backlog, and next action for a single Atlas warehouse.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/app/inventory/"):
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "sku-detail", Title: "Atlas SKU Detail", Description: "Inspect warehouse breakdown, thresholds, and activity for a single Atlas SKU.", Canonical: parsePath}
	case parsePath == "/app/transfers":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "transfers", Title: "Atlas Transfers", Description: "Plan, review, and approve cross-warehouse transfer recommendations.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/app/transfers/"):
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "transfer-detail", Title: "Atlas Transfer Detail", Description: "Inspect one Atlas transfer, including approval state, lane context, and audit activity.", Canonical: parsePath}
	case parsePath == "/app/purchase-orders":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "purchase-orders", Title: "Atlas Purchase Orders", Description: "Review vendor approvals, inbound shipment rows, and purchase-order planning context.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/app/purchase-orders/"):
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "purchase-order-detail", Title: "Atlas Purchase Order Detail", Description: "Inspect one Atlas purchase order, including vendor state, ETA, and inbound shipment rows.", Canonical: parsePath}
	case parsePath == "/app/receiving":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "receiving", Title: "Atlas Receiving", Description: "Track inbound sessions, discrepancies, and receiving closeout state.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/app/receiving/"):
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "receiving-session-detail", Title: "Atlas Receiving Session", Description: "Inspect one receiving session, including discrepancy classification and closeout readiness.", Canonical: parsePath}
	case parsePath == "/app/comments":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "comments", Title: "Atlas Buyer Inbox", Description: "Review buyer questions, moderation decisions, and follow-up paths into product, inventory, or warehouse work.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/app/comments/moderation/"):
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "comments-moderation", Title: "Atlas Buyer Inbox Moderation", Description: "Filter Atlas buyer inbox records by moderation posture and review queue-level decisions in one route.", Canonical: parsePath}
	case strings.HasPrefix(parsePath, "/app/comments/"):
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "comment-detail", Title: "Atlas Buyer Comment Detail", Description: "Inspect one buyer inbox record while keeping moderation actions and route context in view.", Canonical: parsePath}
	case parsePath == "/app/settings/appearance":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "settings-appearance", Title: "Atlas Settings Appearance", Description: "Tune Atlas theme and density preferences for the internal shell workspace.", Canonical: parsePath}
	case parsePath == "/app/settings/locale":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "settings-locale", Title: "Atlas Settings Locale", Description: "Review locale behavior and language direction settings for Atlas operator routes.", Canonical: parsePath}
	case parsePath == "/app/settings/workspace-defaults":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "settings-workspace-defaults", Title: "Atlas Settings Workspace Defaults", Description: "Manage default warehouse routing and saved-view workspace defaults for Atlas operators.", Canonical: parsePath}
	case parsePath == "/app/settings":
		return routeMeta{Path: parsePath, Surface: "internal", Screen: "settings", Title: "Atlas Settings", Description: "Manage theme, locale, density, default warehouse, and saved-view preferences.", Canonical: parsePath}
	default:
		return routeMeta{Path: parsePath, Surface: "public", Screen: "recovery", Title: "Atlas Route Not Found", Description: "Atlas recovery route.", Canonical: parsePath}
	}
}

func sessionOwnerID(parseSession *serverauth.Session) string {
	if parseSession == nil {
		return "public"
	}
	return parseSession.UserID
}

// atlasLocaleCookieName is the sticky half of the locale contract.
//
// It is deliberately NOT HttpOnly: the wasm client writes the same cookie when a
// client-side navigation changes the locale (see resolveClientLocale in
// client/main.go), which is what keeps a single-page navigation and a hard reload
// agreeing about the language. A cookie only the server can write would make the
// two lanes disagree the moment the router took over.
const atlasLocaleCookieName = "atlas_locale"

// atlasLocaleCookieMaxAge is one year in seconds. Language choice is a long-lived
// preference, and a session cookie here would silently revert the storefront to
// English every time the browser restarted.
const atlasLocaleCookieMaxAge = 365 * 24 * 60 * 60

// resolveRequestLocale is THE LOCALE CONTRACT. Precedence, highest first:
//
//  1. the `locale` query parameter, when it names a supported locale. Explicit and
//     shareable: a pasted URL must show its recipient the language it encodes,
//     whoever they are and whatever they previously chose. It wins on both surfaces
//     and it wins over a signed-in operator's saved preference.
//  2. the atlas_locale cookie, when it names a supported locale. This is the memory
//     of a previous (1): the language switcher only decorates the CURRENT path, so
//     without the cookie the very next link — /warehouses, which carries no locale
//     parameter — would drop the visitor back to English. persistLocaleCookie
//     writes it whenever (1) fires.
//  3. the owner's stored preference row (/app/settings for a signed-in operator,
//     the shared "public" row otherwise).
//  4. "en".
//
// Anything unrecognized at any level falls through to the next level rather than
// being echoed back: an unsupported tag in the URL is a 200 in the fallback
// language, never a reflected value in <html lang> and never a 404.
//
// WHY THIS PRECEDENCE AND NOT THE OTHER ONE: cookie-over-query is the arrangement
// that produces the classic bug — a link shared into a French chat renders English
// for everyone who ever clicked EN, and no one can reproduce it because the state is
// invisible and per-browser. URL-wins keeps the visible thing authoritative and
// leaves the invisible thing as memory only.
func resolveRequestLocale(parseR *http.Request, parseRouteQuery url.Values, parseStoredLocale string) string {
	if parseLocale, parseOK := supportedLocale(parseRouteQuery.Get("locale")); parseOK {
		return parseLocale
	}
	if parseCookie, parseErr := parseR.Cookie(atlasLocaleCookieName); parseErr == nil {
		if parseLocale, parseOK := supportedLocale(parseCookie.Value); parseOK {
			return parseLocale
		}
	}
	if parseLocale, parseOK := supportedLocale(parseStoredLocale); parseOK {
		return parseLocale
	}
	return "en"
}

// supportedLocale validates one candidate tag against atlas.SupportedLocales and
// returns it in the canonical (lower-case) spelling the payload and <html lang> use.
//
// Validating against the same list the switcher renders from is the whole defence
// here: the value lands in an HTML attribute and in a Set-Cookie header, so an
// allow-list — not escaping — is what keeps `?locale="><script>` from being a
// question anyone has to think about.
func supportedLocale(parseCandidate string) (string, bool) {
	parseCandidate = strings.ToLower(strings.TrimSpace(parseCandidate))
	if parseCandidate == "" {
		return "", false
	}
	for _, parseSupported := range atlas.SupportedLocales() {
		if parseCandidate == strings.ToLower(parseSupported) {
			return parseSupported, true
		}
	}
	return "", false
}

// persistLocaleCookie makes an explicit ?locale= choice sticky for later requests
// that do not carry one.
//
// It fires only when the QUERY supplied a valid locale, never on the cookie or
// preference path. Re-writing the cookie on every request would let level (2) of the
// contract refresh itself forever and quietly outlive the choice that created it;
// writing it only on level (1) means the cookie always records a decision a visitor
// actually made in a URL.
//
// SameSite=Lax rather than None: the cookie is read on top-level GETs only, and Lax
// is what makes a cross-site link that lands on Atlas still see it. Not HttpOnly, by
// design — see atlasLocaleCookieName.
func persistLocaleCookie(parseW http.ResponseWriter, parseRouteQuery url.Values) {
	parseLocale, parseOK := supportedLocale(parseRouteQuery.Get("locale"))
	if !parseOK {
		return
	}
	http.SetCookie(parseW, &http.Cookie{
		Name:     atlasLocaleCookieName,
		Value:    parseLocale,
		Path:     "/",
		MaxAge:   atlasLocaleCookieMaxAge,
		SameSite: http.SameSiteLaxMode,
	})
}

func cloneQuery(parseValues url.Values) map[string][]string {
	parseClone := map[string][]string{}
	for parseKey, parseItem := range parseValues {
		parseClone[parseKey] = append([]string(nil), parseItem...)
	}
	return parseClone
}

func toSavedViewPayloads(parseViews []repository.SavedView) []atlas.SavedViewPayload {
	parsePayloads := make([]atlas.SavedViewPayload, 0, len(parseViews))
	for _, parseView := range parseViews {
		parsePayloads = append(parsePayloads, atlas.SavedViewPayload{
			Name:          parseView.Name,
			Scope:         parseView.Scope,
			SortKey:       parseView.SortKey,
			SortDirection: parseView.SortDirection,
			Filters:       map[string]string{"density": parseView.Density, "warehouse": parseView.WarehouseID},
		})
	}
	return parsePayloads
}

func (parseS *atlasServer) writeJSON(parseW http.ResponseWriter, parseStatus int, parsePayload any) {
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	parseW.WriteHeader(parseStatus)
	_ = json.NewEncoder(parseW).Encode(parsePayload)
}

func (parseS *atlasServer) writeError(parseW http.ResponseWriter, parseStatus int, parseCode string, parseErr error) {
	parseS.writeJSON(parseW, parseStatus, map[string]any{
		"error":   parseCode,
		"message": strings.ReplaceAll(parseCode, "_", " "),
		"detail":  parseErr.Error(),
	})
}

func (parseS *atlasServer) writeValidationError(parseW http.ResponseWriter, parseStatus int, parseCode string, parseSummary string, parseFields map[string]string) {
	parseS.writeJSON(parseW, parseStatus, map[string]any{
		"error":   parseCode,
		"message": parseSummary,
		"fields":  parseFields,
	})
}

func parsePositiveInt(parseValue string, parseFallback int) int {
	if parseValue == "" {
		return parseFallback
	}
	var parseParsed int
	if _, parseErr := fmt.Sscanf(parseValue, "%d", &parseParsed); parseErr != nil || parseParsed < 1 {
		return parseFallback
	}
	return parseParsed
}

// atlasWASMAssetURL is the URL the browser fetches for the client bundle. It is
// built from atlasWASMAssetPath so it cannot drift from config.AtlasWASM, which
// is the same path resolved under StaticDir.
const atlasWASMAssetURL = "/assets/" + atlasWASMAssetPath

// atlasWASMExecURL is the browser-visible URL of the Go toolchain's own wasm
// glue. It is named here rather than only in the <head> string so the fallback
// copy can tell a reader exactly which request failed when `Go` is undefined.
const atlasWASMExecURL = "/assets/script/wasm_exec.js"

// atlasBootOptions is the set of deployment facts the boot layer needs to be
// specific instead of vague. All three are derived from the same constants the
// document and the presence check use, so the copy cannot name a URL the server
// does not actually serve.
func atlasBootOptions() bootfallback.Options {
	return bootfallback.Options{
		WASMURL:      atlasWASMAssetURL,
		LoaderURL:    atlasWASMExecURL,
		BuildCommand: atlasWASMBuildCommand,
	}
}

// atlasBootFallbackMarkup is the server-rendered half of the boot story: the
// <noscript> payload plus the reason blocks the snippet reveals. It is Go, and
// deliberately so — the copy, the markup, and the inline styling are the parts a
// JavaScript bootstrap has no business owning, and keeping them here is what lets
// the snippet stay ~20 lines. See shared/bootfallback for the full rationale.
//
// It must be emitted after <div id="app"></div> and never inside it: markup
// inside the mount point gets diffed against the client tree during hydration.
func atlasBootFallbackMarkup(isWasmPresent bool) string {
	return bootfallback.Markup(atlasBootOptions(), isWasmPresent)
}

// wasmRuntimeSnippet returns the document's inline boot script.
//
// THIS IS THE ENTIRE HAND-WRITTEN JAVASCRIPT SURFACE OF ATLAS. The document links
// one other script, /assets/script/wasm_exec.js, which is the Go toolchain's own
// BSD-licensed glue copied out of $GOROOT/lib/wasm — it implements the host side
// of the Go wasm ABI and cannot be written in Go, because it is what makes Go code
// runnable in the first place. Everything else the page does is compiled Go.
//
// With the module absent this returns "" rather than a console.warn. The previous
// version logged a warning nobody read while the page rendered blank; the server
// already knows the module is missing at render time, so
// atlasBootFallbackMarkup(false) puts that explanation on the screen instead and
// there is nothing left for a script to do. That is the zero-JavaScript path.
func wasmRuntimeSnippet(isWasmPresent bool) string {
	if !isWasmPresent {
		return ""
	}
	return bootfallback.BootScript(atlasBootOptions())
}

// atlasWASMStartupWarning returns the operator-facing warning for a missing
// client bundle, or "" when the bundle is present.
//
// This is the terminal half of the same message the browser now shows: the SSR
// shell renders an empty <div id="app"></div>, so a missing bundle means an empty
// page, and the operator who started the server is the only person who can fix
// it. The browser half is atlasBootFallbackMarkup(false), which renders visible
// in-page copy carrying the same URL and the same build command — the two are
// intentionally redundant, because the person who sees the page and the person who
// sees the log are often not the same person.
func atlasWASMStartupWarning(parseCfg config) string {
	if fileExists(parseCfg.AtlasWASM) {
		return ""
	}
	return fmt.Sprintf(
		"WARNING: client bundle missing at %s\n"+
			"         %s is served from that file, so every page will render the SSR shell and never hydrate.\n"+
			"         Pages will show an in-page explanation instead of rendering blank.\n"+
			"         Build it from the repo root with: %s\n",
		parseCfg.AtlasWASM,
		atlasWASMAssetURL,
		atlasWASMBuildCommand,
	)
}

func fileExists(parsePath string) bool {
	_, parseErr := os.Stat(parsePath)
	return parseErr == nil
}

func nonEmpty(parseValue, parseFallback string) string {
	if strings.TrimSpace(parseValue) == "" {
		return parseFallback
	}
	return parseValue
}

func roleLabel(parseRole string) string {
	switch parseRole {
	case "inventory_manager":
		return "Inventory Manager"
	case "warehouse_supervisor":
		return "Warehouse Supervisor"
	case "ops_lead":
		return "Operations Lead"
	default:
		return strings.ReplaceAll(parseRole, "_", " ")
	}
}

func roleDescription(parseRole string) string {
	switch parseRole {
	case "inventory_manager":
		return "Default inventory triage role with access to stock, receiving, and moderation surfaces."
	case "warehouse_supervisor":
		return "Warehouse-focused operator role for pressure review, receiving, and transfer follow-through."
	case "ops_lead":
		return "Cross-network operations role for dashboard review, vendor approvals, and workflow signoff."
	default:
		return "Mock Atlas internal role."
	}
}

func sanitizeNextPath(parseNext string) string {
	parseTrimmed := strings.TrimSpace(parseNext)
	if parseTrimmed == "" || !strings.HasPrefix(parseTrimmed, "/") || strings.HasPrefix(parseTrimmed, "//") {
		return "/app/dashboard"
	}
	parseParsed, parseErr := url.Parse(parseTrimmed)
	if parseErr != nil {
		return "/app/dashboard"
	}
	if parseCandidate := parseParsed.RequestURI(); strings.HasPrefix(parseCandidate, "/") && !strings.HasPrefix(parseCandidate, "//") {
		return parseCandidate
	}
	return "/app/dashboard"
}

type bufferedResponseWriter struct {
	header http.Header
	body   strings.Builder
	status int
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header), status: http.StatusOK}
}

func (parseW *bufferedResponseWriter) Header() http.Header {
	return parseW.header
}

func (parseW *bufferedResponseWriter) WriteHeader(parseStatus int) {
	parseW.status = parseStatus
}

func (parseW *bufferedResponseWriter) Write(parseData []byte) (int, error) {
	return parseW.body.Write(parseData)
}

func (parseW *bufferedResponseWriter) FlushTo(parseTarget http.ResponseWriter) {
	for parseKey, parseValues := range parseW.header {
		for _, parseValue := range parseValues {
			parseTarget.Header().Add(parseKey, parseValue)
		}
	}
	parseTarget.WriteHeader(parseW.status)
	_, _ = parseTarget.Write([]byte(parseW.body.String()))
}

// localeDirection is a thin alias for atlas.LocaleDirection.
//
// It used to be a second, byte-identical implementation of the same `ar -> rtl`
// rule, one here and one in shared/atlas. That is the shape of a bug that has not
// happened yet: adding a fourth locale means editing two functions, the server's
// <html dir> and the payload's direction field come from different ones, and the
// disagreement shows up as text flowing the wrong way on exactly one surface. The
// client derives direction from atlas.LocaleDirection too, so there is now one rule
// for all three consumers.
func localeDirection(parseLocale string) string {
	return atlas.LocaleDirection(parseLocale)
}
