---
phase: 05-parallel-query-optimization
plan: 01
type: execute
wave: 1
depends_on: []
files_modified: [pkg/comparator/comparator.go]
autonomous: true
requirements: [PERF-01]

must_haves:
  truths:
    - "Source and target batch queries execute concurrently using goroutines"
    - "Both queries must complete before comparison logic proceeds"
    - "Either query failure cancels the other and returns error immediately"
    - "Timing logs show serialized vs parallel execution time comparison"
    - "Progress display shows dual progress: Source X% | Target Y%"
  artifacts:
    - path: "pkg/comparator/comparator.go"
      provides: "Parallel batch query implementation in compareTableDataFull"
      contains: "goroutine|errgroup|sync.WaitGroup"
      exports: ["compareTableDataFull"]
  key_links:
    - from: "compareTableDataFull batch loop"
      to: "context.WithCancel"
      via: "error propagation"
      pattern: "context\\.WithCancel|cancel\\(\\)"
    - from: "compareTableDataFull batch loop"
      to: "goroutine execution"
      via: "concurrent query execution"
      pattern: "go func\\(|errgroup\\.Go"
    - from: "compareTableDataFull progress output"
      to: "dual progress tracking"
      via: "separate source/target counters"
      pattern: "Source:.*Target:"
---

<objective>
Refactor `compareTableDataFull` function to execute source and target database batch queries in parallel using goroutines, reducing wait time in network latency scenarios.

Purpose: Improve data comparison performance by ~50% in high network latency scenarios (100ms+) by eliminating sequential query wait times.
Output: Parallelized batch query logic with proper error handling, context cancellation, timing instrumentation, and dual progress display.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/05-parallel-query-optimization/05-CONTEXT.md

[Target function to refactor:]
@pkg/comparator/comparator.go

[Config structure for BatchSize:]
@pkg/config/config.go
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add parallel query helper function</name>
  <files>pkg/comparator/comparator.go</files>
  <behavior>
    - Test 1: fetchBatchParallel executes both queries concurrently (measured time < sequential time)
    - Test 2: Source query error cancels target query (context cancelled)
    - Test 3: Target query error cancels source query (context cancelled)
    - Test 4: Both queries succeed returns both result sets
    - Test 5: Either query returns empty result, other continues normally
  </behavior>
  <action>
    Extract batch query logic into a new helper function that executes source and target queries in parallel:

    ```go
    // fetchBatchParallel fetches source and target data concurrently
    // Returns error if either query fails, with context cancellation for the other
    func fetchBatchParallel(
        ctx context.Context,
        sourceConn, targetConn *database.Connection,
        table string,
        columns []database.ColumnMeta,
        filteredCols []database.ColumnMeta,
        pkCols []string,
        currentKey interface{},
        batchSize int,
        hashFilter bool,
        sourceDriver, targetDriver string,
    ) (sourceBatch, targetBatch []map[string]interface{}, sourceNext, targetNext []interface{}, sourceHash, targetHash string, err error)
    ```

    Implementation using errgroup (preferred) or sync.WaitGroup:
    1. Create context.WithCancel from input ctx
    2. Use errgroup.WithContext for coordinated error handling
    3. g.Go() for source query - calls existing fetch logic
    4. g.Go() for target query - calls existing fetch logic
    5. g.Wait() - blocks until both complete or one fails
    6. On error: cancel() propagates to other goroutine

    For hashFilter=true case:
    - Source query uses fetchMySQLDataWithHash (MySQL) or fetchDataByKeyRange (Oracle)
    - Target query uses fetchMySQLDataWithHash (MySQL) or fetchDataByKeyRange (Oracle)

    For hashFilter=false case:
    - Both use fetchDataByKeyRange

    Error handling (per 05-CONTEXT.md decision 2 - Fast Fail):
    - Either query fails → cancel() called immediately
    - Other goroutine checks ctx.Done() and exits early
    - Return error with context information
  </action>
  <verify>
    <automated>go build ./pkg/comparator/... succeeds</automated>
  </verify>
  <done>fetchBatchParallel helper function implemented with goroutines, context cancellation, and errgroup coordination</done>
</task>

<task type="auto">
  <name>Task 2: Refactor compareTableDataFull batch loop</name>
  <files>pkg/comparator/comparator.go</files>
  <read_first>pkg/comparator/comparator.go (lines 566-636)</read_first>
  <action>
    Refactor the main batch loop in compareTableDataFull (lines 566-724) to use the new fetchBatchParallel helper:

    Current sequential pattern (lines 566-636):
    ```go
    for {
        // Sequential queries - SLOW
        sourceBatch, sourceNext, ... = fetch...()  // Wait for source
        targetBatch, targetNext, ... = fetch...()  // Then wait for target
        // ... rest of loop
    }
    ```

    New parallel pattern:
    ```go
    for {
        // Parallel queries - FAST
        sourceBatch, targetBatch, sourceNext, targetNext, ..., err = fetchBatchParallel(
            ctx, sourceConn, targetConn, table, columns, filteredCols,
            pkCols, currentKey, batchSize, hashFilter, sourceDriver, targetDriver,
        )
        if err != nil {
            return diff, err
        }
        // ... rest of loop unchanged
    }
    ```

    Changes required:
    1. Create context at function start: ctx, cancel := context.WithCancel(context.Background()), defer cancel()
    2. Replace sequential fetch calls (lines 573-636) with single fetchBatchParallel call
    3. Handle the returned results (same variables, no logic change needed)
    4. Remove duplicate error handling - single error return from parallel function

    Preserve existing logic:
    - Empty batch detection (line 638-639)
    - Progress counter updates (lines 642-645)
    - Map building and comparison (lines 654-698)
    - Next key determination (lines 704-723)
    - Remaining records handling (lines 726-733)
  </action>
  <verify>
    <automated>go build ./pkg/comparator/... succeeds</automated>
  </verify>
  <done>Batch loop refactored to use fetchBatchParallel, all existing logic preserved, compiles successfully</done>
</task>

<task type="auto">
  <name>Task 3: Add timing instrumentation</name>
  <files>pkg/comparator/comparator.go</files>
  <read_first>pkg/comparator/comparator.go (lines 473-500)</read_first>
  <action>
    Add performance timing logs per 05-CONTEXT.md decision 3:

    1. Function-level timing (add after line 475 `startTime := time.Now()`):
       ```go
       batchStartTime := time.Now()  // For batch-level timing
       var totalBatchTimeSerial, totalBatchTimeParallel int64
       var batchCount int
       ```

    2. Per-batch timing in fetchBatchParallel:
       - Record batch start time
       - Measure actual parallel execution time
       - Estimate serialized time (sum of individual query times)
       - Return timing info along with results

    3. Log output format (update progress line 647-651):
       ```go
       if cfg.ShowProgress {
           fmt.Printf("\r[%s] Source: %.1f%% (%d/%d) | Target: %.1f%% (%d/%d), 已耗时 %v, 本批：串行预估 %dms, 并行实际 %dms, 节省 %.1f%%",
               table,
               sourceProgress, processedSource, totalSource,
               targetProgress, processedTarget, totalTarget,
               elapsed, batchSerialMs, batchParallelMs, savingsPercent)
       }
       ```

    4. Function exit log (update line 735-738):
       ```go
       if cfg.ShowProgress {
           avgSavings := float64(totalBatchTimeSerial-totalBatchTimeParallel) / float64(totalBatchTimeSerial) * 100
           fmt.Printf("\r[%s] 比对完成，总耗时 %v (串行预估：%v, 并行实际：%v), 平均节省 %.1f%%\n",
               table, time.Since(startTime),
               time.Duration(totalBatchTimeSerial)*time.Millisecond,
               time.Duration(totalBatchTimeParallel)*time.Millisecond,
               avgSavings)
       }
       ```

    Timing tracking implementation:
    - Add batchStartTime, batchSerialEstimate, batchActualTime to fetchBatchParallel return values
    - Accumulate totals in compareTableDataFull loop
    - Calculate savings percentage for each batch and average
  </action>
  <verify>
    <automated>go build ./pkg/comparator/... succeeds</automated>
  </verify>
  <done>Timing instrumentation added with per-batch and total timing logs showing serial vs parallel comparison</done>
</task>

<task type="auto">
  <name>Task 4: Implement dual progress display</name>
  <files>pkg/comparator/comparator.go</files>
  <read_first>pkg/comparator/comparator.go (lines 550-560, 647-651)</read_first>
  <action>
    Refactor progress tracking to show dual progress per 05-CONTEXT.md decision 4:

    Current single progress (line 647-651):
    ```go
    if cfg.ShowProgress {
        progress := float64(processedCount) / float64(totalCount) * 100
        fmt.Printf("\r[%s] 进度：%.1f%% (%d/%d), 已耗时 %v", table, progress, processedCount, totalCount, elapsed)
    }
    ```

    New dual progress format:
    ```go
    fmt.Printf("\r[%s] Source: %.1f%% (%d/%d) | Target: %.1f%% (%d/%d), 已耗时 %v",
        table,
        sourceProgress, processedSource, totalSource,
        targetProgress, processedTarget, totalTarget,
        elapsed)
    ```

    Implementation changes:
    1. Split processedCount into processedSource and processedTarget counters
    2. Split totalCount into totalSource and totalTarget (from sourceMinMax.Count and targetMinMax.Count)
    3. Update progress calculation:
       - sourceProgress = float64(processedSource) / float64(totalSource) * 100
       - targetProgress = float64(processedTarget) / float64(totalTarget) * 100
    4. Track source and target records processed separately in the loop

    Progress update locations:
    - After fetchBatchParallel returns (update both source and target counters based on batch sizes)
    - Handle edge cases where one side is empty (show 100% or 0% appropriately)

    Example output:
    ```
    [users] Source: 75% (1500/2000) | Target: 45% (900/2000), 已耗时 2.3s
    ```
  </action>
  <verify>
    <automated>go build ./pkg/comparator/... succeeds</automated>
  </verify>
  <done>Dual progress display implemented showing Source: X% | Target: Y% format with independent counters</done>
</task>

</tasks>

<verification>
Before declaring plan complete:
- [ ] go build ./pkg/comparator/... succeeds (no compile errors)
- [ ] go test ./pkg/comparator/... passes (existing tests still pass)
- [ ] fetchBatchParallel uses goroutines for concurrent execution
- [ ] Context cancellation propagates on error (either failure cancels other)
- [ ] Timing logs show serial vs parallel time comparison
- [ ] Progress output shows dual progress format
- [ ] BatchSize config continues to work (no behavior change to batching logic)
</verification>

<success_criteria>
- All 4 tasks completed
- compareTableDataFull executes source and target queries in parallel
- Error handling uses context.WithCancel for fast fail
- Timing instrumentation shows performance improvement
- Dual progress display shows Source: X% | Target: Y%
- All existing v2.0 tests continue to pass
- Performance measurable: ~50% time reduction in 100ms network latency scenario
</success_criteria>

<output>
After completion, create `.planning/phases/05-parallel-query-optimization/05-01-SUMMARY.md`
</output>
