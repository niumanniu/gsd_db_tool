# Phase 4 Plan 01 Summary

**Phase:** 04-oracle-cross-db-support
**Plan:** 01 (Wave 1)
**Date:** 2026-03-28
**Status:** COMPLETE

---

## Objectives Completed

Build foundation layer: driver abstraction interface, MySQL driver extraction, Oracle driver implementation, and config extension.

## Tasks Completed

### Task 1: Create driver abstraction layer
**File:** `pkg/database/driver.go`

- Created `Driver` interface with 4 methods:
  - `Connect(cfg config.Database) (*Connection, error)`
  - `FetchMetadata(conn *Connection, schema string) (*DatabaseMeta, error)`
  - `FetchRowCount(conn *Connection, table string) (int64, error)`
  - `FetchData(conn *Connection, table string, columns []ColumnMeta) ([]map[string]interface{}, error)`
- Defined `Connection` struct with `DB *sql.DB` field
- Added `Close()` method to Connection

**Verification:** `go build ./pkg/database/...` succeeds

---

### Task 2: Extract MySQL driver implementation
**File:** `pkg/database/mysql_driver.go`

- Created `MySQLDriver` struct implementing `Driver` interface
- Implemented `Connect()` using `github.com/go-sql-driver/mysql`
- Implemented `FetchMetadata()` using `information_schema` tables:
  - `information_schema.TABLES` for table list
  - `information_schema.COLUMNS` for column details
  - `information_schema.STATISTICS` for indexes
  - `information_schema.TABLE_CONSTRAINTS` + `KEY_COLUMN_USAGE` for constraints
- Implemented `FetchRowCount()` using `SELECT COUNT(*)`
- Implemented `FetchData()` using `SELECT * ORDER BY`

**Verification:** `go build ./pkg/database/...` succeeds, existing MySQL functionality preserved

---

### Task 3: Implement Oracle driver
**File:** `pkg/database/oracle_driver.go` (behind `//go:build oracle` tag)
**Stub File:** `pkg/database/oracle_driver_stub.go` (default build)

- Created `OracleDriver` struct implementing `Driver` interface
- Uses `github.com/sijms/go-ora/v2` driver (pure Go, no Oracle client needed)
- Implemented `Connect()` with Oracle DSN: `oracle://user:pass@host:port/service_name`
- Implemented `FetchMetadata()` using Oracle `USER_*` views:
  - `USER_TABLES` for table list
  - `USER_TAB_COLUMNS` for column details
  - `USER_IND_COLUMNS` + `USER_INDEXES` for indexes
  - `USER_CONSTRAINTS` + `USER_CONS_COLUMNS` for constraints
- Implemented `FetchRowCount()` and `FetchData()` with proper Oracle quoting

**Note:** Oracle driver requires network to download `go-ora/v2` dependency. Stub implementation returns error message directing user to rebuild with `-tags oracle`.

**Verification:** `go build ./pkg/database/...` succeeds (Oracle driver excluded by default)

---

### Task 4: Extend config with driver and schema fields
**File:** `pkg/config/config.go`

- Added `Driver string` field to `Database` struct (yaml:"driver")
- Added `Schema string` field to `Database` struct (yaml:"schema")
- Implemented `GetSchema()` method:
  - Returns `Schema` if set
  - Returns `User` for Oracle (default schema = user)
  - Returns `Database` for MySQL
- Refactored `DSN()` to dispatch based on driver:
  - `MySQLDSN()`: `user:pass@tcp(host:port)/database?parseTime=true`
  - `OracleDSN()`: `oracle://user:pass@host:port/database`
- Updated `Validate()` to set default driver to "mysql"

**Verification:** `go test ./pkg/config/...` passes

---

## Additional Changes

### Updated Files

**pkg/database/connection.go:**
- Removed duplicate `Connection` struct (now in driver.go)
- Removed duplicate `Close()` method
- Kept deprecated `Connect()` convenience function (uses MySQLDriver internally)
- Kept `TestConnection()` function

**pkg/database/metadata.go:**
- Removed `FetchMetadata()` method from Connection (now in driver implementations)
- Kept all metadata structs: `TableMeta`, `ColumnMeta`, `IndexMeta`, `ConstraintMeta`, `DatabaseMeta`

**pkg/comparator/comparator.go:**
- Added `getDriver()` helper function to obtain driver instances
- Updated `Compare()` to use driver interface:
  - Uses `cfg.Source.Driver` and `cfg.Target.Driver`
  - Calls `driver.Connect()` and `driver.FetchMetadata()`
  - Uses `cfg.GetSchema()` for schema parameter

---

## Build Verification

```bash
# Build all packages
go build ./...
# Result: SUCCESS

# Run all tests
go test ./...
# Result: All tests pass
?   	db-diff/cmd	[no test files]
ok  	db-diff/pkg/comparator	0.642s
ok  	db-diff/pkg/config	1.039s
?   	db-diff/pkg/database	[no test files]
?   	db-diff/pkg/report	[no test files]
```

---

## Files Created/Modified

### Created
- `pkg/database/driver.go` - Driver interface definition
- `pkg/database/mysql_driver.go` - MySQL driver implementation
- `pkg/database/oracle_driver.go` - Oracle driver implementation (behind build tag)
- `pkg/database/oracle_driver_stub.go` - Oracle stub for default build

### Modified
- `pkg/config/config.go` - Extended with Driver and Schema fields
- `pkg/database/connection.go` - Refactored to use driver interface
- `pkg/database/metadata.go` - Removed FetchMetadata method
- `pkg/comparator/comparator.go` - Updated to use driver interface
- `go.mod` - (Pending: go-ora/v2 dependency when network available)

---

## Known Limitations

1. **Oracle Driver Dependency**: Network is currently blocked, cannot download `github.com/sijms/go-ora/v2`. Oracle driver is behind `//go:build oracle` tag with stub implementation for default build.

2. **To Enable Oracle Support**: When network is available:
   ```bash
   go get github.com/sijms/go-ora/v2
   go build -tags oracle ./...
   ```

---

## Success Criteria Met

- [x] Driver interface defined in `pkg/database/driver.go`
- [x] `MySQLDriver` extracted and implements `Driver` interface
- [x] `OracleDriver` implemented with `USER_*` views (behind build tag)
- [x] Config supports `Driver` and `Schema` fields
- [x] `go build ./pkg/database/...` succeeds
- [x] `go test ./pkg/config/...` passes
- [x] Comparator updated to use driver interface

---

## Next Steps

Proceed to **Plan 04-02 (Wave 2)**:
- Task 5: Extend CLI with driver and schema parameters (`cmd/root.go`)
- Task 6: Create type mapping module (`pkg/comparator/type_mapper.go`)
- Task 7: Update comparator to use type mapping

---

*Last updated: 2026-03-28*
