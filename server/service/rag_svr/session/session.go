package session

import (
	"context"
	"fmt"

	"server/framework/mysql"
)

// GetSessionInfo 获取会话信息
func GetSessionInfo(ctx context.Context, sessionID uint64) (map[string]interface{}, error) {
	var session mysql.ChatSession
	if err := mysql.GetDB().Where("id = ?", sessionID).First(&session).Error; err != nil {
		return nil, fmt.Errorf("获取会话信息失败: %v", err)
	}

	return map[string]interface{}{
		"id":         session.ID,
		"user_id":    session.UserID,
		"title":      session.Title,
		"created_at": session.CreatedAt,
		"updated_at": session.UpdatedAt,
		"status":     session.Status,
	}, nil
}

// GetChatHistory 获取对话历史
func GetChatHistory(ctx context.Context, sessionID uint64, limit int) ([]map[string]interface{}, error) {
	var records []mysql.ChatRecord
	if err := mysql.GetDB().Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("获取对话历史失败: %v", err)
	}

	result := make([]map[string]interface{}, len(records))
	for i, record := range records {
		result[i] = map[string]interface{}{
			"id":         record.ID,
			"role":       record.Role,
			"content":    record.Content,
			"created_at": record.CreatedAt,
		}
	}

	return result, nil
}

// GetUserPreferences 获取用户偏好
func GetUserPreferences(ctx context.Context, userID uint64) (map[string]interface{}, error) {
	var preferences mysql.UserPreference
	if err := mysql.GetDB().Where("user_id = ?", userID).First(&preferences).Error; err != nil {
		return nil, fmt.Errorf("获取用户偏好失败: %v", err)
	}

	return map[string]interface{}{
		"user_id":      preferences.UserID,
		"language":     preferences.Language,
		"theme":        preferences.Theme,
		"notification": preferences.Notification,
		"created_at":   preferences.CreatedAt,
		"updated_at":   preferences.UpdatedAt,
	}, nil
}
