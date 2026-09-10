package shortlink

import (
	"testing"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	shortlinkmodel "github.com/adehusnim37/lihatin-go/models/shortlink"
)

func TestShortCodeCapacities(t *testing.T) {
	expected := map[int]uint64{
		2: 3_844,
		3: 238_328,
		4: 14_776_336,
		5: 916_132_832,
		6: 56_800_235_584,
		7: 3_521_614_606_208,
		8: 218_340_105_584_896,
	}

	for length, want := range expected {
		if got := shortlinkmodel.ShortCodeCapacity(length); got != want {
			t.Fatalf("capacity(%d) = %d, want %d", length, got, want)
		}
	}
}

func TestPermutationCoversEntireTwoCharacterSpaceWithoutCollision(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	capacity := shortlinkmodel.ShortCodeCapacity(2)
	seen := make(map[string]struct{}, capacity)

	for ordinal := uint64(0); ordinal < capacity; ordinal++ {
		permuted, err := permuteShortCodeOrdinal(ordinal, 2, key)
		if err != nil {
			t.Fatalf("permute ordinal %d: %v", ordinal, err)
		}
		code, err := encodeBase62Fixed(permuted, 2)
		if err != nil {
			t.Fatalf("encode ordinal %d: %v", ordinal, err)
		}
		if _, exists := seen[code]; exists {
			t.Fatalf("duplicate code %q generated for ordinal %d", code, ordinal)
		}
		seen[code] = struct{}{}
	}

	if uint64(len(seen)) != capacity {
		t.Fatalf("generated %d distinct codes, want %d", len(seen), capacity)
	}
}

func TestPermutationIsDeterministicAndKeyed(t *testing.T) {
	keyA := []byte("0123456789abcdef0123456789abcdef")
	keyB := []byte("fedcba9876543210fedcba9876543210")

	first, err := permuteShortCodeOrdinal(42, 4, keyA)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := permuteShortCodeOrdinal(42, 4, keyA)
	if err != nil {
		t.Fatal(err)
	}
	otherKey, err := permuteShortCodeOrdinal(42, 4, keyB)
	if err != nil {
		t.Fatal(err)
	}

	if first != repeated {
		t.Fatalf("same key produced %d then %d", first, repeated)
	}
	if first == otherKey {
		t.Fatalf("different keys unexpectedly produced the same value %d", first)
	}
}

func TestLoadShortCodePermutationKeyRequiresConfiguredSecret(t *testing.T) {
	t.Setenv(config.EnvShortCodePermutationSecret, "")
	if _, err := loadShortCodePermutationKey(); err == nil {
		t.Fatal("expected missing secret error")
	}

	t.Setenv(config.EnvShortCodePermutationSecret, "0123456789abcdef0123456789abcdef")
	key, err := loadShortCodePermutationKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 32 {
		t.Fatalf("derived key length = %d, want 32", len(key))
	}
}
