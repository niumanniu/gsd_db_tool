package main

import (
	"db-diff/pkg/comparator"
	"db-diff/pkg/config"
	"db-diff/pkg/report"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	sourceDSN   string
	targetDSN   string
	configFile  string
	outputFormat string
	outputFile   string
	verbose     bool
	dataMode    string
	sampleRatio float64
	sampleSize  int
	maxSampleSize int
	tablesFlag  string

	// 数据对比优化选项
	includeColumns string
	excludeColumns string
	hashFilter     bool
	showFullData   bool
	showProgress   bool

	// Driver and schema flags for cross-database support
	sourceDriver string
	targetDriver string
	sourceSchema string
	targetSchema string
)

var rootCmd = &cobra.Command{
	Use:   "db-diff",
	Short: "数据库表结构对比工具",
	Long: `db-diff 是一个用于对比两个数据库表结构的命令行工具。

支持对比表、字段、索引、约束的差异，并生成清晰的对比报告。
支持的数据库：MySQL, Oracle
支持跨数据库对比：MySQL ↔ Oracle`,
	RunE: runDiff,
}

func init() {
	rootCmd.Flags().StringVarP(&sourceDSN, "source", "s", "", "源数据库 DSN (格式：user:pass@tcp(host:port)/db)")
	rootCmd.Flags().StringVarP(&targetDSN, "target", "t", "", "目标数据库 DSN (格式：user:pass@tcp(host:port)/db)")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "配置文件路径 (YAML 格式)")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "输出格式 (text|html)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "输出文件路径 (默认：stdout)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "显示详细信息")
	rootCmd.Flags().StringVarP(&dataMode, "data-mode", "m", "count", "数据对比模式 (count|full|sample)")
	rootCmd.Flags().Float64VarP(&sampleRatio, "sample-ratio", "r", 0.1, "抽样比例 (0.0-1.0)")
	rootCmd.Flags().IntVar(&sampleSize, "sample-size", 0, "抽样数量（指定后忽略 sample-ratio）")
	rootCmd.Flags().IntVar(&maxSampleSize, "max-sample-size", 1000, "最大抽样数量")
	rootCmd.Flags().StringVarP(&tablesFlag, "tables", "T", "", "指定表列表，逗号分隔")

	// 数据对比优化选项
	rootCmd.Flags().StringVar(&includeColumns, "include-columns", "", "只比对的字段列表，逗号分隔")
	rootCmd.Flags().StringVar(&excludeColumns, "exclude-columns", "", "跳过比对的字段列表，逗号分隔")
	rootCmd.Flags().BoolVar(&hashFilter, "hash-filter", false, "启用 hash 预筛选（提高大宽表性能）")
	rootCmd.Flags().BoolVar(&showFullData, "show-full-data", false, "显示完整数据（源数据和目标数据）")
	rootCmd.Flags().BoolVar(&showProgress, "show-progress", false, "显示进度和耗时")

	// Cross-database support flags
	rootCmd.Flags().StringVar(&sourceDriver, "source-driver", "mysql", "源数据库驱动 (mysql|oracle)")
	rootCmd.Flags().StringVar(&targetDriver, "target-driver", "mysql", "目标数据库驱动 (mysql|oracle)")
	rootCmd.Flags().StringVar(&sourceSchema, "source-schema", "", "源数据库 schema (Oracle，默认为 user)")
	rootCmd.Flags().StringVar(&targetSchema, "target-schema", "", "目标数据库 schema (Oracle，默认为 user)")
}

func runDiff(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.Load(configFile, sourceDSN, targetDSN)
	if err != nil {
		return fmt.Errorf("加载配置失败：%w", err)
	}

	// 命令行参数覆盖（只有当用户实际指定参数时才覆盖）
	if cmd.Flags().Changed("data-mode") {
		cfg.CompareOptions.DataMode = config.DataMode(dataMode)
	}
	if cmd.Flags().Changed("sample-ratio") {
		cfg.CompareOptions.SampleRatio = sampleRatio
	}
	if cmd.Flags().Changed("sample-size") {
		cfg.CompareOptions.SampleSize = sampleSize
	}
	if cmd.Flags().Changed("max-sample-size") {
		cfg.CompareOptions.MaxSampleSize = maxSampleSize
	}
	if cmd.Flags().Changed("tables") {
		// 解析逗号分隔的表名列表
		cfg.CompareOptions.Tables = strings.Split(tablesFlag, ",")
		for i, t := range cfg.CompareOptions.Tables {
			cfg.CompareOptions.Tables[i] = strings.TrimSpace(t)
		}
	}
	if cmd.Flags().Changed("include-columns") {
		// 解析逗号分隔的字段列表
		cfg.CompareOptions.IncludeColumns = strings.Split(includeColumns, ",")
		for i, c := range cfg.CompareOptions.IncludeColumns {
			cfg.CompareOptions.IncludeColumns[i] = strings.TrimSpace(c)
		}
	}
	if cmd.Flags().Changed("exclude-columns") {
		// 解析逗号分隔的字段列表
		cfg.CompareOptions.ExcludeColumns = strings.Split(excludeColumns, ",")
		for i, c := range cfg.CompareOptions.ExcludeColumns {
			cfg.CompareOptions.ExcludeColumns[i] = strings.TrimSpace(c)
		}
	}
	if cmd.Flags().Changed("hash-filter") {
		cfg.CompareOptions.HashFilter = hashFilter
	}
	if cmd.Flags().Changed("show-full-data") {
		cfg.CompareOptions.ShowFullData = showFullData
	}
	if cmd.Flags().Changed("show-progress") {
		cfg.CompareOptions.ShowProgress = showProgress
	}

	// Driver and schema parameters (command line takes precedence)
	if sourceDriver != "" {
		cfg.Source.Driver = sourceDriver
	}
	if targetDriver != "" {
		cfg.Target.Driver = targetDriver
	}
	if sourceSchema != "" {
		cfg.Source.Schema = sourceSchema
	}
	if targetSchema != "" {
		cfg.Target.Schema = targetSchema
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败：%w", err)
	}

	// 执行对比
	result, err := comparator.Compare(cfg)
	if err != nil {
		return fmt.Errorf("对比失败：%w", err)
	}

	// 生成报告
	switch outputFormat {
	case "html":
		return report.GenerateHTML(result, outputFile, verbose)
	case "text":
		fallthrough
	default:
		return report.GenerateText(result, outputFile, verbose, cfg.CompareOptions.ShowFullData)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
