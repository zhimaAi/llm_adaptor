// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import "strings"

func joinURLPath(baseURL, path string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path = strings.TrimSpace(path)
	if baseURL == "" {
		if path == "" {
			return ""
		}
		return "/" + strings.TrimLeft(path, "/")
	}
	if path == "" {
		return baseURL
	}
	return baseURL + "/" + strings.TrimLeft(path, "/")
}

func appendURLSegment(baseURL, segment string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	segment = strings.Trim(strings.TrimSpace(segment), "/")
	if segment == "" || strings.EqualFold(lastURLPathSegment(baseURL), segment) {
		return baseURL
	}
	return joinURLPath(baseURL, segment)
}

func appendAzureOpenAIV1Path(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	lower := strings.ToLower(baseURL)
	if strings.HasSuffix(lower, "/openai/v1") {
		return baseURL
	}
	if strings.HasSuffix(lower, "/openai") {
		return joinURLPath(baseURL, "v1")
	}
	return joinURLPath(baseURL, "openai/v1")
}

func lastURLPathSegment(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if index := strings.LastIndexByte(value, '/'); index >= 0 {
		return value[index+1:]
	}
	return value
}
