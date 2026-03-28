//go:build !oracle

package database

import (
	"db-diff/pkg/config"
	"fmt"
)

// OracleDriver Oracle 数据库驱动实现（stub，需要 oracle 构建标签）
// 注意：完整实现在 oracle_driver.go 中，需要网络下载 go-ora 依赖
type OracleDriver struct{}

// Connect 连接 Oracle 数据库（stub 实现）
func (o *OracleDriver) Connect(cfg config.Database) (*Connection, error) {
	return nil, fmt.Errorf("Oracle driver not available: rebuild with -tags oracle after running 'go get github.com/sijms/go-ora/v2'")
}

// FetchMetadata 获取 Oracle 数据库元数据（stub 实现）
func (o *OracleDriver) FetchMetadata(conn *Connection, schema string) (*DatabaseMeta, error) {
	return nil, fmt.Errorf("Oracle driver not available: rebuild with -tags oracle")
}

// FetchRowCount 获取 Oracle 表记录数（stub 实现）
func (o *OracleDriver) FetchRowCount(conn *Connection, table string) (int64, error) {
	return 0, fmt.Errorf("Oracle driver not available: rebuild with -tags oracle")
}

// FetchData 获取 Oracle 表数据（stub 实现）
func (o *OracleDriver) FetchData(conn *Connection, table string, columns []ColumnMeta) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("Oracle driver not available: rebuild with -tags oracle")
}
