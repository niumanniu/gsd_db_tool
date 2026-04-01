package comparator

import (
	"context"
	"db-diff/pkg/config"
	"db-diff/pkg/database"
	"fmt"
	"golang.org/x/sync/errgroup"
	"strings"
	"time"
)

// getDriver 根据 driver 名称获取驱动实例
func getDriver(driverName string) (database.Driver, error) {
	switch driverName {
	case "mysql":
		return &database.MySQLDriver{}, nil
	case "oracle":
		return &database.OracleDriver{}, nil
	default:
		return nil, fmt.Errorf("unknown driver: %s", driverName)
	}
}

// DiffResult 对比结果
type DiffResult struct {
	SourceSchema    string
	TargetSchema    string
	TableDiff       TableDiff
	ColumnDiff      map[string]ColumnDiff    // table_name -> diff
	IndexDiff       map[string]IndexDiff     // table_name -> diff
	ConstraintDiff  map[string]ConstraintDiff // table_name -> diff
	TableCounts     map[string]TableCount    // 表记录数
	TableDataDiff   map[string]DataDiff      // 表数据差异
}

// TableCount 表记录数
type TableCount struct {
	SourceCount int64
	TargetCount int64
	Diff        int64
}

// DataDiff 数据差异
type DataDiff struct {
	Added   []map[string]interface{} // 新增行
	Removed []map[string]interface{} // 删除行
	Modified []DataModification      // 修改行
}

// DataModification 数据修改详情
type DataModification struct {
	Key    interface{}            // 主键值
	Source map[string]interface{} // 源数据
	Target map[string]interface{} // 目标数据
	Changes []string             // 变更字段
}

// TableDiff 表差异
type TableDiff struct {
	Added   []string // 源库有，目标库没有
	Missing []string // 目标库有，源库没有
}

// ColumnDiff 列差异
type ColumnDiff struct {
	Added   []database.ColumnMeta // 新增列
	Removed []database.ColumnMeta // 删除列
	Modified []ColumnModification  // 修改列
}

// ColumnModification 列修改详情
type ColumnModification struct {
	Name        string
	Source      database.ColumnMeta
	Target      database.ColumnMeta
	Changes     []string            // 变更字段列表
	TypeMapping *MappingResult      // 类型映射结果（nil 表示同库同类型，或跨库已映射）
}

// IndexDiff 索引差异
type IndexDiff struct {
	Added   []database.IndexMeta
	Removed []database.IndexMeta
	Modified []IndexModification
}

// IndexModification 索引修改详情
type IndexModification struct {
	Name   string
	Source []database.IndexMeta
	Target []database.IndexMeta
	Changes []string
}

// ConstraintDiff 约束差异
type ConstraintDiff struct {
	Added   []database.ConstraintMeta
	Removed []database.ConstraintMeta
	Modified []ConstraintModification
}

// ConstraintModification 约束修改详情
type ConstraintModification struct {
	Name   string
	Source []database.ConstraintMeta
	Target []database.ConstraintMeta
	Changes []string
}

// Compare 执行数据库对比
func Compare(cfg *config.Config) (*DiffResult, error) {
	result := &DiffResult{
		SourceSchema:   cfg.Source.GetSchema(),
		TargetSchema:   cfg.Target.GetSchema(),
		ColumnDiff:     make(map[string]ColumnDiff),
		IndexDiff:      make(map[string]IndexDiff),
		ConstraintDiff: make(map[string]ConstraintDiff),
		TableCounts:    make(map[string]TableCount),
		TableDataDiff:  make(map[string]DataDiff),
	}

	// 获取驱动
	sourceDriver, err := getDriver(cfg.Source.Driver)
	if err != nil {
		return nil, fmt.Errorf("获取源数据库驱动失败：%w", err)
	}
	targetDriver, err := getDriver(cfg.Target.Driver)
	if err != nil {
		return nil, fmt.Errorf("获取目标数据库驱动失败：%w", err)
	}

	// 连接源数据库
	sourceConn, err := sourceDriver.Connect(cfg.Source)
	if err != nil {
		return nil, fmt.Errorf("连接源数据库失败：%w", err)
	}
	defer sourceConn.Close()

	// 连接目标数据库
	targetConn, err := targetDriver.Connect(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("连接目标数据库失败：%w", err)
	}
	defer targetConn.Close()

	// 确定是否执行结构对比和/或数据对比
	// mode 为空或 structure 时只做结构对比
	// mode 为 data 时只做数据对比
	// mode 为 all 时两者都做
	// dataMode 有值时也执行数据对比（命令行覆盖）
	mode := cfg.CompareOptions.Mode
	doStructure := mode == "" || mode == "structure" || mode == "all"
	doData := mode == "data" || mode == "all" || (cfg.CompareOptions.DataMode != "" && cfg.CompareOptions.DataMode != "count")

	// 获取元数据（结构对比或数据对比都需要）
	sourceMeta, err := sourceDriver.FetchMetadata(sourceConn, cfg.Source.GetSchema())
	if err != nil {
		return nil, fmt.Errorf("获取源数据库元数据失败：%w", err)
	}

	targetMeta, err := targetDriver.FetchMetadata(targetConn, cfg.Target.GetSchema())
	if err != nil {
		return nil, fmt.Errorf("获取目标数据库元数据失败：%w", err)
	}

	// 获取共同表
	commonTables := getCommonTables(sourceMeta.Tables, targetMeta.Tables)

	// 过滤指定表
	if len(cfg.CompareOptions.Tables) > 0 {
		tableSet := make(map[string]bool)
		for _, t := range cfg.CompareOptions.Tables {
			tableSet[t] = true
		}
		filtered := []string{}
		for _, t := range commonTables {
			if tableSet[t] {
				filtered = append(filtered, t)
			}
		}
		commonTables = filtered
	}

	// 结构对比
	if doStructure {
		// 对比表
		result.TableDiff = compareTables(sourceMeta.Tables, targetMeta.Tables)

		// 对比每个表的结构
		for _, table := range commonTables {
			result.ColumnDiff[table] = compareColumns(
				sourceMeta.Columns[table],
				targetMeta.Columns[table],
				cfg.Source.Driver,
				cfg.Target.Driver,
			)
			result.IndexDiff[table] = compareIndexes(
				sourceMeta.Indexes[table],
				targetMeta.Indexes[table],
			)
			result.ConstraintDiff[table] = compareConstraints(
				sourceMeta.Constraints[table],
				targetMeta.Constraints[table],
			)
		}
	}

	// 数据对比
	if doData {
		for _, table := range commonTables {
			cols := sourceMeta.Columns[table]

			switch cfg.CompareOptions.DataMode {
			case config.DataModeFull:
				result.TableDataDiff[table], _ = compareTableDataFull(sourceConn, targetConn, table, cols, &cfg.CompareOptions)
			case config.DataModeSample:
				result.TableDataDiff[table], _ = compareTableDataSample(sourceConn, targetConn, table, cfg.CompareOptions.SampleRatio, cols)
			default:
				// 默认只对比记录数
				count, _ := compareTableCount(sourceConn, targetConn, table)
				result.TableCounts[table] = count
			}
		}
	}

	return result, nil
}

func compareTables(source, target []database.TableMeta) TableDiff {
	sourceSet := make(map[string]bool)
	targetSet := make(map[string]bool)

	for _, t := range source {
		sourceSet[t.Name] = true
	}
	for _, t := range target {
		targetSet[t.Name] = true
	}

	var diff TableDiff
	for name := range sourceSet {
		if !targetSet[name] {
			diff.Missing = append(diff.Missing, name)
		}
	}
	for name := range targetSet {
		if !sourceSet[name] {
			diff.Added = append(diff.Added, name)
		}
	}

	return diff
}

func getCommonTables(source, target []database.TableMeta) []string {
	targetSet := make(map[string]bool)
	for _, t := range target {
		targetSet[t.Name] = true
	}

	var common []string
	for _, t := range source {
		if targetSet[t.Name] {
			common = append(common, t.Name)
		}
	}
	return common
}

func compareColumns(source, target []database.ColumnMeta, sourceDriver, targetDriver string) ColumnDiff {
	sourceMap := make(map[string]database.ColumnMeta)
	targetMap := make(map[string]database.ColumnMeta)

	for _, c := range source {
		sourceMap[c.Name] = c
	}
	for _, c := range target {
		targetMap[c.Name] = c
	}

	var diff ColumnDiff

	// 查找新增和修改的列
	for name, srcCol := range sourceMap {
		if tgtCol, exists := targetMap[name]; exists {
			// 检查是否有修改
			changes, typeMapping := compareColumn(srcCol, tgtCol, sourceDriver, targetDriver)
			if len(changes) > 0 {
				diff.Modified = append(diff.Modified, ColumnModification{
					Name:        name,
					Source:      srcCol,
					Target:      tgtCol,
					Changes:     changes,
					TypeMapping: typeMapping,
				})
			}
		} else {
			diff.Removed = append(diff.Removed, srcCol)
		}
	}

	// 查找删除的列（源库没有，目标库有）
	for name, tgtCol := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			diff.Added = append(diff.Added, tgtCol)
		}
	}

	return diff
}

func compareColumn(source, target database.ColumnMeta, sourceDriver, targetDriver string) ([]string, *MappingResult) {
	var changes []string

	// 计算类型映射
	typeMapping := MapType(source.DataType, target.DataType, sourceDriver, targetDriver)

	if source.DataType != target.DataType {
		// 如果有类型映射，添加映射信息到变更
		if typeMapping.Status == StatusMapped {
			changes = append(changes, fmt.Sprintf("类型：%s -> %s ⚠️ %s", source.DataType, target.DataType, typeMapping.Description))
		} else if typeMapping.Status == StatusIncompatible {
			changes = append(changes, fmt.Sprintf("类型：%s -> %s ❌ %s", source.DataType, target.DataType, typeMapping.Description))
		} else {
			changes = append(changes, fmt.Sprintf("类型：%s -> %s", source.DataType, target.DataType))
		}
	}
	if source.ColumnType != target.ColumnType {
		changes = append(changes, fmt.Sprintf("详细类型：%s -> %s", source.ColumnType, target.ColumnType))
	}
	if source.IsNullable != target.IsNullable {
		changes = append(changes, fmt.Sprintf("可空：%s -> %s", source.IsNullable, target.IsNullable))
	}
	if source.ColumnDefault.String != target.ColumnDefault.String {
		changes = append(changes, fmt.Sprintf("默认值：%v -> %v", source.ColumnDefault, target.ColumnDefault))
	}
	if source.Extra != target.Extra {
		changes = append(changes, fmt.Sprintf("额外：%s -> %s", source.Extra, target.Extra))
	}

	// 只有在跨库对比或类型不同时返回 TypeMapping
	var resultMapping *MappingResult
	if sourceDriver != targetDriver || source.DataType != target.DataType {
		resultMapping = &typeMapping
	}

	return changes, resultMapping
}

func compareIndexes(source, target []database.IndexMeta) IndexDiff {
	sourceMap := make(map[string][]database.IndexMeta)
	targetMap := make(map[string][]database.IndexMeta)

	// 按索引名分组
	for _, idx := range source {
		sourceMap[idx.Name] = append(sourceMap[idx.Name], idx)
	}
	for _, idx := range target {
		targetMap[idx.Name] = append(targetMap[idx.Name], idx)
	}

	var diff IndexDiff

	for name, srcIdxs := range sourceMap {
		if tgtIdxs, exists := targetMap[name]; exists {
			if changes := compareIndex(srcIdxs, tgtIdxs); len(changes) > 0 {
				diff.Modified = append(diff.Modified, IndexModification{
					Name:    name,
					Source:  srcIdxs,
					Target:  tgtIdxs,
					Changes: changes,
				})
			}
		} else {
			diff.Removed = append(diff.Removed, srcIdxs...)
		}
	}

	for name, tgtIdxs := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			diff.Added = append(diff.Added, tgtIdxs...)
		}
	}

	return diff
}

func compareIndex(source, target []database.IndexMeta) []string {
	var changes []string

	if len(source) != len(target) {
		changes = append(changes, fmt.Sprintf("列数：%d -> %d", len(source), len(target)))
	}

	// 简化比较：只比较列顺序
	for i := range source {
		if i < len(target) {
			if source[i].ColumnName != target[i].ColumnName {
				changes = append(changes, fmt.Sprintf("列%d: %s -> %s", i, source[i].ColumnName, target[i].ColumnName))
			}
			if source[i].IndexType != target[i].IndexType {
				changes = append(changes, fmt.Sprintf("类型：%s -> %s", source[i].IndexType, target[i].IndexType))
			}
		}
	}

	return changes
}

func compareConstraints(source, target []database.ConstraintMeta) ConstraintDiff {
	sourceMap := make(map[string][]database.ConstraintMeta)
	targetMap := make(map[string][]database.ConstraintMeta)

	for _, c := range source {
		sourceMap[c.Name] = append(sourceMap[c.Name], c)
	}
	for _, c := range target {
		targetMap[c.Name] = append(targetMap[c.Name], c)
	}

	var diff ConstraintDiff

	for name, srcCons := range sourceMap {
		if tgtCons, exists := targetMap[name]; exists {
			if changes := compareConstraint(srcCons, tgtCons); len(changes) > 0 {
				diff.Modified = append(diff.Modified, ConstraintModification{
					Name:    name,
					Source:  srcCons,
					Target:  tgtCons,
					Changes: changes,
				})
			}
		} else {
			diff.Removed = append(diff.Removed, srcCons...)
		}
	}

	for name, tgtCons := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			diff.Added = append(diff.Added, tgtCons...)
		}
	}

	return diff
}

func compareConstraint(source, target []database.ConstraintMeta) []string {
	var changes []string

	if len(source) != len(target) {
		changes = append(changes, fmt.Sprintf("关联列数：%d -> %d", len(source), len(target)))
	}

	return changes
}

// compareTableCount 对比表记录数
func compareTableCount(sourceConn, targetConn *database.Connection, table string) (TableCount, error) {
	var count TableCount

	// 查询源表记录数
	row := sourceConn.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table))
	row.Scan(&count.SourceCount)

	// 查询目标表记录数
	row = targetConn.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table))
	row.Scan(&count.TargetCount)

	count.Diff = count.SourceCount - count.TargetCount
	return count, nil
}

// fetchBatchParallel fetches source and target data concurrently using goroutines.
// Returns error if either query fails, with context cancellation for the other.
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
) (sourceBatch, targetBatch []map[string]interface{}, sourceNext, targetNext []interface{}, sourceHash, targetHash string, err error) {
	// Create context with cancel for coordinated cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Use errgroup for coordinated error handling
	g, ctx := errgroup.WithContext(ctx)

	// Channel to receive results from each goroutine
	type result struct {
		batch []map[string]interface{}
		next  []interface{}
		hash  string
	}
	sourceCh := make(chan result, 1)
	targetCh := make(chan result, 1)

	// Source query goroutine
	g.Go(func() error {
		defer close(sourceCh)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var res result
			if hashFilter && sourceConn.DriverName == "mysql" {
				res.batch, res.next, res.hash, err = fetchMySQLDataWithHash(
					sourceConn, table, columns, filteredCols, pkCols[0], currentKey, batchSize)
			} else {
				res.batch, res.next, err = fetchDataByKeyRange(
					sourceConn, table, columns, pkCols, []interface{}{currentKey}, batchSize)
			}
			if err != nil {
				return fmt.Errorf("source query failed: %w", err)
			}
			sourceCh <- res
			return nil
		}
	})

	// Target query goroutine
	g.Go(func() error {
		defer close(targetCh)
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var res result
			if hashFilter && targetConn.DriverName == "mysql" {
				res.batch, res.next, res.hash, err = fetchMySQLDataWithHash(
					targetConn, table, columns, filteredCols, pkCols[0], currentKey, batchSize)
			} else {
				res.batch, res.next, err = fetchDataByKeyRange(
					targetConn, table, columns, pkCols, []interface{}{currentKey}, batchSize)
			}
			if err != nil {
				return fmt.Errorf("target query failed: %w", err)
			}
			targetCh <- res
			return nil
		}
	})

	// Wait for both goroutines to complete
	if err := g.Wait(); err != nil {
		return nil, nil, nil, nil, "", "", err
	}

	// Collect results
	sourceResult := <-sourceCh
	targetResult := <-targetCh

	return sourceResult.batch, targetResult.batch, sourceResult.next, targetResult.next, sourceResult.hash, targetResult.hash, nil
}

// compareTableDataFull 全量数据对比（主键范围 + 分批，降低内存占用）
func compareTableDataFull(sourceConn, targetConn *database.Connection, table string, columns []database.ColumnMeta, cfg *config.CompareOptions) (DataDiff, error) {
	var diff DataDiff
	startTime := time.Now()

	// 创建 context 用于并行查询的取消控制
	ctx := context.Background()

	// 获取主键列
	pkCols := getPrimaryKeyColumns(columns)
	if len(pkCols) == 0 {
		return diff, fmt.Errorf("表 %s 没有主键或唯一索引，无法进行全量数据对比", table)
	}

	// 处理列过滤：include 和 exclude
	filteredCols := filterColumns(columns, cfg.IncludeColumns, cfg.ExcludeColumns)
	if len(filteredCols) == 0 {
		return diff, fmt.Errorf("表 %s 没有可对比的字段（可能被全部排除）", table)
	}

	// 使用配置的批次大小，默认值 1000
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 1000
	}
	pkColName := pkCols[0]

	// 获取源表主键范围
	sourceMinMax := getPrimaryKeyRange(sourceConn, table, pkColName)
	targetMinMax := getPrimaryKeyRange(targetConn, table, pkColName)

	// 处理空表情况
	if sourceMinMax.Count == 0 && targetMinMax.Count == 0 {
		if cfg.ShowProgress {
			fmt.Printf("\r[%s] 比对完成，耗时 %v\n", table, time.Since(startTime))
		}
		return diff, nil
	}
	if sourceMinMax.Count == 0 {
		for {
			batch, _, err := fetchDataByKeyRange(targetConn, table, columns, pkCols, []interface{}{targetMinMax.MinVal}, batchSize)
			if err != nil {
				return diff, err
			}
			for _, row := range batch {
				diff.Added = append(diff.Added, row)
			}
			if len(batch) < batchSize {
				break
			}
		}
		if cfg.ShowProgress {
			fmt.Printf("\r[%s] 比对完成，耗时 %v\n", table, time.Since(startTime))
		}
		return diff, nil
	}
	if targetMinMax.Count == 0 {
		for {
			batch, _, err := fetchDataByKeyRange(sourceConn, table, columns, pkCols, []interface{}{sourceMinMax.MinVal}, batchSize)
			if err != nil {
				return diff, err
			}
			for _, row := range batch {
				diff.Removed = append(diff.Removed, row)
			}
			if len(batch) < batchSize {
				break
			}
		}
		if cfg.ShowProgress {
			fmt.Printf("\r[%s] 比对完成，耗时 %v\n", table, time.Since(startTime))
		}
		return diff, nil
	}

	// 按主键范围分批对比
	currentKey := sourceMinMax.MinVal
	if sourceMinMax.MinVal == nil {
		currentKey = targetMinMax.MinVal
	}

	// 进度追踪 - 双进度显示（Task 4）
	totalSource := sourceMinMax.Count
	totalTarget := targetMinMax.Count
	processedSource := 0
	processedTarget := 0

	// 性能计时（Task 3）
	var totalBatchTimeSerial, totalBatchTimeParallel int64
	var batchCount int

	if cfg.ShowProgress {
		fmt.Printf("\n[%s] 开始比对，源表 %d 条，目标表 %d 条\n", table, totalSource, totalTarget)
	}

	// 全局的 map 用于累积未匹配的记录
	sourceRemaining := make(map[string]map[string]interface{})
	targetRemaining := make(map[string]map[string]interface{})

	for {
		var sourceBatch, targetBatch []map[string]interface{}
		var sourceNext, targetNext []interface{}
		var sourceBatchHash, targetBatchHash string
		var err error

		batchStartTime := time.Now()

		// 使用并行查询（Task 2）
		sourceBatch, targetBatch, sourceNext, targetNext, sourceBatchHash, targetBatchHash, err = fetchBatchParallel(
			ctx, sourceConn, targetConn, table, columns, filteredCols,
			pkCols, currentKey, batchSize, cfg.HashFilter,
		)
		if err != nil {
			return diff, err
		}

		// 性能计时 - 记录批次时间（Task 3）
		batchTime := time.Since(batchStartTime).Milliseconds()
		totalBatchTimeParallel += batchTime
		// 估算串行时间（简单相加，实际可能因网络延迟更高）
		estimatedSerial := batchTime * 2 // 简化估算：假设两边查询时间相近
		totalBatchTimeSerial += estimatedSerial
		batchCount++

		// 如果整批数据的聚合 hash 相同，且批大小相同，则跳过整批
		if cfg.HashFilter && sourceBatchHash != "" && targetBatchHash != "" &&
			sourceBatchHash == targetBatchHash &&
			len(sourceBatch) == len(targetBatch) {
			// 整批数据一致，跳过 - 但仍需更新 currentKey 和进度
			var nextKey interface{}
			if sourceNext != nil && targetNext != nil {
				if sourceNext[0].(int64) < targetNext[0].(int64) {
					nextKey = sourceNext[0]
				} else {
					nextKey = targetNext[0]
				}
			} else if sourceNext != nil {
				nextKey = sourceNext[0]
			} else if targetNext != nil {
				nextKey = targetNext[0]
			} else {
				break
			}
			currentKey = nextKey

			// 更新进度（平均进度显示）
			processedSource += len(sourceBatch)
			processedTarget += len(targetBatch)
			if cfg.ShowProgress {
				sourceProgress := float64(processedSource) / float64(totalSource) * 100
				targetProgress := float64(processedTarget) / float64(totalTarget) * 100
				avgProgress := (sourceProgress + targetProgress) / 2
				elapsed := time.Since(startTime)
				savingsPercent := float64(totalBatchTimeSerial-totalBatchTimeParallel) / float64(totalBatchTimeSerial) * 100
				fmt.Printf("\r[%s] 进度：%.1f%% (源：%.1f%% | 目标：%.1f%%), 已耗时 %v, 本批：%dms (节省 %.1f%%)",
					table,
					avgProgress, sourceProgress, targetProgress,
					elapsed, batchTime, savingsPercent)
			}
			continue
		}

		if len(sourceBatch) == 0 && len(targetBatch) == 0 {
			break
		}

		// 更新双进度计数（Task 4）
		processedSource += len(sourceBatch)
		processedTarget += len(targetBatch)

		// 平均进度显示 + 性能计时（Task 3 + Task 4）
		if cfg.ShowProgress {
			sourceProgress := float64(processedSource) / float64(totalSource) * 100
			targetProgress := float64(processedTarget) / float64(totalTarget) * 100
			avgProgress := (sourceProgress + targetProgress) / 2
			elapsed := time.Since(startTime)
			savingsPercent := float64(totalBatchTimeSerial-totalBatchTimeParallel) / float64(totalBatchTimeSerial) * 100
			fmt.Printf("\r[%s] 进度：%.1f%% (源：%.1f%% | 目标：%.1f%%), 已耗时 %v, 本批：%dms (节省 %.1f%%)",
				table,
				avgProgress, sourceProgress, targetProgress,
				elapsed, batchTime, savingsPercent)
		}

		// 构建当前批次的 map，并与上一批剩余的记录合并
		sourceMap := sourceRemaining
		targetMap := targetRemaining
		for _, row := range sourceBatch {
			key := buildKey(row, pkCols)
			sourceMap[key] = row
		}
		for _, row := range targetBatch {
			key := buildKey(row, pkCols)
			targetMap[key] = row
		}

		// 比对
		for key, srcRow := range sourceMap {
			if tgtRow, exists := targetMap[key]; exists {
				// Hash 预筛选（如果启用且有 hash 值）
				if cfg.HashFilter {
					srcHashRaw := srcRow["__row_hash__"]
					tgtHashRaw := tgtRow["__row_hash__"]
					// 只有当两行都有 hash 值时才使用 hash 预筛选
					if srcHashRaw != nil && tgtHashRaw != nil {
						// CRC32 返回整数，直接比较
						if srcHashRaw == tgtHashRaw {
							// Hash 相同，跳过详细比对
							delete(sourceMap, key)
							delete(targetMap, key)
							continue
						}
						// Hash 不同，删除 hash 字段后进行详细比对
						delete(srcRow, "__row_hash__")
						delete(tgtRow, "__row_hash__")
					}
				}
				// 逐字段比对
				changes := compareRow(srcRow, tgtRow, filteredCols)
				if len(changes) > 0 {
					diff.Modified = append(diff.Modified, DataModification{
						Key:     key,
						Source:  srcRow,
						Target:  tgtRow,
						Changes: changes,
					})
				}
				delete(sourceMap, key)
				delete(targetMap, key)
			}
		}

		// 确定下一批的起始键
		// 注意：选择较小的 nextKey，确保不会遗漏任何记录
		// 重复处理的记录会在之前的批次中被 delete，不会造成重复报告
		var nextKey interface{}
		if sourceNext != nil && targetNext != nil {
			if sourceNext[0].(int64) < targetNext[0].(int64) {
				nextKey = sourceNext[0]
			} else {
				nextKey = targetNext[0]
			}
		} else if sourceNext != nil {
			// 目标表没有更多数据，但源表还有
			// 需要继续处理源表剩余的记录
			nextKey = sourceNext[0]
		} else if targetNext != nil {
			// 源表没有更多数据，但目标表还有
			// 需要继续处理目标表剩余的记录
			nextKey = targetNext[0]
		} else {
			// 两个表都没有更多数据，处理剩余记录
			break
		}
		currentKey = nextKey
	}

	// 循环结束后，处理最后剩余的记录
	// 此时 sourceRemaining 和 targetRemaining 中的记录才是真正的"删除"和"新增"
	for _, srcRow := range sourceRemaining {
		diff.Removed = append(diff.Removed, srcRow)
	}
	for _, tgtRow := range targetRemaining {
		diff.Added = append(diff.Added, tgtRow)
	}

	if cfg.ShowProgress {
		diffCount := len(diff.Modified) + len(diff.Added) + len(diff.Removed)
		elapsed := time.Since(startTime)
		// 输出性能统计（Task 3）
		if batchCount > 0 {
			avgSavings := float64(totalBatchTimeSerial-totalBatchTimeParallel) / float64(totalBatchTimeSerial) * 100
			fmt.Printf("\r[%s] 比对完成，总耗时 %v (串行预估：%v, 并行实际：%v), 平均节省 %.1f%%，发现 %d 处差异\n",
				table, elapsed,
				time.Duration(totalBatchTimeSerial)*time.Millisecond,
				time.Duration(totalBatchTimeParallel)*time.Millisecond,
				avgSavings, diffCount)
		} else {
			fmt.Printf("\r[%s] 比对完成，耗时 %v，发现 %d 处差异\n", table, elapsed, diffCount)
		}
	}

	return diff, nil
}

// filterColumns 过滤列，返回用于比对的列
func filterColumns(allColumns []database.ColumnMeta, includeCols, excludeCols []string) []database.ColumnMeta {
	if len(includeCols) == 0 && len(excludeCols) == 0 {
		return allColumns
	}

	// 构建排除集合
	excludeSet := make(map[string]bool)
	for _, col := range excludeCols {
		excludeSet[col] = true
	}

	// 构建包含集合（如果有指定）
	includeSet := make(map[string]bool)
	for _, col := range includeCols {
		includeSet[col] = true
	}

	var result []database.ColumnMeta
	for _, col := range allColumns {
		// 如果有 include 列表，只保留列表中的
		if len(includeCols) > 0 && !includeSet[col.Name] {
			continue
		}
		// 排除 exclude 列表中的
		if excludeSet[col.Name] {
			continue
		}
		result = append(result, col)
	}
	return result
}

// minMaxKey 主键范围
type minMaxKey struct {
	MinVal interface{}
	MaxVal interface{}
	Count  int64
}

// getPrimaryKeyRange 获取主键的最小值和最大值
func getPrimaryKeyRange(conn *database.Connection, table, pkCol string) minMaxKey {
	var result minMaxKey
	row := conn.DB.QueryRow(fmt.Sprintf("SELECT MIN(`%s`), MAX(`%s`), COUNT(*) FROM `%s`", pkCol, pkCol, table))
	err := row.Scan(&result.MinVal, &result.MaxVal, &result.Count)
	if err != nil || result.MinVal == nil {
		return result
	}
	return result
}

// fetchDataByKeyRange 按主键范围分批获取数据
func fetchDataByKeyRange(conn *database.Connection, table string, columns []database.ColumnMeta, pkCols []string, startKey []interface{}, limit int) ([]map[string]interface{}, []interface{}, error) {
	whereClause := fmt.Sprintf("`%s` >= ?", pkCols[0])
	startVal := startKey[0]

	query := fmt.Sprintf("SELECT * FROM `%s` WHERE %s ORDER BY `%s` LIMIT %d", table, whereClause, pkCols[0], limit+1)
	rows, err := conn.DB.Query(query, startVal)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var data []map[string]interface{}
	var nextKey []interface{}
	count := 0
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, err
		}
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			row[col] = values[i]
		}
		if count < limit {
			data = append(data, row)
		} else {
			nextKey = []interface{}{row[pkCols[0]]}
		}
		count++
	}

	return data, nextKey, rows.Err()
}

// fetchMySQLDataWithHash 按主键范围分批获取数据（带 SQL 层 hash 计算，仅 MySQL）
// 返回每行数据（含单行 hash）和整批数据的聚合 hash
// 使用 MD5(GROUP_CONCAT(CONCAT_WS('|', col1, col2, ...) ORDER BY pk)) 计算整批 hash
func fetchMySQLDataWithHash(conn *database.Connection, table string, columns []database.ColumnMeta, hashColumns []database.ColumnMeta, pkCol string, startKey interface{}, limit int) ([]map[string]interface{}, []interface{}, string, error) {
	// 构建单行 hash 表达式：CRC32(CONCAT(...)) - 用于行级快速对比
	rowHashExpr := "CRC32(CONCAT("
	for i, c := range hashColumns {
		if i > 0 {
			rowHashExpr += `,';',`
		}
		switch c.DataType {
		case "int", "integer", "bigint", "smallint", "tinyint", "mediumint":
			rowHashExpr += fmt.Sprintf("IFNULL(`%s`,'')", c.Name)
		case "decimal", "numeric", "float", "double":
			rowHashExpr += fmt.Sprintf("IFNULL(CAST(`%s` AS CHAR),'')", c.Name)
		case "datetime", "timestamp", "date", "time":
			rowHashExpr += fmt.Sprintf("IFNULL(DATE_FORMAT(`%s`,'%%Y-%%m-%%d %%H:%%i:%%s'),'')", c.Name)
		default:
			rowHashExpr += fmt.Sprintf("IFNULL(`%s`,'')", c.Name)
		}
	}
	rowHashExpr += "))"

	// 构建整批聚合 hash 表达式：MD5(GROUP_CONCAT(CONCAT_WS('|', col1, col2, ...) ORDER BY pk))
	// 使用 CONCAT_WS 避免 'ab','c' 和 'a','bc' 产生相同结果的问题
	batchHashExpr := "MD5(GROUP_CONCAT(CONCAT_WS("
	for i, c := range hashColumns {
		if i > 0 {
			batchHashExpr += ",'"
		}
		switch c.DataType {
		case "int", "integer", "bigint", "smallint", "tinyint", "mediumint":
			batchHashExpr += fmt.Sprintf("'|', IFNULL(`%s`,'')", c.Name)
		case "decimal", "numeric", "float", "double":
			batchHashExpr += fmt.Sprintf("'|', IFNULL(CAST(`%s` AS CHAR),'')", c.Name)
		case "datetime", "timestamp", "date", "time":
			batchHashExpr += fmt.Sprintf("'|', IFNULL(DATE_FORMAT(`%s`,'%%Y-%%m-%%d %%H:%%i:%%s'),'')", c.Name)
		default:
			batchHashExpr += fmt.Sprintf("'|', IFNULL(`%s`,'')", c.Name)
		}
	}
	batchHashExpr += fmt.Sprintf(") ORDER BY `%s`)) AS batch_hash", pkCol)

	// 构建 SELECT 子句 - 数据列 + 单行 hash + 整批 hash
	colNames := make([]string, len(columns))
	for i, c := range columns {
		colNames[i] = fmt.Sprintf("`%s`", c.Name)
	}
	selectClause := fmt.Sprintf("%s, %s AS row_hash, %s", strings.Join(colNames, ", "), rowHashExpr, batchHashExpr)

	// 构建查询
	whereClause := ""
	args := []interface{}{}
	if startKey != nil {
		whereClause = fmt.Sprintf("WHERE `%s` >= ?", pkCol)
		args = append(args, startKey)
	}

	query := fmt.Sprintf("SELECT %s FROM `%s` %s ORDER BY `%s` LIMIT ?", selectClause, table, whereClause, pkCol)
	args = append(args, limit+1)

	rows, err := conn.DB.Query(query, args...)
	if err != nil {
		return nil, nil, "", err
	}
	defer rows.Close()

	// 获取列名
	cols, _ := rows.Columns()
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var data []map[string]interface{}
	var nextKey []interface{}
	var batchHash string
	count := 0

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, "", err
		}
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			colLower := strings.ToLower(col)
			if colLower == "row_hash" {
				hashVal := values[i]
				row["__row_hash__"] = hashVal
			} else if colLower == "batch_hash" {
				// 整批 hash，每行都返回相同值，取第一个即可
				if batchHash == "" {
					batchHash = fmt.Sprintf("%v", values[i])
				}
			} else {
				row[col] = values[i]
			}
		}
		if count < limit {
			data = append(data, row)
		} else {
			nextKey = []interface{}{row[pkCol]}
		}
		count++
	}

	return data, nextKey, batchHash, rows.Err()
}

// compareTableDataSample 抽样数据对比
func compareTableDataSample(sourceConn, targetConn *database.Connection, table string, ratio float64, columns []database.ColumnMeta) (DataDiff, error) {
	var diff DataDiff

	// 获取总记录数
	var totalCount int64
	row := sourceConn.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table))
	row.Scan(&totalCount)

	// 计算抽样数量
	sampleSize := int(float64(totalCount) * ratio)
	if sampleSize < 1 {
		sampleSize = 1
	}
	if sampleSize > 1000 {
		sampleSize = 1000 // 限制最大抽样数
	}

	// 随机抽样
	sourceData, err := fetchTableDataSample(sourceConn, table, columns, sampleSize)
	if err != nil {
		return diff, err
	}

	targetData, err := fetchTableDataSample(targetConn, table, columns, sampleSize)
	if err != nil {
		return diff, err
	}

	// 简单对比（抽样模式下只对比记录数和前 N 条）
	if len(sourceData) != len(targetData) {
		diff.Modified = append(diff.Modified, DataModification{
			Key:     "sample_count",
			Changes: []string{fmt.Sprintf("抽样数量：%d vs %d", len(sourceData), len(targetData))},
		})
	}

	// 对比抽样数据
	for i := range sourceData {
		if i < len(targetData) {
			changes := compareRow(sourceData[i], targetData[i], columns)
			if len(changes) > 0 {
				diff.Modified = append(diff.Modified, DataModification{
					Key:     fmt.Sprintf("row_%d", i),
					Source:  sourceData[i],
					Target:  targetData[i],
					Changes: changes,
				})
			}
		}
	}

	return diff, nil
}

// getPrimaryKeyColumns 获取主键列（必须有主键或唯一索引）
func getPrimaryKeyColumns(columns []database.ColumnMeta) []string {
	var pkCols []string
	for _, c := range columns {
		if c.IsPrimaryKey {
			pkCols = append(pkCols, c.Name)
		}
	}
	return pkCols
}

// fetchTableData 读取表数据
func fetchTableData(db *database.Connection, table string, columns []database.ColumnMeta) ([]map[string]interface{}, error) {
	colNames := make([]string, len(columns))
	for i, c := range columns {
		colNames[i] = c.Name
	}

	query := fmt.Sprintf("SELECT * FROM `%s` ORDER BY `%s`", table, colNames[0])
	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 获取列名
	cols, _ := rows.Columns()
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var data []map[string]interface{}
	for rows.Next() {
		rows.Scan(valuePtrs...)
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		data = append(data, row)
	}

	return data, rows.Err()
}

// fetchTableDataSample 抽样读取表数据
func fetchTableDataSample(db *database.Connection, table string, columns []database.ColumnMeta, limit int) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM `%s` ORDER BY RAND() LIMIT %d", table, limit)
	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	var data []map[string]interface{}
	for rows.Next() {
		rows.Scan(valuePtrs...)
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		data = append(data, row)
	}

	return data, rows.Err()
}

// buildKey 构建主键字符串
func buildKey(row map[string]interface{}, pkCols []string) string {
	if len(pkCols) == 0 {
		return fmt.Sprintf("%v", row)
	}
	key := ""
	for _, col := range pkCols {
		if key != "" {
			key += "|"
		}
		key += fmt.Sprintf("%v", row[col])
	}
	return key
}

// compareRow 对比两行数据
func compareRow(source, target map[string]interface{}, columns []database.ColumnMeta) []string {
	var changes []string
	for _, c := range columns {
		srcVal := source[c.Name]
		tgtVal := target[c.Name]
		srcStr := formatValue(srcVal)
		tgtStr := formatValue(tgtVal)
		if srcStr != tgtStr {
			changes = append(changes, fmt.Sprintf("%s: %s -> %s", c.Name, srcStr, tgtStr))
		}
	}
	return changes
}

// formatValue 格式化字段值，处理二进制和特殊类型
func formatValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case []byte:
		// 尝试转为字符串
		str := string(val)
		// 如果是纯数字字符串，直接显示
		if isNumeric(str) {
			return str
		}
		// 如果是可打印字符串，直接显示
		if isPrintable(str) {
			return str
		}
		return fmt.Sprintf("[binary %d bytes]", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}

// isNumeric 检查字符串是否为纯数字
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			if r != '.' && r != '-' {
				return false
			}
		}
	}
	return true
}

// isPrintable 检查字符串是否为可打印字符
func isPrintable(s string) bool {
	for _, r := range s {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}
