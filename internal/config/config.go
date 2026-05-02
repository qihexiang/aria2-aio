package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DataDir   string       `yaml:"data_dir"`
	Server    ServerConfig `yaml:"server"`
	Defaults  Defaults     `yaml:"defaults"`
	TaskTracker TrackerConfig `yaml:"task_tracker"`
	Log       LogConfig    `yaml:"log"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Defaults struct {
	RPCPortRange  PortRange          `yaml:"rpc_port_range"`
	Aria2Options  map[string]string  `yaml:"aria2_options"`
	Aria2cPath    string             `yaml:"aria2c_path"`
}

type PortRange struct {
	Start int `yaml:"start"`
	End   int `yaml:"end"`
}

type TrackerConfig struct {
	PollInterval        int `yaml:"poll_interval"`
	MaxHistoryPerInstance int `yaml:"max_history_per_instance"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

var DefaultConfig = Config{
	DataDir: "./data",
	Server: ServerConfig{
		Host: "0.0.0.0",
		Port: 8080,
	},
	Defaults: Defaults{
		RPCPortRange: PortRange{
			Start: 6801,
			End:   6899,
		},
		Aria2Options: map[string]string{
			"max-concurrent-downloads": "5",
			"split":                    "5",
			"max-download-limit":       "0",
			"max-upload-limit":         "0",
			"continue":                 "true",
			"auto-file-renaming":       "true",
			"disk-cache":               "32M",
		},
		Aria2cPath: "",
	},
	TaskTracker: TrackerConfig{
		PollInterval:          2,
		MaxHistoryPerInstance: 0,
	},
	Log: LogConfig{
		Level:  "info",
		Format: "text",
	},
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Defaults.RPCPortRange.Start >= c.Defaults.RPCPortRange.End {
		return fmt.Errorf("invalid rpc port range: start %d >= end %d",
			c.Defaults.RPCPortRange.Start, c.Defaults.RPCPortRange.End)
	}
	if c.TaskTracker.PollInterval < 1 {
		return fmt.Errorf("poll interval must be >= 1 second")
	}
	return nil
}

func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, "aria2-aio.db")
}

func (c *Config) ConfigFilePath() string {
	return filepath.Join(c.DataDir, "config.yaml")
}

func (c *Config) InstanceDir(instanceID string) string {
	return filepath.Join(c.DataDir, "instances", instanceID)
}

func (c *Config) ResolveAria2cPath() string {
	if c.Defaults.Aria2cPath != "" {
		return c.Defaults.Aria2cPath
	}
	if p, err := execLookPath("aria2c"); err == nil {
		return p
	}
	return "aria2c"
}

func SaveDefault(path string) error {
	data, err := yaml.Marshal(&DefaultConfig)
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}