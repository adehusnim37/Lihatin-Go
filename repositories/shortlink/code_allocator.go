package shortlink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/adehusnim37/lihatin-go/internal/pkg/config"
	shortlinkmodel "github.com/adehusnim37/lihatin-go/models/shortlink"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const permutationRounds = 8

var (
	errShortCodePermutationSecretMissing = errors.New("short code permutation secret is missing")
	errShortCodeSpaceExhausted           = errors.New("short code space is exhausted")
)

type shortCodeAllocatorSession struct {
	tx           *gorm.DB
	key          []byte
	current      *shortlinkmodel.ShortCodeAllocator
	originalNext uint64
}

func loadShortCodePermutationKey() ([]byte, error) {
	secret := strings.TrimSpace(config.GetEnvOrDefault(config.EnvShortCodePermutationSecret, ""))
	if secret == "" {
		return nil, errShortCodePermutationSecretMissing
	}

	material := []byte(secret)
	if len(material) < 32 {
		return nil, fmt.Errorf("%w: minimum 32 characters", errShortCodePermutationSecretMissing)
	}

	derived := sha256.Sum256(material)
	return derived[:], nil
}

func newShortCodeAllocatorSession(tx *gorm.DB, key []byte) *shortCodeAllocatorSession {
	return &shortCodeAllocatorSession{tx: tx, key: key}
}

func (s *shortCodeAllocatorSession) nextCode() (string, error) {
	if s.current == nil || s.current.NextValue >= s.current.Capacity {
		if err := s.flush(); err != nil {
			return "", err
		}
		if err := s.lockNextAvailableAllocator(); err != nil {
			return "", err
		}
	}

	ordinal := s.current.NextValue
	s.current.NextValue++

	permuted, err := permuteShortCodeOrdinal(ordinal, int(s.current.CodeLength), s.key)
	if err != nil {
		return "", err
	}
	return encodeBase62Fixed(permuted, int(s.current.CodeLength))
}

func (s *shortCodeAllocatorSession) lockNextAvailableAllocator() error {
	var allocator shortlinkmodel.ShortCodeAllocator
	err := s.tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("next_value < capacity").
		Order("code_length ASC").
		First(&allocator).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errShortCodeSpaceExhausted
	}
	if err != nil {
		return fmt.Errorf("lock short code allocator: %w", err)
	}

	s.current = &allocator
	s.originalNext = allocator.NextValue
	return nil
}

func (s *shortCodeAllocatorSession) flush() error {
	if s.current == nil || s.current.NextValue == s.originalNext {
		s.current = nil
		return nil
	}

	result := s.tx.Model(&shortlinkmodel.ShortCodeAllocator{}).
		Where("code_length = ? AND next_value = ?", s.current.CodeLength, s.originalNext).
		Update("next_value", s.current.NextValue)
	if result.Error != nil {
		return fmt.Errorf("advance short code allocator: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return errors.New("short code allocator changed while locked")
	}

	s.current = nil
	return nil
}

// permuteShortCodeOrdinal is a keyed, bijective permutation. The Feistel
// network operates on a 64^length domain; cycle walking maps it exactly onto
// the smaller 62^length Base62 domain without introducing collisions.
func permuteShortCodeOrdinal(ordinal uint64, length int, key []byte) (uint64, error) {
	capacity := shortlinkmodel.ShortCodeCapacity(length)
	if length < shortlinkmodel.MinShortCodeLength || length > shortlinkmodel.MaxShortCodeLength {
		return 0, fmt.Errorf("invalid short code length: %d", length)
	}
	if ordinal >= capacity {
		return 0, fmt.Errorf("short code ordinal %d exceeds capacity %d", ordinal, capacity)
	}
	if len(key) == 0 {
		return 0, errShortCodePermutationSecretMissing
	}

	value := ordinal
	for {
		value = feistelPermute(value, length, key)
		if value < capacity {
			return value, nil
		}
	}
}

func feistelPermute(value uint64, length int, key []byte) uint64 {
	halfBits := uint(length * 3)
	halfMask := uint64(1)<<halfBits - 1
	left := value >> halfBits
	right := value & halfMask

	var message [10]byte
	message[0] = byte(length)
	for round := byte(0); round < permutationRounds; round++ {
		message[1] = round
		binary.BigEndian.PutUint64(message[2:], right)

		mac := hmac.New(sha256.New, key)
		_, _ = mac.Write(message[:])
		roundValue := binary.BigEndian.Uint64(mac.Sum(nil)[:8]) & halfMask
		left, right = right, left^roundValue
	}

	return left<<halfBits | right
}

func encodeBase62Fixed(value uint64, length int) (string, error) {
	if length < 1 || value >= shortlinkmodel.ShortCodeCapacity(length) {
		return "", fmt.Errorf("value %d does not fit in %d Base62 characters", value, length)
	}

	code := make([]byte, length)
	base := uint64(len(shortlinkmodel.ShortCodeAlphabet))
	for i := length - 1; i >= 0; i-- {
		code[i] = shortlinkmodel.ShortCodeAlphabet[value%base]
		value /= base
	}
	return string(code), nil
}
