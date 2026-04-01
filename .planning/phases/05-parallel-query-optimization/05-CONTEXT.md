# Phase 5 Context: 并行查询优化

**Phase:** 5 (v2.1)
**Created:** 2026-04-01
**Status:** Decisions captured, ready for research + planning

---

## 决策汇总

### 1. 并行范围

**决策：仅并行化批次数据读取（compareTableDataFull）**

只在 `compareTableDataFull` 函数中并行化源表和目标表的批次数据读取操作。元数据获取（FetchMetadata）保持串行。

**范围明确：**
- ✅ 并行：`compareTableDataFull` 中的 source/target 批次查询
- ❌ 不并行：`Compare` 函数中的元数据获取
- ❌ 不并行：多表之间的并行（表级别串行执行）

**理由：** 最小改动，风险最低。数据对比是网络延迟敏感的主要场景，并行化收益最大。

**对 downstream 的影响：**
- **Researcher:** 无需额外调研
- **Planner:** 需要包含 `compareTableDataFull` 函数重构任务，使用 goroutine + channel/waitgroup 实现并行

---

### 2. 错误处理

**决策：立即取消（Fast Fail）**

使用 `context.WithCancel` 实现：任一查询失败时，立即取消另一个查询并返回错误。

**实现要点：**
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// 启动两个 goroutine 分别查询 source 和 target
// 任一失败时调用 cancel()，另一个 goroutine 通过 ctx.Done() 检测并退出
```

**理由：** 快速失败，避免浪费资源和用户等待时间。

**对 downstream 的影响：**
- **Planner:** 需要包含 context 集成、错误传播、资源清理任务

---

### 3. 性能度量

**决策：日志输出耗时对比**

在关键路径添加 timing 日志，输出串行/并行耗时对比。

**实现位置：**
- `compareTableDataFull` 函数开始和结束时刻
- 每个批次查询的开始/结束时刻
- 输出格式示例：
  ```
  [table_name] 批次对比：串行预估 200ms, 并行实际 105ms, 节省 47.5%
  ```

**理由：** 简单直接，运行时可见，无需额外测试框架。

**对 downstream 的影响：**
- **Planner:** 需要包含 timing 日志添加任务

---

### 4. 进度显示

**决策：双进度显示**

分别显示 source 和 target 的进度状态，格式：
```
[table_name] Source: 50% (1000/2000) | Target: 30% (600/2000)
```

**理由：** 透明展示两个查询的独立进度，用户能清楚看到哪个查询是瓶颈。

**对 downstream 的影响：**
- **Planner:** 需要修改现有的进度显示逻辑，支持双进度追踪

---

### 5. 批次大小

**决策：保持现有配置**

使用现有的 `cfg.BatchSize` 配置项，不针对并行场景做特殊调整。

**理由：** 当前批次大小（默认 1000）已经过验证，并行化后仍能保持良好性能。

**对 downstream 的影响：**
- **Planner:** 无需额外任务

---

### 6. 降级策略

**决策：不支持降级**

并行是唯一的实现方式，不提供串行降级开关。

**理由：** 增加复杂度但收益有限。并行化是性能优化，不应成为可选项。

**对 downstream 的影响：**
- **Planner:** 无需 CLI 参数扩展任务

---

## Canonical Refs（规范引用）

以下文件是 Phase 5 实现的权威参考，按优先级排序：

1. `.planning/phases/04-oracle-cross-db-support/04-CONTEXT.md` — 现有并行/异步模式参考（如有）
2. `pkg/comparator/comparator.go` — 需要重构的目标函数 `compareTableDataFull`
3. `pkg/config/config.go` — 配置结构（BatchSize 等）

---

## Deferred Ideas（延期到后续版本）

| 想法 | 说明 | 可能归属的 phase |
|------|------|----------------|
| 元数据获取并行化 | FetchMetadata 也并行执行 | v2.2 性能优化 |
| 多表并行对比 | 表与表之间并行执行 | v2.2 性能优化 |
| 可配置并行度 | 允许用户控制并发连接数 | v2.2 性能优化 |
| 动态批次大小 | 根据网络延迟自动调整 | 后续优化 |

---

## 成功标准（Success Criteria）

Phase 5 完成后应满足：

- [ ] 源表和目标表的批次数据读取同时发起（使用 goroutine）
- [ ] 等待两个查询都完成后再进行比对
- [ ] 错误处理正确，任一查询失败时另一个查询被取消
- [ ] 性能提升可测量（网络延迟 100ms 场景下，期望 ~50% 时间减少）
- [ ] 进度显示正确展示双进度
- [ ] 所有 v2.0 测试继续通过

---

## 对下游 agents 的行动指南

### gsd-phase-researcher

无需额外调研。如需了解 Go 并发最佳实践，可参考：
- Go `sync.WaitGroup` 文档
- Go `context` 包文档
- Go `errgroup` 包（可选，更优雅的错误处理）

### gsd-planner

需要包含的任务：

1. `compareTableDataFull` 函数重构 — 使用 goroutine 并行化 source/target 查询
2. Context 集成 — 实现取消传播和错误处理
3. 进度显示重构 — 支持双进度输出
4. Timing 日志 — 添加性能度量输出
5. 测试 — 验证并行逻辑正确性（包括错误传播场景）

---

*Last updated: 2026-04-01*
