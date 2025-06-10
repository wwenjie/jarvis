package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"server/framework/id_generator"
	"server/framework/milvus"
	"server/framework/mysql"
	"server/service/rag_svr/embedding"

	"gorm.io/gorm"
)

// 记忆类型
const (
	MemoryTypeFact       = "fact"       // 事实性记忆
	MemoryTypeReminder   = "reminder"   // 提醒类记忆
	MemoryTypePreference = "preference" // 用户偏好
	MemoryTypeContext    = "context"    // 上下文记忆

	// Milvus 集合名称
	MemoryCollectionName = "chat_memories"
	MemoryVectorDim      = 1536 // OpenAI text-embedding-ada-002 的维度
)

// 记忆结构
type Memory struct {
	ID           uint64                 `json:"id"`
	SessionID    uint64                 `json:"session_id"`
	UserID       uint64                 `json:"user_id"`
	Content      string                 `json:"content"`
	Type         string                 `json:"type"`
	Importance   float64                `json:"importance"` // 重要性评分
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`    // 过期时间
	LastAccessed time.Time              `json:"last_accessed"` // 最后访问时间
	AccessCount  int                    `json:"access_count"`  // 访问次数
	Metadata     map[string]interface{} `json:"metadata"`
	Similarity   float32                `json:"similarity"` // 相似度分数
}

// 记忆管理器
type MemoryManager struct {
	idGen *id_generator.IDGenerator
}

var instance *MemoryManager

// GetInstance 获取记忆管理器实例
func GetInstance() *MemoryManager {
	if instance == nil {
		instance = &MemoryManager{
			idGen: id_generator.GetInstance(),
		}
	}
	return instance
}

// AddMemory 添加记忆
func (m *MemoryManager) AddMemory(ctx context.Context, appID uint64, userID uint64, content string, memoryType string, importance float64, metadata map[string]interface{}, expiresAt *time.Time) error {
	// 生成记忆ID
	memoryID := m.idGen.GetMemoryID()

	// 获取向量表示
	embedding, err := embedding.GetEmbedding(content)
	if err != nil {
		return fmt.Errorf("获取向量表示失败: %v", err)
	}

	// 将 metadata 转换为 JSON 字符串
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("转换 metadata 失败: %v", err)
	}

	// 创建记忆记录
	memory := &mysql.ChatMemory{
		ID:         memoryID,
		SessionID:  appID, // 使用 appID 作为 sessionID
		UserID:     userID,
		Content:    content,
		MemoryType: memoryType,
		Importance: float32(importance),
		Metadata:   string(metadataJSON),
		CreatedAt:  time.Now(),
		ExpireTime: time.Now().AddDate(0, 0, 7), // 默认7天过期
	}

	// 设置过期时间
	if expiresAt != nil {
		memory.ExpireTime = *expiresAt
	}

	// 保存到数据库
	if err := mysql.GetDB().Create(memory).Error; err != nil {
		return fmt.Errorf("保存记忆失败: %v", err)
	}

	// 保存到 Milvus
	if err := milvus.InsertVector(ctx, MemoryCollectionName, int64(memoryID), embedding); err != nil {
		// 删除数据库记录
		mysql.GetDB().Delete(memory)
		return fmt.Errorf("保存向量失败: %v", err)
	}

	return nil
}

// SearchMemories 搜索相关记忆
func (m *MemoryManager) SearchMemories(ctx context.Context, userID uint64, query string, limit int) ([]*Memory, error) {
	// 生成查询向量
	queryEmbedding, err := embedding.GetEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}

	// 在 Milvus 中搜索相似向量
	ids, scores, err := milvus.SearchVector(ctx, MemoryCollectionName, queryEmbedding, limit*2) // 获取更多结果用于重排序
	if err != nil {
		return nil, fmt.Errorf("搜索向量失败: %v", err)
	}

	// 获取对应的记忆记录
	var memories []*mysql.ChatMemory
	if err := mysql.GetDB().Where("id IN ? AND user_id = ? AND expire_time > ?", ids, userID, time.Now()).Find(&memories).Error; err != nil {
		return nil, fmt.Errorf("获取记忆记录失败: %v", err)
	}

	// 创建 ID 到分数的映射
	scoreMap := make(map[uint64]float32)
	for i, id := range ids {
		scoreMap[uint64(id)] = scores[i]
	}

	// 转换为 Memory 结构并计算综合分数
	result := make([]*Memory, 0, len(memories))
	for _, mem := range memories {
		similarity := scoreMap[mem.ID]
		// 解析 metadata JSON 字符串
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(mem.Metadata), &metadata); err != nil {
			metadata = make(map[string]interface{})
		}

		result = append(result, &Memory{
			ID:           mem.ID,
			SessionID:    mem.SessionID,
			UserID:       mem.UserID,
			Content:      mem.Content,
			Type:         mem.MemoryType,
			Importance:   float64(mem.Importance),
			CreatedAt:    mem.CreatedAt,
			ExpiresAt:    mem.ExpireTime,
			LastAccessed: mem.UpdatedAt,
			AccessCount:  mem.AccessCount,
			Metadata:     metadata,
			Similarity:   similarity,
		})
	}

	// 重排序：综合考虑相似度、重要性和时间
	sort.Slice(result, func(i, j int) bool {
		// 计算综合分数
		scoreI := float64(result[i].Similarity)*0.6 + // 相似度权重 60%
			float64(result[i].Importance)*0.3 + // 重要性权重 30%
			float64(time.Since(result[i].LastAccessed).Hours())/float64(time.Since(result[i].CreatedAt).Hours())*0.1 // 时间衰减权重 10%

		scoreJ := float64(result[j].Similarity)*0.6 +
			float64(result[j].Importance)*0.3 +
			float64(time.Since(result[j].LastAccessed).Hours())/float64(time.Since(result[j].CreatedAt).Hours())*0.1

		return scoreI > scoreJ
	})

	// 只返回前 limit 个结果
	if len(result) > limit {
		result = result[:limit]
	}

	// 更新访问信息和计数
	for _, mem := range result {
		if err := mysql.GetDB().Model(&mysql.ChatMemory{}).
			Where("id = ?", mem.ID).
			Updates(map[string]interface{}{
				"updated_at":   time.Now(),
				"access_count": gorm.Expr("access_count + 1"),
			}).Error; err != nil {
			continue
		}
	}

	return result, nil
}

// CleanExpiredMemories 清理过期记忆
func (m *MemoryManager) CleanExpiredMemories(ctx context.Context) error {
	// 获取过期的记忆 ID
	var expiredMemories []*mysql.ChatMemory
	if err := mysql.GetDB().Where("expire_time < ?", time.Now()).Find(&expiredMemories).Error; err != nil {
		return fmt.Errorf("查询过期记忆失败: %v", err)
	}

	if len(expiredMemories) == 0 {
		return nil
	}

	// 收集过期的记忆 ID
	expiredIDs := make([]int64, len(expiredMemories))
	for i, memory := range expiredMemories {
		expiredIDs[i] = int64(memory.ID)
	}

	// 从 Milvus 中删除对应的向量
	for _, id := range expiredIDs {
		if err := milvus.DeleteVector(ctx, MemoryCollectionName, id); err != nil {
			return fmt.Errorf("删除 Milvus 向量失败: %v", err)
		}
	}

	// 删除 MySQL 中的记录
	if err := mysql.GetDB().Where("id IN ?", expiredIDs).Delete(&mysql.ChatMemory{}).Error; err != nil {
		return fmt.Errorf("删除过期记忆失败: %v", err)
	}

	return nil
}

// GetMemoryStats 获取记忆统计信息
func (m *MemoryManager) GetMemoryStats(ctx context.Context, userID uint64) (map[string]interface{}, error) {
	var stats []struct {
		Type           string  `gorm:"column:memory_type"`
		Count          int64   `gorm:"column:count"`
		AvgImportance  float64 `gorm:"column:avg_importance"`
		AvgAccessCount float64 `gorm:"column:avg_access_count"`
	}

	if err := mysql.GetDB().Table("chat_memories").
		Select("memory_type, COUNT(*) as count, AVG(importance) as avg_importance, AVG(access_count) as avg_access_count").
		Where("user_id = ?", userID).
		Group("memory_type").
		Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("获取记忆统计信息失败: %v", err)
	}

	result := make(map[string]interface{})
	for _, stat := range stats {
		result[stat.Type] = map[string]interface{}{
			"count":            stat.Count,
			"avg_importance":   stat.AvgImportance,
			"avg_access_count": stat.AvgAccessCount,
		}
	}

	return result, nil
}

// GetRelatedMemories 获取相关记忆
func (m *MemoryManager) GetRelatedMemories(ctx context.Context, memoryID uint64, limit int) ([]map[string]interface{}, error) {
	// 先获取当前记忆
	var currentMemory mysql.ChatMemory
	if err := mysql.GetDB().Where("id = ?", memoryID).First(&currentMemory).Error; err != nil {
		return nil, fmt.Errorf("获取当前记忆失败: %v", err)
	}

	// 获取相关记忆
	var relatedMemories []mysql.ChatMemory
	if err := mysql.GetDB().Where("user_id = ? AND id != ? AND memory_type = ?",
		currentMemory.UserID, memoryID, currentMemory.MemoryType).
		Order("importance DESC").
		Limit(limit).
		Find(&relatedMemories).Error; err != nil {
		return nil, fmt.Errorf("获取相关记忆失败: %v", err)
	}

	result := make([]map[string]interface{}, len(relatedMemories))
	for i, memory := range relatedMemories {
		result[i] = map[string]interface{}{
			"id":          memory.ID,
			"content":     memory.Content,
			"memory_type": memory.MemoryType,
			"importance":  memory.Importance,
			"created_at":  memory.CreatedAt,
			"expire_time": memory.ExpireTime,
		}
	}

	return result, nil
}
