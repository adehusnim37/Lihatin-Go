# Logger Database Indexing Guide

## Overview

This document provides comprehensive guidance on database indexing strategies for the activity logging system. Proper indexing is critical for maintaining optimal query performance as the log volume grows.

## Current Table Structure

**Table:** `activitylog`

### Existing Indexes

1. **Primary Key:** `id` (string/UUID)
2. **Index:** `api_key`
3. **Index:** `username`
4. **Index:** `deleted_at` (for soft delete support)

## Recommended Composite Indexes

Based on the refactored logger system with pagination and filtering, the following composite indexes will significantly improve query performance:

### 1. Username + Created At (Priority: HIGH)

**Purpose:** Optimizes `GetLogsByUsername` queries with sorting by date

```sql
CREATE INDEX idx_username_created_at ON activitylog(username, created_at DESC);
```

**Benefits:**
- Accelerates username-based log retrieval (most common query)
- Supports efficient pagination with date sorting
- Reduces query time from O(n) to O(log n) for username lookups

**Estimated Impact:** 50-80% performance improvement on user-specific log queries

---

### 2. Level + Created At (Priority: HIGH)

**Purpose:** Optimizes filtering by log level (error, warn, info, etc.)

```sql
CREATE INDEX idx_level_created_at ON activitylog(level, created_at DESC);
```

**Benefits:**
- Fast retrieval of error logs for monitoring and debugging
- Supports efficient filtering by severity level
- Critical for alerting and log analysis systems

**Use Cases:**
- Error dashboards
- Security incident investigation
- Log level-based filtering

---

### 3. Status Code + Created At (Priority: MEDIUM)

**Purpose:** Optimizes filtering by HTTP status code

```sql
CREATE INDEX idx_status_code_created_at ON activitylog(status_code, created_at DESC);
```

**Benefits:**
- Quick identification of 4xx/5xx error patterns
- Supports API health monitoring
- Enables efficient status code-based analytics

**Use Cases:**
- Tracking 404 errors
- Monitoring 500 server errors
- API reliability metrics

---

### 4. Action + Created At (Priority: MEDIUM)

**Purpose:** Optimizes filtering by action type (login, logout, create, update, delete, etc.)

```sql
CREATE INDEX idx_action_created_at ON activitylog(action, created_at DESC);
```

**Benefits:**
- Fast retrieval of specific user actions
- Supports audit trail queries
- Enables action-based analytics

**Use Cases:**
- Security audits
- User behavior analysis
- Compliance reporting

---

### 5. Timestamp (Priority: MEDIUM)

**Purpose:** Dedicated index for time-based queries and date range filtering

```sql
CREATE INDEX idx_timestamp ON activitylog(timestamp DESC);
```

**Benefits:**
- Optimizes date range queries (date_from, date_to)
- Supports time-series analysis
- Enables efficient log retention policies

**Use Cases:**
- Historical log analysis
- Date range filtering
- Log retention cleanup

---

### 6. Route + Created At (Priority: LOW)

**Purpose:** Optimizes filtering by API route/endpoint

```sql
CREATE INDEX idx_route_created_at ON activitylog(route, created_at DESC);
```

**Benefits:**
- Fast retrieval of endpoint-specific logs
- Supports API usage analytics
- Enables route-based debugging

**Use Cases:**
- Endpoint performance monitoring
- API usage patterns
- Route-specific debugging

---

### 7. IP Address + Created At (Priority: LOW)

**Purpose:** Optimizes filtering by IP address for security investigations

```sql
CREATE INDEX idx_ip_address_created_at ON activitylog(ip_address, created_at DESC);
```

**Benefits:**
- Quick identification of suspicious IPs
- Supports rate limiting analysis
- Enables geographic analysis

**Use Cases:**
- Security incident response
- DDoS attack analysis
- Geographic traffic patterns

---

## Migration Script

### PostgreSQL

```sql
-- High priority indexes (implement immediately)
CREATE INDEX CONCURRENTLY idx_username_created_at ON activitylog(username, created_at DESC);
CREATE INDEX CONCURRENTLY idx_level_created_at ON activitylog(level, created_at DESC);

-- Medium priority indexes (implement within 1 week)
CREATE INDEX CONCURRENTLY idx_status_code_created_at ON activitylog(status_code, created_at DESC);
CREATE INDEX CONCURRENTLY idx_action_created_at ON activitylog(action, created_at DESC);
CREATE INDEX CONCURRENTLY idx_timestamp ON activitylog(timestamp DESC);

-- Low priority indexes (implement as needed based on usage patterns)
CREATE INDEX CONCURRENTLY idx_route_created_at ON activitylog(route, created_at DESC);
CREATE INDEX CONCURRENTLY idx_ip_address_created_at ON activitylog(ip_address, created_at DESC);
```

**Note:** `CONCURRENTLY` allows index creation without locking the table for writes.

### MySQL

```sql
-- High priority indexes
CREATE INDEX idx_username_created_at ON activitylog(username, created_at DESC);
CREATE INDEX idx_level_created_at ON activitylog(level, created_at DESC);

-- Medium priority indexes
CREATE INDEX idx_status_code_created_at ON activitylog(status_code, created_at DESC);
CREATE INDEX idx_action_created_at ON activitylog(action, created_at DESC);
CREATE INDEX idx_timestamp ON activitylog(timestamp DESC);

-- Low priority indexes
CREATE INDEX idx_route_created_at ON activitylog(route, created_at DESC);
CREATE INDEX idx_ip_address_created_at ON activitylog(ip_address, created_at DESC);
```

---

## Index Maintenance

### Monitoring Index Usage

#### PostgreSQL
```sql
-- Check index usage statistics
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes
WHERE tablename = 'activitylog'
ORDER BY idx_scan DESC;

-- Check unused indexes
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan
FROM pg_stat_user_indexes
WHERE tablename = 'activitylog' AND idx_scan = 0;
```

#### MySQL
```sql
-- Check index usage
SELECT 
    TABLE_NAME,
    INDEX_NAME,
    SEQ_IN_INDEX,
    COLUMN_NAME,
    CARDINALITY
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_NAME = 'activitylog'
ORDER BY SEQ_IN_INDEX;
```

### Index Size Monitoring

#### PostgreSQL
```sql
-- Check index sizes
SELECT 
    indexname,
    pg_size_pretty(pg_relation_size(indexname::regclass)) as index_size
FROM pg_indexes
WHERE tablename = 'activitylog'
ORDER BY pg_relation_size(indexname::regclass) DESC;
```

#### MySQL
```sql
-- Check index sizes
SELECT 
    INDEX_NAME,
    ROUND(SUM(stat_value * @@innodb_page_size) / 1024 / 1024, 2) as size_mb
FROM mysql.innodb_index_stats
WHERE database_name = DATABASE() AND table_name = 'activitylog'
GROUP BY INDEX_NAME
ORDER BY size_mb DESC;
```

---

## Performance Considerations

### Index Tradeoffs

**Benefits:**
- Faster SELECT queries (especially with WHERE, ORDER BY, and pagination)
- Improved response times for filtered log retrieval
- Better support for concurrent read operations

**Costs:**
- Increased storage requirements (estimate: 20-40% additional space)
- Slower INSERT operations (typically 5-15% overhead per index)
- Additional memory usage for index caching

### When to Add Indexes

✅ **Add indexes when:**
- Query execution time exceeds acceptable thresholds (>500ms)
- Specific filters are used frequently (>100 times/day)
- Table size exceeds 100,000 rows
- Pagination performance degrades

❌ **Avoid indexes when:**
- Column has very low cardinality (e.g., boolean fields)
- Write operations vastly outnumber read operations
- Storage space is extremely limited
- Table is very small (<10,000 rows)

---

## Query Optimization Tips

### 1. Use Covered Indexes

For queries that only need specific columns, ensure indexes cover those columns:

```sql
-- Example: Query only needs id, username, created_at
CREATE INDEX idx_username_id_created ON activitylog(username, id, created_at);
```

### 2. Avoid LIKE with Leading Wildcards

```sql
-- ❌ Bad: Cannot use index
WHERE route LIKE '%/user/%'

-- ✅ Good: Can use index
WHERE route LIKE '/user/%'
```

### 3. Use EXPLAIN to Analyze Queries

#### PostgreSQL
```sql
EXPLAIN ANALYZE
SELECT * FROM activitylog 
WHERE username = 'john_doe' 
ORDER BY created_at DESC 
LIMIT 10 OFFSET 0;
```

#### MySQL
```sql
EXPLAIN
SELECT * FROM activitylog 
WHERE username = 'john_doe' 
ORDER BY created_at DESC 
LIMIT 10 OFFSET 0;
```

---

## Partitioning Strategy (Optional - For High Volume)

For systems generating >1M logs per month, consider table partitioning:

### Time-Based Partitioning (PostgreSQL)

```sql
-- Create partitioned table
CREATE TABLE activitylog (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(255),
    created_at TIMESTAMP NOT NULL,
    -- other columns...
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE activitylog_2025_01 PARTITION OF activitylog
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE activitylog_2025_02 PARTITION OF activitylog
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

-- Auto-create partitions with pg_partman extension
```

### Benefits of Partitioning

- Faster queries on recent data
- Easy archival of old logs
- Improved maintenance operations (VACUUM, ANALYZE)
- Better query planner optimization

---

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Query Response Time**
   - Target: <100ms for paginated queries
   - Alert threshold: >500ms

2. **Index Hit Ratio**
   - Target: >95%
   - Alert threshold: <90%

3. **Table Size Growth**
   - Monitor: Daily growth rate
   - Alert: If exceeding 10% per day

4. **Lock Wait Time**
   - Target: <10ms
   - Alert threshold: >100ms

### Grafana Dashboard Queries (Example)

```sql
-- Average query response time (last 24h)
SELECT 
    DATE_TRUNC('hour', created_at) as time,
    AVG(response_time) as avg_response_ms
FROM activitylog
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY time
ORDER BY time;
```

---

## Rollback Plan

If indexes cause performance degradation:

```sql
-- Drop indexes individually
DROP INDEX CONCURRENTLY idx_username_created_at;
DROP INDEX CONCURRENTLY idx_level_created_at;

-- Or drop all at once (not recommended for production)
DROP INDEX idx_status_code_created_at;
DROP INDEX idx_action_created_at;
DROP INDEX idx_timestamp;
DROP INDEX idx_route_created_at;
DROP INDEX idx_ip_address_created_at;
```

---

## Implementation Checklist

- [ ] Analyze current query patterns with EXPLAIN
- [ ] Estimate table growth rate
- [ ] Create high-priority indexes (username, level)
- [ ] Monitor index creation progress
- [ ] Verify index usage with pg_stat_user_indexes
- [ ] Test query performance before/after
- [ ] Create medium-priority indexes if needed
- [ ] Set up monitoring alerts
- [ ] Document index creation in changelog
- [ ] Schedule quarterly index maintenance review

---

## References

- [PostgreSQL Index Documentation](https://www.postgresql.org/docs/current/indexes.html)
- [MySQL Index Documentation](https://dev.mysql.com/doc/refman/8.0/en/optimization-indexes.html)
- [GORM Indexing Best Practices](https://gorm.io/docs/indexes.html)

---

## Support

For questions or issues related to database indexing:
1. Check query execution plans with EXPLAIN
2. Monitor index usage statistics
3. Consult with database administrator
4. Review application logs for slow queries

**Last Updated:** February 2, 2026  
**Maintained By:** Backend Engineering Team
