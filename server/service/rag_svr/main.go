package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"server/framework"
	"server/framework/config"
	"server/framework/etcd"
	"server/framework/logger"
	"server/framework/milvus"
	"server/framework/mongodb"
	"server/framework/mysql"
	"server/framework/redis"
	"server/service/rag_svr/ai"
	"server/service/rag_svr/kitex_gen/rag_svr/ragservice"

	"github.com/cloudwego/kitex/server"
)

func main() {
	// 初始化服务
	logger.Infof("开始初始化服务...")
	err, _, _ := framework.InitService()
	if err != nil {
		logger.Errorf("初始化服务失败: %v", err)
		os.Exit(1)
	}
	logger.Infof("服务初始化成功")

	// 初始化 MySQL
	if err := mysql.InitMySQL(); err != nil {
		logger.Errorf("初始化 MySQL 失败: %v", err)
		os.Exit(1)
	}
	logger.Infof("MySQL 初始化成功")

	// 初始化 Redis
	logger.Infof("开始初始化 Redis...")
	if err := redis.InitRedis(); err != nil {
		logger.Errorf("初始化 Redis 失败: %v", err)
		os.Exit(1)
	}
	logger.Infof("Redis 初始化成功")

	// 初始化 MongoDB
	logger.Infof("开始初始化 MongoDB...")
	if err := mongodb.InitMongoDB(); err != nil {
		logger.Errorf("初始化 MongoDB 失败: %v", err)
		os.Exit(1)
	}
	logger.Infof("MongoDB 初始化成功")

	// 初始化 Milvus
	logger.Infof("开始初始化 Milvus...")
	if err := milvus.InitMilvus(); err != nil {
		logger.Errorf("初始化 Milvus 失败: %v", err)
		os.Exit(1)
	}
	logger.Infof("Milvus 初始化成功")

	// 初始化 Qwen 客户端
	chatModel := config.GlobalConfig.AI.ChatModel
	qwenClient := ai.NewQwenClient(&struct {
		APIKey           string
		Provider         string
		ModelName        string
		Temperature      float64
		MaxTokens        int
		TopP             float64
		FrequencyPenalty float64
		PresencePenalty  float64
	}{
		APIKey:           chatModel.APIKey,
		Provider:         chatModel.Provider,
		ModelName:        chatModel.ModelName,
		Temperature:      chatModel.Temperature,
		MaxTokens:        chatModel.MaxTokens,
		TopP:             chatModel.TopP,
		FrequencyPenalty: chatModel.FrequencyPenalty,
		PresencePenalty:  chatModel.PresencePenalty,
	})

	// 创建服务实例
	svr := &RagServiceImpl{
		qwenClient: qwenClient,
	}

	// 创建并启动Kitex服务器
	logger.Infof("开始创建 Kitex 服务器...")
	s := ragservice.NewServer(
		svr,
		server.WithServerBasicInfo(etcd.GetRegistryInfo(config.GlobalConfig.Service.Name)),
		server.WithServiceAddr(&net.TCPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: config.GlobalConfig.Service.Port,
		}),
	)

	// 优雅退出
	go func() {
		if err := s.Run(); err != nil {
			logger.Fatalf("服务器运行失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 优雅关闭
	if err := s.Stop(); err != nil {
		logger.Fatalf("服务器关闭失败: %v", err)
	}
	logger.Infof("服务器已关闭")
}
