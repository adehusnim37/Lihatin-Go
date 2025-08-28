package repositories

import (
	"database/sql"

	"github.com/adehusnim37/lihatin-go/models/logging"
	"github.com/google/uuid"
)

// LoggerRepository handles database operations for user activity logs
type LoggerRepository struct {
	DB *sql.DB
}

// NewLoggerRepository creates a new logger repository
func NewLoggerRepository(db *sql.DB) *LoggerRepository {
	return &LoggerRepository{
		DB: db,
	}
}

// CreateLog inserts a new activity log into the database
func (r *LoggerRepository) CreateLog(log *logging.ActivityLog) error {
	// Generate a unique ID for the log entry if not provided
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	
	// SQL query to insert log
	query := `
	INSERT INTO ActivityLog (
		id, level, message, username, timestamp, ipaddress,
		useragent, browserinfo, action, route, method, 
		statuscode, requestbody, queryparams, routeparams, 
		contextlocals, responsetime, createdat, updatedat
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.DB.Exec(
		query,
		log.ID, log.Level, log.Message, log.Username, log.Timestamp,
		log.IPAddress, log.UserAgent, log.BrowserInfo, log.Action,
		log.Route, log.Method, log.StatusCode, log.RequestBody,
		log.QueryParams, log.RouteParams, log.ContextLocals, log.ResponseTime,
		log.CreatedAt, log.UpdatedAt,
	)

	return err
}

// GetLogsByUsername retrieves all logs for a specific username
func (r *LoggerRepository) GetLogsByUsername(username string) ([]logging.ActivityLog, error) {
	query := `
	SELECT id, level, message, username, timestamp, ipaddress, 
	       useragent, browserinfo, action, route, method, statuscode,
	       requestbody, queryparams, routeparams, contextlocals, responsetime,
	       createdat, updatedat, deletedat
	FROM ActivityLog
	WHERE username = ? AND deletedat IS NULL
	ORDER BY createdat DESC
	`

	rows, err := r.DB.Query(query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []logging.ActivityLog
	for rows.Next() {
		var log logging.ActivityLog
		var deletedAt sql.NullTime
		var createdAt, updatedAt sql.NullTime
		var requestBody, queryParams, routeParams, contextLocals sql.NullString
		var responseTime sql.NullInt64

		err := rows.Scan(
			&log.ID, &log.Level, &log.Message, &log.Username, &log.Timestamp,
			&log.IPAddress, &log.UserAgent, &log.BrowserInfo, &log.Action,
			&log.Route, &log.Method, &log.StatusCode,
			&requestBody, &queryParams, &routeParams, &contextLocals, &responseTime,
			&createdAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			return nil, err
		}

		// Handle nullable fields
		if requestBody.Valid {
			log.RequestBody = requestBody.String
		}
		if queryParams.Valid {
			log.QueryParams = queryParams.String
		}
		if routeParams.Valid {
			log.RouteParams = routeParams.String
		}
		if contextLocals.Valid {
			log.ContextLocals = contextLocals.String
		}
		if responseTime.Valid {
			log.ResponseTime = responseTime.Int64
		}
		if deletedAt.Valid {
			deletedTime := deletedAt.Time
			log.DeletedAt = &deletedTime
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetAllLogs retrieves all logs from the database
func (r *LoggerRepository) GetAllLogs() ([]logging.ActivityLog, error) {
	query := `
	SELECT id, level, message, username, timestamp, ipaddress, 
	       useragent, browserinfo, action, route, method, statuscode,
	       requestbody, queryparams, routeparams, contextlocals, responsetime,
	       createdat, updatedat, deletedat
	FROM ActivityLog
	WHERE deletedat IS NULL
	ORDER BY createdat DESC
	`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []logging.ActivityLog
	for rows.Next() {
		var log logging.ActivityLog
		var deletedAt sql.NullTime
		var createdAt, updatedAt sql.NullTime
		var requestBody, queryParams, routeParams, contextLocals sql.NullString
		var responseTime sql.NullInt64

		err := rows.Scan(
			&log.ID, &log.Level, &log.Message, &log.Username, &log.Timestamp,
			&log.IPAddress, &log.UserAgent, &log.BrowserInfo, &log.Action,
			&log.Route, &log.Method, &log.StatusCode,
			&requestBody, &queryParams, &routeParams, &contextLocals, &responseTime,
			&createdAt, &updatedAt, &deletedAt,
		)
		if err != nil {
			return nil, err
		}

		// Handle nullable fields
		if requestBody.Valid {
			log.RequestBody = requestBody.String
		}
		if queryParams.Valid {
			log.QueryParams = queryParams.String
		}
		if routeParams.Valid {
			log.RouteParams = routeParams.String
		}
		if contextLocals.Valid {
			log.ContextLocals = contextLocals.String
		}
		if responseTime.Valid {
			log.ResponseTime = responseTime.Int64
		}
		if deletedAt.Valid {
			deletedTime := deletedAt.Time
			log.DeletedAt = &deletedTime
		}

		logs = append(logs, log)
	}

	return logs, nil
}
