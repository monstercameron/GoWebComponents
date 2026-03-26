package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	serverauth "github.com/monstercameron/GoWebComponents/examples/86-atlas-commerce-os/server/auth"
)

// TestModerationFlowIntegration verifies pending comment intake, moderation action, and public approved visibility updates.
func TestModerationFlowIntegration(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parsePublicApprovedBeforeReq := httptest.NewRequest(http.MethodGet, "/api/public/products/frame-desk/comments?status=approved", nil)
	parsePublicApprovedBeforeRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parsePublicApprovedBeforeRes, parsePublicApprovedBeforeReq)
	if parsePublicApprovedBeforeRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parsePublicApprovedBeforeRes.Code)
	}
	var parsePublicApprovedBefore struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if parseErr := json.Unmarshal(parsePublicApprovedBeforeRes.Body.Bytes(), &parsePublicApprovedBefore); parseErr != nil {
		parseT.Fatalf("decode public approved comments before payload: %v", parseErr)
	}

	parsePublicCSRFTok, parsePublicCSRFCookie := loadCSRFFromPage(parseT, parseServer, "/shop/frame-desk")
	parseSubject := "Integration moderation queue visibility check"
	parseCommentCreateForm := url.Values{
		"author_name": {`Integration Reviewer`},
		"reaction":    {"up"},
		"subject":     {parseSubject},
		"body":        {"This integration test verifies pending-to-approved moderation routing."},
		"csrf_token":  {parsePublicCSRFTok},
	}
	parseCommentCreateReq := httptest.NewRequest(http.MethodPost, "/api/public/products/frame-desk/comments", strings.NewReader(parseCommentCreateForm.Encode()))
	parseCommentCreateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseCommentCreateReq.Header.Set("Referer", "http://example.com/shop/frame-desk")
	parseCommentCreateReq.AddCookie(parsePublicCSRFCookie)
	parseCommentCreateRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseCommentCreateRes, parseCommentCreateReq)
	if parseCommentCreateRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d body=%q", http.StatusSeeOther, parseCommentCreateRes.Code, parseCommentCreateRes.Body.String())
	}

	parsePendingReq := httptest.NewRequest(http.MethodGet, "/api/app/comments?status=pending", nil)
	parsePendingReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parsePendingRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parsePendingRes, parsePendingReq)
	if parsePendingRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parsePendingRes.Code)
	}
	var parsePendingPayload struct {
		Items []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
			Status  string `json:"status"`
		} `json:"items"`
	}
	if parseErr := json.Unmarshal(parsePendingRes.Body.Bytes(), &parsePendingPayload); parseErr != nil {
		parseT.Fatalf("decode pending moderation payload: %v", parseErr)
	}
	parsePendingCommentID := ""
	for _, parseItem := range parsePendingPayload.Items {
		if parseItem.Subject == parseSubject {
			parsePendingCommentID = parseItem.ID
			if parseItem.Status != "pending" {
				parseT.Fatalf("expected pending status before moderation, got %q", parseItem.Status)
			}
			break
		}
	}
	if parsePendingCommentID == "" {
		parseT.Fatalf("expected new public comment subject %q in pending queue", parseSubject)
	}

	parseInternalCSRFTok, parseInternalCSRFCookie := loadCSRFFromPage(parseT, parseServer, "/app/comments")
	parseModerateForm := url.Values{
		"status":     {"approved"},
		"reason":     {"Approved during integration moderation flow."},
		"csrf_token": {parseInternalCSRFTok},
	}
	parseModerateReq := httptest.NewRequest(http.MethodPost, "/api/app/comments/"+parsePendingCommentID+"/moderate", strings.NewReader(parseModerateForm.Encode()))
	parseModerateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parseModerateReq.Header.Set("Referer", "http://example.com/app/comments")
	parseModerateReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseModerateReq.AddCookie(parseInternalCSRFCookie)
	parseModerateRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseModerateRes, parseModerateReq)
	if parseModerateRes.Code != http.StatusSeeOther {
		parseT.Fatalf("expected %d, got %d body=%q", http.StatusSeeOther, parseModerateRes.Code, parseModerateRes.Body.String())
	}

	parseApprovedReq := httptest.NewRequest(http.MethodGet, "/api/app/comments?status=approved", nil)
	parseApprovedReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parseApprovedRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parseApprovedRes, parseApprovedReq)
	if parseApprovedRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parseApprovedRes.Code)
	}
	var parseApprovedPayload struct {
		Items []struct {
			ID               string `json:"id"`
			Status           string `json:"status"`
			ModerationReason string `json:"moderationReason"`
		} `json:"items"`
	}
	if parseErr := json.Unmarshal(parseApprovedRes.Body.Bytes(), &parseApprovedPayload); parseErr != nil {
		parseT.Fatalf("decode approved moderation payload: %v", parseErr)
	}
	isParseApprovedVisible := false
	for _, parseItem := range parseApprovedPayload.Items {
		if parseItem.ID != parsePendingCommentID {
			continue
		}
		if parseItem.Status != "approved" {
			parseT.Fatalf("expected moderated status approved, got %q", parseItem.Status)
		}
		if !strings.Contains(parseItem.ModerationReason, "integration moderation flow") {
			parseT.Fatalf("expected moderation reason to persist, got %q", parseItem.ModerationReason)
		}
		isParseApprovedVisible = true
		break
	}
	if !isParseApprovedVisible {
		parseT.Fatalf("expected moderated comment %q in approved queue", parsePendingCommentID)
	}

	parsePendingAfterReq := httptest.NewRequest(http.MethodGet, "/api/app/comments?status=pending", nil)
	parsePendingAfterReq.AddCookie(&http.Cookie{Name: serverauth.MockSessionCookieName, Value: "inventory_manager"})
	parsePendingAfterRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parsePendingAfterRes, parsePendingAfterReq)
	if parsePendingAfterRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parsePendingAfterRes.Code)
	}
	var parsePendingAfter struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if parseErr := json.Unmarshal(parsePendingAfterRes.Body.Bytes(), &parsePendingAfter); parseErr != nil {
		parseT.Fatalf("decode pending queue after moderation payload: %v", parseErr)
	}
	for _, parseItem := range parsePendingAfter.Items {
		if parseItem.ID == parsePendingCommentID {
			parseT.Fatalf("expected moderated comment %q to leave pending queue", parsePendingCommentID)
		}
	}

	parsePublicApprovedAfterReq := httptest.NewRequest(http.MethodGet, "/api/public/products/frame-desk/comments?status=approved", nil)
	parsePublicApprovedAfterRes := httptest.NewRecorder()
	parseServer.routes().ServeHTTP(parsePublicApprovedAfterRes, parsePublicApprovedAfterReq)
	if parsePublicApprovedAfterRes.Code != http.StatusOK {
		parseT.Fatalf("expected %d, got %d", http.StatusOK, parsePublicApprovedAfterRes.Code)
	}
	var parsePublicApprovedAfter struct {
		Items []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
			Status  string `json:"status"`
		} `json:"items"`
	}
	if parseErr := json.Unmarshal(parsePublicApprovedAfterRes.Body.Bytes(), &parsePublicApprovedAfter); parseErr != nil {
		parseT.Fatalf("decode public approved comments after payload: %v", parseErr)
	}
	if len(parsePublicApprovedAfter.Items) < len(parsePublicApprovedBefore.Items)+1 {
		parseT.Fatalf("expected approved public thread growth, before=%d after=%d", len(parsePublicApprovedBefore.Items), len(parsePublicApprovedAfter.Items))
	}
	isParsePublicVisible := false
	for _, parseItem := range parsePublicApprovedAfter.Items {
		if parseItem.ID == parsePendingCommentID && parseItem.Subject == parseSubject && parseItem.Status == "approved" {
			isParsePublicVisible = true
			break
		}
	}
	if !isParsePublicVisible {
		parseT.Fatalf("expected approved public visibility for moderated comment %q", parsePendingCommentID)
	}
}
