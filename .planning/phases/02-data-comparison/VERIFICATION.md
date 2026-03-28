# Phase 2 Verification: 数据对比功能

**Completed:** 2026-03-28

## Success Criteria Verification

### ✓ 1. 支持记录数对比模式

**实现：**
- `--data-mode count` (默认)
- `pkg/comparator/comparator.go:compareTableCount()`

**使用：**
```bash
./db-diff -s "root:pass@tcp(host:3306)/db1" \
          -t "root:pass@tcp(host:3306)/db2" \
          --data-mode count
```

### ✓ 2. 支持全量数据对比模式

**实现：**
- `--data-mode full`
- `pkg/comparator/comparator.go:compareTableDataFull()`
- 按主键对比，支持新增/删除/修改行检测

**使用：**
```bash
./db-diff -s "root:pass@tcp(host:3306)/db1" \
          -t "root:pass@tcp(host:3306)/db2" \
          --data-mode full
```

### ✓ 3. 支持抽样数据对比模式

**实现：**
- `--data-mode sample --sample-ratio 0.1`
- `pkg/comparator/comparator.go:compareTableDataSample()`
- 随机抽样，限制最大 1000 条

**使用：**
```bash
./db-diff -s "root:pass@tcp(host:3306)/db1" \
          -t "root:pass@tcp(host:3306)/db2" \
          --data-mode sample --sample-ratio 0.1
```

### ✓ 4. 支持全库所有表对比

**实现：**
- 默认对比所有共同表
- 自动检测表列表

### ✓ 5. 支持指定表对比

**实现：**
- `--tables table1,table2,table3`
- 配置文件中 `compare_options.tables`

**使用：**
```bash
./db-diff -s "root:pass@tcp(host:3306)/db1" \
          -t "root:pass@tcp(host:3306)/db2" \
          --tables users,posts,comments
```

## Requirements Coverage

### DATA (数据对比)
- [x] **DATA-01**: 记录数对比模式
- [x] **DATA-02**: 全量数据对比模式
- [x] **DATA-03**: 抽样数据对比模式
- [x] **DATA-04**: 支持全库所有表对比
- [x] **DATA-05**: 支持指定表对比

## New CLI Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--data-mode` | `-m` | count | 数据对比模式 (count\|full\|sample) |
| `--sample-ratio` | `-r` | 0.1 | 抽样比例 (0.0-1.0) |
| `--tables` | `-T` | - | 指定表列表，逗号分隔 |

## Config File Example

```yaml
source:
  host: localhost
  port: 3306
  user: root
  password: secret
  database: source_db

target:
  host: localhost
  port: 3306
  user: root
  password: secret
  database: target_db

compare_options:
  data_mode: full        # count|full|sample
  sample_ratio: 0.1      # 10% 抽样
  tables:                # 空表示所有表
    - users
    - posts
```

## Test Results

```
=== RUN   TestCompareTables
--- PASS: TestCompareTables (0.00s)
=== RUN   TestCompareColumns
--- PASS: TestCompareColumns (0.00s)
=== RUN   TestCompareColumn
--- PASS: TestCompareColumn (0.00s)
=== RUN   TestGetCommonTables
--- PASS: TestGetCommonTables (0.00s)
=== RUN   TestLoad_FromConfigFile
--- PASS: TestLoad_FromConfigFile (0.00s)
=== RUN   TestLoad_FromDSN
--- PASS: TestLoad_FromDSN (0.00s)
=== RUN   TestLoad_DSNOverridesConfig
--- PASS: TestLoad_DSNOverridesConfig (0.00s)
=== RUN   TestConfig_Validate
--- PASS: TestConfig_Validate (0.00s)
=== RUN   TestDatabase_DSN
--- PASS: TestDatabase_DSN (0.00s)

All tests passed ✓
```

## Implementation Notes

### 数据对比模式

1. **Count 模式**: 快速对比表记录数差异
   - 查询：`SELECT COUNT(*) FROM table`
   - 输出：记录数差异

2. **Full 模式**: 全量逐行对比
   - 按主键排序读取全表数据
   - 主键检测：auto_increment 字段或 id 字段
   - 对比每行数据，检测新增/删除/修改

3. **Sample 模式**: 随机抽样对比
   - `ORDER BY RAND() LIMIT N`
   - 抽样比例可配置 (默认 10%)
   - 最大抽样 1000 条

### 限制

- 全量对比大表可能较慢
- 抽样对比使用 `ORDER BY RAND()` 在大数据量下性能一般
- 数据对比需要主键才能精确匹配行

## Next Phase

**Phase 3: HTML 报告与完善**

- REPORT-01: 生成 HTML 格式报告
- REPORT-02: 报告展示数据差异详情
- REPORT-03: 报告可折叠/展开详情
- REPORT-04: 报告包含差异统计摘要
