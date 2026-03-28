package database

import (
	"database/sql"
	"fmt"

	"db-diff/pkg/config"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLDriver MySQL 数据库驱动实现
type MySQLDriver struct{}

// Connect 连接 MySQL 数据库
func (m *MySQLDriver) Connect(cfg config.Database) (*Connection, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败：%w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败：%w", err)
	}

	return &Connection{DB: db}, nil
}

// FetchMetadata 获取 MySQL 数据库元数据
func (m *MySQLDriver) FetchMetadata(conn *Connection, schema string) (*DatabaseMeta, error) {
	meta := &DatabaseMeta{
		Columns:     make(map[string][]ColumnMeta),
		Indexes:     make(map[string][]IndexMeta),
		Constraints: make(map[string][]ConstraintMeta),
	}

	// 获取表列表
	tables, err := m.fetchTables(conn, schema)
	if err != nil {
		return nil, fmt.Errorf("获取表列表失败：%w", err)
	}
	meta.Tables = tables

	// 获取每个表的列、索引、约束
	for _, table := range tables {
		columns, err := m.fetchColumns(conn, schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("获取表 %s 列信息失败：%w", table.Name, err)
		}
		meta.Columns[table.Name] = columns

		indexes, err := m.fetchIndexes(conn, schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("获取表 %s 索引信息失败：%w", table.Name, err)
		}
		meta.Indexes[table.Name] = indexes

		constraints, err := m.fetchConstraints(conn, schema, table.Name)
		if err != nil {
			return nil, fmt.Errorf("获取表 %s 约束信息失败：%w", table.Name, err)
		}
		meta.Constraints[table.Name] = constraints
	}

	return meta, nil
}

// FetchRowCount 获取 MySQL 表记录数
func (m *MySQLDriver) FetchRowCount(conn *Connection, table string) (int64, error) {
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)
	err := conn.DB.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// FetchData 获取 MySQL 表数据
func (m *MySQLDriver) FetchData(conn *Connection, table string, columns []ColumnMeta) ([]map[string]interface{}, error) {
	colNames := make([]string, len(columns))
	for i, c := range columns {
		colNames[i] = c.Name
	}

	query := fmt.Sprintf("SELECT * FROM `%s` ORDER BY `%s`", table, colNames[0])
	rows, err := conn.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 获取列名
	cols, _ := rows.Columns()
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var data []map[string]interface{}
	for rows.Next() {
		rows.Scan(valuePtrs...)
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		data = append(data, row)
	}

	return data, rows.Err()
}

// fetchTables 获取 MySQL 表列表
func (m *MySQLDriver) fetchTables(conn *Connection, schema string) ([]TableMeta, error) {
	query := `
		SELECT TABLE_NAME, TABLE_COMMENT
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME
	`
	rows, err := conn.DB.Query(query, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableMeta
	for rows.Next() {
		var t TableMeta
		if err := rows.Scan(&t.Name, &t.Comment); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

// fetchColumns 获取 MySQL 列信息
func (m *MySQLDriver) fetchColumns(conn *Connection, schema, table string) ([]ColumnMeta, error) {
	query := `
		SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE,
		       CHARACTER_MAXIMUM_LENGTH, NUMERIC_PRECISION, NUMERIC_SCALE,
		       IS_NULLABLE, COLUMN_DEFAULT, EXTRA, COLUMN_COMMENT, ORDINAL_POSITION
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`
	rows, err := conn.DB.Query(query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnMeta
	for rows.Next() {
		var col ColumnMeta
		if err := rows.Scan(
			&col.Name, &col.DataType, &col.ColumnType,
			&col.CharacterMaxLen, &col.NumericPrecision, &col.NumericScale,
			&col.IsNullable, &col.ColumnDefault, &col.Extra, &col.Comment,
			&col.OrdinalPosition,
		); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

// fetchIndexes 获取 MySQL 索引信息
func (m *MySQLDriver) fetchIndexes(conn *Connection, schema, table string) ([]IndexMeta, error) {
	query := `
		SELECT INDEX_NAME, COLUMN_NAME, SEQ_IN_INDEX, NON_UNIQUE, INDEX_TYPE
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`
	rows, err := conn.DB.Query(query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexMeta
	for rows.Next() {
		var idx IndexMeta
		if err := rows.Scan(&idx.Name, &idx.ColumnName, &idx.SeqInIndex, &idx.NonUnique, &idx.IndexType); err != nil {
			return nil, err
		}
		idx.TableName = table
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

// fetchConstraints 获取 MySQL 约束信息
func (m *MySQLDriver) fetchConstraints(conn *Connection, schema, table string) ([]ConstraintMeta, error) {
	query := `
		SELECT tc.CONSTRAINT_NAME, tc.CONSTRAINT_TYPE, kcu.COLUMN_NAME
		FROM information_schema.TABLE_CONSTRAINTS tc
		JOIN information_schema.KEY_COLUMN_USAGE kcu
		  ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME AND tc.TABLE_NAME = kcu.TABLE_NAME
		WHERE tc.TABLE_SCHEMA = ? AND tc.TABLE_NAME = ?
		ORDER BY tc.CONSTRAINT_NAME
	`
	rows, err := conn.DB.Query(query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []ConstraintMeta
	for rows.Next() {
		var c ConstraintMeta
		if err := rows.Scan(&c.Name, &c.Type, &c.ColumnName); err != nil {
			return nil, err
		}
		c.TableName = table
		constraints = append(constraints, c)
	}
	return constraints, rows.Err()
}
