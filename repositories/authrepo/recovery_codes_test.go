package authrepo

import (
	"context"
	"encoding/json"
	pkgauth "github.com/adehusnim37/lihatin-go/internal/pkg/auth"
	"github.com/adehusnim37/lihatin-go/models/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func TestConsumeRecoveryCodeOnceAndAccountScoped(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "recovery.db")), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&user.AuthMethod{}); err != nil {
		t.Fatal(err)
	}
	hashes, _ := json.Marshal(pkgauth.HashRecoveryCodes([]string{"ABCD-1234", "EFGH-5678"}))
	for _, id := range []string{"alice", "bob"} {
		if err = db.Create(&user.AuthMethod{ID: id, UserAuthID: id, Type: user.AuthMethodTypeTOTP, IsEnabled: true, IsVerified: true, RecoveryCodes: hashes}).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewAuthMethodRepository(db)
	for _, item := range []struct {
		owner, code string
		want        bool
	}{{"alice", "XXXX-9999", false}, {"alice", "ABCD1234", true}, {"alice", "ABCD-1234", false}, {"bob", "ABCD-1234", true}, {"alice", "EFGH-5678", true}} {
		got, err := repo.ConsumeRecoveryCode(context.Background(), item.owner, item.code)
		if err != nil || got != item.want {
			t.Fatalf("%s/%s got=%v error=%v want=%v", item.owner, item.code, got, err, item.want)
		}
	}
}
