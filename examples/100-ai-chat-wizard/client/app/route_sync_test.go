//go:build js && wasm

package app

import "testing"

func TestShouldResetDraftForRootRoute(t *testing.T) {
	tests := []struct {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldResetDraftForRootRoute(test.threadRoutePublicID, test.activeConvID, test.activeConvPublicID); got != test.want {
				t.Fatalf("shouldResetDraftForRootRoute(%q, %d, %q) = %v, want %v", test.threadRoutePublicID, test.activeConvID, test.activeConvPublicID, got, test.want)
			}
		})
	}
}

func TestShouldWarnPendingRootRoute(t *testing.T) {
	tests := []struct {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldWarnPendingRootRoute(test.threadRoutePublicID, test.activeConvID, test.activeConvPublicID); got != test.want {
				t.Fatalf("shouldWarnPendingRootRoute(%q, %d, %q) = %v, want %v", test.threadRoutePublicID, test.activeConvID, test.activeConvPublicID, got, test.want)
			}
		})
	}
}

func TestShouldResolveConversationRoute(t *testing.T) {
	tests := []struct {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldResolveConversationRoute(test.threadRoutePublicID, test.activeConvPublicID); got != test.want {
				t.Fatalf("shouldResolveConversationRoute(%q, %q) = %v, want %v", test.threadRoutePublicID, test.activeConvPublicID, got, test.want)
			}
		})
	}
}

func TestShouldNormalizeActiveConversationRoute(t *testing.T) {
	tests := []struct {
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldNormalizeActiveConversationRoute(test.threadRoutePublicID, test.activeConvPublicID); got != test.want {
				t.Fatalf("shouldNormalizeActiveConversationRoute(%q, %q) = %v, want %v", test.threadRoutePublicID, test.activeConvPublicID, got, test.want)
			}
		})
	}
}

func TestThreadRoutePublicIDFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/app/thread/thread-a", want: "thread-a"},
		{path: "/app/thread/thread-a/canvas/artifact-1", want: "thread-a"},
		{path: "/app/thread/thread-a?panel=settings-intelligence", want: "thread-a"},
		{path: "/app", want: ""},
		{path: "/", want: ""},
	}

	for _, test := range tests {
		if got := threadRoutePublicIDFromPath(test.path); got != test.want {
			t.Fatalf("threadRoutePublicIDFromPath(%q) = %q, want %q", test.path, got, test.want)
		}
	}
}
