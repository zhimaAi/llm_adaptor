// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

func TestMiniMaxSpeechCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != MiniMaxSpeechPath {
			t.Errorf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get(headerAuthorization) != bearerPrefix+"test-key" {
			t.Errorf("unexpected authorization header")
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body["stream"] != false || body["custom"] != "value" {
			t.Errorf("unexpected body: %#v", body)
		}
		writer.Header().Set(headerContentType, mediaTypeJSON)
		_, _ = io.WriteString(writer, `{"data":{"audio":"abcd","status":2},"extra_info":{"usage_characters":4},"trace_id":"trace-1","provider_field":"kept","base_resp":{"status_code":0,"status_msg":"success"}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Provider:    ProviderMiniMax,
		BaseURL:     server.URL,
		Credentials: CredentialConfig{APIKeys: "test-key"},
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Speech.Create(context.Background(), &speech.CreateRequest{
		Model:     "speech-2.8-hd",
		Text:      "test",
		ExtraBody: map[string]any{"custom": "value"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Data == nil || response.Data.Audio != "abcd" || response.TraceID != "trace-1" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if _, exists := response.ExtraFields["provider_field"]; !exists {
		t.Fatalf("provider extension was not preserved: %#v", response.ExtraFields)
	}
}

func TestMiniMaxSpeechStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set(headerContentType, "text/event-stream")
		_, _ = io.WriteString(writer, "data: {\"data\":{\"audio\":\"aa\",\"status\":1},\"trace_id\":\"trace-1\",\"base_resp\":{\"status_code\":0,\"status_msg\":\"success\"}}\n\n")
		_, _ = io.WriteString(writer, "{\"data\":{\"audio\":\"bb\",\"status\":2},\"trace_id\":\"trace-1\",\"base_resp\":{\"status_code\":0,\"status_msg\":\"success\"}}\n")
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Provider:    ProviderMiniMax,
		BaseURL:     server.URL,
		Credentials: CredentialConfig{APIKeys: "test-key"},
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := client.Speech.Stream(context.Background(), &speech.StreamRequest{
		CreateRequest: speech.CreateRequest{Model: "speech-2.8-turbo", Text: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	first, err := stream.Recv()
	if err != nil || first.Data.Audio != "aa" {
		t.Fatalf("unexpected first chunk: %#v, %v", first, err)
	}
	second, err := stream.Recv()
	if err != nil || second.Data.Audio != "bb" {
		t.Fatalf("unexpected second chunk: %#v, %v", second, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestMiniMaxSpeechContextCancellation(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-release
	}))

	client, err := NewClient(ClientConfig{
		Provider:    ProviderMiniMax,
		BaseURL:     server.URL,
		Credentials: CredentialConfig{APIKeys: "test-key"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = client.Speech.Create(ctx, &speech.CreateRequest{Model: "speech-2.8-hd", Text: "test"})
	close(release)
	server.Close()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestMiniMaxVoiceManagementUsesSelectedCredential(t *testing.T) {
	tempPath := filepath.Join(t.TempDir(), "voice.mp3")
	if err := os.WriteFile(tempPath, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	var authorizations []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		authorizations = append(authorizations, request.Header.Get(headerAuthorization))
		if authorization := request.Header.Get(headerAuthorization); !strings.HasPrefix(authorization, bearerPrefix) || strings.ContainsAny(authorization, ",@") {
			t.Errorf("unexpected authorization: %q", request.Header.Get(headerAuthorization))
		}
		switch request.URL.Path {
		case miniMaxListVoicesPath:
			_, _ = io.WriteString(writer, `{"system_voice":[{"voice_id":"system-1"}],"base_resp":{"status_code":0,"status_msg":"success"}}`)
		case miniMaxUploadVoiceFilePath:
			reader, err := request.MultipartReader()
			if err != nil {
				t.Errorf("multipart reader: %v", err)
				return
			}
			fields := readMultipartFields(t, reader)
			if (fields[miniMaxUploadPurposeField] != miniMaxVoiceClonePurpose && fields[miniMaxUploadPurposeField] != miniMaxPromptAudioPurpose) || fields[miniMaxUploadFileField] != "audio" {
				t.Errorf("unexpected multipart fields: %#v", fields)
			}
			_, _ = io.WriteString(writer, `{"file":{"file_id":123,"bytes":5,"filename":"voice.mp3","purpose":"voice_clone"},"base_resp":{"status_code":0,"status_msg":"success"}}`)
		case miniMaxCloneVoicePath:
			var body speech.CloneVoiceRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode clone request: %v", err)
			}
			if body.FileID != 123 || body.VoiceID != "test-voice" {
				t.Errorf("unexpected clone request: %#v", body)
			}
			_, _ = io.WriteString(writer, `{"input_sensitive":false,"demo_audio":"https://example.com/demo.mp3","base_resp":{"status_code":0,"status_msg":"success"}}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Provider: ProviderMiniMax, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "test-key@2"}})
	if err != nil {
		t.Fatal(err)
	}
	voices, err := client.Speech.ListVoices(context.Background(), &speech.ListVoicesRequest{VoiceType: speech.VoiceTypeAll})
	if err != nil || len(voices.SystemVoices) != 1 {
		t.Fatalf("unexpected voices: %#v, %v", voices, err)
	}
	upload, err := client.Speech.UploadVoiceFile(context.Background(), &speech.UploadVoiceFileRequest{Purpose: miniMaxVoiceClonePurpose, FilePath: tempPath})
	if err != nil || upload.File.FileID != 123 {
		t.Fatalf("unexpected upload: %#v, %v", upload, err)
	}
	clone, err := client.Speech.CloneVoice(context.Background(), &speech.CloneVoiceRequest{FileID: upload.File.FileID, VoiceID: "test-voice"})
	if err != nil || clone.DemoAudio == "" {
		t.Fatalf("unexpected clone response: %#v, %v", clone, err)
	}

	authorizations = nil
	client, err = NewClient(ClientConfig{Provider: ProviderMiniMax, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key-1,key-2"}})
	if err != nil {
		t.Fatal(err)
	}
	atomic, err := client.Speech.CloneVoiceFromFiles(context.Background(), &speech.CloneVoiceFromFilesRequest{
		SourceFilePath: tempPath,
		PromptFilePath: tempPath,
		CloneRequest: speech.CloneVoiceRequest{
			VoiceID:     "test-voice",
			ClonePrompt: &speech.ClonePrompt{PromptText: "prompt"},
		},
	})
	if err != nil || atomic.SourceUpload.File.FileID != 123 || atomic.PromptUpload.File.FileID != 123 || atomic.Clone.DemoAudio == "" {
		t.Fatalf("unexpected atomic clone response: %#v, %v", atomic, err)
	}
	if len(authorizations) != 3 || authorizations[0] != authorizations[1] || authorizations[1] != authorizations[2] {
		t.Fatalf("atomic clone did not pin one credential: %#v", authorizations)
	}
}

func readMultipartFields(t *testing.T, reader *multipart.Reader) map[string]string {
	t.Helper()
	fields := make(map[string]string)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return fields
		}
		if err != nil {
			t.Fatalf("read multipart: %v", err)
		}
		value, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read multipart part: %v", err)
		}
		fields[part.FormName()] = string(value)
	}
}
