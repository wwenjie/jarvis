package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"server/framework/milvus"
	"server/framework/mysql"
	"server/framework/redis"
)

const (
	// 文档相关配置
	documentCachePrefix = "doc:"
	documentCacheTTL    = 24 * time.Hour // 文档缓存24小时
	searchCachePrefix   = "search:"
	searchCacheTTL      = 1 * time.Hour // 搜索结果缓存1小时
)

// Document 文档结构
type Document struct {
	ID        uint64          `json:"id"`
	UserID    uint64          `json:"user_id"`
	Title     string          `json:"title"`
	Content   string          `json:"content"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// DocumentMetadata 文档元数据
type DocumentMetadata struct {
	Embedding []float32 `json:"embedding"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Document   *Document `json:"document"`
	Score      float32   `json:"score"`
	Highlights []string  `json:"highlights"`
}

// DocumentSearchParams 文档搜索参数
type DocumentSearchParams struct {
	Query    string  `json:"query"`
	TopK     int     `json:"top_k"`
	MinScore float32 `json:"min_score"`
}

// IndexDocument 索引文档
func IndexDocument(ctx context.Context, doc *Document) error {
	// 1. 生成文档向量
	embedding, err := GetEmbedding(doc.Content)
	if err != nil {
		return fmt.Errorf("生成文档向量失败: %v", err)
	}

	// 2. 更新元数据
	metadata := DocumentMetadata{
		Embedding: embedding,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}
	doc.Metadata = metadataJSON

	// 3. 保存到数据库
	if err := mysql.GetDB().Create(doc).Error; err != nil {
		return fmt.Errorf("保存文档失败: %v", err)
	}

	// 4. 保存到 Milvus
	if err := milvus.InsertVector(ctx, "documents", int64(doc.ID), embedding); err != nil {
		// 如果 Milvus 插入失败，回滚数据库操作
		mysql.GetDB().Delete(&Document{}, doc.ID)
		return fmt.Errorf("保存向量失败: %v", err)
	}

	// 5. 缓存文档
	cacheKey := fmt.Sprintf("%s%d", documentCachePrefix, doc.ID)
	if docJSON, err := json.Marshal(doc); err == nil {
		redis.Set(ctx, cacheKey, string(docJSON), documentCacheTTL)
	}

	return nil
}

// SearchDocuments 搜索文档
func SearchDocuments(ctx context.Context, params *DocumentSearchParams) ([]*SearchResult, error) {
	// 1. 尝试从缓存获取
	cacheKey := fmt.Sprintf("%s%s:%d:%.2f", searchCachePrefix, params.Query, params.TopK, params.MinScore)
	if cached, err := redis.Get(ctx, cacheKey); err == nil {
		var results []*SearchResult
		if err := json.Unmarshal([]byte(cached), &results); err == nil {
			return results, nil
		}
	}

	// 2. 生成查询向量
	queryEmbedding, err := GetEmbedding(params.Query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}

	// 3. 在 Milvus 中搜索相似向量
	ids, scores, err := milvus.SearchVector(ctx, "documents", queryEmbedding, params.TopK*2) // 获取更多结果用于重排序
	if err != nil {
		return nil, fmt.Errorf("搜索向量失败: %v", err)
	}

	// 4. 获取对应的文档记录
	var documents []*Document
	if err := mysql.GetDB().Where("id IN ?", ids).Find(&documents).Error; err != nil {
		return nil, fmt.Errorf("获取文档记录失败: %v", err)
	}

	// 5. 创建 ID 到分数的映射
	scoreMap := make(map[uint64]float32)
	for i, id := range ids {
		scoreMap[uint64(id)] = scores[i]
	}

	// 6. 构建搜索结果
	results := make([]*SearchResult, 0, len(documents))
	for _, doc := range documents {
		score := scoreMap[doc.ID]
		if score < params.MinScore {
			continue
		}

		// 生成高亮文本
		highlights := generateHighlights(doc.Content, params.Query)

		results = append(results, &SearchResult{
			Document:   doc,
			Score:      score,
			Highlights: highlights,
		})
	}

	// 7. 按分数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 8. 只返回前 TopK 个结果
	if len(results) > params.TopK {
		results = results[:params.TopK]
	}

	// 9. 缓存搜索结果
	if resultsJSON, err := json.Marshal(results); err == nil {
		redis.Set(ctx, cacheKey, string(resultsJSON), searchCacheTTL)
	}

	return results, nil
}

// generateHighlights 生成高亮文本
func generateHighlights(content, query string) []string {
	// TODO: 实现文本高亮逻辑
	return []string{content}
}

// UpdateDocument 更新文档
func UpdateDocument(ctx context.Context, doc *Document) error {
	// 1. 生成新的文档向量
	embedding, err := GetEmbedding(doc.Content)
	if err != nil {
		return fmt.Errorf("生成文档向量失败: %v", err)
	}

	// 2. 更新元数据
	metadata := DocumentMetadata{
		Embedding: embedding,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}
	doc.Metadata = metadataJSON

	// 3. 更新数据库
	if err := mysql.GetDB().Save(doc).Error; err != nil {
		return fmt.Errorf("更新文档失败: %v", err)
	}

	// 4. 更新 Milvus
	if err := milvus.UpdateVector(ctx, "documents", int64(doc.ID), embedding); err != nil {
		// 如果 Milvus 更新失败，回滚数据库操作
		mysql.GetDB().Save(doc)
		return fmt.Errorf("更新向量失败: %v", err)
	}

	// 5. 更新缓存
	cacheKey := fmt.Sprintf("%s%d", documentCachePrefix, doc.ID)
	if docJSON, err := json.Marshal(doc); err == nil {
		redis.Set(ctx, cacheKey, string(docJSON), documentCacheTTL)
	}

	// 6. 删除相关的搜索结果缓存
	pattern := fmt.Sprintf("%s*", searchCachePrefix)
	keys, err := redis.Keys(ctx, pattern)
	if err == nil {
		for _, key := range keys {
			redis.Del(ctx, key)
		}
	}

	return nil
}

// DeleteDocument 删除文档
func DeleteDocument(ctx context.Context, docID uint64) error {
	// 1. 从数据库删除
	if err := mysql.GetDB().Delete(&Document{}, docID).Error; err != nil {
		return fmt.Errorf("删除文档失败: %v", err)
	}

	// 2. 从 Milvus 删除
	if err := milvus.DeleteVector(ctx, "documents", int64(docID)); err != nil {
		// 如果 Milvus 删除失败，回滚数据库操作
		mysql.GetDB().Unscoped().Model(&Document{}).Where("id = ?", docID).Update("deleted_at", nil)
		return fmt.Errorf("删除向量失败: %v", err)
	}

	// 3. 删除缓存
	cacheKey := fmt.Sprintf("%s%d", documentCachePrefix, docID)
	redis.Del(ctx, cacheKey)

	// 4. 删除相关的搜索结果缓存
	pattern := fmt.Sprintf("%s*", searchCachePrefix)
	keys, err := redis.Keys(ctx, pattern)
	if err == nil {
		for _, key := range keys {
			redis.Del(ctx, key)
		}
	}

	return nil
}
