package database

import (
	"db-diff/pkg/config"
)

// Connect 连接数据库（convenience function using MySQL driver）
// Deprecated: Use driver interface instead
func Connect(cfg config.Database) (*Connection, error) {
	driver := &MySQLDriver{}
	return driver.Connect(cfg)
}

// TestConnection 测试数据库连接
func TestConnection(cfg config.Database) error {
	conn, err := Connect(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
