//go:build oracle

package database

import (
	"database/sql"
	"fmt"
	"strings"

	"db-diff/pkg/config"

	_ "github.com/sijms/go-ora/v2"
)

// OracleDriver Oracle 数据库驱动实现
type OracleDriver struct{}

// Connect 连接 Oracle 数据库
func (o *OracleDriver) Connect(cfg config.Database) (*Connection, error) {
	// 构建 Oracle DSN: oracle://user:pass@host:port/service_name
	dsn := o.buildDSN(cfg)

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库连接失败：%w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败：%w", err)
	}

	return &Connection{DB: db, DriverName: "oracle"}, nil
}

// buildDSN 构建 Oracle DSN 连接字符串
func (o *OracleDriver) buildDSN(cfg config.Database) string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
}

// FetchMetadata 获取 Oracle 数据库元数据（使用 USER_* 视图）
func (o *OracleDriver) FetchMetadata(conn *Connection, schema string) (*DatabaseMeta, error) {
	meta := &DatabaseMeta{
		Columns:     make(map[string][]ColumnMeta),
		Indexes:     make(map[string][]IndexMeta),
		Constraints: make(map[string][]ConstraintMeta),
	}

	// 获取表列表
	tables, err := o.fetchTables(conn, schema)
	if err != nil {
		return nil, fmt.Errorf("获取表列表失败：%w", err)
	}
	meta.Tables = tables

	// 获取每个表的列、索引、约束
	for _, table := range tables {
		columns, err := o.fetchColumns(conn, table)
		if err != nil {
			return nil, fmt.Errorf("获取表 %s 列信息失败：%w", table, err)
		}
		meta.Columns[table.Name] = columns

		indexes, err := o.fetchIndexes(conn, table)
		if err != nil {
			return nil, fmt.Errorf("获取表 %s 索引信息失败：%w", table, err)
		}
		meta.Indexes[table.Name] = indexes

		constraints, err := o.fetchConstraints(conn, table)
		if err != nil {
			return nil, fmt.Errorf("获取表 %s 约束信息失败：%w", table, err)
		}
		meta.Constraints[table.Name] = constraints
	}

	return meta, nil
}

// FetchRowCount 获取 Oracle 表记录数
func (o *OracleDriver) FetchRowCount(conn *Connection, table string) (int64, error) {
	var count int64
	// Oracle 表名可能需要双引号（区分大小写）
	query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, table)
	err := conn.DB.QueryRow(query).Scan(&count)
	if err != nil {
		// 尝试不带引号的表名
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err = conn.DB.QueryRow(query).Scan(&count)
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}

// FetchData 获取 Oracle 表数据
func (o *OracleDriver) FetchData(conn *Connection, table string, columns []ColumnMeta) ([]map[string]interface{}, error) {
	colNames := make([]string, len(columns))
	for i, c := range columns {
		colNames[i] = c.Name
	}

	// Oracle  ORDER BY 第一个字段
	query := fmt.Sprintf(`SELECT * FROM "%s" ORDER BY "%s"`, table, colNames[0])
	rows, err := conn.DB.Query(query)
	if err != nil {
		// 尝试不带引号的表名
		query = fmt.Sprintf("SELECT * FROM %s ORDER BY %s", table, colNames[0])
		rows, err = conn.DB.Query(query)
		if err != nil {
			return nil, err
		}
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

// fetchTables 获取 Oracle 表列表（使用 USER_TABLES）
func (o *OracleDriver) fetchTables(conn *Connection, schema string) ([]TableMeta, error) {
	// Oracle 默认大写，但 USER_TABLES 包含当前用户的所有表
	query := `
		SELECT TABLE_NAME, '' as TABLE_COMMENT
		FROM USER_TABLES
		ORDER BY TABLE_NAME
	`
	rows, err := conn.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableMeta
	for rows.Next() {
		var t TableMeta
		var comment string
		if err := rows.Scan(&t.Name, &comment); err != nil {
			return nil, err
		}
		t.Comment = comment
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

// fetchColumns 获取 Oracle 列信息（使用 USER_TAB_COLUMNS）
func (o *OracleDriver) fetchColumns(conn *Connection, table string) ([]ColumnMeta, error) {
	// Oracle 字段名默认大写，查询时使用 UPPER 转换
	query := `
		SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, DATA_PRECISION,
		       DATA_SCALE, NULLABLE, DATA_DEFAULT
		FROM USER_TAB_COLUMNS
		WHERE TABLE_NAME = :1
		ORDER BY COLUMN_ID
	`
	rows, err := conn.DB.Query(query, strings.ToUpper(table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnMeta
	for rows.Next() {
		var col ColumnMeta
		var nullable string
		var dataLength, dataPrecision, dataScale sql.NullInt64

		// Oracle 没有 COLUMN_TYPE, EXTRA, COLUMN_COMMENT, ORDINAL_POSITION
		// 使用 DATA_LENGTH, DATA_PRECISION, DATA_SCALE 替代
		if err := rows.Scan(
			&col.Name, &col.DataType, &dataLength, &dataPrecision, &dataScale,
			&nullable, &col.ColumnDefault,
		); err != nil {
			return nil, err
		}

		// 转换 NULLABLE 字段
		col.IsNullable = nullable
		if nullable == "Y" {
			col.IsNullable = "YES"
		} else {
			col.IsNullable = "NO"
		}

		// 设置精度和标度
		col.NumericPrecision = dataPrecision
		col.NumericScale = dataScale
		col.CharacterMaxLen = dataLength

		// Oracle 没有 ColumnType，用 DataType 填充
		col.ColumnType = col.DataType

		columns = append(columns, col)
	}
	return columns, rows.Err()
}

// fetchIndexes 获取 Oracle 索引信息（使用 USER_IND_COLUMNS + USER_INDEXES）
func (o *OracleDriver) fetchIndexes(conn *Connection, table string) ([]IndexMeta, error) {
	query := `
		SELECT i.INDEX_NAME, ic.COLUMN_NAME, ic.COLUMN_POSITION, i.UNIQUENESS
		FROM USER_IND_COLUMNS ic
		JOIN USER_INDEXES i ON ic.INDEX_NAME = i.INDEX_NAME
		WHERE ic.TABLE_NAME = :1
		ORDER BY i.INDEX_NAME, ic.COLUMN_POSITION
	`
	rows, err := conn.DB.Query(query, strings.ToUpper(table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexMeta
	for rows.Next() {
		var idx IndexMeta
		var uniqueness string
		if err := rows.Scan(&idx.Name, &idx.ColumnName, &idx.SeqInIndex, &uniqueness); err != nil {
			return nil, err
		}
		idx.TableName = table
		// Oracle UNIQUENESS: UNIQUE or NONUNIQUE
		if uniqueness == "UNIQUE" {
			idx.NonUnique = 0
		} else {
			idx.NonUnique = 1
		}
		idx.IndexType = "NORMAL" // Oracle 默认索引类型
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

// fetchConstraints 获取 Oracle 约束信息（使用 USER_CONSTRAINTS + USER_CONS_COLUMNS）
func (o *OracleDriver) fetchConstraints(conn *Connection, table string) ([]ConstraintMeta, error) {
	query := `
		SELECT c.CONSTRAINT_NAME, c.CONSTRAINT_TYPE, cc.COLUMN_NAME
		FROM USER_CONSTRAINTS c
		JOIN USER_CONS_COLUMNS cc ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME
		WHERE c.TABLE_NAME = :1
		ORDER BY c.CONSTRAINT_NAME
	`
	rows, err := conn.DB.Query(query, strings.ToUpper(table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []ConstraintMeta
	for rows.Next() {
		var c ConstraintMeta
		var constraintType string
		if err := rows.Scan(&c.Name, &constraintType, &c.ColumnName); err != nil {
			return nil, err
		}
		c.TableName = table
		// Oracle CONSTRAINT_TYPE: P=Primary Key, R=Foreign Key, U=Unique, C=Check
		switch constraintType {
		case "P":
			c.Type = "PRIMARY KEY"
		case "R":
			c.Type = "FOREIGN KEY"
		case "U":
			c.Type = "UNIQUE"
		case "C":
			c.Type = "CHECK"
		default:
			c.Type = constraintType
		}
		constraints = append(constraints, c)
	}
	return constraints, rows.Err()
}
