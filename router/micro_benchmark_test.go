package router

import (
	"net/url"
	"testing"
)

type benchmarkRouteParams struct {
	UserID  string
	OrderID string
}

func (p benchmarkRouteParams) RouteParams() map[string]string {
	return map[string]string{
		"id":      p.UserID,
		"orderID": p.OrderID,
	}
}

type benchmarkRouteQuery struct {
	Tab string
}

func (q benchmarkRouteQuery) RouteQuery() url.Values {
	values := url.Values{}
	values.Set("tab", q.Tab)
	return values
}

func BenchmarkRouteContractMustHrefForMicro(b *testing.B) {
	contract := MustDefineRoute("/users/:id/orders/:orderID")
	params := benchmarkRouteParams{UserID: "42", OrderID: "A-99"}
	query := benchmarkRouteQuery{Tab: "billing"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = contract.MustHrefFor(params, query)
	}
}
