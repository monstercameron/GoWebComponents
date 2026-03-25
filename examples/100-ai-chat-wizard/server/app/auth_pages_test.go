package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthPageDataHelpersAndRendering(t *testing.T) {
	loginData := loginPageData(" user@example.com ", " bad credentials ")
	if loginData.Email != "user@example.com" || loginData.Error != "bad credentials" {
		t.Fatalf("unexpected login page data: %+v", loginData)
	}
	signupData := signupPageData(" Demo ", " user@example.com ", " existing user ")
	if !signupData.ShowNameField || signupData.Name != "Demo" || signupData.Error != "existing user" {
		t.Fatalf("unexpected signup page data: %+v", signupData)
	}

	writer := httptest.NewRecorder()
	renderAuthPage(writer, http.StatusAccepted, signupData)
	response := writer.Result()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("unexpected auth page status: %d", response.StatusCode)
	}
	body := writer.Body.String()
	if !strings.Contains(body, "Create your account") || !strings.Contains(body, "user@example.com") {
		t.Fatalf("rendered auth page missing expected content: %q", body)
	}
}
