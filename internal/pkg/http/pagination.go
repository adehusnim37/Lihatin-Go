package http

import (
	"strconv"

	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func PaginateValidate(pageStr, limitStr, sort, orderBy string, role Role) (page, limit int, outSort, outOrder string, errs map[string]string) {
	errs = map[string]string{}

	// defaults
	if pageStr == "" {
		pageStr = "1"
	}
	if limitStr == "" {
		limitStr = "10"
	}
	if sort == "" {
		sort = "created_at"
	}
	if orderBy == "" {
		orderBy = "desc"
	}

	// page
	p, err := strconv.Atoi(pageStr)
	if err != nil || p < 1 {
		logger.Logger.Warn("Invalid page parameter", "page", pageStr)
		errs["page"] = "Page must be a positive integer"
	}

	// limit
	l, err := strconv.Atoi(limitStr)
	if err != nil || l < 1 || l > 100 {
		logger.Logger.Warn("Invalid limit parameter", "limit", limitStr)
		errs["limit"] = "Limit must be between 1 and 100"
	}

	// whitelist sort
	validSorts := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"short_code": true,
		"title":      true,
	}
	if role == RoleAdmin {
		validSorts["original_url"] = true
		validSorts["is_active"] = true
		validSorts["user_id"] = true
	}
	if !validSorts[sort] {
		opts := "created_at, updated_at, short_code, title"
		if role == RoleAdmin {
			opts += ", original_url, is_active, user_id"
		}
		errs["sort"] = "Sort must be one of: " + opts
	}

	// order_by
	if orderBy != "asc" && orderBy != "desc" {
		errs["order_by"] = "Order by must be either 'asc' or 'desc'"
	}

	if len(errs) > 0 {
		return 0, 0, "", "", errs
	}
	return p, l, sort, orderBy, nil
}
