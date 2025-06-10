package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"server/framework/logger"
)

const (
	OpenAIAPIEndpoint = "https://api.openai.com/v1/embeddings"
	MaxRetries        = 3
	RetryDelay        = time.Second
)

// GetEmbedding 获取文本的向量表示
func GetEmbedding(text string) ([]float32, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 OPENAI_API_KEY 未设置")
	}

	// 构建请求
	reqBody, err := json.Marshal(map[string]interface{}{
		"model": "text-embedding-ada-002",
		"input": text,
	})
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", OpenAIAPIEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	// 发送请求
	client := &http.Client{
		Timeout: time.Duration(30) * time.Second,
	}

	var resp *http.Response
	for retry := 0; retry < MaxRetries; retry++ {
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		logger.Warnf("请求失败，重试 %d/%d: %v", retry+1, MaxRetries, err)
		time.Sleep(RetryDelay)
	}
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 请求失败: %s, 响应: %s", resp.Status, string(body))
	}

	// 解析响应
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("未获取到向量")
	}

	return result.Data[0].Embedding, nil
}
