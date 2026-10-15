package auth

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSecurityGrantOneUseAndSessionBound(t *testing.T) {
	client := startOTPTestRedis(t)
	ctx := context.Background()
	token := strings.Repeat("a", 64)
	if err := client.Set(ctx, "security_grant:"+token, "alice:session-1", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if ConsumeSecurityGrant(ctx, client, "alice", "session-2", token) {
		t.Fatal("cross-session grant accepted")
	}
	if err := client.Set(ctx, "security_grant:"+token, "alice:session-1", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if !ConsumeSecurityGrant(ctx, client, "alice", "session-1", token) || ConsumeSecurityGrant(ctx, client, "alice", "session-1", token) {
		t.Fatal("grant was not single use")
	}
}
func TestPendingTOTPChallengeConsumedOnceConcurrently(t *testing.T) {
	startOTPTestRedis(t)
	ctx := context.Background()
	token, err := GeneratePendingAuthToken(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ConsumePendingAuthToken(ctx, token) == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("pending login consumed %d times", successes.Load())
	}
}
