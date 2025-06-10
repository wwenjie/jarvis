package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"server/framework/logger"
)

const (
	QwenAPIEndpoint = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	MaxRetries      = 3
	RetryDelay      = time.Second
)

type QwenClient struct {
	apiKey     string
	httpClient *http.Client
	config     *struct {
		APIKey           string
		Provider         string
		ModelName        string
		Temperature      float64
		MaxTokens        int
		TopP             float64
		FrequencyPenalty float64
		PresencePenalty  float64
	}
}

type QwenRequest struct {
	Model      string         `json:"model"`
	Input      QwenInput      `json:"input"`
	Parameters QwenParameters `json:"parameters"`
	Functions  []QwenFunction `json:"functions,omitempty"`
}

type QwenInput struct {
	Messages []QwenMessage `json:"messages"`
}

type QwenMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Tools     []QwenTool     `json:"tools,omitempty"`
	ToolCalls []QwenToolCall `json:"tool_calls,omitempty"`
}

type QwenParameters struct {
	ResultFormat string  `json:"result_format"`
	Temperature  float64 `json:"temperature"`
	TopP         float64 `json:"top_p"`
	TopK         int     `json:"top_k"`
	MaxTokens    int     `json:"max_tokens"`
	Stream       bool    `json:"stream"`
}

type QwenResponse struct {
	Output    QwenOutput `json:"output"`
	Usage     QwenUsage  `json:"usage"`
	RequestID string     `json:"request_id"`
}

type QwenOutput struct {
	Text         string         `json:"text"`
	ToolCalls    []QwenToolCall `json:"tool_calls,omitempty"`
	FinishReason string         `json:"finish_reason"`
}

type QwenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// QwenFunction 函数定义
type QwenFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// QwenFunctionCall 函数调用
type QwenFunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// QwenToolCall 工具调用
type QwenToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function QwenFunctionCall `json:"function"`
}

// QwenTool 通义千问工具结构
type QwenTool struct {
	Type     string                 `json:"type"`
	Function QwenFunctionDefinition `json:"function"`
}

// QwenFunctionDefinition 通义千问函数定义结构
type QwenFunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

func NewQwenClient(config *struct {
	APIKey           string
	Provider         string
	ModelName        string
	Temperature      float64
	MaxTokens        int
	TopP             float64
	FrequencyPenalty float64
	PresencePenalty  float64
}) *QwenClient {
	return &QwenClient{
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: time.Duration(30) * time.Second, // 使用默认超时时间
		},
		config: config,
	}
}

// handleFunctionCall 处理函数调用
func (c *QwenClient) handleFunctionCall(ctx context.Context, response *QwenResponse) (*QwenMessage, error) {
	if len(response.Output.ToolCalls) == 0 {
		return nil, nil
	}

	toolCall := response.Output.ToolCalls[0]
	call := &FunctionCall{
		Name:      toolCall.Function.Name,
		Arguments: toolCall.Function.Arguments,
	}

	result, err := ExecuteFunctionCall(ctx, call)
	if err != nil {
		return nil, fmt.Errorf("执行函数调用失败: %v", err)
	}

	// 将结果转换为字符串
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("序列化函数调用结果失败: %v", err)
	}

	return &QwenMessage{
		Role:    "tool",
		Content: string(resultJSON),
		ToolCalls: []QwenToolCall{
			{
				ID:   toolCall.ID,
				Type: "function",
				Function: QwenFunctionCall{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			},
		},
	}, nil
}

// StreamChat 实现与 Qwen 大模型的流式对话
func (c *QwenClient) StreamChat(ctx context.Context, messages []QwenMessage, context string) (<-chan string, <-chan error) {
	responseChan := make(chan string)
	errorChan := make(chan error)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		// 构建系统消息
		systemMessage := QwenMessage{
			Role:    "system",
			Content: fmt.Sprintf("%s\n\n%s", SystemPrompt, FunctionCallPrompt),
		}

		// 添加系统消息到对话历史
		allMessages := append([]QwenMessage{systemMessage}, messages...)

		// 注册函数
		functions := []QwenFunction{
			{
				Name:        "get_session_info",
				Description: "获取会话信息，包括主题、创建时间等",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"session_id": map[string]interface{}{
							"type":        "number",
							"description": "会话ID",
						},
					},
					"required": []string{"session_id"},
				},
			},
			{
				Name:        "get_chat_history",
				Description: "获取最近的对话历史",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"session_id": map[string]interface{}{
							"type":        "number",
							"description": "会话ID",
						},
						"limit": map[string]interface{}{
							"type":        "number",
							"description": "返回记录数量限制",
						},
					},
					"required": []string{"session_id", "limit"},
				},
			},
			{
				Name:        "get_user_preferences",
				Description: "获取用户偏好设置",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"user_id": map[string]interface{}{
							"type":        "number",
							"description": "用户ID",
						},
					},
					"required": []string{"user_id"},
				},
			},
			{
				Name:        "get_related_memories",
				Description: "获取相关的其他记忆",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"memory_id": map[string]interface{}{
							"type":        "number",
							"description": "记忆ID",
						},
						"limit": map[string]interface{}{
							"type":        "number",
							"description": "返回记录数量限制",
						},
					},
					"required": []string{"memory_id", "limit"},
				},
			},
		}

		// 构建请求
		req := QwenRequest{
			Model: c.config.ModelName,
			Input: QwenInput{
				Messages: allMessages,
			},
			Parameters: QwenParameters{
				ResultFormat: "message",
				Temperature:  c.config.Temperature,
				TopP:         c.config.TopP,
				TopK:         10,
				MaxTokens:    c.config.MaxTokens,
				Stream:       true,
			},
			Functions: functions,
		}

		// 序列化请求
		reqBody, err := json.Marshal(req)
		if err != nil {
			errorChan <- fmt.Errorf("序列化请求失败: %v", err)
			return
		}

		// 创建 HTTP 请求
		httpReq, err := http.NewRequestWithContext(ctx, "POST", QwenAPIEndpoint, bytes.NewBuffer(reqBody))
		if err != nil {
			errorChan <- fmt.Errorf("创建请求失败: %v", err)
			return
		}

		// 设置请求头
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

		// 发送请求
		var resp *http.Response
		for retry := 0; retry < MaxRetries; retry++ {
			resp, err = c.httpClient.Do(httpReq)
			if err == nil {
				break
			}
			logger.Warnf("请求失败，重试 %d/%d: %v", retry+1, MaxRetries, err)
			time.Sleep(RetryDelay)
		}
		if err != nil {
			errorChan <- fmt.Errorf("发送请求失败: %v", err)
			return
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorChan <- fmt.Errorf("API 请求失败: %s, 响应: %s", resp.Status, string(body))
			return
		}

		// 处理流式响应
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				errorChan <- fmt.Errorf("读取响应失败: %v", err)
				return
			}

			// 跳过空行
			if strings.TrimSpace(line) == "" {
				continue
			}

			// 解析 SSE 数据
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				var response QwenResponse
				if err := json.Unmarshal([]byte(data), &response); err != nil {
					logger.Errorf("解析响应失败: %v", err)
					continue
				}

				// 处理函数调用
				msg, err := c.handleFunctionCall(ctx, &response)
				if err != nil {
					errorChan <- err
					return
				}

				// 将函数调用结果添加到对话历史
				allMessages = append(allMessages, *msg)

				// 发送文本片段
				if response.Output.Text != "" {
					select {
					case responseChan <- response.Output.Text:
					case <-ctx.Done():
						return
					}
				}

				// 检查是否完成
				if response.Output.FinishReason != "" {
					break
				}
			}
		}
	}()

	return responseChan, errorChan
}

// 修改处理流式响应的部分
func (c *QwenClient) handleStreamResponse(ctx context.Context, response *QwenResponse, responseChan chan<- string, errorChan chan<- error) {
	// 处理函数调用
	if len(response.Output.ToolCalls) > 0 {
		msg, err := c.handleFunctionCall(ctx, response)
		if err != nil {
			errorChan <- err
			return
		}
		if msg != nil {
			responseChan <- msg.Content
		}
		return
	}

	// 发送文本片段
	if response.Output.Text != "" {
		responseChan <- response.Output.Text
	}

	// 检查是否结束
	if response.Output.FinishReason == "stop" {
		close(responseChan)
	}
}
