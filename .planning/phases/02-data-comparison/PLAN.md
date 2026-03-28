# Phase 2 Plan: 数据对比功能

**Goal:** 实现数据对比多种模式

**Success Criteria:**
1. 支持记录数对比模式
2. 支持全量数据对比模式
3. 支持抽样数据对比模式
4. 支持全库或指定表对比

---

## Tasks

### Task 1: 数据对比配置

**Description:** 扩展配置支持数据对比模式

- [ ] 添加数据对比模式枚举 (count/full/sample)
- [ ] 添加抽样比例配置
- [ ] 添加指定表列表配置
- [ ] CLI 参数扩展：`--data-mode`, `--sample-ratio`, `--tables`

### Task 2: 记录数对比

**Description:** 实现快速记录数对比

- [ ] 查询表记录数 (`SELECT COUNT(*)`)
- [ ] 对比记录数差异
- [ ] 记录数差异报告

### Task 3: 全量数据对比

**Description:** 实现逐行数据对比

- [ ] 按主键/全部字段排序读取数据
- [ ] 逐行对比字段值
- [ ] 记录差异（新增行、删除行、修改行）
- [ ] 大表分批处理

### Task 4: 抽样数据对比

**Description:** 实现抽样数据对比

- [ ] 随机抽样 (`ORDER BY RAND() LIMIT N`)
- [ ] 按比例抽样
- [ ] 抽样对比报告

### Task 5: 数据对比报告

**Description:** 扩展报告支持数据对比结果

- [ ] 文本报告扩展数据差异
- [ ] HTML 报告扩展数据差异
- [ ] 差异统计摘要

### Task 6: 测试

**Description:** 数据对比功能测试

- [ ] 记录数对比测试
- [ ] 全量对比测试 (sqlmock)
- [ ] 抽样对比测试

---

## Dependencies

| Task | Depends On |
|------|------------|
| Task 1 | - |
| Task 2 | Task 1 |
| Task 3 | Task 1 |
| Task 4 | Task 1 |
| Task 5 | Task 2, 3, 4 |
| Task 6 | Task 2-5 |
