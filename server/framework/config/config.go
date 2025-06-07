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

	Milvus struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"milvus"`

	Log struct {
		Level      string `yaml:"level"`
		LogPath    string `yaml:"log_path"`
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
	// 验证 service 配置
	if GlobalConfig.Service.Name == "" {
		return fmt.Errorf("service name 未配置")
	}
	if GlobalConfig.Service.Port <= 0 {
		return fmt.Errorf("service port 未配置或配置无效")
	}
	if GlobalConfig.Service.Env == "" {
		return fmt.Errorf("service env 未配置")
	}
	if GlobalConfig.Service.Region == "" {
		return fmt.Errorf("service region 未配置")
	}

	// 验证 etcd 配置
	if len(GlobalConfig.Etcd.Endpoints) == 0 {
		return fmt.Errorf("etcd endpoints 未配置")
	}
	if GlobalConfig.Etcd.Timeout <= 0 {
		return fmt.Errorf("etcd timeout 未配置或配置无效")
	}

	// 验证 log 配置
	if GlobalConfig.Log.Level == "" {
		return fmt.Errorf("log level 未配置")
	}
	if GlobalConfig.Log.LogPath == "" {
		return fmt.Errorf("log path 未配置")
	}
	if GlobalConfig.Log.MaxSize <= 0 {
		return fmt.Errorf("log max_size 未配置或配置无效")
	}
	if GlobalConfig.Log.MaxBackups <= 0 {
		return fmt.Errorf("log max_backups 未配置或配置无效")
	}
	if GlobalConfig.Log.MaxAge <= 0 {
		return fmt.Errorf("log max_age 未配置或配置无效")
	}

	return nil
}
