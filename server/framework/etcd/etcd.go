package etcd

import (
	"fmt"

	"server/framework/config"
	"server/framework/network"

	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/registry"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	registry_etcd "github.com/kitex-contrib/registry-etcd"
)

func InitEtcd() (registry.Registry, discovery.Resolver, error) {
	registry, err := registry_etcd.NewEtcdRegistry(config.GlobalConfig.Etcd.Endpoints)
	if err != nil {
		return nil, nil, err
	}
	resolver, err := registry_etcd.NewEtcdResolver(config.GlobalConfig.Etcd.Endpoints)
	if err != nil {
		return nil, nil, err
	}
	return registry, resolver, nil
}

// GetRegistryInfo 获取服务注册信息
func GetRegistryInfo(serviceName string) *rpcinfo.EndpointBasicInfo {
	return &rpcinfo.EndpointBasicInfo{
		ServiceName: serviceName,
		Tags: map[string]string{
			"version": config.GlobalConfig.Service.Version,
			"weight":  fmt.Sprintf("%d", config.GlobalConfig.Service.Weight),
			"env":     config.GlobalConfig.Service.Env,
			"region":  config.GlobalConfig.Service.Region,
			"ip":      network.GetLocalIP(),
		},
	}
}
