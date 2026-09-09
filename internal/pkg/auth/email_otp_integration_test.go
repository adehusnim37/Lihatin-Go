package auth

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func startOTPTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	redisServer, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server is not installed")
	}

	tempDir, err := os.MkdirTemp("/tmp", "otp-redis-")
	if err != nil {
		t.Fatalf("create redis temp directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	socketPath := filepath.Join(tempDir, "redis.sock")
	var serverOutput bytes.Buffer
	cmd := exec.Command(redisServer,
		"--port", "0",
		"--unixsocket", socketPath,
		"--unixsocketperm", "700",
		"--save", "",
		"--appendonly", "no",
	)
	cmd.Stdout = &serverOutput
	cmd.Stderr = &serverOutput
	if err := cmd.Start(); err != nil {
		t.Fatalf("start redis-server: %v", err)
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- cmd.Wait() }()

	client := redis.NewClient(&redis.Options{Network: "unix", Addr: socketPath})
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, statErr := os.Stat(socketPath); statErr == nil {
			if err := client.Ping(context.Background()).Err(); err == nil {
				break
			}
		}
		select {
		case serverErr := <-serverDone:
			t.Skipf("redis-server cannot run in this environment: %v: %s", serverErr, serverOutput.String())
		default:
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatalf("redis-server did not become ready: %s", serverOutput.String())
		}
		time.Sleep(10 * time.Millisecond)
	}

	previousClient := pendingAuthRedisClient
	InitPendingAuthRedis(client)
	t.Cleanup(func() {
		pendingAuthRedisClient = previousClient
		_ = client.Close()
		_ = cmd.Process.Kill()
		select {
		case <-serverDone:
		case <-time.After(time.Second):
		}
	})
	return client
}

func TestEmailOTPChallengeCanOnlyBeConsumedOnceConcurrently(t *testing.T) {
	startOTPTestRedis(t)
	ctx := context.Background()
	token, code, _, err := GenerateEmailOTPChallenge(ctx, EmailOTPPurposeSupportAccess, "user@example.com", "ticket-a")
	if err != nil {
		t.Fatalf("generate challenge: %v", err)
	}

	type result struct {
		challenge *EmailOTPChallenge
		err       error
	}
	results := make(chan result, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			ready.Done()
			<-start
			challenge, verifyErr := VerifyEmailOTPChallenge(ctx, token, code, EmailOTPPurposeSupportAccess)
			results <- result{challenge: challenge, err: verifyErr}
		}()
	}
	ready.Wait()
	close(start)

	successes := 0
	failures := 0
	for i := 0; i < 2; i++ {
		result := <-results
		if result.err == nil && result.challenge != nil {
			successes++
			continue
		}
		if errors.Is(result.err, ErrEmailOTPChallengeNotFound) {
			failures++
			continue
		}
		t.Fatalf("unexpected concurrent verification result: challenge=%#v err=%v", result.challenge, result.err)
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("successes=%d failures=%d, want exactly one of each", successes, failures)
	}
}

func TestSupportAccessTokenBulkRevocationIsTicketScoped(t *testing.T) {
	startOTPTestRedis(t)
	ctx := context.Background()
	tokenA1, _, err := CreateSupportAccessToken(ctx, "ticket-a", "LHTK-A", "User@Example.com", "version-a")
	if err != nil {
		t.Fatalf("create first ticket A token: %v", err)
	}
	tokenA2, _, err := CreateSupportAccessToken(ctx, "ticket-a", "LHTK-A", "user@example.com", "version-a")
	if err != nil {
		t.Fatalf("create second ticket A token: %v", err)
	}
	tokenB, _, err := CreateSupportAccessToken(ctx, "ticket-b", "LHTK-B", "other@example.com", "version-b")
	if err != nil {
		t.Fatalf("create ticket B token: %v", err)
	}

	if err := RevokeSupportAccessTokensForTicket(ctx, "ticket-a"); err != nil {
		t.Fatalf("revoke ticket A: %v", err)
	}
	for _, token := range []string{tokenA1, tokenA2} {
		if _, err := GetSupportAccessToken(ctx, token); !errors.Is(err, ErrSupportAccessTokenInvalid) {
			t.Fatalf("revoked ticket A token remained valid: %v", err)
		}
	}
	payload, err := GetSupportAccessToken(ctx, tokenB)
	if err != nil || payload == nil || payload.TicketID != "ticket-b" {
		t.Fatalf("ticket B token was affected by ticket A revocation: payload=%#v err=%v", payload, err)
	}
}

func TestConcurrentInvalidOTPAttemptsCannotExceedLimit(t *testing.T) {
	startOTPTestRedis(t)
	ctx := context.Background()
	token, generatedCode, _, err := GenerateEmailOTPChallenge(ctx, EmailOTPPurposeSupportAccess, "user@example.com", "ticket-a")
	if err != nil {
		t.Fatalf("generate challenge: %v", err)
	}
	invalidCode := "000000"
	if generatedCode == invalidCode {
		invalidCode = "111111"
	}

	const concurrentAttempts = 10
	errorsSeen := make(chan error, concurrentAttempts)
	var ready sync.WaitGroup
	ready.Add(concurrentAttempts)
	start := make(chan struct{})
	for i := 0; i < concurrentAttempts; i++ {
		go func() {
			ready.Done()
			<-start
			_, verifyErr := VerifyEmailOTPChallenge(ctx, token, invalidCode, EmailOTPPurposeSupportAccess)
			errorsSeen <- verifyErr
		}()
	}
	ready.Wait()
	close(start)

	invalidCount := 0
	exceededCount := 0
	for i := 0; i < concurrentAttempts; i++ {
		verifyErr := <-errorsSeen
		var invalidErr *EmailOTPInvalidCodeError
		switch {
		case errors.As(verifyErr, &invalidErr):
			invalidCount++
		case errors.Is(verifyErr, ErrEmailOTPAttemptsExceeded):
			exceededCount++
		case errors.Is(verifyErr, ErrEmailOTPChallengeNotFound):
			// Requests serialized after the fifth attempt see a consumed token.
		default:
			t.Fatalf("unexpected verification error: %v", verifyErr)
		}
	}
	if invalidCount != EmailOTPMaxAttempts-1 || exceededCount != 1 {
		t.Fatalf("invalid=%d exceeded=%d, want invalid=%d exceeded=1", invalidCount, exceededCount, EmailOTPMaxAttempts-1)
	}
	if _, err := GetEmailOTPChallenge(ctx, token); !errors.Is(err, ErrEmailOTPChallengeNotFound) {
		t.Fatalf("challenge remained after maximum attempts: %v", err)
	}
}
