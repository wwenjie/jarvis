package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// 意图类型
const (
	IntentTypeQuestion = "question" // 问题
	IntentTypeCommand  = "command"  // 命令
	IntentTypeChat     = "chat"     // 闲聊
	IntentTypeTask     = "task"     // 任务
)

// 情感类型
const (
	SentimentPositive = "positive" // 积极
	SentimentNeutral  = "neutral"  // 中性
	SentimentNegative = "negative" // 消极
)

// IntentResponse 意图分析响应
type IntentResponse struct {
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Entities   []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"entities"`
}

// SentimentResponse 情感分析响应
type SentimentResponse struct {
	Sentiment  string             `json:"sentiment"`
	Confidence float64            `json:"confidence"`
	Emotions   map[string]float64 `json:"emotions"`
}

// AnalyzeIntent 分析用户意图
func AnalyzeIntent(text string) (*IntentResponse, error) {
	// 从环境变量获取配置
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 DASHSCOPE_API_KEY 未设置")
	}
	baseURL := os.Getenv("DASHSCOPE_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("环境变量 DASHSCOPE_BASE_URL 未设置")
	}

	// 获取并发令牌
	requestSemaphore <- struct{}{}
	defer func() { <-requestSemaphore }()

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(retryInterval)
		}

		// 构建请求体
		reqBody := map[string]interface{}{
			"input": map[string]interface{}{
				"messages": []map[string]string{
					{
						"role":    "system",
						"content": "你是一个专业的意图分析助手，请分析用户输入的意图。可能的意图类型包括：问题(question)、命令(command)、闲聊(chat)、任务(task)。",
					},
					{
						"role":    "user",
						"content": text,
					},
				},
			},
			"parameters": map[string]interface{}{
				"temperature": 0.7,
				"top_p":       0.8,
			},
		}
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("序列化请求失败: %v", err)
		}

		// 创建请求
		req, err := http.NewRequest("POST", baseURL+"/services/aigc/text-generation/generation", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}

		// 设置请求头
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		// 发送请求
		client := &http.Client{Timeout: requestTimeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("发送请求失败: %v", err)
			continue
		}

		// 读取响应
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("读取响应失败: %v", err)
			continue
		}

		// 检查响应状态码
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
			continue
		}

		// 解析响应
		var chatResp struct {
			Output struct {
				Text string `json:"text"`
			} `json:"output"`
		}
		if err := json.Unmarshal(body, &chatResp); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %v", err)
			continue
		}

		// 解析意图分析结果
		var intentResp IntentResponse
		if err := json.Unmarshal([]byte(chatResp.Output.Text), &intentResp); err != nil {
			lastErr = fmt.Errorf("解析意图分析结果失败: %v", err)
			continue
		}

		return &intentResp, nil
	}

	return nil, fmt.Errorf("重试%d次后仍然失败: %v", maxRetries, lastErr)
}

// AnalyzeSentiment 分析用户情感
func AnalyzeSentiment(text string) (*SentimentResponse, error) {
	// 从环境变量获取配置
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 DASHSCOPE_API_KEY 未设置")
	}
	baseURL := os.Getenv("DASHSCOPE_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("环境变量 DASHSCOPE_BASE_URL 未设置")
	}

	// 获取并发令牌
	requestSemaphore <- struct{}{}
	defer func() { <-requestSemaphore }()

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(retryInterval)
		}

		// 构建请求体
		reqBody := map[string]interface{}{
			"input": map[string]interface{}{
				"messages": []map[string]string{
					{
						"role":    "system",
						"content": "你是一个专业的情感分析助手，请分析用户输入的情感。可能的情感类型包括：积极(positive)、中性(neutral)、消极(negative)。",
					},
					{
						"role":    "user",
						"content": text,
					},
				},
			},
			"parameters": map[string]interface{}{
				"temperature": 0.7,
				"top_p":       0.8,
			},
		}
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("序列化请求失败: %v", err)
		}

		// 创建请求
		req, err := http.NewRequest("POST", baseURL+"/services/aigc/text-generation/generation", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}

		// 设置请求头
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		// 发送请求
		client := &http.Client{Timeout: requestTimeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("发送请求失败: %v", err)
			continue
		}

		// 读取响应
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("读取响应失败: %v", err)
			continue
		}

		// 检查响应状态码
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
			continue
		}

		// 解析响应
		var chatResp struct {
			Output struct {
				Text string `json:"text"`
			} `json:"output"`
		}
		if err := json.Unmarshal(body, &chatResp); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %v", err)
			continue
		}

		// 解析情感分析结果
		var sentimentResp SentimentResponse
		if err := json.Unmarshal([]byte(chatResp.Output.Text), &sentimentResp); err != nil {
			lastErr = fmt.Errorf("解析情感分析结果失败: %v", err)
			continue
		}

		return &sentimentResp, nil
	}

	return nil, fmt.Errorf("重试%d次后仍然失败: %v", maxRetries, lastErr)
}
