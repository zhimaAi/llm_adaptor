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
	imageFormatJPG            = "jpg"
	imageFormatJPEG           = "jpeg"
)

type imageRequestOptions struct {
	ResponseFormat string
	OutputFormat   string
}

func normalizeImageResponse(ctx context.Context, config ClientConfig, provider Provider, hint string, request imageRequestOptions, response *image.GenerateResponse) error {
	if response == nil || len(response.Data) == 0 {
		return fmt.Errorf("%w: image response contains no data", ErrInvalidRequest)
	}
	format := normalizeImageFormat(response.OutputFormat)
	for index := range response.Data {
		itemFormat, err := normalizeImageData(ctx, config, provider, hint, request, &response.Data[index])
		if err != nil {
			return err
		}
		if format == "" {
			format = itemFormat
		}
	}
	if format == "" {
		format = imageFormatJPEG
	}
	response.OutputFormat = format
	return nil
}

func normalizeImageData(ctx context.Context, config ClientConfig, provider Provider, hint string, request imageRequestOptions, data *image.Data) (string, error) {
	originalURL := data.URL
	mimeType := ""
	if strings.HasPrefix(data.URL, "data:") {
		parsedMIMEType, encoded, err := parseImageDataURL(data.URL)
		if err != nil {
			return "", err
		}
		mimeType, data.B64JSON, data.URL = parsedMIMEType, encoded, ""
	}
	if request.ResponseFormat == imageResponseFormatBase64 && data.B64JSON == "" && data.URL != "" {
		payload, downloadedMIMEType, err := downloadImage(ctx, config.HTTPClient, provider, hint, data.URL)
		if err != nil {
			return "", err
		}
		data.B64JSON = base64.StdEncoding.EncodeToString(payload)
		mimeType, data.URL = downloadedMIMEType, ""
	}
	if data.B64JSON != "" {
		if _, err := base64.StdEncoding.DecodeString(data.B64JSON); err != nil {
			return "", fmt.Errorf("%w: invalid base64 image: %v", ErrInvalidRequest, err)
		}
	}
	format := imageFormat(mimeType, originalURL, request.OutputFormat)
	if format == "" {
		format = imageFormatJPEG
	}
	if request.ResponseFormat == imageResponseFormatBase64 && data.B64JSON == "" {
		return "", fmt.Errorf("%w: base64 image data is required", ErrInvalidRequest)
	}
	return format, nil
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
			if format := normalizeImageFormat(extension); format != "" {
				return format
			}
		}
	}
	return normalizeImageFormat(fallback)
}

func normalizeImageFormat(format string) string {
	format = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(format)), ".")
	switch format {
	case imageFormatJPG, imageFormatJPEG:
		return "jpeg"
	case imageFormatPNG, "webp", "gif":
		return format
	default:
		return ""
	}
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
