# Requirements

## v1 Requirements (Complete ✓)

### 数据库连接
- [x] **CONN-01**: 支持 MySQL 数据库连接（主机、端口、用户名、密码、数据库名）
- [x] **CONN-02**: 支持命令行参数指定数据库连接
- [x] **CONN-03**: 支持配置文件指定数据库连接
- [x] **CONN-04**: 支持两个数据库连接（源库和目标库）

### 结构对比
- [x] **STRUCT-01**: 对比表列表差异（缺失表、多余表）
- [x] **STRUCT-02**: 对比字段定义（名称、类型、长度、精度）
- [x] **STRUCT-03**: 对比字段属性（nullable、默认值）
- [x] **STRUCT-04**: 对比索引定义（索引名、类型、字段、唯一性）
- [x] **STRUCT-05**: 对比约束定义（主键、外键、唯一约束）

### 数据对比
- [x] **DATA-01**: 记录数对比模式（快速对比）
- [x] **DATA-02**: 全量数据对比模式（逐行对比）
- [x] **DATA-03**: 抽样数据对比模式（可配置抽样比例）
- [x] **DATA-04**: 支持全库所有表对比
- [x] **DATA-05**: 支持指定表对比

### 报告输出
- [x] **REPORT-01**: 生成 HTML 格式报告
- [x] **REPORT-02**: 报告展示表结构差异详情
- [x] **REPORT-03**: 报告展示数据差异详情
- [x] **REPORT-04**: 报告可折叠/展开详情
- [x] **REPORT-05**: 报告包含差异统计摘要

### 易用性
- [x] **UX-01**: 一条命令完成对比
- [x] **UX-02**: 清晰的命令行帮助信息
- [x] **UX-03**: 友好的错误提示

## v2 Requirements (Complete ✓)

- [x] **V2-01**: Oracle 数据库支持 — 支持 Oracle 9i/10g/11g/12c/19c 连接，使用 USER_* 视图获取元数据，支持 schema 参数
- [x] **V2-02**: 跨数据库对比（MySQL ↔ Oracle）— 支持 MySQL ↔ Oracle 双向对比，智能类型映射与警告标记，报告显示类型兼容状态

## v2.1 Requirements (Complete ✓)

### 性能优化
- [x] **PERF-01**: 并行查询优化 — 将源和目标数据库的查询操作改为并行执行，减少总体等待时间

**Acceptance Criteria:**
1. ✓ 使用 goroutine + channel 或 errgroup 实现并发查询
2. ✓ 源表和目标表的批次数据读取同时发起
3. ✓ 等待两个查询都完成后再进行比对
4. ✓ 错误处理正确，任一查询失败时另一个查询被取消
5. ✓ 性能提升可测量（网络延迟 100ms 场景下，期望 ~50% 时间减少）

**Out of Scope:**
- 不改变批次比对算法逻辑
- 不改变 Hash 预筛选机制
- 不改变现有配置接口

---

## Out of Scope

- 数据修复/同步功能 — 只对比，不修改
- 实时同步监控 — 离线对比工具
- GUI 界面 — CLI 优先
- 其他数据库支持（PostgreSQL、SQL Server 等）— 聚焦 MySQL/Oracle

---

## Summary

**v1.0: 21/21 requirements complete** ✓

| Category | Requirements | Status |
|----------|-------------|--------|
| 数据库连接 | 4 | ✓ |
| 结构对比 | 5 | ✓ |
| 数据对比 | 5 | ✓ |
| 报告输出 | 5 | ✓ |
| 易用性 | 3 | ✓ |

**v2.0: 2/2 requirements complete** ✓

| Category | Requirements | Status |
|----------|-------------|--------|
| Oracle 支持 | 1 | ✓ |
| 跨数据库对比 | 1 | ✓ |

**v2.1: 1/1 requirements complete** ✓

| Category | Requirements | Status |
|----------|-------------|--------|
| 性能优化 | 1 | ✓ |
