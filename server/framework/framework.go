package framework

import (
	"fmt"
	"os"
	"path/filepath"
	"server/framework/config"
	"server/framework/etcd"
	"server/framework/logger"

	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/registry"
)

func InitService() (error, registry.Registry, discovery.Resolver) {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %v", err), nil, nil
	}

	configPath := filepath.Join(workDir, "config", "config.yaml")
	if err := config.LoadConfig(configPath); err != nil {
		return fmt.Errorf("加载配置失败: %v", err), nil, nil
	}

	if err := config.ValidateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err), nil, nil
	}

	logger.InitLogger()
	etcdRegistry, etcdResolver, err := etcd.InitEtcd()
	if err != nil {
		return fmt.Errorf("初始化etcd失败: %v", err), nil, nil
	}
	return nil, etcdRegistry, etcdResolver
}
