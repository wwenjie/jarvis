// milvus.go

package milvus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"server/framework/config"
	"server/framework/logger"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

var milvusClient client.Client

// 集合名称常量
const (
	MemoryCollectionName   = "chat_memory"    // 记忆集合
	DocumentCollectionName = "document_chunk" // 文档块集合
)

// MilvusStats 用于记录 Milvus 操作统计信息
type MilvusStats struct {
	SearchLatency time.Duration
	InsertLatency time.Duration
	DeleteLatency time.Duration
	SearchCount   int64
	InsertCount   int64
	DeleteCount   int64
	ErrorCount    int64
}

var stats MilvusStats

// GetStats 获取统计信息
func GetStats() MilvusStats {
	return stats
}

// InitMilvus 初始化 Milvus 客户端
func InitMilvus() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", config.GlobalConfig.Milvus.Host, config.GlobalConfig.Milvus.Port)
	var err error
	milvusClient, err = client.NewGrpcClient(ctx, addr)
	if err != nil {
		return fmt.Errorf("创建 Milvus 客户端失败: %v", err)
	}

	// 确保集合已加载
	if err := milvusClient.LoadCollection(ctx, MemoryCollectionName, true); err != nil {
		return fmt.Errorf("加载记忆集合失败: %v", err)
	}

	if err := milvusClient.LoadCollection(ctx, DocumentCollectionName, true); err != nil {
		return fmt.Errorf("加载文档集合失败: %v", err)
	}

	return nil
}

// InsertVector 插入向量
func InsertVector(ctx context.Context, collectionName string, id int64, vector []float32) error {
	// 准备数据
	logger.Infof("insert vector,collectionName=%s, id=%d", collectionName, id)
	row := map[string]interface{}{
		"id":     id,
		"vector": vector,
	}

	// 插入数据
	_, err := milvusClient.InsertRows(ctx, collectionName, "", []interface{}{row})
	if err != nil {
		return fmt.Errorf("插入向量失败: %v", err)
	}

	// 刷新数据
	err = milvusClient.Flush(ctx, collectionName, false)
	if err != nil {
		return fmt.Errorf("刷新数据失败: %v", err)
	}

	return nil
}

// DeleteVector 删除向量
func DeleteVector(ctx context.Context, collectionName string, id int64) error {
	// 删除数据
	err := milvusClient.Delete(ctx, collectionName, "", fmt.Sprintf("id == %d", id))
	if err != nil {
		return fmt.Errorf("删除向量失败: %v", err)
	}

	// 刷新数据
	err = milvusClient.Flush(ctx, collectionName, false)
	if err != nil {
		return fmt.Errorf("刷新数据失败: %v", err)
	}

	return nil
}

// SearchVector 搜索向量
func SearchVector(ctx context.Context, collectionName string, queryVector []float32, topK int) ([]int64, []float32, error) {
	// 准备搜索参数
	searchParam, err := entity.NewIndexHNSWSearchParam(64) // ef = 64
	if err != nil {
		return nil, nil, fmt.Errorf("创建搜索参数失败: %v", err)
	}

	// 执行搜索
	results, err := milvusClient.Search(
		ctx,
		collectionName,
		[]string{},
		"",
		[]string{"id"},
		[]entity.Vector{entity.FloatVector(queryVector)},
		"vector",
		entity.L2,
		topK,
		searchParam,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("搜索向量失败: %v", err)
	}

	// 解析结果
	if len(results) == 0 {
		return nil, nil, nil
	}

	ids := make([]int64, 0)
	scores := make([]float32, 0)

	for _, result := range results {
		for i := 0; i < result.ResultCount; i++ {
			id := result.IDs.(*entity.ColumnInt64).Data()[i]
			score := result.Scores[i]
			ids = append(ids, id)
			scores = append(scores, score)
		}
	}

	return ids, scores, nil
}

// UpdateVector 更新向量
func UpdateVector(ctx context.Context, collectionName string, id int64, vector []float32) error {
	// 准备数据
	row := map[string]interface{}{
		"id":     id,
		"vector": vector,
	}

	// 删除旧向量
	if err := milvusClient.Delete(ctx, collectionName, "", fmt.Sprintf("id == %d", id)); err != nil {
		return fmt.Errorf("删除旧向量失败: %v", err)
	}

	// 插入新向量
	_, err := milvusClient.InsertRows(ctx, collectionName, "", []interface{}{row})
	if err != nil {
		return fmt.Errorf("插入新向量失败: %v", err)
	}

	// 刷新数据
	err = milvusClient.Flush(ctx, collectionName, false)
	if err != nil {
		return fmt.Errorf("刷新数据失败: %v", err)
	}

	return nil
}

// Close 关闭 Milvus 客户端
func Close() error {
	if milvusClient != nil {
		return milvusClient.Close()
	}
	return nil
}

// BatchInsertVectors 批量插入向量
func BatchInsertVectors(ctx context.Context, collectionName string, ids []int64, vectors [][]float32) error {
	start := time.Now()
	defer func() {
		stats.InsertLatency = time.Since(start)
		stats.InsertCount++
	}()

	// 准备数据
	rows := make([]interface{}, len(ids))
	for i := range ids {
		rows[i] = map[string]interface{}{
			"id":     ids[i],
			"vector": vectors[i],
		}
	}

	// 插入数据
	_, err := milvusClient.InsertRows(ctx, collectionName, "", rows)
	if err != nil {
		stats.ErrorCount++
		return fmt.Errorf("批量插入向量失败: %v", err)
	}

	// 刷新数据
	err = milvusClient.Flush(ctx, collectionName, false)
	if err != nil {
		stats.ErrorCount++
		return fmt.Errorf("刷新数据失败: %v", err)
	}

	return nil
}

// BatchDeleteVectors 批量删除向量
func BatchDeleteVectors(ctx context.Context, collectionName string, ids []int64) error {
	start := time.Now()
	defer func() {
		stats.DeleteLatency = time.Since(start)
		stats.DeleteCount++
	}()

	// 构建删除条件
	expr := "id in [" + strings.Join(func() []string {
		strIDs := make([]string, len(ids))
		for i, id := range ids {
			strIDs[i] = fmt.Sprintf("%d", id)
		}
		return strIDs
	}(), ",") + "]"

	// 删除数据
	err := milvusClient.Delete(ctx, collectionName, "", expr)
	if err != nil {
		stats.ErrorCount++
		return fmt.Errorf("批量删除向量失败: %v", err)
	}

	// 刷新数据
	err = milvusClient.Flush(ctx, collectionName, false)
	if err != nil {
		stats.ErrorCount++
		return fmt.Errorf("刷新数据失败: %v", err)
	}

	return nil
}

// GetClient 获取 Milvus 客户端
func GetClient() client.Client {
	return milvusClient
}
