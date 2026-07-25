package pwa_test

import (
	"github.com/monstercameron/GoWebComponents/v5/pwa"
)

// ExampleBuildCacheStoragePlan shows building a service-worker cache plan from an
// asset plan, the input to a generated service worker for offline support.
func ExampleBuildCacheStoragePlan() {
	parsePlan, parseErr := pwa.BuildCacheStoragePlan(pwa.ServiceWorkerAssetPlan{})
	if parseErr != nil {
		return
	}
	_ = parsePlan
}
