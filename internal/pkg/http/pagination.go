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

// PaginateValidateLogger validates pagination parameters for logger endpoints
// Supports logger-specific sort fields like timestamp, level, action, status_code
func PaginateValidateLogger(pageStr, limitStr, sort, orderBy string) (page, limit int, outSort, outOrder string, errs map[string]string) {
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

	// limit - enforce max 100 to prevent excessive load
	l, err := strconv.Atoi(limitStr)
	if err != nil || l < 1 || l > 100 {
		logger.Logger.Warn("Invalid limit parameter", "limit", limitStr)
		errs["limit"] = "Limit must be between 1 and 100"
	}

	// whitelist sort fields for logger (prevent SQL injection)
	validSorts := map[string]bool{
		"created_at":    true,
		"updated_at":    true,
		"timestamp":     true,
		"level":         true,
		"action":        true,
		"status_code":   true,
		"response_time": true,
		"username":      true,
		"method":        true,
		"route":         true,
	}
	if !validSorts[sort] {
		errs["sort"] = "Sort must be one of: created_at, updated_at, timestamp, level, action, status_code, response_time, username, method, route"
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

// PaginateValidateLoginAttempts validates pagination parameters for login attempts endpoints
// Supports login attempts specific sort fields and higher limit for security monitoring
func PaginateValidateLoginAttempts(pageStr, limitStr, sort, orderBy string, isAdmin bool) (page, limit int, outSort, outOrder string, errs map[string]string) {
	errs = map[string]string{}

	// defaults
	if pageStr == "" {
		pageStr = "1"
	}
	if limitStr == "" {
		limitStr = "50" // Higher default for security monitoring
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

	// limit - higher max for admins monitoring security
	maxLimit := 100
	if isAdmin {
		maxLimit = 500
	}
	l, err := strconv.Atoi(limitStr)
	if err != nil || l < 1 || l > maxLimit {
		logger.Logger.Warn("Invalid limit parameter", "limit", limitStr)
		errs["limit"] = "Limit must be between 1 and " + strconv.Itoa(maxLimit)
	}

	// whitelist sort fields for login attempts
	validSorts := map[string]bool{
		"created_at":        true,
		"updated_at":        true,
		"email_or_username": true,
		"ip_address":        true,
		"success":           true,
	}
	if !validSorts[sort] {
		errs["sort"] = "Sort must be one of: created_at, updated_at, email_or_username, ip_address, success"
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

// ParseLoginAttemptsFilters parses and validates filter parameters for login attempts
// Returns a map of filters ready to use in repository queries
func ParseLoginAttemptsFilters(queryParams map[string]string, isAdmin bool, username string) (filters map[string]interface{}, errs map[string]string) {
	filters = make(map[string]interface{})
	errs = make(map[string]string)

	// Success filter - validate boolean
	if successStr, exists := queryParams["success"]; exists && successStr != "" {
		filters["success_str"] = successStr
	}

	// ID filter - validate UUID format (optional, not strict)
	if id, exists := queryParams["id"]; exists && id != "" {
		filters["id"] = id
	}

	// Search filter - partial match on email_or_username
	if search, exists := queryParams["search"]; exists && search != "" {
		if len(search) < 2 {
			errs["search"] = "Search query must be at least 2 characters"
		} else {
			filters["search"] = search
		}
	}

	// IP Address filter - validate IP format (basic)
	if ipAddress, exists := queryParams["ip_address"]; exists && ipAddress != "" {
		// Basic validation: should contain dots or colons (IPv4/IPv6)
		if len(ipAddress) < 7 || len(ipAddress) > 45 {
			errs["ip_address"] = "Invalid IP address format"
		} else {
			filters["ip_address"] = ipAddress
		}
	}

	// Date range filters - validate RFC3339 format
	if fromDate, exists := queryParams["from_date"]; exists && fromDate != "" {
		filters["from_date"] = fromDate
	}

	if toDate, exists := queryParams["to_date"]; exists && toDate != "" {
		filters["to_date"] = toDate
	}

	// Authorization: non-admin users can only see their own attempts
	if !isAdmin && username != "" {
		filters["email_or_username"] = username
	}

	return filters, errs
}
