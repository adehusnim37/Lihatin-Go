package apikeyrepo

import (
	"strings"
	"sync"
	"testing"

	"github.com/adehusnim37/lihatin-go/dto"
	"github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	apperrors "github.com/adehusnim37/lihatin-go/internal/pkg/errors"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testRepository(t *testing.T) *APIKeyRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, schema := range []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY, role TEXT, deleted_at DATETIME)`,
		`CREATE TABLE user_auth (id TEXT PRIMARY KEY, user_id TEXT, is_email_verified BOOLEAN, account_status TEXT)`,
		`CREATE TABLE auth_methods (id TEXT PRIMARY KEY, user_auth_id TEXT)`,
		`CREATE TABLE user_premium_access (id TEXT PRIMARY KEY, user_id TEXT)`,
		`CREATE TABLE api_keys (id TEXT PRIMARY KEY, user_id TEXT, name TEXT, key TEXT, key_hash TEXT, usage_count INTEGER DEFAULT 0, limit_usage INTEGER, last_ip_used TEXT, allowed_ips TEXT, blocked_ips TEXT, last_used_at DATETIME, expires_at DATETIME, is_active BOOLEAN, permissions TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
	} {
		if err := db.Exec(schema).Error; err != nil {
			t.Fatal(err)
		}
	}
	return NewAPIKeyRepository(db)
}

func TestAuthenticateDoesNotSpendUsageUntilReserved(t *testing.T) {
	repo := testRepository(t)
	for _, query := range []string{
		`INSERT INTO users (id, role) VALUES ('owner', 'user')`,
		`INSERT INTO user_auth (id, user_id, is_email_verified, account_status) VALUES ('auth', 'owner', 1, 'active')`,
	} {
		if err := repo.db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	created, err := repo.CreateAPIKey("owner", dto.CreateAPIKeyRequest{Name: "test key"})
	if err != nil {
		t.Fatal(err)
	}
	account, key, err := repo.AuthenticateAPIKey(created.Key, "127.0.0.1")
	if err != nil || account.ID != "owner" || key.ID != created.ID {
		t.Fatalf("authentication failed: account=%+v key=%+v error=%v", account, key, err)
	}
	var count int64
	if err := repo.db.Raw(`SELECT usage_count FROM api_keys WHERE id = ?`, key.ID).Scan(&count).Error; err != nil || count != 0 {
		t.Fatalf("authentication spent quota: count=%d error=%v", count, err)
	}
	if err := repo.ReserveAPIKeyUsage(key, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := repo.db.Raw(`SELECT usage_count FROM api_keys WHERE id = ?`, key.ID).Scan(&count).Error; err != nil || count != 1 {
		t.Fatalf("reservation count=%d error=%v", count, err)
	}
	parts := strings.SplitN(created.Key, ".", 2)
	if len(parts) != 2 || !auth.ValidateAPISecretKey(parts[1], key.KeyHash) {
		t.Fatal("new key secret was not verifiable")
	}
}

func TestReserveAPIKeyUsageIsAtomicAtLimit(t *testing.T) {
	repo := testRepository(t)
	if err := repo.db.Exec(`INSERT INTO api_keys (id, user_id, name, key, key_hash, usage_count, limit_usage, is_active) VALUES ('one', 'user', 'key', 'key-id', 'hash', 0, 1, 1)`).Error; err != nil {
		t.Fatal(err)
	}
	const attempts = 12
	results := make(chan error, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := user.APIKey{ID: "one", Key: "key-id", KeyHash: "hash"}
			results <- repo.reserveAPIKeyUsage(&key, "key-id", "127.0.0.1")
		}()
	}
	wg.Wait()
	close(results)
	success, limited := 0, 0
	for err := range results {
		switch err {
		case nil:
			success++
		case apperrors.ErrAPIKeyRateLimitExceeded:
			limited++
		default:
			t.Fatalf("unexpected result: %v", err)
		}
	}
	if success != 1 || limited != attempts-1 {
		t.Fatalf("success=%d limited=%d, want 1/%d", success, limited, attempts-1)
	}
	var count int64
	if err := repo.db.Raw(`SELECT usage_count FROM api_keys WHERE id = 'one'`).Scan(&count).Error; err != nil || count != 1 {
		t.Fatalf("stored usage_count=%d, error=%v", count, err)
	}
}

func TestGetAPIKeyByIDUsesRequestedIDAndOwnership(t *testing.T) {
	repo := testRepository(t)
	for _, query := range []string{
		`INSERT INTO users (id, role) VALUES ('owner', 'user'), ('other', 'user')`,
		`INSERT INTO api_keys (id, user_id, name, key, key_hash, is_active) VALUES ('first', 'owner', 'first', 'key-1', 'hash', 1), ('second', 'owner', 'second', 'key-2', 'hash', 1), ('foreign', 'other', 'foreign', 'key-3', 'hash', 1)`,
	} {
		if err := repo.db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.GetAPIKeyByID(dto.APIKeyIDRequest{ID: "second"}, "owner")
	if err != nil || got.ID != "second" {
		t.Fatalf("requested second, got %+v, error=%v", got, err)
	}
	if _, err := repo.GetAPIKeyByID(dto.APIKeyIDRequest{ID: "foreign"}, "owner"); err != apperrors.ErrAPIKeyNotFound {
		t.Fatalf("foreign key lookup: got %v", err)
	}
}

func TestUpdateAPIKeyCanClearOptionalTotalUseCap(t *testing.T) {
	repo := testRepository(t)
	for _, query := range []string{
		`INSERT INTO users (id, role) VALUES ('owner', 'user')`,
		`INSERT INTO api_keys (id, user_id, name, key, key_hash, usage_count, limit_usage, is_active) VALUES ('capped', 'owner', 'key', 'key-id', 'hash', 2, 5, 1)`,
	} {
		if err := repo.db.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	keyID := dto.APIKeyIDRequest{ID: "capped"}
	if _, err := repo.UpdateAPIKey(keyID, "owner", dto.UpdateAPIKeyRequest{ClearLimitUsage: true, LimitUsage: int64Pointer(10)}); err != apperrors.ErrAPIKeyInvalidLimitUsage {
		t.Fatalf("conflicting cap request: %v", err)
	}
	updated, err := repo.UpdateAPIKey(keyID, "owner", dto.UpdateAPIKeyRequest{ClearLimitUsage: true})
	if err != nil || updated.LimitUsage != nil || updated.UsageCount != 2 {
		t.Fatalf("cleared cap: updated=%+v error=%v", updated, err)
	}
}

func int64Pointer(value int64) *int64 { return &value }

func TestCreateAPIKeyStopsAtThreeIncludingCustomPrefix(t *testing.T) {
	repo := testRepository(t)
	if err := repo.db.Exec(`INSERT INTO user_auth (id, user_id, is_email_verified, account_status) VALUES ('auth', 'owner', 1, 'active')`).Error; err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first", "second", "third"} {
		if _, err := repo.CreateAPIKey("owner", dto.CreateAPIKeyRequest{Name: name}); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	if _, err := repo.CreateAPIKeyWithCustomPrefix("owner", "fourth", "custom", nil, nil); err != apperrors.ErrAPIKeyLimitReached {
		t.Fatalf("fourth key: got %v", err)
	}
}
