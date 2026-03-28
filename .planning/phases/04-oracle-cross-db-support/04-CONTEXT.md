# Phase 4 Context: Oracle + 跨数据库支持

**Phase:** 4 (v2.0)
**Created:** 2026-03-28
**Status:** Decisions captured, ready for research + planning

---

## 决策汇总

### 1. 数据库驱动架构

**决策：抽象接口层（Driver Interface）**

当前 MySQL 专用代码需要重构为接口 + 实现模式：

```go
// 新增：pkg/database/driver.go
type Driver interface {
    Connect(cfg Database) (*Connection, error)
    FetchMetadata(conn *Connection, schema string) (*DatabaseMeta, error)
    FetchRowCount(conn *Connection, table string) (int64, error)
    FetchData(conn *Connection, table string, columns []ColumnMeta) ([]map[string]interface{}, error)
}

// MySQL 实现
type MySQLDriver struct{}

// Oracle 实现
type OracleDriver struct{}
```

**对 downstream 的影响：**
- **Researcher:** 需要调研 Go 的 Oracle 驱动选型（`godror` vs `go-ora`），评估稳定性、社区活跃度、Oracle 官方支持情况
- **Planner:** 需要包含「抽象接口层重构」任务，这是 Oracle 支持的前置条件

---

### 2. 跨数据库对比语义

**决策：智能类型映射 + 警告标记**

MySQL 和 Oracle 类型不同，通过映射表判断"兼容性"，报告中标注映射状态：

| 状态 | 说明 | 示例 |
|------|------|------|
| ✅ 相同 | 类型完全一致 | `INT` ↔ `INT`（罕见） |
| ⚠️ 已映射匹配 | 通过映射表匹配 | `VARCHAR` ↔ `VARCHAR2` |
| ❌ 不兼容 | 无法映射的类型 | `GEOMETRY` ↔ （无对应） |

**类型映射表（初步）：**
```go
var mysqlOracleTypeMap = map[string]string{
    "VARCHAR":    "VARCHAR2",
    "CHAR":       "CHAR",
    "TEXT":       "CLOB",
    "BLOB":       "BLOB",
    "INT":        "NUMBER",
    "BIGINT":     "NUMBER",
    "SMALLINT":   "NUMBER",
    "TINYINT":    "NUMBER(1)",
    "DATETIME":   "DATE",
    "TIMESTAMP":  "TIMESTAMP",
    "DATE":       "DATE",
    "DECIMAL":    "NUMBER",
    "FLOAT":      "BINARY_FLOAT",
    "DOUBLE":     "BINARY_DOUBLE",
}
```

**对比逻辑：**
1. 精确匹配 → ✅
2. 映射匹配 → ⚠️ + 显示映射关系
3. 无法映射 → ❌

**对 downstream 的影响：**
- **Researcher:** 需要完善类型映射表，调研边缘情况（精度、长度、符号位）
- **Planner:** 需要包含「类型映射模块」和「报告警告展示」任务

---

### 3. 连接配置扩展

**决策：添加 `driver` 字段**

配置结构扩展：

```yaml
# 配置文件示例
source:
  driver: mysql
  host: localhost
  port: 3306
  user: root
  password: secret
  database: testdb

target:
  driver: oracle
  host: localhost
  port: 1521
  user: MY_SCHEMA
  password: secret
  database: ORCL
  schema: MY_SCHEMA  # 可选，默认等于 user
```

**CLI 参数扩展：**
```bash
db-diff \
  --source-driver=mysql --source-host=localhost --source-port=3306 \
  --target-driver=oracle --target-host=localhost --target-port=1521 \
  --target-schema=MY_SCHEMA
```

**对 downstream 的影响：**
- **Researcher:** 需要确认 Oracle 连接参数约定（服务名 vs SID，schema 命名惯例）
- **Planner:** 需要包含 `config.go` 扩展、CLI 参数扩展任务

---

### 4. Oracle 特定功能处理

**决策：显式支持 schema 参数**

Oracle 用户与 schema 分离，需要显式支持：

- `Database.schema` 字段（可选，默认等于 `Database.user`）
- Oracle 元数据查询使用 `USER_*` 视图（当前用户 schema）
- 文档说明边缘情况：
  - 大小写敏感（Oracle 默认大写，加引号区分大小写）
  - 空字符串 `''` = `NULL`
  - 自增主键使用 `SEQUENCE + TRIGGER` 而非 `AUTO_INCREMENT`

**Oracle 元数据查询（USER 视图）：**
```sql
-- 表列表
SELECT TABLE_NAME FROM USER_TABLES

-- 列信息
SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, DATA_PRECISION,
       DATA_SCALE, NULLABLE, DATA_DEFAULT
FROM USER_TAB_COLUMNS
WHERE TABLE_NAME = :1

-- 索引信息
SELECT INDEX_NAME, COLUMN_NAME, COLUMN_POSITION, UNIQUENESS
FROM USER_IND_COLUMNS
JOIN USER_INDEXES USING (INDEX_NAME)

-- 约束信息
SELECT CONSTRAINT_NAME, CONSTRAINT_TYPE, COLUMN_NAME
FROM USER_CONSTRAINTS
JOIN USER_CONS_COLUMNS USING (CONSTRAINT_NAME)
```

**对 downstream 的影响：**
- **Researcher:** 需要验证 `USER_*` 视图权限要求，确认跨 schema 查询是否需要 `ALL_*` 视图
- **Planner:** 需要包含 Oracle 专用元数据查询实现任务

---

### 5. 报告输出格式

**决策：单一报告 + 类型映射警告**

保持 v1.0 单一报告格式，内嵌类型映射警告：

```html
<tr class="match">
  <td>id</td>
  <td>INT</td>
  <td>NUMBER</td>
  <td>⚠️ 已映射匹配</td>
</tr>
<tr class="diff">
  <td>geo_data</td>
  <td>GEOMETRY</td>
  <td>-</td>
  <td>❌ 类型不兼容</td>
</tr>
```

**图例：**
- ✅ 相同
- ⚠️ 已映射匹配
- ❌ 不兼容/缺失

**对 downstream 的影响：**
- **Researcher:** 无需额外调研
- **Planner:** 需要包含报告模块扩展任务（HTML + 文本）

---

##  deferred ideas（延期到后续版本）

| 想法 | 说明 | 可能归属的 phase |
|------|------|----------------|
| PostgreSQL 支持 | 第 3 个数据库 | v2.1 或 Phase 5 |
| 类型映射自动学习 | 从用户反馈中优化映射 | 后续优化 |
| 跨库数据修复 | 生成同步 SQL | 超出范围（只对比不修改） |
| GUI 界面 | 可视化操作 | 超出范围（CLI 优先） |

---

## 下游 agents 行动指南

### gsd-phase-researcher 需要调查：

1. **Oracle 驱动选型**
   - `github.com/godror/godror`（Oracle 官方推荐）
   - `github.com/sijms/go-ora`（纯 Go 实现）
   - 对比：稳定性、性能、Oracle 版本支持、社区活跃度

2. **类型映射完整性**
   - MySQL 所有数据类型列表
   - Oracle 所有数据类型列表
   - 边缘情况：精度、长度、符号位、NULL 处理

3. **Oracle 权限模型**
   - `USER_*` vs `ALL_*` vs `DBA_*` 视图权限差异
   - 跨 schema 查询是否需要额外权限

### gsd-planner 需要包含的任务：

1. 驱动抽象层设计与实现
2. Oracle 驱动集成
3. Oracle 元数据查询实现
4. 配置结构扩展（driver 字段、schema 参数）
5. CLI 参数扩展
6. 类型映射模块
7. 跨库对比逻辑（类型映射 + 警告）
8. 报告模块扩展（警告展示）
9. 测试（MySQL→Oracle 跨库对比场景）

---

## 成功标准（Success Criteria）

Phase 4 完成后应满足：

- [ ] 支持 Oracle 9i/10g/11g/12c/19c 连接
- [ ] 支持 MySQL → Oracle 跨库对比
- [ ] 支持 Oracle → MySQL 跨库对比
- [ ] 支持 Oracle → Oracle 同构对比
- [ ] 报告正确展示类型映射警告
- [ ] 配置支持 `driver` 和 `schema` 参数
- [ ] 所有 v1.0 测试继续通过

---

*Last updated: 2026-03-28*
