package chat

import "fmt"

type Accumulator struct {
	response  CreateResponse
	choices   map[int]*Choice
	toolCalls map[int]map[int]*ToolCall
}

func NewAccumulator() *Accumulator {
	return &Accumulator{
		choices:   make(map[int]*Choice),
		toolCalls: make(map[int]map[int]*ToolCall),
	}
}

func (a *Accumulator) Add(chunk *StreamChunk) error {
	if chunk == nil {
		return fmt.Errorf("stream chunk is nil")
	}
	if a.response.ID == "" {
		a.response.ID = chunk.ID
		a.response.Object = "chat.completion"
		a.response.Created = chunk.Created
		a.response.Model = chunk.Model
		a.response.SystemFingerprint = chunk.SystemFingerprint
		a.response.ServiceTier = chunk.ServiceTier
	}
	if chunk.Usage != nil {
		a.response.Usage = *chunk.Usage
	}
	for _, chunkChoice := range chunk.Choices {
		choice := a.choices[chunkChoice.Index]
		if choice == nil {
			choice = &Choice{Index: chunkChoice.Index}
			a.choices[chunkChoice.Index] = choice
		}
		if chunkChoice.Delta.Role != "" {
			choice.Message.Role = chunkChoice.Delta.Role
		}
		choice.Message.Content = appendMessageContent(choice.Message.Content, chunkChoice.Delta.Content)
		choice.Message.ReasoningContent += chunkChoice.Delta.ReasoningContent
		choice.Message.Refusal += chunkChoice.Delta.Refusal
		if chunkChoice.FinishReason != "" {
			choice.FinishReason = chunkChoice.FinishReason
		}
		a.addToolCalls(chunkChoice.Index, chunkChoice.Delta.ToolCalls)
	}
	return nil
}

func (a *Accumulator) Response() *CreateResponse {
	response := a.response
	response.Choices = make([]Choice, 0, len(a.choices))
	for index := 0; len(response.Choices) < len(a.choices); index++ {
		choice := a.choices[index]
		if choice == nil {
			continue
		}
		if calls := a.toolCalls[index]; len(calls) > 0 {
			choice.Message.ToolCalls = make([]ToolCall, 0, len(calls))
			for callIndex := 0; len(choice.Message.ToolCalls) < len(calls); callIndex++ {
				if call := calls[callIndex]; call != nil {
					choice.Message.ToolCalls = append(choice.Message.ToolCalls, *call)
				}
			}
		}
		response.Choices = append(response.Choices, *choice)
	}
	return &response
}

func (a *Accumulator) addToolCalls(choiceIndex int, chunks []ToolCall) {
	if len(chunks) == 0 {
		return
	}
	if a.toolCalls[choiceIndex] == nil {
		a.toolCalls[choiceIndex] = make(map[int]*ToolCall)
	}
	for position, chunk := range chunks {
		callIndex := position
		if chunk.Index != nil {
			callIndex = *chunk.Index
		}
		call := a.toolCalls[choiceIndex][callIndex]
		if call == nil {
			call = &ToolCall{Index: &callIndex}
			a.toolCalls[choiceIndex][callIndex] = call
		}
		if chunk.ID != "" {
			call.ID = chunk.ID
		}
		if chunk.Type != "" {
			call.Type = chunk.Type
		}
		call.Function.Name += chunk.Function.Name
		call.Function.Arguments += chunk.Function.Arguments
	}
}

func appendMessageContent(current, delta MessageContent) MessageContent {
	if delta.Text != nil {
		if current.Text == nil {
			empty := ""
			current.Text = &empty
		}
		*current.Text += *delta.Text
	}
	if len(delta.Parts) > 0 {
		current.Parts = append(current.Parts, delta.Parts...)
	}
	return current
}
