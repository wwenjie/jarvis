package framework

import (
	"log"
	"net"
	"os"

	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/registry"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	etcd "github.com/kitex-contrib/registry-etcd"
)

// GetLocalIP 获取本机非回环IP地址
func GetLocalIP() string {
	host, err := os.Hostname()
	if err != nil {
		log.Fatalf("获取主机名失败: %v", err)
	}

	addrs, err := net.LookupIP(host)
	if err != nil || len(addrs) == 0 {
		log.Fatalf("获取IP地址失败: %v", err)
	}

	// 通常选择第一个非回环地址
	var ipAddr string
	for _, addr := range addrs {
		if !addr.IsLoopback() {
			ipAddr = addr.String()
			break
		}
	}

	if ipAddr == "" {
		ipAddr = "127.0.0.1"
	}

	return ipAddr
}

// NewEtcdRegistry 创建etcd服务注册组件
func NewEtcdRegistry() (registry.Registry, error) {
	return etcd.NewEtcdRegistry(EtcdEndpoints)
}

// NewEtcdResolver 创建etcd服务发现组件
func NewEtcdResolver() (discovery.Resolver, error) {
	return etcd.NewEtcdResolver(EtcdEndpoints)
}

// GetRegistryInfo 获取服务注册信息
func GetRegistryInfo(serviceName string) *rpcinfo.EndpointBasicInfo {
	return &rpcinfo.EndpointBasicInfo{
		ServiceName: serviceName,
		Tags: map[string]string{
			"version": "v1.0.0",
			"weight":  "100",
		},
	}
}
