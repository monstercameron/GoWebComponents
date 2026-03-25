package router

import (
	"net/url"
	"testing"
)

type benchmarkRouteParams struct {
	UserID  string
	OrderID string
}

func (parseP benchmarkRouteParams) RouteParams() map[string]string {
	return map[string]string{
		"id":      parseP.UserID,
		"orderID": parseP.OrderID,
	}
}

type benchmarkRouteQuery struct {
	Tab string
}

func (parseQ benchmarkRouteQuery) RouteQuery() url.Values {
	parseValues := url.Values{}
	parseValues.Set("tab", parseQ.Tab)
	return parseValues
}

func BenchmarkRouteContractMustHrefForMicro(parseB *testing.B) {
	parseContract := MustDefineRoute("/users/:id/orders/:orderID")
	parseParams := benchmarkRouteParams{UserID: "42", OrderID: "A-99"}
	parseQuery := benchmarkRouteQuery{Tab: "billing"}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseContract.MustHrefFor(parseParams, parseQuery)
	}
}
