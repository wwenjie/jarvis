package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	Service struct {
		Name    string `yaml:"name"`
		Port    int    `yaml:"port"`
		Version string `yaml:"version"`
		Env     string `yaml:"env"`
		Region  string `yaml:"region"`
		Weight  int    `yaml:"weight"`
	} `yaml:"service"`

	Etcd struct {
		Endpoints []string `yaml:"endpoints"`
		Timeout   int      `yaml:"timeout"`
	} `yaml:"etcd"`

	Log struct {
		Level      string `yaml:"level"`
		Filename   string `yaml:"filename"`
		MaxSize    int    `yaml:"max_size"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAge     int    `yaml:"max_age"`
		Compress   bool   `yaml:"compress"`
	} `yaml:"log"`
}

var GlobalConfig = &Config{}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}
	return yaml.Unmarshal(data, GlobalConfig)
}

// ValidateConfig 验证配置
func ValidateConfig() error {
	// ... 验证代码 ...
	return nil
}
