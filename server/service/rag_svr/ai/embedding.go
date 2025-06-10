package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"server/framework/redis"
)

const (
	// 请求配置
	maxRetries     = 3
	retryInterval  = time.Second
	requestTimeout = 10 * time.Second

	// 缓存配置
	vectorCachePrefix = "vector:"
	vectorCacheTTL    = 24 * time.Hour // 向量缓存24小时
)

var (
	// 并发控制
	requestSemaphore = make(chan struct{}, 10) // 限制最大并发请求数
)

// EmbeddingResponse 向量表示响应
type EmbeddingResponse struct {
	Output struct {
		Embeddings []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"embeddings"`
	} `json:"output"`
}

// GetEmbedding 获取文本的向量表示
func GetEmbedding(text string) ([]float32, error) {
	ctx := context.Background()

	// 尝试从缓存获取
	cacheKey := vectorCachePrefix + text
	if cached, err := redis.Get(ctx, cacheKey); err == nil {
		var vector []float32
		if err := json.Unmarshal([]byte(cached), &vector); err == nil {
			return vector, nil
		}
	}

	// 从环境变量获取配置
	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 EMBEDDING_API_KEY 未设置")
	}
	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("环境变量 EMBEDDING_BASE_URL 未设置")
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
				"texts": []string{text},
			},
		}
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("序列化请求失败: %v", err)
		}

		// 创建请求
		req, err := http.NewRequest("POST", baseURL+"/v1/services/embeddings/text-embedding-v4/text-embedding", bytes.NewBuffer(jsonData))
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
		var embeddingResp EmbeddingResponse
		if err := json.Unmarshal(body, &embeddingResp); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %v", err)
			continue
		}

		if len(embeddingResp.Output.Embeddings) == 0 {
			lastErr = fmt.Errorf("未获取到向量表示")
			continue
		}

		vector := embeddingResp.Output.Embeddings[0].Embedding

		// 缓存向量
		if vectorJSON, err := json.Marshal(vector); err == nil {
			redis.Set(ctx, cacheKey, string(vectorJSON), vectorCacheTTL)
		}

		return vector, nil
	}

	return nil, fmt.Errorf("重试%d次后仍然失败: %v", maxRetries, lastErr)
}

// BatchGetEmbedding 批量获取文本的向量表示
func BatchGetEmbedding(texts []string) ([][]float32, error) {
	ctx := context.Background()
	vectors := make([][]float32, len(texts))
	missedIndices := make([]int, 0)
	missedTexts := make([]string, 0)

	// 尝试从缓存获取
	for i, text := range texts {
		cacheKey := vectorCachePrefix + text
		if cached, err := redis.Get(ctx, cacheKey); err == nil {
			var vector []float32
			if err := json.Unmarshal([]byte(cached), &vector); err == nil {
				vectors[i] = vector
				continue
			}
		}
		missedIndices = append(missedIndices, i)
		missedTexts = append(missedTexts, text)
	}

	// 如果所有向量都在缓存中，直接返回
	if len(missedTexts) == 0 {
		return vectors, nil
	}

	// 获取未命中的向量
	missedVectors, err := getEmbeddingBatch(missedTexts)
	if err != nil {
		return nil, err
	}

	// 更新结果并缓存
	for i, idx := range missedIndices {
		vectors[idx] = missedVectors[i]
		// 缓存向量
		if vectorJSON, err := json.Marshal(missedVectors[i]); err == nil {
			cacheKey := vectorCachePrefix + missedTexts[i]
			redis.Set(ctx, cacheKey, string(vectorJSON), vectorCacheTTL)
		}
	}

	return vectors, nil
}

// getEmbeddingBatch 批量获取向量（内部方法）
func getEmbeddingBatch(texts []string) ([][]float32, error) {
	// 从环境变量获取配置
	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 EMBEDDING_API_KEY 未设置")
	}
	baseURL := os.Getenv("EMBEDDING_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("环境变量 EMBEDDING_BASE_URL 未设置")
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
				"texts": texts,
			},
		}
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("序列化请求失败: %v", err)
		}

		// 创建请求
		req, err := http.NewRequest("POST", baseURL+"/v1/services/embeddings/text-embedding-v4/text-embedding", bytes.NewBuffer(jsonData))
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
		var embeddingResp EmbeddingResponse
		if err := json.Unmarshal(body, &embeddingResp); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %v", err)
			continue
		}

		if len(embeddingResp.Output.Embeddings) == 0 {
			lastErr = fmt.Errorf("未获取到向量表示")
			continue
		}

		vectors := make([][]float32, len(embeddingResp.Output.Embeddings))
		for i, embedding := range embeddingResp.Output.Embeddings {
			vectors[i] = embedding.Embedding
		}

		return vectors, nil
	}

	return nil, fmt.Errorf("重试%d次后仍然失败: %v", maxRetries, lastErr)
}
