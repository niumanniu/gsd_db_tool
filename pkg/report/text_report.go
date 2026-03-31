package report

import (
	"db-diff/pkg/comparator"
	"fmt"
	"io"
	"os"
	"strings"
)

// GenerateText 生成文本格式报告
func GenerateText(result *comparator.DiffResult, outputFile string, verbose bool, showFullData bool) error {
	var out io.Writer = os.Stdout

	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("创建输出文件失败：%w", err)
		}
		defer f.Close()
		out = f
	}

	// 判断是否有结构差异和数据差异
	hasStructDiff := len(result.TableDiff.Added) > 0 || len(result.TableDiff.Missing) > 0
	for _, diff := range result.ColumnDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			hasStructDiff = true
			break
		}
	}
	for _, diff := range result.IndexDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			hasStructDiff = true
			break
		}
	}
	for _, diff := range result.ConstraintDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			hasStructDiff = true
			break
		}
	}

	hasDataDiff := false
	for _, diff := range result.TableDataDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			hasDataDiff = true
			break
		}
	}

	hasCountDiff := false
	for _, count := range result.TableCounts {
		if count.Diff != 0 {
			hasCountDiff = true
			break
		}
	}

	// 根据对比内容动态显示标题
	fmt.Fprintln(out, "========================================")
	if hasStructDiff && hasDataDiff {
		fmt.Fprintln(out, "   数据库结构与数据对比报告")
	} else if hasDataDiff || hasCountDiff {
		fmt.Fprintln(out, "   数据库数据对比报告")
	} else {
		fmt.Fprintln(out, "   数据库结构对比报告")
	}
	fmt.Fprintln(out, "========================================")
	fmt.Fprintf(out, "源数据库：%s\n", result.SourceSchema)
	fmt.Fprintf(out, "目标数据库：%s\n\n", result.TargetSchema)

	// 只显示结构对比相关的图例
	if hasStructDiff {
		fmt.Fprintln(out, "图例：[相同] [已映射匹配] [类型不兼容]")
		fmt.Fprintln(out)
	}

	// 表差异摘要
	printTableDiff(out, result.TableDiff)

	// 列差异摘要
	hasColumnDiff := false
	for table, diff := range result.ColumnDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			if !hasColumnDiff {
				fmt.Fprintln(out, "\n----------------------------------------")
				fmt.Fprintln(out, "列差异:")
				hasColumnDiff = true
			}
			fmt.Fprintf(out, "\n【表：%s】\n", table)
			printColumnDiff(out, diff, verbose)
		}
	}

	// 索引差异
	hasIndexDiff := false
	for table, diff := range result.IndexDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			if !hasIndexDiff {
				fmt.Fprintln(out, "\n----------------------------------------")
				fmt.Fprintln(out, "索引差异:")
				hasIndexDiff = true
			}
			fmt.Fprintf(out, "\n【表：%s】\n", table)
			printIndexDiff(out, diff, verbose)
		}
	}

	// 约束差异
	hasConstraintDiff := false
	for table, diff := range result.ConstraintDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			if !hasConstraintDiff {
				fmt.Fprintln(out, "\n----------------------------------------")
				fmt.Fprintln(out, "约束差异:")
				hasConstraintDiff = true
			}
			fmt.Fprintf(out, "\n【表：%s】\n", table)
			printConstraintDiff(out, diff, verbose)
		}
	}

	// 记录数对比
	if len(result.TableCounts) > 0 {
		fmt.Fprintln(out, "\n----------------------------------------")
		fmt.Fprintln(out, "记录数对比:")
		fmt.Fprintf(out, "%-20s %-15s %-15s %-10s\n", "表名", "源库", "目标库", "差异")
		for table, count := range result.TableCounts {
			diffStr := ""
			if count.Diff > 0 {
				diffStr = fmt.Sprintf("+%d", count.Diff)
			} else if count.Diff < 0 {
				diffStr = fmt.Sprintf("%d", count.Diff)
			} else {
				diffStr = "="
			}
			fmt.Fprintf(out, "%-20s %-15d %-15d %-10s\n", table, count.SourceCount, count.TargetCount, diffStr)
		}
	}

	// 详细数据差异（非 count 模式）
	detailedDataDiff := false
	for table, diff := range result.TableDataDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			if !detailedDataDiff {
				fmt.Fprintln(out, "\n----------------------------------------")
				fmt.Fprintln(out, "数据差异:")
				detailedDataDiff = true
			}
			fmt.Fprintf(out, "\n【表：%s】\n", table)
			printDataDiff(out, diff, verbose, showFullData)
		}
	}

	// 总结
	fmt.Fprintln(out, "\n========================================")
	fmt.Fprintln(out, "对比完成")
	if !hasColumnDiff && !hasIndexDiff && !hasConstraintDiff && !detailedDataDiff &&
		!hasCountDiff && len(result.TableDiff.Added) == 0 && len(result.TableDiff.Missing) == 0 {
		fmt.Fprintln(out, "✓ 两个数据库完全一致")
	} else {
		fmt.Fprintln(out, "✗ 发现差异")
	}

	return nil
}

func printTableDiff(out io.Writer, diff comparator.TableDiff) {
	fmt.Fprintln(out, "表差异:")
	if len(diff.Added) == 0 && len(diff.Missing) == 0 {
		fmt.Fprintln(out, "  无差异")
		return
	}
	if len(diff.Added) > 0 {
		fmt.Fprintf(out, "  + 新增表 (%d): %s\n", len(diff.Added), strings.Join(diff.Added, ", "))
	}
	if len(diff.Missing) > 0 {
		fmt.Fprintf(out, "  - 缺失表 (%d): %s\n", len(diff.Missing), strings.Join(diff.Missing, ", "))
	}
}

func printColumnDiff(out io.Writer, diff comparator.ColumnDiff, verbose bool) {
	if len(diff.Added) > 0 {
		fmt.Fprintf(out, "  + 新增列 (%d):\n", len(diff.Added))
		for _, col := range diff.Added {
			fmt.Fprintf(out, "    • %s (%s)\n", col.Name, col.ColumnType)
		}
	}
	if len(diff.Removed) > 0 {
		fmt.Fprintf(out, "  - 删除列 (%d):\n", len(diff.Removed))
		for _, col := range diff.Removed {
			fmt.Fprintf(out, "    • %s (%s)\n", col.Name, col.ColumnType)
		}
	}
	if len(diff.Modified) > 0 {
		fmt.Fprintf(out, "  ~ 修改列 (%d):\n", len(diff.Modified))
		for _, mod := range diff.Modified {
			fmt.Fprintf(out, "    • %s:\n", mod.Name)
			// Show type mapping status
			if mod.TypeMapping != nil {
				statusText := typeMappingStatusText(string(mod.TypeMapping.Status))
				fmt.Fprintf(out, "      类型状态：%s", statusText)
				if mod.TypeMapping.Status == "mapped" {
					fmt.Fprintf(out, " → %s", mod.TypeMapping.MappedType)
				}
				fmt.Fprintln(out)
			}
			if verbose {
				for _, change := range mod.Changes {
					fmt.Fprintf(out, "      %s\n", change)
				}
			} else {
				fmt.Fprintf(out, "      %s\n", strings.Join(mod.Changes, ", "))
			}
		}
	}
}

// typeMappingStatusText returns Chinese text for mapping status
func typeMappingStatusText(status string) string {
	switch status {
	case "same":
		return "[相同]"
	case "mapped":
		return "[已映射匹配]"
	case "incompatible":
		return "[类型不兼容]"
	default:
		return "[未知]"
	}
}

func printIndexDiff(out io.Writer, diff comparator.IndexDiff, verbose bool) {
	if len(diff.Added) > 0 {
		fmt.Fprintf(out, "  + 新增索引 (%d):\n", len(diff.Added))
		for _, idx := range diff.Added {
			fmt.Fprintf(out, "    • %s (%s)\n", idx.Name, idx.IndexType)
		}
	}
	if len(diff.Removed) > 0 {
		fmt.Fprintf(out, "  - 删除索引 (%d):\n", len(diff.Removed))
		for _, idx := range diff.Removed {
			fmt.Fprintf(out, "    • %s (%s)\n", idx.Name, idx.IndexType)
		}
	}
	if len(diff.Modified) > 0 {
		fmt.Fprintf(out, "  ~ 修改索引 (%d):\n", len(diff.Modified))
		for _, mod := range diff.Modified {
			fmt.Fprintf(out, "    • %s\n", mod.Name)
			if verbose {
				for _, change := range mod.Changes {
					fmt.Fprintf(out, "      %s\n", change)
				}
			}
		}
	}
}

func printConstraintDiff(out io.Writer, diff comparator.ConstraintDiff, verbose bool) {
	if len(diff.Added) > 0 {
		fmt.Fprintf(out, "  + 新增约束 (%d):\n", len(diff.Added))
		for _, con := range diff.Added {
			fmt.Fprintf(out, "    • %s (%s)\n", con.Name, con.Type)
		}
	}
	if len(diff.Removed) > 0 {
		fmt.Fprintf(out, "  - 删除约束 (%d):\n", len(diff.Removed))
		for _, con := range diff.Removed {
			fmt.Fprintf(out, "    • %s (%s)\n", con.Name, con.Type)
		}
	}
	if len(diff.Modified) > 0 {
		fmt.Fprintf(out, "  ~ 修改约束 (%d):\n", len(diff.Modified))
		for _, mod := range diff.Modified {
			fmt.Fprintf(out, "    • %s\n", mod.Name)
		}
	}
}

func printDataDiff(out io.Writer, diff comparator.DataDiff, verbose bool, showFullData bool) {
	if len(diff.Added) > 0 {
		fmt.Fprintf(out, "  + 新增行 (%d):\n", len(diff.Added))
		for i, row := range diff.Added {
			if i >= 10 && !verbose {
				fmt.Fprintf(out, "    ... 还有 %d 行\n", len(diff.Added)-10)
				break
			}
			fmt.Fprintf(out, "    • 行 %d:\n", i+1)
			for k, v := range row {
				valStr := formatValue(v)
				fmt.Fprintf(out, "        %s: %s\n", k, valStr)
			}
		}
	}
	if len(diff.Removed) > 0 {
		fmt.Fprintf(out, "  - 删除行 (%d):\n", len(diff.Removed))
		for i, row := range diff.Removed {
			if i >= 10 && !verbose {
				fmt.Fprintf(out, "    ... 还有 %d 行\n", len(diff.Removed)-10)
				break
			}
			fmt.Fprintf(out, "    • 行 %d:\n", i+1)
			for k, v := range row {
				valStr := formatValue(v)
				fmt.Fprintf(out, "        %s: %s\n", k, valStr)
			}
		}
	}
	if len(diff.Modified) > 0 {
		fmt.Fprintf(out, "  ~ 修改行 (%d):\n", len(diff.Modified))
		for i, mod := range diff.Modified {
			if i >= 10 && !verbose {
				fmt.Fprintf(out, "    ... 还有 %d 行\n", len(diff.Modified)-10)
				break
			}
			fmt.Fprintf(out, "    • 主键=%v:\n", mod.Key)
			for _, change := range mod.Changes {
				fmt.Fprintf(out, "      %s\n", change)
			}
			// 根据配置显示完整数据
			if showFullData {
				fmt.Fprintln(out, "      源数据:")
				for k, v := range mod.Source {
					valStr := formatValue(v)
					fmt.Fprintf(out, "        %s: %s\n", k, valStr)
				}
				fmt.Fprintln(out, "      目标数据:")
				for k, v := range mod.Target {
					valStr := formatValue(v)
					fmt.Fprintf(out, "        %s: %s\n", k, valStr)
				}
			}
		}
	}
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
			if len(str) > 100 {
				return str[:100] + "..."
			}
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
