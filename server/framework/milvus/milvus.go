// milvus.go

package milvus

import (
	"context"
	"fmt"
	"time"

	"server/framework/config"
	"server/framework/logger"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

var (
	milvusClient client.Client
)

// InitMilvus 初始化 Milvus 客户端
func InitMilvus() error {
	ctx := context.Background()

	// 验证 Milvus 配置
	if config.GlobalConfig.Milvus.Host == "" {
		return fmt.Errorf("Milvus host 未配置")
	}
	if config.GlobalConfig.Milvus.Port <= 0 {
		return fmt.Errorf("Milvus port 未配置或配置无效")
	}

	// 创建 Milvus 客户端
	milvusAddr := fmt.Sprintf("%s:%d", config.GlobalConfig.Milvus.Host, config.GlobalConfig.Milvus.Port)
	c, err := client.NewGrpcClient(ctx, milvusAddr)
	if err != nil {
		return err
	}

	// 设置全局客户端
	milvusClient = c

	// 测试连接
	_, err = c.ListCollections(ctx)
	if err != nil {
		return err
	}

	logger.Infof("Milvus 客户端初始化成功，连接到 %s", milvusAddr)
	return nil
}

// GetClient 获取 Milvus 客户端
func GetClient() client.Client {
	return milvusClient
}

// CreateCollection 创建集合
func CreateCollection(ctx context.Context, collectionName string, dim int) error {
	schema := &entity.Schema{
		CollectionName: collectionName,
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     true,
			},
			{
				Name:     "vector",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": string(dim),
				},
			},
		},
	}

	return milvusClient.CreateCollection(ctx, schema, 2)
}

// InsertVector 插入向量
func InsertVector(ctx context.Context, collectionName string, dim int, vectors [][]float32) error {
	// 准备插入数据
	insertData := []entity.Column{
		entity.NewColumnFloatVector("vector", dim, vectors),
	}

	// 插入数据
	_, err := milvusClient.Insert(ctx, collectionName, "", insertData...)
	return err
}

// SearchVector 搜索向量
func SearchVector(ctx context.Context, collectionName string, searchVector []float32, topK int) ([]int64, []float32, error) {
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 创建 HNSW 搜索参数，ef 可以根据需要调整
	searchParam, err := entity.NewIndexHNSWSearchParam(50) // 50 是 ef
	if err != nil {
		return nil, nil, err
	}

	// 执行搜索
	results, err := milvusClient.Search(
		ctx,
		collectionName,
		[]string{},     // partitionNames
		"",             // expr
		[]string{"id"}, // outputFields
		[]entity.Vector{entity.FloatVector(searchVector)},
		"vector",    // 搜索字段
		entity.L2,   // metricType
		topK,        // topK
		searchParam, // HNSW 搜索参数
	)

	if err != nil {
		return nil, nil, err
	}

	// 处理结果
	var ids []int64
	var distances []float32

	if len(results) == 0 {
		return ids, distances, nil
	}

	for _, result := range results {
		for _, score := range result.Scores {
			distances = append(distances, score)
		}
		idColumn := result.IDs.(*entity.ColumnInt64)
		for _, id := range idColumn.Data() {
			ids = append(ids, id)
		}
	}

	return ids, distances, nil
}
