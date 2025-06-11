package session

import (
	"context"
	"fmt"

	"server/framework/mysql"
)

// GetSessionInfo 获取会话信息
func GetSessionInfo(ctx context.Context, sessionID uint64) (map[string]interface{}, error) {
	var session mysql.Session
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

// AddMessage 添加消息到会话
func AddMessage(ctx context.Context, sessionID uint64, message *mysql.ChatRecord) error {
	// 验证会话是否存在
	var session mysql.Session
	if err := mysql.GetDB().Model(&mysql.Session{}).Where("id = ?", sessionID).First(&session).Error; err != nil {
		return fmt.Errorf("会话不存在: %v", err)
	}

	// 设置消息的会话ID
	message.SessionID = sessionID

	// 创建消息记录
	if err := mysql.GetDB().Model(&mysql.ChatRecord{}).Create(message).Error; err != nil {
		return fmt.Errorf("创建消息记录失败: %v", err)
	}

	// 更新会话的最后活动时间
	if err := mysql.GetDB().Model(&mysql.Session{}).Where("id = ?", sessionID).
		Update("last_active_time", message.CreatedAt).Error; err != nil {
		return fmt.Errorf("更新会话活动时间失败: %v", err)
	}

	return nil
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
			"message":    record.Message,
			"response":   record.Response,
			"created_at": record.CreatedAt,
		}
	}

	return result, nil
}

// GetUserPreferences 获取用户偏好
func GetUserPreferences(ctx context.Context, userID uint64) (map[string]interface{}, error) {
	var user mysql.User
	if err := mysql.GetDB().Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %v", err)
	}

	return map[string]interface{}{
		"user_id":    user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"status":     user.Status,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}, nil
}
