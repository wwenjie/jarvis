package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"server/framework/id_generator"
	"server/framework/logger"
	"server/framework/milvus"
	"server/framework/mysql"
	"server/service/rag_svr/ai"
	rag_svr "server/service/rag_svr/kitex_gen/rag_svr"
	"server/service/rag_svr/memory"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RagServiceImpl implements the last service interface defined in the IDL.
type RagServiceImpl struct {
	qwenClient *ai.QwenClient
}

// Test implements the RagServiceImpl interface.
func (s *RagServiceImpl) Test(ctx context.Context, req *rag_svr.TestReq) (resp *rag_svr.TestRsp, err error) {
	logger.Infof("收到测试请求: seq_id=%d, msg=%s", req.GetSeqId(), req.GetMsg())

	return &rag_svr.TestRsp{
		Code: 0,
		Msg:  "test success",
	}, nil
}

// Test2 implements the RagServiceImpl interface.
func (s *RagServiceImpl) Test2(ctx context.Context, req *rag_svr.Test2Req) (resp *rag_svr.Test2Rsp, err error) {
	logger.Infof("收到测试请求: seq_id=%d, msg=%s", req.GetSeqId(), req.GetMsg())

	return &rag_svr.Test2Rsp{
		Code: 0,
		Msg:  "test2 success",
	}, nil
}

// CreateUser implements the RagServiceImpl interface.
func (s *RagServiceImpl) CreateUser(ctx context.Context, req *rag_svr.CreateUserReq) (resp *rag_svr.CreateUserRsp, err error) {
	logger.Infof("创建用户请求: username=%s, email=%s", req.Username, req.Email)

	userID := id_generator.GetInstance().GetUserID()
	if userID == 0 {
		logger.Errorf("获取用户ID失败")
		return &rag_svr.CreateUserRsp{
			Code: 1,
			Msg:  "获取用户ID失败",
		}, nil
	}

	// 创建用户
	user := &mysql.User{
		ID:       userID,
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // 注意：实际应用中应该对密码进行加密
	}

	if err := mysql.GetDB().Table("user").Create(user).Error; err != nil {
		logger.Errorf("创建用户失败: %v", err)
		return &rag_svr.CreateUserRsp{
			Code: 1,
			Msg:  fmt.Sprintf("创建用户失败: %v", err),
		}, nil
	}

	logger.Infof("用户创建成功: user_id=%d", user.ID)

	// 返回用户信息
	return &rag_svr.CreateUserRsp{
		Code: 0,
		Msg:  "success",
		UserInfo: &rag_svr.UserInfo{
			UserId:     user.ID,
			UserName:   user.Username,
			Email:      user.Email,
			CreateTime: uint64(user.CreatedAt.Unix()),
			UpdateTime: uint64(user.UpdatedAt.Unix()),
		},
	}, nil
}

// CreateSession implements the RagServiceImpl interface.
func (s *RagServiceImpl) CreateSession(ctx context.Context, req *rag_svr.CreateSessionReq) (resp *rag_svr.CreateSessionRsp, err error) {
	logger.Infof("创建会话请求: user_id=%d", req.UserId)

	sessionID := id_generator.GetInstance().GetSessionID()
	if sessionID == 0 {
		logger.Errorf("获取会话ID失败")
		return &rag_svr.CreateSessionRsp{
			Code: 1,
			Msg:  "获取会话ID失败",
		}, nil
	}
	// 创建会话
	session := &mysql.ChatSession{
		ID:             sessionID,
		UserID:         req.UserId,
		Status:         "active",
		LastActiveTime: time.Now(),
	}

	// 初始化用户状态
	if err := session.SetUserState(make(map[string]string)); err != nil {
		logger.Errorf("初始化用户状态失败: %v", err)
		return &rag_svr.CreateSessionRsp{
			Code: 1,
			Msg:  fmt.Sprintf("初始化用户状态失败: %v", err),
		}, nil
	}

	// 初始化系统状态
	if err := session.SetSystemState(make(map[string]string)); err != nil {
		logger.Errorf("初始化系统状态失败: %v", err)
		return &rag_svr.CreateSessionRsp{
			Code: 1,
			Msg:  fmt.Sprintf("初始化系统状态失败: %v", err),
		}, nil
	}

	// 初始化元数据
	if err := session.SetMetadata(make(map[string]interface{})); err != nil {
		logger.Errorf("初始化元数据失败: %v", err)
		return &rag_svr.CreateSessionRsp{
			Code: 1,
			Msg:  fmt.Sprintf("初始化元数据失败: %v", err),
		}, nil
	}

	if err := mysql.GetDB().Table("chat_session").Create(session).Error; err != nil {
		logger.Errorf("创建会话失败: %v", err)
		return &rag_svr.CreateSessionRsp{
			Code: 1,
			Msg:  fmt.Sprintf("创建会话失败: %v", err),
		}, nil
	}

	logger.Infof("会话创建成功: session_id=%d", session.ID)

	// 获取用户状态和系统状态
	userState, err := session.GetUserState()
	if err != nil {
		logger.Errorf("获取用户状态失败: %v", err)
		userState = make(map[string]string)
	}

	systemState, err := session.GetSystemState()
	if err != nil {
		logger.Errorf("获取系统状态失败: %v", err)
		systemState = make(map[string]string)
	}

	// 获取元数据
	metadata, err := session.GetMetadata()
	if err != nil {
		logger.Errorf("获取元数据失败: %v", err)
		metadata = make(map[string]interface{})
	}

	// 转换为字符串类型的元数据
	metadataStr := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			metadataStr[k] = str
		} else {
			// 将非字符串类型转换为 JSON 字符串
			if jsonStr, err := json.Marshal(v); err == nil {
				metadataStr[k] = string(jsonStr)
			}
		}
	}

	// 返回会话信息
	return &rag_svr.CreateSessionRsp{
		Code: 0,
		Msg:  "success",
		SessionInfo: &rag_svr.SessionInfo{
			SessionId:   session.ID,
			UserId:      session.UserID,
			Title:       session.Title,
			Summary:     session.Summary,
			Status:      session.Status,
			CreateTime:  uint64(session.CreatedAt.Unix()),
			UpdateTime:  uint64(session.UpdatedAt.Unix()),
			UserState:   userState,
			SystemState: systemState,
			Metadata:    metadataStr,
		},
	}, nil
}

// EndSession implements the RagServiceImpl interface.
func (s *RagServiceImpl) EndSession(ctx context.Context, req *rag_svr.EndSessionReq) (resp *rag_svr.EndSessionRsp, err error) {
	logger.Infof("结束会话请求: session_id=%d", req.SessionId)

	// 更新会话状态
	if err := mysql.GetDB().Table("chat_session").Model(&mysql.ChatSession{}).
		Where("id = ?", req.SessionId).
		Update("status", "closed").Error; err != nil {
		logger.Errorf("结束会话失败: %v", err)
		return &rag_svr.EndSessionRsp{
			Code: 1,
			Msg:  fmt.Sprintf("结束会话失败: %v", err),
		}, nil
	}

	logger.Infof("会话已结束: session_id=%d", req.SessionId)

	return &rag_svr.EndSessionRsp{
		Code: 0,
		Msg:  "success",
	}, nil
}

// extractAndStoreMemories 提取并存储重要信息
func (s *RagServiceImpl) extractAndStoreMemories(ctx context.Context, userID, sessionID uint64, message, response string) error {
	memoryManager := memory.GetInstance()

	// 分析用户消息和AI回复
	intent, err := ai.AnalyzeIntent(message)
	if err != nil {
		return fmt.Errorf("分析意图失败: %v", err)
	}

	sentiment, err := ai.AnalyzeSentiment(message)
	if err != nil {
		return fmt.Errorf("分析情感失败: %v", err)
	}

	// 根据意图类型提取不同类型的记忆
	switch intent.Intent {
	case ai.IntentTypeQuestion:
		// 提取事实性记忆
		if err := memoryManager.AddMemory(ctx, sessionID, userID,
			message,
			memory.MemoryTypeFact,
			0.8,
			map[string]interface{}{
				"intent":               intent.Intent,
				"intent_confidence":    intent.Confidence,
				"entities":             intent.Entities,
				"sentiment":            sentiment.Sentiment,
				"sentiment_confidence": sentiment.Confidence,
				"emotions":             sentiment.Emotions,
				"source":               "question",
			},
			nil,
		); err != nil {
			return fmt.Errorf("存储事实性记忆失败: %v", err)
		}

	case ai.IntentTypeCommand:
		// 提取命令相关的上下文记忆
		if err := memoryManager.AddMemory(ctx, sessionID, userID,
			message,
			memory.MemoryTypeContext,
			0.9,
			map[string]interface{}{
				"intent":               intent.Intent,
				"intent_confidence":    intent.Confidence,
				"entities":             intent.Entities,
				"sentiment":            sentiment.Sentiment,
				"sentiment_confidence": sentiment.Confidence,
				"emotions":             sentiment.Emotions,
				"source":               "command",
				"response":             response,
			},
			nil,
		); err != nil {
			return fmt.Errorf("存储命令记忆失败: %v", err)
		}

	case ai.IntentTypeTask:
		// 提取任务相关的记忆
		if err := memoryManager.AddMemory(ctx, sessionID, userID,
			message,
			memory.MemoryTypeContext,
			0.7,
			map[string]interface{}{
				"intent":               intent.Intent,
				"intent_confidence":    intent.Confidence,
				"entities":             intent.Entities,
				"sentiment":            sentiment.Sentiment,
				"sentiment_confidence": sentiment.Confidence,
				"emotions":             sentiment.Emotions,
				"source":               "task",
				"response":             response,
			},
			nil,
		); err != nil {
			return fmt.Errorf("存储任务记忆失败: %v", err)
		}

	case ai.IntentTypeChat:
		// 提取用户偏好
		if sentiment.Confidence > 0.8 {
			// 当情感分析置信度高时，提取用户偏好
			if err := memoryManager.AddMemory(ctx, sessionID, userID,
				message,
				memory.MemoryTypePreference,
				0.6,
				map[string]interface{}{
					"intent":               intent.Intent,
					"intent_confidence":    intent.Confidence,
					"entities":             intent.Entities,
					"sentiment":            sentiment.Sentiment,
					"sentiment_confidence": sentiment.Confidence,
					"emotions":             sentiment.Emotions,
					"source":               "chat",
				},
				nil,
			); err != nil {
				return fmt.Errorf("存储用户偏好失败: %v", err)
			}
		}
	}

	// 检查是否包含时间相关的实体
	for _, entity := range intent.Entities {
		if entity.Type == "time" || entity.Type == "date" {
			// 提取提醒类记忆
			if err := memoryManager.AddMemory(ctx, sessionID, userID,
				message,
				memory.MemoryTypeReminder,
				0.9,
				map[string]interface{}{
					"intent":               intent.Intent,
					"intent_confidence":    intent.Confidence,
					"entities":             intent.Entities,
					"sentiment":            sentiment.Sentiment,
					"sentiment_confidence": sentiment.Confidence,
					"emotions":             sentiment.Emotions,
					"source":               "reminder",
					"time_entity":          entity.Value,
				},
				nil,
			); err != nil {
				return fmt.Errorf("存储提醒记忆失败: %v", err)
			}
		}
	}

	return nil
}

// AddDocument 实现添加文档
func (s *RagServiceImpl) AddDocument(ctx context.Context, req *rag_svr.AddDocumentReq) (resp *rag_svr.AddDocumentRsp, err error) {
	logger.Infof("添加文档请求: user_id=%d, title=%s", req.UserId, req.Title)

	// 参数校验
	if req.UserId == 0 || req.Title == "" {
		return &rag_svr.AddDocumentRsp{
			Code: 1,
			Msg:  "用户ID和标题不能为空",
		}, nil
	}

	docID, err := ai.GetDocumentServiceInstance().AddDocument(ctx, req)
	if err != nil {
		logger.Errorf("添加文档失败: %v", err)
		return &rag_svr.AddDocumentRsp{
			Code: 1,
			Msg:  err.Error(),
		}, nil
	}

	logger.Infof("文档添加成功: doc_id=%d", docID)
	return &rag_svr.AddDocumentRsp{
		Code:  0,
		Msg:   "success",
		DocId: docID,
	}, nil
}

// filterEmptyStrings 过滤空字符串
func filterEmptyStrings(strs []string) []string {
	var result []string
	for _, str := range strs {
		if str = strings.TrimSpace(str); str != "" {
			result = append(result, str)
		}
	}
	return result
}

// SearchDocument 实现搜索文档
func (s *RagServiceImpl) SearchDocument(ctx context.Context, req *rag_svr.SearchDocumentReq) (resp *rag_svr.SearchDocumentRsp, err error) {
	logger.Infof("搜索文档请求: user_id=%d, query=%s, top_k=%d", req.UserId, req.Query, req.TopK)
	return ai.GetDocumentServiceInstance().SearchDocument(ctx, req)
}

// calculateRelevanceScore 计算相关性分数
func calculateRelevanceScore(chunk mysql.DocumentChunk, queryKeywords []string) float32 {
	// 解析块关键词
	var chunkKeywords []string
	if err := json.Unmarshal([]byte(chunk.Keywords), &chunkKeywords); err != nil {
		return 0
	}

	// 计算关键词匹配度
	matchedKeywords := 0
	for _, qk := range queryKeywords {
		for _, ck := range chunkKeywords {
			if qk == ck {
				matchedKeywords++
				break
			}
		}
	}

	// 计算分数 (关键词匹配度 / 查询关键词总数)
	return float32(matchedKeywords) / float32(len(queryKeywords))
}

// GetSessionList implements the RagServiceImpl interface.
func (s *RagServiceImpl) GetSessionList(ctx context.Context, req *rag_svr.GetSessionListReq) (resp *rag_svr.GetSessionListRsp, err error) {
	logger.Infof("获取会话列表请求: user_id=%d, status=%s, page=%d, page_size=%d",
		req.UserId, req.Status, req.Page, req.PageSize)

	// 获取用户的会话列表
	var sessions []mysql.ChatSession
	query := mysql.GetDB().Table("chat_session").Model(&mysql.ChatSession{}).Where("user_id = ?", req.UserId)

	// 根据状态筛选
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 根据时间范围筛选
	if req.StartTime != "" {
		query = query.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		query = query.Where("created_at <= ?", req.EndTime)
	}

	// 分页
	offset := int((req.Page - 1) * req.PageSize)
	if err := query.Order("last_active_time DESC").
		Offset(offset).
		Limit(int(req.PageSize)).
		Find(&sessions).Error; err != nil {
		logger.Errorf("获取会话列表失败: %v", err)
		return &rag_svr.GetSessionListRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取会话列表失败: %v", err),
		}, nil
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取会话总数失败: %v", err)
		return &rag_svr.GetSessionListRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取会话总数失败: %v", err),
		}, nil
	}

	logger.Infof("获取会话列表成功: 总数=%d, 当前页=%d", total, len(sessions))

	// 构建响应
	sessionInfos := make([]*rag_svr.SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		// 获取用户状态
		userState, err := session.GetUserState()
		if err != nil {
			logger.Errorf("获取用户状态失败: %v", err)
			userState = make(map[string]string)
		}

		// 获取系统状态
		systemState, err := session.GetSystemState()
		if err != nil {
			logger.Errorf("获取系统状态失败: %v", err)
			systemState = make(map[string]string)
		}

		// 获取元数据
		metadata, err := session.GetMetadata()
		if err != nil {
			logger.Errorf("获取元数据失败: %v", err)
			metadata = make(map[string]interface{})
		}

		// 转换为字符串类型的元数据
		metadataStr := make(map[string]string)
		for k, v := range metadata {
			if str, ok := v.(string); ok {
				metadataStr[k] = str
			} else {
				// 将非字符串类型转换为 JSON 字符串
				if jsonStr, err := json.Marshal(v); err == nil {
					metadataStr[k] = string(jsonStr)
				}
			}
		}

		sessionInfos = append(sessionInfos, &rag_svr.SessionInfo{
			SessionId:   session.ID,
			UserId:      session.UserID,
			Title:       session.Title,
			Summary:     session.Summary,
			Status:      session.Status,
			CreateTime:  uint64(session.CreatedAt.Unix()),
			UpdateTime:  uint64(session.UpdatedAt.Unix()),
			UserState:   userState,
			SystemState: systemState,
			Metadata:    metadataStr,
		})
	}

	return &rag_svr.GetSessionListRsp{
		Code:        0,
		Msg:         "success",
		Total:       total,
		Page:        req.Page,
		PageSize:    req.PageSize,
		SessionList: sessionInfos,
	}, nil
}

// CleanInactiveSessions 清理不活跃的会话
func (s *RagServiceImpl) CleanInactiveSessions(ctx context.Context, req *rag_svr.CleanInactiveSessionsReq) (resp *rag_svr.CleanInactiveSessionsRsp, err error) {
	logger.Infof("清理不活跃会话请求: inactive_days=%d", req.InactiveDays)

	// 获取不活跃时间阈值
	inactiveTime := time.Now().Add(-time.Duration(req.InactiveDays) * 24 * time.Hour)

	// 更新会话状态
	result := mysql.GetDB().Table("chat_session").Model(&mysql.ChatSession{}).
		Where("status = ? AND last_active_time < ?", "active", inactiveTime).
		Update("status", "archived")

	if result.Error != nil {
		logger.Errorf("清理不活跃会话失败: %v", result.Error)
		return &rag_svr.CleanInactiveSessionsRsp{
			Code: 1,
			Msg:  fmt.Sprintf("清理不活跃会话失败: %v", result.Error),
		}, nil
	}

	logger.Infof("清理不活跃会话成功: 清理数量=%d", result.RowsAffected)

	return &rag_svr.CleanInactiveSessionsRsp{
		Code:         0,
		Msg:          "success",
		CleanedCount: result.RowsAffected,
	}, nil
}

// ListDocument 实现搜索文档
func (s *RagServiceImpl) ListDocument(ctx context.Context, req *rag_svr.ListDocumentReq) (resp *rag_svr.ListDocumentRsp, err error) {
	logger.Infof("获取文档列表请求: user_id=%d, page=%d, page_size=%d", req.UserId, req.Page, req.PageSize)

	// 获取用户的文档列表
	var documents []mysql.Document
	query := mysql.GetDB().Table("document").Model(&mysql.Document{})

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取文档总数失败: %v", err)
		return &rag_svr.ListDocumentRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取文档总数失败: %v", err),
		}, nil
	}

	// 分页
	offset := int((req.Page - 1) * req.PageSize)
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(int(req.PageSize)).
		Find(&documents).Error; err != nil {
		logger.Errorf("获取文档列表失败: %v", err)
		return &rag_svr.ListDocumentRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取文档列表失败: %v", err),
		}, nil
	}

	logger.Infof("获取文档列表成功: 总数=%d, 当前页=%d", total, len(documents))

	// 构建响应
	docList := make([]*rag_svr.Document, 0, len(documents))
	for _, doc := range documents {
		docList = append(docList, &rag_svr.Document{
			DocId:      doc.DocID,
			UserId:     doc.UserID,
			Title:      doc.Title,
			Content:    "", // 文档内容不返回
			Metadata:   doc.Metadata,
			CreateTime: uint64(doc.CreatedAt.Unix()),
			UpdateTime: uint64(doc.UpdatedAt.Unix()),
		})
	}

	return &rag_svr.ListDocumentRsp{
		Code:      0,
		Msg:       "success",
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
		Documents: docList,
	}, nil
}

// GetSession implements the RagServiceImpl interface.
func (s *RagServiceImpl) GetSession(ctx context.Context, req *rag_svr.GetSessionReq) (resp *rag_svr.GetSessionRsp, err error) {
	logger.Infof("获取会话详情请求: session_id=%d, user_id=%d", req.SessionId, req.UserId)

	// 获取会话信息
	var session mysql.ChatSession
	if err := mysql.GetDB().Table("chat_session").Model(&mysql.ChatSession{}).First(&session, req.SessionId).Error; err != nil {
		logger.Errorf("获取会话信息失败: %v", err)
		return &rag_svr.GetSessionRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取会话信息失败: %v", err),
		}, nil
	}

	// 验证用户权限
	if session.UserID != req.UserId {
		logger.Errorf("用户无权限访问该会话: user_id=%d, session_user_id=%d", req.UserId, session.UserID)
		return &rag_svr.GetSessionRsp{
			Code: 1,
			Msg:  "无权限访问该会话",
		}, nil
	}

	// 获取用户状态
	userState, err := session.GetUserState()
	if err != nil {
		logger.Errorf("获取用户状态失败: %v", err)
		userState = make(map[string]string)
	}

	// 获取系统状态
	systemState, err := session.GetSystemState()
	if err != nil {
		logger.Errorf("获取系统状态失败: %v", err)
		systemState = make(map[string]string)
	}

	// 获取元数据
	metadata, err := session.GetMetadata()
	if err != nil {
		logger.Errorf("获取元数据失败: %v", err)
		metadata = make(map[string]interface{})
	}

	// 转换为字符串类型的元数据
	metadataStr := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			metadataStr[k] = str
		} else {
			// 将非字符串类型转换为 JSON 字符串
			if jsonStr, err := json.Marshal(v); err == nil {
				metadataStr[k] = string(jsonStr)
			}
		}
	}

	// 获取会话的对话记录
	var chatRecords []mysql.ChatRecord
	if err := mysql.GetDB().Table("chat_record").Model(&mysql.ChatRecord{}).
		Where("session_id = ?", req.SessionId).
		Order("created_at ASC").
		Find(&chatRecords).Error; err != nil {
		logger.Errorf("获取对话记录失败: %v", err)
	}

	// 构建响应
	return &rag_svr.GetSessionRsp{
		Code: 0,
		Msg:  "success",
		SessionInfo: &rag_svr.SessionInfo{
			SessionId:   session.ID,
			UserId:      session.UserID,
			Title:       session.Title,
			Summary:     session.Summary,
			Status:      session.Status,
			CreateTime:  uint64(session.CreatedAt.Unix()),
			UpdateTime:  uint64(session.UpdatedAt.Unix()),
			UserState:   userState,
			SystemState: systemState,
			Metadata:    metadataStr,
			ChatRecords: func() []*rag_svr.ChatRecord {
				records := make([]*rag_svr.ChatRecord, 0, len(chatRecords))
				for _, record := range chatRecords {
					records = append(records, &rag_svr.ChatRecord{
						ChatId:      record.ID,
						SessionId:   record.SessionID,
						UserId:      record.UserID,
						Message:     record.Message,
						Response:    record.Response,
						CreateTime:  uint64(record.CreatedAt.Unix()),
						MessageType: record.MessageType,
						Status:      record.Status,
					})
				}
				return records
			}(),
		},
	}, nil
}

// DeleteDocument 实现删除文档
func (s *RagServiceImpl) DeleteDocument(ctx context.Context, req *rag_svr.DeleteDocumentReq) (resp *rag_svr.DeleteDocumentRsp, err error) {
	logger.Infof("删除文档请求: doc_id=%d, user_id=%d", req.DocId, req.UserId)
	return ai.GetDocumentServiceInstance().DeleteDocument(ctx, req)
}

// AddChatRecord 实现添加聊天记录
func (s *RagServiceImpl) AddChatRecord(ctx context.Context, req *rag_svr.AddChatRecordReq) (resp *rag_svr.AddChatRecordRsp, err error) {
	logger.Infof("添加聊天记录请求: session_id=%d, user_id=%d", req.SessionId, req.UserId)

	chatRecordID := id_generator.GetInstance().GetChatRecordID()
	if chatRecordID == 0 {
		logger.Errorf("获取聊天记录ID失败")
		return &rag_svr.AddChatRecordRsp{
			Code: 1,
			Msg:  "获取聊天记录ID失败",
		}, nil
	}

	// 创建聊天记录
	chatRecord := &mysql.ChatRecord{
		ID:            chatRecordID,
		SessionID:     req.SessionId,
		UserID:        req.UserId,
		Message:       req.Message,
		Response:      req.Response,
		MessageType:   req.MessageType,
		Status:        "completed",
		Context:       req.Context,
		FunctionCalls: req.FunctionCalls,
		Metadata:      req.Metadata,
	}

	if err := mysql.GetDB().Table("chat_record").Create(chatRecord).Error; err != nil {
		logger.Errorf("创建聊天记录失败: %v", err)
		return &rag_svr.AddChatRecordRsp{
			Code: 1,
			Msg:  fmt.Sprintf("创建聊天记录失败: %v", err),
		}, nil
	}

	logger.Infof("聊天记录添加成功: chat_id=%d", chatRecord.ID)

	return &rag_svr.AddChatRecordRsp{
		Code:   0,
		Msg:    "success",
		ChatId: chatRecord.ID,
	}, nil
}

// GetChatRecords 实现获取聊天记录列表
func (s *RagServiceImpl) GetChatRecords(ctx context.Context, req *rag_svr.GetChatRecordsReq) (resp *rag_svr.GetChatRecordsRsp, err error) {
	logger.Infof("获取聊天记录列表请求: session_id=%d, page=%d, page_size=%d", req.SessionId, req.Page, req.PageSize)

	// 获取聊天记录列表
	var records []mysql.ChatRecord
	query := mysql.GetDB().Table("chat_record").Model(&mysql.ChatRecord{}).Where("session_id = ?", req.SessionId)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取聊天记录总数失败: %v", err)
		return &rag_svr.GetChatRecordsRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取聊天记录总数失败: %v", err),
		}, nil
	}

	// 分页
	offset := int((req.Page - 1) * req.PageSize)
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(int(req.PageSize)).
		Find(&records).Error; err != nil {
		logger.Errorf("获取聊天记录列表失败: %v", err)
		return &rag_svr.GetChatRecordsRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取聊天记录列表失败: %v", err),
		}, nil
	}

	logger.Infof("获取聊天记录列表成功: 总数=%d, 当前页=%d", total, len(records))

	// 构建响应
	chatRecords := make([]*rag_svr.ChatRecord, 0, len(records))
	for _, record := range records {
		chatRecords = append(chatRecords, &rag_svr.ChatRecord{
			ChatId:      record.ID,
			SessionId:   record.SessionID,
			UserId:      record.UserID,
			Message:     record.Message,
			Response:    record.Response,
			MessageType: record.MessageType,
			Status:      record.Status,
			CreateTime:  uint64(record.CreatedAt.Unix()),
		})
	}

	return &rag_svr.GetChatRecordsRsp{
		Code:     0,
		Msg:      "success",
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Records:  chatRecords,
	}, nil
}

// AddMemory implements the RagServiceImpl interface.
func (s *RagServiceImpl) AddMemory(ctx context.Context, req *rag_svr.AddMemoryReq) (resp *rag_svr.AddMemoryRsp, err error) {
	logger.Infof("添加记忆请求: user_id=%d, memory_type=%s", req.UserId, req.MemoryType)

	// 获取记忆管理器实例
	memoryManager := memory.GetInstance()

	// 解析 metadata
	var metadata map[string]interface{}
	if req.Metadata != "" {
		if err := json.Unmarshal([]byte(req.Metadata), &metadata); err != nil {
			logger.Errorf("解析 metadata 失败: %v", err)
			return &rag_svr.AddMemoryRsp{
				Code: 1,
				Msg:  fmt.Sprintf("解析 metadata 失败: %v", err),
			}, nil
		}
	}

	// 添加记忆
	if err := memoryManager.AddMemory(ctx, req.SessionId, req.UserId, req.Content, req.MemoryType, req.Importance, metadata, nil); err != nil {
		logger.Errorf("添加记忆失败: %v", err)
		return &rag_svr.AddMemoryRsp{
			Code: 1,
			Msg:  fmt.Sprintf("添加记忆失败: %v", err),
		}, nil
	}

	logger.Infof("记忆添加成功")

	return &rag_svr.AddMemoryRsp{
		Code: 0,
		Msg:  "success",
	}, nil
}

// verifyMemoryAccess 验证用户是否有权限访问记忆
func (s *RagServiceImpl) verifyMemoryAccess(ctx context.Context, memoryID uint64, userID uint64) error {
	var memory mysql.ChatMemory
	if err := mysql.GetDB().Table("chat_memory").Model(&mysql.ChatMemory{}).First(&memory, memoryID).Error; err != nil {
		return fmt.Errorf("获取记忆失败: %v", err)
	}

	if memory.UserID != userID {
		return fmt.Errorf("无权限访问该记忆")
	}

	return nil
}

// GetMemory implements the RagServiceImpl interface.
func (s *RagServiceImpl) GetMemory(ctx context.Context, req *rag_svr.GetMemoryReq) (resp *rag_svr.GetMemoryRsp, err error) {
	logger.Infof("获取记忆请求: memory_id=%d", req.MemoryId)

	// 验证权限
	if err := s.verifyMemoryAccess(ctx, req.MemoryId, req.UserId); err != nil {
		logger.Errorf("权限验证失败: %v", err)
		return &rag_svr.GetMemoryRsp{
			Code: 1,
			Msg:  fmt.Sprintf("权限验证失败: %v", err),
		}, nil
	}

	// 获取记忆管理器实例
	memoryManager := memory.GetInstance()

	// 获取记忆
	memories, err := memoryManager.SearchMemories(ctx, fmt.Sprintf("id:%d", req.MemoryId), 1)
	if err != nil {
		logger.Errorf("获取记忆失败: %v", err)
		return &rag_svr.GetMemoryRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取记忆失败: %v", err),
		}, nil
	}

	if len(memories) == 0 {
		return &rag_svr.GetMemoryRsp{
			Code: 1,
			Msg:  "记忆不存在",
		}, nil
	}

	memory := memories[0]
	// 将 metadata 转换为 JSON 字符串
	metadataJSON, err := json.Marshal(memory.Metadata)
	if err != nil {
		logger.Errorf("序列化 metadata 失败: %v", err)
		return &rag_svr.GetMemoryRsp{
			Code: 1,
			Msg:  fmt.Sprintf("序列化 metadata 失败: %v", err),
		}, nil
	}

	return &rag_svr.GetMemoryRsp{
		Code: 0,
		Msg:  "success",
		Memory: &rag_svr.Memory{
			MemoryId:    memory.ID,
			SessionId:   memory.SessionID,
			UserId:      memory.UserID,
			Content:     memory.Content,
			MemoryType:  memory.Type,
			Importance:  memory.Importance,
			Metadata:    string(metadataJSON),
			CreateTime:  uint64(memory.CreatedAt.Unix()),
			UpdateTime:  uint64(memory.LastAccessed.Unix()),
			ExpireTime:  uint64(memory.ExpiresAt.Unix()),
			AccessCount: int32(memory.AccessCount),
		},
	}, nil
}

// SearchMemories 搜索记忆
func (s *RagServiceImpl) SearchMemories(ctx context.Context, req *rag_svr.SearchMemoriesReq) (resp *rag_svr.SearchMemoriesRsp, err error) {
	logger.Infof("搜索记忆请求: query=%s, limit=%d", req.Query, req.Limit)

	// 调用记忆管理器搜索记忆
	memories, err := memory.GetInstance().SearchMemories(ctx, req.Query, int(req.Limit))
	if err != nil {
		logger.Errorf("搜索记忆失败: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("搜索记忆失败: %v", err))
	}

	// 转换为响应格式
	memoryList := make([]*rag_svr.Memory, 0, len(memories))
	for _, mem := range memories {
		// 将 metadata 转换为 JSON 字符串
		metadataJSON, err := json.Marshal(mem.Metadata)
		if err != nil {
			logger.Errorf("序列化 metadata 失败: %v", err)
			continue
		}

		memoryList = append(memoryList, &rag_svr.Memory{
			MemoryId:    mem.ID,
			SessionId:   mem.SessionID,
			UserId:      mem.UserID,
			Content:     mem.Content,
			MemoryType:  mem.Type,
			Importance:  float64(mem.Importance),
			Metadata:    string(metadataJSON),
			CreateTime:  uint64(mem.CreatedAt.Unix()),
			UpdateTime:  uint64(mem.LastAccessed.Unix()),
			ExpireTime:  uint64(mem.ExpiresAt.Unix()),
			AccessCount: int32(mem.AccessCount),
		})
	}

	memoryIdList := make([]uint64, 0, len(memoryList))
	for _, mem := range memoryList {
		memoryIdList = append(memoryIdList, mem.MemoryId)
	}

	logger.Infof("搜索记忆完成: 找到%d个结果, query: %s, limit: %d, memoryIdList: %v", len(memoryList), req.Query, req.Limit, memoryIdList)
	return &rag_svr.SearchMemoriesRsp{
		Code:     0,
		Msg:      "success",
		Memories: memoryList,
	}, nil
}

// DeleteMemory implements the RagServiceImpl interface.
func (s *RagServiceImpl) DeleteMemory(ctx context.Context, req *rag_svr.DeleteMemoryReq) (resp *rag_svr.DeleteMemoryRsp, err error) {
	logger.Infof("删除记忆请求: memory_id=%d", req.MemoryId)

	// 验证权限
	if err := s.verifyMemoryAccess(ctx, req.MemoryId, req.UserId); err != nil {
		logger.Errorf("权限验证失败: %v, memoryId: %d, userId: %d", err, req.MemoryId, req.UserId)
		return &rag_svr.DeleteMemoryRsp{
			Code: 1,
			Msg:  fmt.Sprintf("权限验证失败: %v", err),
		}, nil
	}

	// 从数据库中删除记忆
	if err := mysql.GetDB().Table("chat_memory").Where("id = ?", req.MemoryId).Delete(&mysql.ChatMemory{}).Error; err != nil {
		logger.Errorf("删除记忆失败: %v, memoryId: %d, userId: %d", err, req.MemoryId, req.UserId)
		return &rag_svr.DeleteMemoryRsp{
			Code: 1,
			Msg:  fmt.Sprintf("删除记忆失败: %v", err),
		}, nil
	}

	// 从 Milvus 中删除向量
	err = milvus.DeleteVector(ctx, milvus.MemoryCollectionName, int64(req.MemoryId))
	if err != nil {
		logger.Errorf("删除 Milvus 向量失败: %v, memoryId: %d, userId: %d", err, req.MemoryId, req.UserId)
		// 这里可以选择忽略错误，或者返回部分成功
	}

	logger.Infof("记忆删除成功, memoryId: %d, userId: %d", req.MemoryId, req.UserId)

	return &rag_svr.DeleteMemoryRsp{
		Code: 0,
		Msg:  "success",
	}, nil
}

// GetWeather implements the RagServiceImpl interface.
func (s *RagServiceImpl) GetWeather(ctx context.Context, req *rag_svr.GetWeatherReq) (resp *rag_svr.GetWeatherRsp, err error) {
	resp = &rag_svr.GetWeatherRsp{
		Code: 0,
		Msg:  "success",
	}

	// 获取天气信息
	weather, err := ai.GetWeather(ctx, req.Location)
	if err != nil {
		resp.Code = 500
		resp.Msg = fmt.Sprintf("获取天气信息失败: %v", err)
		return resp, nil
	}

	// 转换天气信息
	resp.Weather = &rag_svr.WeatherInfo{
		Location:    weather.Location,
		Weather:     weather.Weather,
		Temperature: float32(weather.Temperature),
		Humidity:    float32(weather.Humidity),
		WindSpeed:   float32(weather.WindSpeed),
		WindDir:     weather.WindDir,
		UpdateTime:  weather.UpdateTime.Format("2006-01-02 15:04:05"),
	}
	logger.Infof("获取天气信息成功: %v", resp.Weather)

	return resp, nil
}

// GetHourlyWeather implements the RagServiceImpl interface.
func (s *RagServiceImpl) GetHourlyWeather(ctx context.Context, req *rag_svr.GetHourlyWeatherReq) (resp *rag_svr.GetHourlyWeatherRsp, err error) {
	resp = &rag_svr.GetHourlyWeatherRsp{
		Code: 0,
		Msg:  "success",
	}

	// 调用天气服务获取24小时预报
	hourlyWeather, err := ai.GetHourlyWeather(ctx, req.Location)
	if err != nil {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("获取24小时天气预报失败: %v", err)
		return resp, nil
	}

	// 构建响应
	resp.Location = hourlyWeather.Location
	resp.Hourly = make([]*rag_svr.HourlyWeather, len(hourlyWeather.Hourly))
	for i, hour := range hourlyWeather.Hourly {
		resp.Hourly[i] = &rag_svr.HourlyWeather{
			Time:        hour.Time,
			Weather:     hour.Weather,
			Temperature: float32(hour.Temperature),
			Humidity:    float32(hour.Humidity),
			WindSpeed:   float32(hour.WindSpeed),
			WindDir:     hour.WindDir,
		}
	}

	return resp, nil
}

// GetDailyWeather implements the RagServiceImpl interface.
func (s *RagServiceImpl) GetDailyWeather(ctx context.Context, req *rag_svr.GetDailyWeatherReq) (resp *rag_svr.GetDailyWeatherRsp, err error) {
	resp = &rag_svr.GetDailyWeatherRsp{
		Code: 0,
		Msg:  "success",
	}

	// 调用天气服务获取15天预报
	dailyWeather, err := ai.GetDailyWeather(ctx, req.Location)
	if err != nil {
		resp.Code = 1
		resp.Msg = fmt.Sprintf("获取15天天气预报失败: %v", err)
		return resp, nil
	}

	// 构建响应
	resp.Location = dailyWeather.Location
	resp.Daily = make([]*rag_svr.DailyWeather, len(dailyWeather.Daily))
	for i, day := range dailyWeather.Daily {
		resp.Daily[i] = &rag_svr.DailyWeather{
			Date:      day.Date,
			TextDay:   day.TextDay,
			TextNight: day.TextNight,
			HighTemp:  float32(day.HighTemp),
			LowTemp:   float32(day.LowTemp),
			Rainfall:  float32(day.Rainfall),
			Precip:    float32(day.Precip),
			WindDir:   day.WindDir,
			WindSpeed: float32(day.WindSpeed),
			WindScale: day.WindScale,
			Humidity:  float32(day.Humidity),
		}
	}

	return resp, nil
}
