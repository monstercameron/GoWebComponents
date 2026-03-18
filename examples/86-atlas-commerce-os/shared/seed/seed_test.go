package seed

import "testing"

func TestInventorySeedValues(t *testing.T) {
	if len(InventoryColumns) < 6 {
		t.Fatalf("expected inventory columns to be defined")
	}
	if len(InventoryFilters) < 4 {
		t.Fatalf("expected inventory filters to be defined")
	}
	if len(SavedViews) < 2 {
		t.Fatalf("expected saved views to be defined")
	}
}

func TestOperationalScreensHaveStats(t *testing.T) {
	for _, screen := range []ScreenData{Inventory, SKUDetail, Transfers, Receiving, Comments, Settings} {
		if len(screen.Stats) == 0 {
			t.Fatalf("expected stats for screen %s", screen.Title)
		}
	}
}

func TestPrimaryPublicScreensExposeTitlesAndBullets(t *testing.T) {
	for _, screen := range []ScreenData{Landing, Catalog, Product, Warehouses} {
		if screen.Title == "" {
			t.Fatal("expected public screen title")
		}
		if len(screen.Bullets) < 3 {
			t.Fatalf("expected richer guidance bullets for %s", screen.Title)
		}
	}
}
