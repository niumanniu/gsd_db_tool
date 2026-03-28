# Phase 1 Discussion Notes

**Completed:** 2026-03-28

## Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI 设计 | 双 DSN 模式 (`--source --target`) | 简单易用，一条命令完成 |
| 配置文件 | YAML | 易于阅读和编辑 |
| 元数据查询 | INFORMATION_SCHEMA | 标准 SQL，结构化好解析 |
| 输出形式 | 终端摘要 + HTML 报告 | 兼顾快速查看和详细分享 |
| 测试策略 | sqlmock 模拟 | 无需真实数据库，快速可靠 |

## Implementation Approach

**技术栈：**
- 语言：Go 1.21+
- MySQL 驱动：`go-sql-driver/mysql`
- CLI 框架：Cobra
- 配置文件：`gopkg.in/yaml.v3`
- HTML 模板：标准库 `html/template`
- 测试：`sqlmock`

**Phase 1 范围确认：**
- CONN-01~04: 数据库连接
- STRUCT-01~05: 结构对比
- UX-01~03: 易用性

## Open Questions

None - ready for planning.
