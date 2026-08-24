// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"bufio"
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

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

type miniMaxRoundTripFunc func(*http.Request) (*http.Response, error)

func (f miniMaxRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type contextReadCloser struct {
	ctx context.Context
}

func (r *contextReadCloser) Read(_ []byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

func (r *contextReadCloser) Close() error { return nil }

type errorReadCloser struct {
	err error
}

func (r *errorReadCloser) Read(_ []byte) (int, error) { return 0, r.err }

func (r *errorReadCloser) Close() error { return nil }

func TestMiniMaxDefaultAndOverseasEndpoints(t *testing.T) {
	var endpoints []string
	responses := map[string]string{
		"/v1/chat/completions": `{"choices":[{"index":0,"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`,
		"/v1/t2a_v2":           `{"data":{"audio":"abcd","status":2},"base_resp":{"status_code":0,"status_msg":"success"}}`,
		"/v1/get_voice":        `{"system_voice":[],"base_resp":{"status_code":0,"status_msg":"success"}}`,
		"/v1/files/upload":     `{"file":{"file_id":123},"base_resp":{"status_code":0,"status_msg":"success"}}`,
		"/v1/voice_clone":      `{"demo_audio":"https://example.com/demo.mp3","base_resp":{"status_code":0,"status_msg":"success"}}`,
	}
	httpClient := &http.Client{Transport: miniMaxRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		endpoints = append(endpoints, request.URL.String())
		body, ok := responses[request.URL.Path]
		if !ok {
			t.Fatalf("unexpected MiniMax endpoint: %s", request.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})}
	client, err := NewClient(ClientConfig{
		Provider:    ProviderMiniMax,
		Credentials: CredentialConfig{APIKeys: "test-key"},
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err = client.Chat.Create(ctx, &chat.CreateRequest{
		Model: "MiniMax-M2.7",
		Messages: []chat.Message{{
			Role:    chat.RoleUser,
			Content: chat.TextContent("test"),
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Speech.Create(ctx, &speech.CreateRequest{Model: "speech-2.8-hd", Text: "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Speech.ListVoices(ctx, &speech.ListVoicesRequest{VoiceType: speech.VoiceTypeAll}); err != nil {
		t.Fatal(err)
	}
	voicePath := filepath.Join(t.TempDir(), "voice.mp3")
	if err = os.WriteFile(voicePath, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Speech.UploadVoiceFile(ctx, &speech.UploadVoiceFileRequest{Purpose: miniMaxVoiceClonePurpose, FilePath: voicePath}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Speech.CloneVoice(ctx, &speech.CloneVoiceRequest{FileID: 123, VoiceID: "test-voice"}); err != nil {
		t.Fatal(err)
	}
	wantPaths := []string{"/v1/chat/completions", "/v1/t2a_v2", "/v1/get_voice", "/v1/files/upload", "/v1/voice_clone"}
	if len(endpoints) != len(wantPaths) {
		t.Fatalf("unexpected endpoint count: %#v", endpoints)
	}
	for index, endpoint := range endpoints {
		if endpoint != "https://api.minimaxi.com"+wantPaths[index] {
			t.Fatalf("unexpected default endpoint %d: %s", index, endpoint)
		}
	}

	overseas, err := NewClient(ClientConfig{
		Provider:    ProviderMiniMax,
		BaseURL:     "https://api.minimax.io/v1",
		Credentials: CredentialConfig{APIKeys: "test-key"},
		HTTPClient:  httpClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = overseas.Speech.Create(ctx, &speech.CreateRequest{Model: "speech-2.8-hd", Text: "test"}); err != nil {
		t.Fatal(err)
	}
	if got := endpoints[len(endpoints)-1]; got != "https://api.minimax.io/v1/t2a_v2" {
		t.Fatalf("unexpected overseas endpoint: %s", got)
	}
}

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
	response, err := client.Speech.Create(nil, &speech.CreateRequest{
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
	stream, err := client.Speech.Stream(nil, &speech.StreamRequest{
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

func TestMiniMaxSpeechStreamStopsAfterPayloadError(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "invalid JSON", raw: "data: {invalid-json}\n"},
		{name: "business error", raw: "data: {\"trace_id\":\"trace-1\",\"base_resp\":{\"status_code\":1001,\"status_msg\":\"failed\"}}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			body := io.NopCloser(strings.NewReader(test.raw))
			stream := newMiniMaxTestStream(ctx, cancel, body)
			if _, err := stream.Recv(); err == nil || errors.Is(err, io.EOF) {
				t.Fatalf("expected stream error, got %v", err)
			}
			assertMiniMaxStreamFinished(t, stream)
		})
	}
}

func TestMiniMaxSpeechStreamStopsAfterReadError(t *testing.T) {
	wantErr := errors.New("read failed")
	ctx, cancel := context.WithCancel(context.Background())
	body := &errorReadCloser{err: wantErr}
	stream := newMiniMaxTestStream(ctx, cancel, body)
	if _, err := stream.Recv(); !errors.Is(err, wantErr) {
		t.Fatalf("expected read error, got %v", err)
	}
	assertMiniMaxStreamFinished(t, stream)
}

func TestMiniMaxSpeechStreamPreservesContextError(t *testing.T) {
	tests := []struct {
		name    string
		context func() (context.Context, context.CancelFunc)
		wantErr error
	}{
		{
			name: "canceled",
			context: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr: context.Canceled,
		},
		{
			name: "deadline exceeded",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 20*time.Millisecond)
			},
			wantErr: context.DeadlineExceeded,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := test.context()
			defer cancel()
			body := &contextReadCloser{ctx: ctx}
			stream := newMiniMaxTestStream(ctx, cancel, body)
			if _, err := stream.Recv(); !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
			assertMiniMaxStreamFinished(t, stream)
		})
	}
}

func newMiniMaxTestStream(ctx context.Context, cancel context.CancelFunc, body io.ReadCloser) *miniMaxSpeechStream {
	return &miniMaxSpeechStream{
		ctx: ctx, scanner: bufio.NewScanner(body), terminal: newStreamTerminal(cancel, body.Close),
	}
}

func assertMiniMaxStreamFinished(t *testing.T, stream *miniMaxSpeechStream) {
	t.Helper()
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after stream error, got %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
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
	voices, err := client.Speech.ListVoices(nil, &speech.ListVoicesRequest{VoiceType: speech.VoiceTypeAll})
	if err != nil || len(voices.SystemVoices) != 1 {
		t.Fatalf("unexpected voices: %#v, %v", voices, err)
	}
	upload, err := client.Speech.UploadVoiceFile(nil, &speech.UploadVoiceFileRequest{Purpose: miniMaxVoiceClonePurpose, FilePath: tempPath})
	if err != nil || upload.File.FileID != 123 {
		t.Fatalf("unexpected upload: %#v, %v", upload, err)
	}
	clone, err := client.Speech.CloneVoice(nil, &speech.CloneVoiceRequest{FileID: upload.File.FileID, VoiceID: "test-voice"})
	if err != nil || clone.DemoAudio == "" {
		t.Fatalf("unexpected clone response: %#v, %v", clone, err)
	}

	authorizations = nil
	client, err = NewClient(ClientConfig{Provider: ProviderMiniMax, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key-1,key-2"}})
	if err != nil {
		t.Fatal(err)
	}
	atomic, err := client.Speech.CloneVoiceFromFiles(nil, &speech.CloneVoiceFromFilesRequest{
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
