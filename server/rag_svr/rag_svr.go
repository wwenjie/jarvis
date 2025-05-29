package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"your_project/kitex_gen/your_service"
	"your_project/kitex_gen/your_service/proto"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	etcd "github.com/kitex-contrib/registry-etcd"
)

type UserServiceImpl struct{}

func (s *UserServiceImpl) GetUser(ctx context.Context, req *proto.GetUserRequest) (resp *proto.GetUserResponse, err error) {
	klog.Infof("收到获取用户请求: ID=%s", req.GetId())

	user := &proto.User{
		Id:   req.GetId(),
		Name: "User " + req.GetId(),
		Age:  30,
	}

	return &proto.GetUserResponse{
		User: user,
	}, nil
}

func main() {
	// 创建etcd服务注册组件
	r, err := etcd.NewEtcdRegistry([]string{"localhost:2379"})
	if err != nil {
		log.Fatalf("创建etcd注册器失败: %v", err)
	}

	// 获取本机IP地址
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

	// 创建并启动Kitex服务器
	svr := your_service.NewServer(
		&UserServiceImpl{},
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "your-service-name"}),
		server.WithServiceAddr(&net.TCPAddr{IP: net.ParseIP(ipAddr), Port: 8888}),
		server.WithRegistry(r), // 注册到etcd
		server.WithRegistryInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "your-service-name",
			Tags: map[string]string{
				"version": "v1.0.0",
				"weight":  "100",
			},
		}),
		server.WithExitWaitTime(3*time.Second), // 优雅关闭等待时间
	)

	// 启动服务器
	go func() {
		if err := svr.Run(); err != nil {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := svr.Shutdown(ctx); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}
}
