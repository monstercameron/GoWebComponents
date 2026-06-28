package typedroutesdemo

import "testing"

// TestGeneratedLinksResolve proves the gwc-routes-gen'd constructors build correct URLs
// from typed params — a typo'd or missing param would be a compile error, not a runtime one.
func TestGeneratedLinksResolve(t *testing.T) {
	if got := LinkHome(); got != "/" {
		t.Fatalf("LinkHome() = %q, want /", got)
	}
	if got := LinkUser("42"); got != "/users/42" {
		t.Fatalf("LinkUser(42) = %q, want /users/42", got)
	}
	if got := LinkPost("42", "7"); got != "/users/42/posts/7" {
		t.Fatalf("LinkPost(42,7) = %q, want /users/42/posts/7", got)
	}
}
