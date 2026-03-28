package report

import (
	"db-diff/pkg/comparator"
	"fmt"
	"io"
	"os"
	"strings"
)

// GenerateText 生成文本格式报告
func GenerateText(result *comparator.DiffResult, outputFile string, verbose bool) error {
	var out io.Writer = os.Stdout

	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("创建输出文件失败：%w", err)
		}
		defer f.Close()
		out = f
	}

	fmt.Fprintln(out, "========================================")
	fmt.Fprintln(out, "   数据库结构对比报告")
	fmt.Fprintln(out, "========================================")
	fmt.Fprintf(out, "源数据库：%s\n", result.SourceSchema)
	fmt.Fprintf(out, "目标数据库：%s\n\n", result.TargetSchema)
	fmt.Fprintln(out, "图例：[相同] [已映射匹配] [类型不兼容]")
	fmt.Fprintln(out)

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

	// 数据差异
	hasDataDiff := false
	for table, diff := range result.TableDataDiff {
		if len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0 {
			if !hasDataDiff {
				fmt.Fprintln(out, "\n----------------------------------------")
				fmt.Fprintln(out, "数据差异:")
				hasDataDiff = true
			}
			fmt.Fprintf(out, "\n【表：%s】\n", table)
			printDataDiff(out, diff, verbose)
		}
	}

	// 总结
	fmt.Fprintln(out, "\n========================================")
	fmt.Fprintln(out, "对比完成")
	if !hasColumnDiff && !hasIndexDiff && !hasConstraintDiff && !hasDataDiff &&
		len(result.TableDiff.Added) == 0 && len(result.TableDiff.Missing) == 0 {
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

func printDataDiff(out io.Writer, diff comparator.DataDiff, verbose bool) {
	if len(diff.Added) > 0 {
		fmt.Fprintf(out, "  + 新增行 (%d):\n", len(diff.Added))
		for i, row := range diff.Added {
			if i >= 10 && !verbose {
				fmt.Fprintf(out, "    ... 还有 %d 行\n", len(diff.Added)-10)
				break
			}
			fmt.Fprintf(out, "    • 主键=%v\n", row["id"])
			if verbose {
				for k, v := range row {
					fmt.Fprintf(out, "      %s: %v\n", k, v)
				}
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
			fmt.Fprintf(out, "    • 主键=%v\n", row["id"])
			if verbose {
				for k, v := range row {
					fmt.Fprintf(out, "      %s: %v\n", k, v)
				}
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
		}
	}
}
