package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"server/framework"
	rag_svr "server/service/rag_svr/kitex_gen/rag_svr/ragservice"

	"github.com/cloudwego/kitex/server"
)

func main() {
	// 创建etcd服务注册组件
	r, err := framework.NewEtcdRegistry()
	if err != nil {
		log.Fatalf("创建etcd注册器失败: %v", err)
	}

	// 获取本机IP地址
	ipAddr := framework.GetLocalIP()

	// 创建并启动Kitex服务器
	svr := rag_svr.NewServer(
		new(RagServiceImpl),
		server.WithServerBasicInfo(framework.GetRegistryInfo("rag_svr")),
		server.WithServiceAddr(&net.TCPAddr{IP: net.ParseIP(ipAddr), Port: 8888}),
		server.WithRegistry(r),                 // 注册到etcd
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

	// 停止服务器
	if err := svr.Stop(); err != nil {
		log.Fatalf("服务器关闭失败: %v", err)
	}
}
