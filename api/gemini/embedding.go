// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package gemini

type EmbeddingRequest struct {
	Content              Content `json:"content"`
	TaskType             string  `json:"taskType,omitempty"`
	Title                string  `json:"title,omitempty"`
	OutputDimensionality int     `json:"outputDimensionality,omitempty"`
}

type Content struct {
	Role  string `json:"role,omitempty"`
	Parts []Part `json:"parts"`
}
type Part struct {
	Text             string            `json:"text"`
	Thought          bool              `json:"thought,omitempty"`
	ThoughtSignature string            `json:"thoughtSignature,omitempty"`
	InlineData       *Blob             `json:"inlineData,omitempty"`
	FunctionCall     string            `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
	FileData         *FileData         `json:"fileData,omitempty"`
}

type Blob struct {
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}
type FunctionResponse struct {
	Name string `json:"name,omitempty"`
	Args string `json:"args,omitempty"`
}
type FileData struct {
	MimeType string `json:"mimeType,omitempty"`
	FileUri  string `json:"fileUri,omitempty"`
}

type EmbeddingResponse struct {
	Embedding ContentEmbedding `json:"embedding"`
}
type ContentEmbedding struct {
	Values []float64 `json:"values"`
}
