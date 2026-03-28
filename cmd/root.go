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
	tablesFlag  string

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
	rootCmd.Flags().StringVarP(&tablesFlag, "tables", "T", "", "指定表列表，逗号分隔")

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

	// 命令行参数覆盖
	if dataMode != "" {
		cfg.DataMode = config.DataMode(dataMode)
	}
	if sampleRatio > 0 {
		cfg.SampleRatio = sampleRatio
	}
	if tablesFlag != "" {
		// 解析逗号分隔的表名列表
		cfg.Tables = strings.Split(tablesFlag, ",")
		for i, t := range cfg.Tables {
			cfg.Tables[i] = strings.TrimSpace(t)
		}
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
		return report.GenerateText(result, outputFile, verbose)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
