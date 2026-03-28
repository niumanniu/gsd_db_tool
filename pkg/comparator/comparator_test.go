package comparator

import (
	"db-diff/pkg/database"
	"testing"
)

func TestCompareTables(t *testing.T) {
	source := []database.TableMeta{
		{Name: "users"},
		{Name: "posts"},
		{Name: "comments"},
	}
	target := []database.TableMeta{
		{Name: "users"},
		{Name: "posts"},
		{Name: "tags"},
	}

	diff := compareTables(source, target)

	if len(diff.Added) != 1 || diff.Added[0] != "tags" {
		t.Errorf("Added = %v, want [tags]", diff.Added)
	}
	if len(diff.Missing) != 1 || diff.Missing[0] != "comments" {
		t.Errorf("Missing = %v, want [comments]", diff.Missing)
	}
}

func TestCompareColumns(t *testing.T) {
	source := []database.ColumnMeta{
		{Name: "id", DataType: "int", ColumnType: "int(11)", IsNullable: "NO", Extra: "auto_increment"},
		{Name: "name", DataType: "varchar", ColumnType: "varchar(255)", IsNullable: "NO"},
		{Name: "email", DataType: "varchar", ColumnType: "varchar(255)", IsNullable: "YES"},
		{Name: "deleted", DataType: "tinyint", ColumnType: "tinyint(1)", IsNullable: "YES"},
	}
	target := []database.ColumnMeta{
		{Name: "id", DataType: "int", ColumnType: "int(11)", IsNullable: "NO", Extra: "auto_increment"},
		{Name: "name", DataType: "varchar", ColumnType: "varchar(500)", IsNullable: "NO"}, // 修改：长度变化
		{Name: "email", DataType: "varchar", ColumnType: "varchar(255)", IsNullable: "NO"},  // 修改：nullable 变化
		{Name: "created_at", DataType: "timestamp", ColumnType: "timestamp", IsNullable: "YES"}, // 新增
	}

	diff := compareColumns(source, target, "mysql", "mysql")

	if len(diff.Added) != 1 || diff.Added[0].Name != "created_at" {
		t.Errorf("Added = %v, want [created_at]", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].Name != "deleted" {
		t.Errorf("Removed = %v, want [deleted]", diff.Removed)
	}
	if len(diff.Modified) != 2 {
		t.Errorf("Modified = %d, want 2", len(diff.Modified))
	}
}

func TestCompareColumn(t *testing.T) {
	tests := []struct {
		name   string
		source database.ColumnMeta
		target database.ColumnMeta
		wantChanges int
	}{
		{
			name: "same column",
			source: database.ColumnMeta{DataType: "int", ColumnType: "int(11)", IsNullable: "NO"},
			target: database.ColumnMeta{DataType: "int", ColumnType: "int(11)", IsNullable: "NO"},
			wantChanges: 0,
		},
		{
			name: "different type",
			source: database.ColumnMeta{DataType: "int", ColumnType: "int(11)"},
			target: database.ColumnMeta{DataType: "bigint", ColumnType: "bigint(20)"},
			wantChanges: 2,
		},
		{
			name: "different nullable",
			source: database.ColumnMeta{IsNullable: "NO"},
			target: database.ColumnMeta{IsNullable: "YES"},
			wantChanges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, _ := compareColumn(tt.source, tt.target, "mysql", "mysql")
			if len(changes) != tt.wantChanges {
				t.Errorf("compareColumn() changes = %d, want %d", len(changes), tt.wantChanges)
			}
		})
	}
}

func TestGetCommonTables(t *testing.T) {
	source := []database.TableMeta{
		{Name: "users"},
		{Name: "posts"},
		{Name: "comments"},
	}
	target := []database.TableMeta{
		{Name: "users"},
		{Name: "posts"},
		{Name: "tags"},
	}

	common := getCommonTables(source, target)

	if len(common) != 2 {
		t.Errorf("getCommonTables() = %d, want 2", len(common))
	}

	commonSet := make(map[string]bool)
	for _, t := range common {
		commonSet[t] = true
	}
	if !commonSet["users"] || !commonSet["posts"] {
		t.Errorf("getCommonTables() missing expected tables")
	}
}
