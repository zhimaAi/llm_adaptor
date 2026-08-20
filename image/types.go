package image

import "encoding/json"

type GenerateRequest struct {
	Model             string         `json:"model,omitempty"`
	Prompt            string         `json:"prompt"`
	Image             []string       `json:"image,omitempty"`
	N                 *int           `json:"n,omitempty"`
	Quality           string         `json:"quality,omitempty"`
	ResponseFormat    string         `json:"response_format,omitempty"`
	Size              string         `json:"size,omitempty"`
	Style             string         `json:"style,omitempty"`
	User              string         `json:"user,omitempty"`
	Background        string         `json:"background,omitempty"`
	OutputFormat      string         `json:"output_format,omitempty"`
	OutputCompression *int           `json:"output_compression,omitempty"`
	Moderation        string         `json:"moderation,omitempty"`
	ExtraBody         map[string]any `json:"-"`
}

type StreamRequest struct {
	GenerateRequest
}

type Data struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	Format        string `json:"-"`
	MIMEType      string `json:"-"`
}

type Usage struct {
	InputTokens     int `json:"input_tokens,omitempty"`
	OutputTokens    int `json:"output_tokens,omitempty"`
	TotalTokens     int `json:"total_tokens,omitempty"`
	GeneratedImages int `json:"generated_images,omitempty"`
}

type GenerateResponse struct {
	Created     int64                      `json:"created,omitempty"`
	Data        []Data                     `json:"data"`
	Usage       Usage                      `json:"usage,omitempty"`
	ExtraFields map[string]json.RawMessage `json:"-"`
	RawResponse json.RawMessage            `json:"-"`
}

type StreamChunk struct {
	Type        string                     `json:"type,omitempty"`
	Model       string                     `json:"model,omitempty"`
	Created     int64                      `json:"created,omitempty"`
	ImageIndex  int                        `json:"image_index,omitempty"`
	URL         string                     `json:"url,omitempty"`
	B64JSON     string                     `json:"b64_json,omitempty"`
	Size        string                     `json:"size,omitempty"`
	Format      string                     `json:"-"`
	MIMEType    string                     `json:"-"`
	Usage       Usage                      `json:"usage,omitempty"`
	ExtraFields map[string]json.RawMessage `json:"-"`
	RawResponse json.RawMessage            `json:"-"`
}

type Stream interface {
	Recv() (*StreamChunk, error)
	Close() error
}
