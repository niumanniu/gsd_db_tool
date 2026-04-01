# Roadmap

## Phase 1: MySQL 连接与结构对比 ✓

**Status:** Complete (2026-03-28)

**Goal:** 实现 MySQL 数据库连接和表结构对比核心功能

**Requirements:** CONN-01, CONN-02, CONN-03, CONN-04, STRUCT-01, STRUCT-02, STRUCT-03, STRUCT-04, STRUCT-05, UX-01, UX-02, UX-03

**Success Criteria:** ✓ All met

## Phase 2: 数据对比功能 ✓

**Status:** Complete (2026-03-28)

**Goal:** 实现数据对比多种模式

**Requirements:** DATA-01, DATA-02, DATA-03, DATA-04, DATA-05

**Success Criteria:** ✓ All met

## Phase 3: HTML 报告与完善 ✓

**Status:** Complete (2026-03-28)

**Goal:** 实现 HTML 报告输出，完成 v1

**Requirements:** REPORT-01, REPORT-02, REPORT-03, REPORT-04, REPORT-05

**Success Criteria:** ✓ All met

---

## v1.0 Summary

**All 3 phases complete.** 21 requirements implemented.

| Phase | Requirements | Status |
|-------|-------------|--------|
| Phase 1 | 11 | ✓ |
| Phase 2 | 5 | ✓ |
| Phase 3 | 5 | ✓ |

---

## v2.0 Summary

**All 1 phase complete.** 2 requirements implemented.

| Phase | Requirements | Status |
|-------|-------------|--------|
| Phase 4 | 2 | ✓ |

---

## v2.1 Summary

**1/1 phases complete.** 1 requirement implemented.

| Phase | Requirements | Status |
|-------|-------------|--------|
| Phase 5 | 1 | ✓ |

---

## Phase 5: 并行查询优化 ✓

**Status:** Complete (2026-04-01)

**Goal:** 将源和目标数据库的查询操作改为并行执行，提升比对速度

**Requirements:**
- **PERF-01:** 并行查询优化 — 使用 goroutine 实现并发查询，减少网络延迟场景下的总体等待时间

**Success Criteria:** ✓ All met

**Plans:** 1/1 complete

Plans:
- [x] 05-01-PLAN.md — Refactor compareTableDataFull to use parallel goroutines for batch queries with context cancellation, timing instrumentation, and dual progress display

---

## Phase 4: Oracle + 跨数据库支持 ✓

**Status:** Complete (2026-03-28)

**Goal:** 支持 Oracle 数据库和 MySQL ↔ Oracle 跨数据库对比

**Requirements:**
- **V2-01:** Oracle 数据库支持 — 支持 Oracle 9i/10g/11g/12c/19c 连接，使用 USER_* 视图获取元数据，支持 schema 参数
- **V2-02:** 跨数据库对比 — 支持 MySQL ↔ Oracle 双向对比，智能类型映射与警告标记，报告显示类型兼容状态

**Success Criteria:** ✓ All met

**Plans:** 3/3 complete
- [x] 04-01-PLAN.md — 驱动抽象层、MySQL 驱动、Oracle 驱动、Config 扩展
- [x] 04-02-PLAN.md — CLI 参数、类型映射模块、Comparator 集成
- [x] 04-03-PLAN.md — 报告类型警告显示、E2E 验证

---

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CONN-01 | 1 | ✓ |
| CONN-02 | 1 | ✓ |
| CONN-03 | 1 | ✓ |
| CONN-04 | 1 | ✓ |
| STRUCT-01 | 1 | ✓ |
| STRUCT-02 | 1 | ✓ |
| STRUCT-03 | 1 | ✓ |
| STRUCT-04 | 1 | ✓ |
| STRUCT-05 | 1 | ✓ |
| DATA-01 | 2 | ✓ |
| DATA-02 | 2 | ✓ |
| DATA-03 | 2 | ✓ |
| DATA-04 | 2 | ✓ |
| DATA-05 | 2 | ✓ |
| REPORT-01 | 3 | ✓ |
| REPORT-02 | 3 | ✓ |
| REPORT-03 | 3 | ✓ |
| REPORT-04 | 3 | ✓ |
| REPORT-05 | 3 | ✓ |
| UX-01 | 1 | ✓ |
| UX-02 | 1 | ✓ |
| UX-03 | 1 | ✓ |
| V2-01 | 4 | ✓ |
| V2-02 | 4 | ✓ |
