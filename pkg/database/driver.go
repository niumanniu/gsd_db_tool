package database

import (
	"database/sql"

	"db-diff/pkg/config"
)

// Driver 数据库驱动接口
type Driver interface {
	// Connect 连接数据库
	Connect(cfg config.Database) (*Connection, error)

	// FetchMetadata 获取数据库元数据
	FetchMetadata(conn *Connection, schema string) (*DatabaseMeta, error)

	// FetchRowCount 获取表记录数
	FetchRowCount(conn *Connection, table string) (int64, error)

	// FetchData 获取表数据
	FetchData(conn *Connection, table string, columns []ColumnMeta) ([]map[string]interface{}, error)
}

// Connection 数据库连接
type Connection struct {
	DB *sql.DB
}

// Close 关闭连接
func (c *Connection) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
