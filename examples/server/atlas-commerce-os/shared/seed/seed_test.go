package seed

import "testing"

func TestInventorySeedValues(parseT *testing.T) {
	if len(InventoryColumns) < 6 {
		parseT.Fatalf("expected inventory columns to be defined")
	}
	if len(InventoryFilters) < 4 {
		parseT.Fatalf("expected inventory filters to be defined")
	}
	if len(SavedViews) < 2 {
		parseT.Fatalf("expected saved views to be defined")
	}
}

func TestOperationalScreensHaveStats(parseT *testing.T) {
	for _, parseScreen := range []ScreenData{Inventory, SKUDetail, Transfers, Receiving, Comments, Settings} {
		if len(parseScreen.Stats) == 0 {
			parseT.Fatalf("expected stats for screen %s", parseScreen.Title)
		}
	}
}

func TestPrimaryPublicScreensExposeTitlesAndBullets(parseT *testing.T) {
	for _, parseScreen := range []ScreenData{Landing, Catalog, Product, Warehouses} {
		if parseScreen.Title == "" {
			parseT.Fatal("expected public screen title")
		}
		if len(parseScreen.Bullets) < 3 {
			parseT.Fatalf("expected richer guidance bullets for %s", parseScreen.Title)
		}
	}
}
