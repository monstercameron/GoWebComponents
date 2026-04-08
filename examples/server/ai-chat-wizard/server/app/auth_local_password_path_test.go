package app

import "testing"

// TestLocalPasswordAuthPathSupportsQACredentialsAndResetWithoutExternalIdentity verifies
// the local email/password path remains first-class for QA accounts, durable sessions,
// and password reset flows without requiring external identity linkage.
func TestLocalPasswordAuthPathSupportsQACredentialsAndResetWithoutExternalIdentity(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())

	parseCustomerUser, parseErr := parseAuth.parseSignup("customer@email.com", "password", "Customer User")
	if parseErr != nil {
		parseT.Fatalf("parseSignup(customer): %v", parseErr)
	}
	parseAdminUser, parseErr := parseAuth.parseSignup("admin@email.com", "password", "Admin User")
	if parseErr != nil {
		parseT.Fatalf("parseSignup(admin): %v", parseErr)
	}

	if _, parseErr = parseAuth.parseLogin("customer@email.com", "password"); parseErr != nil {
		parseT.Fatalf("parseLogin(customer): %v", parseErr)
	}
	parseAdminLoginUser, parseErr := parseAuth.parseLogin("admin@email.com", "password")
	if parseErr != nil {
		parseT.Fatalf("parseLogin(admin): %v", parseErr)
	}
	parseAdminToken, parseErr := parseAuth.issueToken(parseAdminLoginUser)
	if parseErr != nil {
		parseT.Fatalf("issueToken(admin): %v", parseErr)
	}
	_, parseClaims, parseErr := parseAuth.parseTokenWithMetadata(parseAdminToken, parseAuthRequestMetadata{})
	if parseErr != nil {
		parseT.Fatalf("parseTokenWithMetadata(admin): %v", parseErr)
	}
	parseSessionRow, isParseSessionFound, parseErr := parseStore.parseGetAuthSessionBySessionID(parseClaims.SessionID)
	if parseErr != nil {
		parseT.Fatalf("parseGetAuthSessionBySessionID(admin): %v", parseErr)
	}
	if !isParseSessionFound || parseSessionRow.UserID != parseAdminUser.ID || parseSessionRow.TokenVersion <= 0 {
		parseT.Fatalf("unexpected admin auth session row: found=%v row=%+v", isParseSessionFound, parseSessionRow)
	}

	parseResetToken, parseErr := parseAuth.parseBeginPasswordResetToken("customer@email.com", "127.0.0.1")
	if parseErr != nil {
		parseT.Fatalf("parseBeginPasswordResetToken(customer): %v", parseErr)
	}
	if parseResetToken == "" {
		parseT.Fatal("expected non-empty customer reset token")
	}
	if parseErr = parseAuth.parseCompletePasswordResetWithToken(parseResetToken, "password-updated"); parseErr != nil {
		parseT.Fatalf("parseCompletePasswordResetWithToken(customer): %v", parseErr)
	}
	if _, parseErr = parseAuth.parseLogin("customer@email.com", "password"); parseErr == nil {
		parseT.Fatal("expected old customer password login to fail after reset")
	}
	if _, parseErr = parseAuth.parseLogin("customer@email.com", "password-updated"); parseErr != nil {
		parseT.Fatalf("parseLogin(customer, updated): %v", parseErr)
	}

	parseCustomerIdentities, parseErr := parseStore.parseListAuthIdentitiesByUser(parseCustomerUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListAuthIdentitiesByUser(customer): %v", parseErr)
	}
	parseAdminIdentities, parseErr := parseStore.parseListAuthIdentitiesByUser(parseAdminUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseListAuthIdentitiesByUser(admin): %v", parseErr)
	}
	if len(parseCustomerIdentities) != 0 || len(parseAdminIdentities) != 0 {
		parseT.Fatalf("expected local password flow without external identities, customer=%d admin=%d", len(parseCustomerIdentities), len(parseAdminIdentities))
	}
}
