package utils

import "errors"

var (
	// Repository errors
	ErrShortLinkNotFound       = errors.New("short link not found")
	ErrShortLinkExpired        = errors.New("short link has expired")
	ErrShortLinkInactive       = errors.New("short link is inactive")
	ErrShortLinkUnauthorized   = errors.New("unauthorized to access this link")
	ErrDuplicateShortCode      = errors.New("short code already exists")
	ErrInvalidOriginalURL      = errors.New("invalid original URL")
	ErrShortLinkAlreadyDeleted = errors.New("short link has already been deleted")
	ErrEmptyCodesList          = errors.New("codes list cannot be empty")
	ErrSomeShortLinksNotFound  = errors.New("some short links not found")

	// Passcode errors
	ErrPasscodeRequired  = errors.New("passcode required")
	ErrPasscodeIncorrect = errors.New("incorrect passcode")

	// Limit errors
	ErrClickLimitReached = errors.New("click limit reached")
)
