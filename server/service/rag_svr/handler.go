package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"server/framework/id_generator"
	"server/framework/logger"
	"server/framework/mongodb"
	"server/framework/mysql"
	"server/service/rag_svr/ai"
	rag_svr "server/service/rag_svr/kitex_gen/rag_svr"
	"server/service/rag_svr/memory"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	// 创建用户
	user := &mysql.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // 注意：实际应用中应该对密码进行加密
	}

	if err := mysql.GetDB().Create(user).Error; err != nil {
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
			CreateTime: user.CreatedAt.Format(time.RFC3339),
			UpdateTime: user.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

// CreateSession implements the RagServiceImpl interface.
func (s *RagServiceImpl) CreateSession(ctx context.Context, req *rag_svr.CreateSessionReq) (resp *rag_svr.CreateSessionRsp, err error) {
	logger.Infof("创建会话请求: user_id=%d", req.UserId)

	// 创建会话
	session := &mysql.Session{
		UserID: req.UserId,
		Status: "active",
	}

	if err := mysql.GetDB().Create(session).Error; err != nil {
		logger.Errorf("创建会话失败: %v", err)
		return &rag_svr.CreateSessionRsp{
			Code: 1,
			Msg:  fmt.Sprintf("创建会话失败: %v", err),
		}, nil
	}

	logger.Infof("会话创建成功: session_id=%d", session.ID)

	// 返回会话信息
	return &rag_svr.CreateSessionRsp{
		Code: 0,
		Msg:  "success",
		SessionInfo: &rag_svr.SessionInfo{
			SessionId:  session.ID,
			UserId:     session.UserID,
			CreateTime: session.CreatedAt.Format(time.RFC3339),
			Status:     session.Status,
		},
	}, nil
}

// EndSession implements the RagServiceImpl interface.
func (s *RagServiceImpl) EndSession(ctx context.Context, req *rag_svr.EndSessionReq) (resp *rag_svr.EndSessionRsp, err error) {
	logger.Infof("结束会话请求: session_id=%d", req.SessionId)

	// 更新会话状态
	if err := mysql.GetDB().Model(&mysql.Session{}).
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

// SendMessage implements the RagServiceImpl interface.
func (s *RagServiceImpl) SendMessage(ctx context.Context, req *rag_svr.SendMessageReq) (resp *rag_svr.SendMessageRsp, err error) {
	// 获取会话信息
	var session mysql.Session
	if err := mysql.GetDB().First(&session, req.SessionId).Error; err != nil {
		return &rag_svr.SendMessageRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取会话信息失败: %v", err),
		}, nil
	}

	// 获取用户状态和系统状态
	userState, err := session.GetUserState()
	if err != nil {
		return &rag_svr.SendMessageRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取用户状态失败: %v", err),
		}, nil
	}

	systemState, err := session.GetSystemState()
	if err != nil {
		return &rag_svr.SendMessageRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取系统状态失败: %v", err),
		}, nil
	}

	// 获取最近的对话记录作为上下文
	var recentRecords []mysql.ChatRecord
	if err := mysql.GetDB().Where("session_id = ?", req.SessionId).
		Order("created_at DESC").
		Limit(10).
		Find(&recentRecords).Error; err != nil {
		return &rag_svr.SendMessageRsp{
			Code: 1,
			Msg:  fmt.Sprintf("获取历史记录失败: %v", err),
		}, nil
	}

	// 搜索相关记忆
	memoryManager := memory.GetInstance()
	relatedMemories, err := memoryManager.SearchMemories(ctx, req.UserId, req.Message, 5)
	if err != nil {
		logger.Errorf("搜索相关记忆失败: %v", err)
	}

	// 分析用户意图和情感
	intent, err := ai.AnalyzeIntent(req.Message)
	if err != nil {
		logger.Errorf("分析用户意图失败: %v", err)
	} else {
		userState["last_intent"] = intent.Intent
		userState["intent_confidence"] = fmt.Sprintf("%.2f", intent.Confidence)
	}

	sentiment, err := ai.AnalyzeSentiment(req.Message)
	if err != nil {
		logger.Errorf("分析用户情感失败: %v", err)
	} else {
		userState["last_sentiment"] = sentiment.Sentiment
		userState["sentiment_confidence"] = fmt.Sprintf("%.2f", sentiment.Confidence)
	}

	// 根据意图和情感调整系统状态
	if intent != nil {
		switch intent.Intent {
		case ai.IntentTypeQuestion:
			systemState["response_mode"] = "informative"
		case ai.IntentTypeCommand:
			systemState["response_mode"] = "action"
		case ai.IntentTypeChat:
			systemState["response_mode"] = "conversational"
		case ai.IntentTypeTask:
			systemState["response_mode"] = "task-oriented"
		}
	}

	if sentiment != nil {
		switch sentiment.Sentiment {
		case ai.SentimentPositive:
			systemState["tone"] = "friendly"
		case ai.SentimentNeutral:
			systemState["tone"] = "professional"
		case ai.SentimentNegative:
			systemState["tone"] = "empathetic"
		}
	}

	// 更新会话状态
	if err := session.SetUserState(userState); err != nil {
		logger.Errorf("更新用户状态失败: %v", err)
	}
	if err := session.SetSystemState(systemState); err != nil {
		logger.Errorf("更新系统状态失败: %v", err)
	}
	if err := mysql.GetDB().Save(&session).Error; err != nil {
		logger.Errorf("保存会话状态失败: %v", err)
	}

	// 构建上下文
	context := map[string]interface{}{
		"session_summary":  session.Summary,
		"user_state":       userState,
		"system_state":     systemState,
		"recent_messages":  recentRecords,
		"related_memories": relatedMemories,
		"intent":           intent,
		"sentiment":        sentiment,
	}

	// 将 context 转换为 JSON 字符串并记录到日志
	contextStr, err := json.Marshal(context)
	if err != nil {
		logger.Errorf("序列化上下文失败: %v", err)
	} else {
		logger.Infof("当前上下文: %s", string(contextStr))
	}

	// 创建对话记录
	chatRecord := &mysql.ChatRecord{
		SessionID:   req.SessionId,
		UserID:      req.UserId,
		Message:     req.Message,
		MessageType: req.MessageType,
		Status:      "pending",
	}

	if err := mysql.GetDB().Create(chatRecord).Error; err != nil {
		return &rag_svr.SendMessageRsp{
			Code: 1,
			Msg:  fmt.Sprintf("创建对话记录失败: %v", err),
		}, nil
	}

	// 提取并存储记忆
	if intent != nil {
		// 根据意图类型设置记忆类型和重要性
		var memoryType string
		var importance float64
		switch intent.Intent {
		case ai.IntentTypeQuestion:
			memoryType = memory.MemoryTypeFact
			importance = 0.7
		case ai.IntentTypeCommand:
			memoryType = memory.MemoryTypeContext
			importance = 0.8
		case ai.IntentTypeChat:
			memoryType = memory.MemoryTypeContext
			importance = 0.5
		case ai.IntentTypeTask:
			memoryType = memory.MemoryTypeContext
			importance = 0.9
		}

		// 设置记忆元数据
		metadata := map[string]interface{}{
			"intent":               intent.Intent,
			"intent_confidence":    intent.Confidence,
			"entities":             intent.Entities,
			"sentiment":            sentiment.Sentiment,
			"sentiment_confidence": sentiment.Confidence,
			"emotions":             sentiment.Emotions,
		}

		// 添加记忆
		if err := memoryManager.AddMemory(ctx, req.SessionId, req.UserId, req.Message, memoryType, importance, metadata, nil); err != nil {
			logger.Errorf("添加记忆失败: %v", err)
		}
	}

	// 构建响应
	return &rag_svr.SendMessageRsp{
		Code: 0,
		Msg:  "success",
		SessionInfo: &rag_svr.SessionInfo{
			SessionId:  session.ID,
			UserId:     session.UserID,
			CreateTime: session.CreatedAt.Format(time.RFC3339),
			Status:     session.Status,
			ChatRecords: []*rag_svr.ChatRecord{
				{
					ChatId:      chatRecord.ID,
					Message:     chatRecord.Message,
					MessageType: chatRecord.MessageType,
					Status:      chatRecord.Status,
					CreateTime:  chatRecord.CreatedAt.Format(time.RFC3339),
				},
			},
		},
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

// AddDocument implements the RagServiceImpl interface.
func (s *RagServiceImpl) AddDocument(ctx context.Context, req *rag_svr.AddDocumentReq) (resp *rag_svr.AddDocumentRsp, err error) {
	logger.Infof("添加文档请求: user_id=%d, title=%s", req.UserId, req.Title)

	// 获取新的文档ID
	docID := id_generator.GetInstance().GetDocumentID()
	if docID == 0 {
		logger.Errorf("获取文档ID失败")
		return &rag_svr.AddDocumentRsp{
			Code: 1,
			Msg:  "获取文档ID失败",
		}, nil
	}

	// 创建文档
	doc := &mysql.Document{
		ID:       docID,
		UserID:   req.UserId,
		Title:    req.Title,
		Content:  req.Content,
		Metadata: req.Metadata,
	}

	if err := mysql.GetDB().Create(doc).Error; err != nil {
		logger.Errorf("创建文档失败: %v", err)
		return &rag_svr.AddDocumentRsp{
			Code: 1,
			Msg:  fmt.Sprintf("创建文档失败: %v", err),
		}, nil
	}

	// 将文档存储到MongoDB
	mongoDoc := bson.M{
		"doc_id":      doc.ID,
		"user_id":     doc.UserID,
		"title":       doc.Title,
		"content":     doc.Content,
		"metadata":    doc.Metadata,
		"create_time": doc.CreatedAt,
		"update_time": doc.UpdatedAt,
	}

	if _, err := mongodb.InsertOne(ctx, "documents", mongoDoc); err != nil {
		logger.Errorf("存储文档到MongoDB失败: %v", err)
		return &rag_svr.AddDocumentRsp{
			Code: 1,
			Msg:  fmt.Sprintf("存储文档到MongoDB失败: %v", err),
		}, nil
	}

	logger.Infof("文档添加成功: doc_id=%d", doc.ID)

	return &rag_svr.AddDocumentRsp{
		Code:  0,
		Msg:   "success",
		DocId: doc.ID,
	}, nil
}

// SearchDocument implements the RagServiceImpl interface.
func (s *RagServiceImpl) SearchDocument(ctx context.Context, req *rag_svr.SearchDocumentReq) (resp *rag_svr.SearchDocumentRsp, err error) {
	logger.Infof("搜索文档请求: user_id=%d, query=%s, top_k=%d", req.UserId, req.Query, req.TopK)

	// 构建搜索条件
	filter := bson.M{
		"user_id": req.UserId,
		"$text": bson.M{
			"$search": req.Query,
		},
	}

	// 设置搜索选项
	opts := options.Find().
		SetLimit(int64(req.TopK)).
		SetSort(bson.M{"score": bson.M{"$meta": "textScore"}})

	// 执行搜索
	cursor, err := mongodb.Find(ctx, "documents", filter, opts)
	if err != nil {
		logger.Errorf("搜索文档失败: %v", err)
		return &rag_svr.SearchDocumentRsp{
			Code: 1,
			Msg:  fmt.Sprintf("搜索文档失败: %v", err),
		}, nil
	}
	defer cursor.Close(ctx)

	// 解析结果
	var documents []*rag_svr.Document
	var scores []float32

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			logger.Errorf("解析文档失败: %v", err)
			continue
		}

		// 获取文档详情
		var mysqlDoc mysql.Document
		if err := mysql.GetDB().First(&mysqlDoc, doc["doc_id"]).Error; err != nil {
			logger.Errorf("获取文档详情失败: %v", err)
			continue
		}

		documents = append(documents, &rag_svr.Document{
			DocId:      mysqlDoc.ID,
			UserId:     mysqlDoc.UserID,
			Title:      mysqlDoc.Title,
			Content:    mysqlDoc.Content,
			Metadata:   mysqlDoc.Metadata,
			CreateTime: mysqlDoc.CreatedAt.Format(time.RFC3339),
			UpdateTime: mysqlDoc.UpdatedAt.Format(time.RFC3339),
		})

		// 获取相关性分数
		if score, ok := doc["score"].(float64); ok {
			scores = append(scores, float32(score))
		}
	}

	logger.Infof("文档搜索完成: 找到%d个结果", len(documents))

	return &rag_svr.SearchDocumentRsp{
		Code:      0,
		Msg:       "success",
		Documents: documents,
		Scores:    scores,
	}, nil
}

// GetSessionList implements the RagServiceImpl interface.
func (s *RagServiceImpl) GetSessionList(ctx context.Context, req *rag_svr.GetSessionListReq) (resp *rag_svr.GetSessionListRsp, err error) {
	logger.Infof("获取会话列表请求: user_id=%d, status=%s, page=%d, page_size=%d",
		req.UserId, req.Status, req.Page, req.PageSize)

	// 获取用户的会话列表
	var sessions []mysql.Session
	query := mysql.GetDB().Where("user_id = ?", req.UserId)

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
		sessionInfos = append(sessionInfos, &rag_svr.SessionInfo{
			SessionId:  session.ID,
			UserId:     session.UserID,
			Title:      session.Title,
			Summary:    session.Summary,
			Status:     session.Status,
			CreateTime: session.CreatedAt.Format(time.RFC3339),
			UpdateTime: session.UpdatedAt.Format(time.RFC3339),
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
	result := mysql.GetDB().Model(&mysql.Session{}).
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

func (s *RagServiceImpl) SendMessageStream(req *rag_svr.SendMessageReq, stream rag_svr.RagService_SendMessageStreamServer) (err error) {
	// 获取会话信息
	var session mysql.Session
	if err := mysql.GetDB().First(&session, req.SessionId).Error; err != nil {
		return fmt.Errorf("获取会话信息失败: %v", err)
	}

	// 获取用户状态和系统状态
	userState, err := session.GetUserState()
	if err != nil {
		return fmt.Errorf("获取用户状态失败: %v", err)
	}

	systemState, err := session.GetSystemState()
	if err != nil {
		return fmt.Errorf("获取系统状态失败: %v", err)
	}

	// 获取最近的对话记录作为上下文
	var recentRecords []mysql.ChatRecord
	if err := mysql.GetDB().Where("session_id = ?", req.SessionId).
		Order("created_at DESC").
		Limit(10).
		Find(&recentRecords).Error; err != nil {
		return fmt.Errorf("获取历史记录失败: %v", err)
	}

	// 创建对话记录
	chatRecord := &mysql.ChatRecord{
		SessionID:   req.SessionId,
		UserID:      req.UserId,
		Message:     req.Message,
		MessageType: req.MessageType,
		Status:      "pending",
	}

	if err := mysql.GetDB().Create(chatRecord).Error; err != nil {
		return fmt.Errorf("创建对话记录失败: %v", err)
	}

	// 构建对话历史
	messages := make([]ai.QwenMessage, 0)
	for _, record := range recentRecords {
		messages = append(messages, ai.QwenMessage{
			Role:    "user",
			Content: record.Message,
		})
		if record.Response != "" {
			messages = append(messages, ai.QwenMessage{
				Role:    "assistant",
				Content: record.Response,
			})
		}
	}
	messages = append(messages, ai.QwenMessage{
		Role:    "user",
		Content: req.Message,
	})

	// 构建上下文
	context := map[string]interface{}{
		"session_summary":  session.Summary,
		"user_state":       userState,
		"system_state":     systemState,
		"recent_messages":  recentRecords,
		"related_memories": nil,
	}

	// 将 context 转换为 JSON 字符串并记录到日志
	contextStr, err := json.Marshal(context)
	if err != nil {
		logger.Errorf("序列化上下文失败: %v", err)
	} else {
		logger.Infof("当前上下文: %s", string(contextStr))
	}

	// 调用 Qwen 客户端进行流式对话
	responseChan, errorChan := s.qwenClient.StreamChat(stream.Context(), messages, string(contextStr))

	// 处理流式响应
	var fullResponse strings.Builder
	for {
		select {
		case resp, ok := <-responseChan:
			if !ok {
				// 响应完成
				chatRecord.Response = fullResponse.String()
				chatRecord.Status = "completed"
				if err := mysql.GetDB().Save(chatRecord).Error; err != nil {
					return fmt.Errorf("更新对话记录失败: %v", err)
				}

				// 发送最终响应
				return stream.Send(&rag_svr.SendMessageRsp{
					Code: 0,
					Msg:  "success",
					ChatRecord: &rag_svr.ChatRecord{
						ChatId:      chatRecord.ID,
						SessionId:   chatRecord.SessionID,
						UserId:      chatRecord.UserID,
						Message:     chatRecord.Message,
						Response:    chatRecord.Response,
						CreateTime:  chatRecord.CreatedAt.Format(time.RFC3339),
						MessageType: chatRecord.MessageType,
						Status:      chatRecord.Status,
					},
					SessionInfo: &rag_svr.SessionInfo{
						SessionId:  session.ID,
						UserId:     session.UserID,
						Title:      session.Title,
						Summary:    session.Summary,
						Status:     session.Status,
						CreateTime: session.CreatedAt.Format(time.RFC3339),
						UpdateTime: session.UpdatedAt.Format(time.RFC3339),
					},
				})
			}

			// 累积完整响应
			fullResponse.WriteString(resp)

			// 发送部分响应
			if err := stream.Send(&rag_svr.SendMessageRsp{
				Code: 0,
				Msg:  "success",
				ChatRecord: &rag_svr.ChatRecord{
					ChatId:      chatRecord.ID,
					SessionId:   chatRecord.SessionID,
					UserId:      chatRecord.UserID,
					Message:     chatRecord.Message,
					Response:    resp,
					CreateTime:  chatRecord.CreatedAt.Format(time.RFC3339),
					MessageType: chatRecord.MessageType,
					Status:      "streaming",
				},
			}); err != nil {
				return fmt.Errorf("发送流式响应失败: %v", err)
			}

		case err := <-errorChan:
			if err != nil {
				chatRecord.Status = "failed"
				chatRecord.Response = fmt.Sprintf("生成回复失败: %v", err)
				if err := mysql.GetDB().Save(chatRecord).Error; err != nil {
					return fmt.Errorf("更新对话记录失败: %v", err)
				}
				return err
			}

		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *RagServiceImpl) Chat(ctx context.Context, req *rag_svr.SendMessageReq) (resp *rag_svr.SendMessageRsp, err error) {
	// 获取上下文信息
	contextJSON, err := s.qwenClient.GetUserHistory(ctx, fmt.Sprintf("%d", req.UserId))
	if err != nil {
		logger.Errorf("获取用户历史记录失败: %v", err)
		// 继续执行，不返回错误
	}

	// 构建消息
	messages := []ai.QwenMessage{
		{
			Role:    "user",
			Content: req.Message,
		},
	}

	// 获取流式响应
	responseChan, errorChan := s.qwenClient.StreamChat(ctx, messages, strings.Join(contextJSON, "\n"))

	// 处理响应
	var responseBuilder strings.Builder
	for {
		select {
		case text, ok := <-responseChan:
			if !ok {
				// 通道已关闭，返回结果
				return &rag_svr.SendMessageRsp{
					Code: 0,
					Msg:  "success",
					SessionInfo: &rag_svr.SessionInfo{
						ChatRecords: []*rag_svr.ChatRecord{
							{
								Message: responseBuilder.String(),
							},
						},
					},
				}, nil
			}
			responseBuilder.WriteString(text)
		case err := <-errorChan:
			if err != nil {
				return &rag_svr.SendMessageRsp{
					Code: 1,
					Msg:  fmt.Sprintf("聊天失败: %v", err),
				}, nil
			}
		case <-ctx.Done():
			return &rag_svr.SendMessageRsp{
				Code: 1,
				Msg:  "请求超时",
			}, nil
		}
	}
}
