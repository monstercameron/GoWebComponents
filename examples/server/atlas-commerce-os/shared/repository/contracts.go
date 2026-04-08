package repository

import "context"

type Product struct {
	SKU            string
	Slug           string
	Title          string
	Category       string
	PriceCents     int
	Status         string
	Summary        string
	SEODescription string
}

type InventoryRow struct {
	ID            string
	SKU           string
	Slug          string
	Title         string
	Category      string
	PriceCents    int
	ProductStatus string
	WarehouseID   string
	WarehouseName string
	OnHand        int
	Reserved      int
	Available     int
	CoverDays     int
	Inbound       int
	Damaged       int
	ReorderPoint  int
	SafetyStock   int
	Status        string
	WeeklyUnits   int
	WeeklyRevenue int
	SellThrough   int
	DemandScore   int
	RegionalShare int
	ReorderUnits  int
	MarketPressure string
	MarketSignal  string
	UpdatedAt     string
}

type InventoryQuery struct {
	Warehouse     string
	Category      string
	Supplier      string
	StockHealth   string
	Availability  string
	Search        string
	SortKey       string
	SortDirection string
}

type ModerationQuery struct {
	Status   string
	Search   string
	SortKey  string
	PageSize int
}

type SavedView struct {
	ID            string
	Name          string
	Scope         string
	SortKey       string
	SortDirection string
	Density       string
	WarehouseID   string
	FiltersJSON   string
}

type ProductRepository interface {
	Catalog(context.Context) ([]Product, error)
	BySlug(context.Context, string) (Product, error)
}

type InventoryRepository interface {
	List(context.Context, InventoryQuery) ([]InventoryRow, error)
	BySKU(context.Context, string) (InventoryRow, error)
}

type ModerationRepository interface {
	List(context.Context, ModerationQuery) ([]string, error)
	Approve(context.Context, string) error
	Reject(context.Context, string) error
	Flag(context.Context, string) error
}

type PreferencesRepository interface {
	Save(context.Context, map[string]string) error
	Load(context.Context, string) (map[string]string, error)
}

type SavedViewsRepository interface {
	List(context.Context, string) ([]SavedView, error)
	Save(context.Context, SavedView) error
}
