package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
)

// These simulations exercise the Redis admission gate without sending attack
// traffic to a running API or spending CPU on repeated PBKDF2 checks.
func TestAPIKeyAuthAttackSimulation(t *testing.T) {
	ctx := context.Background()

	t.Run("same IP and key ID", func(t *testing.T) {
		throttle := &apiKeyAuthThrottle{redis: testAPIKeyAuthRedis(t)}
		ip, key := "192.0.2.10", "sk_lh_target.secret"
		var admitted, blocked int
		for i := 0; i < 15; i++ {
			allowed, _, err := throttle.check(ctx, ip, key)
			if err != nil {
				t.Fatal(err)
			}
			if allowed {
				admitted++
				if err := throttle.recordFailure(ctx, ip, key); err != nil {
					t.Fatal(err)
				}
			} else {
				blocked++
			}
		}
		if admitted != 10 || blocked != 5 {
			t.Fatalf("admitted=%d blocked=%d; want 10 and 5", admitted, blocked)
		}
		t.Logf("15 sequential invalid attempts: %d reach authentication, %d rejected beforehand", admitted, blocked)
	})

	t.Run("rotating key IDs from one IP", func(t *testing.T) {
		throttle := &apiKeyAuthThrottle{redis: testAPIKeyAuthRedis(t)}
		ip := "192.0.2.11"
		var admitted, blocked int
		for i := 0; i < apiKeyAuthRequestsPerMinute+5; i++ {
			key := fmt.Sprintf("sk_lh_key%05d.secret", i)
			allowed, _, err := throttle.check(ctx, ip, key)
			if err != nil {
				t.Fatal(err)
			}
			if allowed {
				admitted++
				if err := throttle.recordFailure(ctx, ip, key); err != nil {
					t.Fatal(err)
				}
			} else {
				blocked++
			}
		}
		if admitted != apiKeyAuthRequestsPerMinute || blocked != 5 {
			t.Fatalf("admitted=%d blocked=%d; want %d and 5", admitted, blocked, apiKeyAuthRequestsPerMinute)
		}
		t.Logf("125 different key IDs: %d reach authentication, %d rejected by IP ceiling", admitted, blocked)
	})

	t.Run("two source IPs", func(t *testing.T) {
		throttle := &apiKeyAuthThrottle{redis: testAPIKeyAuthRedis(t)}
		for _, ip := range []string{"192.0.2.12", "198.51.100.12"} {
			for i := 0; i < apiKeyAuthRequestsPerMinute; i++ {
				key := fmt.Sprintf("sk_lh_key%05d.secret", i)
				allowed, _, err := throttle.check(ctx, ip, key)
				if err != nil || !allowed {
					t.Fatalf("ip=%s attempt=%d allowed=%v err=%v", ip, i+1, allowed, err)
				}
				if err := throttle.recordFailure(ctx, ip, key); err != nil {
					t.Fatal(err)
				}
			}
		}
		t.Logf("2 IPs with rotating key IDs: %d invalid attempts reach authentication", 2*apiKeyAuthRequestsPerMinute)
	})

	t.Run("parallel attempts against one key ID", func(t *testing.T) {
		throttle := &apiKeyAuthThrottle{redis: testAPIKeyAuthRedis(t)}
		const attempts = 40
		var admitted atomic.Int64
		var wg sync.WaitGroup
		start := make(chan struct{})
		record := make(chan struct{})
		checked := make(chan struct{}, attempts)
		for i := 0; i < attempts; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				allowed, _, err := throttle.check(ctx, "192.0.2.13", "sk_lh_target.secret")
				if err != nil {
					t.Errorf("check: %v", err)
					checked <- struct{}{}
					return
				}
				if allowed {
					admitted.Add(1)
				}
				checked <- struct{}{}
				<-record // all checks finish before any KDF result is recorded
				if allowed {
					if err := throttle.recordFailure(ctx, "192.0.2.13", "sk_lh_target.secret"); err != nil {
						t.Errorf("record: %v", err)
					}
				}
			}()
		}
		close(start)
		// Wait for all decisions before allowing failures to be recorded.
		for i := 0; i < attempts; i++ {
			<-checked
		}
		close(record)
		wg.Wait()
		if admitted.Load() != attempts {
			t.Fatalf("admitted=%d; want %d", admitted.Load(), attempts)
		}
		t.Logf("%d parallel checks pass before failure counters are updated (race window)", attempts)
	})

	t.Run("Redis unavailable", func(t *testing.T) {
		client := testAPIKeyAuthRedis(t)
		throttle := &apiKeyAuthThrottle{redis: client}
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
		allowed, _, err := throttle.check(ctx, "192.0.2.14", "sk_lh_target.secret")
		if err == nil || allowed {
			t.Fatalf("limiter failed open: allowed=%v err=%v", allowed, err)
		}
		t.Log("Redis outage rejects API key authentication instead of bypassing limits")
	})
}

func TestAPIKeyAuthUntrustedForwardedIPSimulation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	router.GET("/probe", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })
	for _, forged := range []string{"198.51.100.1", "203.0.113.200"} {
		request := httptest.NewRequest(http.MethodGet, "/probe", nil)
		request.RemoteAddr = "192.0.2.15:12345"
		request.Header.Set("X-Forwarded-For", forged)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Body.String() != "192.0.2.15" {
			t.Fatalf("forged X-Forwarded-For %q changed client IP to %q", forged, response.Body.String())
		}
	}
}

func TestAPIKeyAuthHTTPAttackSimulation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	throttle := &apiKeyAuthThrottle{redis: testAPIKeyAuthRedis(t)}
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	router.Use(authRepositoryAPIKeyMiddleware(nil, func() (*apiKeyAuthThrottle, error) { return throttle, nil }))
	router.GET("/api/short", func(c *gin.Context) { c.Status(http.StatusOK) })

	for attempt := 1; attempt <= 12; attempt++ {
		request := httptest.NewRequest(http.MethodGet, "/api/short", nil)
		request.RemoteAddr = "192.0.2.30:12345"
		request.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", attempt))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		want := http.StatusUnauthorized
		if attempt > apiKeyAuthFailuresPerWindow {
			want = http.StatusTooManyRequests
			if response.Header().Get("Retry-After") == "" {
				t.Fatal("429 response lacks Retry-After")
			}
		}
		if response.Code != want {
			t.Fatalf("attempt %d: got %d, want %d", attempt, response.Code, want)
		}
	}

	// Oversized secrets must be rejected before repository lookup or PBKDF2.
	request := httptest.NewRequest(http.MethodGet, "/api/short", nil)
	request.RemoteAddr = "192.0.2.31:12345"
	request.Header.Set("X-API-Key", strings.Repeat("a", maxAPIKeyHeaderLength+1))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("oversized header: got %d, want 400", response.Code)
	}

	if err := throttle.redis.Close(); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodGet, "/api/short", nil)
	request.RemoteAddr = "192.0.2.32:12345"
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("Redis unavailable: got %d, want 503", response.Code)
	}
}
