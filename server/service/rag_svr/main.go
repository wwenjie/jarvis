package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"server/framework"
	"server/framework/etcd"
	"server/framework/logger"
	rag_svr "server/service/rag_svr/kitex_gen/rag_svr/ragservice"

	"github.com/cloudwego/kitex/server"
)

func main() {
	// 创建etcd服务注册组件
	err, etcdRegistry, _ := framework.InitService()
	if err != nil {
		log.Fatalf("初始化服务失败: %v", err)
	}

	// 创建并启动Kitex服务器
	svr := rag_svr.NewServer(
		new(RagServiceImpl),
		server.WithServerBasicInfo(etcd.GetRegistryInfo("rag_svr")),
		server.WithServiceAddr(&net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8888}), // 监听所有网络接口
		server.WithRegistry(etcdRegistry),                                            // 注册到etcd
		server.WithExitWaitTime(3*time.Second),                                       // 优雅关闭等待时间
	)

	// log.Printf("rag_svr start succ!")
	logger.LogInfo("rag_svr", "main", "rag_svr start succ!")

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
