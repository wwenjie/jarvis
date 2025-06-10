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

// ChatResponse 对话响应
type ChatResponse struct {
	Output struct {
		Text string `json:"text"`
	} `json:"output"`
}

// GetSummary 获取文本摘要
func GetSummary(text string) (string, error) {
	// 从环境变量获取配置
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("环境变量 DASHSCOPE_API_KEY 未设置")
	}
	baseURL := os.Getenv("DASHSCOPE_BASE_URL")
	if baseURL == "" {
		return "", fmt.Errorf("环境变量 DASHSCOPE_BASE_URL 未设置")
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
						"content": "你是一个专业的文本摘要助手，请对输入的文本进行简洁的摘要。",
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
			return "", fmt.Errorf("序列化请求失败: %v", err)
		}

		// 创建请求
		req, err := http.NewRequest("POST", baseURL+"/services/aigc/text-generation/generation", bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("创建请求失败: %v", err)
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
		var chatResp ChatResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %v", err)
			continue
		}

		if chatResp.Output.Text == "" {
			lastErr = fmt.Errorf("未获取到摘要")
			continue
		}

		return chatResp.Output.Text, nil
	}

	return "", fmt.Errorf("重试%d次后仍然失败: %v", maxRetries, lastErr)
}
