---
plan: 05-PLAN.md
phase: 05-parallel-query-optimization
status: complete
started: 2026-04-01T00:00:00Z
completed: 2026-04-01T00:00:00Z
autonomous: true
---

## Objective

Refactor `compareTableDataFull` function to execute source and target database batch queries in parallel using goroutines, reducing wait time in network latency scenarios.

## What Was Built

**Parallel Query Implementation:**
- Created `fetchBatchParallel` helper function that executes source and target queries concurrently using goroutines
- Used `errgroup.WithContext` for coordinated error handling and context cancellation
- Implemented fast-fail mechanism: either query failure triggers immediate cancellation of the other

**Performance Timing Instrumentation:**
- Added per-batch timing tracking (batchStartTime, batchTime)
- Accumulated total parallel time vs estimated serial time
- Log output shows: serial estimate, parallel actual time, and savings percentage

**Dual Progress Display:**
- Split progress tracking into independent source and target counters
- Progress format: `Source: X% (processed/total) | Target: Y% (processed/total)`
- Independent tracking for source and target record counts

## Files Modified

| File | Changes |
|------|---------|
| `pkg/comparator/comparator.go` | Added fetchBatchParallel function, refactored compareTableDataFull loop, added timing and dual progress |
| `go.mod` | Added golang.org/x/sync dependency |
| `go.sum` | Added dependency checksum |

## Key Implementation Details

### fetchBatchParallel Function (lines 448-560)
```go
func fetchBatchParallel(ctx context.Context, ...) (..., err error)
```
- Creates context.WithCancel for coordinated cancellation
- Uses errgroup.WithContext for error propagation
- Spawns two goroutines: one for source query, one for target query
- On error from either goroutine, context is cancelled and other goroutine exits early
- Returns both result sets, next keys, and hash values (if applicable)

### Refactored compareTableDataFull Loop
- Replaced sequential fetch calls (lines 573-636 old) with single fetchBatchParallel call
- Preserved all existing logic: empty batch detection, map building, comparison, next key determination
- Added timing accumulation variables
- Updated progress display to use dual format with timing stats

### Timing Output Format
```
[users] Source: 75% (1500/2000) | Target: 45% (900/2000), 已耗时 2.3s, 本批：串行预估 100ms, 并行实际 55ms, 节省 45.0%
[users] 比对完成，总耗时 5.2s (串行预估：10.5s, 并行实际：5.2s), 平均节省 50.5%
```

## Verification

- [x] `go build ./pkg/comparator/...` succeeds
- [x] `go test ./pkg/comparator/...` passes (4 tests)
- [x] fetchBatchParallel uses goroutines for concurrent execution
- [x] Context cancellation propagates on error
- [x] Timing logs show serial vs parallel comparison
- [x] Progress output shows dual progress format
- [x] BatchSize config continues to work

## Requirements Satisfied

| Requirement | Status | Evidence |
|-------------|--------|----------|
| PERF-01: Source and target batch queries execute concurrently | ✓ | fetchBatchParallel uses errgroup.Go() for both queries |
| Fast-fail on error | ✓ | context.WithCancel called on error, other goroutine checks ctx.Done() |
| Timing instrumentation | ✓ | Per-batch and total timing logged |
| Dual progress display | ✓ | Source: X% | Target: Y% format implemented |

## Performance Expectations

Based on the implementation:
- **High network latency (100ms+):** ~50% time reduction expected
- **Low latency (local):** Minimal improvement, slight overhead from goroutine management
- **Hash filter enabled:** Same parallel benefits, hash computation still works correctly

## Notes

- errgroup chosen over raw WaitGroup for cleaner error propagation
- Channel-based result collection ensures both queries complete before returning
- Context cancellation provides clean shutdown on error conditions
