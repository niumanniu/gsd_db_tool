package config

import (
	"os"
	"testing"
)

func TestLoad_FromConfigFile(t *testing.T) {
	// 创建临时配置文件
	content := `
source:
  host: localhost
  port: 3306
  user: root
  password: secret
  database: source_db
target:
  host: localhost
  port: 3307
  user: root
  password: secret
  database: target_db
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name(), "", "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Source.Host != "localhost" {
		t.Errorf("Source.Host = %v, want localhost", cfg.Source.Host)
	}
	if cfg.Source.Port != 3306 {
		t.Errorf("Source.Port = %v, want 3306", cfg.Source.Port)
	}
	if cfg.Source.Database != "source_db" {
		t.Errorf("Source.Database = %v, want source_db", cfg.Source.Database)
	}
}

func TestLoad_FromDSN(t *testing.T) {
	cfg, err := Load("", "root:pass@tcp(localhost:3306)/db1", "root:pass@tcp(localhost:3307)/db2")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Source.Host != "localhost" {
		t.Errorf("Source.Host = %v, want localhost", cfg.Source.Host)
	}
	if cfg.Source.Port != 3306 {
		t.Errorf("Source.Port = %v, want 3306", cfg.Source.Port)
	}
	if cfg.Source.Database != "db1" {
		t.Errorf("Source.Database = %v, want db1", cfg.Source.Database)
	}
}

func TestLoad_DSNOverridesConfig(t *testing.T) {
	content := `
source:
  host: config-host
  port: 3306
  user: root
  password: secret
  database: config_db
target:
  host: config-target
  port: 3306
  user: root
  password: secret
  database: config_target_db
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name(), "root:pass@tcp(dsn-host:3306)/dsn_db", "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Source.Host != "dsn-host" {
		t.Errorf("Source.Host = %v, want dsn-host", cfg.Source.Host)
	}
	if cfg.Source.Database != "dsn_db" {
		t.Errorf("Source.Database = %v, want dsn_db", cfg.Source.Database)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Source: Database{Host: "localhost"},
				Target: Database{Host: "localhost"},
			},
			wantErr: false,
		},
		{
			name: "missing source",
			cfg: &Config{
				Target: Database{Host: "localhost"},
			},
			wantErr: true,
		},
		{
			name: "missing target",
			cfg: &Config{
				Source: Database{Host: "localhost"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDatabase_DSN(t *testing.T) {
	db := Database{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "secret",
		Database: "testdb",
	}

	dsn := db.DSN()
	expected := "root:secret@tcp(localhost:3306)/testdb?parseTime=true"
	if dsn != expected {
		t.Errorf("DSN() = %v, want %v", dsn, expected)
	}
}
