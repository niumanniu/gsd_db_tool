# Phase 1 Plan: MySQL 连接与结构对比

**Goal:** 实现 MySQL 数据库连接和表结构对比核心功能

**Success Criteria:**
1. 能通过命令行或配置文件连接两个 MySQL 数据库
2. 能对比并输出表结构差异（表、字段、索引、约束）
3. 命令行帮助信息清晰可用

---

## Tasks

### Task 1: 项目初始化

**Description:** 创建 Go 项目结构和基础配置

- [ ] 初始化 Go module (`go mod init db-diff`)
- [ ] 创建目录结构：
  ```
  cmd/
  pkg/
    config/
    database/
    comparator/
    report/
  templates/
  ```
- [ ] 添加依赖：Cobra, mysql driver, yaml.v3
- [ ] 创建基础 CLI 框架

### Task 2: 配置管理

**Description:** 实现命令行参数和 YAML 配置文件解析

- [ ] 定义 CLI 参数：`--source`, `--target`, `--config`, `--format`
- [ ] 定义 Config 结构体（支持 DSN 和 YAML 配置）
- [ ] 实现 YAML 配置加载
- [ ] 实现参数优先级：命令行 > 配置文件
- [ ] 添加配置验证

### Task 3: 数据库连接

**Description:** 实现 MySQL 数据库连接管理

- [ ] 实现数据库连接函数
- [ ] 实现连接测试
- [ ] 实现连接关闭
- [ ] 添加连接错误处理

### Task 4: 元数据查询

**Description:** 从 INFORMATION_SCHEMA 获取表结构信息

- [ ] 查询表列表 (`information_schema.tables`)
- [ ] 查询列定义 (`information_schema.columns`)
- [ ] 查询索引定义 (`information_schema.statistics`)
- [ ] 查询表约束 (`information_schema.table_constraints`, `key_column_usage`)
- [ ] 定义元数据结构体

### Task 5: 结构对比引擎

**Description:** 对比两个数据库的结构差异

- [ ] 对比表列表（缺失表、多余表）
- [ ] 对比列定义（名称、类型、长度、精度、nullable、默认值）
- [ ] 对比索引定义
- [ ] 对比约束定义
- [ ] 生成对比结果数据结构

### Task 6: 终端输出

**Description:** 实现终端摘要输出

- [ ] 实现差异摘要打印
- [ ] 使用颜色标识差异类型（添加/删除/修改）
- [ ] 实现详细模式 (`--verbose`)
- [ ] 添加进度指示

### Task 7: HTML 报告

**Description:** 生成 HTML 格式对比报告

- [ ] 设计 HTML 模板结构
- [ ] 实现可折叠/展开的详情区域
- [ ] 实现差异统计摘要
- [ ] 添加 CSS 样式
- [ ] 实现报告文件写入

### Task 8: 测试

**Description:** 使用 sqlmock 实现单元测试

- [ ] 配置 sqlmock
- [ ] 测试配置加载
- [ ] 测试元数据查询
- [ ] 测试对比逻辑
- [ ] 测试报告生成

---

## Verification Loop

**Plan Checker Agent:** 验证计划能实现 Phase 1 目标

- [ ] 所有 CONN 需求覆盖
- [ ] 所有 STRUCT 需求覆盖
- [ ] 所有 UX 需求覆盖
- [ ] 任务依赖关系合理

---

## Dependencies

| Task | Depends On |
|------|------------|
| Task 1 | - |
| Task 2 | Task 1 |
| Task 3 | Task 2 |
| Task 4 | Task 3 |
| Task 5 | Task 4 |
| Task 6 | Task 5 |
| Task 7 | Task 5 |
| Task 8 | Task 2-7 |
