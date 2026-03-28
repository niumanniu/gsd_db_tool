package comparator

import "fmt"

// MappingStatus 类型映射状态
type MappingStatus string

const (
	StatusSame       MappingStatus = "same"       // ✅ 相同 - 类型完全一致
	StatusMapped     MappingStatus = "mapped"     // ⚠️ 已映射匹配 - 通过映射表匹配
	StatusIncompatible MappingStatus = "incompatible" // ❌ 不兼容 - 无法映射的类型
)

// MappingResult 类型映射结果
type MappingResult struct {
	Status      MappingStatus
	MappedType  string // 映射后的类型（仅在 StatusMapped 时非空）
	Description string // 人类可读的描述
}

// mysqlToOracleTypeMap MySQL 到 Oracle 类型映射表
var mysqlToOracleTypeMap = map[string]string{
	"VARCHAR":   "VARCHAR2",
	"CHAR":      "CHAR",
	"TEXT":      "CLOB",
	"BLOB":      "BLOB",
	"INT":       "NUMBER",
	"BIGINT":    "NUMBER",
	"SMALLINT":  "NUMBER",
	"TINYINT":   "NUMBER(1)",
	"DATETIME":  "DATE",
	"TIMESTAMP": "TIMESTAMP",
	"DATE":      "DATE",
	"DECIMAL":   "NUMBER",
	"FLOAT":     "BINARY_FLOAT",
	"DOUBLE":    "BINARY_DOUBLE",
}

// oracleToMySQLTypeMap Oracle 到 MySQL 类型映射表
var oracleToMySQLTypeMap = map[string]string{
	"VARCHAR2":      "VARCHAR",
	"CHAR":          "CHAR",
	"CLOB":          "TEXT",
	"BLOB":          "BLOB",
	"NUMBER":        "DECIMAL",
	"BINARY_FLOAT":  "FLOAT",
	"BINARY_DOUBLE": "DOUBLE",
	"DATE":          "DATETIME",
	"TIMESTAMP":     "TIMESTAMP",
}

// MapType 映射类型
// sourceType: 源数据类型
// targetType: 目标数据类型
// sourceDriver: 源数据库驱动 (mysql|oracle)
// targetDriver: 目标数据库驱动 (mysql|oracle)
func MapType(sourceType, targetType, sourceDriver, targetDriver string) MappingResult {
	// 相同数据库：直接比较类型
	if sourceDriver == targetDriver {
		if sourceType == targetType {
			return MappingResult{
				Status:      StatusSame,
				MappedType:  "",
				Description: "类型相同",
			}
		}
		// 同库但类型不同，尝试映射（可能是精度/长度不同）
		return MappingResult{
			Status:      StatusMapped,
			MappedType:  targetType,
			Description: "类型不同但可映射",
		}
	}

	// 跨数据库对比
	var typeMap map[string]string
	if sourceDriver == "mysql" && targetDriver == "oracle" {
		typeMap = mysqlToOracleTypeMap
	} else if sourceDriver == "oracle" && targetDriver == "mysql" {
		typeMap = oracleToMySQLTypeMap
	} else {
		return MappingResult{
			Status:      StatusIncompatible,
			MappedType:  "",
			Description: "未知的数据库组合",
		}
	}

	// 查找映射
	if mapped, exists := typeMap[sourceType]; exists {
		// 检查映射后的类型是否与目标类型匹配
		if mapped == targetType {
			return MappingResult{
				Status:      StatusMapped,
				MappedType:  mapped,
				Description: fmt.Sprintf("类型映射：%s → %s", sourceType, mapped),
			}
		}
		// 映射类型与目标类型不同，但仍视为已映射（可能是精度不同）
		return MappingResult{
			Status:      StatusMapped,
			MappedType:  mapped,
			Description: fmt.Sprintf("类型映射：%s → %s (目标：%s)", sourceType, mapped, targetType),
		}
	}

	// 无法映射
	return MappingResult{
		Status:      StatusIncompatible,
		MappedType:  "",
		Description: fmt.Sprintf("类型不兼容：%s 无法映射到 %s", sourceType, targetType),
	}
}

// AreTypesCompatible 检查类型是否兼容
func AreTypesCompatible(sourceType, targetType, sourceDriver, targetDriver string) bool {
	result := MapType(sourceType, targetType, sourceDriver, targetDriver)
	return result.Status == StatusSame || result.Status == StatusMapped
}
