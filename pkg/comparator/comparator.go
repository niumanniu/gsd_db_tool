package comparator

import (
	"db-diff/pkg/config"
	"db-diff/pkg/database"
	"fmt"
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

	// 获取元数据（使用 GetSchema() 获取 schema）
	sourceMeta, err := sourceDriver.FetchMetadata(sourceConn, cfg.Source.GetSchema())
	if err != nil {
		return nil, fmt.Errorf("获取源数据库元数据失败：%w", err)
	}

	targetMeta, err := targetDriver.FetchMetadata(targetConn, cfg.Target.GetSchema())
	if err != nil {
		return nil, fmt.Errorf("获取目标数据库元数据失败：%w", err)
	}

	// 对比表
	result.TableDiff = compareTables(sourceMeta.Tables, targetMeta.Tables)

	// 获取共同表（只对比两个库都存在的表）
	commonTables := getCommonTables(sourceMeta.Tables, targetMeta.Tables)

	// 过滤指定表
	if len(cfg.Tables) > 0 {
		tableSet := make(map[string]bool)
		for _, t := range cfg.Tables {
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

	// 数据对比
	if cfg.DataMode != "" {
		for _, table := range commonTables {
			switch cfg.DataMode {
			case config.DataModeCount:
				result.TableCounts[table], _ = compareTableCount(sourceConn, targetConn, table)
			case config.DataModeFull:
				result.TableDataDiff[table], _ = compareTableDataFull(sourceConn, targetConn, table, sourceMeta.Columns[table])
			case config.DataModeSample:
				result.TableDataDiff[table], _ = compareTableDataSample(sourceConn, targetConn, table, cfg.SampleRatio, sourceMeta.Columns[table])
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

// compareTableDataFull 全量数据对比
func compareTableDataFull(sourceConn, targetConn *database.Connection, table string, columns []database.ColumnMeta) (DataDiff, error) {
	var diff DataDiff

	// 获取主键列
	pkCols := getPrimaryKeyColumns(columns)
	if len(pkCols) == 0 {
		// 没有主键，使用所有列
		for _, c := range columns {
			pkCols = append(pkCols, c.Name)
		}
	}

	// 读取源表数据
	sourceData, err := fetchTableData(sourceConn, table, columns)
	if err != nil {
		return diff, err
	}

	// 读取目标表数据
	targetData, err := fetchTableData(targetConn, table, columns)
	if err != nil {
		return diff, err
	}

	// 构建主键索引
	sourceMap := make(map[string]map[string]interface{})
	targetMap := make(map[string]map[string]interface{})

	for _, row := range sourceData {
		key := buildKey(row, pkCols)
		sourceMap[key] = row
	}
	for _, row := range targetData {
		key := buildKey(row, pkCols)
		targetMap[key] = row
	}

	// 查找差异
	for key, srcRow := range sourceMap {
		if tgtRow, exists := targetMap[key]; exists {
			// 检查是否有修改
			changes := compareRow(srcRow, tgtRow, columns)
			if len(changes) > 0 {
				diff.Modified = append(diff.Modified, DataModification{
					Key:     key,
					Source:  srcRow,
					Target:  tgtRow,
					Changes: changes,
				})
			}
		} else {
			diff.Removed = append(diff.Removed, srcRow)
		}
	}

	for key, tgtRow := range targetMap {
		if _, exists := sourceMap[key]; !exists {
			diff.Added = append(diff.Added, tgtRow)
		}
	}

	return diff, nil
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

// getPrimaryKeyColumns 获取主键列
func getPrimaryKeyColumns(columns []database.ColumnMeta) []string {
	var pkCols []string
	for _, c := range columns {
		if c.Extra == "auto_increment" {
			pkCols = append(pkCols, c.Name)
			break
		}
	}
	if len(pkCols) == 0 {
		// 尝试使用第一个字段作为主键
		for _, c := range columns {
			if c.Name == "id" {
				pkCols = append(pkCols, c.Name)
				break
			}
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
		if fmt.Sprintf("%v", srcVal) != fmt.Sprintf("%v", tgtVal) {
			changes = append(changes, fmt.Sprintf("%s: %v -> %v", c.Name, srcVal, tgtVal))
		}
	}
	return changes
}
