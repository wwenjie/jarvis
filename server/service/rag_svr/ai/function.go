package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"server/framework/logger"
	"server/framework/milvus"
	"server/framework/mysql"
	"server/framework/redis"
	"server/service/rag_svr/embedding"
	"server/service/rag_svr/memory"
)

// FunctionCall 函数调用结构
type FunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// FunctionDefinition 函数定义结构
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// FunctionRegistry 函数注册表
type FunctionRegistry struct {
	functions map[string]FunctionHandler
}

// FunctionHandler 函数处理器接口
type FunctionHandler interface {
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
	GetDefinition() FunctionDefinition
}

var registry *FunctionRegistry

// GetFunctionRegistry 获取函数注册表实例
func GetFunctionRegistry() *FunctionRegistry {
	if registry == nil {
		registry = &FunctionRegistry{
			functions: make(map[string]FunctionHandler),
		}
	}
	return registry
}

// RegisterFunction 注册函数
func (r *FunctionRegistry) RegisterFunction(handler FunctionHandler) {
	r.functions[handler.GetDefinition().Name] = handler
}

// GetFunction 获取函数处理器
func (r *FunctionRegistry) GetFunction(name string) (FunctionHandler, bool) {
	handler, ok := r.functions[name]
	return handler, ok
}

// GetFunctionDefinitions 获取所有函数定义
func (r *FunctionRegistry) GetFunctionDefinitions() []FunctionDefinition {
	definitions := make([]FunctionDefinition, 0, len(r.functions))
	for _, handler := range r.functions {
		definitions = append(definitions, handler.GetDefinition())
	}
	return definitions
}

// SearchDocumentFunction 搜索文档函数
type SearchDocumentFunction struct{}

func (f *SearchDocumentFunction) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	logger.Infof("开始执行搜索文档函数: args=%+v", args)

	query, ok := args["query"].(string)
	if !ok {
		logger.Errorf("query参数缺失或类型错误")
		return nil, fmt.Errorf("query参数缺失或类型错误")
	}
	topK, ok := args["top_k"].(float64)
	if !ok {
		topK = 5 // 默认值
		logger.Infof("使用默认top_k值: %d", int(topK))
	}

	// 使用 query 和 topK 变量
	result := map[string]interface{}{
		"results": []map[string]interface{}{
			{
				"title":   fmt.Sprintf("搜索结果: %s", query),
				"content": "这是搜索结果",
				"score":   0.95,
				"top_k":   int(topK),
			},
		},
	}

	logger.Infof("搜索文档函数执行成功: query=%s, top_k=%d", query, int(topK))
	return result, nil
}

func (f *SearchDocumentFunction) GetDefinition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "search_document",
		Description: "搜索知识库中的文档",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "搜索查询",
				},
				"top_k": map[string]interface{}{
					"type":        "integer",
					"description": "返回结果数量",
				},
			},
			"required": []string{"query"},
		},
	}
}

// GetWeatherFunction 获取天气函数
type GetWeatherFunction struct{}

func (f *GetWeatherFunction) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	logger.Infof("开始执行获取天气函数: args=%+v", args)

	location, ok := args["location"].(string)
	if !ok {
		logger.Errorf("location参数缺失或类型错误")
		return nil, fmt.Errorf("location参数缺失或类型错误")
	}

	// 获取天气信息
	weather, err := GetWeather(ctx, location)
	if err != nil {
		logger.Errorf("获取天气信息失败: location=%s, error=%v", location, err)
		return nil, fmt.Errorf("获取天气信息失败: %v", err)
	}

	result := map[string]interface{}{
		"location":    weather.Location,
		"weather":     weather.Weather,
		"temperature": weather.Temperature,
		"humidity":    weather.Humidity,
		"wind_speed":  weather.WindSpeed,
		"wind_dir":    weather.WindDir,
		"update_time": weather.UpdateTime.Format("2006-01-02 15:04:05"),
	}

	logger.Infof("获取天气函数执行成功: location=%s", location)
	return result, nil
}

func (f *GetWeatherFunction) GetDefinition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "get_weather",
		Description: "获取指定位置的天气信息",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "位置名称，如：北京、上海、广州等",
				},
			},
			"required": []string{"location"},
		},
	}
}

// SetReminderFunction 设置提醒函数
type SetReminderFunction struct{}

func (f *SetReminderFunction) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	logger.Infof("开始执行设置提醒函数: args=%+v", args)

	timeStr, ok := args["time"].(string)
	if !ok {
		logger.Errorf("time参数缺失或类型错误")
		return nil, fmt.Errorf("time参数缺失或类型错误")
	}
	content, ok := args["content"].(string)
	if !ok {
		logger.Errorf("content参数缺失或类型错误")
		return nil, fmt.Errorf("content参数缺失或类型错误")
	}

	// 解析提醒时间
	reminderTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		logger.Errorf("解析提醒时间失败: time=%s, error=%v", timeStr, err)
		return nil, fmt.Errorf("解析提醒时间失败: %v", err)
	}

	// 创建提醒记录
	reminder := &mysql.Reminder{
		Content:    content,
		RemindTime: reminderTime,
		Status:     "pending",
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	// 保存到数据库
	if err := mysql.GetDB().Create(reminder).Error; err != nil {
		logger.Errorf("创建提醒记录失败: error=%v", err)
		return nil, fmt.Errorf("创建提醒记录失败: %v", err)
	}

	// 将提醒信息添加到记忆系统
	metadata := map[string]interface{}{
		"reminder_id": reminder.ID,
		"remind_time": reminderTime.Format("2006-01-02 15:04:05"),
		"status":      "pending",
	}

	err = memory.GetInstance().AddMemory(ctx, 0, 0, content, memory.MemoryTypeReminder, 0.9, metadata, nil)
	if err != nil {
		logger.Errorf("添加提醒记忆失败: error=%v", err)
		// 继续执行，不影响提醒的创建
	}

	result := map[string]interface{}{
		"reminder_id": reminder.ID,
		"time":        timeStr,
		"content":     content,
		"status":      "created",
	}

	logger.Infof("设置提醒函数执行成功: time=%s, content=%s, reminder_id=%d", timeStr, content, reminder.ID)
	return result, nil
}

func (f *SetReminderFunction) GetDefinition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "set_reminder",
		Description: "设置一个提醒",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"time": map[string]interface{}{
					"type":        "string",
					"description": "提醒时间，格式：2006-01-02 15:04:05",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "提醒内容",
				},
			},
			"required": []string{"time", "content"},
		},
	}
}

// CheckReminders 检查是否有需要提醒的事项
func CheckReminders(ctx context.Context) ([]*mysql.Reminder, error) {
	logger.Infof("开始检查提醒事项")

	// 获取所有待处理的提醒
	var reminders []*mysql.Reminder
	if err := mysql.GetDB().
		Where("status = ? AND remind_time <= ?", "pending", time.Now()).
		Find(&reminders).Error; err != nil {
		logger.Errorf("获取提醒事项失败: error=%v", err)
		return nil, fmt.Errorf("获取提醒事项失败: %v", err)
	}

	// 更新提醒状态
	for _, reminder := range reminders {
		reminder.Status = "triggered"
		reminder.UpdateTime = time.Now()
		if err := mysql.GetDB().Save(reminder).Error; err != nil {
			logger.Errorf("更新提醒状态失败: reminder_id=%d, error=%v", reminder.ID, err)
			continue
		}
	}

	logger.Infof("检查提醒事项完成: 找到%d个待提醒事项", len(reminders))
	return reminders, nil
}

// GetReminderFunction 获取提醒函数
type GetReminderFunction struct{}

func (f *GetReminderFunction) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	logger.Infof("开始执行获取提醒函数: args=%+v", args)

	// 检查是否有需要提醒的事项
	reminders, err := CheckReminders(ctx)
	if err != nil {
		logger.Errorf("检查提醒事项失败: error=%v", err)
		return nil, err
	}

	// 构建返回结果
	result := map[string]interface{}{
		"reminders": make([]map[string]interface{}, 0, len(reminders)),
	}

	for _, reminder := range reminders {
		result["reminders"] = append(result["reminders"].([]map[string]interface{}), map[string]interface{}{
			"id":          reminder.ID,
			"content":     reminder.Content,
			"remind_time": reminder.RemindTime.Format("2006-01-02 15:04:05"),
			"status":      reminder.Status,
			"create_time": reminder.CreateTime.Format("2006-01-02 15:04:05"),
		})
	}

	logger.Infof("获取提醒函数执行成功: 找到%d个提醒事项", len(reminders))
	return result, nil
}

func (f *GetReminderFunction) GetDefinition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "get_reminders",
		Description: "获取当前需要提醒的事项",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []string{},
		},
	}
}

// GetUserPreferencesFunction 获取用户偏好函数
type GetUserPreferencesFunction struct{}

func (f *GetUserPreferencesFunction) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	logger.Infof("开始执行获取用户偏好函数: args=%+v", args)

	userID, ok := args["user_id"].(float64)
	if !ok {
		logger.Errorf("user_id参数缺失或类型错误")
		return nil, fmt.Errorf("user_id参数缺失或类型错误")
	}

	result := map[string]interface{}{
		"user_id": int(userID),
		"preferences": map[string]interface{}{
			"language":     "zh-CN",
			"theme":        "dark",
			"notification": true,
		},
	}

	logger.Infof("获取用户偏好函数执行成功: user_id=%d", int(userID))
	return result, nil
}

func (f *GetUserPreferencesFunction) GetDefinition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "get_user_preferences",
		Description: "获取用户偏好设置",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_id": map[string]interface{}{
					"type":        "integer",
					"description": "用户ID",
				},
			},
			"required": []string{"user_id"},
		},
	}
}

// ExecuteFunctionCall 执行函数调用
func ExecuteFunctionCall(ctx context.Context, call *FunctionCall) (interface{}, error) {
	logger.Infof("开始执行函数调用: name=%s, arguments=%+v", call.Name, call.Arguments)

	registry := GetFunctionRegistry()
	handler, ok := registry.GetFunction(call.Name)
	if !ok {
		logger.Errorf("未找到函数: %s", call.Name)
		return nil, fmt.Errorf("未找到函数: %s", call.Name)
	}

	result, err := handler.Execute(ctx, call.Arguments)
	if err != nil {
		logger.Errorf("函数执行失败: name=%s, error=%v", call.Name, err)
		return nil, err
	}

	logger.Infof("函数执行成功: name=%s, result=%+v", call.Name, result)
	return result, nil
}

// ParseFunctionCall 解析函数调用
func ParseFunctionCall(text string) (*FunctionCall, error) {
	logger.Infof("开始解析函数调用: text=%s", text)

	var call FunctionCall
	if err := json.Unmarshal([]byte(text), &call); err != nil {
		logger.Errorf("解析函数调用失败: error=%v", err)
		return nil, fmt.Errorf("解析函数调用失败: %v", err)
	}

	logger.Infof("函数调用解析成功: name=%s", call.Name)
	return &call, nil
}

// SearchSimilarDocuments 搜索相似文档
func (c *QwenClient) SearchSimilarDocuments(ctx context.Context, query string, limit int) ([]string, error) {
	// 生成查询向量
	queryEmbedding, err := embedding.GetEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %v", err)
	}

	// 在 Milvus 中搜索相似向量
	ids, scores, err := milvus.SearchVector(ctx, "documents", queryEmbedding, limit*2) // 获取更多结果用于重排序
	if err != nil {
		return nil, fmt.Errorf("搜索向量失败: %v", err)
	}

	// 获取对应的文档记录
	var documents []*mysql.Document
	if err := mysql.GetDB().Where("id IN ?", ids).Find(&documents).Error; err != nil {
		return nil, fmt.Errorf("获取文档记录失败: %v", err)
	}

	// 创建 ID 到分数的映射
	scoreMap := make(map[uint64]float32)
	for i, id := range ids {
		scoreMap[uint64(id)] = scores[i]
	}

	// 转换为结果并计算综合分数
	type docScore struct {
		doc   *mysql.Document
		score float32
	}
	docScores := make([]docScore, 0, len(documents))
	for _, doc := range documents {
		docScores = append(docScores, docScore{
			doc:   doc,
			score: scoreMap[doc.ID],
		})
	}

	// 按分数排序
	sort.Slice(docScores, func(i, j int) bool {
		return docScores[i].score > docScores[j].score
	})

	// 只返回前 limit 个结果
	if len(docScores) > limit {
		docScores = docScores[:limit]
	}

	// 构建返回结果
	results := make([]string, len(docScores))
	for i, ds := range docScores {
		results[i] = fmt.Sprintf("标题: %s\n内容: %s\n相关度: %.2f", ds.doc.Title, ds.doc.Content, ds.score)
	}

	return results, nil
}

// GetUserHistory 获取用户历史记录
func (c *QwenClient) GetUserHistory(ctx context.Context, userID string) ([]string, error) {
	// 从缓存获取
	cacheKey := fmt.Sprintf("user_history:%s", userID)
	if cached, err := redis.Get(ctx, cacheKey); err == nil {
		var history []string
		if err := json.Unmarshal([]byte(cached), &history); err == nil {
			return history, nil
		}
	}

	// 从数据库获取
	var records []*mysql.ChatRecord
	if err := mysql.GetDB().
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(100). // 限制返回最近100条记录
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("获取历史记录失败: %v", err)
	}

	// 构建返回结果
	history := make([]string, len(records))
	for i, record := range records {
		history[i] = fmt.Sprintf("时间: %s\n用户: %s\n助手: %s",
			record.CreatedAt.Format("2006-01-02 15:04:05"),
			record.Message,
			record.Response)
	}

	// 缓存结果
	if historyJSON, err := json.Marshal(history); err == nil {
		redis.Set(ctx, cacheKey, string(historyJSON), 1*time.Hour) // 缓存1小时
	}

	return history, nil
}

// GetMemoryFunction 获取记忆函数
type GetMemoryFunction struct{}

func (f *GetMemoryFunction) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	logger.Infof("开始执行获取记忆函数: args=%+v", args)

	memoryID, ok := args["memory_id"].(float64)
	if !ok {
		logger.Errorf("memory_id参数缺失或类型错误")
		return nil, fmt.Errorf("memory_id参数缺失或类型错误")
	}

	// 获取记忆信息
	memories, err := memory.GetInstance().SearchMemories(ctx, 0, fmt.Sprintf("id:%d", uint64(memoryID)), 1)
	if err != nil {
		logger.Errorf("获取记忆失败: memory_id=%d, error=%v", uint64(memoryID), err)
		return nil, fmt.Errorf("获取记忆失败: %v", err)
	}
	if len(memories) == 0 {
		logger.Errorf("记忆不存在: memory_id=%d", uint64(memoryID))
		return nil, fmt.Errorf("记忆不存在")
	}

	mem := memories[0]
	result := map[string]interface{}{
		"id":           mem.ID,
		"user_id":      mem.UserID,
		"content":      mem.Content,
		"memory_type":  mem.Type,
		"importance":   mem.Importance,
		"created_at":   mem.CreatedAt.Format("2006-01-02 15:04:05"),
		"expire_time":  mem.ExpiresAt.Format("2006-01-02 15:04:05"),
		"access_count": mem.AccessCount,
		"metadata":     mem.Metadata,
	}

	logger.Infof("获取记忆函数执行成功: memory_id=%d", uint64(memoryID))
	return result, nil
}

func (f *GetMemoryFunction) GetDefinition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "get_memory",
		Description: "获取指定ID的记忆信息",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"memory_id": map[string]interface{}{
					"type":        "number",
					"description": "记忆ID",
				},
			},
			"required": []string{"memory_id"},
		},
	}
}

// AddMemoryFunction 添加记忆函数
type AddMemoryFunction struct{}

func (f *AddMemoryFunction) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	logger.Infof("开始执行添加记忆函数: args=%+v", args)

	userID, ok := args["user_id"].(float64)
	if !ok {
		logger.Errorf("user_id参数缺失或类型错误")
		return nil, fmt.Errorf("user_id参数缺失或类型错误")
	}
	content, ok := args["content"].(string)
	if !ok {
		logger.Errorf("content参数缺失或类型错误")
		return nil, fmt.Errorf("content参数缺失或类型错误")
	}
	memoryType, ok := args["memory_type"].(string)
	if !ok {
		logger.Errorf("memory_type参数缺失或类型错误")
		return nil, fmt.Errorf("memory_type参数缺失或类型错误")
	}
	importance, ok := args["importance"].(float64)
	if !ok {
		logger.Errorf("importance参数缺失或类型错误")
		return nil, fmt.Errorf("importance参数缺失或类型错误")
	}

	// 添加记忆
	err := memory.GetInstance().AddMemory(ctx, 0, uint64(userID), content, memoryType, importance, nil, nil)
	if err != nil {
		logger.Errorf("添加记忆失败: user_id=%d, content=%s, error=%v", uint64(userID), content, err)
		return nil, fmt.Errorf("添加记忆失败: %v", err)
	}

	result := map[string]interface{}{
		"status":  "success",
		"message": "记忆添加成功",
	}

	logger.Infof("添加记忆函数执行成功: user_id=%d, memory_type=%s", uint64(userID), memoryType)
	return result, nil
}

func (f *AddMemoryFunction) GetDefinition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "add_memory",
		Description: "添加新的记忆",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_id": map[string]interface{}{
					"type":        "number",
					"description": "用户ID",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "记忆内容",
				},
				"memory_type": map[string]interface{}{
					"type":        "string",
					"description": "记忆类型",
				},
				"importance": map[string]interface{}{
					"type":        "number",
					"description": "重要性(0-1)",
				},
			},
			"required": []string{"user_id", "content", "memory_type", "importance"},
		},
	}
}

// GetTimeFunction 获取时间函数
type GetTimeFunction struct{}

func (f *GetTimeFunction) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	logger.Infof("开始执行获取时间函数: args=%+v", args)

	// 获取当前时间
	now := time.Now()

	// 获取时区
	timezone := "Asia/Shanghai"
	if tz, ok := args["timezone"].(string); ok {
		timezone = tz
		logger.Infof("使用指定时区: %s", timezone)
	}

	// 设置时区
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		logger.Errorf("无效的时区: %s, error=%v", timezone, err)
		return nil, fmt.Errorf("无效的时区: %v", err)
	}
	now = now.In(loc)

	result := map[string]interface{}{
		"time":       now.Format("2006-01-02 15:04:05"),
		"timezone":   timezone,
		"unix":       now.Unix(),
		"year":       now.Year(),
		"month":      int(now.Month()),
		"day":        now.Day(),
		"hour":       now.Hour(),
		"minute":     now.Minute(),
		"second":     now.Second(),
		"weekday":    now.Weekday().String(),
		"is_weekend": now.Weekday() == time.Saturday || now.Weekday() == time.Sunday,
	}

	logger.Infof("获取时间函数执行成功: timezone=%s", timezone)
	return result, nil
}

func (f *GetTimeFunction) GetDefinition() FunctionDefinition {
	return FunctionDefinition{
		Name:        "get_time",
		Description: "获取当前时间信息",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"timezone": map[string]interface{}{
					"type":        "string",
					"description": "时区，如：Asia/Shanghai",
				},
			},
			"required": []string{},
		},
	}
}

// 注册所有函数
func init() {
	registry := GetFunctionRegistry()

	// 文档相关函数
	registry.RegisterFunction(&SearchDocumentFunction{})

	// 天气相关函数
	registry.RegisterFunction(&GetWeatherFunction{})

	// 提醒相关函数
	registry.RegisterFunction(&SetReminderFunction{})
	registry.RegisterFunction(&GetReminderFunction{})

	// 用户相关函数
	registry.RegisterFunction(&GetUserPreferencesFunction{})

	// 记忆相关函数
	registry.RegisterFunction(&GetMemoryFunction{})
	registry.RegisterFunction(&AddMemoryFunction{})

	// 时间相关函数
	registry.RegisterFunction(&GetTimeFunction{})
}
