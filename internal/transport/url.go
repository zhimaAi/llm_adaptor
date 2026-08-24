// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package transport

import "strings"

func JoinURLPath(baseURL, path string) string {
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

func AppendURLSegment(baseURL, segment string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	segment = strings.Trim(strings.TrimSpace(segment), "/")
	if segment == "" || strings.EqualFold(lastURLPathSegment(baseURL), segment) {
		return baseURL
	}
	return JoinURLPath(baseURL, segment)
}

func lastURLPathSegment(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if index := strings.LastIndexByte(value, '/'); index >= 0 {
		return value[index+1:]
	}
	return value
}
