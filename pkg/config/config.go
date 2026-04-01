package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

// DataMode 数据对比模式
type DataMode string

const (
	DataModeCount  DataMode = "count"  // 只对比记录数
	DataModeFull   DataMode = "full"   // 全量数据对比
	DataModeSample DataMode = "sample" // 抽样对比
)

// CompareOptions 对比选项
type CompareOptions struct {
	Mode            DataMode `yaml:"mode"`              // 对比模式：structure|data
	DataMode        DataMode `yaml:"data_mode"`         // 数据对比模式：count|full|sample
	SampleRatio     float64  `yaml:"sample_ratio"`      // 抽样比例 (0.0-1.0)
	SampleSize      int      `yaml:"sample_size"`       // 抽样数量（指定后优先使用，忽略 sample_ratio）
	MaxSampleSize   int      `yaml:"max_sample_size"`   // 最大抽样数量，默认 1000
	Tables          []string `yaml:"tables"`            // 指定表列表，空表示所有表
	IncludeColumns  []string `yaml:"include_columns"`   // 只比对的字段列表，空表示所有字段
	ExcludeColumns  []string `yaml:"exclude_columns"`   // 跳过比对的字段列表
	HashFilter      bool     `yaml:"hash_filter"`       // 是否启用 hash 预筛选（提高大宽表性能）
	BatchSize       int      `yaml:"batch_size"`        // 数据比对批次大小（每批读取的行数）
	ShowFullData    bool     `yaml:"show_full_data"`    // 是否显示完整数据（源数据和目标数据）
	ShowProgress    bool     `yaml:"show_progress"`     // 是否显示进度和耗时
}

// Config 数据库对比配置
type Config struct {
    Source       Database       `yaml:"source"`
    Target       Database       `yaml:"target"`
    CompareOptions CompareOptions `yaml:"compare_options"`
}

// Database 数据库连接配置
type Database struct {
	Driver   string `yaml:"driver"`   // "mysql" or "oracle"
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Schema   string `yaml:"schema"`   // Oracle schema, defaults to User
}

// DSN 生成 DSN 连接字符串（根据 driver 自动选择）
func (d *Database) DSN() string {
	if d.Driver == "oracle" {
		return d.OracleDSN()
	}
	return d.MySQLDSN()
}

// MySQLDSN 生成 MySQL DSN 连接字符串
func (d *Database) MySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		d.User, d.Password, d.Host, d.Port, d.Database)
}

// OracleDSN 生成 Oracle DSN 连接字符串
func (d *Database) OracleDSN() string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		d.User, d.Password, d.Host, d.Port, d.Database)
}

// GetSchema 获取 schema 名称
func (d *Database) GetSchema() string {
	if d.Schema != "" {
		return d.Schema
	}
	if d.Driver == "oracle" {
		return d.User // Oracle 默认 schema 等于 user
	}
	return d.Database // MySQL 使用 database 名
}

// Load 加载配置，支持命令行参数和配置文件
func Load(configFile, sourceDSN, targetDSN string) (*Config, error) {
	cfg := &Config{}

	// 如果提供了配置文件，从文件加载
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("读取配置文件失败：%w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件失败：%w", err)
		}
	}

	// 命令行参数覆盖配置文件
	if sourceDSN != "" {
		cfg.Source = DatabaseFromDSN(sourceDSN)
	}
	if targetDSN != "" {
		cfg.Target = DatabaseFromDSN(targetDSN)
	}

	return cfg, nil
}

// DatabaseFromDSN 从 DSN 字符串解析数据库配置
func DatabaseFromDSN(dsn string) Database {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return Database{}
	}

	// 解析地址获取 host 和 port
	hostPort := strings.Split(cfg.Addr, ":")
	host := hostPort[0]
	port := 3306
	if len(hostPort) > 1 {
		fmt.Sscanf(hostPort[1], "%d", &port)
	}

	return Database{
		Driver:   "mysql", // 默认 mysql
		Host:     host,
		Port:     port,
		User:     cfg.User,
		Password: cfg.Passwd,
		Database: cfg.DBName,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Source.Host == "" {
		return fmt.Errorf("源数据库配置缺失")
	}
	if c.Target.Host == "" {
		return fmt.Errorf("目标数据库配置缺失")
	}
	if c.Source.Driver == "" {
		c.Source.Driver = "mysql" // 默认 mysql
	}
	if c.Target.Driver == "" {
		c.Target.Driver = "mysql" // 默认 mysql
	}
	if c.CompareOptions.DataMode == "" {
		c.CompareOptions.DataMode = DataModeCount // 默认只对比记录数
	}
	if c.CompareOptions.DataMode != DataModeCount && c.CompareOptions.DataMode != DataModeFull && c.CompareOptions.DataMode != DataModeSample {
		return fmt.Errorf("无效的数据对比模式：%s", c.CompareOptions.DataMode)
	}
	if c.CompareOptions.DataMode == DataModeSample {
		if c.CompareOptions.SampleRatio <= 0 || c.CompareOptions.SampleRatio > 1 {
			c.CompareOptions.SampleRatio = 0.1 // 默认 10% 抽样
		}
		if c.CompareOptions.MaxSampleSize <= 0 {
			c.CompareOptions.MaxSampleSize = 1000 // 默认最大抽样 1000 条
		}
	}
	return nil
}
