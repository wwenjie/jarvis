package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

var (
	GlobalConfig *Config
)

// Config 配置结构体
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

	Redis struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`

	MySQL struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mysql"`

	MongoDB struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mongodb"`

	Milvus struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"milvus"`

	AI struct {
		ChatModel struct {
			APIKey           string  `yaml:"-"`                 // 从环境变量读取
			Provider         string  `yaml:"provider"`          // 支持 dashscope, openai 等
			ModelName        string  `yaml:"model_name"`        // 模型名称
			Temperature      float64 `yaml:"temperature"`       // 温度参数
			MaxTokens        int     `yaml:"max_tokens"`        // 最大生成token数
			TopP             float64 `yaml:"top_p"`             // 采样阈值
			FrequencyPenalty float64 `yaml:"frequency_penalty"` // 频率惩罚
			PresencePenalty  float64 `yaml:"presence_penalty"`  // 存在惩罚
		} `yaml:"chat_model"`
		EmbeddingModel struct {
			Provider  string `yaml:"provider"`   // 支持 dashscope, openai 等
			ModelName string `yaml:"model_name"` // 向量化模型名称
			Dimension int    `yaml:"dimension"`  // 向量维度
		} `yaml:"embedding_model"`
	} `yaml:"ai"`

	Log struct {
		Level      string `yaml:"level"`
		LogPath    string `yaml:"log_path"`
		MaxSize    int    `yaml:"max_size"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAge     int    `yaml:"max_age"`
		Compress   bool   `yaml:"compress"`
	} `yaml:"log"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) error {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析配置文件
	GlobalConfig = &Config{}
	if err := yaml.Unmarshal(data, GlobalConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 从环境变量加载 AI 配置
	if err := loadAIConfig(); err != nil {
		return fmt.Errorf("加载 AI 配置失败: %v", err)
	}

	return nil
}

// loadAIConfig 从环境变量加载 AI 配置
func loadAIConfig() error {
	// 加载 API Key
	GlobalConfig.AI.ChatModel.APIKey = os.Getenv("DASHSCOPE_API_KEY")
	if GlobalConfig.AI.ChatModel.APIKey == "" {
		return fmt.Errorf("环境变量 DASHSCOPE_API_KEY 未设置")
	}

	// 加载模型名称
	if modelName := os.Getenv("MODEL_NAME"); modelName != "" {
		GlobalConfig.AI.ChatModel.ModelName = modelName
	}

	// 加载温度参数
	if temp := os.Getenv("TEMPERATURE"); temp != "" {
		if tempFloat, err := strconv.ParseFloat(temp, 64); err == nil {
			GlobalConfig.AI.ChatModel.Temperature = tempFloat
		}
	}

	return nil
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
