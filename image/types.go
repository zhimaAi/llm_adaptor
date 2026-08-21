// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package image

type GenerateRequest struct {
	Model          string         `json:"model,omitempty"`
	Prompt         string         `json:"prompt"`
	N              *int           `json:"n,omitempty"`
	Quality        string         `json:"quality,omitempty"`
	ResponseFormat string         `json:"response_format,omitempty"`
	Size           string         `json:"size,omitempty"`
	User           string         `json:"user,omitempty"`
	OutputFormat   string         `json:"output_format,omitempty"`
	ExtraBody      map[string]any `json:"-"`
}

type StreamRequest struct {
	GenerateRequest
}

type Input struct {
	FileID   string `json:"file_id,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type EditRequest struct {
	Model          string         `json:"model,omitempty"`
	Images         []Input        `json:"images"`
	Mask           *Input         `json:"mask,omitempty"`
	Prompt         string         `json:"prompt"`
	N              *int           `json:"n,omitempty"`
	Quality        string         `json:"quality,omitempty"`
	ResponseFormat string         `json:"response_format,omitempty"`
	Size           string         `json:"size,omitempty"`
	User           string         `json:"user,omitempty"`
	OutputFormat   string         `json:"output_format,omitempty"`
	ExtraBody      map[string]any `json:"-"`
}

type EditStreamRequest struct {
	EditRequest
}

type Data struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type TokenDetails struct {
	ImageTokens int `json:"image_tokens,omitempty"`
	TextTokens  int `json:"text_tokens,omitempty"`
}

type Usage struct {
	InputTokens         int          `json:"input_tokens,omitempty"`
	InputTokensDetails  TokenDetails `json:"input_tokens_details,omitempty"`
	OutputTokens        int          `json:"output_tokens,omitempty"`
	OutputTokensDetails TokenDetails `json:"output_tokens_details,omitempty"`
	TotalTokens         int          `json:"total_tokens,omitempty"`
}

type GenerateResponse struct {
	Created      int64  `json:"created,omitempty"`
	Background   string `json:"background,omitempty"`
	Data         []Data `json:"data"`
	OutputFormat string `json:"output_format,omitempty"`
	Quality      string `json:"quality,omitempty"`
	Size         string `json:"size,omitempty"`
	Usage        Usage  `json:"usage,omitempty"`
}

type StreamChunk struct {
	Type              string `json:"type,omitempty"`
	B64JSON           string `json:"b64_json,omitempty"`
	PartialImageIndex int    `json:"partial_image_index,omitempty"`
	Created           int64  `json:"created,omitempty"`
	Background        string `json:"background,omitempty"`
	OutputFormat      string `json:"output_format,omitempty"`
	Quality           string `json:"quality,omitempty"`
	Size              string `json:"size,omitempty"`
	Usage             Usage  `json:"usage,omitempty"`
}

type Stream interface {
	Recv() (*StreamChunk, error)
	Close() error
}
