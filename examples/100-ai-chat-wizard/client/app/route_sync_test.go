//go:build js && wasm

package app

import (
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
)

func TestShouldRedirectUnauthenticatedRouteToLanding(parseT *testing.T) {
	parseTests := []struct {
		name            string
		currentPath     string
		isAuthResolved  bool
		isAuthenticated bool
		want            bool
	}{
		{
			name:            "resolved unauthenticated app root redirects",
			currentPath:     chatRouteRoot,
			isAuthResolved:  true,
			isAuthenticated: false,
			want:            true,
		},
		{
			name:            "resolved unauthenticated thread route redirects",
			currentPath:     "/app/thread/thread-public-id",
			isAuthResolved:  true,
			isAuthenticated: false,
			want:            true,
		},
		{
			name:            "resolved unauthenticated landing does not redirect",
			currentPath:     authLandingRoute,
			isAuthResolved:  true,
			isAuthenticated: false,
			want:            false,
		},
		{
			name:            "authenticated app route does not redirect",
			currentPath:     chatRouteRoot,
			isAuthResolved:  true,
			isAuthenticated: true,
			want:            false,
		},
		{
			name:            "unresolved auth route does not redirect",
			currentPath:     chatRouteRoot,
			isAuthResolved:  false,
			isAuthenticated: false,
			want:            false,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := shouldRedirectUnauthenticatedRouteToLanding(parseTest.currentPath, parseTest.isAuthResolved, parseTest.isAuthenticated); parseGot != parseTest.want {
				parseT2.Fatalf("shouldRedirectUnauthenticatedRouteToLanding(%q, %v, %v) = %v, want %v", parseTest.currentPath, parseTest.isAuthResolved, parseTest.isAuthenticated, parseGot, parseTest.want)
			}
		})
	}
}

func TestParseResolvePostLoginRoute(parseT *testing.T) {
	parseTests := []struct {
		name             string
		intentPath       string
		canAccessAdmin   bool
		hasRoleSummary   bool
		wantResolvedPath string
	}{
		{
			name:             "blank intent falls back to chat root",
			intentPath:       "",
			canAccessAdmin:   false,
			hasRoleSummary:   true,
			wantResolvedPath: chatRouteRoot,
		},
		{
			name:             "admin intent denied for non-admin role",
			intentPath:       "/app/dashboard",
			canAccessAdmin:   false,
			hasRoleSummary:   true,
			wantResolvedPath: chatRouteRoot,
		},
		{
			name:             "admin intent allowed for admin role",
			intentPath:       "/app/dashboard/usage",
			canAccessAdmin:   true,
			hasRoleSummary:   true,
			wantResolvedPath: "/app/dashboard/usage",
		},
		{
			name:             "chat thread intent preserved for normal role",
			intentPath:       "/app/thread/thread-public-id",
			canAccessAdmin:   false,
			hasRoleSummary:   true,
			wantResolvedPath: "/app/thread/thread-public-id",
		},
		{
			name:             "non-chat intent fails closed to chat root",
			intentPath:       "/pricing",
			canAccessAdmin:   true,
			hasRoleSummary:   true,
			wantResolvedPath: chatRouteRoot,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			var parseRoleSummary *chatpb.AuthRoleSummary
			if parseTest.hasRoleSummary {
				parseRoleSummary = &chatpb.AuthRoleSummary{CanAccessAdmin: parseTest.canAccessAdmin}
			}
			if parseGot := parseResolvePostLoginRoute(parseTest.intentPath, parseRoleSummary); parseGot != parseTest.wantResolvedPath {
				parseT2.Fatalf("parseResolvePostLoginRoute(%q, can_access_admin=%v, has_summary=%v) = %q, want %q", parseTest.intentPath, parseTest.canAccessAdmin, parseTest.hasRoleSummary, parseGot, parseTest.wantResolvedPath)
			}
		})
	}
}

func TestPostLoginRouteIntentHelpers(parseT *testing.T) {
	if parseGot := parseNormalizePostLoginRouteIntent(marketingPricingRoute); parseGot != "" {
		parseT.Fatalf("parseNormalizePostLoginRouteIntent(%q) = %q, want empty", marketingPricingRoute, parseGot)
	}
	if parseGot := parseNormalizePostLoginRouteIntent(chatRouteRoot); parseGot != chatRouteRoot {
		parseT.Fatalf("parseNormalizePostLoginRouteIntent(%q) = %q, want %q", chatRouteRoot, parseGot, chatRouteRoot)
	}
	parseTests := []struct {
		path string
		want bool
	}{
		{path: "/app", want: false},
		{path: "/app/dashboard", want: true},
		{path: "/app/admin/users", want: true},
		{path: "/app/su/control", want: true},
		{path: "/app/settings?panel=settings-profile", want: false},
		{path: "/app/settings?panel=settings-dashboard", want: true},
		{path: "/app/settings?panel=settings-superuser", want: true},
	}
	for _, parseTest := range parseTests {
		if parseGot := parseIsAdminRouteIntentPath(parseTest.path); parseGot != parseTest.want {
			parseT.Fatalf("parseIsAdminRouteIntentPath(%q) = %v, want %v", parseTest.path, parseGot, parseTest.want)
		}
	}
}

func TestShouldResetDraftForRootRoute(parseT *testing.T) {
	parseTests := []struct {
		name                string
		threadRoutePublicID string
		activeConvID        int64
		activeConvPublicID  string
		want                bool
	}{
		{
			name:                "newly created thread without public id stays in place",
			threadRoutePublicID: "",
			activeConvID:        42,
			activeConvPublicID:  "",
			want:                false,
		},
		{
			name:                "loaded thread clears when user returns to root",
			threadRoutePublicID: "",
			activeConvID:        42,
			activeConvPublicID:  "thread-public-id",
			want:                true,
		},
		{
			name:                "thread route keeps active conversation",
			threadRoutePublicID: "thread-public-id",
			activeConvID:        42,
			activeConvPublicID:  "thread-public-id",
			want:                false,
		},
		{
			name:                "missing active conversation never resets",
			threadRoutePublicID: "",
			activeConvID:        0,
			activeConvPublicID:  "thread-public-id",
			want:                false,
		},
		{
			name:                "blank public id is treated as unresolved",
			threadRoutePublicID: " ",
			activeConvID:        42,
			activeConvPublicID:  " ",
			want:                false,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := shouldResetDraftForRootRoute(parseTest.threadRoutePublicID, parseTest.activeConvID, parseTest.activeConvPublicID); parseGot != parseTest.want {
				parseT2.Fatalf("shouldResetDraftForRootRoute(%q, %d, %q) = %v, want %v", parseTest.threadRoutePublicID, parseTest.activeConvID, parseTest.activeConvPublicID, parseGot, parseTest.want)
			}
		})
	}
}

func TestShouldWarnPendingRootRoute(parseT *testing.T) {
	parseTests := []struct {
		name                string
		threadRoutePublicID string
		activeConvID        int64
		activeConvPublicID  string
		want                bool
	}{
		{
			name:                "newly created thread without public id warns",
			threadRoutePublicID: "",
			activeConvID:        42,
			activeConvPublicID:  "",
			want:                true,
		},
		{
			name:                "loaded thread with public id does not warn",
			threadRoutePublicID: "",
			activeConvID:        42,
			activeConvPublicID:  "thread-public-id",
			want:                false,
		},
		{
			name:                "thread route does not warn",
			threadRoutePublicID: "thread-public-id",
			activeConvID:        42,
			activeConvPublicID:  "",
			want:                false,
		},
		{
			name:                "missing active conversation does not warn",
			threadRoutePublicID: "",
			activeConvID:        0,
			activeConvPublicID:  "",
			want:                false,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := shouldWarnPendingRootRoute(parseTest.threadRoutePublicID, parseTest.activeConvID, parseTest.activeConvPublicID); parseGot != parseTest.want {
				parseT2.Fatalf("shouldWarnPendingRootRoute(%q, %d, %q) = %v, want %v", parseTest.threadRoutePublicID, parseTest.activeConvID, parseTest.activeConvPublicID, parseGot, parseTest.want)
			}
		})
	}
}

func TestShouldResolveConversationRoute(parseT *testing.T) {
	parseTests := []struct {
		name                string
		threadRoutePublicID string
		activeConvPublicID  string
		want                bool
	}{
		{
			name:                "different thread route resolves",
			threadRoutePublicID: "thread-a",
			activeConvPublicID:  "thread-b",
			want:                true,
		},
		{
			name:                "matching thread route does not resolve",
			threadRoutePublicID: "thread-a",
			activeConvPublicID:  "thread-a",
			want:                false,
		},
		{
			name:                "blank thread route does not resolve",
			threadRoutePublicID: " ",
			activeConvPublicID:  "thread-a",
			want:                false,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := shouldResolveConversationRoute(parseTest.threadRoutePublicID, parseTest.activeConvPublicID); parseGot != parseTest.want {
				parseT2.Fatalf("shouldResolveConversationRoute(%q, %q) = %v, want %v", parseTest.threadRoutePublicID, parseTest.activeConvPublicID, parseGot, parseTest.want)
			}
		})
	}
}

func TestShouldNormalizeActiveConversationRoute(parseT *testing.T) {
	parseTests := []struct {
		name                string
		threadRoutePublicID string
		activeConvPublicID  string
		want                bool
	}{
		{
			name:                "root route follows active conversation",
			threadRoutePublicID: "",
			activeConvPublicID:  "thread-a",
			want:                true,
		},
		{
			name:                "matching thread route stays normalized",
			threadRoutePublicID: "thread-a",
			activeConvPublicID:  "thread-a",
			want:                true,
		},
		{
			name:                "different thread route is preserved",
			threadRoutePublicID: "thread-a",
			activeConvPublicID:  "thread-b",
			want:                false,
		},
		{
			name:                "blank active conversation cannot normalize",
			threadRoutePublicID: "thread-a",
			activeConvPublicID:  "",
			want:                false,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := shouldNormalizeActiveConversationRoute(parseTest.threadRoutePublicID, parseTest.activeConvPublicID); parseGot != parseTest.want {
				parseT2.Fatalf("shouldNormalizeActiveConversationRoute(%q, %q) = %v, want %v", parseTest.threadRoutePublicID, parseTest.activeConvPublicID, parseGot, parseTest.want)
			}
		})
	}
}

func TestThreadRoutePublicIDFromPath(parseT *testing.T) {
	parseTests := []struct {
		path string
		want string
	}{
		{path: "/app/thread/thread-a", want: "thread-a"},
		{path: "/app/thread/thread-a/canvas/artifact-1", want: "thread-a"},
		{path: "/app/thread/thread-a?panel=settings-intelligence", want: "thread-a"},
		{path: "/app", want: ""},
		{path: "/", want: ""},
	}

	for _, parseTest := range parseTests {
		if parseGot := parseThreadRoutePublicIDFromPath(parseTest.path); parseGot != parseTest.want {
			parseT.Fatalf("threadRoutePublicIDFromPath(%q) = %q, want %q", parseTest.path, parseGot, parseTest.want)
		}
	}
}

func TestShouldNavigateLandingRoute(parseT *testing.T) {
	parseTests := []struct {
		name        string
		currentPath string
		targetPath  string
		want        bool
	}{
		{
			name:        "home navigates to auth landing",
			currentPath: marketingHomeRoute,
			targetPath:  authLandingRoute,
			want:        true,
		},
		{
			name:        "same auth landing route is ignored",
			currentPath: authLandingRoute,
			targetPath:  authLandingRoute,
			want:        false,
		},
		{
			name:        "pricing navigates to auth landing",
			currentPath: marketingPricingRoute,
			targetPath:  authLandingRoute,
			want:        true,
		},
		{
			name:        "blank target is ignored",
			currentPath: marketingHomeRoute,
			targetPath:  " ",
			want:        false,
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := shouldNavigateLandingRoute(parseTest.currentPath, parseTest.targetPath); parseGot != parseTest.want {
				parseT2.Fatalf("shouldNavigateLandingRoute(%q, %q) = %v, want %v", parseTest.currentPath, parseTest.targetPath, parseGot, parseTest.want)
			}
		})
	}
}
