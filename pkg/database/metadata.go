package database

import (
	"database/sql"
)

// TableMeta 表元数据
type TableMeta struct {
	Name    string
	Comment string
}

// ColumnMeta 列元数据
type ColumnMeta struct {
	Name            string
	DataType        string
	ColumnType      string
	CharacterMaxLen sql.NullInt64
	NumericPrecision sql.NullInt64
	NumericScale    sql.NullInt64
	IsNullable      string
	ColumnDefault   sql.NullString
	Extra           string
	Comment         string
	OrdinalPosition int
}

// IndexMeta 索引元数据
type IndexMeta struct {
	Name       string
	TableName  string
	ColumnName string
	SeqInIndex int
	NonUnique  int
	IndexType  string
}

// ConstraintMeta 约束元数据
type ConstraintMeta struct {
	Name       string
	Type       string
	TableName  string
	ColumnName string
}

// DatabaseMeta 数据库完整元数据
type DatabaseMeta struct {
	Tables      []TableMeta
	Columns     map[string][]ColumnMeta    // table_name -> columns
	Indexes     map[string][]IndexMeta     // table_name -> indexes
	Constraints map[string][]ConstraintMeta // table_name -> constraints
}
