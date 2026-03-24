package model

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Claude Messages API 请求结构
type ClaudeMessagesRequest struct {
	Model      string                   `json:"model"`
	MaxTokens  int                      `json:"max_tokens"`
	Messages   []map[string]interface{} `json:"messages"`
	Stream     bool                     `json:"stream,omitempty"`
	System     interface{}              `json:"system,omitempty"` // Support both string and array formats
	Tools      []map[string]interface{} `json:"tools,omitempty"`
	ToolChoice interface{}              `json:"tool_choice,omitempty"`
}

// Claude Messages API 响应结构
type ClaudeMessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence interface{}    `json:"stop_sequence"`
	Usage        ClaudeUsage    `json:"usage"`
}

// Claude Messages API 流式响应结构
type ClaudeStreamResponse struct {
	Type         string         `json:"type"` // message_start, content_block_start, content_block_delta, content_block_stop, message_delta, message_stop
	Message      *ClaudeMessage `json:"message,omitempty"`
	Index        int            `json:"index,omitempty"`
	ContentBlock *ContentBlock  `json:"content_block,omitempty"`
	Delta        *ContentDelta  `json:"delta,omitempty"`
	Usage        *ClaudeUsage   `json:"usage,omitempty"`
}

type ClaudeMessage struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Content      []interface{} `json:"content"`
	Model        string        `json:"model"`
	StopReason   interface{}   `json:"stop_reason"`
	StopSequence interface{}   `json:"stop_sequence"`
	Usage        ClaudeUsage   `json:"usage"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ContentDelta struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Count Tokens API 请求结构
type CountTokensRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	System   string                   `json:"system,omitempty"`
}

// Count Tokens API 响应结构
type CountTokensResponse struct {
	InputTokens int `json:"input_tokens"`
}

// ReturnClaudeMessagesResponse 返回 Claude Messages API 格式的响应
func ReturnClaudeMessagesResponse(text string, stream bool, gc *gin.Context, model string) error {
	if stream {
		return claudeStreamResponse(text, gc, model)
	} else {
		return claudeNoStreamResponse(text, gc, model)
	}
}

// 流式响应
func claudeStreamResponse(text string, gc *gin.Context, model string) error {
	// 发送 content_block_delta 事件
	deltaResp := &ClaudeStreamResponse{
		Type:  "content_block_delta",
		Index: 0,
		Delta: &ContentDelta{
			Type: "text_delta",
			Text: text,
		},
	}

	jsonBytes, err := json.Marshal(deltaResp)
	if err != nil {
		return err
	}

	jsonBytes = append([]byte("event: content_block_delta\ndata: "), jsonBytes...)
	jsonBytes = append(jsonBytes, []byte("\n\n")...)

	gc.Writer.Write(jsonBytes)
	gc.Writer.Flush()
	return nil
}

// 非流式响应
func claudeNoStreamResponse(text string, gc *gin.Context, model string) error {
	claudeResp := &ClaudeMessagesResponse{
		ID:   "msg_" + uuid.New().String(),
		Type: "message",
		Role: "assistant",
		Content: []ContentBlock{
			{
				Type: "text",
				Text: text,
			},
		},
		Model:        model,
		StopReason:   "end_turn",
		StopSequence: nil,
		Usage: ClaudeUsage{
			InputTokens:  10,            // 这里可以实际计算
			OutputTokens: len(text) / 4, // 粗略估计
		},
	}

	gc.JSON(200, claudeResp)
	return nil
}

// SendClaudeStreamStart 发送流式响应的开始事件
func SendClaudeStreamStart(gc *gin.Context, model string) error {
	messageID := "msg_" + uuid.New().String()

	// message_start 事件
	startResp := &ClaudeStreamResponse{
		Type: "message_start",
		Message: &ClaudeMessage{
			ID:           messageID,
			Type:         "message",
			Role:         "assistant",
			Content:      []interface{}{},
			Model:        model,
			StopReason:   nil,
			StopSequence: nil,
			Usage: ClaudeUsage{
				InputTokens:  10,
				OutputTokens: 0,
			},
		},
	}

	jsonBytes, err := json.Marshal(startResp)
	if err != nil {
		return err
	}
	jsonBytes = append([]byte("event: message_start\ndata: "), jsonBytes...)
	jsonBytes = append(jsonBytes, []byte("\n\n")...)
	gc.Writer.Write(jsonBytes)
	gc.Writer.Flush()

	// content_block_start 事件
	blockStartResp := &ClaudeStreamResponse{
		Type:  "content_block_start",
		Index: 0,
		ContentBlock: &ContentBlock{
			Type: "text",
			Text: "",
		},
	}

	jsonBytes, err = json.Marshal(blockStartResp)
	if err != nil {
		return err
	}
	jsonBytes = append([]byte("event: content_block_start\ndata: "), jsonBytes...)
	jsonBytes = append(jsonBytes, []byte("\n\n")...)
	gc.Writer.Write(jsonBytes)
	gc.Writer.Flush()

	return nil
}

// SendClaudeStreamStop 发送流式响应的结束事件
func SendClaudeStreamStop(gc *gin.Context) error {
	// content_block_stop 事件
	blockStopResp := &ClaudeStreamResponse{
		Type:  "content_block_stop",
		Index: 0,
	}

	jsonBytes, err := json.Marshal(blockStopResp)
	if err != nil {
		return err
	}
	jsonBytes = append([]byte("event: content_block_stop\ndata: "), jsonBytes...)
	jsonBytes = append(jsonBytes, []byte("\n\n")...)
	gc.Writer.Write(jsonBytes)
	gc.Writer.Flush()

	// message_delta 事件
	deltaResp := &ClaudeStreamResponse{
		Type: "message_delta",
		Delta: &ContentDelta{
			Type: "text_delta",
			Text: "",
		},
		Usage: &ClaudeUsage{
			OutputTokens: 100, // 粗略估计
		},
	}

	jsonBytes, err = json.Marshal(deltaResp)
	if err != nil {
		return err
	}
	jsonBytes = append([]byte("event: message_delta\ndata: "), jsonBytes...)
	jsonBytes = append(jsonBytes, []byte("\n\n")...)
	gc.Writer.Write(jsonBytes)
	gc.Writer.Flush()

	// message_stop 事件
	stopResp := &ClaudeStreamResponse{
		Type: "message_stop",
	}

	jsonBytes, err = json.Marshal(stopResp)
	if err != nil {
		return err
	}
	jsonBytes = append([]byte("event: message_stop\ndata: "), jsonBytes...)
	jsonBytes = append(jsonBytes, []byte("\n\n")...)
	gc.Writer.Write(jsonBytes)
	gc.Writer.Flush()

	return nil
}
