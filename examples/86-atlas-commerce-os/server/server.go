package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
	serverdb "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/db"
	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/atlas"
	"github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/shared/repository"
	"github.com/monstercameron/GoWebComponents/ui"
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

func newAtlasServer(cfg config, store *serverdb.Store, sessions *serverauth.MockSessionManager) *atlasServer {
	return &atlasServer{cfg: cfg, store: store, sessions: sessions}
}

func (s *atlasServer) routes() http.Handler {
	mux := http.NewServeMux()
	staticFS := http.FileServer(http.Dir(s.cfg.StaticDir))
	mux.Handle("/assets/", http.StripPrefix("/assets/", staticFS))
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /auth/mock-sign-in", s.handleMockSignInPage)
	mux.HandleFunc("POST /auth/mock-sign-in", s.handleMockSignIn)
	mux.HandleFunc("POST /auth/mock-sign-out", s.handleMockSignOut)
	mux.HandleFunc("GET /api/public/catalog", s.handlePublicCatalog)
	mux.HandleFunc("GET /api/public/products/{slug}", s.handlePublicProduct)
	mux.HandleFunc("GET /api/public/products/{slug}/comments", s.handlePublicProductComments)
	mux.HandleFunc("GET /api/public/products/{slug}/related-products", s.handlePublicRelatedProducts)
	mux.HandleFunc("GET /api/public/warehouses", s.handlePublicWarehouses)
	mux.HandleFunc("GET /api/public/warehouses/{slug}", s.handlePublicWarehouse)
	mux.HandleFunc("GET /api/public/warehouses/{slug}/availability/{productSlug}", s.handlePublicAvailability)
	mux.HandleFunc("GET /api/app/bootstrap", s.handleInternalBootstrap)
	mux.HandleFunc("GET /api/app/dashboard", s.handleInternalDashboard)
	mux.HandleFunc("GET /api/app/preferences", s.handleInternalPreferences)
	mux.HandleFunc("GET /api/app/settings", s.handleInternalSettings)
	mux.HandleFunc("GET /api/app/saved-views", s.handleInternalSavedViews)
	mux.HandleFunc("GET /api/app/saved-views/export", s.handleInternalSavedViewsExport)
	mux.HandleFunc("GET /api/app/comments", s.handleInternalComments)
	mux.HandleFunc("GET /api/app/products", s.handleInternalProducts)
	mux.HandleFunc("GET /api/app/products/{slug}", s.handleInternalProductDetail)
	mux.HandleFunc("GET /api/app/inventory", s.handleInternalInventory)
	mux.HandleFunc("GET /api/app/inventory/{sku}", s.handleInternalInventoryDetail)
	mux.HandleFunc("GET /api/app/inventory/{sku}/threshold-panel", s.handleInternalThresholdPanel)
	mux.HandleFunc("GET /api/app/inventory/{sku}/threshold-history", s.handleInternalThresholdHistory)
	mux.HandleFunc("GET /api/app/inventory/{sku}/transfer-recommendations", s.handleInternalTransferRecommendations)
	mux.HandleFunc("GET /api/app/warehouses", s.handleInternalWarehouses)
	mux.HandleFunc("GET /api/app/warehouses/{warehouseId}", s.handleInternalWarehouseDetail)
	mux.HandleFunc("GET /api/app/warehouses/{warehouseId}/items/{sku}", s.handleInternalWarehouseItemDetail)
	mux.HandleFunc("GET /api/app/transfers", s.handleInternalTransfers)
	mux.HandleFunc("GET /api/app/transfers/{id}", s.handleInternalTransferDetail)
	mux.HandleFunc("GET /api/app/receiving", s.handleInternalReceiving)
	mux.HandleFunc("GET /api/app/receiving/{id}", s.handleInternalReceivingDetail)
	mux.HandleFunc("GET /api/app/purchase-orders", s.handleInternalPurchaseOrders)
	mux.HandleFunc("GET /api/app/purchase-orders/{id}", s.handleInternalPurchaseOrderDetail)
	mux.HandleFunc("POST /api/public/products/{slug}/comments", s.handlePublicCommentCreate)
	mux.HandleFunc("POST /api/public/products/{slug}/quote-requests", s.handlePublicQuoteRequestCreate)
	mux.HandleFunc("POST /api/public/products/{slug}/restock-requests", s.handlePublicRestockRequestCreate)
	mux.HandleFunc("POST /api/app/comments/{id}/moderate", s.handleInternalCommentModeration)
	mux.HandleFunc("POST /api/app/comments/bulk-moderate", s.handleInternalBulkCommentModeration)
	mux.HandleFunc("POST /api/app/products", s.handleInternalProductCreate)
	mux.HandleFunc("POST /api/app/products/{slug}/update", s.handleInternalProductUpdate)
	mux.HandleFunc("POST /api/app/products/{slug}/delete", s.handleInternalProductDelete)
	mux.HandleFunc("POST /api/app/inventory/{sku}/update", s.handleInternalInventoryUpdate)
	mux.HandleFunc("POST /api/app/inventory/{sku}/threshold", s.handleInternalThresholdUpdate)
	mux.HandleFunc("POST /api/app/preferences", s.handleInternalPreferencesSave)
	mux.HandleFunc("PUT /api/app/preferences", s.handleInternalPreferencesSave)
	mux.HandleFunc("POST /api/app/saved-views", s.handleInternalSavedViewCreate)
	mux.HandleFunc("POST /api/app/saved-views/import", s.handleInternalSavedViewImport)
	mux.HandleFunc("POST /api/app/transfers", s.handleInternalTransferCreate)
	mux.HandleFunc("POST /api/app/purchase-orders", s.handleInternalPurchaseOrderCreate)
	mux.HandleFunc("POST /api/app/receiving/{id}/reconcile", s.handleInternalReceivingReconcile)
	mux.HandleFunc("POST /api/app/purchase-orders/{id}/status", s.handleInternalPurchaseOrderStatus)
	mux.HandleFunc("GET /{$}", s.handleLandingPage)
	mux.HandleFunc("GET /shop", s.handleCatalogPage)
	mux.HandleFunc("GET /shop/{slug}", s.handleProductPage)
	mux.HandleFunc("GET /warehouses", s.handleWarehousesPage)
	mux.HandleFunc("GET /warehouses/{slug}", s.handleWarehousePage)
	mux.HandleFunc("GET /warehouses/{slug}/availability/{productSlug}", s.handleAvailabilityPage)
	mux.HandleFunc("GET /app", s.handleAppRoot)
	mux.HandleFunc("GET /app/dashboard", s.handleDashboardPage)
	mux.HandleFunc("GET /app/products", s.handleInternalProductsPage)
	mux.HandleFunc("GET /app/products/{slug}", s.handleInternalProductEditorPage)
	mux.HandleFunc("GET /app/inventory", s.handleInventoryPage)
	mux.HandleFunc("GET /app/inventory/{sku}", s.handleInventoryDetailPage)
	mux.HandleFunc("GET /app/inventory/{sku}/threshold-history", s.handleInventoryThresholdHistoryPage)
	mux.HandleFunc("GET /app/warehouses", s.handleWarehouseOpsPage)
	mux.HandleFunc("GET /app/warehouses/{warehouseId}", s.handleWarehouseOpsDetailPage)
	mux.HandleFunc("GET /app/warehouses/{warehouseId}/items/{sku}", s.handleWarehouseOpsItemPage)
	mux.HandleFunc("GET /app/transfers", s.handleTransfersPage)
	mux.HandleFunc("GET /app/transfers/{id}", s.handleTransferDetailPage)
	mux.HandleFunc("GET /app/purchase-orders", s.handlePurchaseOrdersPage)
	mux.HandleFunc("GET /app/purchase-orders/{id}", s.handlePurchaseOrderDetailPage)
	mux.HandleFunc("GET /app/receiving", s.handleReceivingPage)
	mux.HandleFunc("GET /app/receiving/{id}", s.handleReceivingDetailPage)
	mux.HandleFunc("GET /app/comments", s.handleCommentsPage)
	mux.HandleFunc("GET /app/settings", s.handleSettingsPage)
	mux.HandleFunc("GET /__atlas/bootstrap.json", s.handleBootstrapJSON)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/api/") {
			mux.ServeHTTP(w, r)
			return
		}
		probe := newBufferedResponseWriter()
		mux.ServeHTTP(probe, r)
		if probe.status == http.StatusNotFound {
			s.handleRouteRecoveryPage(w, r)
			return
		}
		probe.FlushTo(w)
	})
}

func (s *atlasServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]any{
		"ok":           true,
		"service":      "atlas-commerce-os",
		"databasePath": s.cfg.SQLitePath,
		"wasmPresent":  fileExists(s.cfg.AtlasWASM),
	}
	s.writeJSON(w, http.StatusOK, status)
}

func (s *atlasServer) handleMockSignInPage(w http.ResponseWriter, r *http.Request) {
	if session := s.sessions.Resolve(r); session != nil {
		http.Redirect(w, r, sanitizeNextPath(r.URL.Query().Get("next")), http.StatusSeeOther)
		return
	}
	roles := make([]map[string]string, 0, len(serverauth.AllowedRoles()))
	for _, role := range serverauth.AllowedRoles() {
		roles = append(roles, map[string]string{
			"value":       role,
			"label":       roleLabel(role),
			"description": roleDescription(role),
		})
	}
	s.renderPageStatus(w, r, http.StatusOK, routeMeta{Path: serverauth.MockSignInPath, Surface: "public", Screen: "mock-sign-in", Title: "Atlas Mock Sign In", Description: "Start a mock internal session for the Atlas operator console.", Canonical: serverauth.MockSignInPath}, map[string]any{
		"next":    sanitizeNextPath(r.URL.Query().Get("next")),
		"roles":   roles,
		"message": "Start a mock Atlas internal session to access the operator console.",
	}, nil)
}

func (s *atlasServer) handleMockSignIn(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.writeValidationError(w, http.StatusBadRequest, "invalid_mock_sign_in", "Mock sign-in could not be parsed.", map[string]string{"role": "Choose one of the supported mock roles."})
		return
	}
	role := strings.TrimSpace(r.Form.Get("role"))
	next := sanitizeNextPath(r.Form.Get("next"))
	cookie := s.sessions.StartCookie(role)
	if cookie.Value == "" {
		if wantsHTMLResponse(r) {
			http.Redirect(w, r, withNotice(serverauth.MockSignInPath+"?next="+url.QueryEscape(next), "invalid-mock-role"), http.StatusSeeOther)
			return
		}
		s.writeValidationError(w, http.StatusBadRequest, "invalid_mock_sign_in", "Choose one of the supported mock roles.", map[string]string{"role": "Unsupported mock role."})
		return
	}
	http.SetCookie(w, cookie)
	if wantsHTMLResponse(r) {
		http.Redirect(w, r, withNotice(next, "mock-session-started"), http.StatusSeeOther)
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "next": next, "role": cookie.Value})
}

func (s *atlasServer) handleMockSignOut(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, s.sessions.ClearCookie())
	next := sanitizeNextPath(r.FormValue("next"))
	if wantsHTMLResponse(r) {
		http.Redirect(w, r, withNotice(next, "mock-session-cleared"), http.StatusSeeOther)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "next": next})
}

func (s *atlasServer) handlePublicCatalog(w http.ResponseWriter, r *http.Request) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	result, err := s.store.Catalog(r.Context(), serverdb.CatalogQuery{
		Search:    strings.TrimSpace(r.URL.Query().Get("q")),
		Category:  strings.TrimSpace(r.URL.Query().Get("category")),
		Warehouse: strings.TrimSpace(r.URL.Query().Get("warehouse")),
		Sort:      strings.TrimSpace(r.URL.Query().Get("sort")),
		Page:      page,
		PageSize:  12,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "catalog_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *atlasServer) handlePublicProduct(w http.ResponseWriter, r *http.Request) {
	_, _, pageData, err := s.publicProductPage(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "product_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, pageData)
}

func (s *atlasServer) handlePublicProductComments(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "approved"
	}
	items, err := s.store.ProductComments(r.Context(), r.PathValue("slug"), status)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "product_comments_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *atlasServer) handlePublicRelatedProducts(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.RelatedProducts(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "related_products_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *atlasServer) handlePublicWarehouses(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.Warehouses(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "warehouse_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *atlasServer) handlePublicWarehouse(w http.ResponseWriter, r *http.Request) {
	warehouse, err := s.store.WarehouseBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "warehouse_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, warehouse)
}

func (s *atlasServer) handlePublicAvailability(w http.ResponseWriter, r *http.Request) {
	availability, err := s.store.Availability(r.Context(), r.PathValue("slug"), r.PathValue("productSlug"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "availability_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, availability)
}

type publicWarehouseProductQuery struct {
	Search   string
	Category string
	Status   string
	Sort     string
}

func parsePublicWarehouseProductQuery(values url.Values) publicWarehouseProductQuery {
	query := publicWarehouseProductQuery{
		Search:   strings.TrimSpace(values.Get("q")),
		Category: strings.TrimSpace(values.Get("category")),
		Status:   strings.TrimSpace(values.Get("status")),
		Sort:     strings.TrimSpace(values.Get("sort")),
	}
	if query.Sort == "" {
		query.Sort = "volume"
	}
	return query
}

func (s *atlasServer) handleInternalBootstrap(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	targetPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if targetPath == "" {
		targetPath = "/app/dashboard"
	}
	payload, meta, err := s.bootstrapForPath(r, targetPath, session)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "bootstrap_failed", err)
		return
	}
	payload.CSRF = ensureCSRFCookie(w, r)
	s.writeJSON(w, http.StatusOK, map[string]any{"meta": meta, "bootstrap": payload})
}

func (s *atlasServer) handleInternalDashboard(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	data, err := s.internalDashboardPageData(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "dashboard_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}

func (s *atlasServer) handleInternalPreferences(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	item, err := s.store.PreferencesByOwner(r.Context(), session.UserID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "preferences_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, item)
}

func (s *atlasServer) handleInternalSettings(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	data, err := s.internalSettingsPageData(r.Context(), session.UserID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "settings_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}

func (s *atlasServer) handleInternalSavedViews(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	items, err := s.store.SavedViewsByOwner(r.Context(), session.UserID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "saved_views_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *atlasServer) handleInternalComments(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	data, err := s.internalCommentsPageData(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "comment_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}

func (s *atlasServer) handleInternalInventory(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	data, err := s.internalInventoryPageData(r.Context(), r.URL.Query())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "inventory_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}

func (s *atlasServer) handleInternalInventoryDetail(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	data, err := s.internalInventoryDetailPageData(r.Context(), r.PathValue("sku"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "inventory_item_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}

func (s *atlasServer) handleInternalThresholdPanel(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	data, err := s.internalInventoryThresholdPanelPageData(r.Context(), r.PathValue("sku"))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "threshold_panel_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}

func (s *atlasServer) handleInternalThresholdHistory(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	items, err := s.store.ThresholdHistory(r.Context(), r.PathValue("sku"))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "threshold_history_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *atlasServer) handleInternalTransferRecommendations(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	items, err := s.store.TransferRecommendations(r.Context(), r.PathValue("sku"))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "transfer_recommendations_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *atlasServer) handleInternalWarehouses(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	data, err := s.internalWarehouseOpsPageData(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "warehouse_pressure_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}

func (s *atlasServer) handleInternalWarehouseDetail(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	item, err := s.internalWarehouseDetailPageDataWithFilters(r.Context(), r.PathValue("warehouseId"), r.URL.Query())
	if err != nil {
		s.writeError(w, http.StatusNotFound, "warehouse_detail_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, item)
}

func (s *atlasServer) handleInternalWarehouseItemDetail(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	item, err := s.internalWarehouseItemPageData(r.Context(), r.PathValue("warehouseId"), r.PathValue("sku"), r.URL.Query())
	if err != nil {
		s.writeError(w, http.StatusNotFound, "warehouse_item_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, item)
}

func (s *atlasServer) handleInternalTransfers(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	items, err := s.store.Transfers(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "transfer_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *atlasServer) handleInternalTransferDetail(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	item, err := s.store.TransferDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "transfer_detail_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, item)
}

func (s *atlasServer) handleInternalReceiving(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	items, err := s.store.ReceivingSessions(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "receiving_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *atlasServer) handleInternalReceivingDetail(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	item, err := s.store.ReceivingDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "receiving_detail_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, item)
}

func (s *atlasServer) handleInternalPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	data, err := s.internalPurchaseOrdersPageData(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "purchase_orders_query_failed", err)
		return
	}
	s.writeJSON(w, http.StatusOK, data)
}

func (s *atlasServer) handleInternalPurchaseOrderDetail(w http.ResponseWriter, r *http.Request) {
	if s.sessions.RequireInternalSession(w, r) == nil {
		return
	}
	item, err := s.store.PurchaseOrderDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeError(w, http.StatusNotFound, "purchase_order_detail_not_found", err)
		return
	}
	s.writeJSON(w, http.StatusOK, item)
}

func (s *atlasServer) handleLandingPage(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, routeMeta{Path: "/", Surface: "public", Screen: "landing", Title: "Atlas Commerce OS", Description: "Premium modular workspace systems with warehouse-aware availability.", Canonical: "/"}, map[string]any{"message": "Atlas storefront landing"}, nil)
}

func (s *atlasServer) handleCatalogPage(w http.ResponseWriter, r *http.Request) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	result, err := s.store.Catalog(r.Context(), serverdb.CatalogQuery{Search: r.URL.Query().Get("q"), Category: r.URL.Query().Get("category"), Warehouse: r.URL.Query().Get("warehouse"), Sort: r.URL.Query().Get("sort"), Page: page, PageSize: 12})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "catalog_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/shop", Surface: "public", Screen: "catalog", Title: "Atlas Shop", Description: "Browse Atlas modular workspace systems with filterable discovery and regional fulfillment context.", Canonical: "/shop"}, result, nil)
}

func (s *atlasServer) handleProductPage(w http.ResponseWriter, r *http.Request) {
	product, _, pageData, err := s.publicProductPage(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: "/shop/" + r.PathValue("slug"), Surface: "public", Screen: "recovery", Title: "Atlas Product Not Found", Description: "The requested Atlas product could not be loaded.", Canonical: "/shop/" + r.PathValue("slug")}, nil, "Product not found", "The requested Atlas product is unavailable or no longer part of the demo seed.", "/shop", "Back to shop", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/shop/" + r.PathValue("slug"), Surface: "public", Screen: "product", Title: "Atlas " + product.Title, Description: product.SEODescription, Canonical: "/shop/" + product.Slug}, pageData, nil)
}

func (s *atlasServer) publicProductPage(ctx context.Context, slug string) (repository.Product, []serverdb.CommentRecord, map[string]any, error) {
	product, err := s.store.ProductBySlug(ctx, slug)
	if err != nil {
		return repository.Product{}, nil, nil, err
	}
	comments, err := s.store.ProductComments(ctx, slug, "approved")
	if err != nil {
		return repository.Product{}, nil, nil, err
	}
	return product, comments, map[string]any{"product": product, "comments": comments}, nil
}

func (s *atlasServer) handleWarehousesPage(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.Warehouses(r.Context())
	if err != nil {
		s.renderRecoveryPage(w, r, http.StatusInternalServerError, routeMeta{Path: "/warehouses", Surface: "public", Screen: "recovery", Title: "Atlas Delivery Regions Unavailable", Description: "The Atlas delivery-region directory could not be loaded.", Canonical: "/warehouses"}, nil, "Delivery regions unavailable", "The Atlas delivery-region directory is temporarily unavailable. Retry the page or return to the storefront.", "/", "Back to storefront", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/warehouses", Surface: "public", Screen: "warehouses", Title: "Atlas Delivery Regions", Description: "Compare Atlas delivery regions, service levels, and stocked highlights before opening a warehouse route.", Canonical: "/warehouses"}, map[string]any{"items": items}, nil)
}

func (s *atlasServer) handleWarehousePage(w http.ResponseWriter, r *http.Request) {
	warehouse, err := s.store.WarehouseBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		path := "/warehouses/" + r.PathValue("slug")
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "public", Screen: "recovery", Title: "Atlas Warehouse Not Found", Description: "The requested warehouse could not be loaded.", Canonical: path}, nil, "Warehouse not found", "That warehouse route is not available in the current Atlas seed set.", "/warehouses", "Back to warehouses", err)
		return
	}
	products, err := s.store.Catalog(r.Context(), serverdb.CatalogQuery{Warehouse: warehouse.ID, Page: 1, PageSize: 3})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "warehouse_catalog_query_failed", err)
		return
	}
	path := "/warehouses/" + warehouse.Slug
	s.renderPage(w, r, routeMeta{Path: path, Surface: "public", Screen: "warehouse-detail", Title: "Atlas Warehouse Detail", Description: warehouse.PublicSummary, Canonical: path}, map[string]any{"warehouse": warehouse, "products": products.Items}, nil)
}

func (s *atlasServer) handleAvailabilityPage(w http.ResponseWriter, r *http.Request) {
	availability, err := s.store.Availability(r.Context(), r.PathValue("slug"), r.PathValue("productSlug"))
	if err != nil {
		path := fmt.Sprintf("/warehouses/%s/availability/%s", r.PathValue("slug"), r.PathValue("productSlug"))
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "public", Screen: "recovery", Title: "Atlas Availability Not Found", Description: "The requested warehouse availability view could not be loaded.", Canonical: path}, nil, "Availability view not found", "That warehouse-specific availability route is not available in the current Atlas demo data.", "/warehouses", "Back to warehouses", err)
		return
	}
	path := fmt.Sprintf("/warehouses/%s/availability/%s", r.PathValue("slug"), r.PathValue("productSlug"))
	s.renderPage(w, r, routeMeta{Path: path, Surface: "public", Screen: "warehouse-availability", Title: "Atlas Warehouse Availability", Description: "Inspect a warehouse-specific product promise for one Atlas item and one regional fulfillment hub.", Canonical: path}, availability, nil)
}

func (s *atlasServer) handleAppRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app/dashboard", http.StatusFound)
}

func (s *atlasServer) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	data, err := s.internalDashboardPageData(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "dashboard_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/app/dashboard", Surface: "internal", Screen: "dashboard", Title: "Atlas Ops Dashboard", Description: "Operational overview of buyer questions, stock pressure, receiving exceptions, and warehouse health.", Canonical: "/app/dashboard"}, data, session)
}

func (s *atlasServer) handleInventoryPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	data, err := s.internalInventoryPageData(r.Context(), r.URL.Query())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "inventory_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/app/inventory", Surface: "internal", Screen: "inventory", Title: "Atlas Inventory", Description: "Review inventory health, saved views, and warehouse-aware stock pressure.", Canonical: "/app/inventory"}, data, session)
}

func (s *atlasServer) handleInventoryDetailPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	data, err := s.internalInventoryDetailPageData(r.Context(), r.PathValue("sku"))
	if err != nil {
		path := "/app/inventory/" + r.PathValue("sku")
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "internal", Screen: "recovery", Title: "Atlas SKU Not Found", Description: "The requested inventory detail route could not be loaded.", Canonical: path}, session, "SKU not found", "That inventory item is not available in the current Atlas seed set.", "/app/inventory", "Back to inventory", err)
		return
	}
	path := "/app/inventory/" + r.PathValue("sku")
	s.renderPage(w, r, routeMeta{Path: path, Surface: "internal", Screen: "sku-detail", Title: "Atlas SKU Detail", Description: "Inspect warehouse breakdown, thresholds, and activity for a single Atlas SKU.", Canonical: path}, data, session)
}

func (s *atlasServer) handleInventoryThresholdHistoryPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	sku := r.PathValue("sku")
	pageData, err := s.internalInventoryDetailPageData(r.Context(), sku)
	if err != nil {
		path := "/app/inventory/" + sku + "/threshold-history"
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "internal", Screen: "recovery", Title: "Atlas Threshold History Unavailable", Description: "The threshold-history route could not be loaded for this SKU.", Canonical: path}, session, "Threshold history unavailable", "That inventory item is not available in the current Atlas seed set.", "/app/inventory", "Back to inventory", err)
		return
	}
	overlayData, err := s.internalInventoryThresholdPanelPageData(r.Context(), sku)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "threshold_panel_query_failed", err)
		return
	}
	path := "/app/inventory/" + sku + "/threshold-history"
	requests := startupRequestsForPage("/app/inventory/"+sku, r.URL.Query(), pageData)
	requests["overlay"] = atlas.Request{
		Method: http.MethodGet,
		URL:    "/api/app/inventory/" + sku + "/threshold-panel",
		Status: http.StatusOK,
		Data:   map[string]any{"overlay": overlayData},
	}
	s.renderPageStatusWithPayload(w, r, http.StatusOK, routeMeta{
		Path:        path,
		Surface:     "internal",
		Screen:      "sku-threshold-history",
		Title:       "Atlas Threshold History",
		Description: "Review threshold edits and transfer cues for one Atlas SKU without leaving the inventory route context.",
		Canonical:   path,
	}, map[string]any{
		"page":    pageData,
		"overlay": overlayData,
	}, requests, session)
}

func (s *atlasServer) handleWarehouseOpsPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	data, err := s.internalWarehouseOpsPageData(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "warehouse_pressure_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/app/warehouses", Surface: "internal", Screen: "warehouse-ops", Title: "Atlas Warehouse Operations", Description: "Compare staffing, backlog, service posture, and warehouse pressure across the Atlas network.", Canonical: "/app/warehouses"}, data, session)
}

func (s *atlasServer) handleWarehouseOpsDetailPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	item, err := s.internalWarehouseDetailPageDataWithFilters(r.Context(), r.PathValue("warehouseId"), r.URL.Query())
	if err != nil {
		path := "/app/warehouses/" + r.PathValue("warehouseId")
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "internal", Screen: "recovery", Title: "Atlas Warehouse Not Found", Description: "The requested internal warehouse route could not be loaded.", Canonical: path}, session, "Warehouse not found", "That internal warehouse route is not available in the current Atlas seed set.", "/app/warehouses", "Back to internal warehouses", err)
		return
	}
	path := "/app/warehouses/" + r.PathValue("warehouseId")
	s.renderPage(w, r, routeMeta{Path: path, Surface: "internal", Screen: "warehouse-detail", Title: "Atlas Warehouse Detail", Description: "Inspect staffing, backlog, and next action for a single Atlas warehouse.", Canonical: path}, item, session)
}

func (s *atlasServer) handleWarehouseOpsItemPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	warehouseID := r.PathValue("warehouseId")
	path := "/app/warehouses/" + warehouseID + "/items/" + r.PathValue("sku")
	pageData, err := s.internalWarehouseDetailPageDataWithFilters(r.Context(), warehouseID, r.URL.Query())
	if err != nil {
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "internal", Screen: "recovery", Title: "Atlas Warehouse Item Not Found", Description: "The requested warehouse item route could not be loaded.", Canonical: path}, session, "Warehouse item not found", "That warehouse item is not available in the current Atlas seed set for this facility.", "/app/warehouses/"+warehouseID, "Back to warehouse items", err)
		return
	}
	item, err := s.internalWarehouseItemPageData(r.Context(), warehouseID, r.PathValue("sku"), r.URL.Query())
	if err != nil {
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "internal", Screen: "recovery", Title: "Atlas Warehouse Item Not Found", Description: "The requested warehouse item route could not be loaded.", Canonical: path}, session, "Warehouse item not found", "That warehouse item is not available in the current Atlas seed set for this facility.", "/app/warehouses/"+r.PathValue("warehouseId"), "Back to warehouse items", err)
		return
	}
	requests := startupRequestsForPage("/app/warehouses/"+warehouseID, r.URL.Query(), pageData)
	requests["item"] = atlas.Request{
		Method: http.MethodGet,
		URL:    startupRequestURL(path, atlasDataQuery(r.URL.Query())),
		Status: http.StatusOK,
		Data:   map[string]any{"item": item},
	}
	s.renderPageStatusWithPayload(w, r, http.StatusOK, routeMeta{Path: path, Surface: "internal", Screen: "warehouse-item-detail", Title: "Atlas Warehouse Item", Description: "Manage one warehouse item with inventory edits, replenishment, and demand context.", Canonical: path}, map[string]any{
		"page": pageData,
		"item": item,
	}, requests, session)
}

func (s *atlasServer) handleTransfersPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	items, err := s.store.Transfers(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "transfer_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/app/transfers", Surface: "internal", Screen: "transfers", Title: "Atlas Transfers", Description: "Plan, review, and approve cross-warehouse transfer recommendations.", Canonical: "/app/transfers"}, map[string]any{"items": items}, session)
}

func (s *atlasServer) handleTransferDetailPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	item, err := s.store.TransferDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		path := "/app/transfers/" + r.PathValue("id")
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "internal", Screen: "recovery", Title: "Atlas Transfer Not Found", Description: "The requested transfer detail route could not be loaded.", Canonical: path}, session, "Transfer not found", "That transfer detail route is not available in the current Atlas seed set.", "/app/transfers", "Back to transfers", err)
		return
	}
	path := "/app/transfers/" + r.PathValue("id")
	s.renderPage(w, r, routeMeta{Path: path, Surface: "internal", Screen: "transfer-detail", Title: "Atlas Transfer Detail", Description: "Inspect one Atlas transfer, including approval state, lane context, and audit activity.", Canonical: path}, item, session)
}

func (s *atlasServer) handlePurchaseOrdersPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	data, err := s.internalPurchaseOrdersPageData(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "purchase_orders_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/app/purchase-orders", Surface: "internal", Screen: "purchase-orders", Title: "Atlas Purchase Orders", Description: "Review vendor approvals, inbound shipment rows, and purchase-order planning context.", Canonical: "/app/purchase-orders"}, data, session)
}

func (s *atlasServer) handlePurchaseOrderDetailPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	item, err := s.store.PurchaseOrderDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		path := "/app/purchase-orders/" + r.PathValue("id")
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "internal", Screen: "recovery", Title: "Atlas Purchase Order Not Found", Description: "The requested purchase-order detail route could not be loaded.", Canonical: path}, session, "Purchase order not found", "That purchase-order route is not available in the current Atlas seed set.", "/app/purchase-orders", "Back to purchase orders", err)
		return
	}
	path := "/app/purchase-orders/" + r.PathValue("id")
	s.renderPage(w, r, routeMeta{Path: path, Surface: "internal", Screen: "purchase-order-detail", Title: "Atlas Purchase Order Detail", Description: "Inspect one Atlas purchase order, including vendor state, ETA, and inbound shipment rows.", Canonical: path}, item, session)
}

func (s *atlasServer) handleReceivingPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	items, err := s.store.ReceivingSessions(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "receiving_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/app/receiving", Surface: "internal", Screen: "receiving", Title: "Atlas Receiving", Description: "Track inbound sessions, discrepancies, and receiving closeout state.", Canonical: "/app/receiving"}, map[string]any{"items": items}, session)
}

func (s *atlasServer) handleReceivingDetailPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	item, err := s.store.ReceivingDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		path := "/app/receiving/" + r.PathValue("id")
		s.renderRecoveryPage(w, r, http.StatusNotFound, routeMeta{Path: path, Surface: "internal", Screen: "recovery", Title: "Atlas Receiving Session Not Found", Description: "The requested receiving-session route could not be loaded.", Canonical: path}, session, "Receiving session not found", "That receiving-session route is not available in the current Atlas seed set.", "/app/receiving", "Back to receiving", err)
		return
	}
	path := "/app/receiving/" + r.PathValue("id")
	s.renderPage(w, r, routeMeta{Path: path, Surface: "internal", Screen: "receiving-session-detail", Title: "Atlas Receiving Session", Description: "Inspect one receiving session, including discrepancy classification and closeout readiness.", Canonical: path}, item, session)
}

func (s *atlasServer) handleCommentsPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	data, err := s.internalCommentsPageData(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "comment_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/app/comments", Surface: "internal", Screen: "comments", Title: "Atlas Buyer Inbox", Description: "Review buyer questions, moderation decisions, and follow-up paths into product, inventory, or warehouse work.", Canonical: "/app/comments"}, data, session)
}

func (s *atlasServer) handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	session := s.sessions.RequireInternalSession(w, r)
	if session == nil {
		return
	}
	data, err := s.internalSettingsPageData(r.Context(), session.UserID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "settings_query_failed", err)
		return
	}
	s.renderPage(w, r, routeMeta{Path: "/app/settings", Surface: "internal", Screen: "settings", Title: "Atlas Settings", Description: "Manage theme, locale, density, default warehouse, and saved-view preferences.", Canonical: "/app/settings"}, data, session)
}

func (s *atlasServer) handleRouteRecoveryPage(w http.ResponseWriter, r *http.Request) {
	var session *serverauth.Session
	if strings.HasPrefix(r.URL.Path, "/app/") {
		session = s.sessions.RequireInternalSession(w, r)
		if session == nil {
			return
		}
	}
	meta := routeMeta{Path: r.URL.Path, Surface: "public", Screen: "recovery", Title: "Atlas Route Not Found", Description: "The requested Atlas route could not be resolved.", Canonical: r.URL.Path}
	recoveryPath := "/shop"
	recoveryLabel := "Back to shop"
	title := "Route not found"
	message := "The requested Atlas route is not part of the current demo route set."
	if strings.HasPrefix(r.URL.Path, "/app/") {
		meta.Surface = "internal"
		meta.Title = "Atlas Internal Route Not Found"
		recoveryPath = "/app/dashboard"
		recoveryLabel = "Back to dashboard"
		message = "The requested Atlas internal route is not part of the current demo route set."
	}
	s.renderRecoveryPage(w, r, http.StatusNotFound, meta, session, title, message, recoveryPath, recoveryLabel, nil)
}

func (s *atlasServer) renderPage(w http.ResponseWriter, r *http.Request, meta routeMeta, data any, session *serverauth.Session) {
	s.renderPageStatus(w, r, http.StatusOK, meta, data, session)
}

func (s *atlasServer) renderPageStatus(w http.ResponseWriter, r *http.Request, status int, meta routeMeta, data any, session *serverauth.Session) {
	s.renderPageStatusWithPayload(w, r, status, meta, map[string]any{"page": data}, startupRequestsForPage(meta.Path, r.URL.Query(), data), session)
}

func (s *atlasServer) renderPageStatusWithPayload(w http.ResponseWriter, r *http.Request, status int, meta routeMeta, payloadData map[string]any, requests map[string]atlas.Request, session *serverauth.Session) {
	payload, _, err := s.bootstrapForPath(r, meta.Path, session)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "bootstrap_failed", err)
		return
	}
	payload.Route.Surface = meta.Surface
	payload.Route.Screen = meta.Screen
	payload.Route.Title = meta.Title
	payload.Route.Description = meta.Description
	payload.Route.Canonical = meta.Canonical
	payload.CSRF = ensureCSRFCookie(w, r)
	payload.Data = payloadData
	payload.Requests = requests
	bootstrapScript, bootstrapBytes, bootstrapMode, err := s.renderBootstrapScript(meta.Path, r.URL.Query(), payload)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "bootstrap_encode_failed", err)
		return
	}
	w.Header().Set("X-Atlas-Bootstrap-Bytes", fmt.Sprintf("%d", bootstrapBytes))
	w.Header().Set("X-Atlas-Bootstrap-Mode", bootstrapMode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html lang=%q class=%q data-atlas-surface=%q><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title data-gwc-router-managed=\"true\">%s</title><meta name=\"description\" content=%q data-gwc-router-managed=\"true\"><link rel=\"canonical\" href=%q data-gwc-router-managed=\"true\"><link rel=\"stylesheet\" href=\"/assets/css/tailwind.css\"><link rel=\"stylesheet\" href=\"/assets/css/example-shell.css\"><script src=\"/assets/script/wasm_exec.js\"></script><script src=\"/assets/script/example-logger.js\"></script></head><body class=\"example-shell\"><div id=\"app\"></div>%s%s</body></html>", payload.I18n.Locale, atlasDocumentClass(payload), payload.Route.Surface, meta.Title, meta.Description, meta.Canonical, bootstrapScript, wasmRuntimeSnippet(fileExists(s.cfg.AtlasWASM)))
}

func atlasDocumentClass(payload atlas.Payload) string {
	themeClass := "atlas-theme-dark"
	if strings.EqualFold(strings.TrimSpace(payload.Theme.Mode), "light") {
		themeClass = "atlas-theme-light"
	}
	densityClass := "atlas-density-compact"
	if strings.EqualFold(strings.TrimSpace(payload.Preferences.Density), "comfortable") {
		densityClass = "atlas-density-comfortable"
	}
	return themeClass + " " + densityClass
}

func cloneURLValues(values url.Values) url.Values {
	cloned := url.Values{}
	for key, items := range values {
		cloned[key] = append([]string(nil), items...)
	}
	return cloned
}

func atlasDataQuery(values url.Values) url.Values {
	filtered := url.Values{}
	for key, items := range values {
		trimmedKey := strings.TrimSpace(strings.ToLower(key))
		if trimmedKey == atlasNoticeQueryKey || trimmedKey == atlasBootstrapModeQueryKey {
			continue
		}
		for _, item := range items {
			filtered.Add(key, item)
		}
	}
	return filtered
}

func atlasBootstrapReferenceURL(path string, query url.Values) string {
	values := url.Values{}
	values.Set("path", path)
	if encoded := cloneURLValues(query).Encode(); encoded != "" {
		values.Set("route_query", encoded)
	}
	return "/__atlas/bootstrap.json?" + values.Encode()
}

func atlasBootstrapMode(values url.Values) string {
	if strings.EqualFold(strings.TrimSpace(values.Get(atlasBootstrapModeQueryKey)), atlasBootstrapModeExternal) {
		return atlasBootstrapModeExternal
	}
	return "inline"
}

func supportsExternalBootstrap(path string) bool {
	return strings.HasPrefix(path, atlas.RouteInventory+"/") && strings.HasSuffix(path, "/threshold-history")
}

func (s *atlasServer) renderBootstrapScript(path string, query url.Values, payload atlas.Payload) (string, int, string, error) {
	bootstrap := payload.ToSSRBootstrap()
	encoded, err := ui.MarshalSSRBootstrap(bootstrap)
	if err != nil {
		return "", 0, "", err
	}
	mode := atlasBootstrapMode(query)
	if mode == atlasBootstrapModeExternal && supportsExternalBootstrap(path) {
		script, err := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{
			URL:    atlasBootstrapReferenceURL(path, query),
			Format: ui.SSRBootstrapFormatJSON,
		}, atlasBootstrapReferenceScriptID)
		return script, len(encoded), mode, err
	}
	mode = "inline"
	script, err := ui.RenderBootstrapScript(bootstrap, atlasBootstrapScriptID)
	return script, len(encoded), mode, err
}

func startupRequestsForPage(path string, query url.Values, pageData any) map[string]atlas.Request {
	requestURL := startupRequestURL(path, atlasDataQuery(query))
	if requestURL == "" {
		return map[string]atlas.Request{}
	}
	return map[string]atlas.Request{
		"page": {
			Method: http.MethodGet,
			URL:    requestURL,
			Status: http.StatusOK,
			Data:   map[string]any{"page": pageData},
		},
	}
}

func startupRequestURL(path string, query url.Values) string {
	return atlas.StartupRequestURL(path, query)
}

func (s *atlasServer) renderRecoveryPage(w http.ResponseWriter, r *http.Request, status int, meta routeMeta, session *serverauth.Session, title, message, recoveryPath, recoveryLabel string, err error) {
	page := map[string]any{
		"title":         title,
		"message":       message,
		"recoveryHref":  recoveryPath,
		"recoveryLabel": recoveryLabel,
	}
	if err != nil {
		page["detail"] = err.Error()
	}
	s.renderPageStatus(w, r, status, meta, page, session)
}

func (s *atlasServer) handleBootstrapJSON(w http.ResponseWriter, r *http.Request) {
	targetPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if targetPath == "" {
		targetPath = "/"
	}
	if !supportsExternalBootstrap(targetPath) {
		s.writeValidationError(w, http.StatusBadRequest, "bootstrap_route_unsupported", "External bootstrap mode is only wired for the inventory threshold-history route today.", map[string]string{"path": "Use /app/inventory/{sku}/threshold-history for the proof-of-concept external bootstrap flow."})
		return
	}
	routeQuery, err := url.ParseQuery(strings.TrimSpace(r.URL.Query().Get("route_query")))
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "bootstrap_query_invalid", err)
		return
	}
	var session *serverauth.Session
	if strings.HasPrefix(targetPath, "/app/") {
		session = s.sessions.RequireInternalSession(w, r)
		if session == nil {
			return
		}
	}
	sku := strings.TrimSuffix(strings.TrimPrefix(targetPath, atlas.RouteInventory+"/"), "/threshold-history")
	pageData, err := s.internalInventoryDetailPageData(r.Context(), sku)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "inventory_item_not_found", err)
		return
	}
	overlayData, err := s.internalInventoryThresholdPanelPageData(r.Context(), sku)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "threshold_panel_query_failed", err)
		return
	}
	payload, _, err := s.bootstrapForRouteQuery(r, targetPath, routeQuery, session)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "bootstrap_failed", err)
		return
	}
	payload.Route.Screen = "sku-threshold-history"
	payload.Route.Title = "Atlas Threshold History"
	payload.Route.Description = "Review threshold edits and transfer cues for one Atlas SKU without leaving the inventory route context."
	payload.Route.Canonical = targetPath
	payload.Data = map[string]any{
		"page":    pageData,
		"overlay": overlayData,
	}
	payload.Requests = startupRequestsForPage("/app/inventory/"+sku, routeQuery, pageData)
	payload.Requests["overlay"] = atlas.Request{
		Method: http.MethodGet,
		URL:    "/api/app/inventory/" + sku + "/threshold-panel",
		Status: http.StatusOK,
		Data:   map[string]any{"overlay": overlayData},
	}
	encoded, err := ui.MarshalSSRBootstrap(payload.ToSSRBootstrap())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "bootstrap_encode_failed", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Atlas-Bootstrap-Bytes", fmt.Sprintf("%d", len(encoded)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
}

func (s *atlasServer) bootstrapForPath(r *http.Request, path string, session *serverauth.Session) (atlas.Payload, routeMeta, error) {
	return s.bootstrapForRouteQuery(r, path, r.URL.Query(), session)
}

func (s *atlasServer) bootstrapForRouteQuery(r *http.Request, path string, routeQuery url.Values, session *serverauth.Session) (atlas.Payload, routeMeta, error) {
	meta := routeMetaForPath(path)
	preferences, _ := s.store.PreferencesByOwner(r.Context(), sessionOwnerID(session))
	savedViews, _ := s.store.SavedViewsByOwner(r.Context(), sessionOwnerID(session))
	query := cloneQuery(routeQuery)
	payload := atlas.Payload{
		Route: atlas.RouteBootstrap{
			Path:        path,
			Query:       query,
			Params:      map[string]string{},
			Surface:     meta.Surface,
			Screen:      meta.Screen,
			Title:       meta.Title,
			Description: meta.Description,
			Canonical:   meta.Canonical,
		},
		Preferences: atlas.PreferencesState{
			Theme:            preferences.Theme,
			Locale:           preferences.Locale,
			Density:          preferences.Density,
			DefaultWarehouse: preferences.DefaultWarehouseID,
		},
		I18n: atlas.I18nState{
			Locale:           nonEmpty(preferences.Locale, "en"),
			SupportedLocales: atlas.SupportedLocales(),
			Direction:        localeDirection(preferences.Locale),
		},
		Theme: atlas.ThemeState{
			Mode:                 nonEmpty(preferences.Theme, "dark"),
			PrefersReducedMotion: false,
		},
		SavedViews: toSavedViewPayloads(savedViews),
		Data: map[string]any{
			"canonical":   meta.Canonical,
			"description": meta.Description,
		},
	}
	if session != nil {
		payload.User = &atlas.UserSession{
			ID:               session.UserID,
			DisplayName:      session.DisplayName,
			Role:             session.Role,
			DefaultWarehouse: session.DefaultWarehouse,
		}
	}
	return payload, meta, nil
}

func routeMetaForPath(path string) routeMeta {
	switch {
	case path == "/":
		return routeMeta{Path: path, Surface: "public", Screen: "landing", Title: "Atlas Commerce OS", Description: "Premium modular workspace systems with warehouse-aware availability.", Canonical: path}
	case path == "/shop":
		return routeMeta{Path: path, Surface: "public", Screen: "catalog", Title: "Atlas Shop", Description: "Browse Atlas modular workspace systems with filterable discovery and regional fulfillment context.", Canonical: path}
	case strings.HasPrefix(path, "/shop/"):
		return routeMeta{Path: path, Surface: "public", Screen: "product", Title: "Atlas Product", Description: "Warehouse-aware availability and premium workspace design for Atlas products.", Canonical: path}
	case path == "/warehouses":
		return routeMeta{Path: path, Surface: "public", Screen: "warehouses", Title: "Atlas Delivery Regions", Description: "Compare Atlas delivery regions, service levels, and stocked highlights before opening a warehouse route.", Canonical: path}
	case path == serverauth.MockSignInPath:
		return routeMeta{Path: path, Surface: "public", Screen: "mock-sign-in", Title: "Atlas Mock Sign In", Description: "Start a mock internal session for the Atlas operator console.", Canonical: path}
	case strings.Contains(path, "/availability/"):
		return routeMeta{Path: path, Surface: "public", Screen: "warehouse-availability", Title: "Atlas Warehouse Availability", Description: "Inspect a warehouse-specific product promise for one Atlas item and one regional fulfillment hub.", Canonical: path}
	case strings.HasPrefix(path, "/warehouses/"):
		return routeMeta{Path: path, Surface: "public", Screen: "warehouse-detail", Title: "Atlas Warehouse Region", Description: "Inspect one Atlas delivery region, including service posture, stocked highlights, and product-specific availability links.", Canonical: path}
	case path == "/app/dashboard":
		return routeMeta{Path: path, Surface: "internal", Screen: "dashboard", Title: "Atlas Ops Dashboard", Description: "Operational overview of buyer questions, stock pressure, receiving exceptions, and warehouse health.", Canonical: path}
	case path == "/app/products":
		return routeMeta{Path: path, Surface: "internal", Screen: "products", Title: "Atlas Product Merchandising", Description: "Manage product copy, pricing, launch posture, and merchandising details for Atlas workspace systems.", Canonical: path}
	case strings.HasPrefix(path, "/app/products/"):
		return routeMeta{Path: path, Surface: "internal", Screen: "product-editor", Title: "Atlas Product Editor", Description: "Edit product copy, pricing, and volume for one Atlas storefront item.", Canonical: path}
	case path == "/app/inventory":
		return routeMeta{Path: path, Surface: "internal", Screen: "inventory", Title: "Atlas Inventory", Description: "Review inventory health, saved views, and warehouse-aware stock pressure.", Canonical: path}
	case strings.HasPrefix(path, "/app/inventory/") && strings.HasSuffix(path, "/threshold-history"):
		return routeMeta{Path: path, Surface: "internal", Screen: "sku-threshold-history", Title: "Atlas Threshold History", Description: "Review threshold edits and transfer cues for one Atlas SKU without leaving the inventory route context.", Canonical: path}
	case path == "/app/warehouses":
		return routeMeta{Path: path, Surface: "internal", Screen: "warehouse-ops", Title: "Atlas Warehouse Operations", Description: "Compare staffing, backlog, service posture, and warehouse pressure across the Atlas network.", Canonical: path}
	case strings.HasPrefix(path, "/app/warehouses/"):
		return routeMeta{Path: path, Surface: "internal", Screen: "warehouse-detail", Title: "Atlas Warehouse Detail", Description: "Inspect staffing, backlog, and next action for a single Atlas warehouse.", Canonical: path}
	case strings.HasPrefix(path, "/app/inventory/"):
		return routeMeta{Path: path, Surface: "internal", Screen: "sku-detail", Title: "Atlas SKU Detail", Description: "Inspect warehouse breakdown, thresholds, and activity for a single Atlas SKU.", Canonical: path}
	case path == "/app/transfers":
		return routeMeta{Path: path, Surface: "internal", Screen: "transfers", Title: "Atlas Transfers", Description: "Plan, review, and approve cross-warehouse transfer recommendations.", Canonical: path}
	case strings.HasPrefix(path, "/app/transfers/"):
		return routeMeta{Path: path, Surface: "internal", Screen: "transfer-detail", Title: "Atlas Transfer Detail", Description: "Inspect one Atlas transfer, including approval state, lane context, and audit activity.", Canonical: path}
	case path == "/app/purchase-orders":
		return routeMeta{Path: path, Surface: "internal", Screen: "purchase-orders", Title: "Atlas Purchase Orders", Description: "Review vendor approvals, inbound shipment rows, and purchase-order planning context.", Canonical: path}
	case strings.HasPrefix(path, "/app/purchase-orders/"):
		return routeMeta{Path: path, Surface: "internal", Screen: "purchase-order-detail", Title: "Atlas Purchase Order Detail", Description: "Inspect one Atlas purchase order, including vendor state, ETA, and inbound shipment rows.", Canonical: path}
	case path == "/app/receiving":
		return routeMeta{Path: path, Surface: "internal", Screen: "receiving", Title: "Atlas Receiving", Description: "Track inbound sessions, discrepancies, and receiving closeout state.", Canonical: path}
	case strings.HasPrefix(path, "/app/receiving/"):
		return routeMeta{Path: path, Surface: "internal", Screen: "receiving-session-detail", Title: "Atlas Receiving Session", Description: "Inspect one receiving session, including discrepancy classification and closeout readiness.", Canonical: path}
	case path == "/app/comments":
		return routeMeta{Path: path, Surface: "internal", Screen: "comments", Title: "Atlas Buyer Inbox", Description: "Review buyer questions, moderation decisions, and follow-up paths into product, inventory, or warehouse work.", Canonical: path}
	case path == "/app/settings":
		return routeMeta{Path: path, Surface: "internal", Screen: "settings", Title: "Atlas Settings", Description: "Manage theme, locale, density, default warehouse, and saved-view preferences.", Canonical: path}
	default:
		return routeMeta{Path: path, Surface: "public", Screen: "recovery", Title: "Atlas Route Not Found", Description: "Atlas recovery route.", Canonical: path}
	}
}

func sessionOwnerID(session *serverauth.Session) string {
	if session == nil {
		return "public"
	}
	return session.UserID
}

func cloneQuery(values url.Values) map[string][]string {
	clone := map[string][]string{}
	for key, item := range values {
		clone[key] = append([]string(nil), item...)
	}
	return clone
}

func toSavedViewPayloads(views []repository.SavedView) []atlas.SavedViewPayload {
	payloads := make([]atlas.SavedViewPayload, 0, len(views))
	for _, view := range views {
		payloads = append(payloads, atlas.SavedViewPayload{
			Name:          view.Name,
			Scope:         view.Scope,
			SortKey:       view.SortKey,
			SortDirection: view.SortDirection,
			Filters:       map[string]string{"density": view.Density, "warehouse": view.WarehouseID},
		})
	}
	return payloads
}

func (s *atlasServer) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *atlasServer) writeError(w http.ResponseWriter, status int, code string, err error) {
	s.writeJSON(w, status, map[string]any{
		"error":   code,
		"message": strings.ReplaceAll(code, "_", " "),
		"detail":  err.Error(),
	})
}

func (s *atlasServer) writeValidationError(w http.ResponseWriter, status int, code string, summary string, fields map[string]string) {
	s.writeJSON(w, status, map[string]any{
		"error":   code,
		"message": summary,
		"fields":  fields,
	})
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func wasmRuntimeSnippet(wasmPresent bool) string {
	if !wasmPresent {
		return `<script>console.warn("atlas-commerce-os.wasm not present yet; SSR shell is running without hydration.");</script>`
	}
	return `<script>const go=new Go();WebAssembly.instantiateStreaming(fetch('/assets/bin/atlas-commerce-os.wasm'),go.importObject).then(result=>go.run(result.instance)).catch(err=>console.error('Failed to hydrate Atlas WASM:',err));</script>`
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func roleLabel(role string) string {
	switch role {
	case "inventory_manager":
		return "Inventory Manager"
	case "warehouse_supervisor":
		return "Warehouse Supervisor"
	case "ops_lead":
		return "Operations Lead"
	default:
		return strings.ReplaceAll(role, "_", " ")
	}
}

func roleDescription(role string) string {
	switch role {
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

func sanitizeNextPath(next string) string {
	trimmed := strings.TrimSpace(next)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "//") {
		return "/app/dashboard"
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "/app/dashboard"
	}
	if candidate := parsed.RequestURI(); strings.HasPrefix(candidate, "/") && !strings.HasPrefix(candidate, "//") {
		return candidate
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

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *bufferedResponseWriter) FlushTo(target http.ResponseWriter) {
	for key, values := range w.header {
		for _, value := range values {
			target.Header().Add(key, value)
		}
	}
	target.WriteHeader(w.status)
	_, _ = target.Write([]byte(w.body.String()))
}

func localeDirection(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "ar") {
		return "rtl"
	}
	return "ltr"
}
