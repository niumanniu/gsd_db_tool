# db-diff v1.0

数据库表结构 & 数据对比工具 - 简单易用，一条命令完成对比

## 功能特性

**v1.0 完整功能：**

- ✅ **结构对比**：表、字段、索引、约束
- ✅ **数据对比**：记录数/全量/抽样三种模式
- ✅ **HTML 报告**：可折叠详情，统计摘要
- ✅ **文本输出**：终端快速查看
- ✅ **灵活配置**：命令行参数或 YAML 文件
- ✅ **指定范围**：全库或指定表对比

## 安装

```bash
# 从源码构建
go build -o db-diff ./cmd/...

# 添加到 PATH
cp db-diff /usr/local/bin/
```

## 快速开始

### 基本用法

```bash
# 结构对比（默认）
./db-diff -s "root:pass@tcp(host:3306)/db1" \
          -t "root:pass@tcp(host:3306)/db2"

# 生成 HTML 报告
./db-diff -s "..." -t "..." -f html -o report.html
```

### 数据对比

```bash
# 记录数对比（快速）
./db-diff -s "..." -t "..." --data-mode count

# 全量数据对比
./db-diff -s "..." -t "..." --data-mode full

# 抽样对比（10%）
./db-diff -s "..." -t "..." --data-mode sample --sample-ratio 0.1
```

### 指定表对比

```bash
./db-diff -s "..." -t "..." --tables users,posts,comments
```

### 配置文件

```bash
# 创建配置文件
cp config.example.yaml config.yaml

# 使用配置文件
./db-diff --config config.yaml
```

## 命令行参数

| 参数 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--source` | `-s` | 源数据库 DSN | - |
| `--target` | `-t` | 目标数据库 DSN | - |
| `--config` | `-c` | 配置文件路径 (YAML) | - |
| `--format` | `-f` | 输出格式 (text\|html) | text |
| `--output` | `-o` | 输出文件路径 | stdout |
| `--data-mode` | `-m` | 数据对比模式 | count |
| `--sample-ratio` | `-r` | 抽样比例 (0.0-1.0) | 0.1 |
| `--tables` | `-T` | 指定表列表 | 全部 |
| `--verbose` | `-v` | 详细信息 | false |

## DSN 格式

```
user:password@tcp(host:port)/database
```

示例：
```
root:secret@tcp(localhost:3306)/mydb
```

## 配置文件格式

```yaml
source:
  host: localhost
  port: 3306
  user: root
  password: your_password
  database: source_db

target:
  host: localhost
  port: 3306
  user: root
  password: your_password
  database: target_db

compare_options:
  data_mode: count     # count|full|sample
  sample_ratio: 0.1
  tables:              # 空表示所有表
    - users
    - posts
```

## 对比范围

db-diff 支持对比：

| 类别 | 对比项 |
|------|--------|
| **表** | 缺失表、额外表 |
| **列** | 名称、类型、长度、精度、可空性、默认值 |
| **索引** | 索引名、类型、字段 |
| **约束** | 主键、外键、唯一约束 |
| **数据** | 记录数、新增行、删除行、修改行 |

## 使用场景

### 数据库迁移验证

```bash
./db-diff -s "root:pass@tcp(old-host:3306)/olddb" \
          -t "root:pass@tcp(new-host:3306)/newdb" \
          -f html -o migration-report.html
```

### 环境间对比

```bash
./db-diff -s "dev:pass@tcp(dev-db:3306)/app" \
          -t "prod:pass@tcp(prod-db:3306)/app" \
          --data-mode count
```

### Schema 变更验证

```bash
# 部署前
./db-diff -s "..." -t "..." --data-mode full -f html -o before.html

# 部署后
./db-diff -s "..." -t "..." --data-mode full -f html -o after.html
```

## 项目结构

```
cmd/
  root.go              # CLI 入口
pkg/
  config/
    config.go          # 配置管理
  database/
    connection.go      # 数据库连接
    metadata.go        # 元数据查询
  comparator/
    comparator.go      # 对比引擎
  report/
    text_report.go     # 文本报告
    html_report.go     # HTML 报告
config.example.yaml    # 配置示例
```

## 技术栈

- **语言**: Go
- **MySQL 驱动**: go-sql-driver/mysql
- **CLI 框架**: Cobra
- **配置文件**: yaml.v3
- **HTML 模板**: html/template
- **测试**: sqlmock

## v2 规划

- [ ] Oracle 数据库支持
- [ ] 跨数据库对比（MySQL ↔ Oracle）
- [ ] 性能优化：并行对比
- [ ] 更多报告格式：PDF、Markdown

## License

MIT
