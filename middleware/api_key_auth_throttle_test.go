package middleware

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func testAPIKeyAuthRedis(t *testing.T) *redis.Client {
	t.Helper()
	server, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server is not installed")
	}
	// macOS/Valkey limits Unix socket paths to roughly 104 bytes. Go's
	// t.TempDir path can exceed that for long subtest names.
	dir, err := os.MkdirTemp("/tmp", "ak-redis-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "redis.sock")
	cmd := exec.Command(server, "--port", "0", "--unixsocket", socket, "--unixsocketperm", "700", "--save", "", "--appendonly", "no")
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatalf("cannot start redis-server: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	client := redis.NewClient(&redis.Options{Network: "unix", Addr: socket})
	t.Cleanup(func() {
		_ = client.Close()
		_ = cmd.Process.Kill()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socket); err == nil && client.Ping(context.Background()).Err() == nil {
			return client
		}
		select {
		case err := <-done:
			t.Fatalf("redis-server exited: %v: %s", err, output.String())
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("redis-server did not become ready: %s", output.String())
	return nil
}

func TestAPIKeyAuthThrottleLimitsFailuresWithoutBlockingAnotherKey(t *testing.T) {
	throttle := &apiKeyAuthThrottle{redis: testAPIKeyAuthRedis(t)}
	ctx := context.Background()
	ip := "192.0.2.10"
	badKey := "sk_lh_first.secret"
	otherKey := "sk_lh_second.secret"
	for i := 0; i < apiKeyAuthFailuresPerWindow; i++ {
		allowed, _, err := throttle.check(ctx, ip, badKey)
		if err != nil || !allowed {
			t.Fatalf("attempt %d rejected early: allowed=%v err=%v", i+1, allowed, err)
		}
		if err := throttle.recordFailure(ctx, ip, badKey); err != nil {
			t.Fatal(err)
		}
	}
	allowed, retryAfter, err := throttle.check(ctx, ip, badKey)
	if err != nil || allowed || retryAfter <= 0 {
		t.Fatalf("bad key should be blocked with retry delay: allowed=%v retry=%v err=%v", allowed, retryAfter, err)
	}
	allowed, _, err = throttle.check(ctx, ip, otherKey)
	if err != nil || !allowed {
		t.Fatalf("other key at same IP should work: allowed=%v err=%v", allowed, err)
	}
	allowed, _, err = throttle.check(ctx, "198.51.100.2", badKey)
	if err != nil || !allowed {
		t.Fatalf("same key at another IP should work: allowed=%v err=%v", allowed, err)
	}
}

func TestAPIKeyAuthRequestCounterIsAtomicAndExpires(t *testing.T) {
	throttle := &apiKeyAuthThrottle{redis: testAPIKeyAuthRedis(t)}
	ctx := context.Background()
	key := apiKeyAuthCounterKey("requests", "192.0.2.11", "")
	results := make(chan int64, apiKeyAuthRequestsPerMinute+1)
	var wg sync.WaitGroup
	for i := 0; i < apiKeyAuthRequestsPerMinute+1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, err := throttle.increment(ctx, key, time.Minute)
			if err != nil {
				t.Errorf("increment: %v", err)
			}
			results <- count
		}()
	}
	wg.Wait()
	close(results)
	seen := make(map[int64]bool)
	for count := range results {
		seen[count] = true
	}
	if len(seen) != apiKeyAuthRequestsPerMinute+1 || !seen[1] || !seen[apiKeyAuthRequestsPerMinute+1] {
		t.Fatalf("counter lost concurrent increments: %d distinct values", len(seen))
	}
	if ttl := throttle.retryAfter(ctx, key); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("unexpected counter TTL: %v", ttl)
	}
	allowed, _, err := throttle.check(ctx, "192.0.2.11", "sk_lh_key.secret")
	if err != nil || allowed {
		t.Fatalf("IP ceiling should block: allowed=%v err=%v", allowed, err)
	}
}
