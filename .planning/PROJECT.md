# DB Diff Tool - Project Context

## What This Is

一个命令行数据库比对工具，用于数据库迁移/重构场景下的表结构和数据差异分析。

**核心价值：** 一条命令完成两个数据库（MySQL/Oracle）的全面对比，生成清晰的 HTML 差异报告。

## Why This Exists

数据库迁移/重构时，需要验证源库和目标库是否一致。手动对比费时费力且容易遗漏。这个工具提供：
- 自动化的结构对比（表、字段、索引、约束）
- 灵活的数据对比（记录数/全量/抽样）
- 清晰的 HTML 报告输出

## Who It's For

- 开发人员：验证迁移前后的数据库一致性
- DBA：数据库重构时的差异分析
- 测试人员：环境间数据库对比

## Core Value

**简单易用** — 一条命令完成对比，配置灵活，报告清晰。

## Tech Stack (Validated)

| Layer | Choice | Rationale |
|-------|--------|-----------|
| 语言 | Go | 单二进制部署，数据库生态好 |
| MySQL 驱动 | go-sql-driver/mysql | 官方推荐，成熟稳定 |
| CLI 框架 | Cobra | Go 生态标准 CLI 框架 |
| 配置文件 | yaml.v3 | 易于阅读和编辑 |
| 报告引擎 | html/template | 标准库，轻量 |
| 测试 | sqlmock | 无需真实数据库 |
| 架构 | 单二进制 CLI | 零依赖部署 |

## Requirements

### Validated (v1.0 ✓)

- ✓ CONN-01~04: MySQL 数据库连接 — Phase 1
- ✓ STRUCT-01~05: 表结构对比 — Phase 1
- ✓ UX-01~03: 易用性 — Phase 1
- ✓ DATA-01~05: 数据对比模式 — Phase 2
- ✓ REPORT-01~05: HTML 报告 — Phase 3

### Validated (v2.0 ✓)

- ✓ V2-01: Oracle 数据库支持 — Phase 4
- ✓ V2-02: 跨数据库对比（MySQL ↔ Oracle）— Phase 4

### Active (v2.1)

- PERF-01: 并行查询优化 — 源和目标数据库查询改为并行执行，提升比对速度

### Out of Scope

- 数据修复/同步功能 — 只对比，不修改
- 实时同步监控 — 离线对比工具
- GUI 界面 — CLI 优先
- 其他数据库支持（PostgreSQL、SQL Server 等）— 聚焦 MySQL/Oracle

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go 语言 | 单二进制部署，数据库生态成熟 | 零依赖运行 ✓ |
| CLI 优先 | 简单易用，易集成到 CI/CD | 命令行交互 ✓ |
| HTML 报告 | 可视化差异，易于分享 | 结构化展示 ✓ |
| 先 MySQL 后 Oracle | 聚焦核心功能，逐步扩展 | 分阶段实现 ✓ |
| 数据对比三种模式 | 满足不同场景需求 | count/full/sample ✓ |

## Context

**Project Path:** `/Volumes/extern/claude_workspace/gsd_db_tool`

**Workspace:** 独立项目，无子 repo

**v2.0 Status:** Complete (2026-03-28)

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-03-28 after v2.0 completion*
