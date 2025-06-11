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
	"sync"
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
	errorChan := make(chan error, 1)

	// 使用 sync.Once 确保通道只被关闭一次
	var closeOnce sync.Once
	closeChannels := func() {
		closeOnce.Do(func() {
			close(responseChan)
			close(errorChan)
		})
	}

	go func() {
		defer closeChannels()

		// 构建系统消息
		systemMessage := QwenMessage{
			Role:    "system",
			Content: context,
		}

		// 将系统消息添加到消息列表的开头
		messages = append([]QwenMessage{systemMessage}, messages...)

		// 构建请求
		request := QwenRequest{
			Model: c.config.ModelName,
			Input: QwenInput{
				Messages: messages,
			},
			Parameters: QwenParameters{
				ResultFormat: "text",
				Temperature:  c.config.Temperature,
				TopP:         c.config.TopP,
				TopK:         10,
				MaxTokens:    c.config.MaxTokens,
				Stream:       true,
			},
		}

		// 序列化请求
		reqBody, err := json.Marshal(request)
		if err != nil {
			logger.Errorf("序列化请求失败: %v", err)
			errorChan <- fmt.Errorf("序列化请求失败: %v", err)
			return
		}

		// 创建 HTTP 请求
		httpReq, err := http.NewRequestWithContext(ctx, "POST", QwenAPIEndpoint, bytes.NewBuffer(reqBody))
		if err != nil {
			logger.Errorf("创建请求失败: %v", err)
			errorChan <- fmt.Errorf("创建请求失败: %v", err)
			return
		}

		// 设置请求头
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		httpReq.Header.Set("X-DashScope-SSE", "enable") // 添加 SSE 支持

		// 发送请求
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			logger.Errorf("发送请求失败: %v", err)
			errorChan <- fmt.Errorf("发送请求失败: %v", err)
			return
		}
		defer resp.Body.Close()

		// 检查响应状态码
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			logger.Errorf("请求失败: status=%d, body=%s", resp.StatusCode, string(body))
			errorChan <- fmt.Errorf("请求失败: status=%d, body=%s", resp.StatusCode, string(body))
			return
		}

		// 处理流式响应
		reader := bufio.NewReader(resp.Body)
		var lastText string

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// 确保最后一个片段被发送
					if lastText != "" {
						select {
						case responseChan <- lastText:
							logger.Infof("发送最后一个文本片段: %s", lastText)
						case <-ctx.Done():
							return
						}
					}
					return
				}
				logger.Errorf("读取响应失败: %v", err)
				errorChan <- fmt.Errorf("读取响应失败: %v", err)
				return
			}

			// 记录原始行
			logger.Infof("收到原始行: %s", line)
			line = strings.TrimSpace(line) // 先去除空白字符
			logger.Infof("处理后的行: %s", line)
			logger.Infof("原始行是否以 data: 开头: %v", strings.HasPrefix(line, "data:"))

			// 解析 SSE 数据
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimPrefix(line, "data:")
				data = strings.TrimSpace(data) // 去除可能的空白字符
				logger.Infof("收到原始响应数据: %s", data)

				// 尝试解析为 JSON
				var response QwenResponse
				if err := json.Unmarshal([]byte(data), &response); err != nil {
					logger.Errorf("解析响应失败: %v, data=%s", err, data)
					continue
				}
				logger.Infof("解析后的响应: finish_reason=%s, text=%s", response.Output.FinishReason, response.Output.Text)

				// 发送文本片段
				if response.Output.Text != "" {
					lastText = response.Output.Text
					select {
					case responseChan <- response.Output.Text:
						logger.Infof("成功发送响应: %s", response.Output.Text)
					case <-ctx.Done():
						return
					}
				}

				// 检查是否完成
				if response.Output.FinishReason == "stop" {
					logger.Infof("收到完成信号，准备发送最后响应")
					// 发送最后的完整响应
					select {
					case responseChan <- response.Output.Text:
						logger.Infof("发送最后的完整响应: %s", response.Output.Text)
					case <-ctx.Done():
						logger.Infof("上下文已取消，退出")
						return
					}
					logger.Infof("响应完成, 原因: %s, 完整响应: %s", response.Output.FinishReason, response.Output.Text)
					// 等待一小段时间确保最后一个片段被处理
					time.Sleep(100 * time.Millisecond)
					return
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
