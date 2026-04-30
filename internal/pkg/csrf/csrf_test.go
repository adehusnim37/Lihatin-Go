package csrf

import "testing"

func TestShouldSkipRouteUsesExactAndStructuredMatching(t *testing.T) {
	rules := []SkipRule{
		{Method: "POST", Path: "/v1/auth/login"},
		{Method: "POST", Path: "/v1/support/tickets/:ticketCode/messages"},
	}

	tests := []struct {
		name        string
		method      string
		requestPath string
		fullPath    string
		want        bool
	}{
		{
			name:        "exact path matches",
			method:      "POST",
			requestPath: "/v1/auth/login",
			fullPath:    "/v1/auth/login",
			want:        true,
		},
		{
			name:        "method mismatch does not skip",
			method:      "GET",
			requestPath: "/v1/auth/login",
			fullPath:    "/v1/auth/login",
			want:        false,
		},
		{
			name:        "dynamic gin route pattern matches full path",
			method:      "POST",
			requestPath: "/v1/support/tickets/LHTK-123/messages",
			fullPath:    "/v1/support/tickets/:ticketCode/messages",
			want:        true,
		},
		{
			name:        "legacy path list now exact only",
			method:      "POST",
			requestPath: "/v1/support/access/request-otp",
			fullPath:    "/v1/support/access/request-otp",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSkipRoute(tt.method, tt.requestPath, tt.fullPath, rules, []string{"/v1/support/access"})
			if got != tt.want {
				t.Fatalf("shouldSkipRoute() = %v, want %v", got, tt.want)
			}
		})
	}
}
