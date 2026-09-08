package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/server/auth"
)

// TestReceivingReconciliationFlowIntegration verifies receiving session load, discrepancy branch updates, and final closeout persistence.
func TestReceivingReconciliationFlowIntegration(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseReceivingBeforeReq := httptest.NewRequest(http.MethodGet, "/api/app/receiving", nil)
	parseReceivingBeforeReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReceivingBeforeRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseReceivingBeforeRes, parseReceivingBeforeReq)
	if parseReceivingBeforeRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseReceivingBeforeRes.Code)
	}
	var parseReceivingBefore struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if parseErr := json.Unmarshal(parseReceivingBeforeRes.Body.Bytes(), &parseReceivingBefore); parseErr != nil {
		parseT.Fatalf("decode receiving queue payload: %v", parseErr)
	}
	if len(parseReceivingBefore.Items) == 0 {
		parseT.Fatal("expected at least one receiving session")
	}

	parseSessionID := "rcv-illinois-001"
	parseReceivingDetailReq := httptest.NewRequest(http.MethodGet, "/api/app/receiving/"+parseSessionID, nil)
	parseReceivingDetailReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReceivingDetailRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseReceivingDetailRes, parseReceivingDetailReq)
	if parseReceivingDetailRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseReceivingDetailRes.Code)
	}
	var parseReceivingDetail struct {
		Session struct {
			ID                 string `json:"id"`
			Status             string `json:"status"`
			DiscrepancySummary string `json:"discrepancySummary"`
		} `json:"session"`
		Lines []struct {
			ExpectedQuantity  int    `json:"expectedQuantity"`
			ActualQuantity    int    `json:"actualQuantity"`
			DiscrepancyReason string `json:"discrepancyReason"`
		} `json:"lines"`
	}
	if parseErr := json.Unmarshal(parseReceivingDetailRes.Body.Bytes(), &parseReceivingDetail); parseErr != nil {
		parseT.Fatalf("decode receiving detail payload: %v", parseErr)
	}
	if parseReceivingDetail.Session.ID != parseSessionID {
		parseT.Fatalf("expected receiving session %q, got %q", parseSessionID, parseReceivingDetail.Session.ID)
	}
	if len(parseReceivingDetail.Lines) == 0 {
		parseT.Fatal("expected receiving detail lines")
	}
	isParseDiscrepancyVisible := false
	for _, parseLine := range parseReceivingDetail.Lines {
		if parseLine.ActualQuantity != parseLine.ExpectedQuantity || strings.TrimSpace(parseLine.DiscrepancyReason) != "" {
			isParseDiscrepancyVisible = true
			break
		}
	}
	if !isParseDiscrepancyVisible {
		parseT.Fatal("expected at least one discrepancy branch candidate in receiving detail lines")
	}

	parseCSRFTok, parseCSRFCookie := loadCSRFFromPage(parseT, parseServer, "/app/receiving/"+parseSessionID)

	parseReviewForm := url.Values{
		"status":              {"in_review"},
		"discrepancy_summary": {"Line recount: expected 18, actual 16 on cable-bridge; dock hold initiated."},
		"csrf_token":          {parseCSRFTok},
	}
	parseReviewReq := httptest.NewRequest(http.MethodPost, "/api/app/receiving/"+parseSessionID+"/reconcile", strings.NewReader(parseReviewForm.Encode()))
	parseReviewReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseReviewReq.Header.Set("Referer", "http://example.com/app/receiving/"+parseSessionID)
	parseReviewReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReviewReq.AddCookie(parseCSRFCookie)
	parseReviewRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseReviewRes, parseReviewReq)
	if parseReviewRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d body=%q", http.StatusSeeOther, parseReviewRes.Code, parseReviewRes.Body.String())
	}

	parseDetailInReviewReq := httptest.NewRequest(http.MethodGet, "/api/app/receiving/"+parseSessionID, nil)
	parseDetailInReviewReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseDetailInReviewRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseDetailInReviewRes, parseDetailInReviewReq)
	if parseDetailInReviewRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseDetailInReviewRes.Code)
	}
	var parseDetailInReview struct {
		Session struct {
			Status             string `json:"status"`
			DiscrepancySummary string `json:"discrepancySummary"`
		} `json:"session"`
	}
	if parseErr := json.Unmarshal(parseDetailInReviewRes.Body.Bytes(), &parseDetailInReview); parseErr != nil {
		parseT.Fatalf("decode receiving in-review payload: %v", parseErr)
	}
	if parseDetailInReview.Session.Status != "in_review" {
		parseT.Fatalf("expected receiving status in_review, got %q", parseDetailInReview.Session.Status)
	}
	if !strings.Contains(parseDetailInReview.Session.DiscrepancySummary, "expected 18, actual 16") {
		parseT.Fatalf("expected discrepancy branch summary to persist, got %q", parseDetailInReview.Session.DiscrepancySummary)
	}

	parseFinalizeForm := url.Values{
		"status":              {"closed"},
		"discrepancy_summary": {"Inventory adjusted and shipment reconciled. Session closed."},
		"csrf_token":          {parseCSRFTok},
	}
	parseFinalizeReq := httptest.NewRequest(http.MethodPost, "/api/app/receiving/"+parseSessionID+"/reconcile", strings.NewReader(parseFinalizeForm.Encode()))
	parseFinalizeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseFinalizeReq.Header.Set("Referer", "http://example.com/app/receiving/"+parseSessionID)
	parseFinalizeReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseFinalizeReq.AddCookie(parseCSRFCookie)
	parseFinalizeRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseFinalizeRes, parseFinalizeReq)
	if parseFinalizeRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d body=%q", http.StatusSeeOther, parseFinalizeRes.Code, parseFinalizeRes.Body.String())
	}

	parseReceivingAfterReq := httptest.NewRequest(http.MethodGet, "/api/app/receiving", nil)
	parseReceivingAfterReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseReceivingAfterRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseReceivingAfterRes, parseReceivingAfterReq)
	if parseReceivingAfterRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseReceivingAfterRes.Code)
	}
	var parseReceivingAfter struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if parseErr := json.Unmarshal(parseReceivingAfterRes.Body.Bytes(), &parseReceivingAfter); parseErr != nil {
		parseT.Fatalf("decode receiving queue after payload: %v", parseErr)
	}
	isParseClosedVisible := false
	for _, parseItem := range parseReceivingAfter.Items {
		if parseItem.ID == parseSessionID && parseItem.Status == "closed" {
			isParseClosedVisible = true
			break
		}
	}
	if !isParseClosedVisible {
		parseT.Fatalf("expected receiving queue to show %q as closed", parseSessionID)
	}

	parseDetailClosedReq := httptest.NewRequest(http.MethodGet, "/api/app/receiving/"+parseSessionID, nil)
	parseDetailClosedReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseDetailClosedRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseDetailClosedRes, parseDetailClosedReq)
	if parseDetailClosedRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseDetailClosedRes.Code)
	}
	var parseDetailClosed struct {
		Session struct {
			Status             string `json:"status"`
			DiscrepancySummary string `json:"discrepancySummary"`
		} `json:"session"`
	}
	if parseErr := json.Unmarshal(parseDetailClosedRes.Body.Bytes(), &parseDetailClosed); parseErr != nil {
		parseT.Fatalf("decode receiving closed payload: %v", parseErr)
	}
	if parseDetailClosed.Session.Status != "closed" {
		parseT.Fatalf("expected receiving status closed, got %q", parseDetailClosed.Session.Status)
	}
	if !strings.Contains(parseDetailClosed.Session.DiscrepancySummary, "Session closed") {
		parseT.Fatalf("expected final reconciliation summary to persist, got %q", parseDetailClosed.Session.DiscrepancySummary)
	}
}
