// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package shared

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

type ImageFileData struct {
	Filename    string
	ContentType string
	Data        []byte
}

func ReadImageFile(file image.File) (ImageFileData, error) {
	if nilReader(file.Reader) {
		return ImageFileData{}, fmt.Errorf("%w: image file reader is required", provider.ErrInvalidRequest)
	}
	data, err := io.ReadAll(file.Reader)
	if err != nil {
		return ImageFileData{}, fmt.Errorf("%w: read image file: %v", provider.ErrInvalidRequest, err)
	}
	if len(data) == 0 {
		return ImageFileData{}, fmt.Errorf("%w: image file is empty", provider.ErrInvalidRequest)
	}
	contentType := strings.TrimSpace(file.ContentType)
	if parsed, _, parseErr := mime.ParseMediaType(contentType); parseErr == nil {
		contentType = parsed
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(file.Filename)))
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}
	filename := strings.TrimSpace(file.Filename)
	if filename == "" {
		filename = "image" + extensionForMIME(contentType)
	}
	return ImageFileData{Filename: filepath.Base(filename), ContentType: contentType, Data: data}, nil
}

func ImageFilesDataURL(files []image.File) ([]string, error) {
	result := make([]string, 0, len(files))
	for index, file := range files {
		data, err := ReadImageFile(file)
		if err != nil {
			return nil, fmt.Errorf("%w: image at index %d: %v", provider.ErrInvalidRequest, index, err)
		}
		result = append(result, "data:"+data.ContentType+";base64,"+base64.StdEncoding.EncodeToString(data.Data))
	}
	return result, nil
}

func nilReader(reader io.Reader) bool {
	if reader == nil {
		return true
	}
	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func extensionForMIME(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}
