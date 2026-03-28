# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2026-03-28

### Added

**Phase 1: MySQL 连接与结构对比**
- MySQL 数据库连接支持
- 命令行参数和 YAML 配置文件
- 表结构对比（表、字段、索引、约束）
- 文本和 HTML 报告输出
- 单元测试

**Phase 2: 数据对比功能**
- 记录数对比模式 (`--data-mode count`)
- 全量数据对比模式 (`--data-mode full`)
- 抽样数据对比模式 (`--data-mode sample`)
- 指定表对比支持 (`--tables`)

**Phase 3: HTML 报告与完善**
- HTML 报告展示数据差异
- 记录数对比表格
- 数据行差异详情（新增/删除/修改）
- 文本报告扩展数据差异
- 折叠/展开详情功能

### Technical

- Go + Cobra CLI
- go-sql-driver/mysql
- yaml.v3 配置解析
- html/template 报告生成
- sqlmock 单元测试

### Stats

- 21 requirements implemented
- 3 phases completed
- 9 unit tests passing

## [Unreleased]

### Planned (v2.0)
- Oracle 数据库支持
- 跨数据库对比（MySQL ↔ Oracle）
- 性能优化：并行对比
- 更多报告格式：PDF、Markdown
