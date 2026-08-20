// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/image"
)

const (
	imageResponseFormatBase64 = "b64_json"
	imageFormatPNG            = "png"
	imageFormatJPEG           = "jpeg"
)

func normalizeImageResponse(ctx context.Context, config ClientConfig, provider Provider, hint string, request *image.GenerateRequest, response *image.GenerateResponse) error {
	for index := range response.Data {
		if err := normalizeImageData(ctx, config, provider, hint, request, &response.Data[index]); err != nil {
			return err
		}
	}
	return nil
}

func normalizeImageData(ctx context.Context, config ClientConfig, provider Provider, hint string, request *image.GenerateRequest, data *image.Data) error {
	if strings.HasPrefix(data.URL, "data:") {
		mimeType, encoded, err := parseImageDataURL(data.URL)
		if err != nil {
			return err
		}
		data.MIMEType, data.B64JSON, data.URL = mimeType, encoded, ""
	}
	if request.ResponseFormat == imageResponseFormatBase64 && data.B64JSON == "" && data.URL != "" {
		payload, mimeType, err := downloadImage(ctx, config.HTTPClient, provider, hint, data.URL)
		if err != nil {
			return err
		}
		data.B64JSON = base64.StdEncoding.EncodeToString(payload)
		data.MIMEType, data.URL = mimeType, ""
	}
	if data.B64JSON != "" {
		if _, err := base64.StdEncoding.DecodeString(data.B64JSON); err != nil {
			return fmt.Errorf("%w: invalid base64 image: %v", ErrInvalidRequest, err)
		}
	}
	data.Format = imageFormat(data.MIMEType, data.URL, request.OutputFormat)
	if data.Format == "" && data.B64JSON != "" {
		data.Format = defaultImageFormat(provider)
	}
	if data.MIMEType == "" {
		data.MIMEType = imageMIMEType(data.Format)
	}
	if request.ResponseFormat == imageResponseFormatBase64 && (data.B64JSON == "" || data.Format == "") {
		return fmt.Errorf("%w: base64 image data and format are required", ErrInvalidRequest)
	}
	return nil
}

func defaultImageFormat(provider Provider) string {
	if provider == ProviderDoubao {
		return imageFormatJPEG
	}
	return imageFormatPNG
}

func parseImageDataURL(value string) (string, string, error) {
	comma := strings.IndexByte(value, ',')
	if comma < 0 {
		return "", "", fmt.Errorf("%w: invalid image data URL", ErrInvalidRequest)
	}
	header, encoded := value[5:comma], value[comma+1:]
	if !strings.Contains(header, ";base64") {
		return "", "", fmt.Errorf("%w: image data URL is not base64 encoded", ErrInvalidRequest)
	}
	mimeType := strings.TrimSpace(strings.SplitN(header, ";", 2)[0])
	if _, err := base64.StdEncoding.DecodeString(encoded); err != nil {
		return "", "", fmt.Errorf("%w: invalid image data URL: %v", ErrInvalidRequest, err)
	}
	return mimeType, encoded, nil
}

func downloadImage(ctx context.Context, client *http.Client, provider Provider, hint, rawURL string) ([]byte, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return nil, "", &APIError{Provider: provider, StatusCode: response.StatusCode, Message: response.Status, CredentialHint: hint}
	}
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", err
	}
	if len(payload) == 0 {
		return nil, "", fmt.Errorf("%w: downloaded image is empty", ErrInvalidRequest)
	}
	mimeType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	return payload, mimeType, nil
}

func imageFormat(mimeType, rawURL, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	}
	if parsed, err := url.Parse(rawURL); err == nil {
		if extension := strings.TrimPrefix(strings.ToLower(path.Ext(parsed.Path)), "."); extension != "" {
			if extension == "jpg" {
				return "jpeg"
			}
			return extension
		}
	}
	fallback = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(fallback)), ".")
	if fallback == "jpg" {
		return "jpeg"
	}
	return fallback
}

func imageMIMEType(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png", "webp", "gif":
		return "image/" + format
	default:
		return ""
	}
}
