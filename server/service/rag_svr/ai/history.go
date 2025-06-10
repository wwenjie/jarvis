package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"server/framework/mysql"
	"server/framework/redis"
)

const (
	// 历史记录相关配置
	historyCachePrefix = "history:"
	historyCacheTTL    = 24 * time.Hour // 历史记录缓存24小时
	maxHistoryCount    = 100            // 每个用户最多保存100条历史记录
)

// HistoryType 历史记录类型
type HistoryType string

const (
	HistoryTypeWeather  HistoryType = "weather"  // 天气查询
	HistoryTypeSearch   HistoryType = "search"   // 文档搜索
	HistoryTypeReminder HistoryType = "reminder" // 提醒设置
)

// History 历史记录
type History struct {
	ID            uint64          `json:"id"`
	SessionID     uint64          `json:"session_id"`
	UserID        uint64          `json:"user_id"`
	Message       string          `json:"message"`
	Response      string          `json:"response"`
	MessageType   string          `json:"message_type"`
	Status        string          `json:"status"`
	Context       json.RawMessage `json:"context"`
	FunctionCalls json.RawMessage `json:"function_calls"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// HistoryMetadata 历史记录元数据
type HistoryMetadata struct {
	Query     string      `json:"query"`
	Result    string      `json:"result"`
	ExtraInfo interface{} `json:"extra_info,omitempty"`
}

// AddHistory 添加历史记录
func AddHistory(ctx context.Context, history *History) error {
	// 1. 保存到数据库
	if err := mysql.GetDB().Create(history).Error; err != nil {
		return fmt.Errorf("保存历史记录失败: %v", err)
	}

	// 2. 更新缓存
	cacheKey := fmt.Sprintf("%s%d", historyCachePrefix, history.UserID)
	if historyJSON, err := json.Marshal(history); err == nil {
		// 使用 Redis List 存储历史记录
		redis.LPush(ctx, cacheKey, string(historyJSON))
		// 限制列表长度
		redis.LTrim(ctx, cacheKey, 0, maxHistoryCount-1)
		// 设置过期时间
		redis.Expire(ctx, cacheKey, historyCacheTTL)
	}

	return nil
}

// GetHistory 获取历史记录
func GetHistory(ctx context.Context, userID uint64, messageType string, limit int) ([]*History, error) {
	if limit <= 0 || limit > maxHistoryCount {
		limit = maxHistoryCount
	}

	// 1. 尝试从缓存获取
	cacheKey := fmt.Sprintf("%s%d", historyCachePrefix, userID)
	historyJSONs, err := redis.LRange(ctx, cacheKey, 0, int64(limit-1))
	if err == nil && len(historyJSONs) > 0 {
		histories := make([]*History, 0, len(historyJSONs))
		for _, jsonStr := range historyJSONs {
			var history History
			if err := json.Unmarshal([]byte(jsonStr), &history); err == nil {
				if messageType == "" || history.MessageType == messageType {
					histories = append(histories, &history)
				}
			}
		}
		if len(histories) > 0 {
			return histories, nil
		}
	}

	// 2. 从数据库获取
	var histories []*History
	query := mysql.GetDB().Where("user_id = ?", userID)
	if messageType != "" {
		query = query.Where("message_type = ?", messageType)
	}
	if err := query.Order("created_at DESC").Limit(limit).Find(&histories).Error; err != nil {
		return nil, fmt.Errorf("获取历史记录失败: %v", err)
	}

	// 3. 更新缓存
	if len(histories) > 0 {
		for _, history := range histories {
			if historyJSON, err := json.Marshal(history); err == nil {
				redis.LPush(ctx, cacheKey, string(historyJSON))
			}
		}
		redis.LTrim(ctx, cacheKey, 0, maxHistoryCount-1)
		redis.Expire(ctx, cacheKey, historyCacheTTL)
	}

	return histories, nil
}

// ClearHistory 清除历史记录
func ClearHistory(ctx context.Context, userID uint64, messageType string) error {
	// 1. 从数据库删除
	query := mysql.GetDB().Where("user_id = ?", userID)
	if messageType != "" {
		query = query.Where("message_type = ?", messageType)
	}
	if err := query.Delete(&History{}).Error; err != nil {
		return fmt.Errorf("删除历史记录失败: %v", err)
	}

	// 2. 删除缓存
	cacheKey := fmt.Sprintf("%s%d", historyCachePrefix, userID)
	if messageType == "" {
		// 删除所有类型的历史记录
		redis.Del(ctx, cacheKey)
	} else {
		// 只删除指定类型的历史记录
		historyJSONs, err := redis.LRange(ctx, cacheKey, 0, -1)
		if err == nil {
			for _, jsonStr := range historyJSONs {
				var history History
				if err := json.Unmarshal([]byte(jsonStr), &history); err == nil {
					if history.MessageType != messageType {
						redis.RPush(ctx, cacheKey, jsonStr)
					}
				}
			}
		}
	}

	return nil
}
